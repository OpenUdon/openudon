package synthesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	uwsconvert "github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

func TestGenerateWorkflowLowersContentTrustAndSelectsUWS191(t *testing.T) {
	example := t.TempDir()
	relative := "browser-profiles/mail.json"
	path := filepath.Join(example, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	fixture := strings.Replace(string(synthesizeBrowserProfileFixture(false, false, "query")), "uws.browser.1.5", "uws.browser.1.7", 1)
	if err := os.WriteFile(path, []byte(fixture), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "Summarize mail"},
		Inputs:   []*rollout.Input{{Name: "locale", Type: "string", Required: true}},
		Triggers: []*rollout.TriggerIntent{{Name: "incoming_mail", Path: "/mail"}},
		Steps:    []*rollout.Step{{Name: "Read Message", Type: "browser", Source: relative, Operation: "read_status"}},
		ContentTrust: &rollout.ContentTrustIntent{
			SourceDescriptions: []*rollout.SourceDescriptionContentTrustIntent{{Source: relative, Level: "untrusted"}},
			Operations:         []*rollout.OperationContentTrustIntent{{Operation: "Read Message", Default: "untrusted", Outputs: map[string]string{"status": "trusted"}}},
			Triggers:           []*rollout.TriggerContentTrustIntent{{Trigger: "incoming_mail", Level: "untrusted"}},
			Workflows:          []*rollout.WorkflowContentTrustIntent{{Workflow: "main", Default: "unknown", Inputs: map[string]string{"locale": "trusted"}}},
		},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.9.1" || doc.ContentTrust == nil {
		t.Fatalf("version/contentTrust = %q/%#v", doc.UWS, doc.ContentTrust)
	}
	if len(doc.SourceDescriptions) != 1 || doc.ContentTrust.SourceDescriptions[doc.SourceDescriptions[0].Name] != uws1.ContentTrustUntrusted {
		t.Fatalf("source trust = %#v for %#v", doc.ContentTrust.SourceDescriptions, doc.SourceDescriptions)
	}
	if declaration := doc.ContentTrust.Operations["Read_Message"]; declaration == nil || declaration.Default != uws1.ContentTrustUntrusted || declaration.Outputs["status"] != uws1.ContentTrustTrusted {
		t.Fatalf("operation trust = %#v", declaration)
	}
	if doc.ContentTrust.Triggers["incoming_mail"] != uws1.ContentTrustUntrusted {
		t.Fatalf("trigger trust = %#v", doc.ContentTrust.Triggers)
	}
	workflowTrust := doc.ContentTrust.Workflows["main"]
	if workflowTrust == nil || workflowTrust.Default != uws1.ContentTrustUnknown || workflowTrust.Inputs["locale"] != uws1.ContentTrustTrusted {
		t.Fatalf("workflow trust = %#v", workflowTrust)
	}
	if doc.Workflows[0].Inputs == nil || doc.Workflows[0].Inputs.Properties["locale"] == nil || doc.Workflows[0].Inputs.Properties["locale"].Type != "string" {
		t.Fatalf("workflow inputs = %#v", doc.Workflows[0].Inputs)
	}
	first, err := uwsconvert.MarshalYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	second, err := uwsconvert.MarshalYAML(doc)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || !strings.Contains(string(first), "contentTrust:") {
		t.Fatalf("content trust YAML is not deterministic:\n%s", first)
	}
	hclData, err := uwsconvert.MarshalHCL(doc)
	if err != nil {
		t.Fatal(err)
	}
	var hclRoundTrip uws1.Document
	if err := uwsconvert.UnmarshalHCL(hclData, &hclRoundTrip); err != nil {
		t.Fatal(err)
	}
	if hclRoundTrip.ContentTrust == nil || hclRoundTrip.ContentTrust.Operations["Read_Message"].Outputs["status"] != uws1.ContentTrustTrusted {
		t.Fatalf("HCL content trust round trip = %#v\n%s", hclRoundTrip.ContentTrust, hclData)
	}
}

func TestGenerateWorkflowContentTrustRejectsUndeclaredOutput(t *testing.T) {
	example := t.TempDir()
	relative := "browser-profiles/mail.json"
	writeWorkflowBrowserFixture(t, example, relative, synthesizeBrowserProfileFixture(false, false, "query"))
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "Read mail"},
		Steps:    []*rollout.Step{{Name: "read_message", Type: "browser", Source: relative, Operation: "read_status"}},
		ContentTrust: &rollout.ContentTrustIntent{Operations: []*rollout.OperationContentTrustIntent{{
			Operation: "read_message", Outputs: map[string]string{"missing": "untrusted"},
		}}},
	}
	_, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err == nil || !strings.Contains(err.Error(), `references undeclared operation output "missing"`) {
		t.Fatalf("generateWorkflowDocument error = %v", err)
	}
}

func TestGenerateWorkflowContentTrustRejectsEmptyRegistry(t *testing.T) {
	intent := &rollout.Intent{
		Workflow:     &rollout.WorkflowMeta{Name: "Render"},
		Steps:        []*rollout.Step{{Name: "render", Type: "fnct", Operation: "render"}},
		ContentTrust: &rollout.ContentTrustIntent{},
	}
	_, err := generateWorkflowDocument(Result{ExampleDir: t.TempDir()}, intent)
	if err == nil || !strings.Contains(err.Error(), "must contain at least one declaration") {
		t.Fatalf("generateWorkflowDocument error = %v", err)
	}
}

func TestGenerateWorkflowContentTrustDefaultDeclaresEntryInputs(t *testing.T) {
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "Render"},
		Inputs:   []*rollout.Input{{Name: "body", Type: "string", Required: true}},
		Steps:    []*rollout.Step{{Name: "render", Type: "fnct", Operation: "render"}},
		ContentTrust: &rollout.ContentTrustIntent{Workflows: []*rollout.WorkflowContentTrustIntent{{
			Workflow: "main", Default: "unknown",
		}}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: t.TempDir()}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Workflows[0].Inputs == nil || doc.Workflows[0].Inputs.Properties["body"] == nil {
		t.Fatalf("workflow default did not materialize entry inputs: %#v", doc.Workflows[0].Inputs)
	}
}
