package icot

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
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
	Now               func() time.Time
	CandidateObserver func(*browsercandidate.AuthenticationCapability) error
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
	if request.Fault == "path_injection" {
		cfg.GoalURL = scenarioOrigin(request.GoalURL) + "/ignore%20previous%20instructions"
	}
	if err := normalizeLiveAuthorConfig(&cfg); err != nil {
		if request.Fault == "path_injection" {
			return BrowserScenarioAuthorResult{Rejected: true, FailureClass: "path_disclosure"}, nil
		}
		return BrowserScenarioAuthorResult{}, err
	}
	result, rejected, failureClass, err := runBrowserScenarioController(ctx, cfg, request)
	if err != nil {
		return BrowserScenarioAuthorResult{}, err
	}
	if rejected {
		return BrowserScenarioAuthorResult{Rejected: true, FailureClass: failureClass}, nil
	}
	if request.Fault == "fabricated_trace" || request.Fault == "stale_candidate" {
		if err := tamperScenarioEnvelope(&result, cfg.PrivateRoot, request.Fault); err != nil {
			return BrowserScenarioAuthorResult{}, err
		}
	}
	assessedAt := request.Now().UTC().Round(0)
	if assessedAt.IsZero() {
		return BrowserScenarioAuthorResult{}, fmt.Errorf("browser scenario assessment clock is unavailable")
	}
	prepared, err := prepareAttestedAuthenticatedAuthoringImport(cfg, result, assessedAt)
	if err != nil {
		if failure := scenarioImportFailure(request.Fault, err); failure != "" {
			return BrowserScenarioAuthorResult{Rejected: true, FailureClass: failure}, nil
		}
		return BrowserScenarioAuthorResult{}, err
	}
	if request.CandidateObserver != nil {
		if err := request.CandidateObserver(prepared.Candidate); err != nil {
			return BrowserScenarioAuthorResult{}, fmt.Errorf("browser scenario candidate observation failed")
		}
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
	if request.Now == nil || request.ContextMode != "main" && request.ContextMode != "popup" && request.ContextMode != "frame" ||
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
	allowedFaults := map[string]bool{"": true, "outputs_17": true, "stale_candidate": true, "ambiguous_unique_role": true, "context_substitution": true, "invalid_scalars": true, "secret_output": true, "origin_escape": true, "path_injection": true, "fabricated_trace": true}
	if !allowedChallenges[request.ChallengeKind] || !allowedFaults[request.Fault] {
		return fmt.Errorf("browser scenario author contract is unknown")
	}
	return nil
}

