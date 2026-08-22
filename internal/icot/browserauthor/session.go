// Package browserauthor owns frontend-neutral orchestration of an isolated
// Browsertools author-session worker. It deliberately exposes only the reduced
// v2 protocol vocabulary; Playwright, credentials, cookies, storage, and raw
// child diagnostics stay behind the worker process boundary.
package browserauthor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/OpenUdon/browsertools/disclosurepath"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

const (
	DefaultOperatorIdle = 30 * time.Minute
	DefaultAbsolute     = 2 * time.Hour
	maxProtocolLine     = 64 << 10
	DoctorVersion       = "browsertools.playwright-doctor.v1"
	EngineChromium      = "chromium"
)

// DoctorCapability mirrors one value-free Browsertools capability decision at
// the isolated worker JSON boundary without importing its Playwright package.
type DoctorCapability struct {
	Name        string `json:"name"`
	Disposition string `json:"disposition"`
	Reason      string `json:"reason"`
}

// DoctorReport is the full process-local worker wire. BrowserExecutable and
// Error are sanitized before any UI state, ETag, or response is created.
type DoctorReport struct {
	Version             string             `json:"version"`
	Engine              string             `json:"engine"`
	PlaywrightGoVersion string             `json:"playwright_go_version"`
	PlaywrightVersion   string             `json:"playwright_version"`
	DriverReady         bool               `json:"driver_ready"`
	BrowserReady        bool               `json:"browser_ready"`
	BrowserExecutable   string             `json:"browser_executable,omitempty"`
	CapabilityPolicy    []DoctorCapability `json:"capability_policy"`
	Error               string             `json:"error,omitempty"`
}

// UIDoctorReport is the closed path-free HTTP shape.
type UIDoctorReport struct {
	Version             string             `json:"version"`
	Engine              string             `json:"engine"`
	PlaywrightGoVersion string             `json:"playwright_go_version"`
	PlaywrightVersion   string             `json:"playwright_version"`
	DriverReady         bool               `json:"driver_ready"`
	BrowserReady        bool               `json:"browser_ready"`
	CapabilityPolicy    []DoctorCapability `json:"capability_policy"`
	Error               string             `json:"error,omitempty"`
}

// UI reduces the full worker report before it reaches server-owned state.
func (report DoctorReport) UI() UIDoctorReport {
	safe := UIDoctorReport{
		Version: report.Version, Engine: report.Engine,
		PlaywrightGoVersion: report.PlaywrightGoVersion, PlaywrightVersion: report.PlaywrightVersion,
		DriverReady: report.DriverReady, BrowserReady: report.BrowserReady,
		CapabilityPolicy: append([]DoctorCapability(nil), report.CapabilityPolicy...),
	}
	if report.Error != "" {
		switch {
		case !report.DriverReady:
			safe.Error = "Playwright driver is unavailable"
		case !report.BrowserReady:
			safe.Error = "installed browser is unavailable"
		default:
			safe.Error = "Playwright teardown failed"
		}
	}
	return safe
}

var (
	profileIDPattern   = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)
	candidateIDPattern = regexp.MustCompile(`^candidate-[0-9a-f]{16}$`)
	approvalIDPattern  = regexp.MustCompile(`^approval-[0-9]{4}$`)
	contextIDPattern   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
	diagnosticPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)
	portableRoles      = map[string]bool{
		"button": true, "link": true, "textbox": true, "checkbox": true, "radio": true,
		"dialog": true, "status": true, "alert": true, "heading": true, "img": true,
		"list": true, "listitem": true, "combobox": true, "option": true, "menu": true,
		"menuitem": true, "tab": true, "tabpanel": true, "table": true, "row": true,
		"cell": true, "region": true, "navigation": true, "article": true, "form": true,
		"search": true, "switch": true, "group": true,
	}
	otpChallengeKinds = map[string]bool{
		"totp": true, "sms_otp": true, "email_otp": true, "voice_otp": true,
	}
	nonInputChallengeKinds = map[string]bool{
		"push": true, "push_number_match": true, "passkey": true, "security_key": true,
	}
)

