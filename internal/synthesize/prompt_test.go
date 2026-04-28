package synthesize

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/genelet/udon/pkg/rollout"
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
	for _, expected := range []string{"## Examples", "single_openapi", "two_step_bind", "runtime_fnct"} {
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
