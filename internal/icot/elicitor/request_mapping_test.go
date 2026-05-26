package elicitor

import (
	"encoding/json"
	"testing"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBuildRequestMappingRequestUsesSelectedOperationMetadata(t *testing.T) {
	session := requestMappingSession()
	docs := requestMappingDocs()
	request := BuildRequestMappingRequest("fetch a support ticket", session, docs, []ReadinessIssue{{
		Code: "missing_required_request_values",
		Slot: "steps.get_ticket.with",
	}}, QuestionPlan{Prompt: "map fields", Slots: []string{"steps.get_ticket.with"}})

	if len(request.Steps) != 1 {
		t.Fatalf("steps = %d, want 1", len(request.Steps))
	}
	step := request.Steps[0]
	if step.OperationID != "getTicket" || step.Operation.OperationID != "getTicket" {
		t.Fatalf("operation context = %#v", step)
	}
	if len(step.MissingFields) != 1 || step.MissingFields[0] != "ticketId" {
		t.Fatalf("missing fields = %#v", step.MissingFields)
	}
	if len(step.Operation.Parameters) != 1 || step.Operation.Parameters[0].Name != "ticketId" {
		t.Fatalf("parameters = %#v", step.Operation.Parameters)
	}
	if len(step.KnownFields) != 1 || step.KnownFields[0] != "path.ticketId" {
		t.Fatalf("known fields = %#v", step.KnownFields)
	}
}

func TestApplyRequestMappingResponseRejectsInventedFields(t *testing.T) {
	session := requestMappingSession()
	docs := requestMappingDocs()
	request := BuildRequestMappingRequest("fetch a support ticket", session, docs, nil, QuestionPlan{Slots: []string{"steps.get_ticket.with"}})

	applied := applyRequestMappingResponse(&session, request, RequestMappingResponse{Steps: []RequestMappingStepResponse{{
		Name: "get_ticket",
		With: map[string]string{
			"ticketId": "inputs.ticketId",
			"other":    "inputs.other",
		},
	}, {
		Name: "invented",
		With: map[string]string{"ticketId": "inputs.nope"},
	}}})

	if applied.Applied != 1 {
		t.Fatalf("applied = %d, want 1", applied.Applied)
	}
	if got := session.Intent.Steps[0].With["ticketId"]; got != "inputs.ticketId" {
		t.Fatalf("ticketId mapping = %q", got)
	}
	if _, ok := session.Intent.Steps[0].With["other"]; ok {
		t.Fatalf("invented field copied into step: %#v", session.Intent.Steps[0].With)
	}
	if len(applied.Rejected) != 2 {
		t.Fatalf("rejected = %#v, want invented field and step", applied.Rejected)
	}
	if len(session.Intent.Inputs) != 1 || session.Intent.Inputs[0].Name != "ticketId" {
		t.Fatalf("inputs = %#v", session.Intent.Inputs)
	}
}

func TestApplyRequestMappingResponseAcceptsQualifiedFieldAlias(t *testing.T) {
	session := requestMappingSession()
	docs := requestMappingDocs()
	request := BuildRequestMappingRequest("fetch a support ticket", session, docs, nil, QuestionPlan{Slots: []string{"steps.get_ticket.with"}})

	applied := applyRequestMappingResponse(&session, request, RequestMappingResponse{Steps: []RequestMappingStepResponse{{
		Name: "get_ticket",
		With: map[string]string{"path.ticketId": "inputs.ticketId"},
	}}})

	if applied.Applied != 1 {
		t.Fatalf("applied = %d, rejected=%#v", applied.Applied, applied.Rejected)
	}
	if got := session.Intent.Steps[0].With["ticketId"]; got != "inputs.ticketId" {
		t.Fatalf("ticketId mapping = %q", got)
	}
}

func TestRequestMappingStepResponseToleratesNumericValues(t *testing.T) {
	var response RequestMappingResponse
	if err := json.Unmarshal([]byte(`{"steps":[{"name":"weather","with":{"lat":43.6532,"lon":-79.3832,"units":"metric"}}]}`), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	got := response.Steps[0].With
	if got["lat"] != "43.6532" || got["lon"] != "-79.3832" || got["units"] != "metric" {
		t.Fatalf("with = %#v", got)
	}
}

func TestRequestMappingStepResponseToleratesListValues(t *testing.T) {
	var response RequestMappingResponse
	raw := `{"steps":[{"name":"weather","with":[{"field":"lat","value":43.6532},{"field":"lon","source":"-79.3832"}]}]}`
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	got := response.Steps[0].With
	if got["lat"] != "43.6532" || got["lon"] != "-79.3832" {
		t.Fatalf("with = %#v", got)
	}
}

func TestRequestMappingResponseToleratesKeyedSteps(t *testing.T) {
	var response RequestMappingResponse
	raw := `{"steps":{"weather":{"with":{"lat":43.6532}},"direct":{"lon":-79.3832}}}`
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	got := map[string]map[string]string{}
	for _, step := range response.Steps {
		got[step.Name] = step.With
	}
	if got["weather"]["lat"] != "43.6532" || got["direct"]["lon"] != "-79.3832" {
		t.Fatalf("steps = %#v", response.Steps)
	}
}

func requestMappingSession() Session {
	return Session{Intent: rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Description: "fetch a support ticket"},
		Steps: []*rollout.Step{{
			Name:      "get_ticket",
			Type:      "http",
			OpenAPI:   "openapi/support.yaml",
			Operation: "getTicket",
			With:      map[string]string{},
		}},
	}}
}

func requestMappingDocs() []APIDocument {
	return []APIDocument{{
		RelativePath: "openapi/support.yaml",
		Title:        "Support",
		Operations: []apitools.OperationSummary{{
			OperationID: "getTicket",
			Method:      "GET",
			Path:        "/tickets/{ticketId}",
			Parameters: []apitools.ParameterSummary{{
				Name:     "ticketId",
				In:       "path",
				Required: true,
				Type:     "string",
			}},
		}},
	}}
}

func TestBuildRequestMappingRequestUsesSourceRef(t *testing.T) {
	session := Session{Intent: rollout.Intent{
		Source: "google-discovery/gmail.json",
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/gmail.json",
			Operation: "gmail_users_messages_send",
			With:      map[string]string{},
		}},
	}}
	docs := []APIDocument{{
		RelativePath: "google-discovery/gmail.json",
		Operations: []apitools.OperationSummary{{
			OperationID: "gmail_users_messages_send",
			Parameters: []apitools.ParameterSummary{{
				Name:     "userId",
				In:       "path",
				Required: true,
			}},
		}},
	}}
	request := BuildRequestMappingRequest("", session, docs, nil, QuestionPlan{})
	if len(request.Steps) != 1 {
		t.Fatalf("steps = %#v", request.Steps)
	}
	if request.Steps[0].Source != "google-discovery/gmail.json" {
		t.Fatalf("source = %q", request.Steps[0].Source)
	}
	if request.Steps[0].OpenAPI != "" {
		t.Fatalf("openapi alias should be empty for native source, got %q", request.Steps[0].OpenAPI)
	}
}
