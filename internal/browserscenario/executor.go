package browserscenario

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browserverify"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot"
	"github.com/OpenUdon/openudon/internal/processgroup"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

const (
	scenarioCommandOutputLimit = 1 << 20
	probeDeadline              = 30 * time.Second
	buildDeadline              = 2 * time.Minute
	scenarioDeadline           = 3 * time.Minute
)

type realExecutor struct {
	mu           sync.Mutex
	prepared     bool
	root         string
	browsertools string
	udon         string
	node         string
	driverEntry  string
	unavailable  bool
	prepareErr   error
}

func NewRealExecutor() ScenarioExecutor { return &realExecutor{} }

func (executor *realExecutor) Close() error {
	executor.mu.Lock()
	root := executor.root
	executor.root = ""
	executor.mu.Unlock()
	if root == "" {
		return nil
	}
	return os.RemoveAll(root)
}

func (executor *realExecutor) Execute(ctx context.Context, manifest Manifest, environment Environment) ScenarioResult {
	if manifest.Suite == SuiteLoopback && strings.TrimSpace(os.Getenv("DISPLAY")) == "" && strings.TrimSpace(os.Getenv("WAYLAND_DISPLAY")) == "" {
		return unavailableScenario(manifest, "fixture_ready")
	}
	executor.prepare(ctx, environment, manifest.Suite)
	if executor.unavailable {
		phase := "fixture_ready"
		if manifest.Suite == SuitePublic {
			phase = "browsertools_probe"
		}
		return unavailableScenario(manifest, phase)
	}
	if executor.prepareErr != nil {
		phase := "fixture_ready"
		if manifest.Suite == SuitePublic {
			phase = "browsertools_probe"
		}
		return failedScenario(manifest, phase, "contract_drift")
	}
	if manifest.Suite == SuitePublic {
		return executor.executePublic(ctx, manifest, environment)
	}
	if manifest.Suite == SuiteJourney {
		return executor.executeJourney(ctx, manifest, environment)
	}
	return executor.executeLoopback(ctx, manifest, environment)
}

func (executor *realExecutor) prepare(ctx context.Context, environment Environment, suite string) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if executor.prepared {
		return
	}
	executor.prepared = true
	root, err := os.MkdirTemp("", "openudon-browser-scenario-")
	if err != nil {
		executor.prepareErr = err
		return
	}
	if err := os.Chmod(root, 0o700); err != nil {
		_ = os.RemoveAll(root)
		executor.prepareErr = err
		return
	}
	executor.root = root
	node, nodeErr := exec.LookPath("node")
	npm, npmErr := exec.LookPath("npm")
	goTool, goErr := exec.LookPath("go")
	if nodeErr != nil || npmErr != nil || goErr != nil {
		executor.unavailable = true
		return
	}
	executor.node, _ = filepath.Abs(node)
	if !commandOutputContains(ctx, environment.BrowserdriverRepo, []string{executor.node, "--version"}, "v"+environment.Lock.NodeVersion+".") ||
		!commandOutputContains(ctx, environment.RepoRoot, []string{goTool, "version"}, "go"+environment.Lock.GoVersion) {
		executor.prepareErr = fmt.Errorf("locked toolchain version is unavailable")
		return
	}
	if !runSilent(ctx, buildDeadline, environment.BrowserdriverRepo, []string{npm, "run", "build", "--silent"}, nil) {
		executor.prepareErr = fmt.Errorf("build Browserdriver")
		return
	}
	executor.driverEntry = filepath.Join(environment.BrowserdriverRepo, "dist", "src", "index.js")
	if !regularFile(executor.driverEntry) {
		executor.prepareErr = fmt.Errorf("Browserdriver entry point is unavailable")
		return
	}
	executor.udon = filepath.Join(root, "udon")
	if !runSilent(ctx, buildDeadline, environment.UdonRepo, []string{goTool, "build", "-o", executor.udon, "./cmd/udon"}, nil) {
		executor.prepareErr = fmt.Errorf("build Udon")
		return
	}
	if suite != SuiteJourney {
		executor.browsertools = filepath.Join(root, "browsertools")
		if !runSilent(ctx, buildDeadline, environment.BrowsertoolsRepo, []string{goTool, "build", "-o", executor.browsertools, "./cmd/browsertools"}, nil) {
			executor.prepareErr = fmt.Errorf("build Browsertools")
			return
		}
		doctor := runBounded(ctx, probeDeadline, environment.BrowsertoolsRepo, []string{executor.browsertools, "playwright", "doctor", "--engine", "chromium", "--format", "json"}, nil, "")
		var readiness struct {
			Version      string `json:"version"`
			Engine       string `json:"engine"`
			DriverReady  bool   `json:"driver_ready"`
			BrowserReady bool   `json:"browser_ready"`
		}
		if json.Unmarshal(doctor.stdout, &readiness) != nil || readiness.Version != "browsertools.playwright-doctor.v1" || readiness.Engine != "chromium" || !readiness.DriverReady || !readiness.BrowserReady || doctor.err != nil {
			executor.unavailable = true
			return
		}
	}
	nodeCheck := `import {createRequire} from "node:module"; import {chromium} from "playwright"; const require=createRequire(import.meta.url); const browser=await chromium.launch({headless:true}); console.log(JSON.stringify({playwright:require("playwright/package.json").version,chromium:browser.version()})); await browser.close();`
	installed := runBounded(ctx, scenarioDeadline, environment.BrowserdriverRepo, []string{executor.node, "--input-type=module", "--eval", nodeCheck}, nil, "")
	var versions struct {
		Playwright string `json:"playwright"`
		Chromium   string `json:"chromium"`
	}
	if installed.err != nil || json.Unmarshal(bytes.TrimSpace(installed.stdout), &versions) != nil || versions.Playwright != environment.Lock.Playwright || versions.Chromium != environment.Lock.Chromium {
		executor.unavailable = true
	}
}

