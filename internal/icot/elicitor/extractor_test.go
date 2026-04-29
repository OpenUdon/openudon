package elicitor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/genelet/udon/pkg/rollout"
)

type fakeChat struct {
	response string
}

func (f fakeChat) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return f.response, nil
}

func (f fakeChat) StructuredChat(ctx context.Context, messages []rollout.ChatMessage, _ json.RawMessage, _ rollout.StructuredOpts) (string, error) {
	return f.Chat(ctx, messages)
}

func TestKickoffStripsInventedOpenAPIPaths(t *testing.T) {
	extractor := NewChatExtractor(fakeChat{response: `{
  "intent": {
    "openapi": "openapi/invented.yaml",
    "workflow": {"name": "support_lookup", "description": "Fetch a ticket"},
    "inputs": [{"name": "ticket_id"}]
  },
  "side_effect_scope": "sandbox-only"
}`}, nil)
	session, err := extractor.Kickoff(context.Background(), "Fetch a ticket")
	if err != nil {
		t.Fatalf("Kickoff failed: %v", err)
	}
	if session.Intent.OpenAPI != "" || len(session.Intent.Inputs) != 0 {
		t.Fatalf("kickoff accepted invented structural fields: %#v", session.Intent)
	}
	if session.Intent.Workflow == nil || session.Intent.Workflow.Name != "support_lookup" {
		t.Fatalf("kickoff lost obvious workflow metadata: %#v", session.Intent.Workflow)
	}
	if len(session.Annotations) != 1 || session.Annotations[0].PromptVersion != PromptVersion {
		t.Fatalf("kickoff annotations missing prompt version: %#v", session.Annotations)
	}
}

func TestRefinePreservesStructuralFields(t *testing.T) {
	base := Session{
		Intent: rollout.Intent{
			OpenAPI:  "openapi/support.yaml",
			Workflow: &rollout.WorkflowMeta{Name: "support_lookup", Description: "Fetch ticket"},
			Inputs:   []*rollout.Input{{Name: "ticket_id", Type: "string", Required: true}},
			Steps:    []*rollout.Step{{Name: "get_ticket", Type: "http", Operation: "getTicket"}},
			Outputs:  []*rollout.Output{{Name: "ticket", From: "get_ticket.received_body"}},
		},
		Credentials:     []string{"support_api_token"},
		SideEffectScope: "sandbox-only",
	}
	extractor := NewChatExtractor(fakeChat{response: `{
  "intent": {
    "openapi": "openapi/changed.yaml",
    "workflow": {"name": "changed", "description": "Fetch the support ticket cleanly"}
  },
  "credentials": ["changed_token"],
  "fallback": "Stop cleanly"
}`}, nil)
	refined, err := extractor.Refine(context.Background(), base)
	if err != nil {
		t.Fatalf("Refine failed: %v", err)
	}
	if refined.Intent.OpenAPI != "openapi/support.yaml" || refined.Intent.Workflow.Name != "support_lookup" || refined.Credentials[0] != "support_api_token" {
		t.Fatalf("refine changed structural fields: %#v", refined)
	}
	if refined.Intent.Workflow.Description != "Fetch the support ticket cleanly" || refined.Fallback != "Stop cleanly" {
		t.Fatalf("refine did not update prose fields: %#v", refined)
	}
}

func TestDisambiguateFiltersInventedPaths(t *testing.T) {
	extractor := NewChatExtractor(fakeChat{response: `{"paths":["openapi/support.yaml","https://example.com/invented.yaml","openapi/missing.yaml"]}`}, nil)
	paths, err := extractor.Disambiguate(context.Background(), "support", []APIDocument{
		{RelativePath: "openapi/support.yaml"},
		{RelativePath: "openapi/billing.yaml"},
	})
	if err != nil {
		t.Fatalf("Disambiguate failed: %v", err)
	}
	if strings.Join(paths, ",") != "openapi/support.yaml" {
		t.Fatalf("paths = %#v", paths)
	}
}

func TestDraftPromptRequestIncludesStructuredParameters(t *testing.T) {
	op := promptOperation(t, DraftRequest{Docs: []APIDocument{{
		RelativePath: "openapi/support.yaml",
		Operations: []*rollout.OperationInfo{{
			OperationID: "getTicket",
			Method:      "GET",
			Path:        "/tickets/{ticketId}",
			Parameters: []*rollout.ParameterInfo{
				{Name: "ticketId", In: "path", Required: true, Type: "string", Description: "Ticket identifier"},
				{Name: "include", In: "query", Type: "string", Description: "Related resources"},
				{Name: "X-Trace-ID", In: "header", Type: "string", Description: "Trace header"},
			},
		}},
	}}})

	assertParameterContext(t, op.Parameters, "ticketId", "path", true, "string", "Ticket identifier")
	assertParameterContext(t, op.Parameters, "include", "query", false, "string", "Related resources")
	assertParameterContext(t, op.Parameters, "X-Trace-ID", "header", false, "string", "Trace header")
}