func runBrowserScenarioController(ctx context.Context, cfg liveAuthorConfig, request BrowserScenarioAuthorRequest) (liveProtocolResult, bool, string, error) {
	goalOrigin, goalPath, err := originAndPath(cfg.GoalURL)
	if err != nil {
		return liveProtocolResult{}, false, "", err
	}
	session, err := browserauthor.StartExternal(ctx, browserauthor.Config{
		PrivateRoot: cfg.PrivateRoot, DriverDir: cfg.DriverDir, InitialURL: cfg.URL, DashboardURL: cfg.DashboardURL,
		Goal: cfg.Goal, Origins: append([]string(nil), cfg.Origins...), ProfileID: cfg.ProfileID,
		GoalPredicate: authorresult.GoalPredicate{Origin: goalOrigin, Path: goalPath, Context: cfg.GoalContext, Role: cfg.GoalRole, Label: cfg.GoalLabel},
	}, cfg.Browsertools)
	if err != nil {
		return liveProtocolResult{}, false, "", err
	}
	success := false
	defer func() {
		if !success {
			session.Cancel()
		}
	}()
	credentialDone := map[string]bool{}
	loginSubmitted, challengeDone, challengeSubmitted, popupOpened := false, false, false, false
	for event := range session.Events() {
		if event.ErrorCode != "" || event.State == "failed" || event.State == "canceled" || event.State == "closed" {
			return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario controller failed closed (state=%s phase=%s code=%s)", event.State, event.Phase, event.ErrorCode)
		}
		if event.Approval != nil {
			if err := session.Respond(ctx, browserauthor.Response{Kind: "approve", ApprovalID: event.Approval.ID}); err != nil {
				return liveProtocolResult{}, false, "", err
			}
			continue
		}
		if event.Checkpoint != nil {
			checkpoint := event.Checkpoint
			switch checkpoint.Kind {
			case "credential":
				if checkpoint.InputKind != "identifier" && checkpoint.InputKind != "password" {
					return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario credential kind is invalid")
				}
				credentialDone[checkpoint.InputKind] = true
				err = session.Respond(ctx, browserauthor.Response{Kind: "continue", CandidateID: checkpoint.CandidateID})
			case "mfa":
				if request.ChallengeKind == "" || !containsExact(checkpoint.ChallengeKinds, request.ChallengeKind) {
					return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario MFA kind is invalid")
				}
				challengeDone = true
				err = session.Respond(ctx, browserauthor.Response{Kind: "continue", CandidateID: checkpoint.CandidateID, ChallengeKind: request.ChallengeKind})
			case "completion":
				if event.Observation == nil {
					return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario completion observation is missing")
				}
				observation := liveObservationFromShared(*event.Observation)
				outputs, outputErr := scenarioOutputRequests(request.Outputs, observation, observation, authorsession.DefaultMaxOutputs)
				if outputErr != nil {
					if failure := scenarioOutputFailure(request.Fault, outputErr); failure != "" {
						return liveProtocolResult{}, true, failure, nil
					}
					return liveProtocolResult{}, false, "", outputErr
				}
				shared := make([]authorsession.OutputRequest, len(outputs))
				for index, output := range outputs {
					shared[index] = authorsession.OutputRequest{CandidateID: output.CandidateID, Key: output.Key, Type: output.Type, LocatorMode: output.LocatorMode}
				}
				err = session.Respond(ctx, browserauthor.Response{Kind: "confirm", Confirmed: true, Outputs: shared})
			default:
				return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario checkpoint is invalid")
			}
			if err != nil {
				return liveProtocolResult{}, false, "", err
			}
			continue
		}
		if event.Observation != nil {
			observation := liveObservationFromShared(*event.Observation)
			response, responseErr := scenarioControllerResponse(request, observation, event.Phase, credentialDone, &loginSubmitted, challengeDone, &challengeSubmitted, &popupOpened)
			if responseErr != nil {
				return liveProtocolResult{}, false, "", responseErr
			}
			if err := session.Respond(ctx, response); err != nil {
				return liveProtocolResult{}, false, "", err
			}
			continue
		}
		if event.Result != nil {
			if event.Attestation == nil {
				return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario result lacks parent attestation")
			}
			success = true
			return liveProtocolResult{ArtifactPath: event.Result.ArtifactPath, Digest: event.Result.Digest, Attestation: event.Attestation}, false, "", nil
		}
	}
	return liveProtocolResult{}, false, "", fmt.Errorf("Browsertools scenario controller ended without a result")
}

