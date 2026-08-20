package synthesize

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/browserworkflow"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const browserSourceReviewVersion = "openudon.browser-source-review.v1"

type browserSourceReview struct {
	Version           string                  `json:"version"`
	Route             string                  `json:"route"`
	SessionPosture    string                  `json:"session_posture"`
	MutationApprovals []string                `json:"mutation_approvals,omitempty"`
	Sources           []browserReviewedSource `json:"sources"`
}

type browserReviewedSource struct {
	ID                 string                  `json:"id"`
	Release            string                  `json:"release,omitempty"`
	TargetPath         string                  `json:"target_path"`
	SHA256             string                  `json:"sha256"`
	SourceSHA256       string                  `json:"source_sha256,omitempty"`
	Title              string                  `json:"title,omitempty"`
	Actions            []string                `json:"actions"`
	Origins            []string                `json:"origins"`
	Lifecycle          string                  `json:"lifecycle"`
	ExpiresAt          string                  `json:"expires_at,omitempty"`
	LoginStateRequired bool                    `json:"login_state_required,omitempty"`
	Provenance         string                  `json:"provenance"`
	Registry           string                  `json:"registry,omitempty"`
	Coordinate         string                  `json:"coordinate,omitempty"`
	Verifications      []browserverify.Summary `json:"verifications,omitempty"`
}

func assessBrowserSources(report *QualityReport, exampleDir string, intent *rollout.Intent) {
	paths, err := packageartifacts.CollectBrowserProfilePaths(exampleDir)
	if err != nil {
		report.add("browser.sources", "fail", "browser profile inputs could not be inventoried", err.Error())
		return
	}
	if len(paths) == 0 {
		if intentHasBrowserStep(intent) {
			report.add("browser.sources", "fail", "browser workflow requires a packaged browser profile", "Add a verified uws.browser.1.5 profile through iCoT source selection.")
			return
		}
		report.add("browser.sources", "pass", "browser source review is not required", "")
		return
	}
	metadataPath := filepath.Join(exampleDir, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath))
	metadataBytes, err := readBrowserSourceReviewFile(metadataPath)
	if err != nil {
		report.add("browser.review", "fail", "browser source review evidence is required", err.Error())
		return
	}
	var review browserSourceReview
	if err := decodeBrowserSourceReview(metadataBytes, &review); err != nil {
		report.add("browser.review", "fail", "browser source review evidence must be valid JSON", err.Error())
		return
	}
	if err := validateBrowserSourceReview(exampleDir, paths, intent, review, time.Now().UTC()); err != nil {
		report.add("browser.review", "fail", "browser source review evidence is invalid", err.Error())
		return
	}
	report.add("browser.sources", "pass", fmt.Sprintf("%d verified active browser profile(s) are packaged", len(paths)), strings.Join(paths, ", "))
	report.add("browser.review", "pass", "browser digests, origins, actions, lifecycle, session posture, and mutation approvals agree", "")
	verificationCount := 0
	for _, source := range review.Sources {
		verificationCount += len(source.Verifications)
	}
	if verificationCount == 0 {
		report.add("browser.verification", "pass", "optional value-free browser verification is not attached", "Portability is review confidence, not a universal runtime requirement.")
	} else {
		report.add("browser.verification", "pass", fmt.Sprintf("%d value-free browser verification report(s) are profile-bound and successful", verificationCount), "Current-page and portability facts were independently revalidated from declared profile paths and fixed diagnostics.")
	}
}

