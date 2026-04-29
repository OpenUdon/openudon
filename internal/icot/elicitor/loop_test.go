package elicitor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/udon/pkg/rollout"
)

func TestRunFillsRuntimeOnlyIntent(t *testing.T) {
	example := t.TempDir()
	input := strings.Join([]string{
		"Render a local summary report from a runtime input.",
		"runtime_only_render",
		"Render a local summary report.",
		"no",
		"summary:string",
		"render_report",
		"fnct",
		"Render the summary report.",
		"summary",
		"",
		"",
		"report",
		"render_report.received_body",
		"sandbox-only",
		"",
		"Sandbox proof runs only",
		"Stop if the summary input is missing",
		"save",
	}, "\n") + "\n"

	artifacts, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir: example,
		NoLLM:      true,
		Extractor:  NewNoopExtractor(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if intent.Workflow.Name != "runtime_only_render" || len(intent.Steps) != 1 {
		t.Fatalf("unexpected intent: %#v", intent)
	}
	if got := intent.Steps[0].With["summary"]; got != "inputs.summary" {
		t.Fatalf("summary binding = %q", got)
	}
}

func TestRunFillsOpenAPIRequiredParams(t *testing.T) {
	example := t.TempDir()
	writeOpenAPI(t, example)
	input := strings.Join([]string{
		"Fetch a support ticket.",
		"support_ticket_lookup",
		"Fetch a support ticket by runtime id.",
		"yes",
		"1",
		"ticketId:string",
		"get_ticket",
		"http",
		"Fetch the ticket.",
		"1",
		"1",
		"",
		"",
		"ticket",
		"get_ticket.received_body",
		"sandbox-only",
		"support_api_token",
		"Sandbox proof runs only",
		"Stop if the support API is unavailable",
		"save",
	}, "\n") + "\n"

	artifacts, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir: example,
		NoLLM:      true,
		Extractor:  NewNoopExtractor(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	if intent.OpenAPI != "openapi/support.yaml" {
		t.Fatalf("openapi = %q", intent.OpenAPI)
	}
	step := intent.Steps[0]
	if step.Operation != "getTicket" {
		t.Fatalf("operation = %q", step.Operation)
	}
	if got := step.With["ticketId"]; got != "inputs.ticketId" {
		t.Fatalf("ticketId binding = %q", got)
	}
}

func TestRunCreatesStepBindFromPriorOutput(t *testing.T) {
	example := t.TempDir()
	input := strings.Join([]string{
		"Fetch a customer and write a draft.",
		"customer_draft",
		"Fetch a customer and write a draft.",
		"no",
		"customerId:string",
		"get_customer",
		"fnct",
		"Fetch the customer.",
		"customerId",
		"",
		"write_draft",
		"fnct",
		"Write the draft.",
		"customerId",
		"get_customer.received_body.id",
		"",
		"draft",
		"write_draft.received_body",
		"sandbox-only",
		"",
		"Sandbox proof runs only",
		"Stop on missing customer data",
		"save",
	}, "\n") + "\n"

	artifacts, err := Run(context.Background(), strings.NewReader(input), &strings.Builder{}, Session{}, Options{
		ExampleDir: example,
		NoLLM:      true,
		Extractor:  NewNoopExtractor(),
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	intent, err := rollout.ParseIntent([]byte(artifacts.IntentHCL), "intent.hcl")
	if err != nil {
		t.Fatalf("parse rendered intent: %v\n%s", err, artifacts.IntentHCL)
	}
	step := intent.Steps[1]
	if len(step.DependsOn) != 1 || step.DependsOn[0] != "get_customer" {
		t.Fatalf("depends_on = %#v", step.DependsOn)
	}
	if len(step.Binds) != 1 || step.Binds[0].Fields["customerId"] != "received_body.id" {
		t.Fatalf("binds = %#v", step.Binds)
	}
}

func TestSessionNormalizeExplicitPolicyMarkersReplaceSeededProjectValues(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{
			ProjectName: "Policy",
			Credentials: []string{"old_token"},
			Safety:      "Old safety note",
			Fallback:    "Old fallback note",
		},
		Intent:         rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "policy", Description: "Test policy edits"}},
		Credentials:    []string{"new_token"},
		CredentialsSet: true,
		Safety:         "",
		SafetySet:      true,
		Fallback:       "New fallback note",
		FallbackSet:    true,
	}
	session.Normalize()
	if len(session.Project.Credentials) != 1 || session.Project.Credentials[0] != "new_token" {
		t.Fatalf("credentials were not replaced: %#v", session.Project.Credentials)
	}
	if session.Project.Safety != "" || session.Safety != "" {
		t.Fatalf("safety was not cleared: project=%q top=%q", session.Project.Safety, session.Safety)
	}
	if session.Project.Fallback != "New fallback note" || session.Fallback != "New fallback note" {
		t.Fatalf("fallback was not replaced: project=%q top=%q", session.Project.Fallback, session.Fallback)
	}
}

func writeOpenAPI(t *testing.T, example string) {
	t.Helper()
	dir := filepath.Join(example, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir openapi: %v", err)
	}
	data := `openapi: 3.0.0
info:
  title: Support API
  version: "1.0"
paths:
  /tickets/{ticketId}:
    get:
      operationId: getTicket
      summary: Get a support ticket
      parameters:
        - name: ticketId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`
	if err := os.WriteFile(filepath.Join(dir, "support.yaml"), []byte(data), 0o644); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
}
