package synthesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/runtimes"
	"github.com/OpenUdon/uws/uws1"
)

func TestGenerateWorkflowDocumentEmitsUWS12TypedSource(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "google-discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(minimalDiscoveryDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Source:   "google-discovery/gmail.json",
		Workflow: &rollout.WorkflowMeta{Name: "gmail_send"},
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/gmail.json",
			Operation: "gmail_users_messages_send",
			With: map[string]string{
				"userId": "me",
			},
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.2.0" {
		t.Fatalf("UWS version = %q, want 1.2.0", doc.UWS)
	}
	if len(doc.SourceDescriptions) != 1 || doc.SourceDescriptions[0].Type != uws1.SourceDescriptionTypeGoogleDiscovery {
		t.Fatalf("sourceDescriptions = %#v", doc.SourceDescriptions)
	}
	if got := doc.Operations[0].SourceOperationID; got != "gmail_users_messages_send" {
		t.Fatalf("sourceOperationId = %q", got)
	}
	if doc.Operations[0].OpenAPIOperationID != "" {
		t.Fatalf("legacy openapiOperationId should be empty, got %q", doc.Operations[0].OpenAPIOperationID)
	}
	path, ok := doc.Operations[0].Request["path"].(map[string]any)
	if !ok || path["userId"] != "me" {
		t.Fatalf("request path binding = %#v", doc.Operations[0].Request)
	}
}

func TestGenerateWorkflowDocumentEmitsUWS13AsyncAPISource(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "asyncapi", "events.yaml"), []byte(minimalAsyncAPIDocument()))
	intent := &rollout.Intent{
		Source:   "asyncapi/events.yaml",
		Workflow: &rollout.WorkflowMeta{Name: "billing_events"},
		Steps: []*rollout.Step{{
			Name:      "publish_invoice",
			Type:      "http",
			Source:    "asyncapi/events.yaml",
			Operation: "publishInvoice",
			With: map[string]string{
				"body.invoice_id": "inputs.invoice_id",
				"header.trace_id": "inputs.trace_id",
			},
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if doc.UWS != "1.3.0" {
		t.Fatalf("UWS version = %q, want 1.3.0", doc.UWS)
	}
	if len(doc.SourceDescriptions) != 1 || doc.SourceDescriptions[0].Type != uws1.SourceDescriptionTypeAsyncAPI {
		t.Fatalf("sourceDescriptions = %#v", doc.SourceDescriptions)
	}
	if got := doc.Operations[0].SourceOperationID; got != "publishInvoice" {
		t.Fatalf("sourceOperationId = %q", got)
	}
	body, ok := doc.Operations[0].Request["body"].(map[string]any)
	if !ok || body["invoice_id"] == nil {
		t.Fatalf("request body binding = %#v", doc.Operations[0].Request)
	}
	header, ok := doc.Operations[0].Request["header"].(map[string]any)
	if !ok || header["trace_id"] == nil {
		t.Fatalf("request header binding = %#v", doc.Operations[0].Request)
	}
}

func TestGenerateWorkflowDocumentEmitsAsyncAPISourceOperationRef(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "asyncapi", "events.yaml"), []byte(minimalAsyncAPIDocument()))
	intent := &rollout.Intent{
		Source:   "asyncapi/events.yaml",
		Workflow: &rollout.WorkflowMeta{Name: "billing_events"},
		Steps: []*rollout.Step{{
			Name:      "publish_invoice",
			Type:      "http",
			Source:    "asyncapi/events.yaml",
			Operation: "#/operations/publishInvoice",
			With:      map[string]string{"body.invoice_id": "inputs.invoice_id"},
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if got := doc.Operations[0].SourceOperationRef; got != "#/operations/publishInvoice" {
		t.Fatalf("sourceOperationRef = %q", got)
	}
	if doc.Operations[0].SourceOperationID != "" {
		t.Fatalf("sourceOperationId should be empty, got %q", doc.Operations[0].SourceOperationID)
	}
}

func TestGenerateWorkflowDocumentRequiresExplicitAsyncAPIRequestPlacement(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "asyncapi", "events.yaml"), []byte(minimalAsyncAPIDocument()))
	intent := &rollout.Intent{
		Source:   "asyncapi/events.yaml",
		Workflow: &rollout.WorkflowMeta{Name: "billing_events"},
		Steps: []*rollout.Step{{
			Name:      "publish_invoice",
			Type:      "http",
			Source:    "asyncapi/events.yaml",
			Operation: "publishInvoice",
			With:      map[string]string{"invoice_id": "inputs.invoice_id"},
		}},
	}
	_, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err == nil || !strings.Contains(err.Error(), "request field \"invoice_id\" is not declared") {
		t.Fatalf("expected explicit AsyncAPI request placement error, got %v", err)
	}
}

func TestGenerateWorkflowDocumentUsesRequestBodyForKnownFnctHelper(t *testing.T) {
	intent := &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "gmail_render"},
		Steps: []*rollout.Step{{
			Name:      "render_weather_report",
			Type:      "fnct",
			Operation: "gmail.render_raw",
			With: map[string]string{
				"to":            "inputs.recipient_email",
				"subject":       "Weather report",
				"body_template": "Weather report:\n\n{{.}}",
				"input":         "weather_lookup.received_body",
			},
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: t.TempDir()}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Operations) != 1 {
		t.Fatalf("operations = %#v", doc.Operations)
	}
	op := doc.Operations[0]
	runtime, ok, err := runtimes.ReadOperationExtension(op.Extensions)
	if err != nil || !ok {
		t.Fatalf("runtime extension = %#v, %v, %v", runtime, ok, err)
	}
	if runtime.Function != "gmail.render_raw" {
		t.Fatalf("runtime function = %q", runtime.Function)
	}
	if len(runtime.Arguments) != 0 {
		t.Fatalf("known helper should use request body, got arguments %#v", runtime.Arguments)
	}
	body, ok := op.Request["body"].(map[string]any)
	if !ok {
		t.Fatalf("request body = %#v", op.Request)
	}
	if to, ok := body["to"].(map[string]any); !ok || to["$expr"] != "variables.inputs.recipient_email" {
		t.Fatalf("request to = %#v", body["to"])
	}
	if input, ok := body["input"].(map[string]any); !ok || input["$expr"] != "weather_lookup.received_body" {
		t.Fatalf("request input = %#v", body["input"])
	}
	if body["subject"] != "Weather report" {
		t.Fatalf("request subject = %#v", body["subject"])
	}
}

