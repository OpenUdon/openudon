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

func TestConvertAWSProviderS3SingleOpenAPICorpus(t *testing.T) {
	tests := []struct {
		name                  string
		config                string
		expectedOperations    []string
		unexpectedDiagnostics []string
	}{
		{
			name: "bucket accelerate configuration",
			config: `
resource "aws_s3_bucket" "test" {
  bucket = "tf-acc-openudon-bucket"
}

resource "aws_s3_bucket_accelerate_configuration" "test" {
  bucket = aws_s3_bucket.test.bucket
  status = "Enabled"
}
`,
			expectedOperations: []string{"CreateBucket", "PutBucketAccelerateConfiguration"},
			unexpectedDiagnostics: []string{
				"operation.ambiguous",
				"operation.unresolved",
			},
		},
		{
			name: "bucket data source",
			config: `
resource "aws_s3_bucket" "test" {
  bucket = "tf-acc-openudon-bucket-ds"
}

data "aws_s3_bucket" "test" {
  bucket = aws_s3_bucket.test.id
}
`,
			expectedOperations: []string{"CreateBucket", "GetBucketLocation"},
			unexpectedDiagnostics: []string{
				"operation.ambiguous",
				"operation.unresolved",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			configDir := filepath.Join(root, "tf")
			openAPIPath := filepath.Join(root, "s3.yaml")
			writeFileForTest(t, filepath.Join(configDir, "main.tf"), tt.config)
			writeFileForTest(t, openAPIPath, s3OpenAPIForTest())

			result, err := Convert(context.Background(), Options{
				ConfigDir: configDir,
				OpenAPIs:  []OpenAPIInput{{ID: "s3", Path: openAPIPath}},
				Action:    "create",
				OutDir:    filepath.Join(root, "out"),
			})
			if err != nil {
				t.Fatalf("Convert returned error: %v", err)
			}
			for _, code := range tt.unexpectedDiagnostics {
				if hasDiagnostic(result.Diagnostics, code) {
					t.Fatalf("diagnostics should not contain %s: %#v", code, result.Diagnostics)
				}
			}
			intent := readFileForTest(t, result.IntentPath)
			workflow := readFileForTest(t, result.WorkflowPath)
			for _, operationID := range tt.expectedOperations {
				if !strings.Contains(intent, operationID) || !strings.Contains(workflow, operationID) {
					t.Fatalf("expected operation %q in intent and workflow\nintent:\n%s\nworkflow:\n%s", operationID, intent, workflow)
				}
			}
			for _, text := range []string{intent, workflow, readFileForTest(t, result.ReviewPath)} {
				if strings.Contains(text, "todo.") {
					t.Fatalf("known S3 corpus case should not emit operation TODOs:\n%s", text)
				}
			}
			quality := readQualityForTest(t, result.QualityJSONPath)
			for _, code := range []string{"intent.openapi_operations", "conversion.diagnostics"} {
				if qualityHasCheck(quality, code, "fail") {
					t.Fatalf("quality should not fail %s after S3 operation mapping: %#v", code, quality.Checks)
				}
			}
		})
	}
}

func TestConvertAWSProviderLambdaFunctionURLMultiOpenAPICorpus(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	iamOpenAPI := filepath.Join(root, "iam.yaml")
	lambdaOpenAPI := filepath.Join(root, "lambda.yaml")
	stsOpenAPI := filepath.Join(root, "sts.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
data "aws_partition" "current" {}

resource "aws_iam_role_policy" "iam_policy_for_lambda" {
  name = "tf-acc-openudon-lambda-policy"
  role = aws_iam_role.iam_for_lambda.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["logs:CreateLogGroup"],
    "Resource": "arn:${data.aws_partition.current.partition}:logs:*:*:*"
  }]
}
EOF
}

resource "aws_iam_role" "iam_for_lambda" {
  name = "tf-acc-openudon-lambda-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [{
    "Action": "sts:AssumeRole",
    "Principal": {"Service": "lambda.amazonaws.com"},
    "Effect": "Allow"
  }]
}
EOF
}

resource "aws_lambda_function" "test" {
  filename      = "test-fixtures/lambdatest.zip"
  function_name = "tf-acc-openudon-lambda"
  role          = aws_iam_role.iam_for_lambda.arn
  handler       = "exports.example"
  runtime       = "nodejs24.x"
}

