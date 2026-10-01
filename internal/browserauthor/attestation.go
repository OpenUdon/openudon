package browserauthor

import (
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strings"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/browsertools/disclosurepath"
)

// Attestation is process-private evidence of the exact author-session facts
// driven and reviewed by the OpenUdon parent. All fields are deliberately
// unexported so it has no JSON or persistence representation.
type Attestation struct {
	goal            string
	goalPredicate   authorresult.GoalPredicate
	bounds          authorresult.Bounds
	origins         map[string]struct{}
	contexts        map[string]authorresult.Context
	trace           []attestedTraceStep
	approvals       []authorsession.Approval
	approvalIDs     map[string]struct{}
	observations    int
	diagnostics     map[string]struct{}
	authProof       *authorresult.GoalProof
	completion      *authorsession.Observation
	outputRequests  []authorsession.OutputRequest
	pendingInput    *attestedInput
	pendingAction   *authorsession.ClientMessage
	pendingTrace    int
	dashboardOrigin string
	dashboardPath   string
}

type attestedTraceStep struct {
	step           authorresult.TraceStep
	contextsBefore map[string]authorresult.Context
	contextsAfter  map[string]authorresult.Context
}

type attestedInput struct {
	phase     string
	candidate authorsession.Candidate
	context   string
	inputKind string
}

func newAttestation(config Config, bounds authorresult.Bounds) (*Attestation, error) {
	_, dashboardOrigin, err := cleanURL(config.DashboardURL)
	if err != nil {
		return nil, err
	}
	dashboardPath, err := pathForURL(config.DashboardURL)
	if err != nil {
		return nil, err
	}
	a := &Attestation{
		goal: config.Goal, goalPredicate: config.GoalPredicate, bounds: bounds,
		origins: make(map[string]struct{}, len(config.Origins)), contexts: make(map[string]authorresult.Context),
		approvalIDs: make(map[string]struct{}), diagnostics: make(map[string]struct{}), pendingTrace: -1,
		dashboardOrigin: dashboardOrigin, dashboardPath: dashboardPath,
	}
	for _, origin := range config.Origins {
		a.origins[origin] = struct{}{}
	}
	return a, nil
}

func (a *Attestation) originLedger() []string {
	result := make([]string, 0, len(a.origins))
	for origin := range a.origins {
		result = append(result, origin)
	}
	sort.Strings(result)
	return result
}

func (a *Attestation) recordObservation(phase string, observation authorsession.Observation) error {
	if a == nil {
		return errors.New("parent attestation is unavailable")
	}
	if disclosurepath.Validate(observation.Path) != nil || len(observation.Contexts) > authorsession.MaxContexts || len(observation.Diagnostics) > authorsession.MaxUniqueDiagnostics {
		return errors.New("observation disclosure bounds are invalid")
	}
	for id, previous := range a.contexts {
		if current, ok := observation.Contexts[id]; !ok || current != previous {
			return errors.New("observation context inventory is not additive")
		}
	}
	for _, browserContext := range observation.Contexts {
		if browserContext.Path != "" && disclosurepath.Validate(browserContext.Path) != nil {
			return errors.New("observation context disclosure path is invalid")
		}
	}
	a.contexts = cloneAuthorContexts(observation.Contexts)
	a.observations++
	if a.observations > a.bounds.MaxObservations {
		return errors.New("observation limit exceeded")
	}
	for _, diagnostic := range observation.Diagnostics {
		a.diagnostics[diagnostic] = struct{}{}
		if len(a.diagnostics) > authorsession.MaxUniqueDiagnostics {
			return errors.New("diagnostic limit exceeded")
		}
	}
	if a.pendingTrace >= 0 {
		a.trace[a.pendingTrace].contextsAfter = cloneAuthorContexts(observation.Contexts)
		a.pendingTrace = -1
		a.pendingAction = nil
	}
	if phase == "authentication" && observation.Origin == a.dashboardOrigin && observation.Path == a.dashboardPath {
		proof, err := uniqueDashboardProof(observation, a.goalPredicate)
		if err != nil {
			return err
		}
		a.authProof = &proof
	}
	if observationMatchesPredicate(observation, a.goalPredicate) {
		copy := cloneObservation(observation)
		a.completion = &copy
	}
	return nil
}