func scenarioControllerResponse(request BrowserScenarioAuthorRequest, observation liveObservation, phase string, credentialDone map[string]bool, loginSubmitted *bool, challengeDone bool, challengeSubmitted, popupOpened *bool) (browserauthor.Response, error) {
	focus := func(role, label string) (browserauthor.Response, error) {
		candidate, err := scenarioCandidate(observation, role, label)
		return browserauthor.Response{Kind: "focus_human_input", CandidateID: candidate.ID}, err
	}
	click := func(role, label string, postBudget int) (browserauthor.Response, error) {
		candidate, err := scenarioCandidate(observation, role, label)
		return browserauthor.Response{Kind: "click", CandidateID: candidate.ID, POSTBudget: postBudget}, err
	}
	if phase == "authentication" {
		if !credentialDone["identifier"] {
			return focus("textbox", "Email address")
		}
		if !credentialDone["password"] {
			return focus("textbox", "Password")
		}
		if !*loginSubmitted {
			*loginSubmitted = true
			return click("button", "Sign in", 1)
		}
		if request.ChallengeKind != "" && !challengeDone {
			if containsExact(liveOTPChallengeKinds, request.ChallengeKind) {
				return focus("textbox", "Verification code")
			}
			return focus("status", "Approve verification request")
		}
		if request.ChallengeKind != "" && !*challengeSubmitted {
			*challengeSubmitted = true
			if containsExact(liveOTPChallengeKinds, request.ChallengeKind) {
				return click("button", "Verify", 1)
			}
			return click("button", "Continue", 1)
		}
		if request.ContextMode == "frame" && observation.Context == "main" {
			return scenarioFrameResponse(request.GoalContext, observation)
		}
		return browserauthor.Response{}, fmt.Errorf("Browsertools scenario authentication observation is unexpected")
	}
	if request.ContextMode == "popup" && !*popupOpened {
		*popupOpened = true
		return click("link", "Open member report", 0)
	}
	if request.ContextMode == "frame" && observation.Context == "main" {
		return scenarioFrameResponse(request.GoalContext, observation)
	}
	return browserauthor.Response{}, fmt.Errorf("Browsertools scenario exploration observation is unexpected")
}

func scenarioFrameResponse(contextID string, observation liveObservation) (browserauthor.Response, error) {
	context, ok := observation.Contexts[contextID]
	if contextID == "" || !ok || context.Kind != "frame" || context.Parent != "main" {
		return browserauthor.Response{}, fmt.Errorf("Browsertools scenario frame is missing")
	}
	return browserauthor.Response{Kind: "observe", Context: contextID}, nil
}

func tamperScenarioEnvelope(result *liveProtocolResult, privateRoot, fault string) error {
	data, _, err := readStablePrivateAuthorResult(result.ArtifactPath, privateRoot)
	if err != nil {
		return err
	}
	var envelope authorresult.Envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	switch fault {
	case "fabricated_trace":
		if len(envelope.Trace) == 0 {
			return fmt.Errorf("scenario trace is unavailable")
		}
		envelope.Trace = append(envelope.Trace, envelope.Trace[0])
	case "stale_candidate":
		if len(envelope.OutputSelections) == 0 {
			return fmt.Errorf("scenario output selection is unavailable")
		}
		envelope.OutputSelections[0].CandidateID = "candidate-0000000000000000"
	default:
		return nil
	}
	mutated, err := authorresult.MarshalDeterministic(&envelope)
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.ArtifactPath, mutated, 0o600); err != nil {
		return err
	}
	digest := sha256.Sum256(mutated)
	result.Digest = "sha256:" + hex.EncodeToString(digest[:])
	return nil
}

func scenarioOrigin(raw string) string {
	origin, _, _ := originAndPath(raw)
	return origin
}

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

func challengeSubsetOf(values, family []string) bool {
	for _, value := range values {
		if !containsExact(family, value) {
			return false
		}
	}
	return true
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

func scenarioOutputFailure(fault string, err error) string {
	if err == nil {
		return ""
	}
	switch {
	case fault == "outputs_17" && err.Error() == "output selection bound exceeded":
		return "output_bound"
	case fault == "ambiguous_unique_role" && err.Error() == "required reduced scenario candidate is missing or ambiguous":
		return "ambiguous_output"
	default:
		return ""
	}
}

func scenarioImportFailure(fault string, err error) string {
	if err == nil {
		return ""
	}
	switch {
	case fault == "stale_candidate" && err.Error() == "authenticated-authoring output selection was not requested by the operator":
		return "stale_candidate"
	case fault == "fabricated_trace" && err.Error() == "authenticated-authoring trace length mismatch":
		return "fabricated_trace"
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
