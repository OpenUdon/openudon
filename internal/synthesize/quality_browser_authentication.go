package synthesize

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/browserauthentication"
)

const browserAuthenticationReviewVersion = "openudon.browser-authentication-review.v1"

type browserAuthenticationReview struct {
	Version   string                                `json:"version"`
	Approvals []string                              `json:"authentication_approvals"`
	Sessions  []browserAuthenticationSessionBinding `json:"session_bindings"`
	Sources   []browserAuthenticationReviewedSource `json:"sources"`
}

type browserAuthenticationSessionBinding struct {
	Step    string `json:"step"`
	Session string `json:"session"`
}

type browserAuthenticationReviewedSource struct {
	ID                  string              `json:"id"`
	TargetPath          string              `json:"target_path"`
	SHA256              string              `json:"sha256"`
	SourceSHA256        string              `json:"source_sha256,omitempty"`
	Title               string              `json:"title,omitempty"`
	Flows               []string            `json:"flows"`
	FlowCredentialSlots map[string][]string `json:"flow_credential_slots"`
	Origins             []string            `json:"origins"`
	Lifecycle           string              `json:"lifecycle"`
	ExpiresAt           string              `json:"expires_at"`
	Provenance          string              `json:"provenance"`
}

var browserAuthenticationBindingPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)

func assessBrowserAuthenticationSources(report *QualityReport, exampleDir string, intent *rollout.Intent) {
	paths, err := packageartifacts.CollectBrowserAuthenticationProfilePaths(exampleDir)
	if err != nil {
		report.add("browser.authentication.sources", "fail", "browser authentication profiles could not be inventoried", err.Error())
		return
	}
	if len(paths) == 0 {
		if intentHasBrowserAuthenticationStep(intent) {
			report.add("browser.authentication.sources", "fail", "browser authentication requires a packaged profile", "Select a reviewed uws.browser-authentication.1.0 profile through iCoT.")
			return
		}
		report.add("browser.authentication.sources", "pass", "browser authentication review is not required", "")
		return
	}
	metadataPath := filepath.Join(exampleDir, filepath.FromSlash(packageartifacts.BrowserAuthenticationReviewPath))
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		report.add("browser.authentication.review", "fail", "browser authentication review evidence is required", err.Error())
		return
	}
	var review browserAuthenticationReview
	if err := json.Unmarshal(data, &review); err != nil {
		report.add("browser.authentication.review", "fail", "browser authentication review evidence must be valid JSON", err.Error())
		return
	}
	if err := validateBrowserAuthenticationReview(exampleDir, paths, intent, review, time.Now().UTC()); err != nil {
		report.add("browser.authentication.review", "fail", "browser authentication review evidence is invalid", err.Error())
		return
	}
	report.add("browser.authentication.sources", "pass", fmt.Sprintf("%d verified active browser authentication profile(s) are packaged", len(paths)), strings.Join(paths, ", "))
	report.add("browser.authentication.review", "pass", "authentication flows, digests, named sessions, symbolic credentials, timeout, and authoring approvals agree", "")
}

