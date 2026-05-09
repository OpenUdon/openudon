package tfconvert

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/authoring"
)

func TestConvertWritesDraftArtifacts(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
provider "aws" {
  alias  = "west"
  region = "us-west-2"
}

data "aws_ami" "base" {
  provider = aws.west
  owners   = ["self"]
}

resource "aws_instance" "web" {
  provider = aws.west
  ami      = data.aws_ami.base.id
  name     = var.name
}

variable "name" {
  default = "web"
}

output "instance_id" {
  value = aws_instance.web.id
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: AWS Test
  version: v1
paths:
  /amis/{id}:
    get:
      operationId: getAwsAmi
      responses:
        "200":
          description: ok
  /instances:
    post:
      operationId: createAwsInstance
      responses:
        "200":
          description: ok
`)

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "aws", Path: openAPIPath}},
		Action:    "create",
		OutDir:    filepath.Join(root, "out"),
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	for _, path := range []string{
		result.ProjectPath,
		result.IntentPath,
		result.WorkflowPath,
		result.UWSPath,
		result.PlanJSONPath,
		result.PlanMDPath,
		result.DiscoveryPath,
		result.DiagnosticsJSON,
		result.DiagnosticsMD,
		result.RefinementPath,
		result.ReviewPath,
		result.HandoffPath,
		result.QualityJSONPath,
		result.QualityMDPath,
		filepath.Join(result.OutDir, "openapi", "aws.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("artifact %s was not written: %v", path, err)
		}
	}
	intent := readFileForTest(t, result.IntentPath)
	for _, expected := range []string{"createAwsInstance", "getAwsAmi", "openapi/aws.yaml", "aws_west", "terraform_conversion_draft", "body.terraform.ami", "body.terraform.name", "body.terraform.owners"} {
		if !strings.Contains(intent, expected) {
			t.Fatalf("intent missing %q:\n%s", expected, intent)
		}
	}
	project := readFileForTest(t, result.ProjectPath)
	if !strings.Contains(project, "unapproved review scaffolding") || !strings.Contains(project, "aws_instance.web") {
		t.Fatalf("project did not summarize draft posture and resource:\n%s", project)
	}
	quality := readQualityForTest(t, result.QualityJSONPath)
	if quality.Status != "pass" || !result.QualityPassed {
		t.Fatalf("quality did not pass for resolved conversion: result=%t report=%#v", result.QualityPassed, quality)
	}
	handoff := readHandoffForTest(t, result.HandoffPath)
	if handoff.GeneratedState != string(authoring.ReviewStateGenerated) {
		t.Fatalf("handoff should remain generated, got %q", handoff.GeneratedState)
	}
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:   result.OutDir,
		Scope:  "terraform-conversion",
		Inputs: handoff.HandoffInputs,
	})
	if err != nil {
		t.Fatalf("normal package digest did not compute: %v", err)
	}
	if digest == "" {
		t.Fatal("normal package digest was empty")
	}
}

func TestConvertDiagnosesAmbiguousOperation(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
data "aws_ami" "base" {
  owners = ["self"]
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: AWS Test
  version: v1
paths:
  /amis/{id}:
    get:
      operationId: getAwsAmi
      responses:
        "200":
          description: ok
  /ami/{id}:
    get:
      operationId: readAwsAmi
      responses:
        "200":
          description: ok
`)

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "aws", Path: openAPIPath}},
		OutDir:    filepath.Join(root, "out"),
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if !hasDiagnostic(result.Diagnostics, "operation.ambiguous") {
		t.Fatalf("diagnostics missing operation.ambiguous: %#v", result.Diagnostics)
	}
	review := readFileForTest(t, result.ReviewPath)
	if !strings.Contains(review, "todo.data_aws_ami_base.read.read") {
		t.Fatalf("review missing deterministic TODO:\n%s", review)
	}
}

