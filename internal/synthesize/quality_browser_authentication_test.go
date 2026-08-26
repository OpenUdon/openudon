package synthesize

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestValidateBrowserAuthenticationReview(t *testing.T) {
	example := t.TempDir()
	relative := "browser-authentication/member.yaml"
	absolute := filepath.Join(example, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		t.Fatal(err)
	}
	data := synthesizeBrowserAuthenticationFixture()
	if err := os.WriteFile(absolute, data, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	timeout := 120.0
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name: "authenticate", Type: "browser_authentication", Source: relative, AuthenticationFlow: "member_login_push", BrowserSession: "member_portal",
		CredentialBindings: map[string]string{"username": "member_username", "password": "member_password"}, Timeout: &timeout,
	}}}
	review := browserAuthenticationReview{
		Version: browserAuthenticationReviewVersion, Approvals: []string{"authenticate"},
		Sessions: []browserAuthenticationSessionBinding{{Step: "authenticate", Session: "member_portal"}},
		Sources: []browserAuthenticationReviewedSource{{
			ID: "member", TargetPath: relative, SHA256: hex.EncodeToString(digest[:]), Title: "Member login",
			Flows: []string{"member_login_push"}, FlowCredentialSlots: map[string][]string{"member_login_push": {"password", "username"}},
			Origins: []string{"https://example.test", "https://login.example.test"}, Lifecycle: "active", ExpiresAt: "2026-09-14T00:00:00Z", Provenance: "synthetic_fixture",
		}},
	}
	at := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	if err := validateBrowserAuthenticationReview(example, []string{relative}, intent, review, at); err != nil {
		t.Fatal(err)
	}
	withoutApproval := review
	withoutApproval.Approvals = nil
	if err := validateBrowserAuthenticationReview(example, []string{relative}, intent, withoutApproval, at); err == nil || !strings.Contains(err.Error(), "lacks operation-specific") {
		t.Fatalf("missing approval error = %v", err)
	}
	if err := validateBrowserAuthenticationReview(example, []string{relative}, intent, review, time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expiry error = %v", err)
	}
	invalid, err := intent.Clone()
	if err != nil {
		t.Fatal(err)
	}
	invalid.Steps[0].AuthenticationFlow = "invented"
	if err := validateBrowserAuthenticationReview(example, []string{relative}, invalid, review, at); err == nil || !strings.Contains(err.Error(), "invents authentication flow") {
		t.Fatalf("invented flow error = %v", err)
	}
	invalid, err = intent.Clone()
	if err != nil {
		t.Fatal(err)
	}
	invalid.Steps[0].CredentialBindings["username"] = "member.username"
	if err := validateBrowserAuthenticationReview(example, []string{relative}, invalid, review, at); err == nil || !strings.Contains(err.Error(), "credential bindings") {
		t.Fatalf("non-portable binding error = %v", err)
	}
}

func TestValidateBrowserAuthenticationReviewAcceptsCredentiallessFlow(t *testing.T) {
	example := t.TempDir()
	relative := "browser-authentication/member-passkey.yaml"
	absolute := filepath.Join(example, filepath.FromSlash(relative))
	data := synthesizeCredentiallessBrowserAuthenticationFixture()
	mustWriteSynthesizeTestFile(t, absolute, data)
	digest := sha256.Sum256(data)
	timeout := 120.0
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name: "authenticate", Type: "browser_authentication", Source: relative,
		AuthenticationFlow: "member_passkey", BrowserSession: "member_portal", Timeout: &timeout,
	}}}
	review := browserAuthenticationReview{
		Version: browserAuthenticationReviewVersion, Approvals: []string{"authenticate"},
		Sessions: []browserAuthenticationSessionBinding{{Step: "authenticate", Session: "member_portal"}},
		Sources: []browserAuthenticationReviewedSource{{
			ID: "member-passkey", TargetPath: relative, SHA256: hex.EncodeToString(digest[:]), Title: "Member passkey",
			Flows: []string{"member_passkey"}, FlowCredentialSlots: map[string][]string{"member_passkey": {}},
			Origins: []string{"https://example.test", "https://login.example.test"}, Lifecycle: "active",
			ExpiresAt: "2026-09-14T00:00:00Z", Provenance: "synthetic_fixture",
		}},
	}
	if err := validateBrowserAuthenticationReview(example, []string{relative}, intent, review, time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("credential-less review rejected: %v", err)
	}
}

