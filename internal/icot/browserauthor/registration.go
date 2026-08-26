package browserauthor

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/browsertools/disclosurepath"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

const maxRegistrationProtocolLine = registrationauthorsession.MaxProtocolLineBytes

var (
	registrationCandidateID   = regexp.MustCompile(`^candidate-[0-9a-f]{16}$`)
	registrationTransactionID = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	registrationSymbol        = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)
)

var registrationAssessmentClock = time.Now

// RegistrationConfig fixes the process and transaction authority before the
// separate no-submit worker is launched.
type RegistrationConfig struct {
	PrivateRoot   string
	DriverDir     string
	TransactionID string
	OperatorIdle  time.Duration
	Absolute      time.Duration
}

// RegistrationCommand is the closed parent-side command union. Confirmed is
// consumed by OpenUdon and never serialized to the Browsertools worker.
type RegistrationCommand struct {
	Type               string
	ProfileID          string
	URL                string
	Origins            []string
	Bounds             *registrationauthorsession.Bounds
	Method             string
	Profile            []byte
	CandidateIDs       []string
	Flow               string
	CleanupDisposition string
	Confirmed          bool
	CredentialBindings []browsertransaction.CredentialBinding
}

// RegistrationEvent exposes only the strict, value-free protocol surface and
// the final adopted candidate. It has no private-result locator or raw output.
type RegistrationEvent struct {
	State       string
	Phase       string
	Bounds      *registrationauthorsession.Bounds
	Observation *registrationauthorsession.Observation
	Diagnostic  string
	Candidate   *browsercandidate.Registration
	ErrorCode   string
}

// RegistrationSession owns one isolated worker process.
type RegistrationSession struct {
	mu       sync.Mutex
	cancel   context.CancelFunc
	commands chan RegistrationCommand
	events   chan RegistrationEvent
	done     chan struct{}
	closed   bool
}

// StartExternalRegistration runs an explicitly selected Browsertools binary
// through the same stabilized, process-group-contained controller.
func StartExternalRegistration(ctx context.Context, config RegistrationConfig, executable string) (*RegistrationSession, error) {
	if ctx == nil {
		return nil, errors.New("registration author context is required")
	}
	config, inbox, err := normalizeRegistrationConfig(config)
	if err != nil {
		return nil, err
	}
	stable, cleanup, err := stabilizeExecutable(strings.TrimSpace(executable), config.PrivateRoot)
	if err != nil {
		_ = inbox.Close()
		return nil, err
	}
	args := []string{stable, "registration-author-session", "chromium", "--private-root", config.PrivateRoot}
	if config.DriverDir != "" {
		args = append(args, "--driver-dir", config.DriverDir)
	}
	return startRegistrationProcess(ctx, config, inbox, args, cleanup)
}

func startRegistrationProcess(ctx context.Context, config RegistrationConfig, inbox *browsercandidate.PrivateInbox, args []string, cleanup func()) (*RegistrationSession, error) {
	bounded, cancel := context.WithTimeout(ctx, config.Absolute)
	child, err := processgroup.StartInteractive(bounded, args, minimalEnvironment(), io.Discard)
	if err != nil {
		cancel()
		cleanup()
		_ = inbox.Close()
		return nil, errors.New("start isolated registration worker")
	}
	session := &RegistrationSession{
		cancel: cancel, commands: make(chan RegistrationCommand), events: make(chan RegistrationEvent, 2), done: make(chan struct{}),
	}
	go session.run(bounded, config, inbox, child, cleanup)
	return session, nil
}

// Events yields value-free state until the worker terminates.
func (session *RegistrationSession) Events() <-chan RegistrationEvent { return session.events }

