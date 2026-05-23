package elicitor

import (
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestApplyDraftReviewRepairsAllowsRequestMapping(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{{
		Name: "gmail",
		Type: "http",
		With: map[string]string{"raw": "gmail.received_body"},
	}}}}
	changed, rejected := applyDraftReviewRepairs(&session, []DraftReviewIssue{{
		Code:            "disconnected_report",
		Message:         "Gmail raw should consume the rendered report.",
		Slot:            "steps.gmail.with.raw",
		SuggestedAnswer: "raw=render_report.received_body.raw",
	}})
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if got := session.Intent.Steps[0].With["raw"]; got != "render_report.received_body.raw" {
		t.Fatalf("raw = %q", got)
	}
	if len(session.DecisionEvidence) == 0 {
		t.Fatalf("missing repair decision evidence")
	}
}

func TestApplyDraftReviewRepairsRejectsOperationMutation(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{{
		Name:      "gmail",
		Type:      "http",
		Operation: "gmail_users_messages_send",
	}}}}
	changed, rejected := applyDraftReviewRepairs(&session, []DraftReviewIssue{{
		Code:            "wrong_operation",
		Message:         "Operation should change.",
		Slot:            "steps.gmail.operation",
		SuggestedAnswer: "gmail_users_messages_get",
	}})
	if changed || len(rejected) == 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if got := session.Intent.Steps[0].Operation; got != "gmail_users_messages_send" {
		t.Fatalf("operation changed to %q", got)
	}
}

func TestApplyDraftReviewRepairsAllowsDependsOnOnlyForKnownStep(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{
		{Name: "render_report", Type: "fnct"},
		{Name: "gmail", Type: "http"},
	}}}
	changed, rejected := applyDraftReviewRepairs(&session, []DraftReviewIssue{{
		Code:            "missing_dependency",
		Message:         "Gmail should wait for report rendering.",
		Slot:            "steps.gmail.depends_on",
		SuggestedAnswer: "depends_on=render_report",
	}})
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if got := session.Intent.Steps[1].DependsOn; len(got) != 1 || got[0] != "render_report" {
		t.Fatalf("depends_on = %#v", got)
	}
}

func TestApplyDraftReviewRepairsAllowsIndexedOutputFromSlot(t *testing.T) {
	session := Session{Intent: rollout.Intent{Outputs: []*rollout.Output{{
		Name: "result",
		From: "gmail.received_body",
	}}}}
	changed, rejected := applyDraftReviewRepairs(&session, []DraftReviewIssue{{
		Code:            "output_transport_response",
		Message:         "The output should return the rendered report.",
		Slot:            "outputs[0].from",
		SuggestedAnswer: "from=render_report.received_body",
	}})
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if got := session.Intent.Outputs[0].From; got != "render_report.received_body" {
		t.Fatalf("output from = %q", got)
	}
}

func TestApplyDraftReviewRepairsRejectsMissingSuggestedAnswer(t *testing.T) {
	session := Session{Intent: rollout.Intent{Outputs: []*rollout.Output{{
		Name: "result",
		From: "gmail.received_body",
	}}}}
	changed, rejected := applyDraftReviewRepairs(&session, []DraftReviewIssue{{
		Code:    "output_transport_response",
		Message: "The output should return the rendered report.",
		Slot:    "intent.outputs.result",
	}})
	if changed || len(rejected) != 1 || !strings.Contains(rejected[0], "missing suggested answer") {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if got := session.Intent.Outputs[0].From; got != "gmail.received_body" {
		t.Fatalf("output changed to %q", got)
	}
}
