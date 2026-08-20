package icot

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
)

// BrowserScenarioOutput is one fixture-owned, human-equivalent output choice.
// Candidate IDs remain Browsertools-owned and are resolved from the current
// reduced observation inside RunBrowserScenarioAuthor.
type BrowserScenarioOutput struct {
	Key         string
	Type        string
	Role        string
	Name        string
	LocatorMode string
}

// BrowserScenarioAuthorRequest is a deliberately narrow loopback-only seam for
// release qualification. It cannot accept arbitrary plans, selectors, target
// origins, credential values, or model output.
type BrowserScenarioAuthorRequest struct {
	BrowsertoolsPath  string
	DriverDir         string
	ExampleDir        string
	PrivateRoot       string
	InitialURL        string
	AuthenticationURL string
	GoalURL           string
	GoalContext       string
	GoalRole          string
	GoalLabel         string
	ChallengeKind     string
	ContextMode       string
	Outputs           []BrowserScenarioOutput
	Fault             string
	Now               time.Time
}

// BrowserScenarioAuthorResult contains profile/review facts only. It contains
// no protocol transcript, page content, input value, cookie, or browser state.
type BrowserScenarioAuthorResult struct {
	Rejected                 bool
	FailureClass             string
	AuthenticationPath       string
	CapabilityPath           string
	AuthenticationProfile    string
	CapabilityProfile        string
	ReviewedChallengeKind    string
	ReviewedOutputKeys       []string
	CredentialSlotKinds      map[string]string
	EnvelopeDigest           string
	PrivateEnvelopePreserved bool
}

// RunBrowserScenarioAuthor drives the production v2 protocol with choices
// derived only from an embedded loopback manifest. The same OpenUdon envelope
// validation, profile reconstruction, review generation, and atomic staging
// used by interactive authoring are retained.
func RunBrowserScenarioAuthor(ctx context.Context, request BrowserScenarioAuthorRequest) (BrowserScenarioAuthorResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := validateBrowserScenarioAuthorRequest(request); err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	cfg := liveAuthorConfig{
		ExampleDir: request.ExampleDir, Browsertools: request.BrowsertoolsPath, DriverDir: request.DriverDir,
		URL: request.InitialURL, DashboardURL: request.AuthenticationURL, GoalURL: request.GoalURL, Goal: "qualify deterministic browser scenario",
		Origins: []string{scenarioOrigin(request.InitialURL)}, PrivateRoot: request.PrivateRoot,
		ProfileID: "scenario", AfterAuthentication: "continue_current_page",
		GoalRole: request.GoalRole, GoalLabel: request.GoalLabel, GoalContext: request.GoalContext,
		NoLLM: true,
	}
	if err := normalizeLiveAuthorConfig(&cfg); err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	result, rejected, failureClass, err := runBrowserScenarioProtocol(ctx, cfg, request)
	if err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	if rejected {
		return BrowserScenarioAuthorResult{Rejected: true, FailureClass: failureClass}, nil
	}
	prepared, err := prepareAuthenticatedAuthoringImport(cfg, result, request.Now.UTC())
	if err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	if err := stageAuthenticatedAuthoringImport(prepared); err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	envelopeData, _, err := readStablePrivateAuthorResult(result.ArtifactPath, cfg.PrivateRoot)
	if err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	envelope, err := decodeAuthenticatedAuthoringEnvelope(envelopeData)
	if err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	authentication, err := authProfileSummary(envelope.AuthenticationProfile)
	if err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	keys := make([]string, 0, len(envelope.OutputSelections))
	for _, selection := range envelope.OutputSelections {
		keys = append(keys, selection.Key)
	}
	sort.Strings(keys)
	return BrowserScenarioAuthorResult{
		AuthenticationPath:    filepath.Join(cfg.ExampleDir, prepared.AuthenticationTarget),
		CapabilityPath:        filepath.Join(cfg.ExampleDir, prepared.CapabilityTarget),
		AuthenticationProfile: prepared.AuthenticationSchema, CapabilityProfile: prepared.CapabilitySchema,
		ReviewedChallengeKind: request.ChallengeKind, ReviewedOutputKeys: keys,
		CredentialSlotKinds: authentication, EnvelopeDigest: prepared.EnvelopeDigest,
		PrivateEnvelopePreserved: true,
	}, nil
}

