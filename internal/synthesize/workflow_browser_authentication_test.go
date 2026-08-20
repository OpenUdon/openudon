package synthesize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/browserauthentication"
	"github.com/OpenUdon/uws/schemas"
	"github.com/OpenUdon/uws/uws1"
)

func TestGenerateWorkflowSelectsUWS18ForContextAuthentication(t *testing.T) {
	if _, err := schemas.BrowserAuthenticationProfileSchema("uws.browser-authentication.1.1"); err != nil {
		t.Skip("the standalone dependency pin predates UWS 1.8")
	}
	example := t.TempDir()
	relative := filepath.ToSlash(filepath.Join("browser-authentication", "member.json"))
	path := filepath.Join(example, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := strings.Replace(string(synthesizeBrowserAuthenticationFixture()), "uws.browser-authentication.1.0", "uws.browser-authentication.1.1", 1)
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_login"},
		Steps: []*rollout.Step{{
			Name: "authenticate", Type: "browser_authentication", Source: relative,
			AuthenticationFlow: "member_login", BrowserSession: "member_portal",
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.8.0" {
		t.Fatalf("UWS version = %q, want 1.8.0", doc.UWS)
	}
	if got := doc.Operations[0].Extensions[uws1.ExtensionOperationProfile]; got != "uws.browser-authentication-call.1.1" {
		t.Fatalf("authentication call profile = %#v", got)
	}
}

func TestGenerateWorkflowSelectsUWS18ForContextCapability(t *testing.T) {
	if _, err := schemas.BrowserSourceProfileSchema("uws.browser.1.6"); err != nil {
		t.Skip("the standalone dependency pin predates UWS 1.8")
	}
	example := t.TempDir()
	relative := filepath.ToSlash(filepath.Join("browser-profiles", "member.json"))
	path := filepath.Join(example, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := strings.Replace(string(synthesizeBrowserProfileFixture(false, false, "query")), "uws.browser.1.5", "uws.browser.1.6", 1)
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_dashboard"},
		Steps:    []*rollout.Step{{Name: "read_status", Type: "browser", Source: relative, Operation: "read_status"}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.8.0" {
		t.Fatalf("UWS version = %q, want 1.8.0", doc.UWS)
	}
}

func TestGenerateWorkflowSelectsUWS19ForScalarAccessibilityCapability(t *testing.T) {
	if _, err := schemas.BrowserSourceProfileSchema("uws.browser.1.7"); err != nil {
		t.Fatal("the pinned UWS dependency lacks browser 1.7:", err)
	}
	example := t.TempDir()
	relative := filepath.ToSlash(filepath.Join("browser-profiles", "member.json"))
	path := filepath.Join(example, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := strings.Replace(string(synthesizeBrowserProfileFixture(false, false, "query")), "uws.browser.1.5", "uws.browser.1.7", 1)
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_dashboard"},
		Steps:    []*rollout.Step{{Name: "read_balance", Type: "browser", Source: relative, Operation: "read_status"}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.9.0" {
		t.Fatalf("UWS version = %q, want 1.9.0", doc.UWS)
	}
}

func TestGenerateWorkflowLowersBrowserAuthenticationAndNamedSession(t *testing.T) {
	example := t.TempDir()
	writeWorkflowBrowserFixture(t, example, "browser-authentication/member.yaml", synthesizeBrowserAuthenticationFixture())
	writeWorkflowBrowserFixture(t, example, "browser-profiles/member.yaml", synthesizeBrowserProfileFixture(false, true, "query"))
	timeout := 120.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_link", Description: "Open a member-only link."},
		Steps: []*rollout.Step{
			{Name: "authenticate", Type: "browser_authentication", Source: "browser-authentication/member.yaml", AuthenticationFlow: "member_login_push", BrowserSession: "member_portal", CredentialBindings: map[string]string{"username": "member_username", "password": "member_password"}, Timeout: &timeout},
			{Name: "open_link", Type: "browser", Source: "browser-profiles/member.yaml", Operation: "read_status", BrowserSession: "member_portal"},
		},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
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
	example := t.TempDir()
	writeWorkflowBrowserFixture(t, example, "browser-authentication/member.yaml", synthesizeCredentiallessBrowserAuthenticationFixture())
	timeout := 120.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_passkey", Description: "Sign in with a passkey."},
		Steps: []*rollout.Step{{
			Name: "authenticate", Type: "browser_authentication", Source: "browser-authentication/member.yaml",
			AuthenticationFlow: "member_passkey", BrowserSession: "member_portal", Timeout: &timeout,
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
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

func TestGenerateWorkflowFailsClosedForUnreadableBrowserProfile(t *testing.T) {
	intent := &rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "browser"}, Steps: []*rollout.Step{{
		Name: "read_status", Type: "browser", Source: "browser-profiles/missing.yaml", Operation: "read_status",
	}}}
	if _, err := generateWorkflowDocument(Result{ExampleDir: t.TempDir()}, intent); err == nil || !strings.Contains(err.Error(), "read browser-profiles/missing.yaml") {
		t.Fatalf("missing browser profile did not fail closed: %v", err)
	}
}

func writeWorkflowBrowserFixture(t *testing.T, root, relative string, data []byte) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