func TestDraftPromptRequestFlattensNestedRequestBody(t *testing.T) {
	op := promptOperation(t, DraftRequest{Docs: []APIDocument{{
		RelativePath: "openapi/orders.yaml",
		Operations: []*rollout.OperationInfo{{
			OperationID: "createOrder",
			Method:      "POST",
			Path:        "/orders",
			RequestBody: &rollout.RequestBodyInfo{
				Required:    true,
				ContentType: "application/json",
				Schema: map[string]any{
					"type":     "object",
					"required": []any{"customer", "items"},
					"properties": map[string]any{
						"customer": map[string]any{
							"type":     "object",
							"required": []any{"email"},
							"properties": map[string]any{
								"email": map[string]any{"type": "string", "description": "Customer email", "example": "ada@example.com"},
								"name":  map[string]any{"type": "string", "default": "Ada"},
							},
						},
						"items": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type":     "object",
								"required": []any{"sku"},
								"properties": map[string]any{
									"quantity": map[string]any{"type": "integer", "default": 1},
									"sku":      map[string]any{"type": "string", "description": "Product SKU"},
								},
							},
						},
						"note": map[string]any{"type": "string"},
					},
				},
			},
		}},
	}}})

	body := op.RequestBody
	if !body.Required || body.ContentType != "application/json" || body.Type != "object" {
		t.Fatalf("request body context = %#v", body)
	}
	assertBodyField(t, body.Fields, "customer.email", true, "string", "Customer email", "ada@example.com", nil)
	assertBodyField(t, body.Fields, "customer.name", false, "string", "", nil, "Ada")
	assertBodyField(t, body.Fields, "items[].sku", true, "string", "Product SKU", nil, nil)
	assertBodyField(t, body.Fields, "items[].quantity", false, "integer", "", nil, 1)
	if !containsString(body.RequiredFieldPaths, "customer.email") || !containsString(body.RequiredFieldPaths, "items[].sku") {
		t.Fatalf("required field paths = %#v", body.RequiredFieldPaths)
	}
}

func TestDraftPromptRequestFallsBackForUnknownRequestBodySchema(t *testing.T) {
	op := promptOperation(t, DraftRequest{Docs: []APIDocument{{
		RelativePath: "openapi/upload.yaml",
		Operations: []*rollout.OperationInfo{{
			OperationID: "upload",
			Method:      "POST",
			Path:        "/upload",
			RequestBody: &rollout.RequestBodyInfo{Required: true, ContentType: "application/octet-stream"},
		}},
	}}})

	body := op.RequestBody
	if len(body.Fields) != 1 || body.Fields[0].Path != "body" || !body.Fields[0].Required {
		t.Fatalf("request body fallback = %#v", body.Fields)
	}
	if got := op.RequiredFields; len(got) != 1 || got[0] != "body" {
		t.Fatalf("required_fields fallback = %#v", got)
	}
}

func TestDraftPromptRequestIncludesSecurityCredentialFields(t *testing.T) {
	op := promptOperation(t, DraftRequest{Docs: []APIDocument{{
		RelativePath: "openapi/support.yaml",
		Operations: []*rollout.OperationInfo{{
			OperationID: "getTicket",
			Method:      "GET",
			Path:        "/tickets/{ticketId}",
			Security:    []string{"BearerAuth", "ApiKeyAuth"},
		}},
	}}})

	security := op.Security
	if !containsString(security.Schemes, "ApiKeyAuth") || !containsString(security.Schemes, "BearerAuth") {
		t.Fatalf("security schemes = %#v", security.Schemes)
	}
	if !containsString(security.CredentialFields, "api_key_auth") || !containsString(security.CredentialFields, "Authorization") {
		t.Fatalf("credential fields = %#v", security.CredentialFields)
	}
}

func promptOperation(t *testing.T, request DraftRequest) operationPromptContext {
	t.Helper()
	payload := draftPromptRequest(request)
	docs := payload["docs"].([]map[string]any)
	if len(docs) != 1 {
		t.Fatalf("docs = %#v", docs)
	}
	ops := docs[0]["operations"].([]operationPromptContext)
	if len(ops) != 1 {
		t.Fatalf("operations = %#v", ops)
	}
	return ops[0]
}

func assertParameterContext(t *testing.T, parameters []parameterPromptContext, name, location string, required bool, typ, description string) {
	t.Helper()
	for _, parameter := range parameters {
		if parameter.Name != name {
			continue
		}
		if parameter.Location != location || parameter.Required != required || parameter.Type != typ || parameter.Description != description {
			t.Fatalf("parameter %q = %#v", name, parameter)
		}
		return
	}
	t.Fatalf("missing parameter %q in %#v", name, parameters)
}

func assertBodyField(t *testing.T, fields []requestBodyFieldContext, path string, required bool, typ, description string, example, defaultValue any) {
	t.Helper()
	for _, field := range fields {
		if field.Path != path {
			continue
		}
		if field.Required != required || field.Type != typ || field.Description != description || field.Example != example || field.Default != defaultValue {
			t.Fatalf("body field %q = %#v", path, field)
		}
		return
	}
	t.Fatalf("missing body field %q in %#v", path, fields)
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