// Send submits one typed no-submit command.
func (session *RegistrationSession) Send(ctx context.Context, command RegistrationCommand) error {
	if session == nil {
		return errors.New("registration author session is unavailable")
	}
	session.mu.Lock()
	closed := session.closed
	session.mu.Unlock()
	if closed {
		return errors.New("registration author session is closed")
	}
	select {
	case session.commands <- cloneRegistrationCommand(command):
		return nil
	case <-session.done:
		return errors.New("registration author session is closed")
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Cancel terminates the complete worker process tree.
func (session *RegistrationSession) Cancel() {
	if session != nil {
		session.cancel()
	}
}

type registrationRunState struct {
	phase           string
	started         bool
	profileID       string
	origins         []string
	bounds          registrationauthorsession.Bounds
	generation      int
	observations    int
	minimumRequests int
	observation     *registrationauthorsession.Observation
	review          *browsercandidate.RegistrationReview
	bindings        []browsertransaction.CredentialBinding
}

func (session *RegistrationSession) run(ctx context.Context, config RegistrationConfig, inbox *browsercandidate.PrivateInbox, child *processgroup.InteractiveChild, cleanup func()) {
	defer cleanup()
	defer inbox.Close()
	defer close(session.events)
	defer close(session.done)
	defer func() {
		session.mu.Lock()
		session.closed = true
		session.mu.Unlock()
	}()
	waited := false
	defer func() {
		_ = child.Input().Close()
		if !waited {
			if err := child.Terminate(); errors.Is(err, processgroup.ErrTerminationTimeout) {
				session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_teardown"})
			}
		}
	}()
	messages := make(chan registrationauthorsession.ServerMessage)
	readDone := make(chan error, 1)
	go scanRegistrationMessages(ctx, child.Output(), messages, readDone)
	first, err := receiveRegistration(ctx, messages, readDone)
	if err != nil || !validRegistrationHello(first) {
		session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "protocol_negotiation"})
		return
	}
	if !session.publish(ctx, RegistrationEvent{State: "ready"}) {
		return
	}
	state := registrationRunState{phase: "awaiting_start"}
	for {
		command, ok := session.awaitRegistrationCommand(ctx, config.OperatorIdle)
		if !ok {
			return
		}
		message, review, err := prepareRegistrationCommand(command, state)
		if err != nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "invalid_response"})
			return
		}
		data, err := json.Marshal(message)
		if err != nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_write"})
			return
		}
		if _, err := child.Input().Write(append(data, '\n')); err != nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_write"})
			return
		}
		response, err := receiveRegistration(ctx, messages, readDone)
		if err != nil {
			session.publishTerminal(registrationReceiveFailure(ctx, err))
			return
		}
		if response.Protocol != registrationauthorsession.Protocol {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "protocol_mismatch"})
			return
		}
		if response.Type == "diagnostic" {
			if response.Diagnostic == nil || !registrationauthorsession.ValidDiagnostic(response.Diagnostic.Code) {
				session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "malformed_diagnostic"})
			} else {
				session.publishTerminal(RegistrationEvent{State: "failed", Diagnostic: response.Diagnostic.Code, ErrorCode: "worker_failed"})
			}
			return
		}
		event, terminal, err := applyRegistrationResponse(&state, command, response, review)
		if err != nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_protocol"})
			return
		}
		if !session.publish(ctx, event) {
			return
		}
		if !terminal {
			continue
		}
		_ = child.Input().Close()
		if err := drainRegistrationOutput(ctx, messages, readDone); err != nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_protocol"})
			return
		}
		waitErr := child.Wait()
		waited = true
		if waitErr != nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_exit"})
			return
		}
		if command.Type == "close" {
			return
		}
		if state.review == nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "review_missing"})
			return
		}
		candidate, err := inbox.AdoptNewRegistration(browsercandidate.AdoptRegistrationRequest{
			TransactionID: config.TransactionID, CredentialBindings: state.bindings,
			Review: *state.review, AssessedAt: registrationAssessmentClock().UTC().Truncate(time.Second),
		})
		if err != nil {
			session.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "candidate_rejected"})
			return
		}
		session.publishTerminal(RegistrationEvent{State: "candidate", Candidate: candidate})
		return
	}
}

func (session *RegistrationSession) awaitRegistrationCommand(ctx context.Context, idle time.Duration) (RegistrationCommand, bool) {
	timer := time.NewTimer(idle)
	defer timer.Stop()
	select {
	case command := <-session.commands:
		return command, true
	case <-timer.C:
		session.publishTerminal(RegistrationEvent{State: "canceled", ErrorCode: "operator_idle_timeout"})
		session.cancel()
		return RegistrationCommand{}, false
	case <-ctx.Done():
		session.publishTerminal(RegistrationEvent{State: "canceled"})
		return RegistrationCommand{}, false
	}
}

func (session *RegistrationSession) publish(ctx context.Context, event RegistrationEvent) bool {
	select {
	case session.events <- event:
		return true
	case <-ctx.Done():
		return false
	}
}

func (session *RegistrationSession) publishTerminal(event RegistrationEvent) {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	select {
	case session.events <- event:
	case <-timer.C:
	}
}

