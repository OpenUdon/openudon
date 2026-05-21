package elicitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
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
		Operations: []apitools.OperationSummary{{
			OperationID: "getTicket",
			Method:      "GET",
			Path:        "/tickets/{ticketId}",
			Tags:        []string{"support"},
			Parameters: []apitools.ParameterSummary{
				{Name: "ticketId", In: "path", Required: true, Type: "string", Description: "Ticket identifier"},
				{Name: "include", In: "query", Type: "string", Description: "Related resources"},
				{Name: "X-Trace-ID", In: "header", Type: "string", Description: "Trace header"},
			},
		}},
	}}})

	assertParameterContext(t, op.Parameters, "ticketId", "path", true, "string", "Ticket identifier")
	assertParameterContext(t, op.Parameters, "include", "query", false, "string", "Related resources")
	assertParameterContext(t, op.Parameters, "X-Trace-ID", "header", false, "string", "Trace header")
	if !containsString(op.Tags, "support") {
		t.Fatalf("tags = %#v", op.Tags)
	}
}

func TestDraftPromptRequestFlattensNestedRequestBody(t *testing.T) {
	op := promptOperation(t, DraftRequest{Docs: []APIDocument{{
		RelativePath: "openapi/orders.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "createOrder",
			Method:      "POST",
			Path:        "/orders",
			RequestBody: &apitools.RequestBodySummary{
				Required:     true,
				ContentTypes: []string{"application/json"},
				Schema:       &apitools.SchemaSummary{Type: "object"},
				Fields: []apitools.RequestFieldSummary{
					{Path: "customer.email", Required: true, Type: "string", Description: "Customer email"},
					{Path: "customer.name", Type: "string"},
					{Path: "items[].sku", Required: true, Type: "string", Description: "Product SKU"},
					{Path: "items[].quantity", Type: "integer"},
				},
				RequiredFieldPaths: []string{"customer.email", "items[].sku"},
			},
		}},
	}}})

	body := op.RequestBody
	if !body.Required || body.ContentType != "application/json" || body.Type != "object" {
		t.Fatalf("request body context = %#v", body)
	}
	assertBodyField(t, body.Fields, "customer.email", true, "string", "Customer email")
	assertBodyField(t, body.Fields, "customer.name", false, "string", "")
	assertBodyField(t, body.Fields, "items[].sku", true, "string", "Product SKU")
	assertBodyField(t, body.Fields, "items[].quantity", false, "integer", "")
	if !containsString(body.RequiredFieldPaths, "customer.email") || !containsString(body.RequiredFieldPaths, "items[].sku") {
		t.Fatalf("required field paths = %#v", body.RequiredFieldPaths)
	}
}

