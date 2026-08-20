package trustedrunner

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browserworkflow"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type browserSourceApprovals struct {
	Version           string            `json:"version"`
	Route             string            `json:"route"`
	SessionPosture    string            `json:"session_posture"`
	MutationApprovals []string          `json:"mutation_approvals,omitempty"`
	Sources           []json.RawMessage `json:"sources"`
}

type browserAuthenticationApprovals struct {
	Version   string            `json:"version"`
	Approvals []string          `json:"authentication_approvals"`
	Sessions  []json.RawMessage `json:"session_bindings"`
	Sources   []json.RawMessage `json:"sources"`
}

func buildBrowserRunConfig(packageRoot, driver string, driverArgs, env []string, dryRun bool) (*udonrunner.BrowserConfig, error) {
	planData, _, err := evidencefile.ReadRegular(filepath.Join(packageRoot, "expected", "plan.json"), evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("read browser runtime plan: %w", err)
	}
	var plan synthesize.WorkflowPlan
	if err := evidencefile.DecodeStrict(planData, &plan); err != nil {
		return nil, fmt.Errorf("decode browser runtime plan: %w", err)
	}
	hasBrowserSteps := false
	for _, step := range plan.Steps {
		kind := strings.ToLower(strings.TrimSpace(step.Type))
		hasBrowserSteps = hasBrowserSteps || kind == "browser" || kind == "browser_authentication"
	}
	if !hasBrowserSteps {
		if strings.TrimSpace(driver) != "" || len(driverArgs) != 0 {
			return nil, fmt.Errorf("--browser-driver and --browser-driver-arg require a browser workflow")
		}
		return nil, nil
	}
	browserPaths, err := packageartifacts.CollectBrowserProfilePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	authenticationPaths, err := packageartifacts.CollectBrowserAuthenticationProfilePaths(packageRoot)
	if err != nil {
		return nil, err
	}
	intent, err := rollout.ParseIntentFile(filepath.Join(packageRoot, filepath.FromSlash(rollout.IntentPath)))
	if err != nil {
		return nil, fmt.Errorf("read browser runtime intent: %w", err)
	}
	driver = strings.TrimSpace(driver)
	if !dryRun && driver == "" {
		return nil, fmt.Errorf("browser workflow execution requires --browser-driver")
	}

	protocolRank := 1
	for _, relative := range browserPaths {
		value, err := readBrowserProfile(packageRoot, relative)
		if err != nil {
			return nil, err
		}
		switch value.Schema {
		case "uws.browser.1.5":
		case "uws.browser.1.6", "uws.browser.1.7":
			protocolRank = max(protocolRank, 3)
		default:
			return nil, fmt.Errorf("browser profile %s has unsupported discriminator %q", relative, value.Schema)
		}
	}
	for _, relative := range authenticationPaths {
		value, err := readBrowserAuthenticationProfile(packageRoot, relative)
		if err != nil {
			return nil, err
		}
		switch value.Profile {
		case "uws.browser-authentication.1.0":
			protocolRank = max(protocolRank, 2)
		case "uws.browser-authentication.1.1":
			protocolRank = max(protocolRank, 3)
		default:
			return nil, fmt.Errorf("browser authentication profile %s has unsupported discriminator %q", relative, value.Profile)
		}
	}

	credentials := map[string]bool{}
	hasNamedSession := false
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		if strings.TrimSpace(step.BrowserSession) != "" {
			hasNamedSession = true
		}
		if strings.EqualFold(strings.TrimSpace(step.Type), "browser_authentication") {
			for _, binding := range step.CredentialBindings {
				if value := strings.TrimSpace(binding); value != "" {
					credentials[value] = true
				}
			}
		}
	})
	if hasNamedSession {
		protocolRank = max(protocolRank, 2)
	}

	approvedOperations, sessionPosture, err := readBrowserOperationApprovals(packageRoot, len(browserPaths) != 0)
	if err != nil {
		return nil, err
	}
	approvedOperations, err = runtimeApprovalIDs(approvedOperations)
	if err != nil {
		return nil, fmt.Errorf("browser operation approvals: %w", err)
	}
	approvedAuthentication, err := readBrowserAuthenticationApprovals(packageRoot, len(authenticationPaths) != 0)
	if err != nil {
		return nil, err
	}
	approvedAuthentication, err = runtimeApprovalIDs(approvedAuthentication)
	if err != nil {
		return nil, fmt.Errorf("browser authentication approvals: %w", err)
	}
	analysis := browserworkflow.Analyze(intent)
	externalSessions := analysis.ExternalSessions()
	if sessionPosture == "opaque-runtime-binding-required" && len(externalSessions) == 0 {
		return nil, fmt.Errorf("legacy opaque browser sessions cannot execute through openudon run; select a named browser_session and rebuild")
	}

	credentialBindings := sortedSet(credentials)
	credentialEnvironment := make([]udonrunner.EnvironmentBinding, 0, len(credentialBindings))
	for _, name := range credentialBindings {
		credentialEnvironment = append(credentialEnvironment, udonrunner.EnvironmentBinding{Name: name, Environment: udonrunner.CredentialEnvironmentName(name)})
	}
	sessionEnvironment := make([]udonrunner.EnvironmentBinding, 0, len(externalSessions))
	for _, name := range externalSessions {
		sessionEnvironment = append(sessionEnvironment, udonrunner.EnvironmentBinding{Name: name, Environment: udonrunner.SessionEnvironmentName(name)})
	}
	return &udonrunner.BrowserConfig{
		DriverPath:             driver,
		DriverArgs:             append([]string(nil), driverArgs...),
		DriverEnvironment:      udonrunner.AvailableBrowserDriverEnvironment(env),
		Protocol:               fmt.Sprintf("v%d", protocolRank),
		CredentialEnvironment:  credentialEnvironment,
		SessionEnvironment:     sessionEnvironment,
		ApprovedOperations:     approvedOperations,
		ApprovedAuthentication: approvedAuthentication,
	}, nil
}