func normalizeRegistrationConfig(config RegistrationConfig) (RegistrationConfig, *browsercandidate.PrivateInbox, error) {
	if config.OperatorIdle <= 0 {
		config.OperatorIdle = DefaultOperatorIdle
	}
	if config.Absolute <= 0 {
		config.Absolute = DefaultAbsolute
	}
	if config.OperatorIdle > DefaultOperatorIdle || config.Absolute > DefaultAbsolute || !registrationTransactionID.MatchString(config.TransactionID) {
		return RegistrationConfig{}, nil, errors.New("registration author configuration is invalid")
	}
	config.DriverDir = strings.TrimSpace(config.DriverDir)
	inbox, err := browsercandidate.OpenPrivateInbox(config.PrivateRoot)
	if err != nil {
		return RegistrationConfig{}, nil, err
	}
	return config, inbox, nil
}

func prepareRegistrationCommand(command RegistrationCommand, state registrationRunState) (registrationauthorsession.ClientMessage, *browsercandidate.RegistrationReview, error) {
	message := registrationauthorsession.ClientMessage{Protocol: registrationauthorsession.Protocol, Type: command.Type}
	switch command.Type {
	case "start":
		if state.started || command.Confirmed || command.Method != "" || len(command.Profile) != 0 || len(command.CandidateIDs) != 0 || len(command.CredentialBindings) != 0 ||
			command.Flow != "" || command.CleanupDisposition != "" || !validRegistrationBounds(expectedRegistrationBounds(command.Bounds)) {
			return message, nil, errors.New("invalid registration start")
		}
		message.ProfileID, message.URL = command.ProfileID, command.URL
		message.Origins = append([]string(nil), command.Origins...)
		if command.Bounds != nil {
			copyBounds := *command.Bounds
			message.Bounds = &copyBounds
		}
	case "observe":
		if !state.started || registrationCommandHasPayload(command) {
			return message, nil, errors.New("invalid registration observation")
		}
	case "navigate":
		if !state.started || command.Confirmed || command.ProfileID != "" || len(command.Origins) != 0 || command.Bounds != nil ||
			len(command.Profile) != 0 || len(command.CandidateIDs) != 0 || len(command.CredentialBindings) != 0 || command.Flow != "" || command.CleanupDisposition != "" {
			return message, nil, errors.New("invalid registration navigation")
		}
		message.Method, message.URL = command.Method, command.URL
	case "review":
		if !state.started || !command.Confirmed || state.generation <= 0 || command.ProfileID != "" || command.URL != "" ||
			len(command.Origins) != 0 || command.Bounds != nil || command.Method != "" || len(command.Profile) == 0 || state.observation == nil {
			return message, nil, errors.New("invalid registration review")
		}
		bindings, err := normalizeRegistrationBindings(command.CredentialBindings)
		if err != nil {
			return message, nil, err
		}
		command.CredentialBindings = bindings
		profile, err := registrationprofile.Parse(command.Profile)
		if err != nil {
			return message, nil, err
		}
		if !bindingsMatchRegistrationProfile(bindings, profile) {
			return message, nil, errors.New("registration author symbolic bindings do not match the reviewed profile")
		}
		canonical, err := registrationprofile.MarshalJSON(profile)
		if err != nil || !bytes.Equal(canonical, command.Profile) {
			return message, nil, errors.New("registration review source must be canonical")
		}
		message.Profile = append(json.RawMessage(nil), canonical...)
		message.CandidateIDs = append([]string(nil), command.CandidateIDs...)
		message.Flow, message.CleanupDisposition = command.Flow, command.CleanupDisposition
		reviewedCandidates, err := selectedRegistrationCandidates(command.CandidateIDs, *state.observation)
		if err != nil {
			return message, nil, err
		}
		review := &browsercandidate.RegistrationReview{
			Confirmed: true, ProfileID: state.profileID, Flow: command.Flow,
			SourceSHA256: registrationDigest(canonical), ReviewedCandidates: reviewedCandidates,
			CleanupDisposition: command.CleanupDisposition, Origins: append([]string(nil), state.origins...),
			Bounds: state.bounds, Observations: state.observations, MinimumRequests: state.minimumRequests,
		}
		return message, review, nil
	case "finish":
		if state.review == nil || !command.Confirmed || registrationCommandHasNonConfirmationPayload(command) {
			return message, nil, errors.New("registration finish requires confirmed review")
		}
	case "close":
		if registrationCommandHasPayload(command) {
			return message, nil, errors.New("invalid registration close")
		}
	default:
		return message, nil, errors.New("unsupported registration command")
	}
	return message, nil, nil
}

