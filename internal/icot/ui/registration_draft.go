package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/credentialpolicy"
	"github.com/OpenUdon/uws/browserregistration"
)

const registrationAccessibilityDisclosure = registrationauthorsession.AccessibilityLabelDisclosure

const registrationSuccessProofOperatorReviewedDeferred = "operator_reviewed_deferred"

var registrationDraftSymbol = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)

var errRegistrationDraftBindingsInvalid = errors.New("registration draft environment symbols are invalid")

type registrationDraftRequest struct {
	Title            string                        `json:"title"`
	Provider         string                        `json:"provider,omitempty"`
	Confidence       string                        `json:"confidence"`
	ExpiresAfter     string                        `json:"expires_after"`
	UIStabilityScore *float64                      `json:"ui_stability_score,omitempty"`
	CredentialSlots  []registrationDraftSlot       `json:"credential_slots"`
	Flow             registrationDraftFlow         `json:"flow"`
	CallControls     registrationDraftCallControls `json:"call_controls"`
}

type registrationDraftSlot struct {
	Slot    string `json:"slot"`
	Kind    string `json:"kind"`
	Binding string `json:"binding"`
}

type registrationDraftFlow struct {
	Name               string                   `json:"name"`
	Description        string                   `json:"description,omitempty"`
	Steps              []registrationDraftStep  `json:"steps"`
	Effects            []string                 `json:"effects"`
	ConfirmationPrompt string                   `json:"confirmation_prompt,omitempty"`
	Success            registrationDraftSuccess `json:"success"`
}

type registrationDraftStep struct {
	Type           string `json:"type"`
	Navigate       string `json:"navigate,omitempty"`
	CandidateID    string `json:"candidate_id,omitempty"`
	Slot           string `json:"slot,omitempty"`
	CheckpointKind string `json:"checkpoint_kind,omitempty"`
}

type registrationDraftSuccess struct {
	Origin           string                          `json:"origin"`
	Path             string                          `json:"path,omitempty"`
	Proof            string                          `json:"proof"`
	OperatorReviewed bool                            `json:"operator_reviewed"`
	Locator          registrationDraftSuccessLocator `json:"locator"`
}

type registrationDraftSuccessLocator struct {
	Role string `json:"role"`
	Name string `json:"name"`
}

type registrationDraftCallControls struct {
	Approval            string `json:"approval"`
	DuplicatePrevention string `json:"duplicate_prevention"`
	OnDuplicate         string `json:"on_duplicate"`
	AmbiguousOutcome    string `json:"ambiguous_outcome"`
	CleanupDisposition  string `json:"cleanup_disposition"`
}

type RegistrationDraftDisclosure struct {
	ProfileSHA256       string                                 `json:"profile_sha256"`
	Canonical           json.RawMessage                        `json:"canonical"`
	Flow                string                                 `json:"flow"`
	CleanupDisposition  string                                 `json:"cleanup_disposition"`
	CallControls        registrationDraftCallControls          `json:"call_controls"`
	CredentialBindings  []browsertransaction.CredentialBinding `json:"credential_bindings"`
	RetainedQueries     []RetainedQueryDisclosure              `json:"retained_queries"`
	AccessibilityLabels string                                 `json:"accessibility_labels"`
	SuccessProof        RegistrationSuccessProofDisclosure     `json:"success_proof"`
}

type RegistrationSuccessProofDisclosure struct {
	ReviewKind              string `json:"review_kind"`
	ObservedDuringAuthoring bool   `json:"observed_during_authoring"`
	RuntimeProofRequired    bool   `json:"runtime_proof_required"`
}

type RetainedQueryDisclosure struct {
	Navigation string                   `json:"navigation"`
	Parameters []RetainedQueryParameter `json:"parameters"`
}