func (executor *realExecutor) executeLoopback(ctx context.Context, manifest Manifest, environment Environment) ScenarioResult {
	result := ScenarioResult{ID: manifest.ID, Attempts: 1}
	fixture, err := NewLoopbackFixture(manifest)
	if err != nil {
		return failedScenario(manifest, "fixture_ready", "fixture_failed")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "fixture_ready", Status: StatusPass, Detail: "ok"})
	caseRoot, err := os.MkdirTemp(executor.root, "case-")
	if err != nil {
		fixture.Close()
		return appendFailure(result, "authoring_v2", "staging_failed")
	}
	exampleDir, privateRoot := filepath.Join(caseRoot, "example"), filepath.Join(caseRoot, "private")
	if os.Mkdir(exampleDir, 0o700) != nil || os.Mkdir(privateRoot, 0o700) != nil {
		fixture.Close()
		_ = os.RemoveAll(caseRoot)
		return appendFailure(result, "authoring_v2", "staging_failed")
	}
	author, authorErr := icot.RunBrowserScenarioAuthor(ctx, icot.BrowserScenarioAuthorRequest{
		BrowsertoolsPath: executor.browsertools, ExampleDir: exampleDir, PrivateRoot: privateRoot,
		InitialURL: fixture.InitialURL(), AuthenticationURL: fixture.AuthenticationURL(), GoalURL: fixture.GoalURL(),
		GoalContext: manifest.Goal.Context, GoalRole: manifest.Goal.Role, GoalLabel: manifest.Goal.Name,
		ChallengeKind: manifest.Authentication.ChallengeKind, ContextMode: manifest.Authentication.ContextMode,
		Outputs: scenarioAuthorOutputs(manifest.Outputs), Fault: manifest.Fault, Now: time.Now().UTC().Round(0),
	})
	if authorErr != nil {
		fixture.Close()
		_ = os.RemoveAll(caseRoot)
		return appendFailure(result, "authoring_v2", "authoring_failed")
	}
	if manifest.Expected.Authoring == "rejected" {
		if !author.Rejected || author.FailureClass != manifest.Expected.FailureCode {
			fixture.Close()
			_ = os.RemoveAll(caseRoot)
			return appendFailure(result, "authoring_v2", "contract_drift")
		}
		result.Phases = append(result.Phases, PhaseResult{ID: "authoring_v2", Status: StatusPass, Detail: "ok"})
		result.Assertions = []string{"author_session_v2", "expected_rejection", "private_material_absent"}
		return finishLoopbackResult(result, fixture, caseRoot)
	}
	if author.Rejected || !validAuthorResult(manifest, author) {
		fixture.Close()
		_ = os.RemoveAll(caseRoot)
		return appendFailure(result, "authoring_v2", "contract_drift")
	}
	result.Phases = append(result.Phases,
		PhaseResult{ID: "authoring_v2", Status: StatusPass, Detail: "ok"},
		PhaseResult{ID: "profiles_staged", Status: StatusPass, Detail: "ok"},
		PhaseResult{ID: "profile_versions", Status: StatusPass, Detail: "ok"},
	)
	bindings := make(map[string]string, len(author.CredentialSlotKinds))
	for slot := range author.CredentialSlotKinds {
		bindings[slot] = "scenario_" + slot
	}
	workflow, workflowErr := synthesize.WriteBrowserScenarioWorkflow(synthesize.BrowserScenarioWorkflowRequest{
		ExampleDir: exampleDir, AuthenticationPath: author.AuthenticationPath, CapabilityPath: author.CapabilityPath,
		AuthenticationFlow: "authenticated_goal", CapabilityAction: "reach_authenticated_goal", Session: "scenario_session",
		CredentialSlotBindings: bindings,
	})
	if workflowErr != nil || workflow.UWSVersion != manifest.Expected.UWSVersion {
		fixture.Close()
		_ = os.RemoveAll(caseRoot)
		return appendFailure(result, "profiles_staged", "profile_mismatch")
	}
	if manifest.Fault == "context_substitution" {
		if mutateScenarioContext(author.CapabilityPath) != nil {
			fixture.Close()
			_ = os.RemoveAll(caseRoot)
			return appendFailure(result, "profiles_staged", "staging_failed")
		}
	}
	variants := append([]string(nil), manifest.ReplayVariants...)
	if len(variants) == 0 {
		variants = []string{""}
	}
	for _, variant := range variants {
		// Each replay variant gets independent server-side session and proof
		// counters, matching its fresh Browserdriver lifecycle.
		fixture.SetRuntime(true)
		if fixture.SetReplayVariant(variant) != nil {
			fixture.Close()
			_ = os.RemoveAll(caseRoot)
			return appendFailure(result, "udon_v3", "contract_drift")
		}
		replay := executor.runUdon(ctx, manifest, exampleDir, workflow.Path, author.CredentialSlotKinds, bindings, fixture)
		if manifest.Expected.Replay == "pass" {
			if replay.failureCode != "" {
				fixture.Close()
				_ = os.RemoveAll(caseRoot)
				return appendFailure(result, "browserdriver_replay", replay.failureCode)
			}
			if !scenarioOutputsEqual(replay.outputs, fixture.ExpectedOutputs(manifest.Outputs)) {
				fixture.Close()
				_ = os.RemoveAll(caseRoot)
				return appendFailure(result, "browserdriver_replay", "output_mismatch")
			}
			if !fixture.AuthenticatedReplayObserved() {
				fixture.Close()
				_ = os.RemoveAll(caseRoot)
				return appendFailure(result, "browserdriver_replay", "authentication_not_proven")
			}
		} else if replay.failureCode != manifest.Expected.FailureCode {
			fixture.Close()
			_ = os.RemoveAll(caseRoot)
			detail := replay.failureCode
			if detail == "" {
				detail = udonreport.CodeUnclassified
			}
			return appendFailure(result, "browserdriver_replay", detail)
		}
	}
	result.Phases = append(result.Phases,
		PhaseResult{ID: "udon_v3", Status: StatusPass, Detail: "ok"},
		PhaseResult{ID: "browserdriver_replay", Status: StatusPass, Detail: "ok"},
		PhaseResult{ID: "outputs_validated", Status: StatusPass, Detail: "ok"},
	)
	result.Assertions = []string{"author_session_v2", "profiles_reconstructed", "oldest_sufficient_versions", "udon_v3_lowering", "browserdriver_replay", "private_material_absent"}
	if manifest.Authentication.ChallengeKind != "" {
		result.Assertions = append(result.Assertions, "reviewed_mfa_kind")
	}
	if len(manifest.Outputs) != 0 {
		result.Assertions = append(result.Assertions, "reviewed_outputs")
	}
	if manifest.Expected.Replay == "pass" {
		result.Assertions = append(result.Assertions, "typed_outputs_exact")
	} else {
		result.Assertions = append(result.Assertions, "expected_rejection")
	}
	return finishLoopbackResult(result, fixture, caseRoot)
}

