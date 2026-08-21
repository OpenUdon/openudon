// Package browserauthor owns frontend-neutral orchestration of an isolated
// Browsertools author-session worker. It deliberately exposes only the reduced
// v2 protocol vocabulary; Playwright, credentials, cookies, storage, and raw
// child diagnostics stay behind the worker process boundary.
package browserauthor

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/browsertools/capture"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

const (
	DefaultOperatorIdle = 30 * time.Minute
	DefaultAbsolute     = 2 * time.Hour
	maxProtocolLine     = 64 << 10
)

var profileIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// Config fixes all browser authority before the worker is launched.
type Config struct {
	PrivateRoot   string
	DriverDir     string
	InitialURL    string
	DashboardURL  string
	Goal          string
	Origins       []string
	ProfileID     string
	GoalPredicate authorresult.GoalPredicate
	OperatorIdle  time.Duration
	Absolute      time.Duration
}

// Event is one reduced, browser-safe state transition.
type Event struct {
	State       string                     `json:"state"`
	Phase       string                     `json:"phase,omitempty"`
	Observation *authorsession.Observation `json:"observation,omitempty"`
	Approval    *authorsession.Approval    `json:"approval,omitempty"`
	Checkpoint  *authorsession.Checkpoint  `json:"checkpoint,omitempty"`
	Result      *authorsession.Result      `json:"result,omitempty"`
	ErrorCode   string                     `json:"error_code,omitempty"`
}

// Response is the closed set of human decisions accepted for a pending event.
// It contains no field capable of carrying a credential or challenge value.
type Response struct {
	Kind          string                        `json:"kind"`
	CandidateID   string                        `json:"candidate_id,omitempty"`
	Action        string                        `json:"action,omitempty"`
	URL           string                        `json:"url,omitempty"`
	Context       string                        `json:"context,omitempty"`
	POSTBudget    int                           `json:"post_budget,omitempty"`
	ApprovalID    string                        `json:"approval_id,omitempty"`
	ChallengeKind string                        `json:"challenge_kind,omitempty"`
	Confirmed     bool                          `json:"confirmed,omitempty"`
	Outputs       []authorsession.OutputRequest `json:"outputs,omitempty"`
}

// Session is one asynchronously driven worker process.
type Session struct {
	mu        sync.Mutex
	cancel    context.CancelFunc
	responses chan Response
	events    chan Event
	done      chan struct{}
	closed    bool
}

// Doctor performs Browsertools' typed Chromium readiness check with the
// required 30-second upper bound. It never installs a driver or browser.
func Doctor(ctx context.Context, privateRoot, driverDir string) (capture.DoctorReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := validatePrivateRoot(privateRoot); err != nil {
		return capture.DoctorReport{}, err
	}
	executable, err := os.Executable()
	if err != nil {
		return capture.DoctorReport{}, fmt.Errorf("locate iCoT executable: %w", err)
	}
	stable, cleanup, err := stabilizeExecutable(executable, privateRoot)
	if err != nil {
		return capture.DoctorReport{}, err
	}
	defer cleanup()
	args := []string{stable, "__browsertools-worker", "playwright-doctor", "chromium"}
	if strings.TrimSpace(driverDir) != "" {
		args = append(args, "--driver-dir", strings.TrimSpace(driverDir))
	}
	var output bytes.Buffer
	runErr := processgroup.Run(bounded, 30*time.Second, processgroup.Invocation{Args: args, Env: minimalEnvironment(), Stdout: &output, Stderr: io.Discard})
	if errors.Is(runErr, processgroup.ErrTerminationTimeout) {
		return capture.DoctorReport{}, fmt.Errorf("isolated Chromium doctor teardown failed: %w", runErr)
	}
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	decoder.DisallowUnknownFields()
	var report capture.DoctorReport
	if err := decoder.Decode(&report); err != nil || report.Version != capture.DoctorVersion || report.Engine != capture.EngineChromium {
		return capture.DoctorReport{}, errors.New("isolated Chromium doctor returned an invalid report")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return capture.DoctorReport{}, errors.New("isolated Chromium doctor returned trailing data")
	}
	if runErr != nil {
		return report, errors.New("isolated Chromium readiness check failed")
	}
	return report, nil
}

// TeardownFailed reports whether an isolated Browsertools process tree did
// not confirm termination within its containment deadline.
func TeardownFailed(err error) bool {
	return errors.Is(err, processgroup.ErrTerminationTimeout)
}