type RetainedQueryParameter struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func buildRegistrationDraft(request registrationDraftRequest, start registrationAuthoringStartRequest, observation registrationauthorsession.Observation, now time.Time) ([]byte, []string, []browsertransaction.CredentialBinding, *RegistrationDraftDisclosure, error) {
	if now.IsZero() || strings.TrimSpace(request.Title) == "" || len(request.Title) > 256 || len(request.Provider) > 256 ||
		(request.Confidence != "low" && request.Confidence != "medium" && request.Confidence != "high") || strings.TrimSpace(request.ExpiresAfter) == "" ||
		!registrationDraftSymbol.MatchString(request.Flow.Name) || len(request.Flow.Description) > 1024 || len(request.Flow.ConfirmationPrompt) > 512 {
		return nil, nil, nil, nil, errors.New("registration draft metadata is invalid")
	}
	controls := request.CallControls
	if controls.Approval != "browser_registration_submit" || controls.DuplicatePrevention != "operator_attestation" || controls.OnDuplicate != "fail" ||
		controls.AmbiguousOutcome != "stop_without_retry" || controls.CleanupDisposition != "delete_separately" && controls.CleanupDisposition != "retain_dedicated_test_identity" {
		return nil, nil, nil, nil, errors.New("registration draft call controls are invalid")
	}
	origins := append([]string(nil), start.Origins...)
	sort.Strings(origins)
	if len(origins) == 0 || observation.Generation <= 0 || len(observation.Candidates) == 0 {
		return nil, nil, nil, nil, errors.New("registration draft observation authority is missing")
	}
	for index, origin := range origins {
		if strings.TrimSpace(origin) != origin || index > 0 && origins[index-1] == origin {
			return nil, nil, nil, nil, errors.New("registration draft origins are invalid")
		}
	}
	byID := make(map[string]registrationauthorsession.Candidate, len(observation.Candidates))
	for _, candidate := range observation.Candidates {
		if candidate.Matches == 1 && candidate.Label != "" {
			byID[candidate.ID] = candidate
		}
	}

	slots := make(map[string]browserregistration.CredentialSlot, len(request.CredentialSlots))
	bindings := make([]browsertransaction.CredentialBinding, 0, len(request.CredentialSlots))
	seenBindings := map[string]bool{}
	if len(request.CredentialSlots) == 0 || len(request.CredentialSlots) > 32 {
		return nil, nil, nil, nil, errors.New("registration draft slots are invalid")
	}
	for _, slot := range request.CredentialSlots {
		if !registrationDraftSymbol.MatchString(slot.Slot) ||
			slot.Kind != "identifier" && slot.Kind != "password" || slots[slot.Slot].Kind != "" {
			return nil, nil, nil, nil, errors.New("registration draft slots are invalid")
		}
		if !validRegistrationDraftBindingName(slot.Binding) || seenBindings[slot.Binding] {
			return nil, nil, nil, nil, errRegistrationDraftBindingsInvalid
		}
		slots[slot.Slot] = browserregistration.CredentialSlot{Kind: slot.Kind}
		seenBindings[slot.Binding] = true
		bindings = append(bindings, browsertransaction.CredentialBinding{Slot: slot.Slot, Binding: slot.Binding})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Slot < bindings[j].Slot })

	if len(request.Flow.Steps) == 0 || len(request.Flow.Steps) > 256 {
		return nil, nil, nil, nil, errors.New("registration draft steps are invalid")
	}
	steps := make([]browserregistration.Step, 0, len(request.Flow.Steps))
	selected := map[string]bool{}
	submitCount := 0
	for _, raw := range request.Flow.Steps {
		step, candidateID, err := buildRegistrationDraftStep(raw, slots, byID)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if candidateID != "" {
			selected[candidateID] = true
		}
		if step.Submit != nil {
			submitCount++
		}
		steps = append(steps, step)
	}
	if submitCount != 1 {
		return nil, nil, nil, nil, errors.New("registration draft must contain exactly one submit")
	}
	success, err := buildRegistrationDraftSuccess(request.Flow.Success, origins)
	if err != nil {
		return nil, nil, nil, nil, errors.New("registration draft success proof is invalid")
	}
	effects := append([]string(nil), request.Flow.Effects...)
	sort.Strings(effects)

	stamp := now.UTC().Truncate(time.Second).Format(time.RFC3339)
	profile := &browserregistration.Profile{
		Profile: browserregistration.ProfileName,
		Info: browserregistration.Info{
			Title: strings.TrimSpace(request.Title), Provider: strings.TrimSpace(request.Provider),
			ApplicationOrigins: append([]string(nil), origins...), RegistrationOrigins: append([]string(nil), origins...),
		},
		ObservationKind: "accessibility_snapshot",
		Evidence:        browserregistration.Evidence{LearnedAt: stamp, Source: "icot_no_submit_observation_operator_reviewed_success"},
		Confidence:      request.Confidence, ExpiresAfter: request.ExpiresAfter,
		Verification:    browserregistration.Verification{LastVerifiedAt: stamp, UIStabilityScore: request.UIStabilityScore},
		CredentialSlots: slots,
		Flows: map[string]browserregistration.Flow{request.Flow.Name: {
			Description: request.Flow.Description, Sequence: steps, Effects: effects,
			ConfirmationPolicy: browserregistration.ConfirmationPolicy{Required: true, Prompt: request.Flow.ConfirmationPrompt},
			Success:            success,
		}},
	}
	canonical, err := registrationprofile.MarshalJSON(profile)
	if err != nil || registrationprofile.ValidateAt(profile, now.UTC()) != nil || registrationprofile.ValidateRetainedNavigationV2(profile) != nil {
		return nil, nil, nil, nil, errors.New("registration draft does not satisfy the public profile contract")
	}
	candidateIDs := make([]string, 0, len(selected))
	for candidateID := range selected {
		candidateIDs = append(candidateIDs, candidateID)
	}
	sort.Strings(candidateIDs)
	digest := sha256.Sum256(canonical)
	disclosure := &RegistrationDraftDisclosure{
		ProfileSHA256: "sha256:" + hex.EncodeToString(digest[:]), Canonical: append(json.RawMessage(nil), canonical...),
		Flow: request.Flow.Name, CleanupDisposition: controls.CleanupDisposition, CallControls: controls,
		CredentialBindings: append([]browsertransaction.CredentialBinding(nil), bindings...),
		RetainedQueries:    retainedQueryDisclosures(profile), AccessibilityLabels: registrationAccessibilityDisclosure,
		SuccessProof: RegistrationSuccessProofDisclosure{
			ReviewKind: registrationSuccessProofOperatorReviewedDeferred, ObservedDuringAuthoring: false, RuntimeProofRequired: true,
		},
	}
	return canonical, candidateIDs, bindings, disclosure, nil
}