resource "aws_lambda_function_url" "test" {
  function_name      = aws_lambda_function.test.function_name
  authorization_type = "NONE"
}
`)
	writeFileForTest(t, iamOpenAPI, iamOpenAPIForTest())
	writeFileForTest(t, lambdaOpenAPI, lambdaOpenAPIForTest())
	writeFileForTest(t, stsOpenAPI, stsOpenAPIForTest())

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs: []OpenAPIInput{
			{ID: "iam", Path: iamOpenAPI},
			{ID: "lambda", Path: lambdaOpenAPI},
			{ID: "sts", Path: stsOpenAPI},
		},
		Action: "create",
		OutDir: filepath.Join(root, "out"),
		Strict: true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	for _, code := range []string{"operation.ambiguous", "operation.unresolved"} {
		if hasDiagnostic(result.Diagnostics, code) {
			t.Fatalf("diagnostics should not contain %s: %#v", code, result.Diagnostics)
		}
	}

	intent := readFileForTest(t, result.IntentPath)
	workflow := readFileForTest(t, result.WorkflowPath)
	review := readFileForTest(t, result.ReviewPath)
	project := readFileForTest(t, result.ProjectPath)
	for _, expected := range []string{
		"POST_CreateRole",
		"POST_PutRolePolicy",
		"CreateFunction",
		"CreateFunctionUrlConfig",
		"openapi/iam.yaml",
		"openapi/lambda.yaml",
		"aws_hmac",
		"Action",
		"Version",
		"CreateRole",
		"PutRolePolicy",
		"2010-05-08",
	} {
		if !strings.Contains(intent, expected) || !strings.Contains(workflow, expected) {
			t.Fatalf("expected %q in intent and workflow\nintent:\n%s\nworkflow:\n%s", expected, intent, workflow)
		}
	}
	if !strings.Contains(workflow, "FunctionName") || !strings.Contains(workflow, "aws_lambda_function_url_test_create_function_name") {
		t.Fatalf("workflow should bind Lambda FunctionName path parameter from function_name:\n%s", workflow)
	}
	for _, text := range []string{intent, workflow, review} {
		if strings.Contains(text, "todo.") || strings.Contains(text, "aws_partition.current_read") {
			t.Fatalf("known Lambda/IAM corpus case should not emit operation TODOs or partition operations:\n%s", text)
		}
	}
	if !strings.Contains(project, "data.aws_partition.current") || !strings.Contains(project, "provider-local metadata") {
		t.Fatalf("project should classify aws_partition as provider-local metadata:\n%s", project)
	}
	quality := readQualityForTest(t, result.QualityJSONPath)
	for _, code := range []string{"intent.openapi_operations", "conversion.diagnostics", "workflow.plan_match", "workflow.credentials_bound"} {
		if qualityHasCheck(quality, code, "fail") {
			t.Fatalf("quality should not fail %s after AWS multi-document mapping: %#v", code, quality.Checks)
		}
	}
}

func TestConvertAWSCredentialBindingPreservesProviderAlias(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "s3.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
provider "aws" {
  alias  = "west"
  region = "us-west-2"
}

resource "aws_s3_bucket" "test" {
  provider = aws.west
  bucket   = "tf-acc-openudon-alias"
}
`)
	writeFileForTest(t, openAPIPath, s3OpenAPIWithHMACForTest())

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "s3", Path: openAPIPath}},
		Action:    "create",
		OutDir:    filepath.Join(root, "out"),
		Strict:    true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	intent := readFileForTest(t, result.IntentPath)
	workflow := readFileForTest(t, result.WorkflowPath)
	project := readFileForTest(t, result.ProjectPath)
	for _, expected := range []string{"aws_west_hmac", "aws_west"} {
		if !strings.Contains(intent, expected) || !strings.Contains(workflow, expected) || !strings.Contains(project, expected) {
			t.Fatalf("expected aliased credential binding %q\nintent:\n%s\nworkflow:\n%s\nproject:\n%s", expected, intent, workflow, project)
		}
	}
	if strings.Contains(workflow, `Authorization = "aws_hmac"`) {
		t.Fatalf("workflow collapsed aliased AWS credential to default aws_hmac:\n%s", workflow)
	}
}