func validateBrowserScenarioAuthorRequest(request BrowserScenarioAuthorRequest) error {
	if request.Now.IsZero() || request.ContextMode != "main" && request.ContextMode != "popup" && request.ContextMode != "frame" ||
		request.GoalContext == "" || request.GoalRole == "" || request.GoalLabel == "" || len(request.Outputs) > 17 {
		return fmt.Errorf("browser scenario author request is invalid")
	}
	for _, value := range []string{request.InitialURL, request.AuthenticationURL, request.GoalURL} {
		if !strings.HasPrefix(value, "http://127.0.0.1:") || strings.ContainsAny(value, "?#") {
			return fmt.Errorf("browser scenario author target must be clean loopback HTTP")
		}
	}
	if scenarioOrigin(request.InitialURL) == "" || scenarioOrigin(request.InitialURL) != scenarioOrigin(request.AuthenticationURL) || scenarioOrigin(request.InitialURL) != scenarioOrigin(request.GoalURL) {
		return fmt.Errorf("browser scenario author origins disagree")
	}
	allowedChallenges := map[string]bool{"": true, "totp": true, "sms_otp": true, "email_otp": true, "voice_otp": true, "push": true, "push_number_match": true, "passkey": true, "security_key": true}
	allowedFaults := map[string]bool{"": true, "outputs_17": true, "stale_candidate": true, "ambiguous_unique_role": true, "context_substitution": true, "invalid_scalars": true, "secret_output": true, "origin_escape": true}
	if !allowedChallenges[request.ChallengeKind] || !allowedFaults[request.Fault] {
		return fmt.Errorf("browser scenario author contract is unknown")
	}
	return nil
}

func scenarioOrigin(raw string) string {
	origin, _ := originAndPath(raw)
	return origin
}