func (executor *realExecutor) executeJourney(ctx context.Context, manifest Manifest, environment Environment) ScenarioResult {
	result := ScenarioResult{ID: manifest.ID, Attempts: 1}
	fixture, err := NewJourneyFixture(manifest)
	if err != nil {
		return failedScenario(manifest, "fixture_ready", "fixture_failed")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "fixture_ready", Status: StatusPass, Detail: "ok"})
	caseRoot, err := os.MkdirTemp(executor.root, "journey-")
	if err != nil {
		fixture.Close()
		return appendFailure(result, "bundle_authored", "staging_failed")
	}
	fail := func(phase, detail string) ScenarioResult {
		return finishJourneyResult(appendFailure(result, phase, detail), fixture, caseRoot)
	}
	exampleDir := filepath.Join(caseRoot, "example")
	privateDir := filepath.Join(caseRoot, "private")
	if os.Mkdir(exampleDir, 0o700) != nil || os.Mkdir(privateDir, 0o700) != nil {
		return fail("bundle_authored", "staging_failed")
	}

	blueprint, err := journeyScenarioBlueprint(manifest, fixture.Origin())
	if err != nil {
		return fail("bundle_authored", "contract_drift")
	}
	bundle, err := buildJourneyBundle(blueprint.actions, fixture.Origin(), environment.Now)
	if err != nil {
		return fail("bundle_authored", "authoring_failed")
	}
	bundlePath := filepath.Join(privateDir, "guided-authoring.json")
	authenticationSourcePath := filepath.Join(privateDir, "authentication.json")
	if os.WriteFile(bundlePath, bundle, 0o600) != nil || writeJourneyAuthentication(authenticationSourcePath, fixture.Origin(), environment.Now) != nil {
		return fail("bundle_authored", "staging_failed")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "bundle_authored", Status: StatusPass, Detail: "ok"})

	capabilityPath, authenticationPath, err := stageJourneySources(ctx, exampleDir, bundlePath, authenticationSourcePath, environment.Now)
	if err != nil {
		return fail("profile_imported", "profile_mismatch")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "profile_imported", Status: StatusPass, Detail: "ok"})

	workflowRequest := synthesize.BrowserScenarioWorkflowRequest{
		ExampleDir: exampleDir, AuthenticationPath: authenticationPath, CapabilityPath: capabilityPath,
		AuthenticationFlow: journeyAuthenticationFlow, Session: journeySession,
		CredentialSlotBindings: map[string]string{}, Inputs: blueprint.inputs, Actions: blueprint.workflow,
	}
	workflow, err := synthesize.WriteBrowserScenarioWorkflow(workflowRequest)
	if err != nil || workflow.UWSVersion != manifest.Expected.UWSVersion {
		return fail("uws_synthesized", "profile_mismatch")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "uws_synthesized", Status: StatusPass, Detail: "ok"})

	dataPath := filepath.Join(exampleDir, "journey-data.hcl")
	if manifest.Journey.Kind == "parameter_contract_rejected" {
		if detail := executor.executeJourneyParameterCases(ctx, manifest, exampleDir, dataPath, workflow.Path, workflowRequest, blueprint, fixture); detail != "" {
			return fail("browserdriver_replay", detail)
		}
	} else if manifest.Journey.Kind == "session_lifecycle" {
		first := executor.runJourneyUdon(ctx, exampleDir, dataPath, workflow.Path, blueprint.values, blueprint.approvedOperations, blueprint.workflow[len(blueprint.workflow)-1].Name)
		second := executor.runJourneyUdon(ctx, exampleDir, dataPath, workflow.Path, blueprint.values, blueprint.approvedOperations, blueprint.workflow[len(blueprint.workflow)-1].Name)
		if first.failureCode != "" {
			return fail("browserdriver_replay", first.failureCode)
		}
		if second.failureCode != "" {
			return fail("browserdriver_replay", second.failureCode)
		}
		if first.outputs["run_marker"] != "Run 1" || second.outputs["run_marker"] != "Run 2" || fixture.SessionCount() != 2 {
			return fail("browserdriver_replay", "output_mismatch")
		}
	} else {
		lastStep := blueprint.workflow[len(blueprint.workflow)-1].Name
		replay := executor.runJourneyUdon(ctx, exampleDir, dataPath, workflow.Path, blueprint.values, blueprint.approvedOperations, lastStep)
		if manifest.Expected.Replay == "pass" {
			if replay.failureCode != "" {
				return fail("browserdriver_replay", replay.failureCode)
			}
			if !scenarioOutputsEqual(replay.outputs, blueprint.expectedOutputs) {
				return fail("browserdriver_replay", "output_mismatch")
			}
		} else if replay.failureCode != manifest.Expected.FailureCode {
			return fail("browserdriver_replay", "replay_failed")
		}
	}
	result.Phases = append(result.Phases,
		PhaseResult{ID: "udon_v3", Status: StatusPass, Detail: "ok"},
		PhaseResult{ID: "browserdriver_replay", Status: StatusPass, Detail: "ok"},
	)

	if !validJourneyPostconditions(manifest, fixture) {
		return fail("postconditions", "contract_drift")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "postconditions", Status: StatusPass, Detail: "ok"})
	result.Status, result.Detail = StatusPass, "ok"
	result.Assertions = journeyAssertions(manifest)
	return finishJourneyResult(result, fixture, caseRoot)
}

