package synthesize

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/browsertools/registrationreview"
	"github.com/OpenUdon/openudon/internal/browserworkflow"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const browserRegistrationReviewVersion = "openudon.browser-registration-review.v1"

type browserRegistrationReview struct {
	Version string                              `json:"version"`
	Calls   []browserRegistrationReviewedCall   `json:"registration_calls"`
	Sources []browserRegistrationReviewedSource `json:"sources"`
}

type browserRegistrationReviewedCall struct {
	Step                string            `json:"step"`
	Source              string            `json:"source"`
	Flow                string            `json:"flow"`
	CredentialBindings  map[string]string `json:"credential_bindings"`
	Approval            string            `json:"approval"`
	DuplicatePrevention string            `json:"duplicate_prevention"`
	OnDuplicate         string            `json:"on_duplicate"`
	AmbiguousOutcome    string            `json:"ambiguous_outcome"`
	CleanupDisposition  string            `json:"cleanup_disposition"`
	Timeout             float64           `json:"timeout"`
}

type browserRegistrationReviewedSource struct {
	ID                  string              `json:"id"`
	TargetPath          string              `json:"target_path"`
	SHA256              string              `json:"sha256"`
	ReviewPath          string              `json:"review_path"`
	ReviewSHA256        string              `json:"review_sha256"`
	ProfileDigest       string              `json:"profile_digest"`
	Title               string              `json:"title"`
	Flows               []string            `json:"flows"`
	FlowCredentialSlots map[string][]string `json:"flow_credential_slots"`
	Origins             []string            `json:"origins"`
	Lifecycle           string              `json:"lifecycle"`
	ExpiresAt           string              `json:"expires_at"`
	Provenance          string              `json:"provenance"`
}

func assessBrowserRegistrationSources(report *QualityReport, exampleDir string, intent *rollout.Intent) {
	paths, err := packageartifacts.CollectBrowserRegistrationProfilePaths(exampleDir)
	if err != nil {
		report.add("browser.registration.sources", "fail", "browser registration profiles could not be inventoried", err.Error())
		return
	}
	if len(paths) == 0 {
		if intentHasBrowserRegistrationStep(intent) {
			report.add("browser.registration.sources", "fail", "browser registration requires a packaged profile", "Select a reviewed uws.browser-registration.1.0 profile.")
			return
		}
		report.add("browser.registration.sources", "pass", "browser registration review is not required", "")
		return
	}
	if _, err := packageartifacts.CollectBrowserRegistrationBundlePaths(exampleDir); err != nil {
		report.add("browser.registration.review", "fail", "browser registration review bundles are required", err.Error())
		return
	}
	data, err := readBrowserSourceReviewFile(filepath.Join(exampleDir, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)))
	if err != nil {
		report.add("browser.registration.review", "fail", "browser registration review evidence is required", err.Error())
		return
	}
	var review browserRegistrationReview
	if err := evidencefile.DecodeStrict(data, &review); err != nil {
		report.add("browser.registration.review", "fail", "browser registration review evidence must be valid JSON", err.Error())
		return
	}
	if err := validateBrowserRegistrationReview(exampleDir, paths, intent, review, time.Now().UTC()); err != nil {
		report.add("browser.registration.review", "fail", "browser registration review evidence is invalid", err.Error())
		return
	}
	report.add("browser.registration.sources", "pass", fmt.Sprintf("%d verified active browser registration profile(s) are packaged", len(paths)), strings.Join(paths, ", "))
	report.add("browser.registration.review", "pass", "profile and review digests, origins, flow slots, symbolic bindings, mutation policy, cleanup disposition, timeout, and exact approval agree", "")
}