func (a *Attestation) recordClient(message authorsession.ClientMessage, phase string, observation *authorsession.Observation) error {
	if a == nil {
		return errors.New("parent attestation is unavailable")
	}
	switch message.Type {
	case "observe":
		return nil
	case "focus_human_input":
		candidate, ok := observedCandidate(observation, message.CandidateID)
		if !ok {
			return errors.New("attested human-input candidate is unavailable")
		}
		a.pendingInput = &attestedInput{phase: phase, candidate: candidate, context: observation.Context}
		return nil
	case "human_input_complete":
		if a.pendingInput == nil || a.pendingInput.candidate.ID != message.CandidateID {
			return errors.New("attested human-input checkpoint is stale")
		}
		input := a.pendingInput
		a.pendingInput = nil
		a.trace = append(a.trace, attestedTraceStep{step: authorresult.TraceStep{
			Kind: "focus_human_input", Phase: input.phase, CandidateID: input.candidate.ID,
			Context: input.context, Role: input.candidate.Role, Label: input.candidate.Label,
			InputKind: input.inputKind, ChallengeKind: message.ChallengeKind,
		}, contextsBefore: cloneAuthorContexts(a.contexts), contextsAfter: cloneAuthorContexts(a.contexts)})
		return nil
	case "execute":
		step := authorresult.TraceStep{Kind: strings.TrimSuffix(message.Action, "_get"), Phase: phase, Context: normalizedContext(message.Context), URL: cleanAttestedURL(message.URL), POSTBudget: message.POSTBudget}
		if message.Action == "click" {
			candidate, ok := observedCandidate(observation, message.CandidateID)
			if !ok {
				return errors.New("attested click candidate is unavailable")
			}
			step.CandidateID, step.Context, step.Role, step.Label = candidate.ID, normalizedContext(observation.Context), candidate.Role, candidate.Label
		}
		a.trace = append(a.trace, attestedTraceStep{step: step, contextsBefore: cloneAuthorContexts(a.contexts)})
		a.pendingTrace = len(a.trace) - 1
		copy := message
		a.pendingAction = &copy
		return nil
	case "human_complete":
		if message.Outputs == nil || a.completion == nil {
			return errors.New("attested completion observation is unavailable")
		}
		a.outputRequests = append([]authorsession.OutputRequest(nil), (*message.Outputs)...)
		return nil
	default:
		return nil
	}
}

func (a *Attestation) recordCheckpoint(checkpoint authorsession.Checkpoint, observation *authorsession.Observation) error {
	if checkpoint.Kind == "completion" {
		return nil
	}
	if a.pendingInput == nil || a.pendingInput.candidate.ID != checkpoint.CandidateID {
		return errors.New("worker checkpoint does not match the parent-driven candidate")
	}
	inputKind := checkpoint.InputKind
	if observation == nil || checkpoint.Kind == "credential" && inputKind != "identifier" && inputKind != "password" || checkpoint.Kind == "mfa" && inputKind != "otp" && inputKind != "mfa" {
		return errors.New("worker checkpoint input kind is invalid")
	}
	a.pendingInput.inputKind = inputKind
	return nil
}

func (a *Attestation) recordApproval(approval authorsession.Approval) error {
	if _, exists := a.approvalIDs[approval.ID]; exists {
		return errors.New("approval identifier was reused")
	}
	if a.pendingAction == nil || approval.Action != a.pendingAction.Action || approval.CandidateID != a.pendingAction.CandidateID || approval.POSTBudget != a.pendingAction.POSTBudget {
		return errors.New("approval does not match the pending parent action")
	}
	switch approval.Kind {
	case "action":
		if approval.Origin != "" || approval.Action != "click" {
			return errors.New("action approval is malformed")
		}
	case "origin":
		_, origin, err := cleanURL(a.pendingAction.URL)
		if err != nil || approval.Action != "navigate_get" || approval.Origin != origin {
			return errors.New("origin approval does not match the pending navigation")
		}
	case "origin_action":
		if approval.Action != "click" || approval.Origin == "" {
			return errors.New("origin action approval is malformed")
		}
	default:
		return errors.New("approval kind is invalid")
	}
	a.approvalIDs[approval.ID] = struct{}{}
	a.approvals = append(a.approvals, approval)
	if approval.Origin != "" {
		a.origins[approval.Origin] = struct{}{}
	}
	return nil
}