// Start stabilizes the current iCoT executable beneath the private root and
// re-executes its hidden Browsertools worker entry point.
func Start(ctx context.Context, config Config) (*Session, error) {
	if ctx == nil {
		return nil, errors.New("browser author context is required")
	}
	config, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate iCoT executable: %w", err)
	}
	stable, cleanup, err := stabilizeExecutable(executable, config.PrivateRoot)
	if err != nil {
		return nil, err
	}
	bounded, cancel := context.WithTimeout(ctx, config.Absolute)
	args := []string{stable, "__browsertools-worker", "author-session", "chromium", "--private-root", config.PrivateRoot}
	if config.DriverDir != "" {
		args = append(args, "--driver-dir", config.DriverDir)
	}
	child, err := processgroup.StartInteractive(bounded, args, minimalEnvironment(), io.Discard)
	if err != nil {
		cancel()
		cleanup()
		return nil, fmt.Errorf("start isolated browser worker: %w", err)
	}
	session := &Session{
		cancel: cancel, responses: make(chan Response), events: make(chan Event, 2), done: make(chan struct{}),
	}
	go session.run(bounded, config, child, cleanup)
	return session, nil
}

// Events yields reduced state changes until the session terminates.
func (s *Session) Events() <-chan Event { return s.events }

// Respond submits one typed decision for the current pending event.
func (s *Session) Respond(ctx context.Context, response Response) error {
	if s == nil {
		return errors.New("browser author session is unavailable")
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return errors.New("browser author session is closed")
	}
	select {
	case s.responses <- response:
		return nil
	case <-s.done:
		return errors.New("browser author session is closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Cancel terminates the complete worker process tree.
func (s *Session) Cancel() {
	if s == nil {
		return
	}
	s.cancel()
}

func (s *Session) run(ctx context.Context, config Config, child *processgroup.InteractiveChild, cleanup func()) {
	defer cleanup()
	defer close(s.events)
	defer close(s.done)
	defer func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
	}()
	defer func() {
		_ = child.Input().Close()
		if err := child.Terminate(); errors.Is(err, processgroup.ErrTerminationTimeout) {
			s.publishTerminal(Event{State: "failed", ErrorCode: "worker_teardown"})
		}
	}()
	messages := make(chan authorsession.ServerMessage)
	readErrors := make(chan error, 1)
	go scanMessages(child.Output(), messages, readErrors)
	write := func(message authorsession.ClientMessage) error {
		message.Protocol = authorsession.Protocol
		data, err := json.Marshal(message)
		if err != nil {
			return err
		}
		_, err = child.Input().Write(append(data, '\n'))
		return err
	}
	fail := func(code string) {
		s.publishTerminal(Event{State: "failed", ErrorCode: code})
	}
	first, err := receive(ctx, messages, readErrors)
	if err != nil || first.Type != "hello" || first.Protocol != authorsession.Protocol || !requiredCapabilities(first.Capabilities) {
		fail("protocol_negotiation")
		return
	}
	s.publish(ctx, Event{State: "launching"})
	bounds := authorresult.Bounds{
		NavigationTimeoutMS: authorsession.DefaultNavigationTimeout.Milliseconds(), TotalTimeoutMS: authorsession.DefaultTotalTimeout.Milliseconds(),
		MaxRequests: authorsession.DefaultMaxRequests, MaxResponseBytes: authorsession.DefaultMaxResponseBytes,
		MaxObservations: authorsession.DefaultMaxObservations, MaxCandidates: authorsession.DefaultMaxCandidates, MaxOutputs: authorsession.DefaultMaxOutputs,
	}
	if err := write(authorsession.ClientMessage{
		Type: "start", Title: config.ProfileID, URL: config.InitialURL, DashboardURL: config.DashboardURL,
		Goal: config.Goal, Origins: config.Origins, GoalPredicate: &config.GoalPredicate, Bounds: &bounds,
	}); err != nil {
		fail("worker_write")
		return
	}
	phase := "authentication"
	var completionObservation *authorsession.Observation
	for {
		message, err := receive(ctx, messages, readErrors)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				fail("absolute_timeout")
			} else if ctx.Err() != nil {
				s.publishTerminal(Event{State: "canceled"})
			} else {
				fail("worker_protocol")
			}
			return
		}
		if message.Protocol != authorsession.Protocol {
			fail("protocol_mismatch")
			return
		}
		switch message.Type {
		case "state":
			if message.Phase != "" {
				phase = message.Phase
			}
			if phase == "completed" {
				if err := write(authorsession.ClientMessage{Type: "finish"}); err != nil {
					fail("worker_write")
					return
				}
				continue
			}
			state := "exploration"
			if phase == "authentication" {
				state = "authentication"
			}
			s.publish(ctx, Event{State: state, Phase: phase})
			contextID := message.Context
			if contextID == "" {
				contextID = "main"
			}
			if err := write(authorsession.ClientMessage{Type: "observe", Context: contextID}); err != nil {
				fail("worker_write")
				return
			}
		case "observation":
			if message.Observation == nil || !safeObservation(*message.Observation, config.Origins) {
				fail("malformed_observation")
				return
			}
			if observationMatchesGoal(*message.Observation, config.GoalPredicate) {
				copy := *message.Observation
				copy.Candidates = append([]authorsession.Candidate(nil), message.Observation.Candidates...)
				completionObservation = &copy
				continue
			}
			response, ok := s.awaitResponse(ctx, config.OperatorIdle, Event{State: "exploration", Phase: phase, Observation: message.Observation})
			if !ok {
				return
			}
			client, err := observationResponse(response, *message.Observation, config)
			if err != nil || write(client) != nil {
				fail("invalid_response")
				return
			}
		case "approval_required":
			if message.Approval == nil || !safeApproval(*message.Approval, config.Origins) {
				fail("malformed_approval")
				return
			}
			response, ok := s.awaitResponse(ctx, config.OperatorIdle, Event{State: "action_approval", Phase: phase, Approval: message.Approval})
			if !ok {
				return
			}
			kind := "deny"
			if response.Kind == "approve" && response.ApprovalID == message.Approval.ID {
				kind = "approve"
			} else if response.Kind != "deny" || response.ApprovalID != message.Approval.ID {
				fail("invalid_response")
				return
			}
			if err := write(authorsession.ClientMessage{Type: kind, ApprovalID: message.Approval.ID}); err != nil {
				fail("worker_write")
				return
			}
		case "human_checkpoint":
			if message.Checkpoint == nil {
				fail("malformed_checkpoint")
				return
			}
			state := "human_input"
			if message.Checkpoint.Kind == "completion" {
				state = "completion_review"
			}
			event := Event{State: state, Phase: phase, Checkpoint: message.Checkpoint}
			if message.Checkpoint.Kind == "completion" {
				event.Observation = completionObservation
			}
			response, ok := s.awaitResponse(ctx, config.OperatorIdle, event)
			if !ok {
				return
			}
			client, err := checkpointResponse(response, *message.Checkpoint)
			if err != nil || write(client) != nil {
				fail("invalid_response")
				return
			}
		case "result":
			if message.Result == nil || message.Result.ArtifactPath == "" || !strings.HasPrefix(message.Result.Digest, "sha256:") {
				fail("malformed_result")
				return
			}
			s.publish(ctx, Event{State: "completion_review", Phase: "completed", Result: message.Result})
			_ = child.Input().Close()
			_ = child.Wait()
			return
		case "diagnostic":
			if message.Diagnostic == nil || message.Diagnostic.Code == "" {
				fail("malformed_diagnostic")
				return
			}
		case "error":
			fail("worker_failed")
			return
		default:
			fail("unexpected_message")
			return
		}
	}
}

