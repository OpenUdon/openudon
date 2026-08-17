package synthesize

import (
	"testing"

	"github.com/OpenUdon/uws/uws1"
)

func TestWireBrowserScenarioOutputsUsesUWSStepContext(t *testing.T) {
	document := &uws1.Document{
		Operations: []*uws1.Operation{{OperationID: "authenticate"}, {OperationID: "read"}},
		Workflows: []*uws1.Workflow{{
			WorkflowID: "main",
			Steps: []*uws1.Step{
				{StepID: "authenticate", OperationRef: "authenticate"},
				{StepID: "read", OperationRef: "read"},
			},
		}},
	}

	if err := wireBrowserScenarioOutputs(document); err != nil {
		t.Fatal(err)
	}
	if got := document.Operations[1].Outputs["received_body"]; got != "$response.body" {
		t.Fatalf("operation output = %q", got)
	}
	if got := document.Workflows[0].Steps[1].Outputs["received_body"]; got != "$response.body" {
		t.Fatalf("step output = %q", got)
	}
	if got := document.Workflows[0].Outputs["scenario_result"]; got != "$steps.read.outputs.received_body" {
		t.Fatalf("workflow output = %q", got)
	}
}

func TestWireBrowserScenarioOutputsRejectsMissingReadStep(t *testing.T) {
	document := &uws1.Document{
		Operations: []*uws1.Operation{{OperationID: "read"}},
		Workflows:  []*uws1.Workflow{{WorkflowID: "main"}},
	}
	if err := wireBrowserScenarioOutputs(document); err == nil {
		t.Fatal("missing read step was accepted")
	}
}
