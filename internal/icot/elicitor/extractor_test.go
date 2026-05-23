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

type sequenceChat struct {
	responses []string
	messages  [][]rollout.ChatMessage
}

func (f *sequenceChat) Chat(_ context.Context, messages []rollout.ChatMessage) (string, error) {
	f.messages = append(f.messages, append([]rollout.ChatMessage(nil), messages...))
	if len(f.responses) == 0 {
		return `{}`, nil
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func (f *sequenceChat) StructuredChat(ctx context.Context, messages []rollout.ChatMessage, _ json.RawMessage, _ rollout.StructuredOpts) (string, error) {
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

func TestExtractorSchemasAreStrictStructuredOutputSchemas(t *testing.T) {
	for name, schema := range map[string]string{
		"kickoff":         kickoffSchema,
		"disambiguate":    disambiguateSchema,
		"catalog_plan":    catalogPlanSchema,
		"request_mapping": requestMappingsSchema,
		"draft_review":    draftReviewSchema,
		"draft":           draftSchema,
	} {
		t.Run(name, func(t *testing.T) {
			var raw map[string]any
			if err := json.Unmarshal([]byte(schema), &raw); err != nil {
				t.Fatalf("schema is invalid JSON: %v\n%s", err, schema)
			}
			assertStrictStructuredSchema(t, "$", raw)
		})
	}
}

func TestDraftSchemaIncludesStructuredStepMappings(t *testing.T) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(draftSchema), &raw); err != nil {
		t.Fatalf("schema is invalid JSON: %v", err)
	}
	step := raw["properties"].(map[string]any)["intent"].(map[string]any)["properties"].(map[string]any)["steps"].(map[string]any)["items"].(map[string]any)
	props := step["properties"].(map[string]any)
	if _, ok := props["with"]; !ok {
		t.Fatalf("draft step schema missing with: %#v", props)
	}
	if _, ok := props["bind"]; !ok {
		t.Fatalf("draft step schema missing bind: %#v", props)
	}
}

func TestDraftStructuredMappingArraysPopulateIntentStep(t *testing.T) {
	extractor := NewChatExtractor(fakeChat{response: `{
  "intent": {
    "workflow": {"name": "demo", "description": "Demo"},
    "steps": [{
      "name": "send",
      "type": "http",
      "with": [{"field":"body","source":"render.received_body"}],
      "bind": [{"from":"render","fields":[{"field":"body","source":"received_body"}]}]
    }],
    "outputs": [{"name":"result","from":"send.received_body"}]
  }
}`}, nil)
	session, err := extractor.Draft(context.Background(), DraftRequest{Opening: "Send rendered content"})
	if err != nil {
		t.Fatalf("Draft failed: %v", err)
	}
	if len(session.Intent.Steps) != 1 {
		t.Fatalf("steps = %#v", session.Intent.Steps)
	}
	step := session.Intent.Steps[0]
	if got := step.With["body"]; got != "render.received_body" {
		t.Fatalf("with = %#v", step.With)
	}
	if len(step.Binds) != 1 || step.Binds[0].From != "render" || step.Binds[0].Fields["body"] != "received_body" {
		t.Fatalf("binds = %#v", step.Binds)
	}
}

func assertStrictStructuredSchema(t *testing.T, path string, schema map[string]any) {
	t.Helper()
	if schema["type"] == "object" {
		if got, ok := schema["additionalProperties"].(bool); !ok || got {
			t.Fatalf("%s additionalProperties = %#v, want false", path, schema["additionalProperties"])
		}
		props, _ := schema["properties"].(map[string]any)
		requiredValues, _ := schema["required"].([]any)
		required := map[string]bool{}
		for _, value := range requiredValues {
			required[fmt.Sprint(value)] = true
		}
		for name := range props {
			if !required[name] {
				t.Fatalf("%s property %q is not required", path, name)
			}
		}
		if len(required) != len(props) {
			t.Fatalf("%s required count = %d, properties = %d", path, len(required), len(props))
		}
		for name, value := range props {
			if nested, ok := value.(map[string]any); ok {
				assertStrictStructuredSchema(t, path+"."+name, nested)
			}
		}
	}
	if schema["type"] == "array" {
		if item, ok := schema["items"].(map[string]any); ok {
			assertStrictStructuredSchema(t, path+"[]", item)
		}
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

func TestDraftPromptRequestIncludesFullCatalogButOnlySelectedDetails(t *testing.T) {
	request := DraftRequest{
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
			Title:        "Many API",
			Operations:   numberedOperations(15),
		}},
	}
	payload := draftPromptRequest(request)
	ops := payload["docs"].([]map[string]any)[0]["operations"].([]operationPromptContext)
	if got := operationPromptIDs(ops); len(got) != 1 || got[0] != "operation14" {
		t.Fatalf("detailed operations = %#v, want operation14 only", got)
	}
	catalog := payload["operation_catalog"].([]operationCatalogDocumentContext)
	if len(catalog) != 1 || len(catalog[0].Operations) != 15 {
		t.Fatalf("catalog = %#v", catalog)
	}
	if catalog[0].Operations[14].OperationID != "operation14" {
		t.Fatalf("catalog missing selected operation: %#v", catalog[0].Operations)
	}
	encoded, err := json.Marshal(catalog)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	for _, forbidden := range []string{"required_fields", "parameters", "request_body", "security"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("catalog includes detailed field %q: %s", forbidden, encoded)
		}
	}
}

