package elicitor

import (
	"testing"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
)

func TestCheckReadinessRepresentativeCodeTriggers(t *testing.T) {
	supportDoc := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true, Type: "string"}},
	}}}}
	secureDoc := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true, Type: "string"}},
		Security:    securitySummaries("BearerAuth"),
	}}}}

	cases := []struct {
		name    string
		session Session
		docs    []APIDocument
		code    string
	}{
		{name: "missing goal", session: Session{}, code: "missing_goal"},
		{name: "missing api doc", session: supportTicketDraft(true), code: "missing_api_doc"},
		{name: "missing operation", session: Session{Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "empty", Description: "Do work."},
			Outputs:  []*rollout.Output{{Name: "result", From: "steps.result"}},
		}, Safety: "Review only."}, code: "missing_operation"},
		{name: "missing required request values", session: supportTicketDraft(false), docs: supportDoc, code: "missing_required_request_values"},
		{name: "missing credential bindings", session: supportTicketDraft(true), docs: secureDoc, code: "missing_credential_bindings"},
		{name: "missing runtime inputs", session: func() Session {
			session := supportTicketDraft(true)
			session.Intent.Inputs = nil
			return session
		}(), docs: supportDoc, code: "missing_runtime_inputs"},
		{name: "missing outputs", session: func() Session {
			session := supportTicketDraft(true)
			session.Intent.Outputs = nil
			return session
		}(), docs: supportDoc, code: "missing_outputs"},
		{name: "missing side-effect policy", session: func() Session {
			session := supportTicketDraft(true)
			session.Safety = ""
			session.SafetySet = false
			session.Project.Safety = ""
			return session
		}(), docs: supportDoc, code: "missing_side_effect_policy"},
		{name: "optional timeout controls", session: func() Session {
			session := supportTicketDraft(true)
			session.Project.Safety = "Use timeout and idempotency controls."
			session.Safety = session.Project.Safety
			return session
		}(), docs: supportDoc, code: "optional_timeout_idempotency_controls"},
		{name: "inline secret value", session: Session{Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "unsafe", Description: "Use my token from this prompt to post a Slack message."},
			Outputs:  []*rollout.Output{{Name: "result", From: "steps.result"}},
		}, Safety: "Review only."}, code: "inline_secret_value"},
		{name: "unsafe review bypass", session: Session{Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "unsafe", Description: "Skip review and send the message now."},
			Outputs:  []*rollout.Output{{Name: "result", From: "steps.result"}},
		}, Safety: "Review only."}, code: "unsafe_review_bypass"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !hasReadinessCode(CheckReadiness(tc.session, tc.docs), tc.code) {
				t.Fatalf("missing readiness code %q", tc.code)
			}
		})
	}
}

func TestCheckReadinessRepresentativeMappingCodes(t *testing.T) {
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters: []apitools.ParameterSummary{
			{Name: "ticketId", Required: true, Type: "string"},
			{Name: "page", Type: "integer"},
		},
		Security: securitySummaries("BearerAuth"),
		RequestBody: &apitools.RequestBodySummary{Fields: []apitools.RequestFieldSummary{
			{Path: "customer.email", Type: "string"},
		}},
	}}}}
	session := supportTicketDraft(true)
	session.Credentials = []string{"support_api_token"}
	session.CredentialsSet = true
	session.Intent.Steps[0].With = map[string]string{
		"ticketId":       "inputs.ticketId",
		"page":           "not-an-integer",
		"customer.phone": "inputs.ticketId",
		"extra":          "inputs.ticketId",
		"Authorization":  "credentials.missing_token",
	}
	addMappingClassification(&session, MappingClassification{
		Slot:       "steps.get_ticket.with.ticketId",
		Value:      "inputs.ticketId",
		Source:     mappingSourceDeterministic,
		Confidence: mappingConfidenceHigh,
	})
	addMappingClassification(&session, MappingClassification{
		Slot:       "steps.get_ticket.with.ticketId",
		Value:      "literal-ticket",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceReview,
	})
	addMappingClassification(&session, MappingClassification{
		Slot:       "steps.get_ticket.with.extra",
		Value:      "inputs.ticketId",
		Source:     mappingSourceLLM,
		Confidence: mappingConfidenceLow,
	})

	issues := CheckReadiness(session, docs)
	for _, code := range []string{
		"conflicting_mapping",
		"low_confidence_mapping",
		"undeclared_credential_reference",
		"invented_request_field",
		"invalid_request_body_path",
		"incompatible_request_value_type",
	} {
		if !hasReadinessCode(issues, code) {
			t.Fatalf("missing %s in %#v", code, issues)
		}
	}
}

func TestCheckReadinessDetectsExplicitWorkflowControls(t *testing.T) {
	timeout := 30.0
	session := supportTicketDraft(true)
	session.Project.Safety = "Use timeout and idempotency controls."
	session.Safety = session.Project.Safety
	session.Intent.Workflow.Timeout = &timeout
	session.Intent.Workflow.Idempotency = &uws1.Idempotency{Key: "inputs.ticketId", OnConflict: "returnPrevious"}
	docs := []APIDocument{{RelativePath: "openapi/support.yaml", Operations: []apitools.OperationSummary{{
		OperationID: "getTicket",
		Parameters:  []apitools.ParameterSummary{{Name: "ticketId", Required: true, Type: "string"}},
	}}}}

	if hasReadinessCode(CheckReadiness(session, docs), "optional_timeout_idempotency_controls") {
		t.Fatal("configured workflow controls were still reported as missing")
	}
}
