package workflowintent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestParseIntentRejectsUnknownHCLWithOriginalLine(t *testing.T) {
	for _, tc := range []struct {
		name string
		hcl  string
		line string
	}{
		{name: "attribute", hcl: "openapi = \"openapi/items.yaml\"\n\nunknown_value = true\n", line: "intent.hcl:3"},
		{name: "block", hcl: "openapi = \"openapi/items.yaml\"\n\nunknown_block {\n}\n", line: "intent.hcl:3"},
		{name: "compatibility rewrite", hcl: "openapi = \"openapi/items.yaml\"\nstep \"read\" {\n  type = \"http\"\n  operation = \"read\"\n  bind \"prior\" {\n    unknown_value = true\n  }\n}\n", line: "intent.hcl:6"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseIntent([]byte(tc.hcl), "intent.hcl")
			if err == nil || !strings.Contains(err.Error(), tc.line) {
				t.Fatalf("ParseIntent error = %v, want original location %q", err, tc.line)
			}
		})
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

func TestIntentSourceAliasNormalizesForGeneration(t *testing.T) {
	intent, err := ParseIntent([]byte(`source = "google-discovery/gmail.json"

workflow {
  name = "gmail_send"
}

step "send" {
  type      = "http"
  source    = "google-discovery/gmail.json"
  operation = "gmail_users_messages_send"
  with = {
    "path.userId" = "me"
  }
}
`), IntentPath)
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := intent.NormalizedForGeneration()
	if err != nil {
		t.Fatal(err)
	}
	if normalized.OpenAPI != "google-discovery/gmail.json" || normalized.Steps[0].OpenAPI != "google-discovery/gmail.json" {
		t.Fatalf("source alias was not normalized: %#v", normalized)
	}
	rendered, err := RenderIntentHCL(intent)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered, `source = "google-discovery/gmail.json"`) {
		t.Fatalf("rendered intent missing source alias:\n%s", rendered)
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

func TestCopilotGPT5OmitsUnsupportedTemperature(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, got)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"{}"}`))
	}))
	defer server.Close()
	t.Setenv("COPILOT_API_BASE_URL", server.URL)
	temp := 0.2

	client, _, _, err := NewLLMClientFromEnvWithOptions("", "", LLMOptions{Temperature: &temp})
	if err != nil {
		t.Fatal(err)
	}
	chat := client.(ChatClient)
	if _, err := chat.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}); err != nil {
		t.Fatal(err)
	}
	structured := client.(StructuredChat)
	if _, err := structured.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "emit json"}}, json.RawMessage(`{"type":"object"}`), StructuredOpts{Temperature: &temp}); err != nil {
		t.Fatal(err)
	}

	if len(bodies) != 2 {
		t.Fatalf("request bodies = %d, want 2", len(bodies))
	}
	for _, body := range bodies {
		if _, ok := body["temperature"]; ok {
			t.Fatalf("copilot-api GPT-5 request included unsupported temperature: %#v", body)
		}
	}
}

func TestUnsupportedOptionalRequestParameterRetriesWithoutParameter(t *testing.T) {
	var bodies []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var got map[string]any
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, got)
		w.Header().Set("Content-Type", "application/json")
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"Unsupported parameter: 'max_output_tokens' is not supported with this model."}}`))
			return
		}
		_, _ = w.Write([]byte(`{"output_text":"{}"}`))
	}))
	defer server.Close()
	t.Setenv("COPILOT_API_BASE_URL", server.URL)

	client, _, _, err := NewLLMClientFromEnvWithOptions("", "", LLMOptions{})
	if err != nil {
		t.Fatal(err)
	}
	structured := client.(StructuredChat)
	if _, err := structured.StructuredChat(context.Background(), []ChatMessage{{Role: "user", Content: "emit json"}}, json.RawMessage(`{"type":"object"}`), StructuredOpts{MaxTokens: 123}); err != nil {
		t.Fatal(err)
	}

	if len(bodies) != 2 {
		t.Fatalf("request bodies = %d, want 2", len(bodies))
	}
	if _, ok := bodies[0]["max_output_tokens"]; !ok {
		t.Fatalf("first request missing max_output_tokens: %#v", bodies[0])
	}
	if _, ok := bodies[1]["max_output_tokens"]; ok {
		t.Fatalf("retry still included unsupported max_output_tokens: %#v", bodies[1])
	}
	if _, ok := bodies[1]["text"]; !ok {
		t.Fatalf("retry dropped unrelated structured output config: %#v", bodies[1])
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

func TestGeminiAPIKeyUsesHeaderAndNeverURL(t *testing.T) {
	const secret = "gemini-secret-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || strings.Contains(r.RequestURI, secret) {
			t.Fatalf("Gemini key leaked into request URL: %s", r.RequestURI)
		}
		if got := r.Header.Get("x-goog-api-key"); got != secret {
			t.Fatalf("x-goog-api-key = %q", got)
		}
		_, _ = w.Write([]byte(`{"candidates":[{"content":{"parts":[{"text":"ok"}]}}]}`))
	}))
	defer server.Close()
	t.Setenv("GEMINI_BASE_URL", server.URL)
	client := &providerLLMClient{provider: "gemini", model: "gemini-test", apiKey: secret, client: server.Client(), timeout: defaultLLMTimeout}
	if got, err := client.Chat(context.Background(), []ChatMessage{{Role: "user", Content: "hello"}}); err != nil || got != "ok" {
		t.Fatalf("Gemini chat = %q, %v", got, err)
	}
}

