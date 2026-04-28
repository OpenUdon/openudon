package synthesize

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/udon/pkg/rollout"
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
  "steps": [
    {"name": "render_report", "type": "fnct", "do": "Render the local report"}
  ],
  "outputs": [{"name": "report", "from": "render_report.received_body"}]
}`, nil
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
	for _, path := range []string{result.IntentPath, result.WorkflowPath, result.UWSPath, result.PlanJSONPath, result.PlanMDPath, result.ReviewPath, result.QualityJSONPath, result.QualityMDPath} {
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
	report, err := Assess(Options{ExampleDir: example, SchemaPath: schemaPath})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed() {
		t.Fatalf("quality did not pass: %#v", report.Checks)
	}
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

## Credentials and Secrets

- No credentials are required.

## Safety and Approval Boundary

- Generate and validate artifacts only.

## Fallback Behavior

- Stop if no approved function runtime exists.
`), 0o644); err != nil {
		t.Fatal(err)
	}
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
