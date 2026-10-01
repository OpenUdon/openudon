package elicitor

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/browsertools/draft"
	bevidence "github.com/OpenUdon/browsertools/evidence"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/review"
	eartifact "github.com/OpenUdon/evidence/artifact"
	"github.com/OpenUdon/evidence/redact"
)

const (
	browserGuidedAuthoringVersion  = "browsertools.guided-authoring.v1"
	browserGuidedSourceKind        = browsertools.LocalSourceKind("guided_authoring_bundle")
	browserGuidedMaxCatalogRecords = 256
	browserGuidedMaxRecords        = 2048
	browserGuidedMaxPerAction      = 64
	browserGuidedMaxActions        = 64
	browserGuidedMaxStepsPerAction = 128
	browserGuidedMaxLocators       = 8192
	browserGuidedMaxOutputs        = 4096
	browserGuidedMaxDecisions      = 8192
	browserGuidedMaxMatchingWork   = 2_000_000
)

var browserGuidedTemplatePattern = regexp.MustCompile(`\{\{([A-Za-z0-9_-]{1,128})\}\}`)

// browserGuidedBundle is a downstream adapter for Browsertools' reviewed
// guided-authoring envelope. OpenUdon never stages this evidence envelope; it
// replays Browsertools' existing draft and review gates and stages only the
// resulting browser.1.5 profile after proposal approval.
type browserGuidedBundle struct {
	Version   string                      `json:"version"`
	Spec      draft.Spec                  `json:"spec"`
	Profile   profile.Profile             `json:"profile"`
	Evidence  []bevidence.Record          `json:"evidence"`
	Decisions []bevidence.LocatorDecision `json:"decisions"`
	Review    review.Bundle               `json:"review"`
}

func inspectExplicitGuidedBundle(path string, at time.Time) (browsertools.LocalSourceCandidate, bool, error) {
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return browsertools.LocalSourceCandidate{}, false, nil
	}
	data, err := readStableBrowserCandidate(path, "")
	if err != nil {
		return browsertools.LocalSourceCandidate{}, false, err
	}
	var discriminator struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(bytes.NewReader(data)).Decode(&discriminator); err != nil {
		return browsertools.LocalSourceCandidate{}, false, nil
	}
	version := strings.TrimSpace(discriminator.Version)
	if version != browserGuidedAuthoringVersion {
		if strings.HasPrefix(version, "browsertools.guided-authoring.") {
			return browsertools.LocalSourceCandidate{}, true, fmt.Errorf("unsupported guided-authoring version %q", version)
		}
		return browsertools.LocalSourceCandidate{}, false, nil
	}
	bundle, err := parseBrowserGuidedBundle(data, at)
	if err != nil {
		return browsertools.LocalSourceCandidate{}, true, err
	}
	expiresAt, err := browserProfileExpiry(&bundle.Profile)
	if err != nil {
		return browsertools.LocalSourceCandidate{}, true, err
	}
	status := eartifact.LifecycleActive
	if !at.Before(expiresAt) {
		status = eartifact.LifecycleStale
	}
	actions := bundle.Profile.SortedActionNames()
	return browsertools.LocalSourceCandidate{
		Kind: browserGuidedSourceKind, Title: bundle.Profile.Info.Title,
		Origins: append([]string(nil), bundle.Profile.Info.Origin...), Actions: actions,
		ActionCount: len(actions), Score: 100, Path: path, SizeBytes: int64(len(data)),
		Digest: fmt.Sprintf("sha256:%x", sha256.Sum256(data)), Status: status,
		Provenance: firstNonEmpty(bundle.Profile.Evidence.Source, "browsertools-guided-authoring"),
	}, true, nil
}

