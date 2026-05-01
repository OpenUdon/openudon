package synthesize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/hcllight/light"
	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/ramen/internal/uwsvalidate"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runner"
	"github.com/genelet/udon/pkg/uwsprofile"
	"github.com/tabilet/uws/uws1"
)

type fakeClient struct{}

func (fakeClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "openapi": "openapi/support.yaml",
  "workflow": {"name": "support_ticket", "description": "Fetch support ticket details."},
  "steps": [
    {"name": "get_ticket", "type": "http", "do": "Fetch the support ticket", "operation": "getTicket", "with": {"ticketId": "ticket_123"}}
  ],
  "outputs": [{"name": "ticket", "from": "get_ticket.received_body"}]
}`, nil
}

type fakeRuntimeOnlyClient struct{}

func (fakeRuntimeOnlyClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "workflow": {"name": "runtime_only", "description": "Render a local report."},
  "inputs": [{"name": "summary", "type": "string", "required": true}],
  "steps": [
    {"name": "render_report", "type": "fnct", "do": "Render the local report", "with": {"summary": "inputs.summary"}}
  ],
  "outputs": [{"name": "report", "from": "render_report.received_body"}]
}`, nil
}

type fakeLoopClient struct{}

type cancelAfterChatClient struct {
	cancel        context.CancelFunc
	generateCalls int
}

func (c *cancelAfterChatClient) Chat(ctx context.Context, messages []rollout.ChatMessage) (string, error) {
	if c.cancel != nil {
		c.cancel()
	}
	return fakeRuntimeOnlyClient{}.Chat(ctx, messages)
}

func (c *cancelAfterChatClient) Generate(ctx context.Context, prompt string) (string, error) {
	c.generateCalls++
	return fakeRuntimeOnlyClient{}.Generate(ctx, prompt)
}

type countingRuntimeOnlyClient struct {
	generateCalls int
}

func (c *countingRuntimeOnlyClient) Chat(ctx context.Context, messages []rollout.ChatMessage) (string, error) {
	return fakeRuntimeOnlyClient{}.Chat(ctx, messages)
}

func (c *countingRuntimeOnlyClient) Generate(ctx context.Context, prompt string) (string, error) {
	c.generateCalls++
	return fakeRuntimeOnlyClient{}.Generate(ctx, prompt)
}

type scriptedCancelContext struct {
	context.Context
	errAfter int
	errCalls int
}

func (c *scriptedCancelContext) Err() error {
	c.errCalls++
	if c.errAfter > 0 && c.errCalls >= c.errAfter {
		return context.Canceled
	}
	if c.Context == nil {
		return nil
	}
	return c.Context.Err()
}

func (fakeLoopClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "workflow": {"name": "customer_summary_loop", "description": "Render a summary for each customer."},
  "inputs": [{"name": "customers", "type": "array", "required": true}],
  "steps": [
    {
      "name": "render_customer_summaries",
      "type": "loop",
      "do": "Render one summary for each customer.",
      "items": "inputs.customers",
      "batch_size": "2",
      "steps": [
        {"name": "render_customer_summary", "type": "fnct", "do": "Render one customer summary.", "with": {"customer": "each.value"}}
      ]
    }
  ],
  "outputs": [{"name": "customer_summaries", "from": "render_customer_summaries"}]
}`, nil
}

type retryWorkflowClient struct {
	chatCalls     int
	generateCalls int
}

func (c *retryWorkflowClient) Chat(ctx context.Context, messages []rollout.ChatMessage) (string, error) {
	c.chatCalls++
	if c.chatCalls == 1 {
		return `{
  "openapi": "openapi/support.yaml",
  "workflow": {"name": "support_ticket", "description": "Fetch support ticket details."},
  "steps": [
    {"name": "get_ticket", "type": "http", "do": "Fetch the support ticket", "operation": "missingTicketOperation", "with": {"ticketId": "ticket_123"}}
  ],
  "outputs": [{"name": "ticket", "from": "get_ticket.received_body"}]
}`, nil
	}
	return fakeClient{}.Chat(ctx, messages)
}

func (c *retryWorkflowClient) Generate(ctx context.Context, prompt string) (string, error) {
	c.generateCalls++
	return fakeClient{}.Generate(ctx, prompt)
}

type badInputSourceClient struct{}

type failingChatClient struct{}

func (failingChatClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return "", context.Canceled
}

func (failingChatClient) Generate(context.Context, string) (string, error) {
	return "", context.Canceled
}

func (badInputSourceClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "openapi": "openapi/support.yaml",
  "workflow": {"name": "support_ticket", "description": "Fetch support ticket details."},
  "inputs": [{"name": "ticketId", "type": "string", "required": true}],
  "steps": [
    {"name": "get_ticket", "type": "http", "do": "Fetch the support ticket", "operation": "getTicket"}
  ],
  "outputs": [{"name": "ticket", "from": "get_ticket.received_body"}]
}`, nil
}

func (badInputSourceClient) Generate(context.Context, string) (string, error) {
	return "```json\n" + `{
  "body": {
    "blocks": [
      {"type": "provider", "body": {"attributes": {"openapi": "openapi/support.yaml"}}}
    ]
  },
  "steps": [
    {
      "type": "http",
      "name": "get_ticket",
      "body": {
        "attributes": {"operation": "getTicket"},
        "blocks": [
          {"type": "request", "body": {"attributes": {"ticketId": "wrong_literal"}}}
        ]
      }
    }
  ]
}` + "\n```", nil
}

type fakeWeatherChainClient struct{}

func (fakeWeatherChainClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "openapi": "openapi/weather.yaml",
  "workflow": {"name": "weather_lookup", "description": "Get weather for Toronto."},
  "steps": [
    {"name": "get_coordinates", "type": "http", "do": "Resolve Toronto, Canada to coordinates", "operation": "direct_get", "with": {"q": "Toronto,CA"}},
    {"name": "get_weather", "type": "http", "do": "Get weather for the resolved coordinates", "operation": "getWeatherData", "depends_on": ["get_coordinates"], "with": {"appid": "weather_appid"}, "bind": [{"from": "get_coordinates", "fields": {"lat": "body[0].lat", "lon": "body[0].lon"}}]}
  ],
  "outputs": [{"name": "weather", "from": "get_weather.received_body"}]
}`, nil
}

func (fakeWeatherChainClient) Generate(context.Context, string) (string, error) {
	return "```json\n" + `{
  "body": {
    "blocks": [
      {"type": "provider", "body": {"attributes": {"openapi": "openapi/weather.yaml"}}}
    ]
  },
  "steps": [
    {
      "type": "http",
      "name": "get_coordinates",
      "body": {
        "attributes": {"operation": "direct_get"},
        "blocks": [
          {"type": "request", "body": {"attributes": {"q": "Toronto,CA"}}}
        ]
      }
    },
    {
      "type": "http",
      "name": "get_weather",
      "depends_on": ["get_coordinates"],
      "body": {
        "attributes": {"operation": "getWeatherData"},
        "blocks": [
          {"type": "request", "body": {"attributes": {
            "lat": {"expr": "get_coordinates.received_body[0].lat"},
            "lon": {"expr": "get_coordinates.received_body[0].lon"},
            "appid": "weather_appid"
          }}}
        ]
      }
    }
  ]
}` + "\n```", nil
}

type fakeStructuralSwitchClient struct{}

func (fakeStructuralSwitchClient) Chat(context.Context, []rollout.ChatMessage) (string, error) {
	return `{
  "openapi": "openapi/support.yaml",
  "workflow": {"name": "support_priority_switch", "description": "Fetch a support ticket and route handling by severity."},
  "inputs": [{"name": "ticketId", "type": "string", "required": true}],
  "steps": [
    {"name": "get_ticket", "type": "http", "do": "Fetch support ticket details.", "operation": "getTicket", "with": {"ticketId": "inputs.ticketId"}},
    {
      "name": "route_by_severity",
      "type": "switch",
      "depends_on": ["get_ticket"],
      "cases": [
        {
          "name": "urgent",
          "when": "get_ticket.received_body.severity == \"urgent\"",
          "steps": [
            {"name": "prepare_urgent_result", "type": "fnct", "do": "Prepare urgent handling result.", "bind": [{"from": "get_ticket", "fields": {"ticket": "received_body"}}]}
          ]
        }
      ],
      "default": {
        "steps": [
          {"name": "prepare_standard_result", "type": "fnct", "do": "Prepare standard handling result.", "bind": [{"from": "get_ticket", "fields": {"ticket": "received_body"}}]}
        ]
      }
    }
  ],
  "outputs": [{"name": "routing_result", "from": "route_by_severity"}]
}`, nil
}

func (fakeStructuralSwitchClient) Generate(context.Context, string) (string, error) {
	return "```json\n" + `{
  "body": {
    "blocks": [
      {"type": "provider", "body": {"attributes": {"openapi": "openapi/support.yaml"}}}
    ]
  },
  "steps": [
    {
      "type": "http",
      "name": "get_ticket",
      "body": {
        "attributes": {"operation": "getTicket"},
        "blocks": [
          {"type": "request", "body": {"attributes": {"ticketId": {"expr": "inputs.ticketId"}}}}
        ]
      }
    },
    {
      "type": "switch",
      "name": "route_by_severity",
      "depends_on": ["get_ticket"],
      "cases": [
        {
          "name": "urgent",
          "when": {"expr": "get_ticket.received_body.severity == \"urgent\""},
          "steps": [
            {
              "type": "fnct",
              "name": "prepare_urgent_result",
              "body": {"attributes": {"function": "identity", "arguments": [{"expr": "get_ticket.received_body"}]}}
            }
          ]
        }
      ],
      "default": [
        {
          "type": "fnct",
          "name": "prepare_standard_result",
          "body": {"attributes": {"function": "identity", "arguments": [{"expr": "get_ticket.received_body"}]}}
        }
      ]
    }
  ]
}` + "\n```", nil
}