func validateBrowserAuthenticationReview(exampleDir string, paths []string, intent *rollout.Intent, review browserAuthenticationReview, at time.Time) error {
	if review.Version != browserAuthenticationReviewVersion {
		return fmt.Errorf("version must be %q", browserAuthenticationReviewVersion)
	}
	metadata := map[string]browserAuthenticationReviewedSource{}
	for _, source := range review.Sources {
		path, err := packageartifacts.CleanRelativePath(source.TargetPath)
		if err != nil || !strings.HasPrefix(path, "browser-authentication/") {
			return fmt.Errorf("authentication source target %q is not a safe browser-authentication path", source.TargetPath)
		}
		if _, duplicate := metadata[path]; duplicate {
			return fmt.Errorf("authentication source target %q is duplicated", path)
		}
		source.TargetPath = path
		metadata[path] = source
	}
	if len(metadata) != len(paths) {
		return fmt.Errorf("authentication review inventory has %d source(s), package has %d", len(metadata), len(paths))
	}
	profiles := map[string]*authprofile.Profile{}
	for _, path := range paths {
		source, ok := metadata[path]
		if !ok {
			return fmt.Errorf("authentication review is missing %s", path)
		}
		absolute := filepath.Join(exampleDir, filepath.FromSlash(path))
		data, err := os.ReadFile(absolute)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(data)
		if !strings.EqualFold(source.SHA256, hex.EncodeToString(digest[:])) {
			return fmt.Errorf("authentication source %s digest mismatch", path)
		}
		value, err := authprofile.LoadFile(absolute)
		if err != nil {
			return fmt.Errorf("authentication source %s: %w", path, err)
		}
		if err := authprofile.ValidateAt(value, at); err != nil {
			return fmt.Errorf("authentication source %s: %w", path, err)
		}
		expiresAt, _ := authprofile.ExpiresAt(value)
		flows := authprofile.SortedFlowNames(value)
		if source.Lifecycle != "active" || source.ExpiresAt != expiresAt.Format(time.RFC3339) || !equalSortedStrings(source.Flows, flows) || !equalSortedStrings(source.Origins, authprofile.Origins(value)) {
			return fmt.Errorf("authentication source %s flow, origin, lifecycle, or expiry evidence does not match", path)
		}
		if strings.TrimSpace(source.ID) == "" || strings.TrimSpace(source.Provenance) == "" {
			return fmt.Errorf("authentication source %s requires an ID and provenance", path)
		}
		if len(source.FlowCredentialSlots) != len(flows) {
			return fmt.Errorf("authentication source %s credential-slot inventory does not exactly match its flows", path)
		}
		for _, flowName := range flows {
			if !equalSortedStrings(source.FlowCredentialSlots[flowName], authenticationFlowSlots(value.Flows[flowName])) {
				return fmt.Errorf("authentication source %s flow %s credential-slot evidence does not match", path, flowName)
			}
		}
		profiles[path] = value
	}
	approvals := stringSet(review.Approvals)
	if len(approvals) != len(review.Approvals) {
		return fmt.Errorf("authentication approval inventory contains duplicates")
	}
	bindings := map[string]string{}
	for _, binding := range review.Sessions {
		if strings.TrimSpace(binding.Step) == "" || !browserAuthenticationBindingPattern.MatchString(strings.TrimSpace(binding.Session)) {
			return fmt.Errorf("authentication session binding requires a step and safe symbolic session name")
		}
		if _, duplicate := bindings[binding.Step]; duplicate {
			return fmt.Errorf("authentication session binding for step %s is duplicated", binding.Step)
		}
		bindings[binding.Step] = binding.Session
	}
	expectedSessions := map[string]string{}
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step != nil && strings.TrimSpace(step.BrowserSession) != "" {
			expectedSessions[strings.TrimSpace(step.Name)] = strings.TrimSpace(step.BrowserSession)
		}
	})
	if len(bindings) != len(expectedSessions) {
		return fmt.Errorf("authentication review session inventory has %d binding(s), intent has %d", len(bindings), len(expectedSessions))
	}
	for step, session := range expectedSessions {
		if bindings[step] != session {
			return fmt.Errorf("authentication review session binding for step %s does not match intent", step)
		}
	}
	var stepErrors []string
	expectedApprovals := map[string]bool{}
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step == nil || !strings.EqualFold(strings.TrimSpace(step.Type), "browser_authentication") {
			return
		}
		name := firstNonEmpty(step.Name, "<unnamed>")
		expectedApprovals[name] = true
		ref := normalizeAPISourceRef(firstNonEmpty(step.Source, step.OpenAPI, intentSourceRef(intent)))
		value := profiles[ref]
		if value == nil {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s references unreviewed authentication source %q", name, ref))
			return
		}
		flow, ok := value.Flows[strings.TrimSpace(step.AuthenticationFlow)]
		if !ok {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s invents authentication flow %q", name, step.AuthenticationFlow))
			return
		}
		if !exactAuthenticationBindings(step.CredentialBindings, authenticationFlowSlots(flow)) {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s credential bindings do not exactly cover the selected flow", name))
		}
		if step.Timeout == nil || *step.Timeout <= 0 || *step.Timeout > 600 {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s requires a timeout from 1 through 600 seconds", name))
		}
		if !browserAuthenticationBindingPattern.MatchString(strings.TrimSpace(step.BrowserSession)) || bindings[name] != step.BrowserSession {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s named session does not match review evidence", name))
		}
		if !approvals[name] {
			stepErrors = append(stepErrors, fmt.Sprintf("step %s lacks operation-specific browser authentication authoring approval", name))
		}
	})
	if len(approvals) != len(expectedApprovals) {
		stepErrors = append(stepErrors, "authentication approval inventory must exactly match authentication steps")
	}
	if len(stepErrors) > 0 {
		sort.Strings(stepErrors)
		return fmt.Errorf("%s", strings.Join(stepErrors, "; "))
	}
	return nil
}

func authenticationFlowSlots(flow browserauthentication.Flow) []string {
	set := map[string]bool{}
	for _, step := range flow.Sequence {
		if step.TypeCredential != nil && strings.TrimSpace(step.TypeCredential.Slot) != "" {
			set[step.TypeCredential.Slot] = true
		}
		if step.Challenge != nil && strings.TrimSpace(step.Challenge.Slot) != "" {
			set[step.Challenge.Slot] = true
		}
	}
	values := make([]string, 0, len(set))
	for slot := range set {
		values = append(values, slot)
	}
	sort.Strings(values)
	return values
}

func exactAuthenticationBindings(bindings map[string]string, slots []string) bool {
	if len(bindings) != len(slots) {
		return false
	}
	for _, slot := range slots {
		if !browserAuthenticationBindingPattern.MatchString(strings.TrimSpace(bindings[slot])) {
			return false
		}
	}
	return true
}

func intentHasBrowserAuthenticationStep(intent *rollout.Intent) bool {
	found := false
	walkIntentSteps(intentSteps(intent), func(step *rollout.Step) {
		if step != nil && strings.EqualFold(strings.TrimSpace(step.Type), "browser_authentication") {
			found = true
		}
	})
	return found
}
