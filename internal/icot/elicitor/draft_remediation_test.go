package elicitor

import (
	"os"
	"path/filepath"
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
	if changed || len(rejected) != 1 || !strings.Contains(rejected[0], "no safe read-only producer operation") {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
}

func TestApplyDraftReviewRemediationsAddsSafeAPIPrework(t *testing.T) {
	root := t.TempDir()
	openAPIPath := filepath.Join(root, "support.yaml")
	if err := os.WriteFile(openAPIPath, []byte(`openapi: 3.0.0
paths:
  /customers/current:
    get:
      operationId: getCurrentCustomer
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  customerId:
                    type: string
  /tickets:
    post:
      operationId: updateTicket
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
`), 0o644); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{{
		Name:      "update_ticket",
		Type:      "http",
		Source:    "openapi/support.yaml",
		OpenAPI:   "openapi/support.yaml",
		Operation: "updateTicket",
	}}}}
	docs := []APIDocument{{
		Path:         openAPIPath,
		RelativePath: "openapi/support.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "getCurrentCustomer",
			Method:      "GET",
			Path:        "/customers/current",
		}, {
			OperationID: "updateTicket",
			Method:      "POST",
			Path:        "/tickets",
		}},
	}}

	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "missing_lookup",
		Message:           "Resolve the customer id with an API lookup before update.",
		Slot:              "steps.update_ticket.with.customerId",
		GapKind:           flowGapMissingAPIPrework,
		RemediationAction: remediationProposeAPIPrework,
	}}, docs)
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	if len(session.Intent.Steps) != 2 || session.Intent.Steps[0].Operation != "getCurrentCustomer" {
		t.Fatalf("steps = %#v", session.Intent.Steps)
	}
	update := session.Intent.Steps[1]
	if update.With["customerId"] != "lookup_customerId.received_body.customerId" || len(update.DependsOn) != 1 || update.DependsOn[0] != "lookup_customerId" {
		t.Fatalf("update step = %#v", update)
	}
}

func TestApplyDraftReviewRemediationsRejectsCredentialedAPIPrework(t *testing.T) {
	root := t.TempDir()
	openAPIPath := filepath.Join(root, "support.yaml")
	if err := os.WriteFile(openAPIPath, []byte(`openapi: 3.0.0
paths:
  /customers/current:
    get:
      operationId: getCurrentCustomer
      responses:
        "200":
          content:
            application/json:
              schema:
                type: object
                properties:
                  customerId:
                    type: string
`), 0o644); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{{
		Name:      "update_ticket",
		Type:      "http",
		Source:    "openapi/support.yaml",
		OpenAPI:   "openapi/support.yaml",
		Operation: "updateTicket",
	}}}}
	docs := []APIDocument{{
		Path:         openAPIPath,
		RelativePath: "openapi/support.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "getCurrentCustomer",
			Method:      "GET",
			Path:        "/customers/current",
			Security: []apitools.SecuritySummary{{
				Name: "support_api_key",
				Type: "apiKey",
				In:   "header",
			}},
		}},
	}}

	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "missing_lookup",
		Message:           "Resolve the customer id with an API lookup before update.",
		Slot:              "steps.update_ticket.with.customerId",
		GapKind:           flowGapMissingAPIPrework,
		RemediationAction: remediationProposeAPIPrework,
	}}, docs)
	if changed || len(rejected) != 1 || !strings.Contains(rejected[0], "no safe read-only producer operation") {
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

func TestApplyDraftReviewRemediationsUsesNestedRefResponseFields(t *testing.T) {
	root := t.TempDir()
	openAPIPath := filepath.Join(root, "pagerduty.yaml")
	if err := os.WriteFile(openAPIPath, []byte(`openapi: 3.0.0
paths:
  /users/current:
    get:
      operationId: getUser
      responses:
        "200":
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/UserEnvelope'
components:
  schemas:
    UserEnvelope:
      type: object
      properties:
        user:
          type: object
          properties:
            id:
              type: string
            email:
              type: string
            teams:
              type: array
              items:
                type: object
                properties:
                  name:
                    type: string
`), 0o644); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
	session := Session{
		Project: projectwizard.Answers{Goal: "Fetch one user and render a contact card."},
		Intent: rollout.Intent{
			Source:   "openapi/pagerduty.yaml",
			Workflow: &rollout.WorkflowMeta{Name: "contact_card", Description: "Fetch one user and render a contact card."},
			Steps: []*rollout.Step{{
				Name:      "get_user",
				Type:      "http",
				Source:    "openapi/pagerduty.yaml",
				Operation: "getUser",
			}},
			Outputs: []*rollout.Output{{Name: "result", From: "get_user.received_body"}},
		},
	}
	docs := []APIDocument{{
		Path:         openAPIPath,
		RelativePath: "openapi/pagerduty.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "getUser",
			Method:      "GET",
			Path:        "/users/current",
		}},
	}}
	changed, rejected := applyDraftReviewRemediations(&session, []DraftReviewIssue{{
		Code:              "output_transport_response",
		Message:           "The output returns the raw transport response instead of the requested contact card.",
		Slot:              "intent.outputs.result",
		GapKind:           flowGapMissingTransformStep,
		RemediationAction: remediationProposeFnctStep,
	}}, docs)
	if !changed || len(rejected) != 0 {
		t.Fatalf("changed=%v rejected=%v", changed, rejected)
	}
	fnct := session.Intent.Steps[1]
	if len(fnct.Binds) != 1 || fnct.Binds[0].Fields["user_id"] != "received_body.user.id" || fnct.Binds[0].Fields["user_email"] != "received_body.user.email" || fnct.Binds[0].Fields["user_teams_name"] != "received_body.user.teams.name" {
		t.Fatalf("fnct binds = %#v", fnct.Binds)
	}
}