func TestConvertAWSCallerIdentityDataSourceUsesSTS(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	stsOpenAPI := filepath.Join(root, "sts.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
data "aws_caller_identity" "current" {}
`)
	writeFileForTest(t, stsOpenAPI, stsOpenAPIForTest())

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "sts", Path: stsOpenAPI}},
		OutDir:    filepath.Join(root, "out"),
		Strict:    true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if hasDiagnostic(result.Diagnostics, "operation.ambiguous") || hasDiagnostic(result.Diagnostics, "operation.unresolved") {
		t.Fatalf("caller identity should map to STS without operation diagnostics: %#v", result.Diagnostics)
	}
	intent := readFileForTest(t, result.IntentPath)
	workflow := readFileForTest(t, result.WorkflowPath)
	for _, expected := range []string{"POST_GetCallerIdentity", "openapi/sts.yaml", "GetCallerIdentity", "2011-06-15", "aws_hmac"} {
		if !strings.Contains(intent, expected) || !strings.Contains(workflow, expected) {
			t.Fatalf("expected %q in caller identity intent and workflow\nintent:\n%s\nworkflow:\n%s", expected, intent, workflow)
		}
	}
	project := readFileForTest(t, result.ProjectPath)
	if strings.Contains(project, "data.aws_caller_identity.current` is provider-local metadata") {
		t.Fatalf("caller identity should not be classified as provider-local metadata:\n%s", project)
	}
}

func TestConvertAWSIAMRoleUsesNativeSmithySource(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	smithyPath := filepath.Join(root, "iam.json")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "aws_iam_role" "role" {
  name = "tf-acc-openudon-role"
  assume_role_policy = "{}"
}
`)
	writeFileForTest(t, smithyPath, minimalIAMSmithyForTest())

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		APISources: []APISourceInput{{
			Kind: "aws-smithy",
			ID:   "iam",
			Path: smithyPath,
		}},
		Action: "create",
		OutDir: filepath.Join(root, "out"),
		Strict: true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	intent := readFileForTest(t, result.IntentPath)
	workflow := readFileForTest(t, result.WorkflowPath)
	for _, expected := range []string{"aws-smithy/iam.json", "CreateRole", "RoleName", "AssumeRolePolicyDocument"} {
		if !strings.Contains(intent, expected) || !strings.Contains(workflow, expected) {
			t.Fatalf("expected native Smithy mapping %q\nintent:\n%s\nworkflow:\n%s", expected, intent, workflow)
		}
	}
	if !strings.Contains(intent, "aws_hmac") {
		t.Fatalf("intent missing symbolic AWS credential binding:\n%s", intent)
	}
	if _, err := os.Stat(filepath.Join(result.OutDir, "openapi")); !os.IsNotExist(err) {
		t.Fatalf("native Smithy conversion should not stage OpenAPI fallback, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(result.OutDir, "aws-smithy", "iam.json")); err != nil {
		t.Fatalf("staged Smithy source missing: %v", err)
	}
}

func TestConvertGoogleStorageBucketUsesNativeDiscoverySource(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	discoveryPath := filepath.Join(root, "storage.json")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "google_storage_bucket" "bucket" {
  name     = "openudon-bucket"
  location = "US"
  project  = "review-project"
}
`)
	writeFileForTest(t, discoveryPath, minimalStorageDiscoveryForTest())

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		APISources: []APISourceInput{{
			Kind: "google-discovery",
			ID:   "storage",
			Path: discoveryPath,
		}},
		Action: "create",
		OutDir: filepath.Join(root, "out"),
		Strict: true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	intent := readFileForTest(t, result.IntentPath)
	workflow := readFileForTest(t, result.WorkflowPath)
	for _, expected := range []string{"google-discovery/storage.json", "storage.buckets.insert", "project", "location"} {
		if !strings.Contains(intent, expected) || !strings.Contains(workflow, expected) {
			t.Fatalf("expected native Discovery mapping %q\nintent:\n%s\nworkflow:\n%s", expected, intent, workflow)
		}
	}
	if !strings.Contains(intent, "google_oauth2") {
		t.Fatalf("intent missing symbolic Google credential binding:\n%s", intent)
	}
	if _, err := os.Stat(filepath.Join(result.OutDir, "openapi")); !os.IsNotExist(err) {
		t.Fatalf("native Discovery conversion should not stage OpenAPI fallback, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(result.OutDir, "google-discovery", "storage.json")); err != nil {
		t.Fatalf("staged Discovery source missing: %v", err)
	}
}

