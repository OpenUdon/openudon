package browserscenario

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/OpenUdon/openudon/internal/icot"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

const scenarioCommandOutputLimit = 1 << 20

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
	executor.prepare(ctx, environment)
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
	return executor.executeLoopback(ctx, manifest, environment)
}

func (executor *realExecutor) prepare(ctx context.Context, environment Environment) {
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
	if !runSilent(ctx, environment.BrowserdriverRepo, []string{npm, "run", "build", "--silent"}, nil) {
		executor.prepareErr = fmt.Errorf("build Browserdriver")
		return
	}
	executor.driverEntry = filepath.Join(environment.BrowserdriverRepo, "dist", "src", "index.js")
	if !regularFile(executor.driverEntry) {
		executor.prepareErr = fmt.Errorf("Browserdriver entry point is unavailable")
		return
	}
	executor.browsertools = filepath.Join(root, "browsertools")
	if !runSilent(ctx, environment.BrowsertoolsRepo, []string{goTool, "build", "-o", executor.browsertools, "./cmd/browsertools"}, nil) {
		executor.prepareErr = fmt.Errorf("build Browsertools")
		return
	}
	executor.udon = filepath.Join(root, "udon")
	if !runSilent(ctx, environment.UdonRepo, []string{goTool, "build", "-o", executor.udon, "./cmd/udon"}, nil) {
		executor.prepareErr = fmt.Errorf("build Udon")
		return
	}
	doctor := runBounded(ctx, environment.BrowsertoolsRepo, []string{executor.browsertools, "playwright", "doctor", "--engine", "chromium", "--format", "json"}, nil, "")
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
	nodeCheck := `import {chromium} from "playwright"; const browser=await chromium.launch({headless:true}); await browser.close();`
	if !runSilent(ctx, environment.BrowserdriverRepo, []string{executor.node, "--input-type=module", "--eval", nodeCheck}, nil) {
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
	fixture.SetRuntime(true)
	variants := append([]string(nil), manifest.ReplayVariants...)
	if len(variants) == 0 {
		variants = []string{""}
	}
	for _, variant := range variants {
		if fixture.SetReplayVariant(variant) != nil {
			fixture.Close()
			_ = os.RemoveAll(caseRoot)
			return appendFailure(result, "udon_v3", "contract_drift")
		}
		replay := executor.runUdon(ctx, manifest, exampleDir, workflow.Path, author.CredentialSlotKinds, bindings)
		if manifest.Expected.Replay == "pass" {
			if replay.failureCode != "" || !scenarioOutputsEqual(replay.outputs, expectedScenarioOutputs(manifest)) {
				fixture.Close()
				_ = os.RemoveAll(caseRoot)
				return appendFailure(result, "browserdriver_replay", "output_mismatch")
			}
		} else if replay.failureCode != manifest.Expected.FailureCode {
			fixture.Close()
			_ = os.RemoveAll(caseRoot)
			return appendFailure(result, "browserdriver_replay", "replay_failed")
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
	command := runBounded(ctx, environment.BrowsertoolsRepo, args, nil, "")
	summary, inspectErr := browserverify.Inspect(livePath, prof, environment.Now)
	if inspectErr != nil {
		if command.err != nil {
			return fail("browsertools_probe", publicCommandFailure(ctx, command))
		}
		return fail("browsertools_probe", "contract_drift")
	}
	if !validPublicLiveSummary(manifest, summary) {
		return fail("browsertools_probe", "shape_drift")
	}
	if command.err != nil {
		return fail("browsertools_probe", publicCommandFailure(ctx, command))
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
	udon := runBounded(ctx, caseRoot, []string{
		executor.udon, "--workdir", caseRoot, "--workflow", workflow.Path, "--workflow-format", "uws-json",
		"--execution-report", "execution-report.json", "--execution-timeout", "60s",
		"--browser-driver", executor.node, "--browser-driver-arg", executor.driverEntry,
		"--browser-driver-protocol", "v2", "--approve-browser-authentication", "authenticate", "--quiet",
	}, nil, "")
	if udon.err != nil {
		return fail("udon_browserdriver_probe", publicCommandFailure(ctx, udon))
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

func publicCommandFailure(ctx context.Context, command boundedCommand) string {
	if ctx.Err() != nil || bytes.Contains(bytes.ToLower(command.stderr), []byte("timeout")) || bytes.Contains(bytes.ToLower(command.stderr), []byte("deadline")) {
		return "timeout"
	}
	if bytes.Contains(bytes.ToLower(command.stderr), []byte("origin")) {
		return "origin_policy_drift"
	}
	return "target_unreachable"
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

func (executor *realExecutor) runUdon(ctx context.Context, manifest Manifest, exampleDir, workflowPath string, slotKinds, bindings map[string]string) replayResult {
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
	} else if manifest.Authentication.ChallengeKind != "" && manifest.Authentication.ChallengeKind != "totp" {
		input = "y\n"
	}
	command := runBounded(ctx, exampleDir, args, environment, input)
	if command.err != nil {
		return replayResult{failureCode: scenarioFailureCode(filepath.Join(exampleDir, "execution-report.json"), manifest)}
	}
	outputs, err := readScenarioOutputs(filepath.Join(exampleDir, "output", "udon.hcl"))
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

func expectedScenarioOutputs(manifest Manifest) map[string]any {
	result := map[string]any{"goal_present": true}
	values := map[string]any{
		"Account name": "Ada Lovelace", "Item count": float64(42), "Usage ratio": float64(-1250),
		"Feature enabled": true, "Plan summary": true, "Primary status": "Ready", "Secondary status": "Healthy",
		"Summary": "Stable summary", "Embedded status": "Embedded ready",
	}
	for _, output := range manifest.Outputs {
		if strings.HasPrefix(output.Name, "Metric ") {
			result[output.Key] = output.Name
			continue
		}
		if output.LocatorMode == "unique_role" {
			result[output.Key] = "Stable summary"
			continue
		}
		result[output.Key] = values[output.Name]
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
		if literalAttribute(block.Body, "from") != "read" || literalAttribute(block.Body, "kind") != "browser" {
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

func runBounded(ctx context.Context, directory string, args []string, environment map[string]string, input string) boundedCommand {
	if len(args) == 0 {
		return boundedCommand{err: fmt.Errorf("empty scenario command")}
	}
	command := exec.CommandContext(ctx, args[0], args[1:]...)
	command.Dir = directory
	command.Env = scenarioEnvironment(environment)
	command.Stdin = strings.NewReader(input)
	var stdout, stderr limitedWriter
	stdout.limit, stderr.limit = scenarioCommandOutputLimit, scenarioCommandOutputLimit
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	if stdout.overflow || stderr.overflow {
		err = fmt.Errorf("scenario subprocess output exceeded its bound")
	}
	return boundedCommand{stdout: stdout.Bytes(), stderr: stderr.Bytes(), err: err}
}

func runSilent(ctx context.Context, directory string, args []string, environment map[string]string) bool {
	result := runBounded(ctx, directory, args, environment, "")
	return result.err == nil
}

func commandOutputContains(ctx context.Context, directory string, args []string, wanted string) bool {
	result := runBounded(ctx, directory, args, nil, "")
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
	allowed := map[string]bool{"DISPLAY": true, "WAYLAND_DISPLAY": true, "XAUTHORITY": true, "XDG_RUNTIME_DIR": true, "DBUS_SESSION_BUS_ADDRESS": true, "HOME": true, "PATH": true, "LANG": true, "LC_ALL": true, "PLAYWRIGHT_BROWSERS_PATH": true, "TMPDIR": true, "NO_PROXY": true, "HTTP_PROXY": true, "HTTPS_PROXY": true, "ALL_PROXY": true, "no_proxy": true, "http_proxy": true, "https_proxy": true, "all_proxy": true}
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

func scenarioFailureCode(reportPath string, manifest Manifest) string {
	data, err := os.ReadFile(reportPath)
	if err != nil || len(data) == 0 || len(data) > scenarioCommandOutputLimit {
		return "invalid_response"
	}
	var report struct {
		Version        string `json:"version"`
		Status         string `json:"status"`
		StartedAt      string `json:"started_at"`
		FinishedAt     string `json:"finished_at"`
		WorkflowPath   string `json:"workflow_path"`
		WorkflowFormat string `json:"workflow_format"`
		WorkDir        string `json:"workdir"`
		OutputPath     string `json:"output_path,omitempty"`
		OutputDigest   string `json:"output_digest,omitempty"`
		ErrorSummary   string `json:"error_summary,omitempty"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&report) != nil || report.Version != "udon.execution-report.v1" || report.Status != "error" ||
		report.StartedAt == "" || report.FinishedAt == "" || report.WorkflowPath == "" || report.WorkflowFormat == "" || report.WorkDir == "" || report.ErrorSummary == "" {
		return "invalid_response"
	}
	lower := strings.ToLower(report.ErrorSummary)
	switch {
	case strings.Contains(lower, "origin_rejected") || strings.Contains(lower, "outside the allowed origin"):
		return "origin_rejected"
	case strings.Contains(lower, "unknown context") || strings.Contains(lower, "context") && manifest.Fault == "context_substitution":
		return "invalid_context"
	case strings.Contains(lower, "secret-like") || strings.Contains(lower, "raw-capture"):
		return "secret_output"
	case strings.Contains(lower, "invalid_response"):
		return "invalid_response"
	default:
		return "invalid_response"
	}
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
