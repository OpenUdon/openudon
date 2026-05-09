package synthesize

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	rollout "github.com/genelet/ramen/internal/workflowintent"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type captureChatClient struct {
	messages []rollout.ChatMessage
}

func (c *captureChatClient) Chat(_ context.Context, messages []rollout.ChatMessage) (string, error) {
	c.messages = append([]rollout.ChatMessage(nil), messages...)
	return `{"workflow":{"name":"demo"},"steps":[{"name":"render","type":"fnct","do":"Render"}]}`, nil
}

type structuredCaptureChatClient struct {
	messages           []rollout.ChatMessage
	structuredMessages []rollout.ChatMessage
	schema             json.RawMessage
	structuredErr      error
	chatCalled         bool
	structuredCalled   bool
}

func (c *structuredCaptureChatClient) Chat(_ context.Context, messages []rollout.ChatMessage) (string, error) {
	c.chatCalled = true
	c.messages = append([]rollout.ChatMessage(nil), messages...)
	return `{"workflow":{"name":"legacy_demo","description":"Legacy demo"},"steps":[{"name":"render","type":"fnct","do":"Render"}]}`, nil
}

func (c *structuredCaptureChatClient) StructuredChat(_ context.Context, messages []rollout.ChatMessage, schema json.RawMessage, _ rollout.StructuredOpts) (string, error) {
	c.structuredCalled = true
	c.structuredMessages = append([]rollout.ChatMessage(nil), messages...)
	c.schema = append(json.RawMessage(nil), schema...)
	if c.structuredErr != nil {
		return "", c.structuredErr
	}
	return `{"workflow":{"name":"structured_demo","description":"Structured demo"},"steps":[{"name":"render","type":"fnct","do":"Render"}]}`, nil
}

func TestIntentSchemaCompiles(t *testing.T) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(embeddedIntentSchema))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("intent.schema.json", doc); err != nil {
		t.Fatal(err)
	}
	if _, err := compiler.Compile("intent.schema.json"); err != nil {
		t.Fatal(err)
	}
}