func runBrowserScenarioProtocol(ctx context.Context, cfg liveAuthorConfig, request BrowserScenarioAuthorRequest) (liveProtocolResult, bool, string, error) {
	deps := defaultLiveAuthorDependencies()
	executable, cleanupExecutable, err := deps.PrepareExecutable(cfg.Browsertools, cfg.PrivateRoot)
	if err != nil {
		return liveProtocolResult{}, false, "", err
	}
	defer cleanupExecutable()
	child, err := deps.StartProcess(ctx, executable, scenarioBrowsertoolsArgs(cfg), minimalBrowsertoolsEnvironment())
	if err != nil {
		return liveProtocolResult{}, false, "", fmt.Errorf("start Browsertools: %w", err)
	}
	protocol := newLiveProtocol(child.Input(), child.Output())
	finished := false
	defer func() {
		if finished {
			return
		}
		_ = protocol.send(liveClientMessage{Type: "close"})
		_ = child.Input().Close()
		_ = child.Kill()
		_ = child.Wait()
	}()
	hello, err := protocol.receive()
	if err != nil || hello.Type != "hello" {
		return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools v2 negotiation failed")
	}
	for _, capability := range []string{"chromium", "human_credentials", "reviewed_mfa_kind", "reviewed_outputs", "reduced_observation", "popup", "frame", "typed_goal"} {
		if !containsExact(hello.Capabilities, capability) {
			return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools lacks scenario capability %s", capability)
		}
	}
	goalOrigin, goalPath := originAndPath(cfg.GoalURL)
	bounds := defaultLiveAuthorBounds()
	start := liveClientMessage{
		Type: "start", Title: defaultLiveTitle(cfg), URL: cfg.URL, DashboardURL: cfg.DashboardURL,
		Goal: cfg.Goal, Origins: append([]string(nil), cfg.Origins...), Bounds: &bounds,
		GoalPredicate: &liveGoalPredicate{Origin: goalOrigin, Path: goalPath, Context: cfg.GoalContext, Role: cfg.GoalRole, Label: cfg.GoalLabel},
	}
	if err := protocol.send(start); err != nil {
		return liveProtocolResult{}, false, "", err
	}
	protocol.setCandidateCeiling(bounds.MaxCandidates)
	state, err := protocol.receive()
	if err != nil || state.Type != "state" || state.Phase != "authentication" || state.Context != "main" || state.Bounds == nil || *state.Bounds != bounds {
		code := ""
		if state.Diagnostic != nil {
			code = state.Diagnostic.Code
		}
		return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools initial scenario state is invalid (type=%s phase=%s context=%s bounds=%t receive=%t diagnostic=%s)", state.Type, state.Phase, state.Context, state.Bounds != nil, err == nil, code)
	}
	observation, _, err := scenarioObserve(protocol, "main", cfg, request.GoalURL)
	if err != nil {
		return liveProtocolResult{}, false, "", err
	}
	for _, input := range []struct{ role, label string }{{"textbox", "Email address"}, {"textbox", "Password"}} {
		candidate, findErr := scenarioCandidate(observation, input.role, input.label)
		if findErr != nil {
			return liveProtocolResult{}, false, "", findErr
		}
		if err := scenarioHumanInput(protocol, candidate.ID, "", nil); err != nil {
			return liveProtocolResult{}, false, "", err
		}
	}
	login, err := scenarioCandidate(observation, "button", "Sign in")
	if err != nil {
		return liveProtocolResult{}, false, "", err
	}
	currentContext, err := scenarioClick(protocol, login.ID, 1, "authentication")
	if err != nil {
		return liveProtocolResult{}, false, "", err
	}
	observation, goalReached, err := scenarioObserve(protocol, currentContext, cfg, request.GoalURL)
	if err != nil {
		return liveProtocolResult{}, false, "", err
	}
	if request.ChallengeKind != "" {
		role, label := "status", "Approve verification request"
		if containsExact(liveOTPChallengeKinds, request.ChallengeKind) {
			role, label = "textbox", "Verification code"
		}
		candidate, findErr := scenarioCandidate(observation, role, label)
		if findErr != nil {
			return liveProtocolResult{}, false, "", findErr
		}
		compatible := liveMFAChallengeKinds
		if role == "textbox" {
			compatible = liveOTPChallengeKinds
		}
		if err := scenarioHumanInput(protocol, candidate.ID, request.ChallengeKind, compatible); err != nil {
			return liveProtocolResult{}, false, "", err
		}
		buttonName := "Continue"
		if role == "textbox" {
			buttonName = "Verify"
		}
		button, findErr := scenarioCandidate(observation, "button", buttonName)
		if findErr != nil {
			return liveProtocolResult{}, false, "", findErr
		}
		currentContext, err = scenarioClick(protocol, button.ID, 1, "authentication")
		if err != nil {
			return liveProtocolResult{}, false, "", err
		}
		observation, goalReached, err = scenarioObserve(protocol, currentContext, cfg, request.GoalURL)
		if err != nil {
			return liveProtocolResult{}, false, "", err
		}
	}
	if request.ContextMode == "popup" {
		if goalReached {
			return liveProtocolResult{}, false, "", fmt.Errorf("popup scenario reached the goal before its reviewed click")
		}
		open, findErr := scenarioCandidate(observation, "link", "Open member report")
		if findErr != nil {
			return liveProtocolResult{}, false, "", findErr
		}
		currentContext, err = scenarioClick(protocol, open.ID, 0, "exploration")
		if err != nil {
			return liveProtocolResult{}, false, "", err
		}
		observation, goalReached, err = scenarioObserve(protocol, currentContext, cfg, request.GoalURL)
		if err != nil {
			return liveProtocolResult{}, false, "", err
		}
	}
	if request.ContextMode == "frame" {
		if goalReached {
			return liveProtocolResult{}, false, "", fmt.Errorf("frame scenario reached the goal in the wrong context")
		}
		if _, ok := observation.Contexts["frame_1"]; !ok {
			return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools did not discover frame_1")
		}
		observation, goalReached, err = scenarioObserve(protocol, "frame_1", cfg, request.GoalURL)
		if err != nil {
			return liveProtocolResult{}, false, "", err
		}
	}
	if !goalReached {
		return liveProtocolResult{}, false, "", fmt.Errorf("browser scenario goal was not observed")
	}
	checkpoint, err := protocol.receive()
	if err != nil || checkpoint.Type != "human_checkpoint" || checkpoint.Checkpoint.Kind != "completion" {
		return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools completion checkpoint is invalid")
	}
	selectionObservation := observation
	if request.Fault == "stale_candidate" {
		observation, goalReached, err = scenarioObserve(protocol, observation.Context, cfg, request.GoalURL)
		if err != nil || !goalReached {
			return liveProtocolResult{}, false, "", fmt.Errorf("stale-candidate refresh failed")
		}
		checkpoint, err = protocol.receive()
		if err != nil || checkpoint.Type != "human_checkpoint" || checkpoint.Checkpoint.Kind != "completion" {
			return liveProtocolResult{}, false, "", fmt.Errorf("stale-candidate completion checkpoint is invalid")
		}
	}
	outputs, validationErr := scenarioOutputRequests(request.Outputs, selectionObservation, observation, bounds.MaxOutputs)
	if validationErr != nil {
		failure := scenarioAuthorFailure(request.Fault)
		if failure == "" {
			return liveProtocolResult{}, false, "", validationErr
		}
		_ = protocol.send(liveClientMessage{Type: "close"})
		_ = child.Input().Close()
		_ = child.Wait()
		finished = true
		return liveProtocolResult{}, true, failure, nil
	}
	if err := protocol.send(liveClientMessage{Type: "human_complete", Confirmed: true, Outputs: &outputs}); err != nil {
		return liveProtocolResult{}, false, "", err
	}
	completed, err := protocol.receive()
	if err != nil || completed.Type != "state" || completed.Phase != "completed" {
		return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools did not complete scenario authoring")
	}
	if err := protocol.send(liveClientMessage{Type: "finish"}); err != nil {
		return liveProtocolResult{}, false, "", err
	}
	message, err := protocol.receive()
	if err != nil || message.Type != "result" || message.Result == nil {
		return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools did not return a v2 authoring result")
	}
	_ = child.Input().Close()
	if err := child.Wait(); err != nil {
		return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario process failed")
	}
	finished = true
	return *message.Result, false, "", nil
}

