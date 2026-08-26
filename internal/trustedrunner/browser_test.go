package trustedrunner

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/registrationattestation"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBuildBrowserRunConfigDerivesReviewedRuntimeContract(t *testing.T) {
	root := t.TempDir()
	driver := filepath.Join(t.TempDir(), "browserdriver")
	if err := os.WriteFile(driver, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	timeout := 120.0
	intent := &rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "member"}, Steps: []*rollout.Step{
		{Name: "authenticate-member", Type: "browser_authentication", Source: "browser-authentication/member.yaml", AuthenticationFlow: "member_login", BrowserSession: "member", CredentialBindings: map[string]string{"password": "member_password"}, Timeout: &timeout},
		{Name: "read-dashboard", Type: "browser", Source: "browser-profiles/member.json", Operation: "read_dashboard", BrowserSession: "member"},
		{Name: "read_existing", Type: "browser", Source: "browser-profiles/member.json", Operation: "read_dashboard", BrowserSession: "existing_member"},
	}}
	writeBrowserRuntimeFixture(t, root, intent)

	config, err := buildBrowserRunConfig(root, driver, []string{"--headless"}, []string{"PATH=/trusted/bin", "HOME=/trusted/home", "HTTP_PROXY=http://must-not-pass"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if config.Protocol != "v3" || config.DriverPath != driver || !reflect.DeepEqual(config.DriverArgs, []string{"--headless"}) {
		t.Fatalf("browser driver contract = %#v", config)
	}
	if !reflect.DeepEqual(config.DriverEnvironment, []string{"HOME", "PATH"}) {
		t.Fatalf("browser driver environment = %#v", config.DriverEnvironment)
	}
	if want := []string{"authenticate_member"}; !reflect.DeepEqual(config.ApprovedAuthentication, want) {
		t.Fatalf("authentication approvals = %#v, want %#v", config.ApprovedAuthentication, want)
	}
	if want := []string{"read_dashboard"}; !reflect.DeepEqual(config.ApprovedOperations, want) {
		t.Fatalf("operation approvals = %#v, want %#v", config.ApprovedOperations, want)
	}
	if want := []string{"member_password"}; len(config.CredentialEnvironment) != 1 || config.CredentialEnvironment[0].Name != want[0] {
		t.Fatalf("credential mappings = %#v", config.CredentialEnvironment)
	}
	if len(config.SessionEnvironment) != 1 || config.SessionEnvironment[0].Name != "existing_member" {
		t.Fatalf("external session mappings = %#v", config.SessionEnvironment)
	}
}

func TestBuildBrowserRunConfigRequiresDriverOnlyForExecution(t *testing.T) {
	root := t.TempDir()
	intent := &rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "member"}, Steps: []*rollout.Step{{Name: "read_dashboard", Type: "browser", Source: "browser-profiles/member.json", Operation: "read_dashboard"}}}
	writeBrowserRuntimeFixture(t, root, intent)
	if _, err := buildBrowserRunConfig(root, "", nil, nil, false); err == nil {
		t.Fatal("real browser run without a driver was accepted")
	}
	if config, err := buildBrowserRunConfig(root, "", nil, nil, true); err != nil || config == nil {
		t.Fatalf("dry browser validation = %#v, %v", config, err)
	}
}

func TestBuildBrowserRunConfigIgnoresInertAPIFallbackProfiles(t *testing.T) {
	root := t.TempDir()
	intent := &rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "member-api"}, Steps: []*rollout.Step{{Name: "read_dashboard", Type: "http", OpenAPI: "openapi/member.yaml", Operation: "read_dashboard"}}}
	writeBrowserRuntimeFixture(t, root, intent)
	config, err := buildBrowserRunConfig(root, "", nil, nil, false)
	if err != nil {
		t.Fatalf("API-first package with inert browser fallback was rejected: %v", err)
	}
	if config != nil {
		t.Fatalf("API-first package acquired browser runtime authority: %#v", config)
	}
	if _, err := buildBrowserRunConfig(root, "/tmp/browserdriver", nil, nil, false); err == nil {
		t.Fatal("browser driver was accepted for an API-only active workflow")
	}
}

