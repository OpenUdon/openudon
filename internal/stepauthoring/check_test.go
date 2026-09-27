package stepauthoring

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestCheckMatchesRunnablePublishedFixture(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	requestBytes, err := os.ReadFile(filepath.Join(root, "requests", "step-check-runnable.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request CheckRequest
	if err := evidencefile.DecodeStrict(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	outcome := Check(nil, filepath.Join(root, "example"), request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil {
		t.Fatalf("check outcome = %#v", outcome)
	}
	if outcome.Result.Result.Assessment != "compatible" {
		t.Fatalf("assessment = %q, want compatible with published APItools read-effect evidence", outcome.Result.Result.Assessment)
	}
	got, err := json.Marshal(outcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "results", "step-check-runnable.json"))
	if err != nil {
		t.Fatal(err)
	}
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("check result differs from published fixture\ngot:  %s\nwant: %s", got, want)
	}
	for _, forbidden := range []string{"project-api.yaml", "api.example.test", "List projects.", "project_api_key"} {
		if strings.Contains(string(got), forbidden) {
			t.Errorf("result disclosed %q", forbidden)
		}
	}
}

func TestCheckUsesEffectEvidenceAndRejectsConflictingContract(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readRunnableRequest(t, root)
	request.Contract.Effect = "write"
	outcome := Check(nil, filepath.Join(root, "example"), request)
	if outcome.ExitCode != 0 || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("effect mismatch outcome = %#v", outcome.Result)
	}
	found := false
	for _, item := range outcome.Result.Result.Checks {
		if item.Code == "effect.evidence" && item.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Fatalf("effect conflict was not reported: %#v", outcome.Result.Result.Checks)
	}

	request.Contract.Effect = "unknown"
	unknown := Check(nil, filepath.Join(root, "example"), request)
	if unknown.Result.Result == nil {
		t.Fatalf("unknown-effect result = %#v", unknown.Result)
	}
	for _, item := range unknown.Result.Result.Checks {
		if item.Code == "effect.evidence" && item.Status != "indeterminate" {
			t.Fatalf("unknown contract effect was treated as %q: %#v", item.Status, item)
		}
	}
}

func TestCheckRejectsInputAndOutputTypeMismatch(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readRunnableRequest(t, root)
	request.Contract.Inputs.Properties["page_size"].Type = "string"
	request.Contract.Outputs.Properties["projects"].Type = "string"
	outcome := Check(nil, filepath.Join(root, "example"), request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("type-mismatch check outcome = %#v", outcome)
	}
	failed := map[string]bool{}
	for _, item := range outcome.Result.Result.Checks {
		if item.Status == "fail" {
			failed[item.Code] = true
		}
	}
	if !failed["mapping.input_contract"] || !failed["mapping.output_contract"] {
		t.Fatalf("type mismatch was not checked in both dimensions: %#v", outcome.Result.Result.Checks)
	}
}

func TestCheckMarksUnsupportedContractSchemaIndeterminate(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readRunnableRequest(t, root)
	request.Contract.Inputs.Ref = "#/components/schemas/Inputs"
	outcome := Check(nil, filepath.Join(root, "example"), request)
	if outcome.Result.Result == nil || outcome.Result.Result.Assessment != "indeterminate" {
		t.Fatalf("unsupported contract schema result = %#v", outcome.Result)
	}
	for _, item := range outcome.Result.Result.Checks {
		if item.Code == "mapping.input_contract" && item.Status == "indeterminate" {
			return
		}
	}
	t.Fatalf("unsupported contract schema was not surfaced: %#v", outcome.Result.Result.Checks)
}

func TestCheckReturnsStaleIntentAndSourceConflicts(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readRunnableRequest(t, root)
	request.IntentSHA256 = "sha256:" + strings.Repeat("0", 64)
	outcome := Check(nil, filepath.Join(root, "example"), request)
	if outcome.ExitCode != 3 || outcome.Result.Status != "conflict" || outcome.Result.Diagnostics[0].Code != "intent.stale" {
		t.Fatalf("stale intent outcome = %#v", outcome)
	}
	request = readRunnableRequest(t, root)
	request.OperationRef.SourceSHA256 = "sha256:" + strings.Repeat("0", 64)
	outcome = Check(nil, filepath.Join(root, "example"), request)
	if outcome.ExitCode != 3 || outcome.Result.Status != "conflict" || outcome.Result.Diagnostics[0].Code != "source.digest_mismatch" {
		t.Fatalf("stale source outcome = %#v", outcome)
	}
}

func TestCheckReturnsCancellationBeforeReadingExample(t *testing.T) {
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outcome := Check(ctx, t.TempDir(), request)
	if outcome.ExitCode != 4 || outcome.Result.Status != "blocked" || len(outcome.Result.Diagnostics) != 1 || outcome.Result.Diagnostics[0].Code != "request.cancelled" {
		t.Fatalf("cancelled check outcome = %#v", outcome)
	}
}

func TestCheckMissingRequiredMappingIsIncompatible(t *testing.T) {
	root := copyRunnableExample(t)
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(intentBytes), "    page_size = \"inputs.page_size\"\n", "", 1)
	if changed == string(intentBytes) {
		t.Fatal("fixture mapping not found")
	}
	if err := os.WriteFile(intentPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	request.IntentSHA256 = "sha256:" + evidencefile.SHA256([]byte(changed))
	outcome := Check(nil, root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("missing-mapping outcome = %#v", outcome)
	}
	found := false
	for _, item := range outcome.Result.Result.Checks {
		if item.Code == "mapping.incomplete" && item.Status == "fail" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing mapping was not reported: %#v", outcome.Result.Result.Checks)
	}
}

func TestCheckRejectsMappingToUndeclaredWorkflowInput(t *testing.T) {
	root := copyRunnableExample(t)
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(intentBytes), `page_size = "inputs.page_size"`, `page_size = "inputs.missing_page_size"`, 1)
	if changed == string(intentBytes) {
		t.Fatal("fixture mapping not found")
	}
	if err := os.WriteFile(intentPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	request.IntentSHA256 = "sha256:" + evidencefile.SHA256([]byte(changed))
	outcome := Check(nil, root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("undeclared input mapping outcome = %#v", outcome)
	}
	for _, item := range outcome.Result.Result.Checks {
		if item.Code == "mapping.workflow_inputs" && item.Status == "fail" {
			return
		}
	}
	t.Fatalf("undeclared workflow input was not rejected: %#v", outcome.Result.Result.Checks)
}

func TestCheckRejectsSelfReferencesAndDependencyCycles(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	for _, test := range []struct {
		name   string
		mutate func(string) string
	}{
		{
			name: "self output reference",
			mutate: func(intent string) string {
				return strings.Replace(intent, `page_size = "inputs.page_size"`, `page_size = "steps.list_projects.received_body.projects"`, 1)
			},
		},
		{
			name: "transitive back edge",
			mutate: func(intent string) string {
				updated := strings.Replace(intent, `operation = "listProjects"`, "operation = \"listProjects\"\n  depends_on = [\"prepare\"]", 1)
				return updated + `
step "prepare" {
  type = "fnct"
  do = "Prepare the downstream step."
  depends_on = ["list_projects"]
}
`
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := copyRunnableExample(t)
			request := readRunnableRequest(t, fixtureRoot)
			intentPath := filepath.Join(root, "workflows", "intent.hcl")
			intentBytes, err := os.ReadFile(intentPath)
			if err != nil {
				t.Fatal(err)
			}
			changed := test.mutate(string(intentBytes))
			if changed == string(intentBytes) {
				t.Fatal("dependency fixture was not changed")
			}
			if err := os.WriteFile(intentPath, []byte(changed), 0o600); err != nil {
				t.Fatal(err)
			}
			request.IntentSHA256 = sourceDigest([]byte(changed))
			outcome := Check(nil, root, request)
			if outcome.ExitCode != 0 || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
				t.Fatalf("dependency-cycle check = %#v", outcome)
			}
			if !hasCheckItem(outcome.Result.Result.Checks, "dependency.cycle", "fail") {
				t.Fatalf("dependency cycle was not reported: %#v", outcome.Result.Result.Checks)
			}
		})
	}
}

