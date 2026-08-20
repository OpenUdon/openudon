package trustedrunner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/OpenUdon/openudon/internal/synthesize"
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

func TestRuntimeApprovalIDsRejectLoweringCollisions(t *testing.T) {
	if _, err := runtimeApprovalIDs([]string{"read-a", "read_a"}); err == nil {
		t.Fatal("colliding runtime approval IDs were accepted")
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