func TestLocalNativeOperationIndexIncludesDiscoveryAliases(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "google-discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(minimalDiscoveryDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	index := localNativeOperationIndex(example)
	for _, operationID := range []string{"gmail.users.messages.send", "gmail_users_messages_send"} {
		if index[operationKey("google-discovery/gmail.json", operationID)] == nil {
			t.Fatalf("missing native operation alias %q in %#v", operationID, index)
		}
	}
}

func TestGoogleDiscoveryPropertyRequiredForOperationMatchesSanitizedOperationID(t *testing.T) {
	prop := map[string]any{
		"annotations": map[string]any{
			"required": []any{"gmail.users.messages.send"},
		},
	}
	if !googleDiscoveryPropertyRequiredForOperation(prop, "gmail_users_messages_send") {
		t.Fatalf("Discovery required annotation did not match sanitized operation ID")
	}
}

func TestValidateIntentResponsePathsUsesDiscoveryResponseSchema(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(discoveryResponseDocument()))
	intent := &rollout.Intent{Steps: []*rollout.Step{
		{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/gmail.json",
			Operation: "gmail.users.messages.send",
			With:      map[string]string{"userId": "me"},
		},
		{Name: "render", Type: "fnct", With: map[string]string{"message_id": "send.received_body.id"}},
	}}
	result := validateIntentResponsePaths(intent, example, nil, "")
	if len(result.Failures) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("expected Discovery response path to pass, got %#v", result)
	}
}

func TestValidateIntentResponsePathsRejectsAbsentDiscoveryResponseField(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(discoveryResponseDocument()))
	intent := &rollout.Intent{Steps: []*rollout.Step{
		{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/gmail.json",
			Operation: "gmail.users.messages.send",
			With:      map[string]string{"userId": "me"},
		},
		{Name: "render", Type: "fnct", With: map[string]string{"missing": "send.received_body.notThere"}},
	}}
	result := validateIntentResponsePaths(intent, example, nil, "")
	if len(result.Failures) == 0 || !strings.Contains(result.Failures[0], "notThere") {
		t.Fatalf("expected missing Discovery response field failure, got %#v", result)
	}
}

func TestValidateIntentResponsePathsWarnsOnOpaqueDiscoveryResponseSchema(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(discoveryMissingResponseSchemaDocument()))
	intent := &rollout.Intent{Steps: []*rollout.Step{
		{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/gmail.json",
			Operation: "gmail.users.messages.send",
			With:      map[string]string{"userId": "me"},
		},
		{Name: "render", Type: "fnct", With: map[string]string{"message_id": "send.received_body.id"}},
	}}
	result := validateIntentResponsePaths(intent, example, nil, "")
	if len(result.Failures) != 0 || len(result.Warnings) == 0 {
		t.Fatalf("expected opaque Discovery response warning, got %#v", result)
	}
}

func TestValidateIntentResponsePathsUsesSmithyBodyResponseSchema(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/lambda.json", smithyResponseBodyDocument())
	intent := &rollout.Intent{Steps: []*rollout.Step{
		{
			Name:      "get",
			Type:      "http",
			Source:    "aws-smithy/lambda.json",
			Operation: "GetFunction",
			With:      map[string]string{"FunctionName": "hello"},
		},
		{Name: "render", Type: "fnct", With: map[string]string{
			"alias": "get.received_body.Configuration.Aliases.name",
			"arn":   "get.received_body.Configuration.FunctionArn",
			"tag":   "get.received_body.Configuration.Tags.environment",
		}},
	}}
	result := validateIntentResponsePaths(intent, example, nil, "")
	if len(result.Failures) != 0 || len(result.Warnings) != 0 {
		t.Fatalf("expected Smithy response path to pass, got %#v", result)
	}
}

