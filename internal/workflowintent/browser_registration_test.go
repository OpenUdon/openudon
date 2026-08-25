package workflowintent

import (
	"strings"
	"testing"
)

func TestBrowserRegistrationIntentRoundTrip(t *testing.T) {
	timeout := 300.0
	intent := &Intent{Steps: []*Step{{
		Name: "register_test_user", Type: "browser_registration", Do: "Create one dedicated test identity.",
		Source: "browser-registration/dedicated.yaml", RegistrationFlow: "create_dedicated_test_user",
		RegistrationApproval: "register_dedicated_test_user", DuplicatePrevention: "operator_attestation",
		OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "delete_separately",
		CredentialBindings: map[string]string{"identifier": "dedicated_test_identifier", "password": "dedicated_test_password"}, Timeout: &timeout,
	}}}
	data, err := RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"browser_registration", "registration_flow", "registration_approval", "operator_attestation", "stop_without_retry", "delete_separately"} {
		if !strings.Contains(data, want) {
			t.Fatalf("rendered intent missing %q:\n%s", want, data)
		}
	}
	parsed, err := ParseIntent([]byte(data), IntentPath)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Steps[0]
	if got.RegistrationFlow != "create_dedicated_test_user" || got.RegistrationApproval != "register_dedicated_test_user" || got.CredentialBindings["password"] != "dedicated_test_password" {
		t.Fatalf("parsed registration step = %#v", got)
	}
}

func TestBrowserRegistrationIntentFailsClosed(t *testing.T) {
	timeout := 300.0
	base := Step{
		Name: "register", Type: "browser_registration", Do: "Create one dedicated test identity.", Source: "browser-registration/dedicated.yaml",
		RegistrationFlow: "create_test_user", RegistrationApproval: "register_test_user", DuplicatePrevention: "operator_attestation",
		OnDuplicate: "fail", AmbiguousOutcome: "stop_without_retry", CleanupDisposition: "retain_dedicated_test_identity",
		CredentialBindings: map[string]string{"identifier": "test_identifier", "password": "test_password"}, Timeout: &timeout,
	}
	tests := []struct {
		name string
		edit func(*Step)
		want string
	}{
		{"missing approval", func(s *Step) { s.RegistrationApproval = "" }, "registration_approval"},
		{"retry ambiguity", func(s *Step) { s.AmbiguousOutcome = "retry" }, "duplicate and ambiguity"},
		{"session", func(s *Step) { s.BrowserSession = "new_account" }, "browser_session"},
		{"literal credential", func(s *Step) { s.CredentialBindings["password"] = "Bearer " + strings.Repeat("a", 32) }, "symbolic"},
		{"unbounded timeout", func(s *Step) { s.Timeout = nil }, "timeout"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			step := base
			step.CredentialBindings = cloneStringMap(base.CredentialBindings)
			test.edit(&step)
			_, err := RenderIntentHCL(&Intent{Steps: []*Step{&step}})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func cloneStringMap(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for key, value := range values {
		result[key] = value
	}
	return result
}