var copyStabilizedExecutable = io.Copy

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
	Attestation *Attestation               `json:"-"`
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
func Doctor(ctx context.Context, privateRoot, driverDir string) (DoctorReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	bounded, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := validatePrivateRoot(privateRoot); err != nil {
		return DoctorReport{}, err
	}
	executable, err := os.Executable()
	if err != nil {
		return DoctorReport{}, fmt.Errorf("locate iCoT executable: %w", err)
	}
	stable, cleanup, err := stabilizeExecutable(executable, privateRoot)
	if err != nil {
		return DoctorReport{}, err
	}
	defer cleanup()
	args := []string{stable, "__browsertools-worker", "playwright-doctor", "chromium"}
	if strings.TrimSpace(driverDir) != "" {
		args = append(args, "--driver-dir", strings.TrimSpace(driverDir))
	}
	var output bytes.Buffer
	runErr := processgroup.Run(bounded, 30*time.Second, processgroup.Invocation{Args: args, Env: minimalEnvironment(), Stdout: &output, Stderr: io.Discard})
	if errors.Is(runErr, processgroup.ErrTerminationTimeout) {
		return DoctorReport{}, fmt.Errorf("isolated Chromium doctor teardown failed: %w", runErr)
	}
	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	decoder.DisallowUnknownFields()
	var report DoctorReport
	if err := decoder.Decode(&report); err != nil || report.Version != DoctorVersion || report.Engine != EngineChromium {
		return DoctorReport{}, errors.New("isolated Chromium doctor returned an invalid report")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return DoctorReport{}, errors.New("isolated Chromium doctor returned trailing data")
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
	args := []string{stable, "__browsertools-worker", "author-session", "chromium", "--private-root", config.PrivateRoot}
	if config.DriverDir != "" {
		args = append(args, "--driver-dir", config.DriverDir)
	}
	return startProcess(ctx, config, args, cleanup)
}

// StartExternal runs an explicitly selected Browsertools executable through
// the same typed controller and parent-attestation state machine as Start.
func StartExternal(ctx context.Context, config Config, executable string) (*Session, error) {
	if ctx == nil {
		return nil, errors.New("browser author context is required")
	}
	config, err := normalizeConfig(config)
	if err != nil {
		return nil, err
	}
	stable, cleanup, err := stabilizeExecutable(strings.TrimSpace(executable), config.PrivateRoot)
	if err != nil {
		return nil, err
	}
	args := []string{stable, "author-session", "chromium", "--private-root", config.PrivateRoot}
	if config.DriverDir != "" {
		args = append(args, "--driver-dir", config.DriverDir)
	}
	return startProcess(ctx, config, args, cleanup)
}

func startProcess(ctx context.Context, config Config, args []string, cleanup func()) (*Session, error) {
	bounded, cancel := context.WithTimeout(ctx, config.Absolute)
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
	attestation, err := newAttestation(config, bounds)
	if err != nil {
		fail("attestation")
		return
	}
	if err := write(authorsession.ClientMessage{
		Type: "start", Title: config.ProfileID, URL: config.InitialURL, DashboardURL: config.DashboardURL,
		Goal: config.Goal, Origins: config.Origins, GoalPredicate: &config.GoalPredicate, Bounds: &bounds,
	}); err != nil {
		fail("worker_write")
		return
	}
	phase := "authentication"
	receivedInitialState := false
	var completionObservation *authorsession.Observation
	var lastObservation *authorsession.Observation
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
		if !receivedInitialState && message.Type != "state" {
			fail("protocol_state")
			return
		}
		switch message.Type {
		case "state":
			if !safeState(message, bounds, receivedInitialState) {
				fail("malformed_state")
				return
			}
			phase = message.Phase
			receivedInitialState = true
			if phase == "completed" {
				if err := write(authorsession.ClientMessage{Type: "finish"}); err != nil {
					fail("worker_write")
					return
				}
				continue
			}
			if phase == "closed" {
				s.publishTerminal(Event{State: "closed", Phase: phase})
				_ = child.Input().Close()
				_ = child.Wait()
				return
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
			if message.Observation == nil || !safeObservation(*message.Observation, attestation.originLedger()) || attestation.recordObservation(phase, *message.Observation) != nil {
				fail("malformed_observation")
				return
			}
			copyObservation := cloneObservation(*message.Observation)
			lastObservation = &copyObservation
			if phase == "authentication" && message.Observation.Origin == attestation.dashboardOrigin && message.Observation.Path == attestation.dashboardPath {
				phase = "exploration"
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
			if err != nil || attestation.recordClient(client, phase, message.Observation) != nil || write(client) != nil {
				fail("invalid_response")
				return
			}
		case "approval_required":
			if message.Approval == nil || !safeApproval(*message.Approval, attestation.originLedger()) {
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
			if kind == "approve" && attestation.recordApproval(*message.Approval) != nil {
				fail("invalid_response")
				return
			}
			if err := write(authorsession.ClientMessage{Type: kind, ApprovalID: message.Approval.ID}); err != nil {
				fail("worker_write")
				return
			}
		case "human_checkpoint":
			if message.Checkpoint == nil || !safeCheckpoint(*message.Checkpoint) {
				fail("malformed_checkpoint")
				return
			}
			if attestation.recordCheckpoint(*message.Checkpoint, lastObservation) != nil {
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
			if err != nil || attestation.recordClient(client, phase, lastObservation) != nil || write(client) != nil {
				fail("invalid_response")
				return
			}
		case "result":
			if message.Result == nil || message.Result.ArtifactPath == "" || !validSHA256Digest(message.Result.Digest) {
				fail("malformed_result")
				return
			}
			s.publish(ctx, Event{State: "completion_review", Phase: "completed", Result: message.Result, Attestation: attestation})
			_ = child.Input().Close()
			_ = child.Wait()
			return
		case "diagnostic":
			if message.Diagnostic == nil || !diagnosticPattern.MatchString(message.Diagnostic.Code) {
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
		message, err := decodeServerMessage(scanner.Bytes())
		if err != nil {
			failures <- err
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

func decodeServerMessage(data []byte) (authorsession.ServerMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return authorsession.ServerMessage{}, err
	}
	var header struct {
		Protocol string `json:"protocol"`
		Type     string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil || header.Protocol != authorsession.Protocol {
		return authorsession.ServerMessage{}, errors.New("protocol message header is invalid")
	}
	allowedByType := map[string]map[string]bool{
		"hello":             {"protocol": true, "type": true, "capabilities": true},
		"state":             {"protocol": true, "type": true, "phase": true, "context": true, "bounds": true},
		"observation":       {"protocol": true, "type": true, "observation": true},
		"approval_required": {"protocol": true, "type": true, "approval": true},
		"human_checkpoint":  {"protocol": true, "type": true, "checkpoint": true},
		"diagnostic":        {"protocol": true, "type": true, "diagnostic": true},
		"result":            {"protocol": true, "type": true, "result": true},
	}
	allowed, ok := allowedByType[header.Type]
	if !ok {
		return authorsession.ServerMessage{}, errors.New("protocol message type is invalid")
	}
	for name := range fields {
		if !allowed[name] {
			return authorsession.ServerMessage{}, errors.New("protocol message shape is invalid")
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var message authorsession.ServerMessage
	if err := decoder.Decode(&message); err != nil {
		return authorsession.ServerMessage{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return authorsession.ServerMessage{}, errors.New("protocol message contains trailing data")
	}
	return message, nil
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
		dashboardPath, pathErr := pathForURL(dashboard)
		if pathErr != nil {
			return Config{}, pathErr
		}
		config.GoalPredicate = authorresult.GoalPredicate{Origin: dashboardOrigin, Path: dashboardPath, Context: "main", Role: "heading", Label: "Dashboard"}
	}
	if !seen[config.GoalPredicate.Origin] || disclosurepath.Validate(config.GoalPredicate.Path) != nil || config.GoalPredicate.Role == "" {
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
	if disclosurepath.Validate(parsed.EscapedPath()) != nil {
		return "", "", errors.New("URL path must be portable and query-free")
	}
	parsed.Scheme, parsed.Host = strings.ToLower(parsed.Scheme), host
	return parsed.String(), parsed.Scheme + "://" + parsed.Host, nil
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

func pathForURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Host == "" {
		return "", errors.New("URL must be absolute")
	}
	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}
	if disclosurepath.Validate(path) != nil {
		return "", errors.New("URL path must be portable")
	}
	return path, nil
}

func requiredCapabilities(values []string) bool {
	if len(values) == 0 || len(values) > 32 {
		return false
	}
	set := map[string]bool{}
	for _, value := range values {
		if !diagnosticPattern.MatchString(value) || set[value] {
			return false
		}
		set[value] = true
	}
	for _, required := range []string{"chromium", "human_credentials", "reviewed_mfa_kind", "reviewed_outputs", "reduced_observation", "popup", "frame", "typed_goal"} {
		if !set[required] {
			return false
		}
	}
	return true
}

func safeState(message authorsession.ServerMessage, bounds authorresult.Bounds, receivedInitial bool) bool {
	if message.Phase != "authentication" && message.Phase != "exploration" && message.Phase != "completed" && message.Phase != "closed" {
		return false
	}
	if !contextIDPattern.MatchString(message.Context) {
		return false
	}
	if message.Bounds != nil && *message.Bounds != bounds {
		return false
	}
	if !receivedInitial {
		return message.Phase == "authentication" && message.Context == "main" && message.Bounds != nil && *message.Bounds == bounds
	}
	return true
}

func safeObservation(observation authorsession.Observation, origins []string) bool {
	allowed := map[string]bool{}
	for _, origin := range origins {
		allowed[origin] = true
	}
	if !allowed[observation.Origin] || disclosurepath.Validate(observation.Path) != nil || observation.Contexts == nil || len(observation.Contexts) > authorsession.MaxContexts || len(observation.Diagnostics) > authorsession.MaxUniqueDiagnostics || len(observation.Candidates) > authorsession.DefaultMaxCandidates || !contextIDPattern.MatchString(observation.Context) {
		return false
	}
	if observation.Context != "main" {
		if _, ok := observation.Contexts[observation.Context]; !ok {
			return false
		}
	}
	if !safeContextInventory(observation.Contexts, allowed) {
		return false
	}
	seen := map[string]bool{}
	for _, candidate := range observation.Candidates {
		if !candidateIDPattern.MatchString(candidate.ID) || !portableRoles[candidate.Role] || candidate.Matches < 1 || candidate.Matches > authorsession.DefaultMaxCandidates || seen[candidate.ID] {
			return false
		}
		if reduction := authorsession.ReduceAccessibilityLabel(candidate.Label); reduction.Value != candidate.Label {
			return false
		}
		seen[candidate.ID] = true
	}
	for _, diagnostic := range observation.Diagnostics {
		if !diagnosticPattern.MatchString(diagnostic) {
			return false
		}
	}
	return true
}

func safeContextInventory(contexts map[string]authorresult.Context, allowedOrigins map[string]bool) bool {
	for id, browserContext := range contexts {
		if id == "main" || !contextIDPattern.MatchString(id) || !contextIDPattern.MatchString(browserContext.Parent) || !allowedOrigins[browserContext.Origin] || (browserContext.Kind != "popup" && browserContext.Kind != "frame") {
			return false
		}
		if browserContext.Kind == "popup" && (browserContext.Path != "" || browserContext.Name != "") || browserContext.Kind == "frame" && browserContext.Path == "" && browserContext.Name == "" {
			return false
		}
		if browserContext.Path != "" && disclosurepath.Validate(browserContext.Path) != nil {
			return false
		}
		if browserContext.Name != "" {
			reduction := authorsession.ReduceAccessibilityLabel(browserContext.Name)
			if reduction.Reason != authorsession.LabelReasonUnchanged || reduction.Value != browserContext.Name {
				return false
			}
		}
	}
	for id := range contexts {
		seen := map[string]bool{}
		current := id
		for depth := 0; current != "main"; depth++ {
			if depth >= 4 || seen[current] {
				return false
			}
			seen[current] = true
			browserContext, ok := contexts[current]
			if !ok {
				return false
			}
			current = browserContext.Parent
		}
	}
	return true
}

func safeCheckpoint(checkpoint authorsession.Checkpoint) bool {
	switch checkpoint.Kind {
	case "credential":
		return candidateIDPattern.MatchString(checkpoint.CandidateID) && (checkpoint.InputKind == "identifier" || checkpoint.InputKind == "password") && len(checkpoint.ChallengeKinds) == 0
	case "mfa":
		if !candidateIDPattern.MatchString(checkpoint.CandidateID) || (checkpoint.InputKind != "otp" && checkpoint.InputKind != "mfa") || len(checkpoint.ChallengeKinds) == 0 {
			return false
		}
		allowed := otpChallengeKinds
		if checkpoint.InputKind == "mfa" {
			allowed = nonInputChallengeKinds
		}
		seen := map[string]bool{}
		for _, kind := range checkpoint.ChallengeKinds {
			if !allowed[kind] || seen[kind] {
				return false
			}
			seen[kind] = true
		}
		return true
	case "completion":
		return checkpoint.CandidateID == "" && checkpoint.InputKind == "" && len(checkpoint.ChallengeKinds) == 0
	default:
		return false
	}
}

func validSHA256Digest(value string) bool {
	if value != strings.ToLower(strings.TrimSpace(value)) || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	decoded, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil && len(decoded) == sha256.Size
}

func safeApproval(approval authorsession.Approval, origins []string) bool {
	if !approvalIDPattern.MatchString(approval.ID) || approval.POSTBudget < 0 || approval.POSTBudget > 32 {
		return false
	}
	switch approval.Kind {
	case "action":
		return approval.Origin == "" && approval.Action == "click" && candidateIDPattern.MatchString(approval.CandidateID)
	case "origin", "origin_action":
		canonical, err := cleanOrigin(approval.Origin)
		if err != nil || canonical != approval.Origin {
			return false
		}
		for _, origin := range origins {
			if approval.Origin == origin {
				return false
			}
		}
		if approval.Kind == "origin" {
			return approval.Action == "navigate_get" && approval.CandidateID == "" && approval.POSTBudget == 0
		}
		return approval.Action == "click" && candidateIDPattern.MatchString(approval.CandidateID)
	default:
		return false
	}
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
		if err != nil {
			return authorsession.ClientMessage{}, errors.New("navigation URL is invalid")
		}
		_ = origin // The worker requests exact human approval for a new origin.
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

// StabilizeExecutable publishes an immutable-by-mode, content-addressed worker
// beneath the caller's private root. Published entries are reused only after a
// complete byte verification; cleanup is intentionally a no-op because cache
// entries are bounded by distinct executable digests.
func StabilizeExecutable(source, privateRoot string) (string, func(), error) {
	return stabilizeExecutable(source, privateRoot)
}

func stabilizeExecutable(source, privateRoot string) (string, func(), error) {
	info, err := os.Lstat(source)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
		return "", nil, errors.New("iCoT executable is not a safe executable file")
	}
	cacheRoot := filepath.Join(privateRoot, ".openudon-browser-worker-cache")
	if err := ensureWorkerCache(cacheRoot); err != nil {
		return "", nil, err
	}
	sweepStaleWorkerTemps(cacheRoot, time.Now())
	input, err := os.Open(source)
	if err != nil {
		return "", nil, err
	}
	defer input.Close()
	openedBefore, err := input.Stat()
	if err != nil || !os.SameFile(info, openedBefore) || openedBefore.Size() > 256<<20 {
		return "", nil, errors.New("iCoT executable identity changed before stabilization")
	}
	sourceDigestBefore, err := hashExecutable(input, 256<<20)
	if err != nil {
		return "", nil, errors.New("could not hash iCoT executable before stabilization")
	}
	hashedBefore, err := input.Stat()
	if err != nil || !sameExecutableState(openedBefore, hashedBefore) {
		return "", nil, errors.New("iCoT executable changed while hashing before stabilization")
	}
	name := "worker-" + sourceDigestBefore
	if strings.EqualFold(filepath.Ext(source), ".exe") {
		name += ".exe"
	}
	target := filepath.Join(cacheRoot, name)
	if _, err := os.Lstat(target); err == nil {
		if err := verifyCachedExecutable(target, sourceDigestBefore); err != nil {
			return "", nil, err
		}
		return target, func() {}, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", nil, errors.New("could not inspect cached browser worker")
	}
	output, err := os.CreateTemp(cacheRoot, ".openudon-worker-copy-")
	if err != nil {
		return "", nil, err
	}
	temporary := output.Name()
	cleanupTemporary := func() { _ = os.Remove(temporary) }
	written, copyErr := copyStabilizedExecutable(output, io.LimitReader(input, (256<<20)+1))
	syncErr, closeErr := output.Sync(), output.Close()
	openedAfter, statErr := input.Stat()
	sourceDigestAfter, digestErr := hashExecutable(input, 256<<20)
	hashedAfter, hashedStatErr := input.Stat()
	pathAfter, pathErr := os.Lstat(source)
	if copyErr != nil || syncErr != nil || closeErr != nil || statErr != nil || pathErr != nil ||
		digestErr != nil || hashedStatErr != nil || sourceDigestBefore != sourceDigestAfter ||
		written != info.Size() || written > 256<<20 || !os.SameFile(openedBefore, openedAfter) || !os.SameFile(openedBefore, pathAfter) ||
		!sameExecutableState(openedBefore, openedAfter) || !sameExecutableState(openedAfter, hashedAfter) {
		cleanupTemporary()
		return "", nil, errors.New("could not stabilize iCoT browser worker executable")
	}
	if err := os.Chmod(temporary, 0o500); err != nil {
		cleanupTemporary()
		return "", nil, err
	}
	if err := verifyCachedExecutable(temporary, sourceDigestBefore); err != nil {
		cleanupTemporary()
		return "", nil, err
	}
	if err := os.Link(temporary, target); err != nil && !errors.Is(err, os.ErrExist) {
		cleanupTemporary()
		return "", nil, errors.New("could not publish stabilized browser worker")
	}
	cleanupTemporary()
	if err := verifyCachedExecutable(target, sourceDigestBefore); err != nil {
		return "", nil, err
	}
	return target, func() {}, nil
}

func ensureWorkerCache(path string) error {
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || info.Mode().Perm() != 0o700 {
		return errors.New("browser worker cache must be a mode-0700 non-symlink directory")
	}
	return nil
}

func verifyCachedExecutable(path, digest string) error {
	before, err := os.Lstat(path)
	if err != nil || before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() || before.Mode().Perm() != 0o500 || before.Size() > 256<<20 {
		return errors.New("cached browser worker is unsafe")
	}
	file, err := os.Open(path)
	if err != nil {
		return errors.New("could not open cached browser worker")
	}
	actual, digestErr := hashExecutable(file, 256<<20)
	after, statErr := file.Stat()
	closeErr := file.Close()
	pathAfter, pathErr := os.Lstat(path)
	if digestErr != nil || statErr != nil || closeErr != nil || pathErr != nil || actual != digest || !sameExecutableState(before, after) || !sameExecutableState(after, pathAfter) || pathAfter.Mode().Perm() != 0o500 {
		return errors.New("cached browser worker content could not be verified")
	}
	return nil
}

func sweepStaleWorkerTemps(cacheRoot string, now time.Time) {
	entries, err := os.ReadDir(cacheRoot)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), ".openudon-worker-copy-") {
			continue
		}
		path := filepath.Join(cacheRoot, entry.Name())
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || now.Sub(info.ModTime()) < 24*time.Hour {
			continue
		}
		_ = os.Remove(path)
	}
}

func hashExecutable(file *os.File, maxBytes int64) (string, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	hash := sha256.New()
	written, err := io.Copy(hash, io.LimitReader(file, maxBytes+1))
	if err != nil {
		return "", err
	}
	if written > maxBytes {
		return "", errors.New("executable exceeds size bound")
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func sameExecutableState(left, right os.FileInfo) bool {
	return left != nil && right != nil && left.Mode().IsRegular() && right.Mode().IsRegular() &&
		left.Mode() == right.Mode() && left.Size() == right.Size() && left.ModTime().Equal(right.ModTime()) && os.SameFile(left, right)
}