func TestConvertRedactsSensitiveCandidate(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "example_secret" "main" {
  api_token = "do-not-emit"
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: Secret Test
  version: v1
paths:
  /secrets:
    post:
      operationId: createExampleSecret
      responses:
        "200":
          description: ok
`)

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "secrets", Path: openAPIPath}},
		Action:    "create",
		OutDir:    filepath.Join(root, "out"),
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	for _, path := range []string{result.ProjectPath, result.IntentPath, result.DiagnosticsJSON, result.DiagnosticsMD, result.ReviewPath} {
		if text := readFileForTest(t, path); strings.Contains(text, "do-not-emit") {
			t.Fatalf("%s leaked sensitive literal:\n%s", path, text)
		}
	}
	if !hasDiagnostic(result.Diagnostics, "redaction.review_required") {
		t.Fatalf("diagnostics missing redaction review: %#v", result.Diagnostics)
	}
}

func TestConvertWritesFailingQualityForUnresolvedTODOs(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
data "aws_ami" "base" {
  owners = ["self"]
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: Unrelated
  version: v1
paths:
  /users:
    get:
      operationId: getUser
      responses:
        "200":
          description: ok
`)

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "users", Path: openAPIPath}},
		OutDir:    filepath.Join(root, "out"),
	})
	if err != nil {
		t.Fatalf("Convert returned error outside strict mode: %v", err)
	}
	quality := readQualityForTest(t, result.QualityJSONPath)
	if quality.Status != "fail" || result.QualityPassed {
		t.Fatalf("unresolved TODO should fail package quality: result=%t report=%#v", result.QualityPassed, quality)
	}
	if !qualityHasCheck(quality, "conversion.diagnostics", "fail") {
		t.Fatalf("quality missing conversion diagnostics failure: %#v", quality.Checks)
	}
	if _, err := os.Stat(result.WorkflowPath); err != nil {
		t.Fatalf("workflow.hcl should still be generated for review: %v", err)
	}
}

func TestConvertStrictModeFailsUnresolvedTODOs(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
data "aws_ami" "base" {
  owners = ["self"]
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: Unrelated
  version: v1
paths:
  /users:
    get:
      operationId: getUser
      responses:
        "200":
          description: ok
`)

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "users", Path: openAPIPath}},
		OutDir:    filepath.Join(root, "out"),
		Strict:    true,
	})
	if err == nil {
		t.Fatal("Convert succeeded in strict mode with unresolved TODO")
	}
	if !IsStrictFailure(err) {
		t.Fatalf("error is not strict failure: %T %v", err, err)
	}
	if result == nil || !result.StrictFailed {
		t.Fatalf("result did not report strict failure: %#v", result)
	}
	if _, statErr := os.Stat(result.DiagnosticsJSON); statErr != nil {
		t.Fatalf("strict conversion did not write diagnostics: %v", statErr)
	}
}

func TestConvertOutputIsDeterministic(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "aws_instance" "web" {
  name = "web"
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: AWS Test
  version: v1
paths:
  /instances:
    post:
      operationId: createAwsInstance
      responses:
        "200":
          description: ok
`)
	opts := Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "aws", Path: openAPIPath}},
		Action:    "create",
		OutDir:    filepath.Join(root, "out"),
	}
	first, err := Convert(context.Background(), opts)
	if err != nil {
		t.Fatalf("first Convert returned error: %v", err)
	}
	firstIntent := readFileForTest(t, first.IntentPath)
	firstDiagnostics := readFileForTest(t, first.DiagnosticsJSON)
	second, err := Convert(context.Background(), opts)
	if err != nil {
		t.Fatalf("second Convert returned error: %v", err)
	}
	if got := readFileForTest(t, second.IntentPath); got != firstIntent {
		t.Fatalf("intent output was not deterministic:\nfirst:\n%s\nsecond:\n%s", firstIntent, got)
	}
	if got := readFileForTest(t, second.DiagnosticsJSON); got != firstDiagnostics {
		t.Fatalf("diagnostics output was not deterministic:\nfirst:\n%s\nsecond:\n%s", firstDiagnostics, got)
	}
}

