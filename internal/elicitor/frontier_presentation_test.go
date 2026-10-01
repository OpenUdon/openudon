package elicitor

import (
	"reflect"
	"testing"

	"github.com/OpenUdon/apitools"
	publicinterview "github.com/OpenUdon/authoring/interview"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBuildQuestionControlsOwnsChoicesSyntaxAndDeferral(t *testing.T) {
	session := Session{
		Boundary:           WorkflowBoundary{Outcome: "Current workflow"},
		CandidateWorkflows: []CandidateWorkflow{{Title: "Alternate", Outcome: "Render another report"}},
		Interview: publicinterview.State{Nodes: []publicinterview.Node{
			{ID: nodeActorTrigger}, {ID: nodeRemoteLookup}, {ID: nodeSideEffectPosture}, {ID: nodeFallback, Deferrable: true}, {ID: nodeActiveWorkflow},
		}},
	}
	frontier := []QuestionPlan{
		{ID: nodeActorTrigger, Slots: []string{"boundary.actor", "boundary.trigger"}},
		{ID: nodeRemoteLookup, Slots: []string{"source.remote_lookup"}},
		{ID: nodeSideEffectPosture, Slots: []string{"safety"}, Recommendation: projectwizard.SideEffectReadOnly},
		{ID: nodeFallback, Slots: []string{"fallback"}},
		{ID: nodeActiveWorkflow, Slots: []string{"boundary.active_workflow"}},
	}
	controls := BuildQuestionControls(session, nil, frontier)
	if len(controls) != len(frontier) {
		t.Fatalf("controls = %#v", controls)
	}
	if controls[0].InputKind != QuestionInputText || controls[0].Syntax == "" {
		t.Fatalf("actor control = %#v", controls[0])
	}
	if controls[1].InputKind != QuestionInputChoice || !reflect.DeepEqual(controls[1].Options, []QuestionOption{{Value: "never", Label: "never"}, {Value: "allow", Label: "allow"}}) {
		t.Fatalf("remote control = %#v", controls[1])
	}
	wantPolicy := []QuestionOption{
		{Value: projectwizard.SideEffectReadOnly, Label: projectwizard.SideEffectReadOnly},
		{Value: projectwizard.SideEffectSandboxOnly, Label: projectwizard.SideEffectSandboxOnly},
		{Value: projectwizard.SideEffectAfterApproval, Label: projectwizard.SideEffectAfterApproval},
	}
	if !reflect.DeepEqual(controls[2].Options, wantPolicy) {
		t.Fatalf("side-effect control = %#v", controls[2])
	}
	if !controls[3].Deferrable || controls[3].InputKind != QuestionInputText {
		t.Fatalf("deferrable control = %#v", controls[3])
	}
	wantWorkflows := []QuestionOption{{Value: "Current workflow", Label: "Current workflow"}, {Value: "Alternate: Render another report", Label: "Alternate: Render another report"}}
	if !reflect.DeepEqual(controls[4].Options, wantWorkflows) {
		t.Fatalf("workflow control = %#v", controls[4])
	}
}

func TestBuildQuestionControlsUsesStableSecurityAlternativeValues(t *testing.T) {
	step := &rollout.Step{Name: "call_api", Type: "http", Operation: "callAPI"}
	session := Session{Intent: rollout.Intent{Source: "openapi/api.yaml", Steps: []*rollout.Step{step}}}
	docs := []APIDocument{{RelativePath: "openapi/api.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "callAPI",
		SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
			{Requirements: []apitools.SecuritySummary{{Name: "bearer"}}},
			{Requirements: []apitools.SecuritySummary{{Name: "api_key"}, {Name: "client_certificate"}}},
		},
	}}}}
	question := QuestionPlan{ID: "security.alternative.call_api", Slots: []string{securityAlternativeSlot(step)}}
	controls := BuildQuestionControls(session, docs, []QuestionPlan{question})
	want := []QuestionOption{{Value: "1", Label: "1 — bearer"}, {Value: "2", Label: "2 — api_key + client_certificate"}}
	if len(controls) != 1 || controls[0].InputKind != QuestionInputChoice || !reflect.DeepEqual(controls[0].Options, want) {
		t.Fatalf("security control = %#v", controls)
	}
}