func (s *Session) awaitResponse(ctx context.Context, idle time.Duration, event Event) (Response, bool) {
	if !s.publish(ctx, event) {
		return Response{}, false
	}
	timer := time.NewTimer(idle)
	defer timer.Stop()
	select {
	case response := <-s.responses:
		return response, true
	case <-timer.C:
		s.publishTerminal(Event{State: "canceled", ErrorCode: "operator_idle_timeout"})
		s.cancel()
		return Response{}, false
	case <-ctx.Done():
		s.publishTerminal(Event{State: "canceled"})
		return Response{}, false
	}
}

func (s *Session) publish(ctx context.Context, event Event) bool {
	select {
	case s.events <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

func (s *Session) publishTerminal(event Event) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case s.events <- event:
	case <-timer.C:
	}
}

func scanMessages(reader io.Reader, output chan<- authorsession.ServerMessage, failures chan<- error) {
	defer close(output)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), maxProtocolLine)
	for scanner.Scan() {
		decoder := json.NewDecoder(strings.NewReader(scanner.Text()))
		decoder.DisallowUnknownFields()
		var message authorsession.ServerMessage
		if err := decoder.Decode(&message); err != nil {
			failures <- err
			return
		}
		var trailing any
		if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
			failures <- errors.New("protocol message contains trailing data")
			return
		}
		output <- message
	}
	if err := scanner.Err(); err != nil {
		failures <- err
		return
	}
	failures <- io.EOF
}

