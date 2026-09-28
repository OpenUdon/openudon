package stepauthoring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/uws/uws1"
)

func nestedRenameContract() StepContract {
	return StepContract{
		ID:      "project_read_contract",
		Purpose: "List projects for the requested location",
		Inputs: &uws1.ParamSchema{
			Type:     "object",
			Required: []string{"pagination"},
			Properties: map[string]*uws1.ParamSchema{
				"pagination": {
					Type:     "object",
					Required: []string{"limit"},
					Properties: map[string]*uws1.ParamSchema{
						"limit": {Type: "integer"},
					},
				},
			},
		},
		Outputs: &uws1.ParamSchema{
			Type:     "object",
			Required: []string{"projects"},
			Properties: map[string]*uws1.ParamSchema{
				"projects": {Type: "array"},
			},
		},
		Effect: "read",
	}
}

func prepareNestedRenamePackage(t *testing.T) (string, []byte, []byte) {
	t.Helper()
	root := copyRunnableExample(t)
	sourcePath := filepath.Join(root, "openapi", "project-api.yaml")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	oldResponse := `                required: [projects]
                properties:
                  projects:
                    type: array
                    items:
                      type: object
                      properties:
                        id:
                          type: string`
	newResponse := `                required: [data]
                properties:
                  data:
                    type: object
                    required: [items]
                    properties:
                      items:
                        type: array
                        items:
                          type: object
                          properties:
                            id:
                              type: string`
	updatedSource := strings.Replace(string(source), oldResponse, newResponse, 1)
	if updatedSource == string(source) {
		t.Fatal("nested response fixture was not changed")
	}
	if err := os.WriteFile(sourcePath, []byte(updatedSource), 0o600); err != nil {
		t.Fatal(err)
	}
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intent, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	updatedIntent := strings.Replace(string(intent), `input "page_size"`, `input "pagination"`, 1)
	updatedIntent = strings.Replace(updatedIntent, `type     = "integer"`, `type     = "object"`, 1)
	updatedIntent = strings.Replace(updatedIntent, `page_size = "inputs.page_size"`, `"query.page_size" = "inputs.pagination.limit"`, 1)
	updatedIntent = strings.Replace(updatedIntent, `list_projects.received_body.projects`, `list_projects.received_body.data.items`, 1)
	if updatedIntent == string(intent) || !strings.Contains(updatedIntent, `"query.page_size" = "inputs.pagination.limit"`) {
		t.Fatal("nested intent fixture was not changed completely")
	}
	if err := os.WriteFile(intentPath, []byte(updatedIntent), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, []byte(updatedSource), []byte(updatedIntent)
}

func setNestedRenameCheckRequest(t *testing.T, request *CheckRequest, source, intent []byte) {
	t.Helper()
	request.Contract = nestedRenameContract()
	request.IntentSHA256 = "sha256:" + evidencefile.SHA256(intent)
	request.OperationRef.SourceSHA256 = "sha256:" + evidencefile.SHA256(source)
	request.OutputMappings = map[string]string{"projects": "received_body.data.items"}
}

func TestCheckAcceptsExplicitNestedAndRenamedMappings(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root, source, intent := prepareNestedRenamePackage(t)
	request := readRunnableRequest(t, fixtureRoot)
	setNestedRenameCheckRequest(t, &request, source, intent)
	outcome := Check(nil, root, request)
	if outcome.ExitCode != 0 || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "compatible" {
		t.Fatalf("nested/renamed mapping check = %#v", outcome.Result.Result)
	}
	if !hasCheckItem(outcome.Result.Result.Checks, "mapping.input_contract", "pass") ||
		!hasCheckItem(outcome.Result.Result.Checks, "mapping.output_contract", "pass") {
		t.Fatalf("nested/renamed compatibility evidence missing: %#v", outcome.Result.Result.Checks)
	}
}

func TestCheckMatchesPublishedNestedMappingFixture(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	requestBytes, err := os.ReadFile(filepath.Join(fixtureRoot, "requests", "step-check-nested-mapping.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request CheckRequest
	if err := evidencefile.DecodeStrict(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	result := Check(nil, filepath.Join(fixtureRoot, "example-nested-mapping"), request)
	if result.ExitCode != 0 || result.Result.Result == nil || result.Result.Result.Assessment != "compatible" {
		t.Fatalf("nested mapping fixture result = %#v", result.Result.Result)
	}
	gotBytes, err := json.Marshal(result.Result)
	if err != nil {
		t.Fatal(err)
	}
	wantBytes, err := os.ReadFile(filepath.Join(fixtureRoot, "results", "step-check-nested-mapping.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("nested mapping result differs from published fixture\ngot:  %s\nwant: %s", gotBytes, wantBytes)
	}
}

func TestCheckRejectsNestedInputTypeAndRequirednessConflicts(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	for _, test := range []struct {
		name   string
		mutate func(*CheckRequest)
	}{
		{"type mismatch", func(request *CheckRequest) {
			request.Contract.Inputs.Properties["pagination"].Properties["limit"].Type = "string"
		}},
		{"required source mapped to optional nested contract field", func(request *CheckRequest) { request.Contract.Inputs.Properties["pagination"].Required = nil }},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, source, intent := prepareNestedRenamePackage(t)
			request := readRunnableRequest(t, fixtureRoot)
			setNestedRenameCheckRequest(t, &request, source, intent)
			test.mutate(&request)
			outcome := Check(nil, root, request)
			if outcome.ExitCode != 0 || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
				t.Fatalf("mapping conflict result = %#v", outcome.Result)
			}
			if !hasCheckItem(outcome.Result.Result.Checks, "mapping.input_contract", "fail") {
				t.Fatalf("mapping conflict was not identified: %#v", outcome.Result.Result.Checks)
			}
			if !hasCheckPointer(outcome.Result.Result.Checks, "mapping.input_contract", "fail", "/contract/inputs/pagination/limit") {
				t.Fatalf("mapping conflict did not identify its contract field: %#v", outcome.Result.Result.Checks)
			}
		})
	}
}

func TestCheckRejectsNestedOutputTypeAndRequirednessConflicts(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	for _, test := range []struct {
		name string
		edit func(string) string
	}{
		{"type mismatch", func(source string) string {
			return strings.Replace(source, "type: array\n                        items:", "type: string\n                        items:", 1)
		}},
		{"required contract output backed by optional response field", func(source string) string {
			return strings.Replace(source, "required: [items]\n                    properties:", "required: []\n                    properties:", 1)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root, source, intent := prepareNestedRenamePackage(t)
			updatedSource := []byte(test.edit(string(source)))
			if string(updatedSource) == string(source) {
				t.Fatal("source fixture did not change")
			}
			if err := os.WriteFile(filepath.Join(root, "openapi", "project-api.yaml"), updatedSource, 0o600); err != nil {
				t.Fatal(err)
			}
			request := readRunnableRequest(t, fixtureRoot)
			setNestedRenameCheckRequest(t, &request, updatedSource, intent)
			outcome := Check(nil, root, request)
			if outcome.ExitCode != 0 || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
				t.Fatalf("output mapping conflict result = %#v", outcome.Result)
			}
			if !hasCheckItem(outcome.Result.Result.Checks, "mapping.output_contract", "fail") {
				t.Fatalf("output mapping conflict was not identified: %#v", outcome.Result.Result.Checks)
			}
			if !hasCheckPointer(outcome.Result.Result.Checks, "mapping.output_contract", "fail", "/contract/outputs/projects") {
				t.Fatalf("output conflict did not identify its contract field: %#v", outcome.Result.Result.Checks)
			}
		})
	}
}

func TestCheckReportsMissingOutputMappingAtContractPath(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root, source, intent := prepareNestedRenamePackage(t)
	request := readRunnableRequest(t, fixtureRoot)
	setNestedRenameCheckRequest(t, &request, source, intent)
	request.OutputMappings = map[string]string{}
	outcome := Check(nil, root, request)
	if outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("missing output mapping result = %#v", outcome.Result.Result)
	}
	if !hasCheckPointer(outcome.Result.Result.Checks, "mapping.outputs", "fail", "/contract/outputs/projects") {
		t.Fatalf("missing output mapping did not identify the contract field: %#v", outcome.Result.Result.Checks)
	}
}