func scenarioBrowsertoolsArgs(cfg liveAuthorConfig) []string {
	args := []string{"author-session", "chromium", "--private-root", cfg.PrivateRoot}
	if cfg.DriverDir != "" {
		args = append(args, "--driver-dir", cfg.DriverDir)
	}
	return args
}

func scenarioObserve(protocol *liveProtocol, contextID string, cfg liveAuthorConfig, goalURL string) (liveObservation, bool, error) {
	if err := protocol.send(liveClientMessage{Type: "observe", Context: contextID}); err != nil {
		return liveObservation{}, false, err
	}
	message, err := protocol.receive()
	if err != nil || message.Type != "observation" || message.Observation == nil {
		diagnostic := ""
		if message.Diagnostic != nil {
			diagnostic = message.Diagnostic.Code
		}
		return liveObservation{}, false, fmt.Errorf("Browsertools scenario observation is invalid (type=%s receive=%t diagnostic=%s)", message.Type, err == nil, diagnostic)
	}
	if err := validateLiveObservationInventory(*message.Observation, cfg.Origins, message.Observation.Contexts); err != nil {
		return liveObservation{}, false, err
	}
	goal := liveGoalPredicate{Origin: scenarioOrigin(goalURL), Path: secondOriginPath(goalURL), Context: cfg.GoalContext, Role: cfg.GoalRole, Label: cfg.GoalLabel}
	return *message.Observation, liveObservationMatchesGoal(*message.Observation, goal), nil
}

func secondOriginPath(raw string) string { _, path := originAndPath(raw); return path }