func receive(ctx context.Context, messages <-chan authorsession.ServerMessage, failures <-chan error) (authorsession.ServerMessage, error) {
	select {
	case message, ok := <-messages:
		if !ok {
			return authorsession.ServerMessage{}, io.EOF
		}
		return message, nil
	case err := <-failures:
		return authorsession.ServerMessage{}, err
	case <-ctx.Done():
		return authorsession.ServerMessage{}, ctx.Err()
	}
}

func normalizeConfig(config Config) (Config, error) {
	if config.OperatorIdle <= 0 {
		config.OperatorIdle = DefaultOperatorIdle
	}
	if config.Absolute <= 0 {
		config.Absolute = DefaultAbsolute
	}
	if config.OperatorIdle > DefaultOperatorIdle || config.Absolute > DefaultAbsolute {
		return Config{}, errors.New("browser author timeout exceeds the supported ceiling")
	}
	if err := validatePrivateRoot(config.PrivateRoot); err != nil {
		return Config{}, err
	}
	initial, initialOrigin, err := cleanURL(config.InitialURL)
	if err != nil {
		return Config{}, fmt.Errorf("initial URL: %w", err)
	}
	dashboard, dashboardOrigin, err := cleanURL(config.DashboardURL)
	if err != nil {
		return Config{}, fmt.Errorf("dashboard URL: %w", err)
	}
	origins := make([]string, 0, len(config.Origins))
	seen := map[string]bool{}
	for _, raw := range config.Origins {
		origin, err := cleanOrigin(raw)
		if err != nil {
			return Config{}, err
		}
		if !seen[origin] {
			seen[origin] = true
			origins = append(origins, origin)
		}
	}
	sort.Strings(origins)
	if !seen[initialOrigin] || !seen[dashboardOrigin] || len(origins) == 0 {
		return Config{}, errors.New("approved origins must include the initial and dashboard origins")
	}
	config.Goal = strings.TrimSpace(config.Goal)
	config.ProfileID = strings.ToLower(strings.TrimSpace(config.ProfileID))
	if config.Goal == "" || len(config.Goal) > 1024 || !profileIDPattern.MatchString(config.ProfileID) {
		return Config{}, errors.New("browser author goal or profile ID is invalid")
	}
	if config.GoalPredicate.Origin == "" {
		config.GoalPredicate = authorresult.GoalPredicate{Origin: dashboardOrigin, Path: pathOf(dashboard), Context: "main", Role: "heading", Label: "Dashboard"}
	}
	if !seen[config.GoalPredicate.Origin] || !cleanCapturePath(config.GoalPredicate.Path) || config.GoalPredicate.Role == "" {
		return Config{}, errors.New("browser author goal predicate is outside approved authority")
	}
	config.InitialURL, config.DashboardURL, config.Origins = initial, dashboard, origins
	config.DriverDir = strings.TrimSpace(config.DriverDir)
	return config, nil
}

func validatePrivateRoot(privateRoot string) error {
	clean := filepath.Clean(privateRoot)
	info, err := os.Lstat(clean)
	resolved, resolveErr := filepath.EvalSymlinks(clean)
	if err != nil || resolveErr != nil || !filepath.IsAbs(clean) || resolved != clean || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return errors.New("browser author private root must be an absolute mode-0700 non-symlink directory")
	}
	return nil
}

func cleanURL(raw string) (string, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.User != nil || parsed.Host == "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", "", errors.New("URL must be an absolute clean URL")
	}
	host := strings.ToLower(parsed.Host)
	if parsed.Scheme != "https" && !(parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname())) {
		return "", "", errors.New("URL must use HTTPS or loopback HTTP")
	}
	if parsed.Path == "" {
		parsed.Path = "/"
	}
	if !cleanCapturePath(parsed.EscapedPath()) {
		return "", "", errors.New("URL path must be portable and query-free")
	}
	parsed.Scheme, parsed.Host = strings.ToLower(parsed.Scheme), host
	return parsed.String(), parsed.Scheme + "://" + parsed.Host, nil
}

