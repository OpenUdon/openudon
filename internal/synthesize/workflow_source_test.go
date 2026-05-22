package synthesize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
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

func minimalDiscoveryDocument() string {
	return `{
	  "kind": "discovery#restDescription",
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