// VerifyAttestation binds a child-owned authenticated-authoring envelope to
// the exact actions, checkpoints, observations, outputs, and origin approvals
// driven by the parent. Only bounded execution facts remain child-owned.
func VerifyAttestation(a *Attestation, envelope *authorresult.Envelope) error {
	if a == nil || envelope == nil {
		return errors.New("parent attestation is required")
	}
	if envelope.Goal != a.goal || envelope.GoalPredicate != a.goalPredicate || envelope.Bounds != a.bounds || !envelope.HumanConfirmed {
		return errors.New("authenticated-authoring result is not bound to the parent request")
	}
	if !reflect.DeepEqual(envelope.Origins, a.originLedger()) {
		return errors.New("authenticated-authoring origin ledger mismatch")
	}
	if !reflect.DeepEqual(envelope.Contexts, a.contexts) {
		return errors.New("authenticated-authoring context inventory mismatch")
	}
	if a.completion == nil {
		return errors.New("parent did not observe the completion proof")
	}
	wantGoal := proofForPredicate(*a.completion, a.goalPredicate)
	if wantGoal == nil || envelope.GoalProof != *wantGoal {
		return errors.New("authenticated-authoring goal proof mismatch")
	}
	if len(envelope.Trace) != len(a.trace) {
		return errors.New("authenticated-authoring trace length mismatch")
	}
	for index, actual := range envelope.Trace {
		expected := a.trace[index]
		if actual.Kind != expected.step.Kind || actual.Phase != expected.step.Phase || actual.CandidateID != expected.step.CandidateID || normalizedContext(actual.Context) != normalizedContext(expected.step.Context) || actual.Role != expected.step.Role || actual.Label != expected.step.Label || actual.ChallengeKind != expected.step.ChallengeKind || actual.URL != expected.step.URL || actual.POSTBudget != expected.step.POSTBudget || actual.POSTObserved < 0 || actual.POSTObserved > actual.POSTBudget {
			return errors.New("authenticated-authoring trace does not match parent-driven actions")
		}
		if actual.InputKind != expected.step.InputKind {
			return errors.New("authenticated-authoring input trace mismatch")
		}
		if err := verifyOpenedContext(actual, expected); err != nil {
			return err
		}
	}
	if err := verifyOutputSelections(a, envelope.OutputSelections); err != nil {
		return err
	}
	if err := verifyAuthenticationProof(a, envelope.AuthenticationProfile); err != nil {
		return err
	}
	wantDiagnostics := make([]string, 0, len(a.diagnostics))
	for diagnostic := range a.diagnostics {
		wantDiagnostics = append(wantDiagnostics, diagnostic)
	}
	sort.Strings(wantDiagnostics)
	if !reflect.DeepEqual(envelope.Diagnostics, wantDiagnostics) {
		return errors.New("authenticated-authoring diagnostic ledger mismatch")
	}
	return nil
}

func verifyOpenedContext(actual authorresult.TraceStep, expected attestedTraceStep) error {
	addedPopups := []string{}
	for id, browserContext := range expected.contextsAfter {
		if _, existed := expected.contextsBefore[id]; !existed && browserContext.Kind == "popup" {
			addedPopups = append(addedPopups, id)
		}
	}
	sort.Strings(addedPopups)
	if actual.OpensContext == "" {
		if actual.Kind == "click" && len(addedPopups) != 0 {
			return errors.New("authenticated-authoring opened context is unbound")
		}
		return nil
	}
	if actual.Kind != "click" || len(addedPopups) != 1 || addedPopups[0] != actual.OpensContext {
		return errors.New("authenticated-authoring opened context mismatch")
	}
	return nil
}

func verifyOutputSelections(a *Attestation, selections []authorresult.OutputSelection) error {
	requests := append([]authorsession.OutputRequest(nil), a.outputRequests...)
	sort.Slice(requests, func(i, j int) bool { return requests[i].Key < requests[j].Key })
	if len(requests) != len(selections) || a.completion == nil {
		return errors.New("authenticated-authoring output selection mismatch")
	}
	for index, request := range requests {
		selection := selections[index]
		candidate, ok := observedCandidate(a.completion, request.CandidateID)
		if !ok || selection.CandidateID != request.CandidateID || selection.Key != request.Key || selection.Type != request.Type || selection.LocatorMode != request.LocatorMode || selection.Observation != a.observations || selection.Context != a.completion.Context || selection.Role != candidate.Role || selection.Matches != 1 {
			return errors.New("authenticated-authoring output selection was not requested by the operator")
		}
		if selection.RoleMatches < 1 || selection.RoleMatches > a.bounds.MaxCandidates || request.LocatorMode == "exact_name" && selection.Name != candidate.Label || request.LocatorMode == "unique_role" && (selection.Name != "" || selection.RoleMatches != 1) {
			return errors.New("authenticated-authoring output match proof mismatch")
		}
	}
	return nil
}