func TestIntentSchemaAcceptsTimeoutIdempotencyAndRejectsInvalidValues(t *testing.T) {
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(embeddedIntentSchema))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("intent.schema.json", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("intent.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	valid, err := jsonschema.UnmarshalJSON(strings.NewReader(`{
	  "workflow": {
	    "name": "controls",
	    "description": "Controls",
	    "timeout": 120,
	    "idempotency": {"key": "inputs.request_id", "onConflict": "returnPrevious", "ttl": 86400}
	  },
	  "steps": [{"name": "call_api", "type": "http", "do": "Call API", "timeout": 10}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(valid); err != nil {
		t.Fatalf("schema rejected valid timeout/idempotency intent: %v", err)
	}
	invalid, err := jsonschema.UnmarshalJSON(strings.NewReader(`{
	  "workflow": {
	    "name": "controls",
	    "description": "Controls",
	    "timeout": 0,
	    "idempotency": {"key": "inputs.request_id", "onConflict": "replace"}
	  },
	  "steps": [{"name": "call_api", "type": "http", "do": "Call API", "timeout": 0}]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(invalid); err == nil {
		t.Fatalf("schema accepted invalid timeout/idempotency intent")
	}
}

func TestIntentPromptRendersExamplesAndProjectRequirements(t *testing.T) {
	policy := projectPolicy{
		Inputs:  []InputDecl{{Name: "ticket_id", Type: "string", Required: true}},
		Outputs: []OutputDecl{{Name: "result", From: "send_email.received_body"}},
		BindingHints: []BindingHint{{
			From:  "get_ticket.received_body.summary",
			To:    "send_email.body",
			Field: "body",
		}},
		FunctionContracts: []FunctionContract{{
			Name:        "send_email",
			Inputs:      []string{"to", "subject", "body"},
			Outputs:     []string{"status"},
			SideEffects: "sends email through approved runtime",
		}},
	}
	messages := intentPromptMessages("brief", nil, "", policy, "repair this")
	if len(messages) != 2 {
		t.Fatalf("expected system and user messages, got %#v", messages)
	}
	system := messages[0].Content
	for _, expected := range []string{"## Examples", "single_openapi", "two_step_bind", "runtime_fnct", "structural_switch"} {
		if !strings.Contains(system, expected) {
			t.Fatalf("system prompt missing %q:\n%s", expected, system)
		}
	}
	user := messages[1].Content
	for _, expected := range []string{
		"Required by project.md:",
		"Input: ticket_id, string, required",
		"Output: result from send_email.received_body",
		"Step `send_email.body` MUST receive `get_ticket.received_body.summary` as input `body`.",
		"Function `send_email` inputs: to, subject, body",
		"Previous quality failure to repair:",
		`"<primary openapi path provided above>"`,
	} {
		if !strings.Contains(user, expected) {
			t.Fatalf("user prompt missing %q:\n%s", expected, user)
		}
	}
}

func TestStructuredIntentPromptOmitsLegacyJSONInstruction(t *testing.T) {
	legacy := renderIntentSystemPromptForMode(false)
	structured := renderIntentSystemPromptForMode(true)
	if !strings.Contains(legacy, "Return only JSON. Do not include Markdown.") {
		t.Fatalf("legacy prompt missing JSON instruction:\n%s", legacy)
	}
	if strings.Contains(structured, "Return only JSON. Do not include Markdown.") {
		t.Fatalf("structured prompt kept legacy JSON instruction:\n%s", structured)
	}
}

func TestGenerateIntentUsesRenderedPrompt(t *testing.T) {
	client := &captureChatClient{}
	intent, err := generateIntent(context.Background(), client, "brief", nil, "", analyzeProject("OpenAPI: none required"), "")
	if err != nil {
		t.Fatal(err)
	}
	if intent.Workflow == nil || intent.Workflow.Name != "demo" {
		t.Fatalf("unexpected intent: %#v", intent)
	}
	if len(client.messages) != 2 || !strings.Contains(renderPromptSnapshot(client.messages), "## SYSTEM") {
		t.Fatalf("prompt messages were not captured: %#v", client.messages)
	}
}

func TestDecodeIntentSanitizesModelOnlyStructuralNoise(t *testing.T) {
	intent, err := decodeIntentJSON(`{
  "workflow": {"name": "demo", "description": "Demo"},
  "steps": [
    {"name": "get_token", "type": "fnct", "do": "Load token"},
    {
      "name": "get_inventory",
      "type": "http",
      "do": "Fetch inventory",
      "operation": "getInventory",
      "depends_on": ["inputs", "get_token", "inventory_api_key", "get_token", "get_inventory"],
      "with": {},
      "bind": [],
      "default": {}
    }
  ]
}`, "", analyzeProject("OpenAPI: none required"))
	if err != nil {
		t.Fatal(err)
	}
	got := intent.Steps[1]
	if len(got.DependsOn) != 1 || got.DependsOn[0] != "get_token" {
		t.Fatalf("depends_on = %#v, want only prior step dependency", got.DependsOn)
	}
	if got.With != nil || got.Binds != nil || got.Default != nil {
		t.Fatalf("expected empty generated containers to be pruned: %#v", got)
	}
}

func TestDecodeIntentAppliesProjectBindingsAndCredentialBinds(t *testing.T) {
	policy := analyzeProject(`## Data Flow

- Pass inputs.sku to get_inventory.sku.

## Credentials and Secrets

- Use credential binding inventory_api_key.
`)
	intent, err := decodeIntentJSON(`{
  "workflow": {"name": "demo", "description": "Demo"},
  "steps": [
    {
      "name": "get_inventory",
      "type": "http",
      "do": "Fetch inventory",
      "operation": "getInventory",
      "bind": [{"from": "security/inventory_api_key", "fields": {"api_key": "token"}}]
    }
  ]
}`, "", policy)
	if err != nil {
		t.Fatal(err)
	}
	got := intent.Steps[0]
	if len(got.Binds) != 0 {
		t.Fatalf("credential bind should have been converted, got %#v", got.Binds)
	}
	if got.With["sku"] != "inputs.sku" || got.With["api_key"] != "inventory_api_key" {
		t.Fatalf("with = %#v, want project input and credential bindings", got.With)
	}
}

func TestDecodeIntentAppliesLiteralProjectHintsToStructuredOutput(t *testing.T) {
	tests := []struct {
		name    string
		project string
		raw     string
		want    map[string]map[string]string
	}{
		{
			name: "paginated list",
			project: `## Data Flow

- Pass literal page ` + "`1`" + ` and limit ` + "`50`" + ` to the list operation.
`,
			raw:  `{"workflow":{"name":"demo","description":"Demo"},"steps":[{"name":"list_customers","type":"http","do":"List customer records.","operation":"listCustomers"}]}`,
			want: map[string]map[string]string{"list_customers": {"page": "1", "limit": "50"}},
		},
		{
			name: "two pages",
			project: `## Data Flow

- Fetch page 1 with literal ` + "`page = 1`" + ` and literal ` + "`limit = 50`" + `.
- Fetch page 2 with literal ` + "`page = 2`" + ` and literal ` + "`limit = 50`" + `.
`,
			raw: `{"workflow":{"name":"demo","description":"Demo"},"steps":[{"name":"list_customers_page_1","type":"http","do":"Fetch first page.","operation":"listCustomers"},{"name":"list_customers_page_2","type":"http","do":"Fetch second page.","operation":"listCustomers"}]}`,
			want: map[string]map[string]string{
				"list_customers_page_1": {"page": "1", "limit": "50"},
				"list_customers_page_2": {"page": "2", "limit": "50"},
			},
		},
		{
			name: "weather coordinates",
			project: `## Data Flow

- Resolve Toronto to coordinates before fetching weather.
`,
			raw:  `{"workflow":{"name":"demo","description":"Demo"},"steps":[{"name":"get_coordinates","type":"http","do":"Resolve Toronto, Canada to coordinates.","operation":"direct_get"}]}`,
			want: map[string]map[string]string{"get_coordinates": {"q": "Toronto,CA"}},
		},
		{
			name: "support input fallback",
			project: `## Inputs

- ticketId: required string.
`,
			raw:  `{"workflow":{"name":"demo","description":"Demo"},"steps":[{"name":"get_ticket_details","type":"openapi","do":"Fetch support ticket details.","operation":"getTicket"}]}`,
			want: map[string]map[string]string{"get_ticket_details": {"ticketId": "inputs.ticketId"}},
		},
		{
			name: "support step alias",
			project: `## Data Flow

- Pass get_ticket.received_body.summary to send_confirmation_email.body.
`,
			raw:  `{"workflow":{"name":"demo","description":"Demo"},"steps":[{"name":"get_ticket_details","type":"http","do":"Fetch support ticket details.","operation":"getTicket"},{"name":"send_confirmation_email","type":"fnct","do":"Send email."}]}`,
			want: map[string]map[string]string{"send_confirmation_email": {"body": "get_ticket_details.received_body.summary"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := decodeIntentJSON(tt.raw, "", analyzeProject(tt.project))
			if err != nil {
				t.Fatal(err)
			}
			steps := map[string]*rollout.Step{}
			for _, step := range intent.Steps {
				steps[step.Name] = step
			}
			for stepName, wantWith := range tt.want {
				step := steps[stepName]
				if step == nil {
					t.Fatalf("missing step %q in %#v", stepName, intent.Steps)
				}
				for field, source := range wantWith {
					if step.With[field] != source {
						t.Fatalf("%s.with[%s] = %q, want %q; with=%#v", stepName, field, step.With[field], source, step.With)
					}
				}
			}
		})
	}
}

func TestDecodeIntentFoldsParameterSetterStep(t *testing.T) {
	intent, err := decodeIntentJSON(`{
  "workflow": {"name": "demo", "description": "Demo"},
  "steps": [
    {"name": "list_customers", "type": "http", "do": "List customers.", "operation": "listCustomers"},
    {"name": "set_page_and_limit", "type": "fnct", "do": "Set request parameters.", "set": "list_customers.with.page = 1; list_customers.with.limit = 50"}
  ]
}`, "", analyzeProject("OpenAPI: none required"))
	if err != nil {
		t.Fatal(err)
	}
	if len(intent.Steps) != 1 || intent.Steps[0].Name != "list_customers" {
		t.Fatalf("setter step should be folded away: %#v", intent.Steps)
	}
	if intent.Steps[0].With["page"] != "1" || intent.Steps[0].With["limit"] != "50" {
		t.Fatalf("with = %#v, want folded page and limit", intent.Steps[0].With)
	}
}

func TestDecodeIntentAppliesCredentialBindingHints(t *testing.T) {
	intent, err := decodeIntentJSON(`{
  "workflow": {"name": "demo", "description": "Demo"},
  "steps": [
    {"name": "get_weather", "type": "http", "do": "Fetch weather.", "operation": "getWeatherData"}
  ]
}`, "", analyzeProject(`## Credentials and Secrets

- Use credential binding weather_appid.
`))
	if err != nil {
		t.Fatal(err)
	}
	if intent.Steps[0].With["appid"] != "weather_appid" {
		t.Fatalf("with = %#v, want appid credential binding", intent.Steps[0].With)
	}
}

func TestGenerateIntentUsesStructuredChatWhenAvailable(t *testing.T) {
	client := &structuredCaptureChatClient{}
	intent, mode, err := generateIntentWithMode(context.Background(), client, "brief", nil, "", analyzeProject("OpenAPI: none required"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if mode != intentGenerationModeStructured {
		t.Fatalf("mode = %q, want structured", mode)
	}
	if client.chatCalled {
		t.Fatal("legacy Chat should not be called when structured output succeeds")
	}
	if !client.structuredCalled || len(client.schema) == 0 {
		t.Fatalf("structured path did not receive schema")
	}
	if len(client.structuredMessages) != 2 || strings.Contains(client.structuredMessages[0].Content, "Return only JSON. Do not include Markdown.") {
		t.Fatalf("structured prompt was not used: %#v", client.structuredMessages)
	}
	if intent.Workflow == nil || intent.Workflow.Name != "structured_demo" {
		t.Fatalf("unexpected intent: %#v", intent)
	}
}

func TestGenerateIntentFallsBackToLegacyChat(t *testing.T) {
	client := &structuredCaptureChatClient{structuredErr: errors.New("structured output unsupported")}
	intent, mode, err := generateIntentWithMode(context.Background(), client, "brief", nil, "", analyzeProject("OpenAPI: none required"), "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if mode != intentGenerationModeLegacy {
		t.Fatalf("mode = %q, want legacy", mode)
	}
	if !client.structuredCalled || !client.chatCalled {
		t.Fatalf("expected structured attempt and legacy fallback, structured=%v chat=%v", client.structuredCalled, client.chatCalled)
	}
	if len(client.messages) == 0 || !strings.Contains(client.messages[len(client.messages)-1].Content, "Return only JSON. Do not include Markdown.") {
		t.Fatalf("legacy fallback prompt missing JSON instruction: %#v", client.messages)
	}
	if intent.Workflow == nil || intent.Workflow.Name != "legacy_demo" {
		t.Fatalf("unexpected fallback intent: %#v", intent)
	}
}