func TestDraftDetailLoopFetchesRequestedOperationBeforeDraft(t *testing.T) {
	chat := &sequenceChat{responses: []string{
		`{"requested_operation_ids":["geocodeCity"],"detail_request_reason":"Need coordinates before weather lookup."}`,
		`{
  "intent": {
    "openapi": "openapi/weather.yaml",
    "workflow": {"name": "weather_email", "description": "Resolve Toronto, fetch weather, and send a Gmail report."},
    "inputs": [{"name":"city","type":"string","required":true}],
    "steps": [
      {"name":"geocode_city","type":"http","openapi":"openapi/weather.yaml","operation":"geocodeCity","with":{"q":"Toronto,CA"}},
      {"name":"get_weather","type":"http","openapi":"openapi/weather.yaml","operation":"getWeatherByLatLon","with":{"lat":"geocode_city.received_body.lat","lon":"geocode_city.received_body.lon","appid":"credentials.weather_api_key"}},
      {"name":"send_gmail","type":"http","openapi":"openapi/gmail.yaml","operation":"gmail_users_messages_send","with":{"userId":"me","raw":"get_weather.received_body"}}
    ],
    "outputs": [{"name":"sent_message","from":"send_gmail.received_body"}]
  },
  "credentials": ["weather_api_key","gmail_oauth_token"],
  "side_effect_scope": "after-approval",
  "assumptions": [{"id":"op_geocode_city","slot":"steps.geocode_city.operation","value":"geocodeCity","reason":"Toronto must be converted to coordinates before the selected weather operation can run.","evidence":"operation catalog listed geocodeCity and details were fetched","risk":"review","requires_confirmation":true}]
}`,
	}}
	extractor := NewChatExtractor(chat, nil)
	session, err := extractor.Draft(context.Background(), weatherGmailDraftRequest())
	if err != nil {
		t.Fatalf("Draft failed: %v", err)
	}
	if len(chat.messages) != 2 {
		t.Fatalf("chat calls = %d, want 2", len(chat.messages))
	}
	secondPayload := chat.messages[1][1].Content
	if !strings.Contains(secondPayload, `"operationId":"geocodeCity"`) || !strings.Contains(secondPayload, `"parameters"`) {
		t.Fatalf("second prompt did not include geocode details:\n%s", secondPayload)
	}
	if len(session.Intent.Steps) != 3 || session.Intent.Steps[0].Operation != "geocodeCity" {
		t.Fatalf("draft steps = %#v", session.Intent.Steps)
	}
	if len(session.DraftEvents) != 2 || session.DraftEvents[0].Kind != "operation_detail_request" || session.DraftEvents[1].Kind != "operation_detail_fulfilled" {
		t.Fatalf("draft events = %#v", session.DraftEvents)
	}
}