func applyRegistrationResponse(state *registrationRunState, command RegistrationCommand, response registrationauthorsession.ServerMessage, review *browsercandidate.RegistrationReview) (RegistrationEvent, bool, error) {
	switch command.Type {
	case "start":
		if response.Type != "state" || response.Phase != "observing" || response.Bounds == nil || *response.Bounds != expectedRegistrationBounds(command.Bounds) {
			return RegistrationEvent{}, false, errors.New("invalid start response")
		}
		state.started, state.phase, state.profileID = true, response.Phase, command.ProfileID
		state.origins = append([]string(nil), command.Origins...)
		sort.Strings(state.origins)
		state.bounds = *response.Bounds
		state.minimumRequests = 1
		return RegistrationEvent{State: "observing", Phase: response.Phase, Bounds: cloneRegistrationBounds(response.Bounds)}, false, nil
	case "navigate":
		if response.Type != "state" || response.Phase != "observing" || response.Bounds != nil {
			return RegistrationEvent{}, false, errors.New("invalid navigation response")
		}
		state.minimumRequests++
		state.observation = nil
		return RegistrationEvent{State: "observing", Phase: response.Phase}, false, nil
	case "observe":
		if response.Type != "observation" || response.Observation == nil || !safeRegistrationObservation(*response.Observation, *state) {
			return RegistrationEvent{}, false, errors.New("invalid observation response")
		}
		state.generation = response.Observation.Generation
		state.observations++
		observation := cloneRegistrationObservation(*response.Observation)
		state.observation = &observation
		return RegistrationEvent{State: "observation", Phase: state.phase, Observation: &observation}, false, nil
	case "review":
		if response.Type != "state" || response.Phase != "reviewed" || response.Bounds != nil || review == nil {
			return RegistrationEvent{}, false, errors.New("invalid review response")
		}
		bindings, err := normalizeRegistrationBindings(command.CredentialBindings)
		if err != nil {
			return RegistrationEvent{}, false, err
		}
		state.phase, state.review, state.bindings = response.Phase, review, bindings
		return RegistrationEvent{State: "reviewed", Phase: response.Phase}, false, nil
	case "finish", "close":
		if response.Type != "state" || response.Phase != "closed" || response.Bounds != nil {
			return RegistrationEvent{}, false, errors.New("invalid close response")
		}
		state.phase = response.Phase
		return RegistrationEvent{State: "closed", Phase: response.Phase}, true, nil
	default:
		return RegistrationEvent{}, false, errors.New("unsupported registration response")
	}
}

func validRegistrationHello(message registrationauthorsession.ServerMessage) bool {
	want := []string{"get_head_only", "no_submit", "reduced_observation", "registration_review"}
	return message.Protocol == registrationauthorsession.Protocol && message.Type == "hello" && equalRegistrationStrings(message.Capabilities, want)
}

func safeRegistrationObservation(observation registrationauthorsession.Observation, state registrationRunState) bool {
	if observation.Generation != state.observations+1 || observation.Generation <= state.generation || observation.Generation > state.bounds.MaxObservations ||
		len(observation.Candidates) > state.bounds.MaxCandidates || disclosurepath.Validate(observation.Path) != nil ||
		!containsRegistrationString(state.origins, observation.Origin) || len(observation.Diagnostics) > registrationauthorsession.MaxUniqueDiagnostics {
		return false
	}
	seen := map[string]bool{}
	for _, candidate := range observation.Candidates {
		if seen[candidate.ID] || !registrationCandidateID.MatchString(candidate.ID) || candidate.Matches <= 0 || candidate.Matches > state.bounds.MaxCandidates ||
			!portableRoles[candidate.Role] || len(candidate.Label) > 256 || authorsession.ReduceAccessibilityLabel(candidate.Label).Value != candidate.Label {
			return false
		}
		seen[candidate.ID] = true
	}
	for index, code := range observation.Diagnostics {
		if !registrationauthorsession.ValidDiagnostic(code) || index > 0 && observation.Diagnostics[index-1] >= code {
			return false
		}
	}
	return true
}