func validateBrowserSourceReview(exampleDir string, paths []string, intent *rollout.Intent, review browserSourceReview, at time.Time) error {
	if review.Version != browserSourceReviewVersion {
		return fmt.Errorf("version must be %q", browserSourceReviewVersion)
	}
	if review.Route != "browser" && intentHasBrowserStep(intent) {
		return fmt.Errorf("route must be browser when the active workflow uses a browser action")
	}
	if review.SessionPosture != "none" && review.SessionPosture != "opaque-runtime-binding-required" {
		return fmt.Errorf("session_posture must be none or opaque-runtime-binding-required")
	}
	metadata := map[string]browserReviewedSource{}
	for _, source := range review.Sources {
		path, err := packageartifacts.CleanRelativePath(source.TargetPath)
		if err != nil || !strings.HasPrefix(path, "browser-profiles/") {
			return fmt.Errorf("browser source target %q is not a safe browser-profiles path", source.TargetPath)
		}
		if _, duplicate := metadata[path]; duplicate {
			return fmt.Errorf("browser source target %q is duplicated", path)
		}
		source.TargetPath = path
		metadata[path] = source
	}
	if len(metadata) != len(paths) {
		return fmt.Errorf("browser review inventory has %d source(s), package has %d", len(metadata), len(paths))
	}
	profiles := map[string]*profile.Profile{}
	verificationByPath := map[string][]browserverify.Summary{}
	totalVerifications := 0
	for _, path := range paths {
		source, ok := metadata[path]
		if !ok {
			return fmt.Errorf("browser review is missing %s", path)
		}
		data, err := os.ReadFile(filepath.Join(exampleDir, filepath.FromSlash(path)))
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		actualDigest := hex.EncodeToString(digest[:])
		if !strings.EqualFold(source.SHA256, actualDigest) {
			return fmt.Errorf("browser source %s digest mismatch", path)
		}
		if source.SourceSHA256 != "" {
			if _, err := hex.DecodeString(strings.TrimPrefix(source.SourceSHA256, "sha256:")); err != nil || len(strings.TrimPrefix(source.SourceSHA256, "sha256:")) != 64 {
				return fmt.Errorf("browser source %s source_sha256 is invalid", path)
			}
		}
		value, err := loadBrowserProfile(filepath.Join(exampleDir, filepath.FromSlash(path)))
		if err != nil {
			return fmt.Errorf("browser source %s: %w", path, err)
		}
		if err := validatePackagedBrowserProfile(value); err != nil {
			return fmt.Errorf("browser source %s: %w", path, err)
		}
		expiresAt, err := packagedBrowserProfileExpiry(value)
		if err != nil {
			return fmt.Errorf("browser source %s: %w", path, err)
		}
		if !expiresAt.After(at) {
			return fmt.Errorf("browser source %s expired at %s", path, expiresAt.Format(time.RFC3339))
		}
		if source.Lifecycle != "active" || source.ExpiresAt != expiresAt.Format(time.RFC3339) {
			return fmt.Errorf("browser source %s lifecycle or expiry evidence does not match the profile", path)
		}
		if strings.TrimSpace(source.ID) == "" || strings.TrimSpace(source.Provenance) == "" {
			return fmt.Errorf("browser source %s requires an ID and provenance", path)
		}
		if (source.Registry == "") != (source.Coordinate == "") {
			return fmt.Errorf("browser source %s registry and coordinate must be recorded together", path)
		}
		if !equalSortedStrings(source.Actions, value.SortedActionNames()) || !equalSortedStrings(source.Origins, []string(value.Info.Origin)) || source.LoginStateRequired != value.Info.LoginStateRequired {
			return fmt.Errorf("browser source %s action, origin, or login-state evidence does not match", path)
		}
		profiles[path] = value
		if len(source.Verifications) > browserverify.MaxReportsPerProfile {
			return fmt.Errorf("browser source %s has more than %d verification reports", path, browserverify.MaxReportsPerProfile)
		}
		totalVerifications += len(source.Verifications)
		if totalVerifications > browserverify.MaxReports {
			return fmt.Errorf("browser source review has more than %d verification reports", browserverify.MaxReports)
		}
		logicalReports := map[string]bool{}
		for index, summary := range source.Verifications {
			if err := browserverify.ValidateSummary(value, summary, at); err != nil {
				return fmt.Errorf("browser source %s verification[%d]: %w", path, index, err)
			}
			if !summary.OK {
				return fmt.Errorf("browser source %s verification[%d] records a failed check", path, index)
			}
			key := browserverify.LogicalKey(summary)
			if logicalReports[key] {
				return fmt.Errorf("browser source %s verification action set is duplicated", path)
			}
			logicalReports[key] = true
			verificationByPath[path] = append(verificationByPath[path], summary)
		}
	}
	approvals := stringSet(review.MutationApprovals)
	if len(approvals) != len(review.MutationApprovals) {
		return fmt.Errorf("browser mutation approval inventory contains duplicates")
	}
	expectedApprovals := map[string]bool{}
	usedActions := map[string]map[string]bool{}
	var stepErrors []string
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step == nil || !strings.EqualFold(strings.TrimSpace(step.Type), "browser") {
			return
		}
		ref := normalizeAPISourceRef(firstNonEmpty(step.Source, step.OpenAPI, intentSourceRef(intent)))
		value := profiles[ref]
		if value == nil {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s references unreviewed browser source %q", firstNonEmpty(step.Name, "<unnamed>"), ref))
			return
		}
		action, ok := value.Actions[strings.TrimSpace(step.Operation)]
		if !ok {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s invents browser action %q", firstNonEmpty(step.Name, "<unnamed>"), step.Operation))
			return
		}
		if usedActions[ref] == nil {
			usedActions[ref] = map[string]bool{}
		}
		usedActions[ref][strings.TrimSpace(step.Operation)] = true
		mutating := false
		for _, effect := range action.SideEffects {
			if effect != profile.SideEffectReadOnly {
				mutating = true
			}
		}
		if mutating && (!action.ConfirmationPolicy.Required || strings.TrimSpace(action.ConfirmationPolicy.Prompt) == "") {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s selects a mutating action without profile confirmation policy", firstNonEmpty(step.Name, "<unnamed>")))
		}
		if mutating {
			expectedApprovals[strings.TrimSpace(step.Name)] = true
		}
		if mutating && !approvals[strings.TrimSpace(step.Name)] {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s mutates browser state without operation-specific authoring approval", firstNonEmpty(step.Name, "<unnamed>")))
		}
		if value.Info.LoginStateRequired && !intentHasPrecedingAuthenticationSession(intent, step) && review.SessionPosture != "opaque-runtime-binding-required" {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s requires an opaque operator-owned runtime session binding", firstNonEmpty(step.Name, "<unnamed>")))
		}
	})
	if len(approvals) != len(expectedApprovals) {
		stepErrors = append(stepErrors, "browser mutation approval inventory must exactly match mutating browser steps")
	} else {
		for name := range approvals {
			if !expectedApprovals[name] {
				stepErrors = append(stepErrors, fmt.Sprintf("browser mutation approval %q does not name a mutating browser step", name))
			}
		}
	}
	if len(stepErrors) > 0 {
		sort.Strings(stepErrors)
		return fmt.Errorf("%s", strings.Join(stepErrors, "; "))
	}
	for path, summaries := range verificationByPath {
		covered := map[string]bool{}
		for _, summary := range summaries {
			for _, action := range summary.Actions {
				covered[action] = true
			}
		}
		for action := range usedActions[path] {
			if !covered[action] {
				return fmt.Errorf("browser source %s verification does not cover selected action %q", path, action)
			}
		}
	}
	return nil
}

