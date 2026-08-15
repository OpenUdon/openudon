package elicitor

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBrowserAuthenticationDiscoveryAndReadiness(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "example")
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(root, "member-auth.yaml")
	if err := os.WriteFile(authPath, browserAuthenticationFixture(), 0o600); err != nil {
		t.Fatal(err)
	}
	browserPath := filepath.Join(root, "member.browser.json")
	if err := os.WriteFile(browserPath, browserProfileFixture(false, true), 0o600); err != nil {
		t.Fatal(err)
	}
	at := time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)
	discovery, err := DiscoverAuthoringSourcesWithBrowser(context.Background(), example, "open a member link", nil, nil, []BrowserSourceInput{{ID: "member-auth", Path: authPath}, {ID: "member", Path: browserPath}}, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Docs) != 2 || len(discovery.Plans) != 2 {
		t.Fatalf("discovery docs/plans = %d/%d", len(discovery.Docs), len(discovery.Plans))
	}
	var authDoc, actionDoc APIDocument
	for _, doc := range discovery.Docs {
		if isBrowserAuthenticationDocument(doc) {
			authDoc = doc
		} else if isBrowserActionDocument(doc) {
			actionDoc = doc
		}
	}
	if authDoc.RelativePath != "browser-authentication/member-auth.yaml" || len(authDoc.Operations) != 1 {
		t.Fatalf("authentication document = %#v", authDoc)
	}
	session := Session{
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "member_link", Description: "Open a member-only link."},
			Steps:    []*rollout.Step{{Name: "open_member", Type: "browser", Source: actionDoc.RelativePath, Operation: actionDoc.Operations[0].OperationID}},
			Outputs:  []*rollout.Output{{Name: "result", From: "steps.open_member.response.body"}},
		},
		SourcePlan: discovery.Plans, BrowserRoute: "browser", Safety: "after-approval", SafetySet: true,
		SideEffectScope: projectwizard.SideEffectAfterApproval,
	}
	issues := CheckReadiness(session, discovery.Docs)
	if !hasBrowserAuthenticationReadinessCode(issues, readinessMissingBrowserAuthenticationFlow) {
		t.Fatalf("readiness issues missing authentication flow: %#v", issues)
	}
	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps.open_member.authentication_flow"}}, authDoc.RelativePath+"#"+authDoc.Operations[0].OperationID, discovery.Docs)
	if len(session.Intent.Steps) != 2 || session.Intent.Steps[0].Type != "browser_authentication" || session.Intent.Steps[1].BrowserSession == "" || session.Intent.Steps[0].BrowserSession != session.Intent.Steps[1].BrowserSession {
		t.Fatalf("authentication insertion = %#v", session.Intent.Steps)
	}
	authStep := session.Intent.Steps[0]
	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps." + authStep.Name + ".credential_bindings"}}, "username=member_username, password=member_password", discovery.Docs)
	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps." + authStep.Name + ".timeout"}}, "120", discovery.Docs)
	applyProgressiveAnswer(&session, QuestionPlan{Slots: []string{"steps." + authStep.Name + ".authentication_approval"}}, "approve "+authStep.Name, discovery.Docs)
	issues = CheckReadiness(session, discovery.Docs)
	for _, code := range []string{readinessMissingBrowserAuthenticationFlow, readinessMissingBrowserAuthenticationSession, readinessMissingBrowserCredentialBindings, readinessMissingBrowserAuthenticationTimeout, readinessUnconfirmedBrowserAuthentication} {
		if hasBrowserAuthenticationReadinessCode(issues, code) {
			t.Fatalf("readiness still contains %s: %#v", code, issues)
		}
	}
}

func hasBrowserAuthenticationReadinessCode(issues []ReadinessIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestBrowserAuthenticationBindingsUsePortableUWSIdentifiers(t *testing.T) {
	required := []string{"username"}
	if !exactBrowserCredentialBindings(map[string]string{"username": "member_username"}, required) {
		t.Fatal("portable binding was rejected")
	}
	for _, binding := range []string{"member.username", "_member", "member username"} {
		if exactBrowserCredentialBindings(map[string]string{"username": binding}, required) {
			t.Fatalf("non-portable binding %q was accepted", binding)
		}
	}
}

func browserAuthenticationFixture() []byte {
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
    description: Sign in and approve the push notification.
    sequence:
      - navigate: https://example.test/
      - type_credential: {locator: {role: textbox, name: Username}, slot: username}
      - type_credential: {locator: {role: textbox, name: Password}, slot: password}
      - click: {locator: {role: button, name: Sign in}}
      - challenge: {kind: push}
      - wait_for: {locator: {role: heading, name: Member dashboard}}
    effects: [establishes_session, sends_mfa_challenge]
    success: {origin: https://example.test, locator: {role: heading, name: Member dashboard}}
`)
}