func scanRegistrationMessages(ctx context.Context, reader io.Reader, output chan<- registrationauthorsession.ServerMessage, done chan<- error) {
	defer close(output)
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), maxRegistrationProtocolLine)
	for scanner.Scan() {
		message, err := decodeRegistrationServerMessage(scanner.Bytes())
		if err != nil {
			done <- err
			return
		}
		select {
		case output <- message:
		case <-ctx.Done():
			done <- ctx.Err()
			return
		}
	}
	done <- scanner.Err()
}

func decodeRegistrationServerMessage(data []byte) (registrationauthorsession.ServerMessage, error) {
	var fields map[string]json.RawMessage
	if err := evidencefile.DecodeStrict(data, &fields); err != nil {
		return registrationauthorsession.ServerMessage{}, err
	}
	var header struct {
		Protocol string `json:"protocol"`
		Type     string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil || header.Protocol != registrationauthorsession.Protocol {
		return registrationauthorsession.ServerMessage{}, errors.New("registration protocol header is invalid")
	}
	allowedByType := map[string]map[string]bool{
		"hello":       {"protocol": true, "type": true, "capabilities": true},
		"state":       {"protocol": true, "type": true, "phase": true, "bounds": true},
		"observation": {"protocol": true, "type": true, "observation": true},
		"diagnostic":  {"protocol": true, "type": true, "diagnostic": true},
	}
	allowed, ok := allowedByType[header.Type]
	if !ok {
		return registrationauthorsession.ServerMessage{}, errors.New("registration protocol type is invalid")
	}
	for name := range fields {
		if !allowed[name] {
			return registrationauthorsession.ServerMessage{}, errors.New("registration protocol shape is invalid")
		}
	}
	var message registrationauthorsession.ServerMessage
	if err := evidencefile.DecodeStrict(data, &message); err != nil {
		return registrationauthorsession.ServerMessage{}, err
	}
	return message, nil
}

func receiveRegistration(ctx context.Context, messages <-chan registrationauthorsession.ServerMessage, done <-chan error) (registrationauthorsession.ServerMessage, error) {
	select {
	case message, ok := <-messages:
		if !ok {
			return registrationauthorsession.ServerMessage{}, io.EOF
		}
		return message, nil
	case err := <-done:
		if err == nil {
			return registrationauthorsession.ServerMessage{}, io.EOF
		}
		return registrationauthorsession.ServerMessage{}, err
	case <-ctx.Done():
		return registrationauthorsession.ServerMessage{}, ctx.Err()
	}
}

func drainRegistrationOutput(ctx context.Context, messages <-chan registrationauthorsession.ServerMessage, done <-chan error) error {
	for {
		select {
		case _, ok := <-messages:
			if ok {
				return errors.New("registration worker emitted output after close")
			}
			select {
			case err := <-done:
				return err
			default:
				return nil
			}
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func registrationReceiveFailure(ctx context.Context, err error) RegistrationEvent {
	if ctx.Err() != nil {
		return RegistrationEvent{State: "canceled"}
	}
	if errors.Is(err, io.EOF) {
		return RegistrationEvent{State: "failed", ErrorCode: "worker_exit"}
	}
	return RegistrationEvent{State: "failed", ErrorCode: "worker_protocol"}
}

func cloneRegistrationCommand(command RegistrationCommand) RegistrationCommand {
	command.Origins = append([]string(nil), command.Origins...)
	command.Profile = append([]byte(nil), command.Profile...)
	command.CandidateIDs = append([]string(nil), command.CandidateIDs...)
	command.CredentialBindings = append([]browsertransaction.CredentialBinding(nil), command.CredentialBindings...)
	command.Bounds = cloneRegistrationBounds(command.Bounds)
	return command
}

func registrationCommandHasPayload(command RegistrationCommand) bool {
	return command.Confirmed || registrationCommandHasNonConfirmationPayload(command)
}

func registrationCommandHasNonConfirmationPayload(command RegistrationCommand) bool {
	return command.ProfileID != "" || command.URL != "" || len(command.Origins) != 0 || command.Bounds != nil ||
		command.Method != "" || len(command.Profile) != 0 || len(command.CandidateIDs) != 0 || command.Flow != "" ||
		command.CleanupDisposition != "" || len(command.CredentialBindings) != 0
}

func normalizeRegistrationBindings(bindings []browsertransaction.CredentialBinding) ([]browsertransaction.CredentialBinding, error) {
	result := append([]browsertransaction.CredentialBinding(nil), bindings...)
	sort.Slice(result, func(i, j int) bool { return result[i].Slot < result[j].Slot })
	if len(result) == 0 || len(result) > 32 {
		return nil, errors.New("registration author symbolic bindings are invalid")
	}
	for index, binding := range result {
		if !registrationSymbol.MatchString(binding.Slot) || !registrationSymbol.MatchString(binding.Binding) ||
			index > 0 && result[index-1].Slot == binding.Slot {
			return nil, errors.New("registration author symbolic bindings are invalid")
		}
	}
	return result, nil
}

func bindingsMatchRegistrationProfile(bindings []browsertransaction.CredentialBinding, profile *registrationprofile.Profile) bool {
	if profile == nil || len(bindings) != len(profile.CredentialSlots) {
		return false
	}
	slots := make([]string, 0, len(profile.CredentialSlots))
	for slot := range profile.CredentialSlots {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	for index := range slots {
		if bindings[index].Slot != slots[index] {
			return false
		}
	}
	return true
}

func selectedRegistrationCandidates(ids []string, observation registrationauthorsession.Observation) ([]registrationauthorsession.ReviewedCandidate, error) {
	if len(ids) == 0 || len(ids) > len(observation.Candidates) || !sort.StringsAreSorted(ids) {
		return nil, errors.New("registration reviewed candidates are invalid")
	}
	byID := make(map[string]registrationauthorsession.Candidate, len(observation.Candidates))
	for _, candidate := range observation.Candidates {
		byID[candidate.ID] = candidate
	}
	result := make([]registrationauthorsession.ReviewedCandidate, 0, len(ids))
	previous := ""
	for _, id := range ids {
		candidate, ok := byID[id]
		if !ok || id <= previous || candidate.Matches != 1 || candidate.Label == "" ||
			candidate.Label == authorsession.RedactedLabel || candidate.Label == authorsession.UntrustedLabel {
			return nil, errors.New("registration reviewed candidates are invalid")
		}
		previous = id
		result = append(result, registrationauthorsession.ReviewedCandidate{
			ID: candidate.ID, Generation: observation.Generation, Role: candidate.Role,
			Label: candidate.Label, Matches: candidate.Matches,
		})
	}
	return result, nil
}

func cloneRegistrationBounds(bounds *registrationauthorsession.Bounds) *registrationauthorsession.Bounds {
	if bounds == nil {
		return nil
	}
	copyBounds := *bounds
	return &copyBounds
}

func expectedRegistrationBounds(bounds *registrationauthorsession.Bounds) registrationauthorsession.Bounds {
	if bounds != nil {
		return *bounds
	}
	return registrationauthorsession.Bounds{
		NavigationTimeoutMS: registrationauthorsession.DefaultNavigationTimeout.Milliseconds(),
		TotalTimeoutMS:      registrationauthorsession.DefaultTotalTimeout.Milliseconds(),
		MaxRequests:         registrationauthorsession.DefaultMaxRequests,
		MaxResponseBytes:    registrationauthorsession.DefaultMaxResponseBytes,
		MaxObservations:     registrationauthorsession.DefaultMaxObservations,
		MaxCandidates:       registrationauthorsession.DefaultMaxCandidates,
	}
}

func validRegistrationBounds(bounds registrationauthorsession.Bounds) bool {
	return bounds.NavigationTimeoutMS > 0 && bounds.NavigationTimeoutMS <= time.Minute.Milliseconds() &&
		bounds.TotalTimeoutMS >= bounds.NavigationTimeoutMS && bounds.TotalTimeoutMS <= (30*time.Minute).Milliseconds() &&
		bounds.MaxRequests > 0 && bounds.MaxRequests <= 4096 &&
		bounds.MaxResponseBytes > 0 && bounds.MaxResponseBytes <= 128<<20 &&
		bounds.MaxObservations > 0 && bounds.MaxObservations <= 256 &&
		bounds.MaxCandidates > 0 && bounds.MaxCandidates <= 512
}

func cloneRegistrationObservation(observation registrationauthorsession.Observation) registrationauthorsession.Observation {
	observation.Candidates = append([]registrationauthorsession.Candidate(nil), observation.Candidates...)
	observation.Diagnostics = append([]string(nil), observation.Diagnostics...)
	return observation
}

func registrationDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func equalRegistrationStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func containsRegistrationString(values []string, target string) bool {
	index := sort.SearchStrings(values, target)
	return index < len(values) && values[index] == target
}
