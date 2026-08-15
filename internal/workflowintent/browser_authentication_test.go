package workflowintent

import (
	"strings"
	"testing"
)

func TestBrowserAuthenticationIntentRoundTrip(t *testing.T) {
	timeout := 120.0
	intent := &Intent{
		Workflow: &WorkflowMeta{Name: "member_link", Description: "Open a member-only link."},
		Steps: []*Step{
			{Name: "authenticate", Type: "browser_authentication", Source: "browser-authentication/member.yaml", AuthenticationFlow: "member_login_push", BrowserSession: "member_portal", CredentialBindings: map[string]string{"username": "member_username", "password": "member_password"}, Timeout: &timeout},
			{Name: "open_link", Type: "browser", Source: "browser-profiles/member.yaml", Operation: "open_member_link", BrowserSession: "member_portal"},
		},
	}
	hcl, err := RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"browser_authentication"`, `authentication_flow`, `"member_login_push"`, `browser_session`, `"member_portal"`, `credential_bindings`} {
		if !strings.Contains(hcl, want) {
			t.Fatalf("rendered intent missing %q:\n%s", want, hcl)
		}
	}
	parsed, err := ParseIntent([]byte(hcl), "intent.hcl")
	if err != nil {
		t.Fatal(err)
	}
	if got := parsed.Steps[0]; got.AuthenticationFlow != "member_login_push" || got.BrowserSession != "member_portal" || got.CredentialBindings["password"] != "member_password" {
		t.Fatalf("authentication step = %#v", got)
	}
}

func TestBrowserAuthenticationIntentRequiresBoundedTimeout(t *testing.T) {
	timeout := 601.0
	intent := &Intent{Steps: []*Step{{
		Name: "authenticate", Type: "browser_authentication", Source: "browser-authentication/member.yaml", AuthenticationFlow: "login", BrowserSession: "member", CredentialBindings: map[string]string{"username": "member_username"}, Timeout: &timeout,
	}}}
	if _, err := RenderIntentHCL(intent); err == nil || !strings.Contains(err.Error(), "no greater than 600") {
		t.Fatalf("RenderIntentHCL error = %v", err)
	}
}
