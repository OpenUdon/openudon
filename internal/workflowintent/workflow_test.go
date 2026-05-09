package workflowintent

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/authoring"
)

func TestWorkflowFlowParsesValidatesAndRendersIntent(t *testing.T) {
	flow := WorkflowFlow()
	draft, artifacts, diagnostics, err := flow.ParseValidateRender(context.Background(), authoring.Artifact{
		Path:    IntentPath,
		Content: []byte(validIntentHCL()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if authoring.HasErrors(diagnostics) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if draft.Workflow == nil || draft.Workflow.Name != "runtime_only_render" {
		t.Fatalf("draft = %#v", draft)
	}
	if len(artifacts.Artifacts) != 1 || artifacts.Artifacts[0].Path != IntentPath {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	if _, err := ParseIntent(artifacts.Artifacts[0].Content, IntentPath); err != nil {
		t.Fatalf("rendered intent did not parse: %v\n%s", err, artifacts.Artifacts[0].Content)
	}
}

func TestValidateCompleteReportsOpenUdonMissingSlots(t *testing.T) {
	draft := &Intent{
		OpenAPI:  "openapi/support.yaml",
		Workflow: &WorkflowMeta{Name: "support_lookup"},
		Steps: []*Step{{
			Name: "get_ticket",
			Type: "http",
			Do:   "Fetch the ticket.",
		}},
	}
	diagnostics := ValidateComplete(context.Background(), draft)
	var messages []string
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	joined := strings.Join(messages, "\n")
	for _, expected := range []string{"workflow goal", "operation for step get_ticket", "at least one output"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("missing %q in diagnostics:\n%s", expected, joined)
		}
	}
}

func TestChatAdapterConvertsTranscriptAndStructuredOutput(t *testing.T) {
	fake := &fakeStructuredChat{}
	adapter := ChatAdapter{Client: fake, MaxTokens: 42}
	turns := []authoring.TranscriptTurn{{Role: "user", Content: "hello"}}

	reply, err := adapter.Complete(context.Background(), turns)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Role != "assistant" || reply.Content != "plain reply" {
		t.Fatalf("reply = %#v", reply)
	}
	if len(fake.chatMessages) != 1 || fake.chatMessages[0].Role != "user" {
		t.Fatalf("chat messages = %#v", fake.chatMessages)
	}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := adapter.CompleteStructured(context.Background(), turns, json.RawMessage(`{"type":"object"}`), &out); err != nil {
		t.Fatal(err)
	}
	if !out.OK {
		t.Fatalf("structured output = %#v", out)
	}
	if len(fake.structuredMessages) != 1 || string(fake.schema) != `{"type":"object"}` || fake.opts.MaxTokens != 42 {
		t.Fatalf("structured call = %#v %s %#v", fake.structuredMessages, fake.schema, fake.opts)
	}
}

func TestCopilotDefaultGPT5UsesResponsesEndpoint(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"ok"}`))
	}))
	defer server.Close()
	t.Setenv("COPILOT_API_BASE_URL", server.URL)

	client, provider, model, err := NewLLMClientFromEnvWithOptions("", "", LLMOptions{})
	if err != nil {
		t.Fatal(err)
	}
	chat, ok := client.(ChatClient)
	if !ok {
		t.Fatal("client does not implement ChatClient")
	}
	reply, err := chat.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}})
	if err != nil {
		t.Fatal(err)
	}
	if reply != "ok" || provider != "copilot-api" || model != DefaultCopilotAPIModel {
		t.Fatalf("reply/provider/model = %q/%q/%q", reply, provider, model)
	}
	if gotPath != "/v1/responses" || gotBody["model"] != DefaultCopilotAPIModel {
		t.Fatalf("unexpected request path/body: %s %#v", gotPath, gotBody)
	}
}

func TestLLMProviderAndModelCanComeFromOpenUdonEnv(t *testing.T) {
	t.Setenv("OPENUDON_LLM_PROVIDER", "openai")
	t.Setenv("OPENUDON_LLM_MODEL", "gpt-test")
	t.Setenv("OPENAI_API_KEY", "test-key")
	client, provider, model, err := NewLLMClientFromEnvWithOptions("", "", LLMOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if client == nil || provider != "openai" || model != "gpt-test" {
		t.Fatalf("client/provider/model = %T/%q/%q", client, provider, model)
	}
}

func TestProviderStructuredChatSendsProviderNativeSchema(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}}}`)
	t.Run("openai", func(t *testing.T) {
		var got map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/chat/completions" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"ok\":true}"}}]}`))
		}))
		defer server.Close()
		t.Setenv("OPENAI_BASE_URL", server.URL)
		client := &providerLLMClient{provider: "openai", model: "gpt-4.1", apiKey: "key", client: server.Client(), timeout: defaultLLMTimeout}
		if _, err := client.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "emit"}}, schema, StructuredOpts{MaxTokens: 123}); err != nil {
			t.Fatal(err)
		}
		format := got["response_format"].(map[string]any)
		if format["type"] != "json_schema" || got["max_tokens"].(float64) != 123 {
			t.Fatalf("openai structured payload = %#v", got)
		}
	})
	t.Run("anthropic", func(t *testing.T) {
		var got map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/messages" {
				t.Fatalf("path = %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"content":[{"type":"tool_use","name":"emit_json","input":{"ok":true}}]}`))
		}))
		defer server.Close()
		t.Setenv("ANTHROPIC_BASE_URL", server.URL)
		client := &providerLLMClient{provider: "anthropic", model: "claude", apiKey: "key", client: server.Client(), timeout: defaultLLMTimeout}
		if _, err := client.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "emit"}}, schema, StructuredOpts{}); err != nil {
			t.Fatal(err)
		}
		if got["tool_choice"].(map[string]any)["name"] != structuredToolName || len(got["tools"].([]any)) != 1 {
			t.Fatalf("anthropic structured payload = %#v", got)
		}
	})
	t.Run("gemini", func(t *testing.T) {
		var got map[string]any
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, ":generateContent") {
				t.Fatalf("path = %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"{\"ok\":true}"}]}}]}`))
		}))
		defer server.Close()
		t.Setenv("GEMINI_BASE_URL", server.URL)
		client := &providerLLMClient{provider: "gemini", model: "gemini-test", apiKey: "key", client: server.Client(), timeout: defaultLLMTimeout}
		if _, err := client.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "emit"}}, schema, StructuredOpts{}); err != nil {
			t.Fatal(err)
		}
		config := got["generationConfig"].(map[string]any)
		if config["responseMimeType"] != "application/json" || config["responseJsonSchema"] == nil {
			t.Fatalf("gemini structured payload = %#v", got)
		}
	})
}

type fakeStructuredChat struct {
	chatMessages       []ChatMessage
	structuredMessages []ChatMessage
	schema             json.RawMessage
	opts               StructuredOpts
}

func (fake *fakeStructuredChat) Chat(_ context.Context, messages []ChatMessage) (string, error) {
	fake.chatMessages = append([]ChatMessage(nil), messages...)
	return "plain reply", nil
}

func (fake *fakeStructuredChat) StructuredChat(_ context.Context, messages []ChatMessage, schema json.RawMessage, opts StructuredOpts) (string, error) {
	fake.structuredMessages = append([]ChatMessage(nil), messages...)
	fake.schema = append(json.RawMessage(nil), schema...)
	fake.opts = opts
	return `{"ok":true}`, nil
}

func validIntentHCL() string {
	return `
workflow {
  name        = "runtime_only_render"
  description = "Render a local summary report."
}

input "summary" {
  type     = "string"
  required = true
}

step "render_report" {
  type = "fnct"
  do   = "Render the summary report."
  with = {
    summary = "inputs.summary"
  }
}

output "report" {
  from = "render_report.received_body"
}
`
}