func (executor *realExecutor) executeJourneyParameterCases(ctx context.Context, manifest Manifest, exampleDir, dataPath, workflowPath string, request synthesize.BrowserScenarioWorkflowRequest, blueprint journeyBlueprint, fixture *JourneyFixture) string {
	variants := map[string]map[string]any{
		"missing_required": {},
		"wrong_type":       {"target_url": 42},
		"origin_escape":    {"target_url": "http://127.0.0.1:9/escape"},
	}
	for _, variant := range []string{"missing_required", "wrong_type", "origin_escape"} {
		if !journeyContainsString(manifest.ReplayVariants, variant) {
			continue
		}
		replay := executor.runJourneyUdon(ctx, exampleDir, dataPath, workflowPath, variants[variant], nil, blueprint.workflow[len(blueprint.workflow)-1].Name)
		if variant == "origin_escape" {
			if replay.failureCode != "origin_rejected" {
				return journeyReplayFailureDetail(replay.failureCode)
			}
		} else if replay.failureCode != "invalid_parameters" {
			return journeyReplayFailureDetail(replay.failureCode)
		}
	}
	if journeyContainsString(manifest.ReplayVariants, "additional_parameter") {
		request.Inputs = append(request.Inputs, synthesize.BrowserScenarioInput{Name: "unexpected", Type: "string", Required: true})
		request.Actions = append([]synthesize.BrowserScenarioAction(nil), request.Actions...)
		request.Actions[0].With = cloneStringMap(request.Actions[0].With)
		request.Actions[0].With["unexpected"] = "unexpected"
		if _, err := synthesize.WriteBrowserScenarioWorkflow(request); err == nil || !strings.Contains(err.Error(), `parameter "unexpected" is not declared`) {
			return "contract_drift"
		}
	}
	if fixture.MutationPOSTs() != 0 {
		return "contract_drift"
	}
	return ""
}

func journeyReplayFailureDetail(failureCode string) string {
	if allowedDetails[failureCode] {
		return failureCode
	}
	return "replay_failed"
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values)+1)
	for key, value := range values {
		result[key] = value
	}
	return result
}

func journeyContainsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func validJourneyPostconditions(manifest Manifest, fixture *JourneyFixture) bool {
	switch manifest.Journey.Kind {
	case "record_update_approved":
		note, priority, enabled, archived := fixture.RecordState()
		return fixture.MutationPOSTs() == 1 && note == "Reviewed note" && priority == "high" && enabled && !archived
	case "record_update_unapproved", "record_update_ambiguous", "parameter_contract_rejected":
		return fixture.MutationPOSTs() == 0
	case "session_lifecycle":
		return fixture.SessionCount() == 2
	default:
		return fixture.Requests() > 0
	}
}

func journeyAssertions(manifest Manifest) []string {
	assertions := []string{"guided_authoring_v1", "canonical_profile_only", "closed_macro_vocabulary", "udon_v3_lowering", "browserdriver_replay", "private_material_absent"}
	switch manifest.Journey.Kind {
	case "catalog_search_filter", "catalog_pagination", "order_structured_read":
		assertions = append(assertions, "structured_outputs_exact")
	case "record_update_approved":
		assertions = append(assertions, "structured_outputs_exact", "operation_approval_exact", "server_state_exact")
	case "record_update_unapproved", "record_update_ambiguous":
		assertions = append(assertions, "expected_rejection", "operation_approval_exact", "server_state_exact")
	case "parameter_contract_rejected":
		assertions = append(assertions, "expected_rejection", "parameter_contract_exact", "server_state_exact")
	case "session_lifecycle":
		assertions = append(assertions, "structured_outputs_exact", "session_isolated")
	}
	return canonicalAssertions(assertions)
}

func finishJourneyResult(result ScenarioResult, fixture *JourneyFixture, caseRoot string) ScenarioResult {
	fixture.Close()
	if err := os.RemoveAll(caseRoot); err != nil {
		return appendFailure(result, "teardown", "teardown_failed")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "teardown", Status: StatusPass, Detail: "ok"})
	result.Assertions = canonicalAssertions(result.Assertions)
	return result
}

func (executor *realExecutor) executePublic(ctx context.Context, manifest Manifest, environment Environment) ScenarioResult {
	result := ScenarioResult{ID: manifest.ID, Attempts: 1}
	caseRoot, err := os.MkdirTemp(executor.root, "public-")
	if err != nil {
		return failedScenario(manifest, "browsertools_probe", "staging_failed")
	}
	if err := os.Chmod(caseRoot, 0o700); err != nil {
		_ = os.RemoveAll(caseRoot)
		return failedScenario(manifest, "browsertools_probe", "staging_failed")
	}
	fail := func(phase, detail string) ScenarioResult {
		result = appendFailure(result, phase, detail)
		return finishPublicResult(result, caseRoot)
	}

	prof, profileData, err := publicScenarioProfile(manifest, environment.Now)
	if err != nil {
		return fail("browsertools_probe", "contract_drift")
	}
	profileDir := filepath.Join(caseRoot, "browser-profiles")
	authenticationDir := filepath.Join(caseRoot, "browser-authentication")
	if os.MkdirAll(profileDir, 0o700) != nil || os.MkdirAll(authenticationDir, 0o700) != nil {
		return fail("browsertools_probe", "staging_failed")
	}
	profilePath := filepath.Join(profileDir, "canary.json")
	authenticationPath := filepath.Join(authenticationDir, "canary.json")
	authenticationData, err := publicScenarioAuthenticationProfile(manifest, environment.Now)
	if err != nil || os.WriteFile(profilePath, profileData, 0o600) != nil || os.WriteFile(authenticationPath, authenticationData, 0o600) != nil {
		return fail("browsertools_probe", "staging_failed")
	}
	livePath := filepath.Join(caseRoot, "browsertools-live-check.json")
	args := []string{
		executor.browsertools, "live-check", "chromium", "--profile", profilePath,
		"--url", manifest.Target.URL, "--at", environment.Now.UTC().Format(time.RFC3339Nano),
		"--action", publicScenarioAction, "--timeout", "45s", "--navigation-timeout", "30s", "--out", livePath,
	}
	for _, origin := range manifest.Target.Origins {
		args = append(args, "--allow-origin", origin)
	}
	command := runBounded(ctx, scenarioDeadline, environment.BrowsertoolsRepo, args, nil, "")
	summary, inspectErr := browserverify.Inspect(livePath, prof, environment.Now)
	if inspectErr != nil {
		if command.err != nil {
			return fail("browsertools_probe", publicLiveFailure(ctx))
		}
		return fail("browsertools_probe", "contract_drift")
	}
	if !validPublicLiveSummary(manifest, summary) {
		return fail("browsertools_probe", "shape_drift")
	}
	if command.err != nil {
		return fail("browsertools_probe", publicLiveFailure(ctx))
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "browsertools_probe", Status: StatusPass, Detail: "ok"})

	workflow, err := synthesize.WriteBrowserScenarioWorkflow(synthesize.BrowserScenarioWorkflowRequest{
		ExampleDir: caseRoot, AuthenticationPath: authenticationPath, CapabilityPath: profilePath,
		AuthenticationFlow: publicScenarioFlow, CapabilityAction: publicScenarioAction, Session: publicScenarioSession,
		CredentialSlotBindings: map[string]string{},
	})
	if err != nil || workflow.UWSVersion != manifest.Expected.UWSVersion {
		return fail("udon_browserdriver_probe", "staging_failed")
	}
	udon := runBounded(ctx, scenarioDeadline, caseRoot, []string{
		executor.udon, "--workdir", caseRoot, "--workflow", workflow.Path, "--workflow-format", "uws-json",
		"--execution-report", "execution-report.json", "--execution-timeout", "60s",
		"--browser-driver", executor.node, "--browser-driver-arg", executor.driverEntry,
		"--browser-driver-protocol", "v2", "--approve-browser-authentication", "authenticate", "--quiet",
	}, nil, "")
	if udon.err != nil {
		return fail("udon_browserdriver_probe", publicUdonFailure(ctx, filepath.Join(caseRoot, "execution-report.json")))
	}
	outputs, err := readScenarioOutputs(filepath.Join(caseRoot, "output", "udon.hcl"))
	if err != nil || !scenarioOutputsEqual(outputs, expectedPublicOutputs(manifest)) {
		return fail("udon_browserdriver_probe", "shape_drift")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "udon_browserdriver_probe", Status: StatusPass, Detail: "ok"})
	result.Status, result.Detail = StatusPass, "ok"
	result.Assertions = []string{"browsertools_presence", "runtime_presence", "private_material_absent"}
	return finishPublicResult(result, caseRoot)
}