func TestCheckRejectsOutputReferencesOutsideContract(t *testing.T) {
	root := copyRunnableExample(t)
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(intentBytes), "list_projects.received_body.projects", "list_projects.received_body.secret", 1)
	if changed == string(intentBytes) {
		t.Fatal("fixture output reference not found")
	}
	if err := os.WriteFile(intentPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	request.IntentSHA256 = "sha256:" + evidencefile.SHA256([]byte(changed))
	outcome := Check(nil, root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("output-reference outcome = %#v", outcome)
	}
	for _, item := range outcome.Result.Result.Checks {
		if item.Code == "mapping.output_references" && item.Status == "fail" {
			return
		}
	}
	t.Fatalf("outside-contract output reference was not rejected: %#v", outcome.Result.Result.Checks)
}

func TestCheckNotFoundResultUsesEmptyArrays(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readRunnableRequest(t, root)
	request.StepID = "missing_step"
	outcome := Check(nil, filepath.Join(root, "example"), request)
	if outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("not-found result = %#v", outcome)
	}
	encoded, err := json.Marshal(outcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"unresolved_questions":[]`) || !strings.Contains(string(encoded), `"checks":[`) {
		t.Fatalf("not-found arrays are not empty JSON arrays: %s", encoded)
	}
}

func TestCheckBlocksUnsafeSourceAndDoesNotEchoPaths(t *testing.T) {
	root := copyRunnableExample(t)
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(intentBytes), "openapi/project-api.yaml", "../private-secret.yaml", 1)
	if err := os.WriteFile(intentPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	request.IntentSHA256 = "sha256:" + evidencefile.SHA256([]byte(changed))
	outcome := Check(nil, root, request)
	encoded, err := json.Marshal(outcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "private-secret") || strings.Contains(string(encoded), root) {
		t.Fatalf("unsafe path leaked in result: %s", encoded)
	}
	if outcome.Result.Status != "completed" || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("unsafe source path outcome = %#v", outcome)
	}
}

func TestCheckRejectsSymlinkedSource(t *testing.T) {
	root := copyRunnableExample(t)
	source := filepath.Join(root, "openapi", "project-api.yaml")
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "source.yaml")
	if err := os.WriteFile(outside, []byte("not relevant"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, source); err != nil {
		t.Fatal(err)
	}
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	outcome := Check(nil, root, request)
	if outcome.ExitCode != 4 || outcome.Result.Status != "blocked" || outcome.Result.Diagnostics[0].Code != "source.unavailable" {
		t.Fatalf("symlink outcome = %#v", outcome)
	}
	if strings.Contains(outcome.Result.Diagnostics[0].Message, outside) {
		t.Fatalf("symlink target leaked: %#v", outcome.Result.Diagnostics[0])
	}
}

func TestCheckRejectsSecuritySidecarAsOperationSource(t *testing.T) {
	root := copyRunnableExample(t)
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(intentBytes), "openapi/project-api.yaml", "openapi/project-api.security.json", 1)
	if changed == string(intentBytes) {
		t.Fatal("fixture source path not found")
	}
	if err := os.WriteFile(filepath.Join(root, "openapi", "project-api.security.json"), []byte("sidecar-shaped"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(intentPath, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
	request.IntentSHA256 = "sha256:" + evidencefile.SHA256([]byte(changed))
	request.OperationRef.SourceID = sourceIDForPath("openapi/project-api.security.json")
	request.OperationRef.SourceSHA256 = sourceDigest([]byte("sidecar-shaped"))
	outcome := Check(nil, root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil || outcome.Result.Result.Assessment != "incompatible" {
		t.Fatalf("sidecar source check = %#v", outcome)
	}
	if !hasCheckItem(outcome.Result.Result.Checks, "source.selection", "fail") {
		t.Fatalf("security sidecar was not rejected as a source selection: %#v", outcome.Result.Result.Checks)
	}
}

func TestCheckUsesAnyCompleteAuthenticationAlternative(t *testing.T) {
	root := copyRunnableExample(t)
	request := readRunnableRequest(t, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1"))
	sourcePath := filepath.Join(root, "openapi", "project-api.yaml")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(source), "        - apiKeyAuth: []\n", "        - apiKeyAuth: []\n          oauthAuth: [read]\n        - apiKeyAuth: []\n", 1)
	updated = strings.Replace(updated, "    apiKeyAuth:\n      type: apiKey\n      in: header\n      name: X-API-Key\n", "    apiKeyAuth:\n      type: apiKey\n      in: header\n      name: X-API-Key\n    oauthAuth:\n      type: oauth2\n      flows:\n        clientCredentials:\n          tokenUrl: https://auth.example.test/token\n          scopes:\n            read: Read access\n", 1)
	if updated == string(source) || updated == "" {
		t.Fatal("could not prepare alternate auth fixture")
	}
	if err := os.WriteFile(sourcePath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	request.OperationRef.SourceSHA256 = "sha256:" + evidencefile.SHA256([]byte(updated))
	outcome := Check(nil, root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" {
		t.Fatalf("outcome = %#v", outcome)
	}
	for _, item := range outcome.Result.Result.Checks {
		if item.Code == "authentication.alternative" && item.Status != "pass" {
			t.Fatalf("OR-of-AND alternative not recognized: %#v", item)
		}
	}
}

func TestSelectedStepSourceOverridesWorkflowDefault(t *testing.T) {
	intent := &workflowintent.Intent{Source: "openapi/default.yaml"}
	step := &workflowintent.Step{Source: "google-discovery/alternate.json"}
	if got, state := selectedSource(intent, step); state != "ok" || got != "google-discovery/alternate.json" {
		t.Fatalf("selectedSource = %q, %q; want explicit step override", got, state)
	}
}

func readRunnableRequest(t *testing.T, fixtureRoot string) CheckRequest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureRoot, "requests", "step-check-runnable.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request CheckRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		t.Fatal(err)
	}
	return request
}

func copyRunnableExample(t *testing.T) string {
	t.Helper()
	fixture := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "example")
	root := t.TempDir()
	for _, relative := range []string{"workflows/intent.hcl", "openapi/project-api.yaml"} {
		data, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