func TestConvertPrunesStaleStagedOpenAPIs(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	firstOpenAPI := filepath.Join(root, "first.yaml")
	secondOpenAPI := filepath.Join(root, "second.yaml")
	outDir := filepath.Join(root, "out")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "aws_instance" "web" {
  name = "web"
}
`)
	writeFileForTest(t, firstOpenAPI, `openapi: 3.0.0
info:
  title: First
  version: v1
paths:
  /instances:
    post:
      operationId: createAwsInstance
      responses:
        "200":
          description: ok
`)
	writeFileForTest(t, secondOpenAPI, `openapi: 3.0.0
info:
  title: Second
  version: v1
paths:
  /instances:
    post:
      operationId: createAwsInstance
      responses:
        "200":
          description: ok
`)

	if _, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "first", Path: firstOpenAPI}},
		Action:    "create",
		OutDir:    outDir,
	}); err != nil {
		t.Fatalf("first Convert returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "openapi", "first.yaml")); err != nil {
		t.Fatalf("first staged OpenAPI was not written: %v", err)
	}

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "second", Path: secondOpenAPI}},
		Action:    "create",
		OutDir:    outDir,
	})
	if err != nil {
		t.Fatalf("second Convert returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "openapi", "first.yaml")); !os.IsNotExist(err) {
		t.Fatalf("stale staged OpenAPI should have been pruned, stat err=%v", err)
	}
	handoff := readHandoffForTest(t, result.HandoffPath)
	if handoffHasInputForTest(handoff, "openapi/first.yaml") {
		t.Fatalf("handoff retained stale OpenAPI input: %#v", handoff.HandoffInputs)
	}
	if !handoffHasInputForTest(handoff, "openapi/second.yaml") {
		t.Fatalf("handoff missing current OpenAPI input: %#v", handoff.HandoffInputs)
	}
}

func TestConvertNamespacesDuplicateOperationIDsAcrossOpenAPIs(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	firstOpenAPI := filepath.Join(root, "first.yaml")
	secondOpenAPI := filepath.Join(root, "second.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "aws_instance" "web" {
  name = "web"
}
`)
	spec := `openapi: 3.0.0
info:
  title: AWS Test
  version: v1
paths:
  /instances:
    post:
      operationId: createAwsInstance
      responses:
        "200":
          description: ok
`
	writeFileForTest(t, firstOpenAPI, spec)
	writeFileForTest(t, secondOpenAPI, spec)
	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "first", Path: firstOpenAPI}, {ID: "second", Path: secondOpenAPI}},
		Action:    "create",
		OutDir:    filepath.Join(root, "out"),
	})
	if err != nil {
		t.Fatalf("Convert returned error for duplicate cross-document operation IDs: %v", err)
	}
	if hasDiagnostic(result.Diagnostics, "openapi.index_error") {
		t.Fatalf("cross-document duplicate operation IDs produced index error: %#v", result.Diagnostics)
	}
}

type qualityReportForTest struct {
	Status string                `json:"status"`
	Checks []qualityCheckForTest `json:"checks"`
}

type qualityCheckForTest struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}

func readQualityForTest(t *testing.T, path string) qualityReportForTest {
	t.Helper()
	var report qualityReportForTest
	if err := json.Unmarshal([]byte(readFileForTest(t, path)), &report); err != nil {
		t.Fatal(err)
	}
	return report
}

func readHandoffForTest(t *testing.T, path string) authoring.ReviewHandoff {
	t.Helper()
	var handoff authoring.ReviewHandoff
	if err := json.Unmarshal([]byte(readFileForTest(t, path)), &handoff); err != nil {
		t.Fatal(err)
	}
	return handoff
}

func handoffHasInputForTest(handoff authoring.ReviewHandoff, path string) bool {
	for _, input := range handoff.HandoffInputs {
		if input.Path == path {
			return true
		}
	}
	return false
}

func qualityHasCheck(report qualityReportForTest, code, status string) bool {
	for _, check := range report.Checks {
		if check.Code == code && check.Status == status {
			return true
		}
	}
	return false
}

func hasDiagnostic(diags []Diagnostic, code string) bool {
	for _, diag := range diags {
		if diag.Code == code {
			return true
		}
	}
	return false
}

func writeFileForTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
