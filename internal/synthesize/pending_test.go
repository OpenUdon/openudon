package synthesize

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

func TestPendingPackageUsesPublicContractsAndAssessment(t *testing.T) {
	root := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(root, "project.md"), []byte("# Project\n\n## Goal\nReview an unresolved report.\n\n"))
	fields := &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{"report": {Type: "string"}}}
	intent := &workflowintent.Intent{Workflow: &workflowintent.WorkflowMeta{Name: "report", Description: "Review an unresolved report"}, Steps: []*workflowintent.Step{{Name: "review", Type: "pending", Pending: &uws1.PendingStep{Purpose: "Obtain the report", Inputs: &uws1.ParamSchema{Type: "object"}, Outputs: fields, Effect: "unknown"}}}}
	hcl, err := workflowintent.RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteSynthesizeTestFile(t, filepath.Join(root, "workflows/intent.hcl"), []byte(hcl))
	result, report, err := PackageFromIntent(context.Background(), Options{ExampleDir: root})
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || report == nil || report.Passed() {
		t.Fatalf("pending package cannot pass execution assessment: %+v", report)
	}
	data, err := os.ReadFile(result.UWSPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc uws1.Document
	if err := convert.UnmarshalYAML(data, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.12.0" || len(doc.Operations) != 0 || doc.Workflows[0].Steps[0].Pending == nil {
		t.Fatal("pending package was bound or lost")
	}
	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
	if doc.ValidateExecutable() == nil {
		t.Fatal("pending package became executable")
	}
	found := false
	for _, c := range report.Checks {
		if c.Code == "uws.pending_steps" && c.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Fatal("assessment did not distinguish pending contracts")
	}
}

func TestConfirmedEffectEmissionRetainsLegacyVersion(t *testing.T) {
	intent, err := workflowintent.ParseIntentFile(filepath.Join("..", "..", "examples", "eval", "runtime-only-render", "reference", "intent.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	intent.Steps[0].Effect = "write"
	result := resultPaths(t.TempDir())
	doc, err := generateWorkflowDocument(result, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.12.0" || doc.Operations[0].Effect != "write" {
		t.Fatal("confirmed effect lost")
	}
	mustWriteSynthesizeTestFile(t, result.WorkflowPath, []byte(`uws = "1.11.0"`))
	doc, err = generateWorkflowDocument(result, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.11.0" || doc.Operations[0].Effect != "" {
		t.Fatal("legacy wire version changed")
	}
}
