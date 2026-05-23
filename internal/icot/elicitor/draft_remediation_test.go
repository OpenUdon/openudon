package elicitor

import (
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestApplyDraftReviewRemediationsAddsSafeFnctStep(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{
			Goal: "Fetch the audit event and render an audit receipt.",
		},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "audit_receipt", Description: "Fetch the audit event and render an audit receipt."},
			Steps: []*rollout.Step{{
				Name:      "get_audit_event",
				Type:      "http",
				Operation: "getAuditEvent",
			}},
			Outputs: []*rollout.Output{{Name: "result", From: "get_audit_event.received_body"}},
		},
	}
	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "output_transport_response",
		Message:           "The output returns the raw transport response instead of the requested audit receipt.",
		Slot:              "intent.outputs.result",
		Evidence:          "result=get_audit_event.received_body",
		GapKind:           flowGapMissingTransformStep,
		RemediationAction: remediationProposeFnctStep,
	}})
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if len(session.Intent.Steps) != 2 || session.Intent.Steps[1].Name != "render_audit_receipt" {
		t.Fatalf("steps = %#v", session.Intent.Steps)
	}
	fnct := session.Intent.Steps[1]
	if fnct.Type != "fnct" || fnct.With["input"] != "get_audit_event.received_body" {
		t.Fatalf("fnct = %#v", fnct)
	}
	if got := session.Intent.Outputs[0].From; got != "render_audit_receipt.received_body" {
		t.Fatalf("output = %q", got)
	}
	if got := session.Intent.Outputs[0].Name; got != "audit_receipt" {
		t.Fatalf("output name = %q", got)
	}
	if !strings.Contains(session.Project.FunctionContracts, "render_audit_receipt") {
		session.Normalize()
		if !strings.Contains(session.Project.FunctionContracts, "render_audit_receipt") {
			t.Fatalf("function contracts not normalized: %q", session.Project.FunctionContracts)
		}
	}
}

func TestApplyDraftReviewRemediationsAsksOnMultipleFnctProducers(t *testing.T) {
	session := Session{
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "report", Description: "Render a report from selected data."},
			Steps: []*rollout.Step{
				{Name: "get_account", Type: "http", Operation: "getAccount"},
				{Name: "get_order", Type: "http", Operation: "getOrder"},
				{Name: "send_report", Type: "http", Operation: "sendMessage"},
			},
		},
	}
	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "missing_report",
		Message:           "The send step needs a rendered report.",
		Slot:              "steps.send_report.with.body",
		GapKind:           flowGapMissingTransformStep,
		RemediationAction: remediationProposeFnctStep,
	}})
	if changed || len(rejected) != 1 || !strings.Contains(rejected[0], "ambiguous producer step") {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if len(session.Intent.Steps) != 3 {
		t.Fatalf("unexpected steps: %#v", session.Intent.Steps)
	}
}

func TestApplyDraftReviewRemediationsUsesTerminalProducerInLinearChain(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "Fetch coordinates, fetch weather, and email a report."},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather_email", Description: "Fetch coordinates, fetch weather, and email a report."},
			Steps: []*rollout.Step{
				{Name: "geocode", Type: "http", Operation: "geocode"},
				{Name: "get_weather", Type: "http", Operation: "getWeather", DependsOn: []string{"geocode"}},
				{Name: "send_email", Type: "http", Operation: "sendEmail", With: map[string]string{"userId": "me"}},
			},
		},
	}
	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "missing_rendered_body",
		Message:           "The send step needs a rendered email message.",
		Slot:              "steps.send_email.with.raw",
		GapKind:           flowGapMissingTransformStep,
		RemediationAction: remediationProposeFnctStep,
	}})
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	fnct := stepByName(session.Intent.Steps, "render_message")
	if fnct == nil || len(fnct.DependsOn) != 1 || fnct.DependsOn[0] != "get_weather" {
		t.Fatalf("render fnct = %#v", fnct)
	}
	send := stepByName(session.Intent.Steps, "send_email")
	if got := send.With["raw"]; got != "render_message.received_body" {
		t.Fatalf("send raw = %q", got)
	}
}

func TestApplyDraftReviewRemediationsRejectsAPIPrework(t *testing.T) {
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{{
		Name:      "update_customer",
		Type:      "http",
		Operation: "updateCustomer",
	}}}}
	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "missing_lookup",
		Message:           "Resolve the customer id with an API lookup before update.",
		Slot:              "steps.update_customer.with.customerId",
		GapKind:           flowGapMissingAPIPrework,
		RemediationAction: remediationProposeAPIPrework,
	}})
	if changed || len(rejected) != 1 || !strings.Contains(rejected[0], "api prework remediation is not implemented") {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
}

func TestApplyDraftReviewRemediationsRejectsGenericDoOnlyProducer(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "Render a report from selected data."},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "report", Description: "Render a report from selected data."},
			Steps: []*rollout.Step{
				{Name: "describe_goal", Type: "cmd", Do: "Describe what should happen."},
				{Name: "send_report", Type: "http", Operation: "sendMessage"},
			},
		},
	}
	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "missing_report",
		Message:           "The send step needs a rendered report.",
		Slot:              "steps.send_report.with.body",
		GapKind:           flowGapMissingTransformStep,
		RemediationAction: remediationProposeFnctStep,
	}})
	if changed || len(rejected) != 1 || !strings.Contains(rejected[0], "no single producer step") {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if len(session.Intent.Steps) != 2 {
		t.Fatalf("unexpected steps: %#v", session.Intent.Steps)
	}
}

func TestApplyDraftReviewRemediationsUsesResponseFieldsWhenAvailable(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{
			Goal: "Send a Gmail message and render an audit receipt.",
		},
		Intent: rollout.Intent{
			OpenAPI:  "openapi/gmail.yaml",
			Workflow: &rollout.WorkflowMeta{Name: "audit_receipt", Description: "Send a Gmail message and render an audit receipt."},
			Steps: []*rollout.Step{{
				Name:      "send_message",
				Type:      "http",
				OpenAPI:   "openapi/gmail.yaml",
				Operation: "sendMessage",
			}},
			Outputs: []*rollout.Output{{Name: "result", From: "send_message.received_body"}},
		},
	}
	docs := []APIDocument{{
		Path:         "../../../examples/eval/m28-gmail-audit-receipt/openapi/gmail.yaml",
		RelativePath: "openapi/gmail.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "sendMessage",
			Method:      "POST",
			Path:        "/messages/send",
		}},
	}}
	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "output_transport_response",
		Message:           "The output returns the raw transport response instead of the requested audit receipt.",
		Slot:              "intent.outputs.result",
		Evidence:          "result=send_message.received_body",
		GapKind:           flowGapMissingTransformStep,
		RemediationAction: remediationProposeFnctStep,
	}}, docs)
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	fnct := session.Intent.Steps[1]
	if len(fnct.With) != 0 {
		t.Fatalf("response metadata should avoid generic input binding: %#v", fnct.With)
	}
	if len(fnct.Binds) != 1 || fnct.Binds[0].Fields["id"] != "received_body.id" || fnct.Binds[0].Fields["threadId"] != "received_body.threadId" {
		t.Fatalf("fnct binds = %#v", fnct.Binds)
	}
	if got := session.Intent.Outputs[0].Name; got != "audit_receipt" {
		t.Fatalf("output name = %q", got)
	}
}
