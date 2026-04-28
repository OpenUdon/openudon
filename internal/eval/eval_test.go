package eval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/ramen/internal/synthesize"
	"github.com/genelet/udon/pkg/rollout"
)

type fakeRuntimeClient struct{}

func (fakeRuntimeClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "workflow": {"name": "runtime_only_render", "description": "Render a local summary report."},
  "steps": [
    {"name": "render_report", "type": "fnct", "do": "Render the summary report."}
  ],
  "outputs": [{"name": "report", "from": "render_report.received_body"}]
}`, nil
}

func (fakeRuntimeClient) Generate(context.Context, string) (string, error) {
	return "```json\n" + `{
  "steps": [
    {
      "type": "fnct",
      "name": "render_report",
      "body": {"attributes": {"function": "identity"}}
    }
  ]
}` + "\n```", nil
}

func TestCompareIntentsReportsStructuralIssues(t *testing.T) {
	generated := &rollout.Intent{
		Steps: []*rollout.Step{{
			Name:      "fetch_ticket",
			Type:      "http",
			Operation: "wrongOperation",
		}},
	}
	reference := &rollout.Intent{
		Steps: []*rollout.Step{{
			Name:      "fetch_ticket",
			Type:      "http",
			Operation: "getTicket",
		}},
	}
	issues := CompareIntents(generated, reference)
	if len(issues) != 1 || issues[0].Code != "intent.step_operation" {
		t.Fatalf("unexpected issues: %#v", issues)
	}
}

func TestRunOneUsesTempCopyAndReadsRefinement(t *testing.T) {
	example := filepath.Join(t.TempDir(), "runtime-only")
	if err := os.MkdirAll(filepath.Join(example, "reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a local summary report.

## Inputs

- summary: string.

## Outputs

- Rendered report.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop on failure.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "reference", "intent.hcl"), []byte(`workflow {
  name = "runtime_only_render"
}

step "render_report" {
  type = "fnct"
  do   = "Render the summary report."
}

output "report" {
  from = "render_report.received_body"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result := RunOne(context.Background(), example, synthesize.Options{
		LLMClient:  fakeRuntimeClient{},
		ChatClient: fakeRuntimeClient{},
		SchemaPath: schemaPath,
	})
	if !result.Passed {
		t.Fatalf("expected eval to pass: %#v", result)
	}
	if result.PromptVersion == "" || result.AttemptsToPass != 1 {
		t.Fatalf("result missing refinement evidence: %#v", result)
	}
	if result.GeneratedDir == "" || result.GeneratedDir == example {
		t.Fatalf("expected generated temp dir, got %q", result.GeneratedDir)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows")); !os.IsNotExist(err) {
		t.Fatalf("eval dirtied source example workflows dir: %v", err)
	}
}

func TestRegressionErrorDetectsFailedPreviouslyPassingBrief(t *testing.T) {
	err := RegressionError(
		[]EvalResult{{Name: "a", Passed: false}},
		[]EvalResult{{Name: "a", Passed: true}},
	)
	if err == nil || !strings.Contains(err.Error(), "pass rate regressed") {
		t.Fatalf("expected regression error, got %v", err)
	}
}