func validRegistrationDraftBindingName(binding string) bool {
	// Binding is a typed environment-symbol name, not an untyped mapping value.
	// Known credential formats remain rejected by the shared scanner. When the
	// value-oriented entropy heuristic also fires, require a closed descriptive
	// snake_case vocabulary instead of treating arbitrary lowercase runs as
	// words. This keeps opaque letters-only values out of transaction metadata.
	if !registrationDraftSymbol.MatchString(binding) || credentialpolicy.ContainsLikelyValue([]byte(binding)) {
		return false
	}
	return !credentialpolicy.IsLikelyLiteral(binding) || descriptiveRegistrationDraftBindingName(binding)
}

func descriptiveRegistrationDraftBindingName(binding string) bool {
	parts := strings.Split(binding, "_")
	if len(parts) < 2 {
		return false
	}
	compactNamespaceParts := 0
	descriptiveWords := 0
	for _, part := range parts {
		if registrationDraftDescriptiveBindingWord(part) {
			descriptiveWords++
			continue
		}
		if !registrationDraftCompactNamespacePart(part) {
			return false
		}
		compactNamespaceParts++
		if compactNamespaceParts > 1 {
			return false
		}
	}
	// One compact product namespace such as app8 or w8m is useful, but it is
	// never sufficient authority by itself or with only one purpose suffix.
	return descriptiveWords >= 2
}

func registrationDraftDescriptiveBindingWord(value string) bool {
	// This closed vocabulary is part of the transaction-metadata disclosure
	// boundary. Additions require adversarial opaque-token coverage.
	switch value {
	case "account", "app", "application", "auth", "browser", "confirm", "confirmation", "contact", "credential", "customer", "dedicated", "display", "email", "environment", "existing", "first", "form", "full", "identifier", "input", "last", "login", "member", "name", "new", "password", "portal", "primary", "private", "profile", "registration", "runtime", "secondary", "service", "signup", "test", "user", "username", "workspace":
		return true
	default:
		return false
	}
}

func registrationDraftCompactNamespacePart(value string) bool {
	if len(value) == 0 || len(value) > 12 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	digitRuns := 0
	inDigits := false
	for index := 1; index < len(value); index++ {
		char := value[index]
		if char >= '0' && char <= '9' {
			if !inDigits {
				digitRuns++
				inDigits = true
			}
			continue
		}
		if char < 'a' || char > 'z' {
			return false
		}
		inDigits = false
	}
	return digitRuns == 1
}

func buildRegistrationDraftSuccess(raw registrationDraftSuccess, origins []string) (browserregistration.SuccessCondition, error) {
	if raw.Proof != registrationSuccessProofOperatorReviewedDeferred || !raw.OperatorReviewed || raw.Origin == "" || strings.TrimSpace(raw.Origin) != raw.Origin ||
		raw.Path != strings.TrimSpace(raw.Path) || raw.Locator.Role == "" || strings.TrimSpace(raw.Locator.Role) != raw.Locator.Role ||
		raw.Locator.Name == "" || strings.TrimSpace(raw.Locator.Name) != raw.Locator.Name {
		return browserregistration.SuccessCondition{}, errors.New("operator-reviewed deferred success proof is incomplete")
	}
	declaredOrigin := false
	for _, origin := range origins {
		if raw.Origin == origin {
			declaredOrigin = true
			break
		}
	}
	if !declaredOrigin {
		return browserregistration.SuccessCondition{}, errors.New("operator-reviewed deferred success origin is not declared")
	}
	reduced := authorsession.ReduceAccessibilityLabel(raw.Locator.Name).Value
	if reduced != raw.Locator.Name || reduced == authorsession.RedactedLabel || reduced == authorsession.UntrustedLabel {
		return browserregistration.SuccessCondition{}, errors.New("operator-reviewed deferred success name is not disclosure-safe")
	}
	return browserregistration.SuccessCondition{
		Origin: raw.Origin, Path: raw.Path,
		Locator: browserregistration.Locator{Role: raw.Locator.Role, Name: raw.Locator.Name},
	}, nil
}