func TestCheckRejectsInvalidOutputMappingWireSyntax(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readRunnableRequest(t, fixtureRoot)
	request.OutputMappings = map[string]string{"projects": "response.body.projects"}
	outcome := Check(nil, filepath.Join(fixtureRoot, "example"), request)
	if outcome.ExitCode != 2 || outcome.Result.Status != "failed" || outcome.Result.Diagnostics[0].Code != "request.invalid" {
		t.Fatalf("invalid output mapping result = %#v", outcome.Result)
	}
}

func hasCheckPointer(items []CheckItem, code, status, pointer string) bool {
	for _, item := range items {
		if item.Code == code && item.Status == status && item.Pointer == pointer {
			return true
		}
	}
	return false
}

func TestCheckKeepsSelectedNullableOutputIndeterminate(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root, source, intent := prepareNestedRenamePackage(t)
	updatedSource := []byte(strings.Replace(string(source), "items:\n                        type: array", "items:\n                        type: array\n                        nullable: true", 1))
	if string(updatedSource) == string(source) {
		t.Fatal("nullable response fixture was not changed")
	}
	if err := os.WriteFile(filepath.Join(root, "openapi", "project-api.yaml"), updatedSource, 0o600); err != nil {
		t.Fatal(err)
	}
	request := readRunnableRequest(t, fixtureRoot)
	setNestedRenameCheckRequest(t, &request, updatedSource, intent)
	outcome := Check(nil, root, request)
	if outcome.Result.Result == nil || outcome.Result.Result.Assessment != "indeterminate" || !hasCheckItem(outcome.Result.Result.Checks, "mapping.output_contract", "indeterminate") {
		t.Fatalf("nullable selected output result = %#v", outcome.Result.Result)
	}
}