func cleanCapturePath(path string) bool {
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "?#\\") {
		return false
	}
	for _, part := range strings.Split(path, "/")[1:] {
		decoded, err := url.PathUnescape(part)
		if err != nil || decoded == "." || decoded == ".." || strings.ContainsAny(decoded, "/\\") {
			return false
		}
	}
	return true
}

func cleanOrigin(raw string) (string, error) {
	parsed, origin, err := cleanURL(raw)
	if err != nil {
		return "", fmt.Errorf("approved origin is invalid")
	}
	value, _ := url.Parse(parsed)
	if value.Path != "" && value.Path != "/" || value.RawQuery != "" {
		return "", errors.New("approved origin must not contain a path or query")
	}
	return origin, nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	return net.ParseIP(host).IsLoopback()
}

func pathOf(raw string) string {
	parsed, _ := url.Parse(raw)
	if parsed.EscapedPath() == "" {
		return "/"
	}
	return parsed.EscapedPath()
}

func requiredCapabilities(values []string) bool {
	set := map[string]bool{}
	for _, value := range values {
		set[value] = true
	}
	for _, required := range []string{"chromium", "human_credentials", "reviewed_mfa_kind", "reviewed_outputs", "reduced_observation", "typed_goal"} {
		if !set[required] {
			return false
		}
	}
	return true
}

func safeObservation(observation authorsession.Observation, origins []string) bool {
	allowed := map[string]bool{}
	for _, origin := range origins {
		allowed[origin] = true
	}
	if !allowed[observation.Origin] || observation.Path == "" || len(observation.Candidates) > authorsession.DefaultMaxCandidates {
		return false
	}
	seen := map[string]bool{}
	for _, candidate := range observation.Candidates {
		if candidate.ID == "" || candidate.Role == "" || candidate.Matches < 1 || candidate.Matches > 32 || len(candidate.Label) > 256 || strings.ContainsAny(candidate.Label, "\r\n\x00") || seen[candidate.ID] {
			return false
		}
		seen[candidate.ID] = true
	}
	return true
}

func safeApproval(approval authorsession.Approval, origins []string) bool {
	if approval.ID == "" || approval.Kind == "" || approval.POSTBudget < 0 {
		return false
	}
	if approval.Origin == "" {
		return true
	}
	for _, origin := range origins {
		if approval.Origin == origin {
			return true
		}
	}
	return false
}

func observationResponse(response Response, observation authorsession.Observation, config Config) (authorsession.ClientMessage, error) {
	if response.Kind == "authenticated" {
		return authorsession.ClientMessage{Type: "execute", Action: "navigate_get", URL: config.DashboardURL, Context: "main"}, nil
	}
	candidateKnown := false
	for _, candidate := range observation.Candidates {
		if candidate.ID == response.CandidateID {
			candidateKnown = true
		}
	}
	switch response.Kind {
	case "focus_human_input":
		if !candidateKnown {
			return authorsession.ClientMessage{}, errors.New("candidate is not current")
		}
		return authorsession.ClientMessage{Type: "focus_human_input", CandidateID: response.CandidateID}, nil
	case "click":
		if !candidateKnown || response.POSTBudget < 0 {
			return authorsession.ClientMessage{}, errors.New("candidate or POST budget is invalid")
		}
		return authorsession.ClientMessage{Type: "execute", CandidateID: response.CandidateID, Action: "click", POSTBudget: response.POSTBudget}, nil
	case "navigate_get":
		urlValue, origin, err := cleanURL(response.URL)
		if err != nil || !contains(config.Origins, origin) {
			return authorsession.ClientMessage{}, errors.New("navigation is outside approved origins")
		}
		return authorsession.ClientMessage{Type: "execute", Action: "navigate_get", URL: urlValue, Context: response.Context}, nil
	case "observe":
		contextID := strings.TrimSpace(response.Context)
		if contextID == "" {
			contextID = observation.Context
		}
		return authorsession.ClientMessage{Type: "observe", Context: contextID}, nil
	default:
		return authorsession.ClientMessage{}, errors.New("observation response is invalid")
	}
}