func decodeBrowserSourceReview(data []byte, target *browserSourceReview) error {
	return browserverify.DecodeStrictJSON(data, target)
}

func readBrowserSourceReviewFile(path string) ([]byte, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if before.Mode()&os.ModeSymlink != 0 || !before.Mode().IsRegular() {
		return nil, fmt.Errorf("browser source review must be a non-symlink regular file")
	}
	if before.Size() <= 0 || before.Size() > browserverify.MaxReviewBytes {
		return nil, fmt.Errorf("browser source review size must be between 1 and %d bytes", browserverify.MaxReviewBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return nil, fmt.Errorf("browser source review changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, browserverify.MaxReviewBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || len(data) > browserverify.MaxReviewBytes {
		return nil, fmt.Errorf("browser source review size must be between 1 and %d bytes", browserverify.MaxReviewBytes)
	}
	after, err := file.Stat()
	if err != nil || after.Size() != opened.Size() || !after.ModTime().Equal(opened.ModTime()) {
		return nil, fmt.Errorf("browser source review changed while reading")
	}
	return data, nil
}

func intentHasPrecedingAuthenticationSession(intent *rollout.Intent, action *rollout.Step) bool {
	return browserworkflow.Analyze(intent).EstablishedBefore(action)
}

func validatePackagedBrowserProfile(value *profile.Profile) error {
	if value == nil {
		return fmt.Errorf("profile is empty")
	}
	for actionName, action := range value.Actions {
		if browserArtifactNameSensitive(actionName) {
			return fmt.Errorf("action %q is credential, session, or raw-capture shaped", actionName)
		}
		for parameter := range browserProfileSchemaProperties(action.Parameters) {
			if browserArtifactNameSensitive(parameter) {
				return fmt.Errorf("action %q parameter %q is credential, session, or raw-capture shaped", actionName, parameter)
			}
		}
		for output := range action.Outputs {
			if browserArtifactNameSensitive(output) {
				return fmt.Errorf("action %q output %q is credential, session, or raw-capture shaped", actionName, output)
			}
		}
	}
	return nil
}

func packagedBrowserProfileExpiry(value *profile.Profile) (time.Time, error) {
	verified, err := time.Parse(time.RFC3339, value.Verification.LastVerifiedAt)
	if err != nil {
		return time.Time{}, err
	}
	expires, err := value.ExpiresAfter.AddTo(verified)
	if err != nil {
		return time.Time{}, err
	}
	return expires.UTC().Round(0), nil
}

func browserProfileSchemaProperties(schema profile.JSONSchema) map[string]any {
	properties, _ := schema["properties"].(map[string]any)
	return properties
}

func browserArtifactNameSensitive(value string) bool {
	normalized := strings.ToLower(strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(value))
	for _, marker := range []string{"cookie", "session", "storage", "dom", "html", "screenshot", "raw_capture", "raw_browser", "password", "secret", "credential", "access_token", "refresh_token"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func intentHasBrowserStep(intent *rollout.Intent) bool {
	found := false
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step != nil && strings.EqualFold(strings.TrimSpace(step.Type), "browser") {
			found = true
		}
	})
	return found
}

func intentSourceRef(intent *rollout.Intent) string {
	if intent == nil {
		return ""
	}
	return firstNonEmpty(intent.Source, intent.OpenAPI)
}

func equalSortedStrings(left, right []string) bool {
	left = sortedUnique(left)
	right = sortedUnique(right)
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