func TestValidatedPackageBrowserConfigStaysBoundToImmutableSnapshot(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	timeout := 120.0
	intent := &rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "member"}, Steps: []*rollout.Step{
		{Name: "authenticate-member", Type: "browser_authentication", Source: "browser-authentication/member.yaml", AuthenticationFlow: "member_login", BrowserSession: "member", CredentialBindings: map[string]string{"password": "member_password"}, Timeout: &timeout},
		{Name: "read-dashboard", Type: "browser", Source: "browser-profiles/member.json", Operation: "read_dashboard", BrowserSession: "member"},
	}}
	writeBrowserRuntimeFixture(t, example, intent)

	handoffPath := filepath.Join(example, "expected", "review-handoff.json")
	data, err := os.ReadFile(handoffPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest authoring.ReviewHandoff
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"browser-profiles/member.json", "browser-authentication/member.yaml",
		".icot/browser-sources.json", ".icot/browser-authentication.json",
	} {
		manifest.HandoffInputs = append(manifest.HandoffInputs, authoring.ReviewHandoffInput{Path: path, Purpose: "browser runtime input", Required: true})
	}
	manifest.CredentialBindings.Declared = []string{"member_password"}
	manifest.CredentialBindings.ExpectedFromPlan = []string{"member_password"}
	refreshFixtureHandoffDigests(t, example, &manifest)
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, handoffPath, append(data, '\n'))

	validated, err := resolveAndValidatePackageBytes(root, example)
	if err != nil {
		t.Fatal(err)
	}
	before, err := buildBrowserRunConfigFromSnapshot(validated.snapshot, "", nil, []string{"PATH=/trusted/bin"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if before == nil || !reflect.DeepEqual(before.ApprovedOperations, []string{"read_dashboard"}) || !reflect.DeepEqual(before.ApprovedAuthentication, []string{"authenticate_member"}) {
		t.Fatalf("snapshot browser config = %#v", before)
	}

	apiOnly := &rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "changed"}, Steps: []*rollout.Step{{Name: "api", Type: "http", OpenAPI: "openapi/support.yaml", Operation: "getSupport"}}}
	writeBrowserRuntimeFixture(t, example, apiOnly)
	after, err := buildBrowserRunConfigFromSnapshot(validated.snapshot, "", nil, []string{"PATH=/trusted/bin"}, true)
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("snapshot changed after package mutation: before=%#v after=%#v err=%v", before, after, err)
	}

	config, err := buildRunConfig(validated.paths, validated.manifest, validated.snapshot, validated.packageSHA256,
		TierSandbox, filepath.Join(root, "work"), "0123456789abcdef0123456789abcdef",
		validated.handoffSHA256, strings.Repeat("a", 64), before)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := udonrunner.Prepare(context.Background(), config, udonrunner.Options{RepoRoot: root, Env: []string{"PATH=/trusted/bin"}}); err == nil ||
		(!strings.Contains(err.Error(), "digest") && !strings.Contains(err.Error(), "does not match")) {
		t.Fatalf("staging accepted current-file drift: %v", err)
	}
}

