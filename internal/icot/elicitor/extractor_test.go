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