func (fakeRuntimeOnlyClient) Generate(context.Context, string) (string, error) {
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

func (fakeLoopClient) Generate(context.Context, string) (string, error) {
	return "```json\n" + `{
  "steps": [
    {
      "type": "loop",
      "name": "render_customer_summaries",
      "items": {"expr": "inputs.customers"},
      "batch_size": "2",
      "steps": [
        {
          "type": "fnct",
          "name": "render_customer_summary",
          "body": {
            "attributes": {
              "function": "identity",
              "arguments": [{"expr": "each.value"}]
            }
          }
        }
      ]
    }
  ]
}` + "\n```", nil
}

func (fakeClient) Generate(context.Context, string) (string, error) {
	return "```json\n" + `{
  "body": {
    "blocks": [
      {"type": "provider", "body": {"attributes": {"openapi": "openapi/support.yaml"}}}
    ]
  },
  "steps": [
    {
      "type": "http",
      "name": "get_ticket",
      "body": {
        "attributes": {"operation": "getTicket"},
        "blocks": [
          {"type": "request", "body": {"attributes": {"ticketId": "ticket_123"}}}
        ]
      }
    }
  ]
}` + "\n```", nil
}

func TestRunWritesArtifacts(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "support-email")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Support

When a support ticket is created, fetch the ticket details.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "support.yaml"), []byte(`openapi: 3.0.0
info:
  title: Support API
  version: 1.0.0
servers:
  - url: https://support.example.test
paths:
  /tickets:
    get:
      operationId: listTickets
      responses:
        "200":
          description: ok
  /tickets/{ticketId}:
    get:
      operationId: getTicket
      parameters:
        - name: ticketId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
`), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := Run(context.Background(), Options{
		ExampleDir: example,
		Discoverer: &openapidisco.Discoverer{},
		LLMClient:  fakeClient{},
		ChatClient: fakeClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{result.IntentPath, result.WorkflowPath, result.UWSPath, result.PlanJSONPath, result.PlanMDPath, result.RefinementJSONPath, result.RefinementMDPath, result.ReviewPath, result.SymphonyHandoffPath, result.QualityJSONPath, result.QualityMDPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	workflow, err := os.ReadFile(result.WorkflowPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workflow), `openapi = "openapi/support.yaml"`) {
		t.Fatalf("workflow missing OpenAPI binding:\n%s", workflow)
	}
	review, err := os.ReadFile(result.ReviewPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(review), "Side-effectful execution was skipped") {
		t.Fatalf("review missing execution boundary:\n%s", review)
	}
	if !strings.Contains(string(review), "Expected plan") {
		t.Fatalf("review missing expected plan reference:\n%s", review)
	}
	if !strings.Contains(string(review), "Side-Effect Summary") || !strings.Contains(string(review), "Unresolved Risks") || !strings.Contains(string(review), "Trusted proof run") {
		t.Fatalf("review missing hardened audit evidence:\n%s", review)
	}
	handoffData, err := os.ReadFile(result.SymphonyHandoffPath)
	if err != nil {
		t.Fatal(err)
	}
	var handoff SymphonyHandoff
	if err := json.Unmarshal(handoffData, &handoff); err != nil {
		t.Fatal(err)
	}
	if handoff.Version != symphonyHandoffVersion || handoff.GeneratedState != "generated" {
		t.Fatalf("unexpected Symphony handoff metadata: %#v", handoff)
	}
	if !symphonyHandoffHasApprovalStates(handoff) || !symphonyHandoffHasRequiredInputs(handoff) {
		t.Fatalf("Symphony handoff missing required contract: %#v", handoff)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("quality did not pass: %#v", report.Checks)
	}
}

func TestSideEffectPolicyRequiresApprovalOrTrustedRuntime(t *testing.T) {
	policy := analyzeProject(`# Email

## Function Contracts

- send_email
  - Inputs: to, subject, body
  - Outputs: status
  - Side effects: sends email

## Safety and Approval Boundary

- Generate and validate artifacts only.
`)
	report := &QualityReport{}
	assessSideEffectPolicy(report, policy, &rollout.Intent{Steps: []*rollout.Step{{
		Name: "send_email",
		Type: "fnct",
		Do:   "Send email.",
	}}})
	if !hasCheck(report, "side_effects.policy", "fail") {
		t.Fatalf("expected side_effects.policy failure, got %#v", report.Checks)
	}
}

func TestSideEffectPolicyRequiresSandboxProofRun(t *testing.T) {
	policy := analyzeProject(`# Email

## Function Contracts

- send_email
  - Inputs: to, subject, body
  - Outputs: status
  - Side effects: sends email through approved trusted runtime path.

## Safety and Approval Boundary

- Sending email requires approved trusted runner execution.
`)
	report := &QualityReport{}
	assessSideEffectPolicy(report, policy, &rollout.Intent{Steps: []*rollout.Step{{
		Name: "send_email",
		Type: "fnct",
		Do:   "Send email.",
	}}})
	if !hasCheck(report, "side_effects.policy", "fail") {
		t.Fatalf("expected side_effects.policy failure without sandbox proof-run policy, got %#v", report.Checks)
	}
	if !strings.Contains(report.Checks[len(report.Checks)-1].Detail, "sandbox/test proof-run policy") {
		t.Fatalf("expected sandbox detail, got %#v", report.Checks)
	}
}

func TestSideEffectPolicyPassesWithApprovalAndSandboxProofRun(t *testing.T) {
	policy := analyzeProject(`# Email

## Function Contracts

- send_email
  - Inputs: to, subject, body
  - Outputs: status
  - Side effects: sends email through approved trusted runtime path.

## Safety and Approval Boundary

- Sending email requires approved trusted runner execution.
- Use sandbox email endpoints for proof runs before production handoff.
`)
	report := &QualityReport{}
	assessSideEffectPolicy(report, policy, &rollout.Intent{Steps: []*rollout.Step{{
		Name: "send_email",
		Type: "fnct",
		Do:   "Send email.",
	}}})
	if !hasCheck(report, "side_effects.policy", "pass") {
		t.Fatalf("expected side_effects.policy pass, got %#v", report.Checks)
	}
}

func TestSideEffectPolicyIgnoresNoSideEffectsAndDeploymentStatusRead(t *testing.T) {
	report := &QualityReport{}
	policy := analyzeProject(`# Report

## Function Contracts

- render_report
  - Inputs: summary
  - Outputs: report
  - Side effects: none.

## Runtime Policy

- fnct is allowed.

## Safety and Approval Boundary

- Generate and validate artifacts only.
`)
	assessSideEffectPolicy(report, policy, &rollout.Intent{Steps: []*rollout.Step{{
		Name: "render_report",
		Type: "fnct",
		Do:   "Render deployment status report.",
	}}})
	if !hasCheck(report, "side_effects.policy", "pass") {
		t.Fatalf("expected no side-effect policy pass, got %#v", report.Checks)
	}
}

func TestSideEffectProfileDetectsOpenAPIWriteOperation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ticket-write.yaml")
	if err := os.WriteFile(path, []byte(ticketWriteOpenAPI("https://support.example.test")), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := sideEffectProfileForOpenAPI(analyzeProject(`# Tickets

## Safety and Approval Boundary

- Generate and validate artifacts only.
`), &rollout.Intent{
		OpenAPI: "openapi/ticket-write.yaml",
		Steps: []*rollout.Step{{
			Name:      "create_ticket",
			Type:      "http",
			Operation: "createTicket",
		}},
	}, []openapidisco.Candidate{{Path: path, RelativePath: "openapi/ticket-write.yaml"}}, "")
	if !profile.SideEffectful || !strings.Contains(strings.Join(profile.Reasons, "; "), "POST") {
		t.Fatalf("expected OpenAPI write side-effect profile, got %#v", profile)
	}
	report := &QualityReport{}
	assessSideEffectProfile(report, profile)
	if !hasCheck(report, "side_effects.policy", "fail") {
		t.Fatalf("expected side-effect policy failure, got %#v", report.Checks)
	}
}

func TestSideEffectProfileRequiresProductionHandoffForProductionEndpoint(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ticket-prod.yaml")
	if err := os.WriteFile(path, []byte(ticketWriteOpenAPI("https://prod.support.company.com")), 0o644); err != nil {
		t.Fatal(err)
	}
	profile := sideEffectProfileForOpenAPI(analyzeProject(`# Tickets

## Safety and Approval Boundary

- Generate and validate artifacts only.
`), &rollout.Intent{
		OpenAPI: "openapi/ticket-prod.yaml",
		Steps: []*rollout.Step{{
			Name:      "create_ticket",
			Type:      "http",
			Operation: "createTicket",
		}},
	}, []openapidisco.Candidate{{Path: path, RelativePath: "openapi/ticket-prod.yaml"}}, "")
	report := &QualityReport{}
	assessSideEffectProfile(report, profile)
	if !hasCheck(report, "side_effects.environment", "fail") {
		t.Fatalf("expected production environment failure, got %#v profile=%#v", report.Checks, profile)
	}
}

func TestValidateIntentFunctionContractsRequiresDeclaredFnctStep(t *testing.T) {
	policy := analyzeProject(`# Report

## Function Contracts

- render_report
  - Inputs: summary
  - Outputs: report
  - Side effects: none.
`)
	err := validateIntentFunctionContracts(&rollout.Intent{Steps: []*rollout.Step{{
		Name: "format_report",
		Type: "fnct",
		With: map[string]string{"summary": "inputs.summary"},
	}}}, policy)
	if err == nil || !strings.Contains(err.Error(), "missing Function Contracts entries") {
		t.Fatalf("expected missing function contract error, got %v", err)
	}
}

func TestValidateIntentFunctionContractsRejectsUnexpectedFnctStep(t *testing.T) {
	policy := analyzeProject(`# List

## Function Contracts

- No function steps are expected.
`)
	err := validateIntentFunctionContracts(&rollout.Intent{Steps: []*rollout.Step{{
		Name: "render_report",
		Type: "fnct",
	}}}, policy)
	if err == nil || !strings.Contains(err.Error(), "no function steps are expected") {
		t.Fatalf("expected no-function-steps error, got %v", err)
	}
}

func TestValidateIntentFunctionContractsRejectsUndeclaredSimpleInput(t *testing.T) {
	policy := analyzeProject(`# Email

## Function Contracts

- send_email
  - Inputs: to, subject, body
  - Outputs: status
  - Side effects: sends email through approved runtime.
`)
	err := validateIntentFunctionContracts(&rollout.Intent{Steps: []*rollout.Step{{
		Name: "send_email",
		Type: "fnct",
		With: map[string]string{
			"to":      "get_ticket.received_body.email",
			"subject": "get_ticket.received_body.subject",
			"body":    "get_ticket.received_body.summary",
			"bcc":     "audit@example.test",
		},
	}}}, policy)
	if err == nil || !strings.Contains(err.Error(), "fnct inputs not declared by contract") {
		t.Fatalf("expected undeclared fnct input error, got %v", err)
	}
}

func TestValidateIntentFunctionContractsAcceptsNaturalLanguageInputsWithBindingEvidence(t *testing.T) {
	policy := analyzeProject(`# Export

## Function Contracts

- merge_customer_pages
  - Inputs: customer arrays from page 1 and page 2.
  - Outputs: combined payload
  - Side effects: none.
`)
	err := validateIntentFunctionContracts(&rollout.Intent{Steps: []*rollout.Step{{
		Name:      "merge_customer_pages",
		Type:      "fnct",
		DependsOn: []string{"list_page_1", "list_page_2"},
		Binds: []*rollout.StepBind{{
			From:   "list_page_1",
			Fields: map[string]string{"page_1": "received_body.customers"},
		}},
	}}}, policy)
	if err != nil {
		t.Fatalf("unexpected function contract error: %v", err)
	}
}

func TestValidateIntentDataFlowSourcesRejectsUnknownReferences(t *testing.T) {
	err := validateIntentDataFlowSources(&rollout.Intent{
		Inputs: []*rollout.Input{{Name: "ticketId"}},
		Steps: []*rollout.Step{{
			Name:      "get_ticket",
			Type:      "http",
			DependsOn: []string{"missing_step"},
			With:      map[string]string{"ticketId": "missing_step.received_body.id"},
		}},
		Outputs: []*rollout.Output{{Name: "ticket", From: "get_ticket.received_body"}},
	})
	if err == nil || !strings.Contains(err.Error(), "missing_step") {
		t.Fatalf("expected unresolved data-flow source error, got %v", err)
	}
}

func TestValidateIntentDataFlowSourcesAllowsLiteralDomains(t *testing.T) {
	err := validateIntentDataFlowSources(&rollout.Intent{
		Steps: []*rollout.Step{{
			Name: "send_email",
			Type: "fnct",
			With: map[string]string{
				"to":       "audit@example.test",
				"callback": "https://api.example.test/callback",
			},
		}},
	})
	if err != nil {
		t.Fatalf("literal domains should not be treated as step references: %v", err)
	}
}

func TestValidateIntentResponsePathsRejectsAbsentSchemaField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "support.yaml")
	if err := os.WriteFile(path, []byte(supportOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	result := validateIntentResponsePaths(&rollout.Intent{
		OpenAPI: "openapi/support.yaml",
		Steps: []*rollout.Step{
			{Name: "get_ticket", Type: "http", Operation: "getTicket", With: map[string]string{"ticketId": "inputs.ticketId"}},
			{Name: "render", Type: "fnct", With: map[string]string{"email": "get_ticket.received_body.requesterEmail"}},
		},
	}, []openapidisco.Candidate{{Path: path, RelativePath: "openapi/support.yaml"}}, "")
	if len(result.Failures) == 0 || !strings.Contains(result.Failures[0], "requesterEmail") {
		t.Fatalf("expected missing response path failure, got %#v", result)
	}
}

func TestValidateIntentResponsePathsWarnsOnOpaqueSchema(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weather.yaml")
	if err := os.WriteFile(path, []byte(weatherOnlyOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	result := validateIntentResponsePaths(&rollout.Intent{
		OpenAPI: "openapi/weather.yaml",
		Steps: []*rollout.Step{
			{Name: "get_weather", Type: "http", Operation: "getWeatherData", With: map[string]string{"lat": "43.6532", "lon": "-79.3832", "appid": "weather_appid"}},
			{Name: "render", Type: "fnct", With: map[string]string{"summary": "get_weather.received_body.summary"}},
		},
	}, []openapidisco.Candidate{{Path: path, RelativePath: "openapi/weather.yaml"}}, "")
	if len(result.Failures) != 0 || len(result.Warnings) == 0 {
		t.Fatalf("expected opaque schema warning without failure, got %#v", result)
	}
}

func TestValidateIntentOpenAPISecurityRequiresCredentialPolicy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secure.yaml")
	if err := os.WriteFile(path, []byte(secureOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		OpenAPI: "openapi/secure.yaml",
		Steps: []*rollout.Step{{
			Name:      "list_secure_tickets",
			Type:      "http",
			Operation: "listSecureTickets",
			With: map[string]string{
				"api_key": "support_api_key",
			},
		}},
	}
	candidates := []openapidisco.Candidate{{Path: path, RelativePath: "openapi/secure.yaml"}}
	if err := validateIntentOpenAPISecurity(intent, candidates, "", analyzeProject("")); err == nil || !strings.Contains(err.Error(), "Credentials and Secrets") {
		t.Fatalf("expected missing credential policy error, got %v", err)
	}
	policy := analyzeProject(`# Secure

## Credentials and Secrets

- Use credential binding support_api_key.
`)
	if err := validateIntentOpenAPISecurity(intent, candidates, "", policy); err != nil {
		t.Fatalf("unexpected security credential error: %v", err)
	}
}

func TestBuildWorkflowPlanRecordsOpenAPISecurityCredential(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secure.yaml")
	if err := os.WriteFile(path, []byte(secureOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	plan := buildWorkflowPlan(Result{ExampleDir: t.TempDir()}, &rollout.Intent{
		OpenAPI: "openapi/secure.yaml",
		Steps: []*rollout.Step{{
			Name:      "list_secure_tickets",
			Type:      "http",
			Operation: "listSecureTickets",
			With:      map[string]string{"api_key": "support_api_key"},
		}},
	}, []openapidisco.Candidate{{Path: path, RelativePath: "openapi/secure.yaml"}}, analyzeProject(`## Credentials and Secrets

- Use credential binding support_api_key.
`))
	if len(plan.Steps) != 1 || !containsString(plan.Steps[0].Credentials, "api_key") {
		t.Fatalf("expected security credential in plan, got %#v", plan.Steps)
	}
	var found bool
	for _, param := range plan.Steps[0].RequestParams {
		if param.Name == "api_key" && param.Credential && param.ExpectedCredential == "support_api_key" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected security request param with binding, got %#v", plan.Steps[0].RequestParams)
	}
}

func TestWorkflowPlanAllowsFnctBindingsWithoutRequestBlock(t *testing.T) {
	if bindingRequestEvidenceRequired(PlanStep{Type: "fnct", Runtime: "fnct"}) {
		t.Fatal("fnct transform bindings should not require request-block evidence")
	}
}

func TestReviewEvidenceRecordsSideEffectSummary(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "side-effect-review")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	result := resultPaths(example)
	if err := os.WriteFile(result.ProjectPath, []byte(`# Email

## Function Contracts

- send_email
  - Inputs: to, subject, body
  - Outputs: status
  - Side effects: sends email through approved trusted runtime path.

## Safety and Approval Boundary

- Generate and validate artifacts only.
- Use sandbox email endpoints for proof runs before production handoff.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.IntentPath, []byte(`workflow {
  name = "email"
}

step "send_email" {
  type = "fnct"
  do   = "Send email."
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	md := reviewMarkdown(result, "fake", "model")
	for _, expected := range []string{
		"Side-Effect Summary",
		"Side-effectful workflow: yes",
		"Approval/trusted-runtime policy: present",
		"Approval State Requirements",
		"`validated`",
		"`review_required`",
		"`approved_for_sandbox`",
		"`approved_for_production`",
		"`rejected`",
		"Credential Binding Audit",
		"No credential bindings declared or required.",
		"Unresolved Risks",
		"Minimum Review Package",
		"Quality report",
		"Symphony handoff manifest",
		"Credential binding audit",
		"Direct production execution: not performed by Ramen synthesis",
		"Trusted Execution Handoff",
		"Trusted proof run",
	} {
		if !strings.Contains(md, expected) {
			t.Fatalf("review missing %q:\n%s", expected, md)
		}
	}
}

func TestReviewEvidenceRecordsCredentialInventory(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "credential-review")
	if err := os.MkdirAll(filepath.Join(example, "expected"), 0o755); err != nil {
		t.Fatal(err)
	}
	result := resultPaths(example)
	if err := os.WriteFile(result.ProjectPath, []byte(`# Inventory

## Credentials and Secrets

- Use credential binding inventory_api_key.

## Safety and Approval Boundary

- Generate and validate artifacts only.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.PlanJSONPath, []byte(`{
  "steps": [
    {
      "name": "get_inventory",
      "credentials": ["inventory_api_key"],
      "request_params": [
        {
          "name": "api_key",
          "credential": true,
          "source_kind": "credential",
          "expected_credential": "inventory_api_key"
        }
      ]
    }
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	md := reviewMarkdown(result, "", "")
	for _, expected := range []string{
		"Declared credential bindings: `inventory_api_key`",
		"Expected plan credential bindings: `inventory_api_key`",
		"Credential values must stay outside prompts",
	} {
		if !strings.Contains(md, expected) {
			t.Fatalf("review missing %q:\n%s", expected, md)
		}
	}
}

func TestAssessReviewRequiresTrustedExecutionPackage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.md")
	if err := os.WriteFile(path, []byte(`# Ramen Review Evidence

## Side-Effect Summary

- Side-effectful workflow: yes

## Unresolved Risks

- No unresolved execution-boundary risks detected by deterministic review.

Side-effectful execution was skipped.

Trusted proof run:

`+"```bash\n./scripts/run-udon.sh workflows/workflow.hcl example\n```\n"+`
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report := &QualityReport{}
	assessReview(report, path, sideEffectProfile{SideEffectful: true}, projectPolicy{}, nil)
	if !hasCheck(report, "review.package", "fail") {
		t.Fatalf("expected review.package failure, got %#v", report.Checks)
	}
}

func TestAssessReviewRequiresApprovalStateEvidence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.md")
	if err := os.WriteFile(path, []byte(validReviewEvidenceText(false, false)), 0o644); err != nil {
		t.Fatal(err)
	}
	report := &QualityReport{}
	assessReview(report, path, sideEffectProfile{SideEffectful: true}, projectPolicy{}, nil)
	if !hasCheck(report, "review.approval_states", "fail") {
		t.Fatalf("expected review.approval_states failure, got %#v", report.Checks)
	}
}

func TestAssessReviewRequiresCredentialBindingInventory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.md")
	if err := os.WriteFile(path, []byte(validReviewEvidenceText(true, false)), 0o644); err != nil {
		t.Fatal(err)
	}
	report := &QualityReport{}
	assessReview(report, path, sideEffectProfile{}, analyzeProject(`# Credentials

## Credentials and Secrets

- Use credential binding support_api_token.
`), nil)
	if !hasCheck(report, "review.credential_bindings", "fail") {
		t.Fatalf("expected review.credential_bindings failure, got %#v", report.Checks)
	}
}

func TestAssessReviewAcceptsNoCredentialAuditPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review.md")
	if err := os.WriteFile(path, []byte(validReviewEvidenceText(true, true)), 0o644); err != nil {
		t.Fatal(err)
	}
	report := &QualityReport{}
	assessReview(report, path, sideEffectProfile{}, projectPolicy{}, nil)
	for _, code := range []string{"review.approval_states", "review.sandbox_handoff", "review.credential_bindings"} {
		if !hasCheck(report, code, "pass") {
			t.Fatalf("expected %s pass, got %#v", code, report.Checks)
		}
	}
}

func validReviewEvidenceText(includeApprovalStates, includeCredentialInventory bool) string {
	var b strings.Builder
	b.WriteString(`# Ramen Review Evidence

## Minimum Review Package

- Project brief
- Intent HCL
- Workflow HCL
- UWS artifact
- Expected plan
- Quality report
- Refinement report
- Review evidence
- Symphony handoff manifest

## Side-Effect Summary

- Side-effectful workflow: no side-effectful behavior inferred from project policy or intent steps.
- Credential binding audit: runtime binding names only; literal secrets are prohibited in prompts, examples, and artifacts.
- Direct production execution: not performed by Ramen synthesis.

`)
	if includeApprovalStates {
		b.WriteString(`## Approval State Requirements

- Ramen emitted state: ` + "`generated`" + `; no approval is implied by artifact generation.
- ` + "`validated`" + `: required validators and quality gates have passed or known warnings are attached.
- ` + "`review_required`" + `: human review is required before side-effectful execution.
- ` + "`approved_for_sandbox`" + `: sandbox or test-endpoint execution only.
- ` + "`approved_for_production`" + `: production execution through a trusted runner and approved credentials.
- ` + "`rejected`" + `: artifact rejected or regeneration requested.
- ` + "`approved_for_sandbox`" + ` and ` + "`approved_for_production`" + ` are not required unless future changes add side effects.

`)
	}
	b.WriteString(`## Credential Binding Audit

`)
	if includeCredentialInventory {
		b.WriteString("- No credential bindings declared or required.\n")
	}
	b.WriteString(`
## Unresolved Risks

- No unresolved execution-boundary risks detected by deterministic review.

## Validation

- Side-effectful execution was skipped.

## Trusted Execution Handoff

- Direct production execution: not performed by Ramen synthesis.
- Sandbox/test proof run is optional unless future changes add side effects.

Trusted proof run:

` + "```bash\n./scripts/run-udon.sh workflows/workflow.hcl example\n```\n")
	return b.String()
}

func TestAssessReportsMissingOpenAPI(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "missing-openapi")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte("Call a missing API."), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if report.Passed() {
		t.Fatalf("quality should fail")
	}
	var found bool
	for _, check := range report.Checks {
		if check.Code == "openapi.local" && check.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing openapi.local failure in %#v", report.Checks)
	}
	if _, err := os.Stat(filepath.Join(example, "expected", "quality.json")); err != nil {
		t.Fatalf("quality.json not written: %v", err)
	}
}

func TestAssessAllowsExplicitNoOpenAPI(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "runtime-only")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a local report.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed for report rendering.

## Safety and Approval Boundary

- Generate and validate only.

## Fallback Behavior

- Stop if no approved function runtime exists.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "openapi.local", "pass") {
		t.Fatalf("expected openapi.local pass, got %#v", report.Checks)
	}
}

func TestAssessAllowsRequiredRuntimeInputWithoutDefault(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "runtime-input")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Input

## Goal

Render a local summary report.

## Inputs

- summary: required string.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Data Flow

- Pass inputs.summary to render_report.summary.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate only.

## Fallback Behavior

- Stop if no approved function runtime exists.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(`workflow {
  name = "runtime_input"
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
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "intent.parse", "pass") {
		t.Fatalf("expected intent.parse pass, got %#v", report.Checks)
	}
	if !hasCheck(report, "intent.slots", "pass") {
		t.Fatalf("expected required runtime input without default to pass intent.slots, got %#v", report.Checks)
	}
}

func TestAssessReportsProjectAuthoringWarnings(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "weak-brief")
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte("Call an API."), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "project.authoring.runtime_policy", "warn") {
		t.Fatalf("expected runtime policy warning, got %#v", report.Checks)
	}
}

func TestAnalyzeProjectReadsStructuredPolicyBlock(t *testing.T) {
	policy := analyzeProject(`# Runtime Project

` + "```ramen-policy" + `
openapi: none required
runtimes:
  cmd: approved
  ssh: false
credential_bindings:
  - weather_api_key
` + "```" + `
`)
	if !policy.NoOpenAPI {
		t.Fatalf("expected structured no-openapi policy")
	}
	if !policy.AllowedRuntime["cmd"] {
		t.Fatalf("expected structured cmd runtime allowance")
	}
	if policy.AllowedRuntime["ssh"] {
		t.Fatalf("did not expect ssh runtime allowance")
	}
	if !strings.Contains(policy.CredentialSection, "weather_api_key") {
		t.Fatalf("expected structured credential binding, got %q", policy.CredentialSection)
	}
}

func TestAssessRejectsOpenAPIRefsWhenNoOpenAPIRequired(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "bad-runtime-only")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a report.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop on unsupported integration.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(`openapi = "openapi/bad.yaml"

workflow {
  name = "bad"
}

step "bad_api" {
  type = "http"
  do = "Call an API"
  operation = "badOperation"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "intent.openapi_refs", "fail") {
		t.Fatalf("expected intent.openapi_refs failure, got %#v", report.Checks)
	}
}

func TestAssessRejectsUnsupportedRuntimeType(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "bad-runtime")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Bad Runtime

## Goal

Query a database.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop on unsupported runtime.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(`workflow {
  name = "bad"
}

step "query_database" {
  type = "sql"
  do   = "Query a database"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "intent.runtime_policy", "fail") {
		t.Fatalf("expected intent.runtime_policy failure, got %#v", report.Checks)
	}
}

func TestAssessReportsDiscoveryFailures(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "discovery-report")
	if err := os.MkdirAll(filepath.Join(example, "expected"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a local report.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop on unsupported integration.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "expected", "discovery.json"), []byte(`{
  "attempts": [
    {"kind": "url", "source": "https://example.test/openapi.yaml", "status": "fail", "detail": "download OpenAPI document: 404 Not Found"}
  ]
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "openapi.discovery", "warn") {
		t.Fatalf("expected openapi.discovery warning, got %#v", report.Checks)
	}
}

func TestDefaultSchemaPathUsesRepoSiblingSchema(t *testing.T) {
	path := defaultSchemaPath(filepath.Join(t.TempDir(), "external", "example"))
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("default schema path %s is not readable: %v", path, err)
	}
	if !strings.Contains(filepath.ToSlash(path), "/uws/versions/1.0.0.json") {
		t.Fatalf("unexpected schema path %s", path)
	}
}

func TestSynthesizeRuntimeOnlyProject(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "runtime-only")
	writeRuntimeOnlyExample(t, example)
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeRuntimeOnlyClient{},
		ChatClient: fakeRuntimeOnlyClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.PrimaryOpenAPI != "" {
		t.Fatalf("PrimaryOpenAPI = %q, want empty", result.PrimaryOpenAPI)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("quality did not pass: %#v", report.Checks)
	}
}

func TestSynthesizeReturnsRefinementReportWriteError(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "refinement-write-error")
	writeRuntimeOnlyExample(t, example)
	if err := os.MkdirAll(filepath.Join(example, "expected", "refinement.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeRuntimeOnlyClient{},
		ChatClient: fakeRuntimeOnlyClient{},
		SchemaPath: schemaPath,
	})
	if err == nil || !strings.Contains(err.Error(), "write refinement report") {
		t.Fatalf("expected write refinement report error, got %v", err)
	}
}

func TestSynthesizeCanceledContextStopsBeforeWorkflowGeneration(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "cancel-synthesize")
	writeRuntimeOnlyExample(t, example)
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	client := &cancelAfterChatClient{cancel: cancel}
	result, err := Synthesize(ctx, Options{
		ExampleDir: example,
		LLMClient:  client,
		ChatClient: client,
		SchemaPath: schemaPath,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got result=%#v err=%v", result, err)
	}
	if client.generateCalls != 0 {
		t.Fatalf("workflow generation was called %d time(s)", client.generateCalls)
	}
	if _, statErr := os.Stat(filepath.Join(example, "workflows", "workflow.hcl")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("workflow.hcl should not be written after cancellation, stat err=%v", statErr)
	}
}

func TestBuildCanceledContextStopsBeforeWorkflowGeneration(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "cancel-build")
	writeRuntimeOnlyExample(t, example)
	intentHCL, err := runner.RenderIntentHCL(runtimeOnlyIntent())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(intentHCL), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	ctx := &scriptedCancelContext{Context: context.Background(), errAfter: 3}
	client := &countingRuntimeOnlyClient{}
	result, err := Build(ctx, Options{
		ExampleDir: example,
		LLMClient:  client,
		ChatClient: client,
		SchemaPath: schemaPath,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got result=%#v err=%v", result, err)
	}
	if client.generateCalls != 0 {
		t.Fatalf("workflow generation was called %d time(s)", client.generateCalls)
	}
	if _, statErr := os.Stat(filepath.Join(example, "workflows", "workflow.hcl")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("workflow.hcl should not be written after cancellation, stat err=%v", statErr)
	}
}

func TestSynthesizePreservesLoopAndStructuralResult(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "customer-loop")
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Customer Loop

## Goal

Render one summary for each customer in the runtime input list.

## Inputs

- customers: required array.

## Outputs

- customer_summaries: accumulated loop output.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- loop and fnct are allowed for trusted local rendering.
- cmd and ssh are not allowed.

## Function Contracts

- render_customer_summary
  - Inputs: customer.
  - Outputs: one rendered customer summary.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if loop artifacts cannot be validated.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeLoopClient{},
		ChatClient: fakeLoopClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []struct {
		path     string
		contains []string
	}{
		{result.IntentPath, []string{`step "render_customer_summaries"`, `type       = "loop"`, `items      = "inputs.customers"`, `batch_size = "2"`}},
		{result.WorkflowPath, []string{`loop "render_customer_summaries"`, `items = inputs.customers`, `batch_size = "2"`, `fnct "render_customer_summary"`}},
		{result.PlanJSONPath, []string{`"name": "customer_summaries"`, `"kind": "loop"`, `"from": "main.render_customer_summaries"`}},
		{result.ReviewPath, []string{`render_customer_summaries`, "items: `inputs.customers`", "batch_size: `2`"}},
	} {
		data, err := os.ReadFile(artifact.path)
		if err != nil {
			t.Fatal(err)
		}
		for _, expected := range artifact.contains {
			if !strings.Contains(string(data), expected) {
				t.Fatalf("%s missing %q:\n%s", artifact.path, expected, data)
			}
		}
	}
	doc, err := uwsprofile.LoadDocumentFile(result.UWSPath, uwsprofile.DocumentFormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Results) != 1 || doc.Results[0].Name != "customer_summaries" || doc.Results[0].Kind != "loop" || doc.Results[0].From != "main.render_customer_summaries" {
		t.Fatalf("unexpected structural results: %#v", doc.Results)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("quality did not pass: %#v", report.Checks)
	}
	if !hasCheck(report, "uws.structural_results", "pass") {
		t.Fatalf("expected structural result quality pass, got %#v", report.Checks)
	}
}

func TestUWSFailureActionsAndRetriesRemainCompatible(t *testing.T) {
	doc := &uws1.Document{
		UWS: "1.0.0",
		Info: &uws1.Info{
			Title:   "retry compatibility",
			Version: "1.0.0",
		},
		SourceDescriptions: []*uws1.SourceDescription{{
			Name: "support_api",
			Type: "openapi",
			URL:  "openapi/support.yaml",
		}},
		Operations: []*uws1.Operation{{
			OperationID:        "fetch_ticket",
			SourceDescription:  "support_api",
			OpenAPIOperationID: "getTicket",
			Request: map[string]any{
				"path": map[string]any{"ticketId": "inputs.ticketId"},
			},
			SuccessCriteria: []*uws1.Criterion{{
				Condition: "$response.statusCode == 200",
			}},
			OnFailure: []*uws1.FailureAction{
				{Name: "retry_5xx", Type: "retry", RetryAfter: 2, RetryLimit: 3, Criteria: []*uws1.Criterion{{Condition: "$response.statusCode >= 500"}}},
				{Name: "route_failure", Type: "goto", WorkflowID: "failure_handler"},
			},
		}},
		Workflows: []*uws1.Workflow{
			{
				WorkflowID: "main",
				Type:       uws1.WorkflowTypeSequence,
				Steps:      []*uws1.Step{{StepID: "fetch_ticket", OperationRef: "fetch_ticket"}},
			},
			{
				WorkflowID: "failure_handler",
				Type:       uws1.WorkflowTypeSequence,
			},
		},
	}
	if err := uwsprofile.ValidateForExecution(doc); err != nil {
		t.Fatalf("failure actions should be executable-profile compatible: %v", err)
	}
	data, err := uwsprofile.MarshalDocument(doc, uwsprofile.DocumentFormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	data, err = pruneEmptyUWSStepTypes(data)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "workflow.uws.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := uwsvalidate.ValidateFile(schemaPath, path); err != nil {
		t.Fatalf("failure actions should validate against public UWS schema: %v\n%s", err, data)
	}
}

func TestValidateUWSStructuralResultsReportsMissingResult(t *testing.T) {
	err := validateUWSStructuralResults(&uws1.Document{}, []PlanResult{{
		Name:  "customer_summaries",
		Kind:  "loop",
		From:  "main.render_customer_summaries",
		Value: "render_customer_summaries",
	}})
	if err == nil || !strings.Contains(err.Error(), "customer_summaries") {
		t.Fatalf("expected missing structural result error, got %v", err)
	}
}

func TestSynthesizeExpandsBusinessGoalIntoOpenAPIChain(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "weather")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Weather Lookup

## Goal

Search weather in Toronto, Canada.

## Inputs

- City and country are provided in the goal.

## Outputs

- Current weather for the requested city.

## Data Flow

- Resolve the city to coordinates before calling the weather endpoint.

## Function Contracts

- No fnct steps are expected.

## External Systems and OpenAPI

- Use the OpenWeather OpenAPI document under openapi/.

## Runtime Policy

- openapi and http are allowed.
- fnct is allowed only for trusted adapters.
- cmd and ssh are not allowed.

## Credentials and Secrets

- Use credential binding names only.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if no geocoding operation is available.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "weather.yaml"), []byte(weatherOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeWeatherChainClient{},
		ChatClient: fakeWeatherChainClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	intent, err := os.ReadFile(result.IntentPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`step "get_coordinates"`, `step "get_weather"`, `from = "get_coordinates"`, `lat = "body[0].lat"`, `lon = "body[0].lon"`} {
		if !strings.Contains(string(intent), expected) {
			t.Fatalf("intent missing %q:\n%s", expected, intent)
		}
	}
	review, err := os.ReadFile(result.ReviewPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(review), "Inferred Steps And Data Flow") || !strings.Contains(string(review), "bind from `get_coordinates`") {
		t.Fatalf("review missing inferred data-flow evidence:\n%s", review)
	}
	plan, err := os.ReadFile(result.PlanJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"name": "get_coordinates"`, `"name": "get_weather"`, `"operation": "getWeatherData"`, `"target": "lat"`, `"target": "lon"`} {
		if !strings.Contains(string(plan), expected) {
			t.Fatalf("plan missing %q:\n%s", expected, plan)
		}
	}
}

func TestSynthesizePreservesStructuralSwitchArtifacts(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "support-priority-switch")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Support Priority Switch

## Goal

Fetch a support ticket and route the internal handling result by severity.

## Inputs

- ticketId: required string.

## Outputs

- routing_result: selected internal handling result.

## External Systems and OpenAPI

- Use openapi/support.yaml for ticket lookup.

## Data Flow

- Pass inputs.ticketId to get_ticket.ticketId.
- If get_ticket.received_body.severity is urgent, prepare an urgent handling result.
- Otherwise prepare a standard handling result.

## Function Contracts

- prepare_urgent_result
  - Inputs: ticket body from get_ticket.
  - Outputs: urgent internal handling result.
  - Side effects: none.
- prepare_standard_result
  - Inputs: ticket body from get_ticket.
  - Outputs: standard internal handling result.
  - Side effects: none.

## Runtime Policy

- openapi and http are allowed.
- fnct is allowed for trusted local routing adapters.
- cmd and ssh are not allowed.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if the ticket cannot be fetched.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "support.yaml"), []byte(supportOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeStructuralSwitchClient{},
		ChatClient: fakeStructuralSwitchClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(result.WorkflowPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`switch "route_by_severity"`, `case "urgent"`, `default`, `fnct "prepare_urgent_result"`, `fnct "prepare_standard_result"`} {
		if !strings.Contains(string(workflow), expected) {
			t.Fatalf("workflow missing %q:\n%s", expected, workflow)
		}
	}
	plan, err := os.ReadFile(result.PlanJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"name": "route_by_severity"`, `"runtime": "switch"`, `"parent": "route_by_severity"`, `"branch": "urgent"`, `"branch": "default"`, `"branch_when": "get_ticket.received_body.severity == \"urgent\""`} {
		if !strings.Contains(string(plan), expected) {
			t.Fatalf("plan missing %q:\n%s", expected, plan)
		}
	}
	review, err := os.ReadFile(result.ReviewPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(review), "branch: `urgent` when `get_ticket.received_body.severity == \"urgent\"`") {
		t.Fatalf("review missing switch branch evidence:\n%s", review)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("quality did not pass: %#v", report.Checks)
	}
}

func TestAssessFailsWhenWorkflowDivergesFromPlan(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "weather-drift")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Weather Lookup

## Goal

Search weather in Toronto, Canada.

## External Systems and OpenAPI

- Use the OpenWeather OpenAPI document under openapi/.

## Data Flow

- Resolve coordinates before weather lookup.

## Runtime Policy

- openapi and http are allowed.

## Credentials and Secrets

- Use runtime credential bindings only.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop if required coordinates cannot be resolved.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "weather.yaml"), []byte(weatherOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeWeatherChainClient{},
		ChatClient: fakeWeatherChainClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(result.WorkflowPath)
	if err != nil {
		t.Fatal(err)
	}
	drifted := strings.Replace(string(workflow), `operation = "getWeatherData"`, `operation = "direct_get"`, 1)
	if drifted == string(workflow) {
		t.Fatalf("test fixture did not contain expected operation:\n%s", workflow)
	}
	if err := os.WriteFile(result.WorkflowPath, []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "workflow.plan_match", "fail") {
		t.Fatalf("expected workflow.plan_match failure, got %#v", report.Checks)
	}
}

func TestQualityFailsWhenExpectedOperationActionsAreMissing(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "action-drift")
	result := resultPaths(example)
	for _, dir := range []string{filepath.Dir(result.IntentPath), filepath.Dir(result.PlanJSONPath)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(result.ProjectPath, []byte(`# Action Drift

OpenAPI: none required

## Runtime Policy

- fnct is allowed.

## Safety and Approval Boundary

- Generate only.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.IntentPath, []byte(`workflow {
  name        = "action_drift"
  description = "Send an email."
}

step "send_email" {
  type = "fnct"
  do   = "Send email."
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.WorkflowPath, []byte(`fnct "send_email" {
  function = "send.Email"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	doc := uwsDocumentWithRuntime(t, "send_email", &uwsprofile.OperationRuntime{Type: "fnct", Function: "send.Email"})
	data, err := uwsprofile.MarshalDocument(doc, uwsprofile.DocumentFormatYAML)
	if err != nil {
		t.Fatal(err)
	}
	data, err = pruneEmptyUWSStepTypes(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.UWSPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeWorkflowPlan(result, &WorkflowPlan{
		Version:  workflowPlanVersion,
		Workflow: "action_drift",
		Steps: []PlanStep{{
			Name:    "send_email",
			Type:    "fnct",
			Runtime: "fnct",
			OnFailure: []*uws1.FailureAction{
				{Name: "retry_once", Type: "retry", RetryLimit: 1},
			},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.ReviewPath, []byte(validReviewEvidenceText(true, true)), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "workflow.plan_match", "fail") {
		t.Fatalf("expected workflow.plan_match failure, got %#v", report.Checks)
	}
	if !hasCheck(report, "uws.operation_actions", "fail") {
		t.Fatalf("expected uws.operation_actions failure, got %#v", report.Checks)
	}
}

func TestSideEffectfulRetryRequiresRetryOrIdempotencyPolicy(t *testing.T) {
	report := &QualityReport{}
	assessSideEffectRetryPolicy(report, sideEffectProfile{SideEffectful: true}, analyzeProject(`# Email

## Safety and Approval Boundary

- Approved trusted runner only.
`), &WorkflowPlan{Steps: []PlanStep{{
		Name: "send_email",
		OnFailure: []*uws1.FailureAction{
			{Name: "retry_once", Type: "retry", RetryLimit: 1},
		},
	}}})
	if !hasCheck(report, "side_effects.retry_policy", "fail") {
		t.Fatalf("expected side_effects.retry_policy failure, got %#v", report.Checks)
	}

	report = &QualityReport{}
	assessSideEffectRetryPolicy(report, sideEffectProfile{SideEffectful: true}, analyzeProject(`# Email

## Safety and Approval Boundary

- Approved trusted runner only.
- The send operation is idempotent and safe to retry with a bounded retry limit.
`), &WorkflowPlan{Steps: []PlanStep{{
		Name: "send_email",
		OnFailure: []*uws1.FailureAction{
			{Name: "retry_once", Type: "retry", RetryLimit: 1},
		},
	}}})
	if !hasCheck(report, "side_effects.retry_policy", "pass") {
		t.Fatalf("expected side_effects.retry_policy pass, got %#v", report.Checks)
	}
}

func uwsDocumentWithRuntime(t *testing.T, operationID string, ext *uwsprofile.OperationRuntime) *uws1.Document {
	t.Helper()
	op := &uws1.Operation{OperationID: operationID}
	op.Extensions = map[string]any{uws1.ExtensionOperationProfile: uwsprofile.ProfileName}
	if err := uwsprofile.SetOperationExtension(&op.Extensions, ext); err != nil {
		t.Fatal(err)
	}
	doc := &uws1.Document{
		UWS:  "1.0.0",
		Info: &uws1.Info{Title: operationID, Version: "1.0.0"},
		Operations: []*uws1.Operation{
			op,
		},
		Workflows: []*uws1.Workflow{{
			WorkflowID: "main",
			Type:       uws1.WorkflowTypeSequence,
			Steps:      []*uws1.Step{{StepID: operationID, OperationRef: operationID}},
		}},
	}
	if err := uwsprofile.ValidateForExecution(doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestValidateIntentOpenAPIOperationsRequiresOperationOnOpenAPIStep(t *testing.T) {
	err := validateIntentOpenAPIOperations(&rollout.Intent{
		OpenAPI: "openapi/weather.yaml",
		Steps: []*rollout.Step{{
			Name: "get_weather",
			Type: "http",
			Do:   "Fetch weather.",
		}},
	}, nil, "")
	if err == nil || !strings.Contains(err.Error(), "missing operation") {
		t.Fatalf("expected missing operation error, got %v", err)
	}
}

func TestBuildWorkflowPlanSkipsCmdCommandHint(t *testing.T) {
	plan := buildWorkflowPlan(Result{ExampleDir: t.TempDir()}, &rollout.Intent{
		Steps: []*rollout.Step{{
			Name: "get_deployment_status",
			Type: "cmd",
			Do:   "Run deployment status command.",
			With: map[string]string{"command": "get_deploy_status.sh"},
		}},
	}, nil, projectPolicy{})
	if len(plan.Steps) != 1 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if len(plan.Steps[0].Bindings) != 0 {
		t.Fatalf("cmd command hint should not become request binding: %#v", plan.Steps[0].Bindings)
	}
}

func TestProjectPolicyTimeoutIdempotencyPromptAndPlan(t *testing.T) {
	project := "```ramen-policy\n" + `
timeouts:
  workflow: 120
  steps:
    call_api: 10
idempotency:
  key: inputs.request_id
  onConflict: returnPrevious
  ttl: 86400
` + "```\n"
	policy := analyzeProject(project)
	prompt := requiredByProjectPrompt(policy)
	for _, want := range []string{"Workflow timeout: 120 seconds", "Step `call_api` timeout: 10 seconds", "Workflow idempotency: key `inputs.request_id`, onConflict `returnPrevious`, ttl `86400` seconds"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("prompt missing %q:\n%s", want, prompt)
		}
	}
	generatedWorkflowTimeout := 60.0
	generatedStepTimeout := 5.0
	generatedTTL := 100.0
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{
			Name:        "controls",
			Description: "Controls",
			Timeout:     &generatedWorkflowTimeout,
			Idempotency: &uws1.Idempotency{Key: "inputs.other_id", OnConflict: "reject", TTL: &generatedTTL},
		},
		Steps: []*rollout.Step{{Name: "call_api", Type: "fnct", Do: "Call API", Timeout: &generatedStepTimeout}},
	}
	applyProjectTimeoutAndIdempotency(intent, policy)
	plan := buildWorkflowPlan(Result{ExampleDir: t.TempDir()}, intent, nil, policy)
	if plan.Timeout == nil || *plan.Timeout != 120 || plan.Idempotency == nil || plan.Idempotency.Key != "inputs.request_id" {
		t.Fatalf("workflow timeout/idempotency policy did not override generated values: %#v", plan)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Timeout == nil || *plan.Steps[0].Timeout != 10 {
		t.Fatalf("step timeout policy did not override generated values: %#v", plan.Steps)
	}
	md := workflowPlanMarkdown(plan)
	if !strings.Contains(md, "Timeout: `120` seconds") || !strings.Contains(md, "timeout `10`s") {
		t.Fatalf("plan markdown missing timeout evidence:\n%s", md)
	}
}

func TestValidateIntentProjectMetadataPolicyRejectsDrift(t *testing.T) {
	expectedWorkflowTimeout := 120.0
	expectedStepTimeout := 10.0
	expectedTTL := 86400.0
	gotWorkflowTimeout := 60.0
	gotStepTimeout := 5.0
	gotTTL := 100.0
	policy := projectPolicy{
		WorkflowTimeout: &expectedWorkflowTimeout,
		StepTimeouts:    map[string]float64{"call_api": expectedStepTimeout},
		Idempotency:     &uws1.Idempotency{Key: "inputs.request_id", OnConflict: "returnPrevious", TTL: &expectedTTL},
	}
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{
			Timeout:     &gotWorkflowTimeout,
			Idempotency: &uws1.Idempotency{Key: "inputs.other_id", OnConflict: "reject", TTL: &gotTTL},
		},
		Steps: []*rollout.Step{{Name: "call_api", Timeout: &gotStepTimeout}},
	}
	err := validateIntentProjectMetadataPolicy(intent, policy)
	if err == nil || !strings.Contains(err.Error(), "workflow timeout") || !strings.Contains(err.Error(), "call_api timeout") || !strings.Contains(err.Error(), "workflow idempotency") {
		t.Fatalf("expected timeout/idempotency drift, got %v", err)
	}
	applyProjectTimeoutAndIdempotency(intent, policy)
	if err := validateIntentProjectMetadataPolicy(intent, policy); err != nil {
		t.Fatalf("expected project metadata policy to pass after override: %v", err)
	}
}

func TestValidateUWSTimeoutAndIdempotency(t *testing.T) {
	workflowTimeout := 120.0
	stepTimeout := 30.0
	opTimeout := 10.0
	ttl := 86400.0
	doc := &uws1.Document{
		UWS:  "1.1.0",
		Info: &uws1.Info{Title: "Controls", Version: "1.0.0"},
		Operations: []*uws1.Operation{{
			OperationID:              "call_api",
			OperationExecutionFields: uws1.OperationExecutionFields{Timeout: &opTimeout},
		}},
		Workflows: []*uws1.Workflow{{
			WorkflowID:  "main",
			Type:        uws1.WorkflowTypeSequence,
			Idempotency: &uws1.Idempotency{Key: "inputs.request_id", OnConflict: "returnPrevious", TTL: &ttl},
			Steps: []*uws1.Step{{
				StepID:              "fanout",
				Type:                uws1.WorkflowTypeLoop,
				StepExecutionFields: uws1.StepExecutionFields{Timeout: &stepTimeout},
				Steps: []*uws1.Step{{
					StepID:       "call_api",
					OperationRef: "call_api",
				}},
			}},
		}},
	}
	doc.Workflows[0].Timeout = &workflowTimeout
	plan := &WorkflowPlan{
		Timeout:     &workflowTimeout,
		Idempotency: &uws1.Idempotency{Key: "inputs.request_id", OnConflict: "returnPrevious", TTL: &ttl},
		Steps: []PlanStep{
			{Name: "fanout", Type: "loop", Runtime: "loop", Timeout: &stepTimeout},
			{Name: "call_api", Type: "fnct", Runtime: "fnct", Timeout: &opTimeout},
		},
	}
	if err := validateUWSTimeoutAndIdempotency(doc, plan); err != nil {
		t.Fatalf("expected timeout/idempotency validation to pass: %v", err)
	}
	caseTimeout := 15.0
	caseDoc := &uws1.Document{
		UWS: "1.1.0",
		Workflows: []*uws1.Workflow{{
			WorkflowID: "main",
			Type:       uws1.WorkflowTypeSwitch,
			Cases: []*uws1.Case{{
				CaseFields: uws1.CaseFields{Name: "matched"},
				Steps: []*uws1.Step{{
					StepID:              "case_step",
					Type:                uws1.WorkflowTypeSequence,
					StepExecutionFields: uws1.StepExecutionFields{Timeout: &caseTimeout},
				}},
			}},
		}},
	}
	casePlan := &WorkflowPlan{Steps: []PlanStep{{Name: "case_step", Type: "sequence", Runtime: "sequence", Timeout: &caseTimeout}}}
	if err := validateUWSTimeoutAndIdempotency(caseDoc, casePlan); err != nil {
		t.Fatalf("expected top-level case timeout validation to pass: %v", err)
	}
	doc.Operations[0].Timeout = nil
	if err := validateUWSTimeoutAndIdempotency(doc, plan); err == nil || !strings.Contains(err.Error(), "call_api operation timeout") {
		t.Fatalf("expected operation timeout mismatch, got %v", err)
	}
}

func TestDeterministicNoOpenAPICommandWorkflow(t *testing.T) {
	hcl, ok := deterministicNoOpenAPICommandWorkflow(&rollout.Intent{
		Steps: []*rollout.Step{{
			Name: "get_deployment_status",
			Type: "cmd",
			Do:   "Run deployment status.",
			With: map[string]string{"command": `echo "Deployment status: OK"`},
		}},
	}, "")
	if !ok {
		t.Fatal("expected deterministic command workflow")
	}
	if strings.Contains(hcl, "openapi") || !strings.Contains(hcl, `cmd "get_deployment_status"`) || !strings.Contains(hcl, `\"Deployment status: OK\"`) {
		t.Fatalf("unexpected workflow HCL:\n%s", hcl)
	}
}

func TestRequestAttributeEvidenceFindsNestedParamMap(t *testing.T) {
	expr, err := light.AnyToExpression(map[string]any{
		"ticketId": map[string]any{"expr": "workflow.input.ticketId"},
	})
	if err != nil {
		t.Fatal(err)
	}
	evidence, ok := requestAttributeEvidence(&light.Body{Attributes: map[string]*light.Attribute{
		"path_pars": {Name: "path_pars", Expr: expr},
	}}, []string{"path_pars.ticketId"})
	if !ok {
		t.Fatal("expected nested path_pars evidence")
	}
	if !strings.Contains(evidence.Expression, "ticketId") {
		t.Fatalf("unexpected evidence: %#v", evidence)
	}
}

func TestRequestAttributeEvidenceMatchingCanPreferCorrectDuplicate(t *testing.T) {
	expr, err := light.AnyToExpression("inputs.ticketId")
	if err != nil {
		t.Fatal(err)
	}
	wrongExpr, err := light.AnyToExpression("trigger.received_body.ticketId")
	if err != nil {
		t.Fatal(err)
	}
	evidence, ok := requestAttributeEvidenceMatching(&light.Body{
		Attributes: map[string]*light.Attribute{
			"ticketId": {Name: "ticketId", Expr: expr},
		},
		Blocks: []*light.Block{{
			Type: "path_pars",
			Bdy: &light.Body{Attributes: map[string]*light.Attribute{
				"ticketId": {Name: "ticketId", Expr: wrongExpr},
			}},
		}},
	}, []string{"path_pars.ticketId", "ticketId"}, func(candidate requestAttribute) bool {
		return expressionReferencesInputSource(candidate.Expression, "inputs.ticketId")
	})
	if !ok || evidence.Name != "ticketId" {
		t.Fatalf("expected top-level matching evidence, got %#v ok=%v", evidence, ok)
	}
}

func TestAssessFailsWhenPlanIsMissingForGeneratedArtifacts(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "support-plan-missing")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Support

When a support ticket is created, fetch the ticket details.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "support.yaml"), []byte(`openapi: 3.0.0
info:
  title: Support API
  version: 1.0.0
servers:
  - url: https://support.example.test
paths:
  /tickets/{ticketId}:
    get:
      operationId: getTicket
      parameters:
        - name: ticketId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		Discoverer: &openapidisco.Discoverer{},
		LLMClient:  fakeClient{},
		ChatClient: fakeClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(result.PlanJSONPath); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "plan.present", "fail") {
		t.Fatalf("expected plan.present failure, got %#v", report.Checks)
	}
}

func TestAssessFailsWhenWorkflowBindingSourceDivergesFromPlan(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "weather-source-drift")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Weather Lookup

## Goal

Search weather in Toronto, Canada.

## External Systems and OpenAPI

- Use the OpenWeather OpenAPI document under openapi/.

## Data Flow

- Resolve coordinates before weather lookup.

## Runtime Policy

- openapi and http are allowed.

## Credentials and Secrets

- Use runtime credential bindings only.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop if required coordinates cannot be resolved.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "weather.yaml"), []byte(weatherOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeWeatherChainClient{},
		ChatClient: fakeWeatherChainClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(result.WorkflowPath)
	if err != nil {
		t.Fatal(err)
	}
	drifted := strings.ReplaceAll(string(workflow), "get_coordinates.received_body", "wrong_step.received_body")
	if drifted == string(workflow) {
		t.Fatalf("test fixture did not contain expected binding source:\n%s", workflow)
	}
	if err := os.WriteFile(result.WorkflowPath, []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "workflow.binding_sources", "fail") {
		t.Fatalf("expected workflow.binding_sources failure, got %#v", report.Checks)
	}
}

func TestAssessFailsWhenWorkflowCredentialFieldMissing(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "weather-credential-drift")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Weather Lookup

## Goal

Search weather in Toronto, Canada.

## External Systems and OpenAPI

- Use the OpenWeather OpenAPI document under openapi/.

## Data Flow

- Resolve coordinates before weather lookup.

## Runtime Policy

- openapi and http are allowed.

## Credentials and Secrets

- Use runtime credential binding weather_appid.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop if required coordinates cannot be resolved.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "weather.yaml"), []byte(weatherOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  fakeWeatherChainClient{},
		ChatClient: fakeWeatherChainClient{},
		SchemaPath: schemaPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	workflow, err := os.ReadFile(result.WorkflowPath)
	if err != nil {
		t.Fatal(err)
	}
	drifted := strings.Replace(string(workflow), `appid = "weather_appid"`+"\n", "", 1)
	if drifted == string(workflow) {
		t.Fatalf("test fixture did not contain expected credential field:\n%s", workflow)
	}
	if err := os.WriteFile(result.WorkflowPath, []byte(drifted), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "workflow.credentials_bound", "fail") {
		t.Fatalf("expected workflow.credentials_bound failure, got %#v", report.Checks)
	}
}

func TestAssessFailsUnsatisfiedRequiredOpenAPIParams(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "weather-missing-flow")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Weather Lookup

## Goal

Search weather in Toronto, Canada.

## External Systems and OpenAPI

- Use openapi/weather.yaml.

## Runtime Policy

- openapi and http are allowed.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop if coordinates cannot be resolved.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "weather.yaml"), []byte(weatherOnlyOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(`openapi = "openapi/weather.yaml"

workflow {
  name = "weather"
}

step "get_weather" {
  type      = "http"
  do        = "Get weather for Toronto"
  operation = "getWeatherData"
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "intent.data_flow.required_params", "fail") {
		t.Fatalf("expected required params failure, got %#v", report.Checks)
	}
}

func TestAssessFailsCredentialLikeParamsWithoutPolicy(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "weather-missing-credentials")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Weather Lookup

## Goal

Search weather by coordinates.

## External Systems and OpenAPI

- Use openapi/weather.yaml.

## Runtime Policy

- openapi and http are allowed.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop if weather cannot be fetched.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "weather.yaml"), []byte(weatherOnlyOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(`openapi = "openapi/weather.yaml"

workflow {
  name = "weather"
}

step "get_weather" {
  type      = "http"
  do        = "Get weather"
  operation = "getWeatherData"
  with = {
    lat = "43.6532"
    lon = "-79.3832"
  }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "credentials.bindings", "fail") {
		t.Fatalf("expected credentials.bindings failure, got %#v", report.Checks)
	}
}

func TestSpecSummaryIncludesDataFlowPlanningMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "weather.yaml")
	if err := os.WriteFile(path, []byte(weatherOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	summary := specSummary([]openapidisco.Candidate{{Path: path, RelativePath: "openapi/weather.yaml"}})
	for _, expected := range []string{"required_parameters: appid, lat, lon", "response_fields: lat, lon"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("spec summary missing %q:\n%s", expected, summary)
		}
	}
}

func TestSynthesizeRetriesWorkflowGenerationAndWritesRefinementReport(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "support-retry")
	writeSupportExample(t, example, false)
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	client := &retryWorkflowClient{}
	result, err := Synthesize(context.Background(), Options{
		ExampleDir:  example,
		Discoverer:  &openapidisco.Discoverer{APIsGuruListURL: "http://127.0.0.1/list.json"},
		LLMClient:   client,
		ChatClient:  client,
		SchemaPath:  schemaPath,
		MaxAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if client.chatCalls < 2 {
		t.Fatalf("expected refinement retry, got %d chat call(s)", client.chatCalls)
	}
	refinement, err := os.ReadFile(result.RefinementJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(refinement), `"status": "pass"`) || !strings.Contains(string(refinement), "missingTicketOperation") {
		t.Fatalf("refinement report missing retry evidence:\n%s", refinement)
	}
	if !strings.Contains(string(refinement), `"prompt_version": "`+intentPromptVersion+`"`) {
		t.Fatalf("refinement report missing prompt version:\n%s", refinement)
	}
	if !strings.Contains(string(refinement), `"prompt_snapshot":`) {
		t.Fatalf("refinement report missing prompt snapshot:\n%s", refinement)
	}
}

func TestSynthesizeStopsAtMaxAttemptsAndWritesRefinementReport(t *testing.T) {
	root := t.TempDir()
	example := filepath.Join(root, "examples", "support-max")
	writeSupportExample(t, example, true)
	schemaPath, err := filepath.Abs(filepath.Join("..", "..", "..", "uws", "versions", "1.0.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Synthesize(context.Background(), Options{
		ExampleDir:  example,
		LLMClient:   badInputSourceClient{},
		ChatClient:  badInputSourceClient{},
		SchemaPath:  schemaPath,
		MaxAttempts: 2,
	})
	if err == nil {
		t.Fatalf("expected quality failure")
	}
	refinement, readErr := os.ReadFile(filepath.Join(example, "expected", "refinement.json"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(refinement), "maximum refinement attempts reached") && !strings.Contains(string(refinement), "repeated quality failure") {
		t.Fatalf("refinement report missing clean stop reason:\n%s", refinement)
	}
	if !strings.Contains(string(refinement), `"failure_class": "validation"`) {
		t.Fatalf("refinement report missing validation failure class:\n%s", refinement)
	}
}

func TestComplementaryDiscoveryRecordsAttempt(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "weather-complementary")
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "weather.yaml"), []byte(weatherOnlyOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
	candidates, err := openapidisco.LocalFiles(filepath.Join(example, "openapi"), example, "weather")
	if err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		OpenAPI: "openapi/weather.yaml",
		Steps: []*rollout.Step{{
			Name:      "get_weather",
			Type:      "http",
			Operation: "getWeatherData",
		}},
	}
	_, attempts, _ := discoverComplementaryOpenAPI(context.Background(), &openapidisco.Discoverer{
		APIsGuruListURL: "http://127.0.0.1/list.json",
	}, example, "Search weather in Toronto, Canada.", candidates, intent, analyzeProject(""))
	if len(attempts) != 1 || attempts[0].Kind != "apis.guru.complementary" || attempts[0].Status != "fail" {
		t.Fatalf("unexpected complementary discovery attempts: %#v", attempts)
	}
}

func TestAssessUsesPrimaryOpenAPIForIntentWithoutTopLevelOpenAPI(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "primary-fallback")
	writeSupportExample(t, example, false)
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(`workflow {
  name = "support"
}

step "get_ticket" {
  type      = "http"
  do        = "Fetch the support ticket"
  operation = "getTicket"
  with = {
    ticketId = "ticket_123"
  }
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Assess(Options{ExampleDir: example})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCheck(report, "intent.openapi_operations", "pass") {
		t.Fatalf("expected operation validation to use primary OpenAPI, got %#v", report.Checks)
	}
}

func TestBuildAndPromoteRequireProjectBrief(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "missing-project")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Build(context.Background(), Options{ExampleDir: example}); err == nil || !strings.Contains(err.Error(), "read project brief") {
		t.Fatalf("expected build to require project.md, got %v", err)
	}
	if _, err := Promote(context.Background(), Options{ExampleDir: example}); err == nil || !strings.Contains(err.Error(), "read project brief") {
		t.Fatalf("expected promote to require project.md, got %v", err)
	}
}

func TestSynthesizeHonorsCancelledContextBeforeWork(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "cancelled")
	writeSupportExample(t, example, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Synthesize(ctx, Options{ExampleDir: example, LLMClient: fakeClient{}, ChatClient: fakeClient{}}); err == nil || err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestSynthesizePropagatesRefinementReportWriteFailure(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "refinement-write-fail")
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a local report.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop on failure.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "expected"), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Synthesize(context.Background(), Options{
		ExampleDir: example,
		LLMClient:  failingChatClient{},
		ChatClient: failingChatClient{},
	})
	if err == nil || !strings.Contains(err.Error(), "write refinement report") {
		t.Fatalf("expected refinement write failure, got %v", err)
	}
}

func TestClassifierUsesHighestPriorityFailingAction(t *testing.T) {
	report := &QualityReport{Checks: []QualityCheck{
		{Code: "workflow.plan_match", Status: "fail"},
		{Code: "openapi.local", Status: "fail"},
	}}
	action, terminal := classifyRefinementAction(report)
	if action != "discover_openapi" || terminal {
		t.Fatalf("action = %s terminal=%v, want discover_openapi false", action, terminal)
	}
	report.Checks = append(report.Checks, QualityCheck{Code: "artifacts.no_secrets", Status: "fail"})
	action, terminal = classifyRefinementAction(report)
	if action != "stop" || !terminal {
		t.Fatalf("action = %s terminal=%v, want stop true", action, terminal)
	}
}

func TestQualityFailureSignatureUsesStableFamilies(t *testing.T) {
	report := &QualityReport{Checks: []QualityCheck{
		{Code: "workflow.plan_match", Status: "fail"},
		{Code: "workflow.binding_sources", Status: "fail"},
		{Code: "intent.slots", Status: "fail"},
	}}
	if got := qualityFailureSignature(report); got != "intent" {
		t.Fatalf("signature = %q", got)
	}
}

func TestSpecSummaryAppliesGlobalOperationBudget(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.yaml")
	second := filepath.Join(dir, "second.yaml")
	if err := os.WriteFile(first, []byte(manyOperationsOpenAPI("first", 50)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(manyOperationsOpenAPI("second", 50)), 0o644); err != nil {
		t.Fatal(err)
	}
	summary := specSummary([]openapidisco.Candidate{
		{Path: first, RelativePath: "openapi/first.yaml"},
		{Path: second, RelativePath: "openapi/second.yaml"},
	})
	if count := strings.Count(summary, "  operation:"); count != maxPromptOperationsTotal {
		t.Fatalf("operation count = %d, want %d\n%s", count, maxPromptOperationsTotal, summary)
	}
	if !strings.Contains(summary, "omitted_operations: 20") {
		t.Fatalf("summary missing omitted global-budget evidence:\n%s", summary)
	}
}

func TestSecretScannerFlagsCommonTokenFamilies(t *testing.T) {
	values := []string{
		"AIzaabcdefghijklmnopqrstuvwxyz1234567890",
		"ghp_abcdefghijklmnopqrstuvwxyzABCDEFGHIJ",
		"sk-ant-api03-abcdefghijklmnopqrstuvwxyz1234567890",
		"sk-proj-abcdefghijklmnopqrstuvwxyz1234567890",
		"AKIA1234567890ABCDEF",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.TJVA95OrM7E2cBab30RMHrHDcEfxjoYZgeFONFh7HgQ",
		`api_key = "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"`,
		`appid = "abc123abc123abc123"`,
		`password = "abc123abc123abc123"`,
	}
	for _, value := range values {
		checks := LintProjectMarkdown("# Project\n\n" + value + "\n")
		if !qualityChecksContain(checks, "project.no_secrets", "fail") {
			t.Fatalf("project lint did not flag %q: %#v", value, checks)
		}
	}
}

func TestSecretScannerAllowsWorkflowReferencesAndBindings(t *testing.T) {
	values := []string{
		`from = "inputs.ticketId"`,
		`to = "get_ticket.received_body.requesterEmail"`,
		`subject = "get_ticket.received_body.subject"`,
		`body = "get_ticket.received_body.summary"`,
		`lat = "get_coordinates.received_body[0].lat"`,
		`appid = "weather_appid"`,
		`api_key = "weather_api_key"`,
		`token_from = "weather_api_key"`,
		`authorization = "inputs.authorization"`,
		`get_ticket.received_body.requesterEmail`,
		`weather_api_key`,
		`weather_appid`,
	}
	for _, value := range values {
		checks := LintProjectMarkdown("# Project\n\n" + value + "\n")
		if qualityChecksContain(checks, "project.no_secrets", "fail") {
			t.Fatalf("project lint flagged valid reference or binding %q: %#v", value, checks)
		}
	}
}

func TestNoSecretsQualityAllowsSupportEmailReferences(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "support-email")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(example, "expected"), 0o755); err != nil {
		t.Fatal(err)
	}
	result := resultPaths(example)
	files := map[string]string{
		result.ProjectPath: `# Support Email

## Goal

Send a support confirmation email.

## Credentials and Secrets

- Use credential binding names only.
`,
		result.WorkflowPath: `step "get_ticket" {
  type = "http"
  with = {
    ticketId = "inputs.ticketId"
  }
}

step "send_confirmation_email" {
  type = "fnct"
  bind {
    from = "get_ticket"
    fields = {
      to      = "get_ticket.received_body.requesterEmail"
      subject = "get_ticket.received_body.subject"
      body    = "get_ticket.received_body.summary"
    }
  }
}
`,
		result.IntentPath: `output "email_status" {
  from = "send_confirmation_email.received_body"
}
`,
		result.PlanJSONPath:       `{"parameters":{"ticketId":"inputs.ticketId"}}`,
		result.PlanMDPath:         "ticketId = inputs.ticketId\n",
		result.DiscoveryJSONPath:  "{}\n",
		result.RefinementJSONPath: "{}\n",
		result.RefinementMDPath:   "No refinement.\n",
		result.ReviewPath:         "Side-effectful execution was skipped.\n",
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	report := &QualityReport{Example: example, Artifacts: result}
	assessSecrets(report, result)
	if !hasCheck(report, "artifacts.no_secrets", "pass") {
		t.Fatalf("expected artifacts.no_secrets pass, got %#v", report.Checks)
	}
}

func TestNoSecretsQualityReportsOnlyArtifactPaths(t *testing.T) {
	example := filepath.Join(t.TempDir(), "examples", "secret-reporting")
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	result := resultPaths(example)
	secret := "sk-proj-abcdefghijklmnopqrstuvwxyz1234567890"
	if err := os.WriteFile(result.WorkflowPath, []byte(`api_key = "`+secret+`"`), 0o644); err != nil {
		t.Fatal(err)
	}
	report := &QualityReport{Example: example, Artifacts: result}
	assessSecrets(report, result)
	if !hasCheck(report, "artifacts.no_secrets", "fail") {
		t.Fatalf("expected artifacts.no_secrets fail, got %#v", report.Checks)
	}
	if len(report.Checks) != 1 || !strings.Contains(report.Checks[0].Detail, "workflows/workflow.hcl") {
		t.Fatalf("expected artifact path in detail, got %#v", report.Checks)
	}
	if strings.Contains(report.Checks[0].Detail, secret) {
		t.Fatalf("secret value leaked in quality detail: %q", report.Checks[0].Detail)
	}
}

func hasCheck(report *QualityReport, code, status string) bool {
	if report == nil {
		return false
	}
	for _, check := range report.Checks {
		if check.Code == code && check.Status == status {
			return true
		}
	}
	return false
}

func qualityChecksContain(checks []QualityCheck, code, status string) bool {
	for _, check := range checks {
		if check.Code == code && check.Status == status {
			return true
		}
	}
	return false
}

func writeSupportExample(t *testing.T, example string, includeInput bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	project := `# Support

## Goal

When a support ticket is created, fetch the ticket details.

## Inputs

- ticketId: required string.

## External Systems and OpenAPI

- Use openapi/support.yaml.

## Runtime Policy

- openapi and http are allowed.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate only.

## Fallback Behavior

- Stop if the ticket cannot be fetched.
`
	if !includeInput {
		project = strings.Replace(project, "- ticketId: required string.", "- ticket_id: required string.", 1)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(project), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "support.yaml"), []byte(supportOpenAPI()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeRuntimeOnlyExample(t *testing.T, example string) {
	t.Helper()
	if err := os.MkdirAll(example, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(`# Runtime Only

## Goal

Render a local report.

## Inputs

- summary: required string.

## Outputs

- Rendered report output.

## External Systems and OpenAPI

OpenAPI: none required

## Runtime Policy

- fnct allowed for trusted local report rendering.
- cmd and ssh are not allowed.

## Function Contracts

- render_report
  - Inputs: summary.
  - Outputs: rendered report.
  - Side effects: none.

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if no approved function runtime exists.
`), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runtimeOnlyIntent() *rollout.Intent {
	return &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{
			Name:        "runtime_only",
			Description: "Render a local report.",
		},
		Inputs: []*rollout.Input{{
			Name:     "summary",
			Type:     "string",
			Required: true,
		}},
		Steps: []*rollout.Step{{
			Name: "render_report",
			Type: "fnct",
			Do:   "Render the local report",
			With: map[string]string{"summary": "inputs.summary"},
		}},
		Outputs: []*rollout.Output{{
			Name: "report",
			From: "render_report.received_body",
		}},
	}
}

func supportOpenAPI() string {
	return `openapi: 3.0.0
info:
  title: Support API
  version: 1.0.0
servers:
  - url: https://support.example.test
paths:
  /tickets/{ticketId}:
    get:
      operationId: getTicket
      parameters:
        - name: ticketId
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
                  severity:
                    type: string
`
}

func weatherOpenAPI() string {
	return `openapi: 3.0.0
info:
  title: Weather API
  version: 1.0.0
servers:
  - url: https://api.openweathermap.org
paths:
  /geo/1.0/direct:
    get:
      operationId: direct_get
      parameters:
        - name: q
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: array
                items:
                  type: object
                  properties:
                    lat:
                      type: number
                    lon:
                      type: number
  /data/2.5/weather:
    get:
      operationId: getWeatherData
      parameters:
        - name: lat
          in: query
          required: true
          schema:
            type: number
        - name: lon
          in: query
          required: true
          schema:
            type: number
        - name: appid
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  lat:
                    type: number
                  lon:
                    type: number
                  weather:
                    type: array
`
}

func weatherOnlyOpenAPI() string {
	return `openapi: 3.0.0
info:
  title: Weather API
  version: 1.0.0
servers:
  - url: https://api.openweathermap.org
paths:
  /data/2.5/weather:
    get:
      operationId: getWeatherData
      parameters:
        - name: lat
          in: query
          required: true
          schema:
            type: number
        - name: lon
          in: query
          required: true
          schema:
            type: number
        - name: appid
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`
}

func secureOpenAPI() string {
	return `openapi: 3.0.0
info:
  title: Secure Support API
  version: 1.0.0
servers:
  - url: https://support.example.test
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: query
      name: api_key
paths:
  /secure/tickets:
    get:
      operationId: listSecureTickets
      security:
        - ApiKeyAuth: []
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                properties:
                  tickets:
                    type: array
`
}

func ticketWriteOpenAPI(server string) string {
	return fmt.Sprintf(`openapi: 3.0.0
info:
  title: Ticket Write API
  version: 1.0.0
servers:
  - url: %s
paths:
  /tickets:
    post:
      operationId: createTicket
      responses:
        "201":
          description: created
          content:
            application/json:
              schema:
                type: object
                properties:
                  id:
                    type: string
`, server)
}

func manyOperationsOpenAPI(title string, count int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "openapi: 3.0.0\ninfo:\n  title: %s\n  version: 1.0.0\npaths:\n", title)
	for i := 0; i < count; i++ {
		fmt.Fprintf(&b, "  /op%d:\n    get:\n      operationId: %sOp%d\n      responses:\n        \"200\":\n          description: ok\n", i, title, i)
	}
	return b.String()
}