func readBrowserProfile(root, relative string) (*profile.Profile, error) {
	data, _, err := evidencefile.ReadRegular(filepath.Join(root, filepath.FromSlash(relative)), evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("read browser profile %s: %w", relative, err)
	}
	if strings.EqualFold(filepath.Ext(relative), ".json") {
		return profile.ParseJSON(data)
	}
	return profile.ParseYAML(data)
}

func readBrowserAuthenticationProfile(root, relative string) (*authprofile.Profile, error) {
	data, _, err := evidencefile.ReadRegular(filepath.Join(root, filepath.FromSlash(relative)), evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("read browser authentication profile %s: %w", relative, err)
	}
	return authprofile.Parse(data)
}

func readBrowserOperationApprovals(root string, required bool) ([]string, string, error) {
	if !required {
		return []string{}, "none", nil
	}
	data, _, err := evidencefile.ReadRegular(filepath.Join(root, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath)), evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, "", fmt.Errorf("read browser source review: %w", err)
	}
	var review browserSourceApprovals
	if err := evidencefile.DecodeStrict(data, &review); err != nil {
		return nil, "", fmt.Errorf("decode browser source review: %w", err)
	}
	if review.Version != "openudon.browser-source-review.v1" {
		return nil, "", fmt.Errorf("unsupported browser source review version %q", review.Version)
	}
	return sortedUniqueRuntimeIDs(review.MutationApprovals), strings.TrimSpace(review.SessionPosture), nil
}

func readBrowserAuthenticationApprovals(root string, required bool) ([]string, error) {
	if !required {
		return []string{}, nil
	}
	data, _, err := evidencefile.ReadRegular(filepath.Join(root, filepath.FromSlash(packageartifacts.BrowserAuthenticationReviewPath)), evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, fmt.Errorf("read browser authentication review: %w", err)
	}
	var review browserAuthenticationApprovals
	if err := evidencefile.DecodeStrict(data, &review); err != nil {
		return nil, fmt.Errorf("decode browser authentication review: %w", err)
	}
	if review.Version != "openudon.browser-authentication-review.v1" {
		return nil, fmt.Errorf("unsupported browser authentication review version %q", review.Version)
	}
	return sortedUniqueRuntimeIDs(review.Approvals), nil
}

func walkIntentSteps(steps []*rollout.Step, visit func(*rollout.Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		visit(step)
		walkIntentSteps(step.Steps, visit)
		for _, branch := range step.Cases {
			if branch != nil {
				walkIntentSteps(branch.Steps, visit)
			}
		}
		if step.Default != nil {
			walkIntentSteps(step.Default.Steps, visit)
		}
	}
}

func sortedUniqueRuntimeIDs(values []string) []string {
	set := map[string]bool{}
	for _, value := range values {
		if name := strings.TrimSpace(value); name != "" {
			set[name] = true
		}
	}
	return sortedSet(set)
}

func runtimeApprovalIDs(values []string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := map[string]string{}
	for _, value := range values {
		id := browserworkflow.RuntimeOperationID(value)
		if id == "" {
			return nil, fmt.Errorf("approval %q has no runtime operation ID", value)
		}
		if prior, ok := seen[id]; ok && prior != value {
			return nil, fmt.Errorf("approvals %q and %q lower to the same runtime operation ID", prior, value)
		}
		seen[id] = value
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