func TestValidateIntentResponsePathsRejectsAbsentSmithyBodyField(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/lambda.json", smithyResponseBodyDocument())
	intent := &rollout.Intent{Steps: []*rollout.Step{
		{
			Name:      "get",
			Type:      "http",
			Source:    "aws-smithy/lambda.json",
			Operation: "GetFunction",
			With:      map[string]string{"FunctionName": "hello"},
		},
		{Name: "render", Type: "fnct", With: map[string]string{"missing": "get.received_body.Configuration.Missing"}},
	}}
	result := validateIntentResponsePaths(intent, example, nil, "")
	if len(result.Failures) == 0 || !strings.Contains(result.Failures[0], "Missing") {
		t.Fatalf("expected missing Smithy response field failure, got %#v", result)
	}
}

func TestValidateIntentResponsePathsExcludesSmithyHeaderAndStatusOutputMetadata(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/lambda.json", smithyResponseBodyDocument())
	intent := &rollout.Intent{Steps: []*rollout.Step{
		{
			Name:      "get",
			Type:      "http",
			Source:    "aws-smithy/lambda.json",
			Operation: "GetFunction",
			With:      map[string]string{"FunctionName": "hello"},
		},
		{Name: "render", Type: "fnct", With: map[string]string{"status": "get.received_body.StatusCode"}},
	}}
	result := validateIntentResponsePaths(intent, example, nil, "")
	if len(result.Failures) == 0 || !strings.Contains(result.Failures[0], "StatusCode") {
		t.Fatalf("expected Smithy responseCode metadata to be absent from body schema, got %#v", result)
	}
}

func TestValidateIntentResponsePathsWarnsOnMissingSmithyOutputMetadata(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/lambda.json", minimalSmithyDocument())
	intent := &rollout.Intent{Steps: []*rollout.Step{
		{
			Name:      "get",
			Type:      "http",
			Source:    "aws-smithy/lambda.json",
			Operation: "GetFunction",
			With:      map[string]string{"FunctionName": "hello"},
		},
		{Name: "render", Type: "fnct", With: map[string]string{"arn": "get.received_body.Configuration.FunctionArn"}},
	}}
	result := validateIntentResponsePaths(intent, example, nil, "")
	if len(result.Failures) != 0 || len(result.Warnings) == 0 {
		t.Fatalf("expected opaque Smithy response warning, got %#v", result)
	}
}