func TestBrowserRegistrationConfigSelectsV4OnlyForExecution(t *testing.T) {
	timeout := 300.0
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name: "register_test_user", Type: "browser_registration", Do: "Create one dedicated test identity.",
		Source: "browser-registration/dedicated.yaml", RegistrationFlow: "create_dedicated_test_user",
		RegistrationApproval: "approve_account_creation", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail",
		AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "retain_dedicated_test_identity",
		CredentialBindings: map[string]string{"identifier": "test_identifier", "password": "test_password"}, Timeout: &timeout,
	}}}
	intentData, err := rollout.RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{
		"expected/plan.json": []byte(`{"version":"openudon.workflow-plan.v1","steps":[{"name":"register_test_user","type":"browser_registration"}]}`),
		rollout.IntentPath:   []byte(intentData),
		"browser-registration/dedicated.yaml": []byte(`profile: uws.browser-registration.1.0
info:
  title: Dedicated registration
  applicationOrigins: [https://example.test]
  registrationOrigins: [https://example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-25T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-25T00:00:00Z"}
credentialSlots:
  identifier: {kind: identifier}
  password: {kind: password}
flows:
  create_dedicated_test_user:
    sequence:
      - navigate: https://example.test/register
      - type_credential: {locator: {role: textbox, name: Test identifier}, slot: identifier}
      - type_credential: {locator: {role: textbox, name: Test password}, slot: password}
      - submit: {locator: {role: button, name: Create account}}
      - wait_for: {locator: {role: heading, name: Complete}}
    effects: [creates_account]
    confirmationPolicy: {required: true}
    success: {origin: https://example.test, locator: {role: heading, name: Complete}}
`),
		packageartifacts.BrowserRegistrationReviewPath: []byte(`{"version":"openudon.browser-registration-review.v1","registration_calls":[{"step":"register_test_user","source":"browser-registration/dedicated.yaml","flow":"create_dedicated_test_user","credential_bindings":{"identifier":"test_identifier","password":"test_password"},"approval":"approve_account_creation","duplicate_prevention":"operator_attestation","on_duplicate":"fail","ambiguous_outcome":"stop_without_retry","cleanup_disposition":"retain_dedicated_test_identity","timeout":300}],"sources":[]}`),
	}
	read := func(path string) ([]byte, error) {
		value, ok := files[path]
		if !ok {
			return nil, os.ErrNotExist
		}
		return value, nil
	}
	config, err := buildBrowserRunConfigFromBytes("synthetic", nil, nil, []string{"browser-registration/dedicated.yaml"}, read, "", nil, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if config.Protocol != "v3" || !reflect.DeepEqual(config.ApprovedRegistration, []string{"approve_account_creation"}) {
		t.Fatalf("registration config = %#v", config)
	}
	if len(config.CredentialEnvironment) != 2 || config.CredentialEnvironment[0].Name != "test_identifier" || config.CredentialEnvironment[1].Name != "test_password" {
		t.Fatalf("registration credential mappings = %#v", config.CredentialEnvironment)
	}
	live, err := buildBrowserRunConfigFromBytes("synthetic", nil, nil, []string{"browser-registration/dedicated.yaml"}, read, "/trusted/browserdriver", nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if live.Protocol != "v4" || len(live.AttestedRegistration) != 0 {
		t.Fatalf("unattested live registration config = %#v", live)
	}
}

func TestTrustedRunnerBrowserRegistrationDryRunNeverInvokesExecutor(t *testing.T) {
	extra := []string{
		"browser-registration/dedicated.yaml",
		"browser-registration/dedicated.review.json",
		packageartifacts.BrowserRegistrationReviewPath,
	}
	root, example := writeFixture(t, fixtureOptions{
		extraRequiredInputs: extra,
		credentialBindings:  []string{"test_identifier", "test_password"},
	})
	timeout := 300.0
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name: "register_test_user", Type: "browser_registration", Do: "Create one dedicated test identity.",
		Source: "browser-registration/dedicated.yaml", RegistrationFlow: "create_dedicated_test_user",
		RegistrationApproval: "register_test_user", DuplicatePrevention: "operator_attestation", OnDuplicate: "fail",
		AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately",
		CredentialBindings: map[string]string{"identifier": "test_identifier", "password": "test_password"}, Timeout: &timeout,
	}}}
	intentData, err := rollout.RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(example, filepath.FromSlash(rollout.IntentPath)), []byte(intentData))
	mustWriteFile(t, filepath.Join(example, "expected", "plan.json"), []byte(`{"version":"openudon.workflow-plan.v1","steps":[{"name":"register_test_user","type":"browser_registration"}]}`))
	mustWriteFile(t, filepath.Join(example, "browser-registration", "dedicated.yaml"), []byte(`profile: uws.browser-registration.1.0
info:
  title: Dedicated registration
  applicationOrigins: [https://example.test]
  registrationOrigins: [https://example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-04-29T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-04-29T00:00:00Z"}
credentialSlots:
  identifier: {kind: identifier}
  password: {kind: password}
flows:
  create_dedicated_test_user:
    sequence:
      - navigate: https://example.test/register
      - type_credential: {locator: {role: textbox, name: Test identifier}, slot: identifier}
      - type_credential: {locator: {role: textbox, name: Test password}, slot: password}
      - submit: {locator: {role: button, name: Create account}}
      - wait_for: {locator: {role: heading, name: Complete}}
    effects: [creates_account]
    confirmationPolicy: {required: true}
    success: {origin: https://example.test, locator: {role: heading, name: Complete}}
`))
	mustWriteFile(t, filepath.Join(example, "browser-registration", "dedicated.review.json"), []byte(`{"version":"browsertools.registration-review.v1"}`))
	mustWriteFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserRegistrationReviewPath)), []byte(`{"version":"openudon.browser-registration-review.v1","registration_calls":[{"step":"register_test_user","source":"browser-registration/dedicated.yaml","flow":"create_dedicated_test_user","credential_bindings":{"identifier":"test_identifier","password":"test_password"},"approval":"register_test_user","duplicate_prevention":"operator_attestation","on_duplicate":"fail","ambiguous_outcome":"stop_without_retry","cleanup_disposition":"delete_separately","timeout":300}],"sources":[]}`))
	refreshFixtureHandoffFile(t, example)
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	invoked := false
	result, err := Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: "sandbox", ApprovalPath: approvalPath,
		DryRun: true, Now: now, Assess: passAssess,
		Invoke: func(context.Context, udonrunner.Invocation) error { invoked = true; return nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.DryRun || invoked {
		t.Fatalf("dry-run result = %#v, invoked = %v", result, invoked)
	}
	configData, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(configData), "example.test") || !strings.Contains(string(configData), "UDON_CREDENTIAL_TEST_IDENTIFIER") || !strings.Contains(string(configData), `"approved_registration"`) || !strings.Contains(string(configData), `"register_test_user"`) {
		t.Fatalf("unexpected registration dry-run config: %s", configData)
	}
	driver := filepath.Join(t.TempDir(), "browserdriver")
	if err := os.WriteFile(driver, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: "sandbox", ApprovalPath: approvalPath,
		Now: now, Assess: passAssess, BrowserDriver: driver,
		Env:    []string{"UDON_CREDENTIAL_TEST_IDENTIFIER=value", "UDON_CREDENTIAL_TEST_PASSWORD=value"},
		Invoke: func(context.Context, udonrunner.Invocation) error { invoked = true; return nil },
	}); err == nil || !strings.Contains(err.Error(), "attestation") {
		t.Fatalf("live registration error = %v", err)
	}
	if invoked {
		t.Fatal("registration executor was invoked")
	}

	validated, err := resolveAndValidatePackageBytes(root, example)
	if err != nil {
		t.Fatal(err)
	}
	profileData, err := os.ReadFile(filepath.Join(example, "browser-registration", "dedicated.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	profileValue, err := registrationprofile.Parse(profileData)
	if err != nil {
		t.Fatal(err)
	}
	profileDigest, err := registrationprofile.Digest(profileValue)
	if err != nil {
		t.Fatal(err)
	}
	attestationData, err := json.Marshal(registrationattestation.Artifact{
		Version: registrationattestation.Version, PackageSHA256: "sha256:" + validated.packageSHA256, ProfileSHA256: profileDigest,
		Operation: "register_test_user", Flow: "create_dedicated_test_user", PriorAttempts: 0, DedicatedTest: true,
		CleanupDisposition: "delete_separately", Reviewer: "Synthetic Reviewer", ExpiresAt: now().Add(time.Hour).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatal(err)
	}
	attestationPath := filepath.Join(t.TempDir(), "registration-attestation.json")
	if err := os.WriteFile(attestationPath, append(attestationData, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	invoked = false
	if _, err := Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: "sandbox", ApprovalPath: approvalPath,
		Now: now, Assess: passAssess, BrowserDriver: driver, RegistrationAttestationPath: attestationPath,
		Env:    []string{"OPENUDON_EXECUTOR=/bin/true", "UDON_CREDENTIAL_TEST_IDENTIFIER=identifier-value", "UDON_CREDENTIAL_TEST_PASSWORD=password-value"},
		Invoke: func(context.Context, udonrunner.Invocation) error { invoked = true; return nil },
	}); err == nil || !strings.Contains(err.Error(), "--approve-browser-registration") {
		t.Fatalf("missing submit approval error = %v", err)
	}
	if invoked {
		t.Fatal("registration executor was invoked without submit approval")
	}

	result, err = Run(context.Background(), Options{
		RepoRoot: root, ExampleDir: example, Tier: "sandbox", ApprovalPath: approvalPath,
		Now: now, Assess: passAssess, BrowserDriver: driver, RegistrationAttestationPath: attestationPath,
		RegistrationSubmitApproval: "register_test_user",
		Env:                        []string{"OPENUDON_EXECUTOR=/bin/true", "UDON_CREDENTIAL_TEST_IDENTIFIER=identifier-value", "UDON_CREDENTIAL_TEST_PASSWORD=password-value"},
		Invoke: func(_ context.Context, invocation udonrunner.Invocation) error {
			invoked = true
			for _, pair := range [][2]string{{"--browser-driver-protocol", "v4"}, {"--attest-browser-registration", "register_test_user"}, {"--approve-browser-registration", "register_test_user"}} {
				if !containsBrowserArgs(invocation.Argv, pair[0], pair[1]) {
					t.Fatalf("registration invocation missing %q %q: %#v", pair[0], pair[1], invocation.Argv)
				}
			}
			joined := strings.Join(invocation.Argv, " ")
			if strings.Contains(joined, attestationPath) || strings.Contains(joined, "identifier-value") || strings.Contains(joined, "password-value") {
				t.Fatalf("private registration material escaped argv: %s", joined)
			}
			writeUdonExecutionReport(t, argValue(t, invocation.Argv, "--execution-report"), "success", now(), "")
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !invoked || result.DryRun {
		t.Fatalf("attested registration result = %#v invoked=%v", result, invoked)
	}
	for index, path := range []string{result.RunConfigPath, result.RunEvidencePath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		bytes := string(data)
		if strings.Contains(bytes, attestationPath) || strings.Contains(bytes, "identifier-value") || strings.Contains(bytes, "password-value") || index == 0 && !strings.Contains(bytes, "udon.execution-report.v3") {
			t.Fatalf("registration handoff artifact %s is unsafe: %s", path, data)
		}
	}
}

func TestRuntimeApprovalIDsRejectLoweringCollisions(t *testing.T) {
	if _, err := runtimeApprovalIDs([]string{"read-a", "read_a"}); err == nil {
		t.Fatal("colliding runtime approval IDs were accepted")
	}
}

func TestOuterRunnerReceivesPrivateAttestationOnlyThroughClosedEnvironment(t *testing.T) {
	config := RunConfig{Browser: &udonrunner.BrowserConfig{Protocol: "v4"}}
	path := "/outside/repository/registration-attestation.json"
	env := outerRunnerEnvironment([]string{"PATH=/trusted/bin", "UNRELATED=forbidden"}, config, path, "register_test_user")
	if !reflect.DeepEqual(env, []string{
		"OPENUDON_BROWSER_REGISTRATION_ATTESTATION=" + path,
		"OPENUDON_BROWSER_REGISTRATION_SUBMIT_APPROVAL=register_test_user",
		"PATH=/trusted/bin",
	}) {
		t.Fatalf("outer runner environment = %#v", env)
	}
}

func writeBrowserRuntimeFixture(t *testing.T, root string, intent *rollout.Intent) {
	t.Helper()
	hcl, err := rollout.RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	write := func(relative string, data []byte) {
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(rollout.IntentPath, []byte(hcl))
	planSteps := make([]synthesize.PlanStep, 0, len(intent.Steps))
	for _, step := range intent.Steps {
		planSteps = append(planSteps, synthesize.PlanStep{Name: step.Name, Type: step.Type, BrowserSession: step.BrowserSession, CredentialBindings: step.CredentialBindings})
	}
	plan, err := json.Marshal(synthesize.WorkflowPlan{Version: "openudon.workflow-plan.v1", Steps: planSteps})
	if err != nil {
		t.Fatal(err)
	}
	write("expected/plan.json", plan)
	write("browser-profiles/member.json", []byte(`{"profile":"uws.browser.1.7","info":{"title":"Member","origin":"https://members.example.test"},"observationKind":"accessibility_snapshot","evidence":{"learnedAt":"2026-08-20T00:00:00Z","source":"reviewed"},"confidence":"high","expiresAfter":"P30D","verification":{"lastVerifiedAt":"2026-08-20T00:00:00Z","successfulRuns":1,"uiStabilityScore":1},"actions":{"read_dashboard":{"description":"Read dashboard.","sequence":[{"navigate":"/dashboard"},{"wait_for":{"role":"heading","name":"Dashboard"}}],"outputs":{"status":{"type":"string","source":"a11y","locator":{"role":"heading","name":"Dashboard"}}},"sideEffects":["read_only"],"confirmationPolicy":{"required":false}}}}`))
	write("browser-authentication/member.yaml", []byte(`profile: uws.browser-authentication.1.1
info:
  title: Member login
  applicationOrigins: [https://members.example.test]
  authenticationOrigins: [https://members.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-20T00:00:00Z", source: reviewed}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-20T00:00:00Z", successfulRuns: 1, uiStabilityScore: 1}
credentialSlots:
  password: {kind: password}
flows:
  member_login:
    description: Sign in.
    sequence:
      - navigate: https://members.example.test/login
      - type_credential: {locator: {role: textbox, name: Password}, slot: password}
      - click: {locator: {role: button, name: Sign in}}
      - wait_for: {locator: {role: heading, name: Dashboard}}
    effects: [establishes_session]
    success: {origin: https://members.example.test, locator: {role: heading, name: Dashboard}}
`))
	write(".icot/browser-sources.json", []byte(`{"version":"openudon.browser-source-review.v1","route":"browser","session_posture":"named","mutation_approvals":["read-dashboard"],"sources":[]}`))
	write(".icot/browser-authentication.json", []byte(`{"version":"openudon.browser-authentication-review.v1","authentication_approvals":["authenticate-member"],"session_bindings":[],"sources":[]}`))
}

func containsBrowserArgs(values []string, first, second string) bool {
	for index := 0; index+1 < len(values); index++ {
		if values[index] == first && values[index+1] == second {
			return true
		}
	}
	return false
}