func parseBrowserGuidedBundle(data []byte, at time.Time) (*browserGuidedBundle, error) {
	if at.IsZero() {
		return nil, fmt.Errorf("guided-authoring verification time is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var bundle browserGuidedBundle
	if err := decoder.Decode(&bundle); err != nil {
		return nil, fmt.Errorf("guided-authoring bundle: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("guided-authoring bundle contains trailing JSON")
		}
		return nil, fmt.Errorf("guided-authoring bundle: %w", err)
	}
	if bundle.Version != browserGuidedAuthoringVersion {
		return nil, fmt.Errorf("guided-authoring version %q is unsupported", bundle.Version)
	}
	if err := validateBrowserGuidedBounds(&bundle); err != nil {
		return nil, err
	}
	if err := validateBrowserGuidedSafety(&bundle); err != nil {
		return nil, err
	}
	rebuilt, err := draft.Build(bundle.Evidence, bundle.Spec)
	if err != nil {
		return nil, fmt.Errorf("guided-authoring draft replay: %w", err)
	}
	if !rebuilt.ReadyForReview() || rebuilt.Profile == nil {
		return nil, fmt.Errorf("guided-authoring draft replay is not ready for review")
	}
	if !reflect.DeepEqual(bundle.Spec.Decisions, bundle.Decisions) || !reflect.DeepEqual(rebuilt.Decisions, bundle.Decisions) {
		return nil, fmt.Errorf("guided-authoring ambiguity decisions do not match the replayed specification")
	}
	if !browserGuidedDecisionsEqual(bundle.Review.Decisions, bundle.Decisions) {
		return nil, fmt.Errorf("guided-authoring review decisions do not match the replayed specification")
	}
	if !reflect.DeepEqual(*rebuilt.Profile, bundle.Profile) {
		return nil, fmt.Errorf("guided-authoring embedded profile does not match the replayed specification and evidence")
	}
	if !bundle.Review.Promotable() {
		return nil, fmt.Errorf("guided-authoring review is not promotable")
	}
	if err := review.Verify(&bundle.Review, &bundle.Profile, bundle.Evidence, at.UTC()); err != nil {
		return nil, fmt.Errorf("guided-authoring review: %w", err)
	}
	if err := validateBrowserAuthoringProfile(&bundle.Profile); err != nil {
		return nil, fmt.Errorf("guided-authoring profile: %w", err)
	}
	return &bundle, nil
}

func browserGuidedDecisionsEqual(left, right []bevidence.LocatorDecision) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !reflect.DeepEqual(left[index], right[index]) {
			return false
		}
	}
	return true
}

func validateBrowserGuidedSafety(bundle *browserGuidedBundle) error {
	data, err := json.Marshal(bundle)
	if err != nil {
		return fmt.Errorf("guided-authoring safety: %w", err)
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return fmt.Errorf("guided-authoring safety: %w", err)
	}
	if _, changed := redact.Document(document, redact.Options{ExtraSensitiveKeys: []string{"cookie", "session", "storage", "raw_capture", "raw_browser", "screenshot"}}); changed {
		return fmt.Errorf("guided-authoring bundle contains secret, credential, session, or private-browser-shaped content")
	}
	for recordIndex, record := range bundle.Evidence {
		for outputIndex, output := range record.CandidateOutputs {
			if sensitiveBrowserName(output.Key) || (output.Property != "" && sensitiveBrowserName(output.Property)) {
				return fmt.Errorf("guided-authoring evidence[%d].candidateOutputs[%d] is credential-shaped", recordIndex, outputIndex)
			}
		}
	}
	for name, action := range bundle.Spec.Actions {
		parameters := browserSchemaProperties(action.Parameters)
		for stepIndex, step := range action.Sequence {
			switch step.Kind {
			case profile.StepTypeText:
				if step.TypeText == nil {
					continue
				}
				if err := validateBrowserGuidedParameterValue(name, stepIndex, step.TypeText.Value, parameters); err != nil {
					return err
				}
			case profile.StepSelectOption:
				if step.SelectOption == nil {
					continue
				}
				if err := validateBrowserGuidedParameterValue(name, stepIndex, step.SelectOption.Value, parameters); err != nil {
					return err
				}
			case profile.StepNavigate:
				resolved := browserGuidedTemplatePattern.ReplaceAllString(step.Navigate, "safe")
				parsed, parseErr := url.Parse(resolved)
				if parseErr != nil || parsed.User != nil {
					return fmt.Errorf("guided-authoring action %q sequence[%d] has an unsafe navigate target", name, stepIndex)
				}
				query, queryErr := url.ParseQuery(parsed.RawQuery)
				if queryErr != nil {
					return fmt.Errorf("guided-authoring action %q sequence[%d] has a malformed navigate query", name, stepIndex)
				}
				for key := range query {
					if sensitiveBrowserName(key) || browserGuidedSensitiveQueryKey(key) {
						return fmt.Errorf("guided-authoring action %q sequence[%d] has a credential-shaped navigate query", name, stepIndex)
					}
				}
			}
		}
	}
	return nil
}

func validateBrowserGuidedParameterValue(action string, step int, value string, parameters map[string]any) error {
	matches := browserGuidedTemplatePattern.FindStringSubmatch(value)
	if len(matches) != 2 || matches[0] != value {
		return fmt.Errorf("guided-authoring action %q sequence[%d] value must be one exact declared parameter template", action, step)
	}
	if _, exists := parameters[matches[1]]; !exists || sensitiveBrowserName(matches[1]) {
		return fmt.Errorf("guided-authoring action %q sequence[%d] references an invalid value parameter", action, step)
	}
	return nil
}

func browserGuidedSensitiveQueryKey(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "auth", "code", "key", "oauth_state", "session", "session_id", "sig", "signature", "state":
		return true
	default:
		return false
	}
}