func TestLocalAdvisorySecuritySidecarAppliesToDiscoveryOperation(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "google-discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(minimalDiscoveryDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	sidecar := `{
	  "security_schemes": [{"name":"gmail_oauth_token","type":"oauth2"}],
	  "operation_security": [{
	    "match": {"operation_id":"gmail.users.messages.send"},
	    "security": [{"scheme":"gmail_oauth_token"}]
	  }]
	}`
	if err := os.WriteFile(filepath.Join(dir, "gmail.security.json"), []byte(sidecar), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name:      "gmail",
		Type:      "http",
		Source:    "google-discovery/gmail.json",
		Operation: "gmail.users.messages.send",
		With:      map[string]string{"userId": "me", "raw": "render_message.received_body"},
	}}}
	if err := validateIntentOpenAPISecurity(intent, example, nil, "", analyzeProject("")); err == nil || !strings.Contains(err.Error(), "Credentials and Secrets") {
		t.Fatalf("expected sidecar security failure without credential policy, got %v", err)
	}
	policy := analyzeProject("## Credentials and Secrets\n- Use credential binding `gmail_oauth_token`.\n")
	if err := validateIntentOpenAPISecurity(intent, example, nil, "", policy); err != nil {
		t.Fatalf("security sidecar should pass with credential policy: %v", err)
	}
}

func TestValidateIntentOpenAPISecurityUsesNativeDiscoveryOAuthScopes(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(discoveryOAuthDocument()))
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name:      "send",
		Type:      "http",
		Source:    "google-discovery/gmail.json",
		Operation: "gmail.users.messages.send",
		With:      map[string]string{"userId": "me", "raw": "render_message.received_body"},
	}}}
	err := validateIntentOpenAPISecurity(intent, example, nil, "", analyzeProject(""))
	if err == nil || !strings.Contains(err.Error(), "Credentials and Secrets") || !strings.Contains(err.Error(), "gmail_oauth_token") {
		t.Fatalf("expected native Discovery OAuth credential policy failure, got %v", err)
	}
	policy := analyzeProject("## Credentials and Secrets\n- Use credential binding `gmail_oauth_token`.\n")
	if err := validateIntentOpenAPISecurity(intent, example, nil, "", policy); err != nil {
		t.Fatalf("native Discovery OAuth security should pass with policy: %v", err)
	}
}

func TestValidateIntentOpenAPISecurityIgnoresDiscoveryWithoutOperationScopes(t *testing.T) {
	example := t.TempDir()
	mustWriteSynthesizeTestFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(minimalDiscoveryDocument()))
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name:      "send",
		Type:      "http",
		Source:    "google-discovery/gmail.json",
		Operation: "gmail.users.messages.send",
		With:      map[string]string{"userId": "me"},
	}}}
	if err := validateIntentOpenAPISecurity(intent, example, nil, "", analyzeProject("")); err != nil {
		t.Fatalf("Discovery operation without scopes should not require native security: %v", err)
	}
}

func TestValidateIntentOpenAPISecurityUsesNativeSmithySigV4(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/lambda.json", minimalSmithyDocument())
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name:      "get",
		Type:      "http",
		Source:    "aws-smithy/lambda.json",
		Operation: "GetFunction",
		With:      map[string]string{"FunctionName": "hello"},
	}}}
	err := validateIntentOpenAPISecurity(intent, example, nil, "", analyzeProject(""))
	if err == nil || !strings.Contains(err.Error(), "Credentials and Secrets") || !strings.Contains(err.Error(), "lambda_sigv4") {
		t.Fatalf("expected native Smithy SigV4 credential policy failure, got %v", err)
	}
	policy := analyzeProject("## Credentials and Secrets\n- Use credential binding `lambda_sigv4`.\n")
	if err := validateIntentOpenAPISecurity(intent, example, nil, "", policy); err != nil {
		t.Fatalf("native Smithy SigV4 security should pass with policy: %v", err)
	}
}

func TestValidateIntentOpenAPISecurityIgnoresSmithyWithoutSigningName(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/lambda.json", smithyWithoutSigV4Document())
	intent := &rollout.Intent{Steps: []*rollout.Step{{
		Name:      "get",
		Type:      "http",
		Source:    "aws-smithy/lambda.json",
		Operation: "GetFunction",
		With:      map[string]string{"FunctionName": "hello"},
	}}}
	if err := validateIntentOpenAPISecurity(intent, example, nil, "", analyzeProject("")); err != nil {
		t.Fatalf("Smithy service without signing name should not require native security: %v", err)
	}
}

func TestGenerateWorkflowDocumentPrefersSourceOverLegacyOpenAPI(t *testing.T) {
	example := t.TempDir()
	discoveryDir := filepath.Join(example, "google-discovery")
	openAPIDir := filepath.Join(example, "openapi")
	if err := os.MkdirAll(discoveryDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(discoveryDir, "gmail.json"), []byte(minimalDiscoveryDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(openAPIDir, "legacy.yaml"), []byte("openapi: 3.0.0\ninfo: {title: Legacy, version: '1.0'}\npaths: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Source:   "google-discovery/gmail.json",
		OpenAPI:  "openapi/legacy.yaml",
		Workflow: &rollout.WorkflowMeta{Name: "gmail_send"},
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Operation: "gmail_users_messages_send",
			With:      map[string]string{"userId": "me"},
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.SourceDescriptions) != 1 || doc.SourceDescriptions[0].URL != "google-discovery/gmail.json" {
		t.Fatalf("sourceDescriptions = %#v, want google discovery source", doc.SourceDescriptions)
	}
	if doc.Operations[0].SourceOperationID != "gmail_users_messages_send" || doc.Operations[0].OpenAPIOperationID != "" {
		t.Fatalf("operation selectors = source %q openapi %q", doc.Operations[0].SourceOperationID, doc.Operations[0].OpenAPIOperationID)
	}
}

func TestGenerateWorkflowDocumentInfersSmithyRequestBindings(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "aws-smithy")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lambda.json"), []byte(minimalSmithyDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Source:   "aws-smithy/lambda.json",
		Workflow: &rollout.WorkflowMeta{Name: "lambda_get"},
		Steps: []*rollout.Step{{
			Name:      "get",
			Type:      "http",
			Source:    "aws-smithy/lambda.json",
			Operation: "GetFunction",
			With: map[string]string{
				"FunctionName": "hello",
				"Qualifier":    "$LATEST",
			},
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.SourceDescriptions) != 1 || doc.SourceDescriptions[0].Type != uws1.SourceDescriptionTypeAWSSmithy {
		t.Fatalf("sourceDescriptions = %#v", doc.SourceDescriptions)
	}
	path, ok := doc.Operations[0].Request["path"].(map[string]any)
	if !ok || path["FunctionName"] != "hello" {
		t.Fatalf("request path binding = %#v", doc.Operations[0].Request)
	}
	query, ok := doc.Operations[0].Request["query"].(map[string]any)
	if !ok || query["Qualifier"] != "$LATEST" {
		t.Fatalf("request query binding = %#v", doc.Operations[0].Request)
	}
}

func TestValidateIntentRequiredParametersUsesSmithyRequestMetadata(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/thing.json", smithyRequiredParamsDocument())

	baseIntent := func(with map[string]string) *rollout.Intent {
		return &rollout.Intent{
			Source:   "aws-smithy/thing.json",
			Workflow: &rollout.WorkflowMeta{Name: "thing_put"},
			Steps: []*rollout.Step{{
				Name:      "put",
				Type:      "http",
				Source:    "aws-smithy/thing.json",
				Operation: "PutThing",
				With:      with,
			}},
		}
	}

	err := validateIntentRequiredParameters(baseIntent(nil), example, nil, "")
	if err == nil {
		t.Fatal("expected missing Smithy required parameters")
	}
	for _, want := range []string{`path parameter "ThingId"`, `query parameter "mode"`, `header parameter "If-Match"`, `body parameter "Description"`, `body parameter "Payload"`} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error missing %q: %v", want, err)
		}
	}

	required := map[string]string{
		"ThingId":     "thing-1",
		"mode":        "replace",
		"If-Match":    "etag",
		"Description": "description",
		"Payload":     "payload",
	}
	if err := validateIntentRequiredParameters(baseIntent(required), example, nil, ""); err != nil {
		t.Fatalf("required Smithy mappings should pass without optional fields: %v", err)
	}

	withOptional := map[string]string{
		"ThingId":     "thing-1",
		"mode":        "replace",
		"If-Match":    "etag",
		"Description": "description",
		"Payload":     "payload",
		"filter":      "all",
		"Note":        "operator note",
	}
	if err := validateIntentRequiredParameters(baseIntent(withOptional), example, nil, ""); err != nil {
		t.Fatalf("Smithy mappings with optional query/body fields should pass: %v", err)
	}
}

func TestGenerateWorkflowDocumentPlacesSmithyRequestMembers(t *testing.T) {
	example := t.TempDir()
	writeSmithySourceForTest(t, example, "aws-smithy/thing.json", smithyRequiredParamsDocument())
	intent := &rollout.Intent{
		Source:   "aws-smithy/thing.json",
		Workflow: &rollout.WorkflowMeta{Name: "thing_put"},
		Steps: []*rollout.Step{{
			Name:      "put",
			Type:      "http",
			Source:    "aws-smithy/thing.json",
			Operation: "PutThing",
			With: map[string]string{
				"ThingId":     "thing-1",
				"mode":        "replace",
				"If-Match":    "etag",
				"Description": "description",
				"Payload":     "payload",
				"filter":      "all",
				"Note":        "operator note",
			},
		}},
	}
	doc, err := generateWorkflowDocument(Result{ExampleDir: example}, intent)
	if err != nil {
		t.Fatal(err)
	}
	request := doc.Operations[0].Request
	if path, ok := request["path"].(map[string]any); !ok || path["ThingId"] != "thing-1" {
		t.Fatalf("request path binding = %#v", request)
	}
	if query, ok := request["query"].(map[string]any); !ok || query["mode"] != "replace" || query["filter"] != "all" {
		t.Fatalf("request query binding = %#v", request)
	}
	if header, ok := request["header"].(map[string]any); !ok || header["If-Match"] != "etag" {
		t.Fatalf("request header binding = %#v", request)
	}
	body, ok := request["body"].(map[string]any)
	if !ok || body["Payload"] != "payload" || body["Description"] != "description" || body["Note"] != "operator note" {
		t.Fatalf("request body binding = %#v", request)
	}
}

func TestSourceDescriptionTypeForPathNormalizesRelativeForms(t *testing.T) {
	for _, path := range []string{"./google-discovery/gmail.json", "/tmp/example/google-discovery/gmail.json", "discovery/legacy.json"} {
		if got := sourceDescriptionTypeForPath(path); got != uws1.SourceDescriptionTypeGoogleDiscovery {
			t.Fatalf("source type for %q = %q, want google-discovery", path, got)
		}
	}
	if got := sourceDescriptionTypeForPath("./aws-smithy/lambda.json"); got != uws1.SourceDescriptionTypeAWSSmithy {
		t.Fatalf("source type = %q, want aws-smithy", got)
	}
}

func TestValidateIntentAPIRefsRejectsMissingNativeSource(t *testing.T) {
	intent := &rollout.Intent{
		Source: "google-discovery/missing.json",
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/missing.json",
			Operation: "gmail_users_messages_send",
		}},
	}
	err := validateIntentOpenAPIRefs(intent, t.TempDir(), nil, "", false)
	if err == nil || !strings.Contains(err.Error(), "google-discovery/missing.json") {
		t.Fatalf("error = %v, want missing google discovery source", err)
	}
}

func TestValidateIntentAPIRefsAcceptsValidNativeSource(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "google-discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(minimalDiscoveryDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	intent := &rollout.Intent{
		Source: "google-discovery/gmail.json",
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Source:    filepath.Join(example, "google-discovery", "gmail.json"),
			Operation: "gmail_users_messages_send",
		}},
	}
	if err := validateIntentOpenAPIRefs(intent, example, nil, "", false); err != nil {
		t.Fatalf("validateIntentOpenAPIRefs failed: %v", err)
	}
	if intent.Steps[0].Source != "google-discovery/gmail.json" {
		t.Fatalf("step source was not normalized: %q", intent.Steps[0].Source)
	}
}

func TestValidateIntentOpenAPIOperationsRejectsMissingNativeOperation(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "google-discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(minimalDiscoveryDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateIntentOpenAPIOperations(&rollout.Intent{
		Source: "google-discovery/gmail.json",
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/gmail.json",
			Operation: "gmail_users_messages_typo",
		}},
	}, example, nil, "")
	if err == nil || !strings.Contains(err.Error(), "gmail_users_messages_typo") {
		t.Fatalf("expected missing native operation error, got %v", err)
	}
}

func TestValidateIntentOpenAPIOperationsRejectsInvalidNativeSource(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "google-discovery")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(`{"kind":"discovery#restDescription","name":"gmail"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateIntentOpenAPIOperations(&rollout.Intent{
		Source: "google-discovery/gmail.json",
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Source:    "google-discovery/gmail.json",
			Operation: "gmail_users_messages_send",
		}},
	}, example, nil, "")
	if err == nil || !strings.Contains(err.Error(), "no operations discovered") {
		t.Fatalf("expected empty native operation registry error, got %v", err)
	}
}