func TestConvertRedactsSensitiveCandidate(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "example_secret" "main" {
  api_token = "do-not-emit"
  config = {
    password = "nested-secret-do-not-emit"
  }
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
		text := readFileForTest(t, path)
		for _, leaked := range []string{"do-not-emit", "nested-secret-do-not-emit"} {
			if strings.Contains(text, leaked) {
				t.Fatalf("%s leaked sensitive literal %q:\n%s", path, leaked, text)
			}
		}
	}
	if !hasDiagnostic(result.Diagnostics, "redaction.review_required") {
		t.Fatalf("diagnostics missing redaction review: %#v", result.Diagnostics)
	}
}

func TestConvertRejectsOpenAPIInputInsideStagingDir(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi", "app.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "example_resource" "main" {
  name = "web"
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: Staging Safety
  version: v1
paths:
  /resources:
    post:
      operationId: createExampleResource
      responses:
        "200":
          description: ok
`)

	_, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "app", Path: openAPIPath}},
		Action:    "create",
		OutDir:    root,
	})
	if err == nil {
		t.Fatal("Convert returned nil error for OpenAPI input inside staging directory")
	}
	if !strings.Contains(err.Error(), "staging directory") {
		t.Fatalf("error did not describe staging overlap: %v", err)
	}
	if text := readFileForTest(t, openAPIPath); !strings.Contains(text, "Staging Safety") {
		t.Fatalf("OpenAPI source was modified or deleted:\n%s", text)
	}
}

func TestConvertRejectsMalformedOpenAPIInputInsideStagingDirWithoutDeletingIt(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi", "app.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "example_resource" "main" {
  name = "web"
}
`)
	writeFileForTest(t, openAPIPath, `not: openapi
`)

	_, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "app", Path: openAPIPath}},
		Action:    "create",
		OutDir:    root,
	})
	if err == nil {
		t.Fatal("Convert returned nil error for malformed OpenAPI input inside staging directory")
	}
	if !strings.Contains(err.Error(), "staging directory") {
		t.Fatalf("error did not describe staging overlap: %v", err)
	}
	if text := readFileForTest(t, openAPIPath); !strings.Contains(text, "not: openapi") {
		t.Fatalf("malformed OpenAPI source was modified or deleted:\n%s", text)
	}
}

func TestConvertRejectsUnownedPreexistingAPISourceStagingDir(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "source.yaml")
	outDir := filepath.Join(root, "out")
	unownedPath := filepath.Join(outDir, "openapi", "keep.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "example_resource" "main" {
  name = "web"
}
`)
	writeFileForTest(t, openAPIPath, `openapi: 3.0.0
info:
  title: Source
  version: v1
paths:
  /resources:
    post:
      operationId: createExampleResource
      responses:
        "200":
          description: ok
`)
	writeFileForTest(t, unownedPath, "do not delete\n")

	_, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs:  []OpenAPIInput{{ID: "source", Path: openAPIPath}},
		Action:    "create",
		OutDir:    outDir,
	})
	if err == nil {
		t.Fatal("Convert returned nil error for unowned pre-existing API source staging directory")
	}
	if !strings.Contains(err.Error(), "not marked as owned") {
		t.Fatalf("error did not describe ownership marker failure: %v", err)
	}
	if got := readFileForTest(t, unownedPath); got != "do not delete\n" {
		t.Fatalf("unowned API source staging content was modified or deleted: %q", got)
	}
}

func TestConvertDiagnosesOpenAPIPackagePathCollision(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	firstOpenAPIPath := filepath.Join(root, "first.yaml")
	secondOpenAPIPath := filepath.Join(root, "second.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "example_resource" "main" {
  name = "web"
}
`)
	writeFileForTest(t, firstOpenAPIPath, `openapi: 3.0.0
info:
  title: First
  version: v1
paths:
  /resources:
    post:
      operationId: createExampleResource
      responses:
        "200":
          description: ok
`)
	writeFileForTest(t, secondOpenAPIPath, `openapi: 3.0.0
info:
  title: Second
  version: v1
paths:
  /other:
    post:
      operationId: createOther
      responses:
        "200":
          description: ok
`)

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs: []OpenAPIInput{
			{ID: "a-b", Path: firstOpenAPIPath},
			{ID: "a_b", Path: secondOpenAPIPath},
		},
		Action: "create",
		OutDir: filepath.Join(root, "out"),
		Strict: true,
	})
	if !IsStrictFailure(err) {
		t.Fatalf("Convert error = %v, want strict failure", err)
	}
	if result == nil || !hasDiagnostic(result.Diagnostics, "api_source.package_path_collision") {
		t.Fatalf("diagnostics missing package path collision: result=%#v", result)
	}
	if staged := readFileForTest(t, filepath.Join(result.OutDir, "openapi", "a_b.yaml")); !strings.Contains(staged, "First") || strings.Contains(staged, "Second") {
		t.Fatalf("staged OpenAPI was overwritten:\n%s", staged)
	}
}