func buildRegistrationDraftStep(raw registrationDraftStep, slots map[string]browserregistration.CredentialSlot, candidates map[string]registrationauthorsession.Candidate) (browserregistration.Step, string, error) {
	switch raw.Type {
	case "navigate":
		if strings.TrimSpace(raw.Navigate) == "" || raw.CandidateID != "" || raw.Slot != "" || raw.CheckpointKind != "" {
			return browserregistration.Step{}, "", errors.New("registration navigate step is invalid")
		}
		return browserregistration.Step{Navigate: raw.Navigate}, "", nil
	case "type_credential":
		candidate, ok := candidates[raw.CandidateID]
		if !ok || slots[raw.Slot].Kind == "" || raw.Navigate != "" || raw.CheckpointKind != "" {
			return browserregistration.Step{}, "", errors.New("registration credential step is invalid")
		}
		return browserregistration.Step{TypeCredential: &browserregistration.TypeCredentialStep{Locator: registrationDraftLocator(candidate), Slot: raw.Slot}}, raw.CandidateID, nil
	case "click":
		candidate, ok := candidates[raw.CandidateID]
		if !ok || raw.Navigate != "" || raw.Slot != "" || raw.CheckpointKind != "" {
			return browserregistration.Step{}, "", errors.New("registration click step is invalid")
		}
		return browserregistration.Step{Click: &browserregistration.ClickStep{Locator: registrationDraftLocator(candidate)}}, raw.CandidateID, nil
	case "submit":
		candidate, ok := candidates[raw.CandidateID]
		if !ok || raw.Navigate != "" || raw.Slot != "" || raw.CheckpointKind != "" {
			return browserregistration.Step{}, "", errors.New("registration submit step is invalid")
		}
		return browserregistration.Step{Submit: &browserregistration.SubmitStep{Locator: registrationDraftLocator(candidate)}}, raw.CandidateID, nil
	case "human_checkpoint":
		if raw.CheckpointKind != "captcha" && raw.CheckpointKind != "email_verification" && raw.CheckpointKind != "mfa" && raw.CheckpointKind != "consent" && raw.CheckpointKind != "other_control" || raw.Navigate != "" || raw.Slot != "" {
			return browserregistration.Step{}, "", errors.New("registration human checkpoint is invalid")
		}
		checkpoint := &browserregistration.HumanCheckpointStep{Kind: raw.CheckpointKind}
		if raw.CandidateID != "" {
			candidate, ok := candidates[raw.CandidateID]
			if !ok {
				return browserregistration.Step{}, "", errors.New("registration checkpoint candidate is invalid")
			}
			locator := registrationDraftLocator(candidate)
			checkpoint.Locator = &locator
		}
		return browserregistration.Step{HumanCheckpoint: checkpoint}, raw.CandidateID, nil
	case "wait_for":
		candidate, ok := candidates[raw.CandidateID]
		if !ok || raw.Navigate != "" || raw.Slot != "" || raw.CheckpointKind != "" {
			return browserregistration.Step{}, "", errors.New("registration wait step is invalid")
		}
		return browserregistration.Step{WaitFor: &browserregistration.WaitForCondition{Locator: registrationDraftLocator(candidate)}}, raw.CandidateID, nil
	default:
		return browserregistration.Step{}, "", errors.New("registration step type is invalid")
	}
}

func registrationDraftLocator(candidate registrationauthorsession.Candidate) browserregistration.Locator {
	return browserregistration.Locator{Role: candidate.Role, Name: candidate.Label}
}

func retainedQueryDisclosures(profile *browserregistration.Profile) []RetainedQueryDisclosure {
	var result []RetainedQueryDisclosure
	for _, flowName := range registrationprofile.SortedFlowNames(profile) {
		for _, step := range profile.Flows[flowName].Sequence {
			if step.Navigate == "" {
				continue
			}
			parsed, err := url.Parse(step.Navigate)
			if err != nil || parsed.RawQuery == "" {
				continue
			}
			values, err := url.ParseQuery(parsed.RawQuery)
			if err != nil {
				continue
			}
			disclosure := RetainedQueryDisclosure{Navigation: step.Navigate}
			keys := make([]string, 0, len(values))
			for key := range values {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				disclosure.Parameters = append(disclosure.Parameters, RetainedQueryParameter{Key: key, Value: values[key][0]})
			}
			result = append(result, disclosure)
		}
	}
	return result
}