func TestPackageFromIntentBuildsBrowserAuthenticationWorkflow(t *testing.T) {
	example := t.TempDir()
	authRel := "browser-authentication/member.yaml"
	browserRel := "browser-profiles/member.json"
	authData := synthesizeBrowserAuthenticationFixture()
	browserData := synthesizeBrowserProfileFixture(false, true, "item")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(authRel)), authData)
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(browserRel)), browserData)
	authDigest := sha256.Sum256(authData)
	browserDigest := sha256.Sum256(browserData)
	timeout := 120.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_link", Description: "Sign in and read the reviewed member status."},
		Inputs:   []*rollout.Input{{Name: "item", Type: "string", Required: true}},
		Steps: []*rollout.Step{
			{Name: "authenticate", Type: "browser_authentication", Do: "Establish the reviewed member browser session.", Source: authRel, AuthenticationFlow: "member_login_push", BrowserSession: "member_portal", CredentialBindings: map[string]string{"username": "member_username", "password": "member_password"}, Timeout: &timeout},
			{Name: "read", Type: "browser", Source: browserRel, Operation: "read_status", BrowserSession: "member_portal", With: map[string]string{"item": "inputs.item"}},
		},
		Outputs: []*rollout.Output{{Name: "status", From: "read.received_body.status"}},
	}
	intentHCL, err := rollout.RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "workflows", "intent.hcl"), []byte(intentHCL))
	project := buildMatrixProject("Member Link", "OpenAPI: none required\n\n- Use reviewed browser profiles for this UI-only workflow.", "- browser and browser_authentication are allowed only through reviewed profiles and the trusted Udon runtime.\n- Browser authentication requires explicit approval and a sandbox proof run.", "- No function runtime is required.", "- Use symbolic runtime bindings member_username and member_password; never store values.")
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "project.md"), []byte(project))
	browserReview := `{"version":"openudon.browser-source-review.v1","route":"browser","session_posture":"none","sources":[{"id":"member","target_path":"` + browserRel + `","sha256":"` + hex.EncodeToString(browserDigest[:]) + `","actions":["read_status"],"origins":["https://example.test"],"lifecycle":"active","expires_at":"2026-09-14T00:00:00Z","login_state_required":true,"provenance":"synthetic_fixture"}]}`
	authReview := `{"version":"openudon.browser-authentication-review.v1","authentication_approvals":["authenticate"],"session_bindings":[{"step":"authenticate","session":"member_portal"},{"step":"read","session":"member_portal"}],"sources":[{"id":"member","target_path":"` + authRel + `","sha256":"` + hex.EncodeToString(authDigest[:]) + `","title":"Member login","flows":["member_login_push"],"flow_credential_slots":{"member_login_push":["password","username"]},"origins":["https://example.test","https://login.example.test"],"lifecycle":"active","expires_at":"2026-09-14T00:00:00Z","provenance":"synthetic_fixture"}]}`
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserSourceReviewPath)), []byte(browserReview))
	mustWriteSynthesizeTestFile(t, filepath.Join(example, filepath.FromSlash(packageartifacts.BrowserAuthenticationReviewPath)), []byte(authReview))
	result, report, err := PackageFromIntent(context.Background(), Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("quality report failed: %#v", report.Checks)
	}
	document, err := loadUWSDocumentFile(result.UWSPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := document.Workflows[0].Outputs["status"]; got != "$steps.read.outputs.status" {
		t.Fatalf("browser workflow output = %q", got)
	}
	assertPackageFileContains(t, example, "workflows/workflow.uws.yaml", "uws: 1.7.0", "uws.browser-authentication-call.1.0", "member_login_push", "member_portal")
	inputs, err := packageartifacts.RequiredPackagePaths(result.ExampleDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{authRel, packageartifacts.BrowserAuthenticationReviewPath} {
		if !containsString(inputs, want) {
			t.Fatalf("required package paths missing %s: %#v", want, inputs)
		}
	}
}

func synthesizeBrowserAuthenticationFixture() []byte {
	return []byte(`profile: uws.browser-authentication.1.0
info:
  title: Member login
  applicationOrigins: [https://example.test]
  authenticationOrigins: [https://login.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-15T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-15T00:00:00Z", successfulRuns: 1, uiStabilityScore: 0.95}
credentialSlots:
  username: {kind: identifier}
  password: {kind: password}
flows:
  member_login_push:
    sequence:
      - navigate: https://example.test/
      - type_credential: {locator: {role: textbox, name: Username}, slot: username}
      - type_credential: {locator: {role: textbox, name: Password}, slot: password}
      - challenge: {kind: push}
      - wait_for: {locator: {role: heading, name: Member dashboard}}
    effects: [establishes_session, sends_mfa_challenge]
    success: {origin: https://example.test, locator: {role: heading, name: Member dashboard}}
`)
}

func synthesizeCredentiallessBrowserAuthenticationFixture() []byte {
	return []byte(`profile: uws.browser-authentication.1.0
info:
  title: Member passkey
  applicationOrigins: [https://example.test]
  authenticationOrigins: [https://login.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-15T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-15T00:00:00Z", successfulRuns: 1}
credentialSlots: {}
flows:
  member_passkey:
    sequence:
      - navigate: https://login.example.test/
      - challenge: {kind: passkey}
      - wait_for: {locator: {role: heading, name: Member dashboard}}
    effects: [establishes_session, sends_mfa_challenge]
    success: {origin: https://example.test, locator: {role: heading, name: Member dashboard}}
`)
}