func publicScenarioAuthenticationProfile(manifest Manifest, now time.Time) ([]byte, error) {
	if manifest.Target == nil || len(manifest.Probes) == 0 {
		return nil, fmt.Errorf("public browser scenario is incomplete")
	}
	probe := manifest.Probes[0]
	locator := map[string]any{"role": probe.Role}
	if probe.Name != "" {
		locator["name"] = probe.Name
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	document := map[string]any{
		"profile": "uws.browser-authentication.1.0",
		"info": map[string]any{
			"title":              "OpenUdon credential-free public browser canary " + manifest.ID,
			"applicationOrigins": manifest.Target.Origins, "authenticationOrigins": manifest.Target.Origins,
		},
		"observationKind": "accessibility_snapshot",
		"evidence":        map[string]any{"learnedAt": stamp, "source": "reviewed_public_canary_manifest"},
		"confidence":      "high", "expiresAfter": "P14D",
		"verification":    map[string]any{"lastVerifiedAt": stamp, "successfulRuns": 1, "uiStabilityScore": 1.0},
		"credentialSlots": map[string]any{},
		"flows": map[string]any{publicScenarioFlow: map[string]any{
			"description": "Open the fixed public target in an ephemeral credential-free session.",
			"sequence":    []any{map[string]any{"navigate": manifest.Target.URL}, map[string]any{"wait_for": map[string]any{"locator": locator}}},
			"effects":     []string{"establishes_session"},
			"success":     map[string]any{"origin": publicTargetOrigin(manifest), "locator": locator},
		}},
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')
	if _, err := authprofile.Parse(data); err != nil {
		return nil, err
	}
	return data, nil
}

func finishPublicResult(result ScenarioResult, caseRoot string) ScenarioResult {
	if err := os.RemoveAll(caseRoot); err != nil {
		return appendFailure(result, "teardown", "teardown_failed")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "teardown", Status: StatusPass, Detail: "ok"})
	result.Assertions = canonicalAssertions(result.Assertions)
	return result
}

func publicScenarioProfile(manifest Manifest, now time.Time) (*profile.Profile, []byte, error) {
	if manifest.Target == nil || len(manifest.Probes) == 0 {
		return nil, nil, fmt.Errorf("public browser scenario is incomplete")
	}
	sequence := []any{map[string]any{"navigate": manifest.Target.URL}}
	outputs := make(map[string]any, len(manifest.Probes))
	for index, probe := range manifest.Probes {
		locator := map[string]any{"role": probe.Role}
		if probe.Name != "" {
			locator["name"] = probe.Name
		}
		sequence = append(sequence, map[string]any{"wait_for": locator})
		outputs[publicProbeOutputKey(index)] = map[string]any{
			"type": "boolean", "source": "a11y", "locator": locator, "presence": true,
		}
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	document := map[string]any{
		"profile":         "uws.browser.1.5",
		"info":            map[string]any{"title": "OpenUdon public browser canary " + manifest.ID, "origin": manifest.Target.Origins},
		"observationKind": "accessibility_snapshot",
		"evidence":        map[string]any{"learnedAt": stamp, "source": "reviewed_public_canary_manifest"},
		"confidence":      "high", "expiresAfter": "P14D",
		"verification": map[string]any{"lastVerifiedAt": stamp, "successfulRuns": 1, "uiStabilityScore": 1.0},
		"actions": map[string]any{publicScenarioAction: map[string]any{
			"description": "Observe only the reviewed public accessibility landmarks.",
			"parameters":  map[string]any{"type": "object", "properties": map[string]any{}},
			"sequence":    sequence, "outputs": outputs, "sideEffects": []string{"read_only"},
			"confirmationPolicy": map[string]any{"required": false},
		}},
	}
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, nil, err
	}
	data = append(data, '\n')
	prof, err := profile.ParseJSON(data)
	if err != nil {
		return nil, nil, err
	}
	return prof, data, nil
}

func validPublicLiveSummary(manifest Manifest, summary browserverify.Summary) bool {
	if manifest.Target == nil || summary.ReportVersion != browserverify.LiveCheckVersion || summary.Engine != "chromium" || !summary.OK ||
		len(summary.Actions) != 1 || summary.Actions[0] != publicScenarioAction || len(summary.Checks) != 2*len(manifest.Probes) {
		return false
	}
	wantOrigin := publicTargetOrigin(manifest)
	if summary.Origin != wantOrigin {
		return false
	}
	want := make(map[string]browserverify.Check, 2*len(manifest.Probes))
	for index, probe := range manifest.Probes {
		want[fmt.Sprintf("actions.%s.sequence[%d].wait_for", publicScenarioAction, index+1)] = browserverify.Check{
			Kind: "locator", OK: true, Matches: probe.MinMatches,
		}
		want[fmt.Sprintf("actions.%s.outputs.%s", publicScenarioAction, publicProbeOutputKey(index))] = browserverify.Check{
			Kind: "output", OK: true, ExpectedType: profile.OutputBoolean, ObservedType: profile.OutputBoolean,
		}
	}
	for _, check := range summary.Checks {
		expected, ok := want[check.Path]
		if !ok || check.Kind != expected.Kind || check.OK != expected.OK || check.ExpectedType != expected.ExpectedType || check.ObservedType != expected.ObservedType {
			return false
		}
		if check.Kind == "locator" && check.Matches != expected.Matches {
			return false
		}
		delete(want, check.Path)
	}
	return len(want) == 0
}

func expectedPublicOutputs(manifest Manifest) map[string]any {
	result := make(map[string]any, len(manifest.Probes))
	for index := range manifest.Probes {
		result[publicProbeOutputKey(index)] = true
	}
	return result
}

func publicProbeOutputKey(index int) string { return fmt.Sprintf("probe_%02d_present", index+1) }

func publicTargetOrigin(manifest Manifest) string {
	if manifest.Target == nil {
		return ""
	}
	for _, origin := range manifest.Target.Origins {
		if strings.HasPrefix(manifest.Target.URL, origin+"/") || manifest.Target.URL == origin {
			return origin
		}
	}
	return ""
}

func publicLiveFailure(ctx context.Context) string {
	if ctx.Err() != nil {
		return "timeout"
	}
	return udonreport.CodeUnclassified
}

func publicUdonFailure(ctx context.Context, reportPath string) string {
	if ctx.Err() != nil {
		return "timeout"
	}
	code := executionFailureCode(reportPath)
	if code == "origin_rejected" {
		return "origin_policy_drift"
	}
	return code
}

func finishLoopbackResult(result ScenarioResult, fixture *LoopbackFixture, caseRoot string) ScenarioResult {
	fixture.Close()
	if err := os.RemoveAll(caseRoot); err != nil {
		return appendFailure(result, "teardown", "teardown_failed")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "teardown", Status: StatusPass, Detail: "ok"})
	result.Status, result.Detail = StatusPass, "ok"
	result.Assertions = canonicalAssertions(result.Assertions)
	return result
}