func TestConvertStrictMissingAPISourceReturnsStrictFailureBeforeSynthesis(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "aws_instance" "web" {
  name = "web"
}
`)

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		Action:    "create",
		OutDir:    filepath.Join(root, "out"),
		Strict:    true,
	})
	if !IsStrictFailure(err) {
		t.Fatalf("Convert error = %T %v, want strict failure", err, err)
	}
	if result == nil || !result.StrictFailed || !hasDiagnostic(result.Diagnostics, "api_source.missing") {
		t.Fatalf("result missing strict api_source.missing diagnostic: %#v", result)
	}
	if _, statErr := os.Stat(result.DiagnosticsJSON); statErr != nil {
		t.Fatalf("strict conversion did not write diagnostics: %v", statErr)
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

func TestConvertAWSHardcodedMappingPrefersExpectedSourceID(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	wrongOpenAPI := filepath.Join(root, "aaa.yaml")
	iamOpenAPI := filepath.Join(root, "iam.yaml")
	writeFileForTest(t, filepath.Join(configDir, "main.tf"), `
resource "aws_iam_role" "role" {
  name = "tf-acc-openudon-role"
}
`)
	writeFileForTest(t, wrongOpenAPI, `openapi: 3.0.0
info:
  title: Wrong Service
  version: v1
paths:
  /wrong:
    post:
      operationId: POST_CreateRole
      responses:
        "200":
          description: ok
`)
	writeFileForTest(t, iamOpenAPI, iamOpenAPIForTest())

	result, err := Convert(context.Background(), Options{
		ConfigDir: configDir,
		OpenAPIs: []OpenAPIInput{
			{ID: "aaa", Path: wrongOpenAPI},
			{ID: "iam", Path: iamOpenAPI},
		},
		Action: "create",
		OutDir: filepath.Join(root, "out"),
		Strict: true,
	})
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if hasDiagnostic(result.Diagnostics, "operation.ambiguous") || hasDiagnostic(result.Diagnostics, "operation.unresolved") {
		t.Fatalf("expected IAM source ID to disambiguate duplicate operation ID: %#v", result.Diagnostics)
	}
	intent := readFileForTest(t, result.IntentPath)
	if !strings.Contains(intent, `source    = "openapi/iam.yaml"`) {
		t.Fatalf("IAM operation did not bind to iam source:\n%s", intent)
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

func s3OpenAPIForTest() string {
	return `openapi: 3.0.0
info:
  title: Amazon Simple Storage Service
  version: "2006-03-01"
paths:
  /:
    get:
      operationId: ListBuckets
      responses:
        "200":
          description: ok
  /{Bucket}:
    put:
      operationId: CreateBucket
      parameters:
        - name: Bucket
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
    head:
      operationId: HeadBucket
      parameters:
        - name: Bucket
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
  /{Bucket}#accelerate:
    get:
      operationId: GetBucketAccelerateConfiguration
      parameters:
        - name: Bucket
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
    put:
      operationId: PutBucketAccelerateConfiguration
      parameters:
        - name: Bucket
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
  /{Bucket}#location:
    get:
      operationId: GetBucketLocation
      parameters:
        - name: Bucket
          in: path
          required: true
          schema:
            type: string
      responses:
        "200":
          description: ok
`
}

func s3OpenAPIWithHMACForTest() string {
	return strings.Replace(s3OpenAPIForTest(), "paths:\n", `security:
  - hmac: []
paths:
`, 1) + `components:
  securitySchemes:
    hmac:
      type: apiKey
      name: Authorization
      in: header
`
}

func iamOpenAPIForTest() string {
	return `openapi: 3.0.0
info:
  title: AWS Identity and Access Management
  version: "2010-05-08"
security:
  - hmac: []
paths:
  /#Action=CreateRole:
    post:
      operationId: POST_CreateRole
      parameters:
        - name: Action
          in: query
          required: true
          schema:
            type: string
            enum: [CreateRole]
        - name: Version
          in: query
          required: true
          schema:
            type: string
            enum: ["2010-05-08"]
      responses:
        "200":
          description: ok
  /#Action=PutRolePolicy:
    post:
      operationId: POST_PutRolePolicy
      parameters:
        - name: Action
          in: query
          required: true
          schema:
            type: string
            enum: [PutRolePolicy]
        - name: Version
          in: query
          required: true
          schema:
            type: string
            enum: ["2010-05-08"]
      responses:
        "200":
          description: ok
components:
  securitySchemes:
    hmac:
      type: apiKey
      name: Authorization
      in: header
`
}

func lambdaOpenAPIForTest() string {
	return `openapi: 3.0.0
info:
  title: AWS Lambda
  version: "2015-03-31"
security:
  - hmac: []
paths:
  /2015-03-31/functions:
    post:
      operationId: CreateFunction
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                FunctionName:
                  type: string
                Role:
                  type: string
                Runtime:
                  type: string
                Handler:
                  type: string
      responses:
        "201":
          description: ok
  /2021-10-31/functions/{FunctionName}/url:
    post:
      operationId: CreateFunctionUrlConfig
      parameters:
        - name: FunctionName
          in: path
          required: true
          schema:
            type: string
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                authorization_type:
                  type: string
      responses:
        "201":
          description: ok
components:
  securitySchemes:
    hmac:
      type: apiKey
      name: Authorization
      in: header
`
}

func stsOpenAPIForTest() string {
	return `openapi: 3.0.0
info:
  title: AWS Security Token Service
  version: "2011-06-15"
security:
  - hmac: []
paths:
  /assume-role:
    post:
      operationId: POST_AssumeRole
      responses:
        "200":
          description: ok
  /#Action=GetCallerIdentity:
    post:
      operationId: POST_GetCallerIdentity
      parameters:
        - name: Action
          in: query
          required: true
          schema:
            type: string
            enum: [GetCallerIdentity]
        - name: Version
          in: query
          required: true
          schema:
            type: string
            enum: ["2011-06-15"]
      responses:
        "200":
          description: ok
components:
  securitySchemes:
    hmac:
      type: apiKey
      name: Authorization
      in: header
`
}

func minimalIAMSmithyForTest() string {
	return `{
  "smithy": "2.0",
  "shapes": {
    "com.amazonaws.iam#IAM": {
      "type": "service",
      "version": "2010-05-08",
      "operations": [{"target": "com.amazonaws.iam#CreateRole"}],
      "traits": {
        "aws.api#service": {"sdkId": "IAM", "endpointPrefix": "iam"},
        "aws.auth#sigv4": {"name": "iam"},
        "aws.protocols#awsQuery": {}
      }
    },
    "com.amazonaws.iam#CreateRole": {
      "type": "operation",
      "input": {"target": "com.amazonaws.iam#CreateRoleRequest"},
      "output": {"target": "com.amazonaws.iam#CreateRoleResponse"}
    },
    "com.amazonaws.iam#CreateRoleRequest": {
      "type": "structure",
      "members": {
        "RoleName": {
          "target": "com.amazonaws.iam#roleNameType",
          "traits": {"smithy.api#required": {}}
        },
        "AssumeRolePolicyDocument": {
          "target": "com.amazonaws.iam#policyDocumentType",
          "traits": {"smithy.api#required": {}}
        }
      },
      "traits": {"smithy.api#input": {}}
    },
    "com.amazonaws.iam#CreateRoleResponse": {"type": "structure", "members": {}},
    "com.amazonaws.iam#roleNameType": {"type": "string"},
    "com.amazonaws.iam#policyDocumentType": {"type": "string"}
  }
}`
}

func minimalStorageDiscoveryForTest() string {
	return `{
  "discoveryVersion": "v1",
  "name": "storage",
  "version": "v1",
  "rootUrl": "https://storage.googleapis.com/",
  "servicePath": "storage/v1/",
  "schemas": {
    "Bucket": {
      "id": "Bucket",
      "type": "object",
      "properties": {
        "name": {"type": "string"},
        "location": {"type": "string"}
      }
    }
  },
  "resources": {
    "buckets": {
      "methods": {
        "insert": {
          "id": "storage.buckets.insert",
          "path": "b",
          "httpMethod": "POST",
          "parameters": {
            "project": {
              "type": "string",
              "required": true,
              "location": "query"
            }
          },
          "request": {"$ref": "Bucket"},
          "response": {"$ref": "Bucket"},
          "scopes": ["https://www.googleapis.com/auth/devstorage.full_control"]
        }
      }
    }
  }
}`
}
