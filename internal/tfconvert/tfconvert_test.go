package tfconvert

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	for _, path := range []string{result.ProjectPath, result.IntentPath, result.DiagnosticsJSON, result.DiagnosticsMD, result.ReviewPath} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("artifact %s was not written: %v", path, err)
		}
	}
	intent := readFileForTest(t, result.IntentPath)
	for _, expected := range []string{"createAwsInstance", "getAwsAmi", "aws_west", "terraform_conversion_draft"} {
		if !strings.Contains(intent, expected) {
			t.Fatalf("intent missing %q:\n%s", expected, intent)
		}
	}
	project := readFileForTest(t, result.ProjectPath)
	if !strings.Contains(project, "unapproved review scaffolding") || !strings.Contains(project, "aws_instance.web") {
		t.Fatalf("project did not summarize draft posture and resource:\n%s", project)
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