func observationMatchesGoal(observation authorsession.Observation, goal authorresult.GoalPredicate) bool {
	contextID := goal.Context
	if contextID == "" {
		contextID = "main"
	}
	if observation.Origin != goal.Origin || observation.Path != goal.Path || observation.Context != contextID {
		return false
	}
	for _, candidate := range observation.Candidates {
		if candidate.Role == goal.Role && candidate.Matches == 1 && (goal.Label == "" || candidate.Label == goal.Label) {
			return true
		}
	}
	return false
}

func checkpointResponse(response Response, checkpoint authorsession.Checkpoint) (authorsession.ClientMessage, error) {
	switch checkpoint.Kind {
	case "credential":
		if response.Kind != "continue" || response.CandidateID != checkpoint.CandidateID {
			return authorsession.ClientMessage{}, errors.New("credential checkpoint response is invalid")
		}
		return authorsession.ClientMessage{Type: "human_input_complete", CandidateID: checkpoint.CandidateID}, nil
	case "mfa":
		if response.Kind != "continue" || response.CandidateID != checkpoint.CandidateID || !contains(checkpoint.ChallengeKinds, response.ChallengeKind) {
			return authorsession.ClientMessage{}, errors.New("MFA checkpoint response is invalid")
		}
		return authorsession.ClientMessage{Type: "human_input_complete", CandidateID: checkpoint.CandidateID, ChallengeKind: response.ChallengeKind}, nil
	case "completion":
		if response.Kind != "confirm" || !response.Confirmed || len(response.Outputs) > authorsession.DefaultMaxOutputs {
			return authorsession.ClientMessage{}, errors.New("completion response is invalid")
		}
		outputs := append([]authorsession.OutputRequest(nil), response.Outputs...)
		return authorsession.ClientMessage{Type: "human_complete", Confirmed: true, Outputs: &outputs}, nil
	default:
		return authorsession.ClientMessage{}, errors.New("checkpoint kind is invalid")
	}
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func minimalEnvironment() []string {
	allowed := []string{"DBUS_SESSION_BUS_ADDRESS", "DISPLAY", "HOME", "LANG", "LC_ALL", "NO_PROXY", "PATH", "PLAYWRIGHT_BROWSERS_PATH", "TMPDIR", "WAYLAND_DISPLAY", "XAUTHORITY", "XDG_RUNTIME_DIR"}
	var result []string
	for _, name := range allowed {
		if value, ok := os.LookupEnv(name); ok && !strings.ContainsAny(value, "\r\n\x00") {
			result = append(result, name+"="+value)
		}
	}
	sort.Strings(result)
	return result
}

func stabilizeExecutable(source, privateRoot string) (string, func(), error) {
	info, err := os.Lstat(source)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", nil, errors.New("iCoT executable is not a safe executable file")
	}
	directory, err := os.MkdirTemp(privateRoot, ".openudon-browser-worker-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(directory) }
	if err := os.Chmod(directory, 0o700); err != nil {
		cleanup()
		return "", nil, err
	}
	input, err := os.Open(source)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	defer input.Close()
	openedBefore, err := input.Stat()
	if err != nil || !os.SameFile(info, openedBefore) || openedBefore.Size() > 256<<20 {
		cleanup()
		return "", nil, errors.New("iCoT executable identity changed before stabilization")
	}
	name := "icot"
	if strings.EqualFold(filepath.Ext(source), ".exe") {
		name += ".exe"
	}
	target := filepath.Join(directory, name)
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o500)
	if err != nil {
		cleanup()
		return "", nil, err
	}
	written, copyErr := io.Copy(output, io.LimitReader(input, (256<<20)+1))
	syncErr, closeErr := output.Sync(), output.Close()
	openedAfter, statErr := input.Stat()
	pathAfter, pathErr := os.Lstat(source)
	if copyErr != nil || syncErr != nil || closeErr != nil || statErr != nil || pathErr != nil ||
		written != info.Size() || written > 256<<20 || !os.SameFile(openedBefore, openedAfter) || !os.SameFile(openedBefore, pathAfter) ||
		openedBefore.Size() != openedAfter.Size() || !openedBefore.ModTime().Equal(openedAfter.ModTime()) {
		cleanup()
		return "", nil, errors.New("could not stabilize iCoT browser worker executable")
	}
	targetInfo, err := os.Lstat(target)
	if err != nil || targetInfo.Mode()&os.ModeSymlink != 0 || !targetInfo.Mode().IsRegular() || targetInfo.Mode().Perm() != 0o500 {
		cleanup()
		return "", nil, errors.New("stabilized iCoT browser worker is unsafe")
	}
	return target, cleanup, nil
}
