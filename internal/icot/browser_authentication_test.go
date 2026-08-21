package icot

import (
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/icot/artifactwriter"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBrowserAuthenticationMetadataContainsOnlySafeReviewState(t *testing.T) {
	session := elicitor.Session{
		SourcePlan: []elicitor.SourceMaterialization{{
			Kind: "browser-authentication", ID: "member", TargetPath: "browser-authentication/member.yaml",
			SHA256: strings.Repeat("a", 64), Title: "Member login", Flows: []string{"member_login_push"},
			FlowCredentialSlots: map[string][]string{"member_login_push": {"password", "username"}},
			Origins:             []string{"https://members.example.test", "https://login.example.test"}, Lifecycle: "active",
			ExpiresAt: "2026-09-14T00:00:00Z", Provenance: "reviewed local observation",
		}},
		Intent: rollout.Intent{Steps: []*rollout.Step{
			{Name: "authenticate", Type: "browser_authentication", BrowserSession: "member_portal"},
			{Name: "open_link", Type: "browser", BrowserSession: "member_portal"},
		}},
		BrowserAuthenticationApprovals: []string{"authenticate"},
	}
	data, ok, err := artifactwriter.BrowserAuthenticationMetadataJSON(session)
	if err != nil || !ok {
		t.Fatalf("metadata: ok=%t err=%v", ok, err)
	}
	for _, want := range []string{"openudon.browser-authentication-review.v1", "member_login_push", "member_portal", "authenticate", "password", "username"} {
		if !strings.Contains(data, want) {
			t.Fatalf("metadata missing %q:\n%s", want, data)
		}
	}
	for _, forbidden := range []string{"credential_value", "otp_value", "cookie", "storage_state"} {
		if strings.Contains(data, forbidden) {
			t.Fatalf("metadata contains forbidden field %q:\n%s", forbidden, data)
		}
	}
}

func TestBrowserAuthenticationMetadataRecursesThroughStructuralSteps(t *testing.T) {
	session := elicitor.Session{
		SourcePlan: []elicitor.SourceMaterialization{{
			Kind: "browser-authentication", ID: "member", TargetPath: "browser-authentication/member.yaml",
			SHA256: strings.Repeat("a", 64), Flows: []string{"member_passkey"},
			FlowCredentialSlots: map[string][]string{"member_passkey": {}},
			Origins:             []string{"https://members.example.test"}, Lifecycle: "active",
			ExpiresAt: "2026-09-14T00:00:00Z", Provenance: "reviewed local observation",
		}},
		Intent: rollout.Intent{Steps: []*rollout.Step{{
			Name: "sign_in_sequence", Type: "sequence", Steps: []*rollout.Step{
				{Name: "authenticate_nested", Type: "browser_authentication", BrowserSession: "member_portal"},
				{Name: "open_nested", Type: "browser", BrowserSession: "member_portal"},
			},
		}}},
		BrowserAuthenticationApprovals: []string{"authenticate_nested"},
	}
	data, ok, err := artifactwriter.BrowserAuthenticationMetadataJSON(session)
	if err != nil || !ok {
		t.Fatalf("metadata: ok=%t err=%v", ok, err)
	}
	for _, step := range []string{"authenticate_nested", "open_nested"} {
		if !strings.Contains(data, `"step": "`+step+`"`) {
			t.Fatalf("metadata missing nested session step %q:\n%s", step, data)
		}
	}
}
