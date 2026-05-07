package workflowintent

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/genelet/udon/pkg/rollout"
	"github.com/OpenUdon/apitools"
)

func TestWorkflowFlowParsesValidatesAndRendersIntent(t *testing.T) {
	flow := WorkflowFlow()
	draft, artifacts, diagnostics, err := flow.ParseValidateRender(context.Background(), apitools.Artifact{
		Path:    IntentPath,
		Content: []byte(validIntentHCL()),
	})
	if err != nil {
		t.Fatal(err)
	}
	if apitools.HasErrors(diagnostics) {
		t.Fatalf("diagnostics = %#v", diagnostics)
	}
	if draft.Workflow == nil || draft.Workflow.Name != "runtime_only_render" {
		t.Fatalf("draft = %#v", draft)
	}
	if len(artifacts.Artifacts) != 1 || artifacts.Artifacts[0].Path != IntentPath {
		t.Fatalf("artifacts = %#v", artifacts)
	}
	if _, err := rollout.ParseIntent(artifacts.Artifacts[0].Content, IntentPath); err != nil {
		t.Fatalf("rendered intent did not parse: %v\n%s", err, artifacts.Artifacts[0].Content)
	}
}

func TestValidateCompleteReportsRamenMissingSlots(t *testing.T) {
	draft := &rollout.Intent{
		OpenAPI:  "openapi/support.yaml",
		Workflow: &rollout.WorkflowMeta{Name: "support_lookup"},
		Steps: []*rollout.Step{{
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
	turns := []apitools.TranscriptTurn{{Role: "user", Content: "hello"}}

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

type fakeStructuredChat struct {
	chatMessages       []rollout.ChatMessage
	structuredMessages []rollout.ChatMessage
	schema             json.RawMessage
	opts               rollout.StructuredOpts
}

func (fake *fakeStructuredChat) Chat(_ context.Context, messages []rollout.ChatMessage) (string, error) {
	fake.chatMessages = append([]rollout.ChatMessage(nil), messages...)
	return "plain reply", nil
}

func (fake *fakeStructuredChat) StructuredChat(_ context.Context, messages []rollout.ChatMessage, schema json.RawMessage, opts rollout.StructuredOpts) (string, error) {
	fake.structuredMessages = append([]rollout.ChatMessage(nil), messages...)
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