func TestDraftPromptRequestFallsBackForUnknownRequestBodySchema(t *testing.T) {
	op := promptOperation(t, DraftRequest{Docs: []APIDocument{{
		RelativePath: "openapi/upload.yaml",
		Operations: []apitools.OperationSummary{{
			OperationID: "upload",
			Method:      "POST",
			Path:        "/upload",
			RequestBody: &apitools.RequestBodySummary{
				Required:     true,
				ContentTypes: []string{"application/octet-stream"},
				Fields:       []apitools.RequestFieldSummary{{Path: "body", Required: true}},
			},
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
		Operations: []apitools.OperationSummary{{
			OperationID: "getTicket",
			Method:      "GET",
			Path:        "/tickets/{ticketId}",
			Security:    securitySummaries("BearerAuth", "ApiKeyAuth"),
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

func TestDraftPromptRequestRanksOperationTextMatches(t *testing.T) {
	ops := promptOperations(t, DraftRequest{
		Opening: "Search support tickets by query.",
		Docs: []APIDocument{{
			RelativePath: "openapi/support.yaml",
			Operations: []apitools.OperationSummary{
				{OperationID: "getTicket", Method: "GET", Path: "/tickets/{ticketId}", Summary: "Get a support ticket"},
				{OperationID: "searchTickets", Method: "GET", Path: "/tickets/search", Summary: "Search support tickets"},
			},
		}},
	}, 0)

	if got := operationPromptIDs(ops); strings.Join(got, ",") != "searchTickets,getTicket" {
		t.Fatalf("ranked operations = %#v", got)
	}
}

func TestOperationRankingBoostsSelectedDocument(t *testing.T) {
	request := DraftRequest{
		Opening: "List records.",
		Session: Session{Intent: rollout.Intent{
			OpenAPI: "openapi/b.yaml",
		}},
		Docs: []APIDocument{
			{
				RelativePath: "openapi/a.yaml",
				Operations:   []apitools.OperationSummary{{OperationID: "listRecordsA", Method: "GET", Path: "/records", Summary: "List records"}},
			},
			{
				RelativePath: "openapi/b.yaml",
				Operations:   []apitools.OperationSummary{{OperationID: "listRecordsB", Method: "GET", Path: "/records", Summary: "List records"}},
			},
		},
	}

	ranked := rankOperationCandidates(request)
	if len(ranked) == 0 || ranked[0].op.OperationID != "listRecordsB" {
		t.Fatalf("ranked candidates = %#v", ranked)
	}
}

func TestDraftPromptRequestCapsUnselectedOperations(t *testing.T) {
	ops := promptOperations(t, DraftRequest{Docs: []APIDocument{{
		RelativePath: "openapi/many.yaml",
		Operations:   numberedOperations(15),
	}}}, 0)

	got := operationPromptIDs(ops)
	if len(got) != maxDraftOperationCandidates {
		t.Fatalf("operation count = %d, ids=%#v", len(got), got)
	}
	if containsString(got, "operation12") || containsString(got, "operation14") {
		t.Fatalf("uncapped operations included: %#v", got)
	}
}

func TestDraftPromptRequestUsesOnlySelectedOperationWhenAvailable(t *testing.T) {
	ops := promptOperations(t, DraftRequest{
		Session: Session{Intent: rollout.Intent{
			OpenAPI: "openapi/many.yaml",
			Steps: []*rollout.Step{{
				Name:      "selected",
				Type:      "http",
				Operation: "operation14",
			}},
		}},
		Docs: []APIDocument{{
			RelativePath: "openapi/many.yaml",
			Operations:   numberedOperations(15),
		}},
	}, 0)

	got := operationPromptIDs(ops)
	if len(got) != 1 {
		t.Fatalf("operation count = %d, ids=%#v", len(got), got)
	}
	if got[0] != "operation14" {
		t.Fatalf("selected operation context = %#v, want operation14 only", got)
	}
}

func promptOperation(t *testing.T, request DraftRequest) operationPromptContext {
	t.Helper()
	ops := promptOperations(t, request, 0)
	if len(ops) != 1 {
		t.Fatalf("operations = %#v", ops)
	}
	return ops[0]
}

func promptOperations(t *testing.T, request DraftRequest, docIndex int) []operationPromptContext {
	t.Helper()
	payload := draftPromptRequest(request)
	docs := payload["docs"].([]map[string]any)
	if len(docs) <= docIndex {
		t.Fatalf("docs = %#v", docs)
	}
	return docs[docIndex]["operations"].([]operationPromptContext)
}

func numberedOperations(count int) []apitools.OperationSummary {
	ops := make([]apitools.OperationSummary, 0, count)
	for i := 0; i < count; i++ {
		id := fmt.Sprintf("operation%02d", i)
		ops = append(ops, apitools.OperationSummary{
			OperationID: id,
			Method:      "GET",
			Path:        "/resources/" + id,
			Summary:     "Resource operation",
		})
	}
	return ops
}

func operationPromptIDs(ops []operationPromptContext) []string {
	out := make([]string, 0, len(ops))
	for _, op := range ops {
		out = append(out, op.OperationID)
	}
	return out
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

func assertBodyField(t *testing.T, fields []requestBodyFieldContext, path string, required bool, typ, description string) {
	t.Helper()
	for _, field := range fields {
		if field.Path != path {
			continue
		}
		if field.Required != required || field.Type != typ || field.Description != description {
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

func securitySummaries(names ...string) []apitools.SecuritySummary {
	out := make([]apitools.SecuritySummary, 0, len(names))
	for _, name := range names {
		out = append(out, apitools.SecuritySummary{Name: name})
	}
	return out
}