func TestValidateIntentOpenAPIOperationsRejectsMisclassifiedAPISource(t *testing.T) {
	example := t.TempDir()
	dir := filepath.Join(example, "openapi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gmail.json"), []byte(minimalDiscoveryDocument()), 0o644); err != nil {
		t.Fatal(err)
	}
	err := validateIntentOpenAPIOperations(&rollout.Intent{
		Source: "openapi/gmail.json",
		Steps: []*rollout.Step{{
			Name:      "send",
			Type:      "http",
			Source:    "openapi/gmail.json",
			Operation: "gmail_users_messages_send",
		}},
	}, example, nil, "")
	if err == nil || !strings.Contains(err.Error(), "source path implies openapi but document looks like google-discovery") {
		t.Fatalf("expected misclassified API source error, got %v", err)
	}
}

func minimalDiscoveryDocument() string {
	return `{
	  "kind": "discovery#restDescription",
	  "discoveryVersion": "v1",
	  "name": "gmail",
	  "title": "Gmail API",
	  "version": "v1",
	  "rootUrl": "https://gmail.googleapis.com/",
	  "servicePath": "",
	  "resources": {
	    "users": {
	      "resources": {
	        "messages": {
	          "methods": {
	            "send": {
	              "id": "gmail.users.messages.send",
	              "path": "gmail/v1/users/{userId}/messages/send",
	              "httpMethod": "POST",
	              "parameters": {
	                "userId": {"type": "string", "location": "path", "required": true}
	              }
	            }
	          }
	        }
	      }
	    }
	  }
	}`
}