func TestBindAcceptsExplicitNestedAndRenamedMappings(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root := copyBindExample(t, fixtureRoot, false)
	sourcePath := filepath.Join(root, "openapi", "project-api.yaml")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	oldResponse := `                required: [projects]
                properties:
                  projects:
                    type: array
                    items:
                      type: object
                      properties:
                        id:
                          type: string`
	newResponse := `                required: [data]
                properties:
                  data:
                    type: object
                    required: [items]
                    properties:
                      items:
                        type: array
                        items:
                          type: object
                          properties:
                            id:
                              type: string`
	updatedSource := []byte(strings.Replace(string(source), oldResponse, newResponse, 1))
	if string(updatedSource) == string(source) {
		t.Fatal("nested response fixture was not changed")
	}
	if err := os.WriteFile(sourcePath, updatedSource, 0o600); err != nil {
		t.Fatal(err)
	}
	request := readBindRequest(t, fixtureRoot)
	request.Contract = nestedRenameContract()
	request.OperationRef.SourceSHA256 = "sha256:" + evidencefile.SHA256(updatedSource)
	request.RequestMappings = map[string]string{"query.page_size": "inputs.pagination.limit"}
	request.OutputMappings = map[string]string{"projects": "received_body.data.items"}
	request.Scaffold.Inputs = []BindField{{Name: "pagination", Type: "object", Required: true}}
	request.Scaffold.Outputs[0].From = "list_projects.received_body.data.items"
	outcome := Bind(nil, root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil || outcome.Result.Result.WriteOutcome != "written" {
		t.Fatalf("nested/renamed bind = %#v", outcome.Result)
	}
}

func TestBindRejectsNestedMappingMismatchWithoutWriting(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root := copyBindExample(t, fixtureRoot, false)
	request := readBindRequest(t, fixtureRoot)
	request.Contract = nestedRenameContract()
	request.Contract.Inputs.Properties["pagination"].Properties["limit"].Type = "string"
	request.RequestMappings = map[string]string{"query.page_size": "inputs.pagination.limit"}
	request.Scaffold.Inputs = []BindField{{Name: "pagination", Type: "object", Required: true}}
	outcome := Bind(nil, root, request)
	if outcome.ExitCode != 4 || outcome.Result.Status != "needs_input" || outcome.Result.Diagnostics[0].Code != "mapping.contract_mismatch" {
		t.Fatalf("nested mismatch bind = %#v", outcome.Result)
	}
	if _, err := os.Lstat(filepath.Join(root, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("rejected nested mapping created intent: %v", err)
	}
}