func verifyAuthenticationProof(a *Attestation, raw json.RawMessage) error {
	if a.authProof == nil {
		return errors.New("parent did not observe authentication success")
	}
	var profile struct {
		Flows map[string]struct {
			Success struct {
				Origin  string `json:"origin"`
				Path    string `json:"path"`
				Context string `json:"context"`
				Locator struct {
					Role string `json:"role"`
					Name string `json:"name"`
				} `json:"locator"`
			} `json:"success"`
		} `json:"flows"`
	}
	if err := json.Unmarshal(raw, &profile); err != nil {
		return errors.New("authenticated-authoring authentication profile is malformed")
	}
	flow, ok := profile.Flows["authenticated_goal"]
	if !ok || flow.Success.Origin != a.authProof.Origin || flow.Success.Path != a.authProof.Path || normalizedContext(flow.Success.Context) != normalizedContext(a.authProof.Context) || flow.Success.Locator.Role != a.authProof.Role || flow.Success.Locator.Name != a.authProof.Label {
		return errors.New("authenticated-authoring authentication proof mismatch")
	}
	return nil
}

func uniqueDashboardProof(observation authorsession.Observation, goal authorresult.GoalPredicate) (authorresult.GoalProof, error) {
	if proof := proofForPredicate(observation, goal); proof != nil {
		return *proof, nil
	}
	priority := map[string]int{"heading": 0, "region": 1, "status": 2, "table": 3}
	candidates := append([]authorsession.Candidate(nil), observation.Candidates...)
	sort.SliceStable(candidates, func(i, j int) bool {
		left, leftOK := priority[candidates[i].Role]
		right, rightOK := priority[candidates[j].Role]
		if !leftOK {
			left = len(priority)
		}
		if !rightOK {
			right = len(priority)
		}
		return left < right
	})
	for _, candidate := range candidates {
		if candidate.Matches != 1 || candidate.Label == authorsession.RedactedLabel || candidate.Label == authorsession.UntrustedLabel {
			continue
		}
		matches := 0
		for _, other := range observation.Candidates {
			if other.Role == candidate.Role && other.Label == candidate.Label {
				matches += other.Matches
			}
		}
		if matches != 1 {
			return authorresult.GoalProof{}, errors.New("dashboard authentication proof is ambiguous")
		}
		return authorresult.GoalProof{Origin: observation.Origin, Path: observation.Path, Context: observation.Context, Role: candidate.Role, Label: candidate.Label, Matches: 1}, nil
	}
	return authorresult.GoalProof{}, errors.New("dashboard authentication proof is missing")
}

func proofForPredicate(observation authorsession.Observation, predicate authorresult.GoalPredicate) *authorresult.GoalProof {
	if !observationMatchesPredicate(observation, predicate) {
		return nil
	}
	for _, candidate := range observation.Candidates {
		if candidate.Role == predicate.Role && candidate.Matches == 1 && (predicate.Label == "" || candidate.Label == predicate.Label) {
			return &authorresult.GoalProof{Origin: observation.Origin, Path: observation.Path, Context: observation.Context, Role: candidate.Role, Label: candidate.Label, Matches: 1}
		}
	}
	return nil
}

func observationMatchesPredicate(observation authorsession.Observation, predicate authorresult.GoalPredicate) bool {
	if observation.Origin != predicate.Origin || observation.Path != predicate.Path || observation.Context != normalizedContext(predicate.Context) {
		return false
	}
	matches := 0
	for _, candidate := range observation.Candidates {
		if candidate.Role == predicate.Role && (predicate.Label == "" || candidate.Label == predicate.Label) {
			matches += candidate.Matches
		}
	}
	return matches == 1
}

func observedCandidate(observation *authorsession.Observation, id string) (authorsession.Candidate, bool) {
	if observation == nil {
		return authorsession.Candidate{}, false
	}
	for _, candidate := range observation.Candidates {
		if candidate.ID == id && candidate.Matches == 1 {
			return candidate, true
		}
	}
	return authorsession.Candidate{}, false
}

func cloneObservation(value authorsession.Observation) authorsession.Observation {
	value.Contexts = cloneAuthorContexts(value.Contexts)
	value.Candidates = append([]authorsession.Candidate(nil), value.Candidates...)
	value.Diagnostics = append([]string(nil), value.Diagnostics...)
	return value
}

func cloneAuthorContexts(values map[string]authorresult.Context) map[string]authorresult.Context {
	result := make(map[string]authorresult.Context, len(values))
	for id, value := range values {
		result[id] = value
	}
	return result
}

func cleanAttestedURL(raw string) string {
	if raw == "" {
		return ""
	}
	value, _, err := cleanURL(raw)
	if err != nil {
		return ""
	}
	return value
}

func normalizedContext(value string) string {
	if value == "" {
		return "main"
	}
	return value
}