func validateBrowserRegistrationReview(exampleDir string, paths []string, intent *rollout.Intent, review browserRegistrationReview, at time.Time) error {
	if review.Version != browserRegistrationReviewVersion {
		return fmt.Errorf("version must be %q", browserRegistrationReviewVersion)
	}
	metadata := map[string]browserRegistrationReviewedSource{}
	for _, source := range review.Sources {
		path, err := packageartifacts.CleanRelativePath(source.TargetPath)
		if err != nil || !strings.HasPrefix(path, "browser-registration/") {
			return fmt.Errorf("registration source target %q is not a safe browser-registration path", source.TargetPath)
		}
		if _, duplicate := metadata[path]; duplicate {
			return fmt.Errorf("registration source target %q is duplicated", path)
		}
		source.TargetPath = path
		metadata[path] = source
	}
	if len(metadata) != len(paths) {
		return fmt.Errorf("registration review inventory has %d source(s), package has %d", len(metadata), len(paths))
	}
	profiles := map[string]*registrationprofile.Profile{}
	for _, path := range paths {
		source, ok := metadata[path]
		if !ok {
			return fmt.Errorf("registration review is missing %s", path)
		}
		profileData, _, err := evidencefile.ReadRegular(filepath.Join(exampleDir, filepath.FromSlash(path)), registrationprofile.MaxProfileBytes)
		if err != nil {
			return err
		}
		profileRawDigest := sha256.Sum256(profileData)
		if source.SHA256 != hex.EncodeToString(profileRawDigest[:]) {
			return fmt.Errorf("registration source %s digest mismatch", path)
		}
		value, err := registrationprofile.Parse(profileData)
		if err != nil {
			return fmt.Errorf("registration source %s: %w", path, err)
		}
		if err := registrationprofile.ValidateAt(value, at); err != nil {
			return fmt.Errorf("registration source %s: %w", path, err)
		}
		profileDigest, err := registrationprofile.Digest(value)
		if err != nil {
			return err
		}
		reviewPath := packageartifacts.BrowserRegistrationBundlePath(path)
		if source.ReviewPath != reviewPath {
			return fmt.Errorf("registration source %s review path mismatch", path)
		}
		reviewData, _, err := evidencefile.ReadRegular(filepath.Join(exampleDir, filepath.FromSlash(reviewPath)), evidencefile.DefaultMaxBytes)
		if err != nil {
			return err
		}
		reviewRawDigest := sha256.Sum256(reviewData)
		if source.ReviewSHA256 != hex.EncodeToString(reviewRawDigest[:]) {
			return fmt.Errorf("registration source %s review digest mismatch", path)
		}
		var bundle registrationreview.Bundle
		if err := evidencefile.DecodeStrict(reviewData, &bundle); err != nil {
			return fmt.Errorf("registration source %s review bundle: %w", path, err)
		}
		if err := registrationreview.Verify(&bundle, at); err != nil {
			return fmt.Errorf("registration source %s review bundle: %w", path, err)
		}
		if bundle.ProfileDigest != profileDigest || source.ProfileDigest != profileDigest {
			return fmt.Errorf("registration source %s canonical profile digest mismatch", path)
		}
		expiresAt, _ := registrationprofile.ExpiresAt(value)
		flows := registrationprofile.SortedFlowNames(value)
		if source.Title != value.Info.Title || source.Provenance != value.Evidence.Source || source.Lifecycle != "active" || source.ExpiresAt != expiresAt.Format(time.RFC3339) || !equalSortedStrings(source.Flows, flows) || !equalSortedStrings(source.Origins, registrationprofile.Origins(value)) {
			return fmt.Errorf("registration source %s title, provenance, flow, origin, lifecycle, or expiry evidence does not match", path)
		}
		if !browserAuthenticationBindingPattern.MatchString(strings.TrimSpace(source.ID)) || strings.TrimSpace(source.Provenance) == "" {
			return fmt.Errorf("registration source %s requires a symbolic ID and provenance", path)
		}
		if len(source.FlowCredentialSlots) != len(flows) {
			return fmt.Errorf("registration source %s credential-slot inventory does not exactly match its flows", path)
		}
		for _, flowName := range flows {
			if !equalSortedStrings(source.FlowCredentialSlots[flowName], registrationFlowSlots(value.Flows[flowName])) {
				return fmt.Errorf("registration source %s flow %s credential-slot evidence does not match", path, flowName)
			}
		}
		profiles[path] = value
	}

	calls := map[string]browserRegistrationReviewedCall{}
	for _, call := range review.Calls {
		if !browserAuthenticationBindingPattern.MatchString(strings.TrimSpace(call.Step)) {
			return fmt.Errorf("registration call step must be a portable symbolic name")
		}
		if _, duplicate := calls[call.Step]; duplicate {
			return fmt.Errorf("registration call for step %s is duplicated", call.Step)
		}
		calls[call.Step] = call
	}
	var stepErrors []string
	expectedCalls := map[string]bool{}
	browserworkflow.WalkEffectiveSources(intent, func(step *rollout.Step, effectiveSource string) {
		if step == nil || !strings.EqualFold(strings.TrimSpace(step.Type), "browser_registration") {
			return
		}
		name := firstNonEmpty(strings.TrimSpace(step.Name), "<unnamed>")
		expectedCalls[name] = true
		ref := normalizeAPISourceRef(effectiveSource)
		value := profiles[ref]
		if value == nil {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s references unreviewed registration source %q", name, ref))
			return
		}
		flow, ok := value.Flows[strings.TrimSpace(step.RegistrationFlow)]
		if !ok {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s invents registration flow %q", name, step.RegistrationFlow))
			return
		}
		if !exactRegistrationBindings(step.CredentialBindings, registrationFlowSlots(flow)) {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s credential bindings do not exactly cover the selected flow", name))
		}
		call, ok := calls[name]
		if !ok || !registrationCallMatchesStep(call, step, ref) {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s does not exactly match registration review call evidence", name))
		}
	})
	if len(calls) != len(expectedCalls) {
		stepErrors = append(stepErrors, "registration call inventory must exactly match registration steps")
	}
	if len(stepErrors) > 0 {
		sort.Strings(stepErrors)
		return fmt.Errorf("%s", strings.Join(stepErrors, "; "))
	}
	return nil
}

func registrationCallMatchesStep(call browserRegistrationReviewedCall, step *rollout.Step, source string) bool {
	if step == nil || step.Timeout == nil || call.Timeout != *step.Timeout || call.Source != source || call.Flow != step.RegistrationFlow ||
		call.Approval != step.RegistrationApproval || call.DuplicatePrevention != step.DuplicatePrevention || call.OnDuplicate != step.OnDuplicate ||
		call.AmbiguousOutcome != step.AmbiguousOutcome || call.CleanupDisposition != step.CleanupDisposition || len(call.CredentialBindings) != len(step.CredentialBindings) {
		return false
	}
	for slot, binding := range step.CredentialBindings {
		if call.CredentialBindings[slot] != binding {
			return false
		}
	}
	return true
}

func intentHasBrowserRegistrationStep(intent *rollout.Intent) bool {
	found := false
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step != nil && strings.EqualFold(strings.TrimSpace(step.Type), "browser_registration") {
			found = true
		}
	})
	return found
}