func scenarioCandidate(observation liveObservation, role, label string) (liveCandidate, error) {
	var matches []liveCandidate
	for _, candidate := range observation.Candidates {
		if candidate.Role == role && (label == "" || candidate.Label == label) {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 || matches[0].Matches != 1 {
		return liveCandidate{}, fmt.Errorf("required reduced scenario candidate is missing or ambiguous")
	}
	return matches[0], nil
}

func scenarioHumanInput(protocol *liveProtocol, candidateID, challengeKind string, compatible []string) error {
	if err := protocol.send(liveClientMessage{Type: "focus_human_input", CandidateID: candidateID}); err != nil {
		return err
	}
	message, err := protocol.receive()
	if err != nil || message.Type != "human_checkpoint" || message.Checkpoint == nil || message.Checkpoint.CandidateID != candidateID {
		return fmt.Errorf("Browsertools human-input checkpoint is invalid")
	}
	if challengeKind == "" {
		if message.Checkpoint.Kind != "credential" || len(message.Checkpoint.ChallengeKinds) != 0 {
			return fmt.Errorf("Browsertools credential checkpoint widened authority")
		}
	} else if message.Checkpoint.Kind != "mfa" || !validLiveChallengeKinds(message.Checkpoint.ChallengeKinds) || !containsExact(message.Checkpoint.ChallengeKinds, challengeKind) || !challengeSubsetOf(message.Checkpoint.ChallengeKinds, compatible) {
		return fmt.Errorf("Browsertools MFA compatibility set is invalid")
	}
	if err := protocol.send(liveClientMessage{Type: "human_input_complete", CandidateID: candidateID, ChallengeKind: challengeKind}); err != nil {
		return err
	}
	state, err := protocol.receive()
	if err != nil || state.Type != "state" || state.Phase != "authentication" {
		return fmt.Errorf("Browsertools human-input completion state is invalid")
	}
	return nil
}

func challengeSubsetOf(values, family []string) bool {
	for _, value := range values {
		if !containsExact(family, value) {
			return false
		}
	}
	return true
}

func scenarioClick(protocol *liveProtocol, candidateID string, postBudget int, expectedPhase string) (string, error) {
	if err := protocol.send(liveClientMessage{Type: "execute", Action: "click", CandidateID: candidateID, POSTBudget: postBudget}); err != nil {
		return "", err
	}
	approval, err := protocol.receive()
	if err != nil || approval.Type != "approval_required" || approval.Approval == nil || approval.Approval.CandidateID != candidateID || approval.Approval.Action != "click" || approval.Approval.POSTBudget != postBudget {
		return "", fmt.Errorf("Browsertools exact action approval is invalid")
	}
	if err := protocol.send(liveClientMessage{Type: "approve", ApprovalID: approval.Approval.ID}); err != nil {
		return "", err
	}
	state, err := protocol.receive()
	if err != nil || state.Type != "state" || state.Phase != expectedPhase || state.Context == "" {
		diagnostic := ""
		if state.Diagnostic != nil {
			diagnostic = state.Diagnostic.Code
		}
		return "", fmt.Errorf("Browsertools approved action state is invalid (type=%s phase=%s context=%s receive=%t diagnostic=%s)", state.Type, state.Phase, state.Context, err == nil, diagnostic)
	}
	return state.Context, nil
}

func scenarioOutputRequests(declarations []BrowserScenarioOutput, selected, current liveObservation, max int) ([]liveOutputRequest, error) {
	if len(declarations) > max || len(declarations) > liveAuthorSelectedMaxOutputs {
		return nil, fmt.Errorf("output selection bound exceeded")
	}
	requests := make([]liveOutputRequest, 0, len(declarations))
	seenKeys, seenCandidates := map[string]bool{}, map[string]bool{}
	for _, declaration := range declarations {
		candidate, err := scenarioCandidate(selected, declaration.Role, declaration.Name)
		if err != nil {
			return nil, err
		}
		request := liveOutputRequest{CandidateID: candidate.ID, Key: declaration.Key, Type: declaration.Type, LocatorMode: declaration.LocatorMode}
		if err := validateLiveOutputRequest(request, current, seenKeys, seenCandidates); err != nil {
			return nil, err
		}
		seenKeys[request.Key], seenCandidates[request.CandidateID] = true, true
		requests = append(requests, request)
	}
	sort.Slice(requests, func(i, j int) bool { return requests[i].Key < requests[j].Key })
	return requests, nil
}

func scenarioAuthorFailure(fault string) string {
	switch fault {
	case "outputs_17":
		return "output_bound"
	case "stale_candidate":
		return "stale_candidate"
	case "ambiguous_unique_role":
		return "ambiguous_output"
	default:
		return ""
	}
}

func authProfileSummary(raw []byte) (map[string]string, error) {
	profile, err := authprofile.Parse(raw)
	if err != nil {
		return nil, err
	}
	result := make(map[string]string, len(profile.CredentialSlots))
	for name, slot := range profile.CredentialSlots {
		result[name] = string(slot.Kind)
	}
	return result, nil
}