func minimalAsyncAPIDocument() string {
	return `asyncapi: 3.0.0
info:
  title: Billing Events
  version: 1.0.0
  description: Billing event bus.
operations:
  publishInvoice:
    action: send
    summary: Publish an invoice event.
    channel:
      $ref: '#/channels/invoices'
    messages:
      - $ref: '#/channels/invoices/messages/invoiceCreated'
channels:
  invoices:
    address: invoices
    messages:
      invoiceCreated:
        payload:
          type: object
`
}

func discoveryRequestBodyDocument() string {
	return `{
	  "kind": "discovery#restDescription",
	  "discoveryVersion": "v1",
	  "name": "gmail",
	  "title": "Gmail API",
	  "version": "v1",
	  "rootUrl": "https://gmail.googleapis.com/",
	  "servicePath": "",
	  "schemas": {
	    "Message": {
	      "id": "Message",
	      "type": "object",
	      "properties": {
	        "raw": {
	          "description": "The entire email message in an RFC 2822 formatted and base64url encoded string.",
	          "annotations": {"required": ["gmail.users.messages.send"]},
	          "type": "string"
	        }
	      }
	    }
	  },
	  "resources": {
	    "users": {
	      "resources": {
	        "messages": {
	          "methods": {
	            "send": {
	              "id": "gmail.users.messages.send",
	              "path": "gmail/v1/users/{userId}/messages/send",
	              "httpMethod": "POST",
	              "parameters": {
	                "userId": {"type": "string", "location": "path", "required": true}
	              },
	              "request": {"$ref": "Message"}
	            }
	          }
	        }
	      }
	    }
	  }
	}`
}