func validateBrowserGuidedBounds(bundle *browserGuidedBundle) error {
	if bundle == nil {
		return fmt.Errorf("guided-authoring bundle is required")
	}
	if len(bundle.Evidence) == 0 || len(bundle.Evidence) > browserGuidedMaxRecords {
		return fmt.Errorf("guided-authoring evidence count must be between 1 and %d", browserGuidedMaxRecords)
	}
	if len(bundle.Spec.Actions) == 0 || len(bundle.Spec.Actions) > browserGuidedMaxActions {
		return fmt.Errorf("guided-authoring action count must be between 1 and %d", browserGuidedMaxActions)
	}
	if len(bundle.Spec.Info.Origin) == 0 || len(bundle.Spec.Info.Origin) > browserGuidedMaxCatalogRecords {
		return fmt.Errorf("guided-authoring origin count must be between 1 and %d", browserGuidedMaxCatalogRecords)
	}
	if len(bundle.Decisions) > browserGuidedMaxDecisions || len(bundle.Spec.Decisions) > browserGuidedMaxDecisions || len(bundle.Review.Decisions) > browserGuidedMaxDecisions {
		return fmt.Errorf("guided-authoring ambiguity decisions exceed %d", browserGuidedMaxDecisions)
	}
	locators, outputs := 0, 0
	candidateWork := map[string]int{}
	perAction := map[string]int{}
	catalogRecords := map[[sha256.Size]byte]bool{}
	for _, record := range bundle.Evidence {
		perAction[record.ActionHint]++
		if perAction[record.ActionHint] > browserGuidedMaxPerAction {
			return fmt.Errorf("guided-authoring action %q exceeds %d evidence records", record.ActionHint, browserGuidedMaxPerAction)
		}
		candidateWork[record.ActionHint] += len(record.CandidateLocators) + len(record.CandidateOutputs)
		catalogRecord := record
		catalogRecord.ActionHint = ""
		encoded, err := json.Marshal(catalogRecord)
		if err != nil {
			return fmt.Errorf("guided-authoring evidence identity: %w", err)
		}
		identity := sha256.Sum256(encoded)
		if catalogRecords[identity] {
			continue
		}
		catalogRecords[identity] = true
		if len(catalogRecords) > browserGuidedMaxCatalogRecords {
			return fmt.Errorf("guided-authoring evidence exceeds %d distinct catalog records", browserGuidedMaxCatalogRecords)
		}
		recordLocators := append([]bevidence.CandidateLocator(nil), record.CandidateLocators...)
		for _, output := range record.CandidateOutputs {
			if output.Locator != nil && !browserGuidedContainsLocator(recordLocators, *output.Locator) {
				recordLocators = append(recordLocators, *output.Locator)
			}
		}
		locators += len(recordLocators)
		outputs += len(record.CandidateOutputs)
		if locators > browserGuidedMaxLocators || outputs > browserGuidedMaxOutputs {
			return fmt.Errorf("guided-authoring evidence exceeds locator/output bounds")
		}
	}
	matchingWork, replayWork := 0, 0
	for name, action := range bundle.Spec.Actions {
		if len(action.Sequence) == 0 || len(action.Sequence) > browserGuidedMaxStepsPerAction {
			return fmt.Errorf("guided-authoring action %q step count must be between 1 and %d", name, browserGuidedMaxStepsPerAction)
		}
		if len(browserSchemaProperties(action.Parameters)) > 128 {
			return fmt.Errorf("guided-authoring action %q parameters exceed 128", name)
		}
		if len(action.Outputs) > browserGuidedMaxOutputs {
			return fmt.Errorf("guided-authoring action %q exceeds output bounds", name)
		}
		declaredLocators := 0
		for _, step := range action.Sequence {
			if step.Locator() != nil {
				declaredLocators++
			}
			if wait := step.PostWait(); wait != nil && wait.Locator != nil {
				declaredLocators++
			}
		}
		for _, output := range action.Outputs {
			if output.Source == profile.OutputA11y && output.Locator != nil {
				declaredLocators++
			}
		}
		matchingWork += candidateWork[name] * (len(action.Sequence) + len(action.Outputs) + 1)
		replayWork += declaredLocators * (candidateWork[name] + len(bundle.Decisions))
		if matchingWork > browserGuidedMaxMatchingWork || replayWork > browserGuidedMaxMatchingWork {
			return fmt.Errorf("guided-authoring evidence and declarations exceed bounded matching work")
		}
	}
	return nil
}

func browserGuidedContainsLocator(values []bevidence.CandidateLocator, candidate bevidence.CandidateLocator) bool {
	for _, value := range values {
		if value.Role == candidate.Role && value.Name == candidate.Name && value.Text == candidate.Text && value.Value == candidate.Value {
			return true
		}
	}
	return false
}

func materializeBrowserGuidedProfile(data []byte, at time.Time) (*profile.Profile, []byte, error) {
	bundle, err := parseBrowserGuidedBundle(data, at)
	if err != nil {
		return nil, nil, err
	}
	materialized, err := json.MarshalIndent(bundle.Profile, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	return &bundle.Profile, append(materialized, '\n'), nil
}
