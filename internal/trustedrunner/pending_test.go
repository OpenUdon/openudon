package trustedrunner

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

func TestPendingAdmissionRefusesEveryPathDespiteStoredPass(t *testing.T) {
	for _, location := range []string{"hcl-only", "yaml-only", "unselected-branch", "unused-workflow"} {
		t.Run(location, func(t *testing.T) {
			root, example := writeFixture(t, fixtureOptions{})
			approval := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, fixedNow())
			pending := &uws1.Step{StepID: "unresolved", Pending: &uws1.PendingStep{Purpose: "Await a reviewed binding", Inputs: &uws1.ParamSchema{Type: "object"}, Outputs: &uws1.ParamSchema{Type: "object"}, Effect: "write"}}
			doc := &uws1.Document{UWS: "1.12.0", Info: &uws1.Info{Title: "Pending refusal", Version: "1.0.0"}, Workflows: []*uws1.Workflow{{WorkflowID: "main", Type: "sequence", Steps: []*uws1.Step{pending}}}}
			if location == "unused-workflow" {
				doc.Operations = []*uws1.Operation{{OperationID: "noop", Extensions: map[string]any{uws1.ExtensionOperationProfile: "test.mock.1"}}}
				doc.Workflows = []*uws1.Workflow{{WorkflowID: "main", Type: "sequence", Steps: []*uws1.Step{{StepID: "ready", OperationRef: "noop"}}}, {WorkflowID: "unused", Type: "sequence", Steps: []*uws1.Step{pending}}}
			}
			if location == "unselected-branch" {
				doc.Workflows[0].Steps = []*uws1.Step{{StepID: "choice", Type: "switch", Cases: []*uws1.Case{{CaseFields: uws1.CaseFields{Name: "never", When: "false"}, Steps: []*uws1.Step{pending}}}}}
			}
			if err := doc.Validate(); err != nil {
				t.Fatal(err)
			}
			hcl, err := convert.MarshalHCL(doc)
			if err != nil {
				t.Fatal(err)
			}
			yaml, err := convert.MarshalYAML(doc)
			if err != nil {
				t.Fatal(err)
			}
			if location != "yaml-only" {
				mustWriteFile(t, filepath.Join(example, "workflows/workflow.hcl"), hcl)
			}
			if location != "hcl-only" {
				mustWriteFile(t, filepath.Join(example, "workflows/workflow.uws.yaml"), yaml)
			}
			refreshFixtureHandoffFile(t, example)
			before := snapshotTree(t, root)
			assessor := func(context.Context, synthesize.Options) (*synthesize.QualityReport, error) {
				t.Fatal("pending package reached assessor")
				return nil, nil
			}
			if _, err := ApprovalTemplate(context.Background(), TemplateOptions{RepoRoot: root, ExampleDir: example, State: StateApprovedForSandbox, Reviewer: "Pending test", Assess: assessor}); err == nil || !strings.Contains(err.Error(), "pending contracts") {
				t.Fatalf("approval admitted pending: %v", err)
			}
			for _, dry := range []bool{true, false} {
				_, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approval, DryRun: dry, Now: fixedNow(), Assess: assessor, Invoke: func(context.Context, udonrunner.Invocation) error {
					t.Fatal("pending package dispatched an executor")
					return nil
				}})
				if err == nil || !strings.Contains(err.Error(), "pending contracts") {
					t.Fatalf("run dry=%v admitted pending: %v", dry, err)
				}
			}
			if !reflect.DeepEqual(before, snapshotTree(t, root)) {
				t.Fatal("refusal changed package or created run artifacts")
			}
		})
	}
}