type replayResult struct {
	outputs     map[string]any
	failureCode string
}

func (executor *realExecutor) runUdon(ctx context.Context, manifest Manifest, exampleDir, workflowPath string, slotKinds, bindings map[string]string, fixture *LoopbackFixture) replayResult {
	args := []string{
		executor.udon, "--workdir", exampleDir, "--workflow", workflowPath, "--workflow-format", "uws-json",
		"--execution-report", "execution-report.json", "--execution-timeout", "60s",
		"--browser-driver", executor.node, "--browser-driver-arg", executor.driverEntry,
		"--browser-driver-protocol", "v3", "--browser-challenge-timeout", "10s",
		"--approve-browser-authentication", "authenticate", "--quiet",
	}
	environment := map[string]string{}
	slots := make([]string, 0, len(slotKinds))
	for slot := range slotKinds {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	for _, slot := range slots {
		environmentName := "OPENUDON_SCENARIO_" + strings.ToUpper(strings.ReplaceAll(slot, "-", "_"))
		args = append(args, "--browser-credential-env", bindings[slot]+"="+environmentName)
		switch slotKinds[slot] {
		case "identifier":
			environment[environmentName] = "member@example.test"
		case "password":
			environment[environmentName] = "scenario-password-value"
		case "totp_seed":
			environment[environmentName] = "JBSWY3DPEHPK3PXP"
		default:
			return replayResult{failureCode: "invalid_response"}
		}
	}
	input := ""
	if allowedOTPChallenge(manifest.Authentication.ChallengeKind) && manifest.Authentication.ChallengeKind != "totp" {
		input = "123456\n"
	}
	var command boundedCommand
	if approvalPrompt := exactBrowserApprovalPrompt(manifest.Authentication.ChallengeKind); approvalPrompt != "" {
		command = runBoundedAfterPrompt(ctx, scenarioDeadline, exampleDir, args, environment, approvalPrompt, func() error {
			return fixture.ApprovePendingChallenge(manifest.Authentication.ChallengeKind)
		})
	} else {
		command = runBounded(ctx, scenarioDeadline, exampleDir, args, environment, input)
	}
	if command.err != nil {
		return replayResult{failureCode: scenarioFailureCode(filepath.Join(exampleDir, "execution-report.json"))}
	}
	outputs, err := readScenarioOutputs(filepath.Join(exampleDir, "output", "udon.hcl"))
	if err != nil {
		return replayResult{failureCode: "invalid_response"}
	}
	return replayResult{outputs: outputs}
}

func exactBrowserApprovalPrompt(kind string) string {
	switch kind {
	case "push_number_match":
		return "Browser authentication push_number_match number 42. Approve? [y/N]: "
	case "push", "passkey", "security_key":
		return "Browser authentication " + kind + " challenge. Approve? [y/N]: "
	default:
		return ""
	}
}

func (executor *realExecutor) runJourneyUdon(ctx context.Context, exampleDir, dataPath, workflowPath string, values map[string]any, approvals []string, resultStep string) replayResult {
	if err := os.RemoveAll(filepath.Join(exampleDir, "output")); err != nil {
		return replayResult{failureCode: "invalid_response"}
	}
	if err := os.Remove(filepath.Join(exampleDir, "execution-report.json")); err != nil && !os.IsNotExist(err) {
		return replayResult{failureCode: "invalid_response"}
	}
	if err := writeJourneyData(dataPath, values); err != nil {
		return replayResult{failureCode: "invalid_response"}
	}
	args := []string{
		executor.udon, "--workdir", exampleDir, "--workflow", workflowPath, "--workflow-format", "uws-json",
		"--datafile", dataPath, "--execution-report", "execution-report.json", "--execution-timeout", "60s",
		"--browser-driver", executor.node, "--browser-driver-arg", executor.driverEntry,
		"--browser-driver-protocol", "v3", "--browser-challenge-timeout", "10s",
		"--approve-browser-authentication", "authenticate", "--quiet",
	}
	for _, operation := range approvals {
		args = append(args, "--approve-browser-operation", operation)
	}
	command := runBounded(ctx, scenarioDeadline, exampleDir, args, nil, "")
	if command.err != nil {
		return replayResult{failureCode: journeyFailureCode(filepath.Join(exampleDir, "execution-report.json"))}
	}
	outputs, err := readBrowserOutputs(filepath.Join(exampleDir, "output", "udon.hcl"), resultStep)
	if err != nil {
		return replayResult{failureCode: "invalid_response"}
	}
	return replayResult{outputs: outputs}
}

func validAuthorResult(manifest Manifest, result icot.BrowserScenarioAuthorResult) bool {
	if result.AuthenticationProfile != "uws.browser-authentication.1.1" || result.CapabilityProfile != manifest.Expected.BrowserProfile ||
		result.ReviewedChallengeKind != manifest.Authentication.ChallengeKind || !result.PrivateEnvelopePreserved || result.EnvelopeDigest == "" {
		return false
	}
	wantKeys := make([]string, 0, len(manifest.Outputs))
	for _, output := range manifest.Outputs {
		wantKeys = append(wantKeys, output.Key)
	}
	sort.Strings(wantKeys)
	if !equalStrings(wantKeys, result.ReviewedOutputKeys) || result.CredentialSlotKinds["credential_1"] != "identifier" || result.CredentialSlotKinds["credential_2"] != "password" {
		return false
	}
	_, hasTOTP := result.CredentialSlotKinds["totp_seed"]
	return hasTOTP == (manifest.Authentication.ChallengeKind == "totp") && len(result.CredentialSlotKinds) == 2+btoi(hasTOTP)
}

func scenarioAuthorOutputs(outputs []Output) []icot.BrowserScenarioOutput {
	result := make([]icot.BrowserScenarioOutput, len(outputs))
	for index, output := range outputs {
		result[index] = icot.BrowserScenarioOutput{Key: output.Key, Type: output.Type, Role: output.Role, Name: output.Name, LocatorMode: output.LocatorMode}
	}
	return result
}

func scenarioOutputsEqual(actual, expected map[string]any) bool {
	left, leftErr := json.Marshal(actual)
	right, rightErr := json.Marshal(expected)
	return leftErr == nil && rightErr == nil && bytes.Equal(left, right)
}

func mutateScenarioContext(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	actions, _ := document["actions"].(map[string]any)
	action, _ := actions["reach_authenticated_goal"].(map[string]any)
	outputs, _ := action["outputs"].(map[string]any)
	for key, raw := range outputs {
		if key == "goal_present" {
			continue
		}
		output, _ := raw.(map[string]any)
		output["context"] = "popup_2"
		break
	}
	data, err = json.Marshal(document)
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func readScenarioOutputs(path string) (map[string]any, error) {
	return readBrowserOutputs(path, "read")
}

func readBrowserOutputs(path, resultStep string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > scenarioCommandOutputLimit {
		return nil, fmt.Errorf("scenario runtime output is unavailable")
	}
	file, diagnostics := hclsyntax.ParseConfig(data, path, hcl.Pos{Line: 1, Column: 1})
	if diagnostics.HasErrors() {
		return nil, fmt.Errorf("parse scenario runtime output")
	}
	for _, block := range file.Body.(*hclsyntax.Body).Blocks {
		if block.Type != "result" {
			continue
		}
		if literalAttribute(block.Body, "from") != resultStep || literalAttribute(block.Body, "kind") != "browser" {
			continue
		}
		for _, nested := range block.Body.Blocks {
			if nested.Type != "received_body" {
				continue
			}
			result := map[string]any{}
			for key, attribute := range nested.Body.Attributes {
				value, valueDiagnostics := attribute.Expr.Value(nil)
				if valueDiagnostics.HasErrors() || !value.IsWhollyKnown() {
					return nil, fmt.Errorf("scenario runtime output is not literal")
				}
				encoded, marshalErr := ctyjson.Marshal(value, value.Type())
				var decoded any
				if marshalErr != nil || json.Unmarshal(encoded, &decoded) != nil {
					return nil, fmt.Errorf("scenario runtime output conversion failed")
				}
				result[key] = decoded
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("scenario runtime result is missing")
}

func literalAttribute(body *hclsyntax.Body, name string) string {
	attribute := body.Attributes[name]
	if attribute == nil {
		return ""
	}
	value, diagnostics := attribute.Expr.Value(nil)
	if !diagnostics.HasErrors() && value.IsKnown() && !value.IsNull() && value.Type().FriendlyName() == "string" {
		return value.AsString()
	}
	return ""
}

type boundedCommand struct {
	stdout []byte
	stderr []byte
	err    error
}

func runBounded(ctx context.Context, timeout time.Duration, directory string, args []string, environment map[string]string, input string) boundedCommand {
	if len(args) == 0 {
		return boundedCommand{err: fmt.Errorf("empty scenario command")}
	}
	var stdout, stderr limitedWriter
	stdout.limit, stderr.limit = scenarioCommandOutputLimit, scenarioCommandOutputLimit
	err := processgroup.Run(ctx, timeout, processgroup.Invocation{
		Args: args, Dir: directory, Env: scenarioEnvironment(environment), Stdin: strings.NewReader(input),
		Stdout: &stdout, Stderr: &stderr,
	})
	if stdout.overflow || stderr.overflow {
		err = fmt.Errorf("scenario subprocess output exceeded its bound")
	}
	return boundedCommand{stdout: stdout.Bytes(), stderr: stderr.Bytes(), err: err}
}

func runBoundedAfterPrompt(ctx context.Context, timeout time.Duration, directory string, args []string, environment map[string]string, prompt string, approve func() error) boundedCommand {
	if len(args) == 0 || prompt == "" || approve == nil {
		return boundedCommand{err: fmt.Errorf("invalid interactive scenario command")}
	}
	var stdout, stderr limitedWriter
	stdout.limit, stderr.limit = scenarioCommandOutputLimit, scenarioCommandOutputLimit
	matched := make(chan struct{})
	observer := &exactPromptWriter{destination: &stdout, prompt: []byte(prompt), matched: matched}
	inputReader, inputWriter := io.Pipe()
	interactionContext, cancelInteraction := context.WithCancel(ctx)
	interactionDone := make(chan error, 1)
	go func() {
		select {
		case <-matched:
			if err := approve(); err != nil {
				interactionDone <- err
				_ = inputWriter.CloseWithError(err)
				return
			}
			_, err := io.WriteString(inputWriter, "y\n")
			interactionDone <- err
		case <-interactionContext.Done():
			interactionDone <- nil
		}
	}()
	err := processgroup.Run(ctx, timeout, processgroup.Invocation{
		Args: args, Dir: directory, Env: scenarioEnvironment(environment), Stdin: inputReader,
		Stdout: observer, Stderr: &stderr,
	})
	cancelInteraction()
	_ = inputWriter.Close()
	_ = inputReader.Close()
	if interactionErr := <-interactionDone; interactionErr != nil {
		err = fmt.Errorf("approve observed browser challenge: %w", interactionErr)
	}
	if stdout.overflow || stderr.overflow {
		err = fmt.Errorf("scenario subprocess output exceeded its bound")
	}
	return boundedCommand{stdout: stdout.Bytes(), stderr: stderr.Bytes(), err: err}
}

type exactPromptWriter struct {
	destination io.Writer
	prompt      []byte
	window      []byte
	matched     chan struct{}
	once        sync.Once
}

func (writer *exactPromptWriter) Write(value []byte) (int, error) {
	written, err := writer.destination.Write(value)
	writer.window = append(writer.window, value...)
	if bytes.Contains(writer.window, writer.prompt) {
		writer.once.Do(func() { close(writer.matched) })
	}
	if keep := len(writer.prompt) - 1; keep > 0 && len(writer.window) > keep {
		writer.window = append(writer.window[:0], writer.window[len(writer.window)-keep:]...)
	}
	return written, err
}

func runSilent(ctx context.Context, timeout time.Duration, directory string, args []string, environment map[string]string) bool {
	result := runBounded(ctx, timeout, directory, args, environment, "")
	return result.err == nil
}

func commandOutputContains(ctx context.Context, directory string, args []string, wanted string) bool {
	result := runBounded(ctx, probeDeadline, directory, args, nil, "")
	return result.err == nil && strings.Contains(string(result.stdout), wanted)
}

type limitedWriter struct {
	bytes.Buffer
	limit    int
	overflow bool
}

func (writer *limitedWriter) Write(value []byte) (int, error) {
	written := len(value)
	remaining := writer.limit - writer.Len()
	if remaining <= 0 {
		writer.overflow = true
		return written, nil
	}
	if len(value) > remaining {
		value = value[:remaining]
		writer.overflow = true
	}
	_, _ = writer.Buffer.Write(value)
	return written, nil
}

func scenarioEnvironment(overrides map[string]string) []string {
	allowed := map[string]bool{"DISPLAY": true, "WAYLAND_DISPLAY": true, "XAUTHORITY": true, "XDG_RUNTIME_DIR": true, "DBUS_SESSION_BUS_ADDRESS": true, "HOME": true, "PATH": true, "LANG": true, "LC_ALL": true, "PLAYWRIGHT_BROWSERS_PATH": true, "TMPDIR": true}
	values := map[string]string{}
	for _, item := range os.Environ() {
		name, value, ok := strings.Cut(item, "=")
		if ok && allowed[name] && !strings.ContainsAny(value, "\x00\r\n") {
			values[name] = value
		}
	}
	for name, value := range overrides {
		if scenarioEnvironmentName.MatchString(name) && !strings.ContainsAny(value, "\x00\r\n") {
			values[name] = value
		}
	}
	names := make([]string, 0, len(values))
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]string, 0, len(names))
	for _, name := range names {
		result = append(result, name+"="+values[name])
	}
	return result
}

func regularFile(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode()&os.ModeSymlink == 0
}

func scenarioFailureCode(reportPath string) string {
	return executionFailureCode(reportPath)
}

func journeyFailureCode(reportPath string) string {
	return executionFailureCode(reportPath)
}

func executionFailureCode(reportPath string) string {
	data, _, err := evidencefile.ReadRegular(reportPath, scenarioCommandOutputLimit)
	if err != nil || len(data) == 0 {
		return udonreport.CodeUnclassified
	}
	return udonreport.FailureCode(data)
}

func unavailableScenario(manifest Manifest, phase string) ScenarioResult {
	return ScenarioResult{ID: manifest.ID, Status: StatusSkipped, Attempts: 1, Detail: "dependency_unavailable", Phases: []PhaseResult{{ID: phase, Status: StatusSkipped, Detail: "dependency_unavailable"}}}
}

func failedScenario(manifest Manifest, phase, detail string) ScenarioResult {
	return ScenarioResult{ID: manifest.ID, Status: StatusFail, Attempts: 1, Detail: detail, Phases: []PhaseResult{{ID: phase, Status: StatusFail, Detail: detail}}}
}

func appendFailure(result ScenarioResult, phase, detail string) ScenarioResult {
	result.Status, result.Detail = StatusFail, detail
	result.Phases = append(result.Phases, PhaseResult{ID: phase, Status: StatusFail, Detail: detail})
	return result
}

func equalStrings(left, right []string) bool {
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

func btoi(value bool) int {
	if value {
		return 1
	}
	return 0
}

var scenarioEnvironmentName = regexp.MustCompile(`^OPENUDON_SCENARIO_[A-Z0-9_]+$`)

const (
	publicScenarioAction  = "canary"
	publicScenarioFlow    = "open_public_canary"
	publicScenarioSession = "public_canary_session"
)