func discoveryResponseDocument() string {
	return `{
	  "kind": "discovery#restDescription",
	  "discoveryVersion": "v1",
	  "name": "gmail",
	  "title": "Gmail API",
	  "version": "v1",
	  "rootUrl": "https://gmail.googleapis.com/",
	  "servicePath": "",
	  "schemas": {
	    "Message": {
	      "id": "Message",
	      "type": "object",
	      "properties": {
	        "id": {"type": "string"},
	        "labelIds": {"type": "array", "items": {"type": "string"}}
	      }
	    }
	  },
	  "resources": {
	    "users": {
	      "resources": {
	        "messages": {
	          "methods": {
	            "send": {
	              "id": "gmail.users.messages.send",
	              "path": "gmail/v1/users/{userId}/messages/send",
	              "httpMethod": "POST",
	              "parameters": {
	                "userId": {"type": "string", "location": "path", "required": true}
	              },
	              "response": {"$ref": "Message"}
	            }
	          }
	        }
	      }
	    }
	  }
	}`
}

func discoveryMissingResponseSchemaDocument() string {
	return strings.Replace(discoveryResponseDocument(), `"Message": {`, `"UnavailableMessage": {`, 1)
}

func discoveryOAuthDocument() string {
	return `{
	  "kind": "discovery#restDescription",
	  "discoveryVersion": "v1",
	  "name": "gmail",
	  "title": "Gmail API",
	  "version": "v1",
	  "rootUrl": "https://gmail.googleapis.com/",
	  "servicePath": "",
	  "auth": {
	    "oauth2": {
	      "scopes": {
	        "https://www.googleapis.com/auth/gmail.send": {"description": "Send mail"}
	      }
	    }
	  },
	  "resources": {
	    "users": {
	      "resources": {
	        "messages": {
	          "methods": {
	            "send": {
	              "id": "gmail.users.messages.send",
	              "path": "gmail/v1/users/{userId}/messages/send",
	              "httpMethod": "POST",
	              "scopes": ["https://www.googleapis.com/auth/gmail.send"],
	              "parameters": {
	                "userId": {"type": "string", "location": "path", "required": true}
	              }
	            }
	          }
	        }
	      }
	    }
	  }
	}`
}

func smithyResponseBodyDocument() string {
	return `{
	  "smithy": "2.0",
	  "shapes": {
	    "com.amazonaws.lambda#Lambda": {
	      "type": "service",
	      "version": "2015-03-31",
	      "operations": [{"target": "com.amazonaws.lambda#GetFunction"}],
	      "traits": {
	        "aws.api#service": {"sdkId": "Lambda", "endpointPrefix": "lambda"},
	        "aws.auth#sigv4": {"name": "lambda"},
	        "aws.protocols#restJson1": {}
	      }
	    },
	    "com.amazonaws.lambda#GetFunction": {
	      "type": "operation",
	      "input": {"target": "com.amazonaws.lambda#GetFunctionRequest"},
	      "output": {"target": "com.amazonaws.lambda#GetFunctionResponse"},
	      "traits": {
	        "smithy.api#http": {"method": "GET", "uri": "/2015-03-31/functions/{FunctionName}", "code": 200}
	      }
	    },
	    "com.amazonaws.lambda#GetFunctionRequest": {
	      "type": "structure",
	      "members": {
	        "FunctionName": {"target": "com.amazonaws.lambda#FunctionName", "traits": {"smithy.api#httpLabel": {}, "smithy.api#required": {}}}
	      }
	    },
	    "com.amazonaws.lambda#GetFunctionResponse": {
	      "type": "structure",
	      "members": {
	        "Configuration": {"target": "com.amazonaws.lambda#FunctionConfiguration"},
	        "Payload": {"target": "smithy.api#Blob", "traits": {"smithy.api#httpPayload": {}}},
	        "RequestId": {"target": "smithy.api#String", "traits": {"smithy.api#httpHeader": "x-amzn-RequestId"}},
	        "StatusCode": {"target": "smithy.api#Integer", "traits": {"smithy.api#httpResponseCode": {}}}
	      }
	    },
	    "com.amazonaws.lambda#FunctionConfiguration": {
	      "type": "structure",
	      "members": {
	        "FunctionArn": {"target": "smithy.api#String"},
	        "Aliases": {"target": "com.amazonaws.lambda#AliasList"},
	        "Tags": {"target": "com.amazonaws.lambda#TagMap"}
	      }
	    },
	    "com.amazonaws.lambda#AliasList": {
	      "type": "list",
	      "member": {"target": "com.amazonaws.lambda#Alias"}
	    },
	    "com.amazonaws.lambda#Alias": {
	      "type": "structure",
	      "members": {
	        "name": {"target": "smithy.api#String"}
	      }
	    },
	    "com.amazonaws.lambda#TagMap": {
	      "type": "map",
	      "key": {"target": "smithy.api#String"},
	      "value": {"target": "smithy.api#String"}
	    },
	    "com.amazonaws.lambda#FunctionName": {"type": "string"}
	  }
	}`
}