func TestDraftDetailLoopRejectsUnknownRequestedOperationIDs(t *testing.T) {
	chat := &sequenceChat{responses: []string{`{
  "requested_operation_ids":["inventedGeocode"],
  "detail_request_reason":"Need a geocoder.",
  "intent": {
    "workflow": {"name":"bad_weather","description":"Fetch weather."},
    "steps": [{"name":"invented","type":"http","openapi":"openapi/weather.yaml","operation":"inventedGeocode"}]
  }
}`}}
	extractor := NewChatExtractor(chat, nil)
	session, err := extractor.Draft(context.Background(), weatherGmailDraftRequest())
	if err != nil {
		t.Fatalf("Draft failed: %v", err)
	}
	if len(chat.messages) != 1 {
		t.Fatalf("chat calls = %d, want 1", len(chat.messages))
	}
	if len(session.Intent.Steps) != 1 || session.Intent.Steps[0].Operation != "" {
		t.Fatalf("unknown operation was accepted: %#v", session.Intent.Steps)
	}
	if !hasAssumption(session.Assumptions, "operation_detail_request_rejected") {
		t.Fatalf("missing rejected detail warning: %#v", session.Assumptions)
	}
	if len(session.DraftEvents) != 1 || session.DraftEvents[0].Kind != "operation_detail_rejected" {
		t.Fatalf("draft events = %#v", session.DraftEvents)
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

func TestResolveDraftDetailRequestsCapsRequestedOperationIDs(t *testing.T) {
	var ops []apitools.OperationSummary
	for i := 0; i < maxDraftRequestedOperations+2; i++ {
		ops = append(ops, apitools.OperationSummary{OperationID: fmt.Sprintf("operation%d", i)})
	}
	var requested []string
	for _, op := range ops {
		requested = append(requested, op.OperationID)
	}

	valid, rejected, capped := resolveDraftDetailRequests([]APIDocument{{
		RelativePath: "openapi/many.yaml",
		Operations:   ops,
	}}, requested, nil)
	if !capped {
		t.Fatal("expected requested operation cap")
	}
	if len(valid) != maxDraftRequestedOperations {
		t.Fatalf("valid requested operations = %d, want %d", len(valid), maxDraftRequestedOperations)
	}
	if len(rejected) != 0 {
		t.Fatalf("rejected = %#v", rejected)
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

func weatherGmailDraftRequest() DraftRequest {
	return DraftRequest{
		Opening: "Get weather for Toronto, Canada, and Gmail me the report.",
		Session: Session{Intent: rollout.Intent{
			OpenAPI: "openapi/weather.yaml",
			Workflow: &rollout.WorkflowMeta{
				Name:        "weather_email",
				Description: "Get weather for Toronto, Canada, and Gmail me the report.",
			},
			Steps: []*rollout.Step{
				{Name: "get_weather", Type: "http", OpenAPI: "openapi/weather.yaml", Operation: "getWeatherByLatLon"},
				{Name: "send_gmail", Type: "http", OpenAPI: "openapi/gmail.yaml", Operation: "gmail_users_messages_send"},
			},
		}},
		Docs: []APIDocument{
			{RelativePath: "openapi/weather.yaml", Title: "Weather API", Operations: []apitools.OperationSummary{
				{
					OperationID: "getWeatherByLatLon",
					Method:      "GET",
					Path:        "/weather",
					Summary:     "Get current weather by coordinates",
					Parameters: []apitools.ParameterSummary{
						{Name: "lat", In: "query", Required: true, Type: "number"},
						{Name: "lon", In: "query", Required: true, Type: "number"},
						{Name: "appid", In: "query", Required: true, Type: "string"},
					},
				},
				{
					OperationID: "geocodeCity",
					Method:      "GET",
					Path:        "/geo/1.0/direct",
					Summary:     "Resolve a city name to latitude and longitude",
					Parameters: []apitools.ParameterSummary{
						{Name: "q", In: "query", Required: true, Type: "string"},
					},
				},
			}},
			{RelativePath: "openapi/gmail.yaml", Title: "Gmail API", Operations: []apitools.OperationSummary{
				{
					OperationID: "gmail_users_messages_send",
					Method:      "POST",
					Path:        "/gmail/v1/users/{userId}/messages/send",
					Summary:     "Send a Gmail message",
					Parameters:  []apitools.ParameterSummary{{Name: "userId", In: "path", Required: true, Type: "string"}},
					RequestBody: &apitools.RequestBodySummary{
						Required: true,
						Fields:   []apitools.RequestFieldSummary{{Path: "raw", Required: true, Type: "string"}},
					},
				},
			}},
		},
	}
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
