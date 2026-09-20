package synthesize

import (
	"strings"
	"testing"

	"github.com/OpenUdon/uws/uws1"
)

func TestNormalizeUWSStepListRejectsOperationRefWithNestedBlocks(t *testing.T) {
	steps := []*uws1.Step{{
		StepID:       "call_and_init",
		OperationRef: "adv_list",
		Steps:        []*uws1.Step{{StepID: "nested", OperationRef: "adv_login"}},
	}}
	err := normalizeUWSStepList(steps, map[string]bool{"adv_list": true, "adv_login": true})
	if err == nil {
		t.Fatal("expected an error for an operation-reference step with nested steps")
	}
	if !strings.Contains(err.Error(), "call_and_init") {
		t.Fatalf("error should name the offending step: %v", err)
	}
}

func TestNormalizeUWSStepListRejectsWorkflowRefWithNestedCases(t *testing.T) {
	steps := []*uws1.Step{{
		StepID:              "delegate",
		StepExecutionFields: uws1.StepExecutionFields{Workflow: "child_workflow"},
		Cases: []*uws1.Case{{
			CaseFields: uws1.CaseFields{Name: "only"},
			Steps:      []*uws1.Step{{StepID: "nested", OperationRef: "op"}},
		}},
	}}
	err := normalizeUWSStepList(steps, map[string]bool{"op": true})
	if err == nil {
		t.Fatal("expected an error for a workflow-reference step with nested cases")
	}
	if !strings.Contains(err.Error(), "delegate") {
		t.Fatalf("error should name the offending step: %v", err)
	}
}

func TestNormalizeUWSStepListAcceptsOrdinaryReferenceAndStructuralSteps(t *testing.T) {
	steps := []*uws1.Step{
		{StepID: "adv_login", OperationRef: "adv_login"},
		{
			StepID: "adv_list",
			Type:   uws1.WorkflowTypeSequence,
			Steps: []*uws1.Step{
				{StepID: "adv_list_call", OperationRef: "adv_list"},
			},
		},
	}
	operationIDs := map[string]bool{"adv_login": true, "adv_list": true}
	if err := normalizeUWSStepList(steps, operationIDs); err != nil {
		t.Fatalf("ordinary reference and structural steps should normalize cleanly: %v", err)
	}
}

func TestNormalizeUWSStepsForSchemaPropagatesNestedBlockError(t *testing.T) {
	doc := &uws1.Document{
		Operations: []*uws1.Operation{{OperationID: "adv_list"}, {OperationID: "adv_login"}},
		Workflows: []*uws1.Workflow{{
			WorkflowID: "main",
			Type:       uws1.WorkflowTypeSequence,
			Steps: []*uws1.Step{{
				StepID:       "adv_list",
				OperationRef: "adv_list",
				Default:      []*uws1.Step{{StepID: "adv_login", OperationRef: "adv_login"}},
			}},
		}},
	}
	if err := normalizeUWSStepsForSchema(doc); err == nil {
		t.Fatal("expected normalizeUWSStepsForSchema to surface the nested-block error")
	}
}
