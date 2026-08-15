package synthesize

import (
	"encoding/json"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/browserauthentication"
	"github.com/OpenUdon/uws/schemas"
)

func TestGenerateWorkflowLowersBrowserAuthenticationAndNamedSession(t *testing.T) {
	timeout := 120.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_link", Description: "Open a member-only link."},
		Steps: []*rollout.Step{
			{Name: "authenticate", Type: "browser_authentication", Source: "browser-authentication/member.yaml", AuthenticationFlow: "member_login_push", BrowserSession: "member_portal", CredentialBindings: map[string]string{"username": "member_username", "password": "member_password"}, Timeout: &timeout},
			{Name: "open_link", Type: "browser", Source: "browser-profiles/member.yaml", Operation: "open_member_link", BrowserSession: "member_portal"},
		},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: t.TempDir()}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.7.0" || len(doc.Operations) != 2 {
		t.Fatalf("generated document version/operations = %s/%d", doc.UWS, len(doc.Operations))
	}
	auth, ok, err := browserauthentication.ReadAuthenticationExtension(doc.Operations[0].Extensions)
	if err != nil || !ok {
		t.Fatalf("authentication extension: ok=%t err=%v", ok, err)
	}
	if auth.Profile != "browser-authentication/member.yaml" || auth.Flow != "member_login_push" || auth.Session != "member_portal" || auth.CredentialBindings["password"] != "member_password" {
		t.Fatalf("authentication extension = %#v", auth)
	}
	session, ok, err := browserauthentication.ReadSessionExtension(doc.Operations[1].Extensions)
	if err != nil || !ok || session.Session != "member_portal" {
		t.Fatalf("session extension = %#v, ok=%t err=%v", session, ok, err)
	}
}

func TestGenerateWorkflowLowersCredentiallessBrowserAuthentication(t *testing.T) {
	timeout := 120.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_passkey", Description: "Sign in with a passkey."},
		Steps: []*rollout.Step{{
			Name: "authenticate", Type: "browser_authentication", Source: "browser-authentication/member.yaml",
			AuthenticationFlow: "member_passkey", BrowserSession: "member_portal", Timeout: &timeout,
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: t.TempDir()}, intent)
	if err != nil {
		t.Fatal(err)
	}
	auth, ok, err := browserauthentication.ReadAuthenticationExtension(doc.Operations[0].Extensions)
	if err != nil || !ok || auth.CredentialBindings == nil || len(auth.CredentialBindings) != 0 {
		t.Fatalf("credential-less authentication extension = %#v, ok=%t err=%v", auth, ok, err)
	}
	data, err := json.Marshal(map[string]any{
		browserauthentication.ExtensionAuthentication: doc.Operations[0].Extensions[browserauthentication.ExtensionAuthentication],
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := schemas.ValidateBrowserAuthenticationCallSupplement(data); err != nil {
		t.Fatalf("credential-less authentication call is invalid: %v", err)
	}
}