func TestProviderResponseLimitAppliesToSuccessAndError(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusBadGateway} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte(strings.Repeat("x", maxProviderResponseBytes+1)))
			}))
			defer server.Close()
			client := &providerLLMClient{provider: "openai", model: "test", apiKey: "secret", client: server.Client(), timeout: defaultLLMTimeout}
			_, err := client.chatOpenAI(context.Background(), server.URL, []ChatMessage{{Role: "user", Content: "hello"}})
			if err == nil || !strings.Contains(err.Error(), "response exceeded") {
				t.Fatalf("oversized response error = %v", err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("provider error leaked API key: %v", err)
			}
		})
	}
}

type secretErrorTransport struct{}

func (secretErrorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("proxy failure Authorization: Bearer transport-secret")
}

func TestProviderTransportErrorsAreFullyRedacted(t *testing.T) {
	client := &providerLLMClient{
		provider: "openai", model: "test", apiKey: "api-secret",
		client: &http.Client{Transport: secretErrorTransport{}}, timeout: defaultLLMTimeout,
	}
	_, err := client.chatOpenAI(context.Background(), "https://api.example.invalid/v1/chat", []ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "transport request failed") {
		t.Fatalf("transport error = %v", err)
	}
	if strings.Contains(err.Error(), "transport-secret") || strings.Contains(err.Error(), "api-secret") || strings.Contains(err.Error(), "Authorization") {
		t.Fatalf("transport error leaked secret material: %v", err)
	}
}

type secretStatusTransport struct{}

func (secretStatusTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusBadGateway,
		Status:     "502 Authorization: Bearer response-secret",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(`{"error":"response-secret"}`)),
		Request:    request,
	}, nil
}

func TestProviderHTTPFailureDoesNotSurfaceStatusOrBodySecrets(t *testing.T) {
	client := &providerLLMClient{
		provider: "openai", model: "test", apiKey: "api-secret",
		client: &http.Client{Transport: secretStatusTransport{}}, timeout: defaultLLMTimeout,
	}
	_, err := client.chatOpenAI(context.Background(), "https://api.example.invalid/v1/chat", []ChatMessage{{Role: "user", Content: "hello"}})
	if err == nil || !strings.Contains(err.Error(), "HTTP status 502") {
		t.Fatalf("provider status error = %v", err)
	}
	for _, secret := range []string{"response-secret", "Authorization", "api-secret"} {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("provider status error leaked %q: %v", secret, err)
		}
	}
}

func TestProviderDeadlineUsesShorterCallerOrProviderBound(t *testing.T) {
	caller, cancelCaller := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelCaller()
	bounded, cancelBounded := contextWithLLMDeadline(caller, time.Minute)
	defer cancelBounded()
	callerDeadline, _ := caller.Deadline()
	boundedDeadline, _ := bounded.Deadline()
	if !boundedDeadline.Equal(callerDeadline) {
		t.Fatalf("shorter caller deadline changed: got %s want %s", boundedDeadline, callerDeadline)
	}

	providerBounded, cancelProvider := contextWithLLMDeadline(context.Background(), 20*time.Second)
	defer cancelProvider()
	providerDeadline, ok := providerBounded.Deadline()
	if !ok || time.Until(providerDeadline) > 20*time.Second || time.Until(providerDeadline) < 19*time.Second {
		t.Fatalf("provider deadline was not applied: %s", providerDeadline)
	}
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