func minimalSmithyDocument() string {
	return `{
	  "smithy": "2.0",
	  "shapes": {
	    "com.amazonaws.lambda#Lambda": {
	      "type": "service",
	      "version": "2015-03-31",
	      "operations": [{"target": "com.amazonaws.lambda#GetFunction"}],
	      "traits": {
	        "aws.api#service": {"sdkId": "Lambda", "endpointPrefix": "lambda"},
	        "aws.auth#sigv4": {"name": "lambda"},
	        "aws.protocols#restJson1": {}
	      }
	    },
	    "com.amazonaws.lambda#GetFunction": {
	      "type": "operation",
	      "input": {"target": "com.amazonaws.lambda#GetFunctionRequest"},
	      "traits": {
	        "smithy.api#http": {"method": "GET", "uri": "/2015-03-31/functions/{FunctionName}", "code": 200}
	      }
	    },
	    "com.amazonaws.lambda#GetFunctionRequest": {
	      "type": "structure",
	      "members": {
	        "FunctionName": {"target": "com.amazonaws.lambda#FunctionName", "traits": {"smithy.api#httpLabel": {}, "smithy.api#required": {}}},
	        "Qualifier": {"target": "com.amazonaws.lambda#Qualifier", "traits": {"smithy.api#httpQuery": "Qualifier"}}
	      }
	    },
	    "com.amazonaws.lambda#FunctionName": {"type": "string"},
	    "com.amazonaws.lambda#Qualifier": {"type": "string"}
	  }
	}`
}

func smithyWithoutSigV4Document() string {
	return `{
	  "smithy": "2.0",
	  "shapes": {
	    "com.amazonaws.lambda#Lambda": {
	      "type": "service",
	      "version": "2015-03-31",
	      "operations": [{"target": "com.amazonaws.lambda#GetFunction"}],
	      "traits": {
	        "aws.api#service": {"sdkId": "Lambda", "endpointPrefix": "lambda"},
	        "aws.protocols#restJson1": {}
	      }
	    },
	    "com.amazonaws.lambda#GetFunction": {
	      "type": "operation",
	      "input": {"target": "com.amazonaws.lambda#GetFunctionRequest"},
	      "traits": {
	        "smithy.api#http": {"method": "GET", "uri": "/2015-03-31/functions/{FunctionName}", "code": 200}
	      }
	    },
	    "com.amazonaws.lambda#GetFunctionRequest": {
	      "type": "structure",
	      "members": {
	        "FunctionName": {"target": "com.amazonaws.lambda#FunctionName", "traits": {"smithy.api#httpLabel": {}, "smithy.api#required": {}}}
	      }
	    },
	    "com.amazonaws.lambda#FunctionName": {"type": "string"}
	  }
	}`
}

func smithyRequiredParamsDocument() string {
	return `{
	  "smithy": "2.0",
	  "shapes": {
	    "com.example#Example": {
	      "type": "service",
	      "version": "2026-05-23",
	      "operations": [{"target": "com.example#PutThing"}],
	      "traits": {
	        "aws.api#service": {"sdkId": "Example", "endpointPrefix": "example"},
	        "aws.auth#sigv4": {"name": "example"},
	        "aws.protocols#restJson1": {}
	      }
	    },
	    "com.example#PutThing": {
	      "type": "operation",
	      "input": {"target": "com.example#PutThingRequest"},
	      "output": {"target": "com.example#PutThingResponse"},
	      "traits": {
	        "smithy.api#http": {"method": "PUT", "uri": "/things/{ThingId}", "code": 200}
	      }
	    },
	    "com.example#PutThingRequest": {
	      "type": "structure",
	      "members": {
	        "ThingId": {"target": "smithy.api#String", "traits": {"smithy.api#httpLabel": {}, "smithy.api#required": {}}},
	        "Mode": {"target": "smithy.api#String", "traits": {"smithy.api#httpQuery": "mode", "smithy.api#required": {}}},
	        "IfMatch": {"target": "smithy.api#String", "traits": {"smithy.api#httpHeader": "If-Match", "smithy.api#required": {}}},
	        "Filter": {"target": "smithy.api#String", "traits": {"smithy.api#httpQuery": "filter"}},
	        "Payload": {"target": "smithy.api#String", "traits": {"smithy.api#httpPayload": {}, "smithy.api#required": {}}},
	        "Description": {"target": "smithy.api#String", "traits": {"smithy.api#required": {}}},
	        "Note": {"target": "smithy.api#String"}
	      }
	    },
	    "com.example#PutThingResponse": {
	      "type": "structure",
	      "members": {
	        "Thing": {"target": "com.example#Thing"}
	      }
	    },
	    "com.example#Thing": {
	      "type": "structure",
	      "members": {
	        "ThingId": {"target": "smithy.api#String"},
	        "Status": {"target": "smithy.api#String"}
	      }
	    }
	  }
	}`
}

func writeSmithySourceForTest(t *testing.T, example, rel, content string) {
	t.Helper()
	path := filepath.Join(example, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
