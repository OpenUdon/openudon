package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

func TestStepCheckCommandUsesCleanJSONStdout(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request, err := os.ReadFile(filepath.Join(root, "requests", "step-check-runnable.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStepCheckCommand([]string{"--example", filepath.Join(root, "example"), "--request", "-"}, bytes.NewReader(request), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code %d; stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr was not empty: %q", stderr.String())
	}
	if strings.Count(stdout.String(), "\n") != 1 {
		t.Fatalf("stdout must contain one JSON line: %q", stdout.String())
	}
	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if result["command"] != "step.check" || result["status"] != "completed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestStepCandidatesCommandMatchesPublishedFixtureAsOneJSONLine(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request, err := os.ReadFile(filepath.Join(root, "requests", "step-candidates.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStepCandidatesCommand([]string{"--example", filepath.Join(root, "example"), "--request", "-"}, bytes.NewReader(request), &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || strings.Count(stdout.String(), "\n") != 1 {
		t.Fatalf("unexpected command output: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var got, want any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &got); err != nil {
		t.Fatal(err)
	}
	wantBytes, err := os.ReadFile(filepath.Join(root, "results", "step-candidates.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("step-candidates result differs from fixture\ngot: %#v\nwant: %#v", got, want)
	}
}

func TestStepCandidatesCommandReportsMalformedJSONAsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStepCandidatesCommand([]string{"--example", "example", "--request", "-"}, strings.NewReader("{broken"), &stdout, &stderr)
	if code != 2 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatal(err)
	}
	if result["command"] != "step.candidates" || result["status"] != "failed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestStepAuthoringCLISequenceBuildsAndAssessesLocally(t *testing.T) {
	root := copyCandidateExampleForBuild(t)
	sourcePath := "openapi/project-api.yaml"
	sourceBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	pathDigest := sha256.Sum256([]byte(sourcePath))
	sourceID := "src-" + hex.EncodeToString(pathDigest[:12])
	inputs := &uws1.ParamSchema{
		Type: "object",
		Properties: map[string]*uws1.ParamSchema{
			"page_size": {Type: "integer"},
		},
		Required: []string{"page_size"},
	}
	outputs := &uws1.ParamSchema{
		Type: "object", Properties: map[string]*uws1.ParamSchema{"projects": {Type: "array"}},
		Required: []string{"projects"},
	}
	contract := stepauthoring.StepContract{
		ID: "project_read_contract", Purpose: "List projects visible to the configured account",
		Inputs: inputs, Outputs: outputs, Effect: "read",
	}
	candidateRequest := stepauthoring.CandidatesRequest{
		Version: stepauthoring.WireVersion, Kind: "request", Command: stepauthoring.CandidatesCommand,
		Contract: contract,
		SourceFilters: []stepauthoring.CandidateSourceFilter{{
			SourceKind: "openapi", SourceID: sourceID,
			SourceSHA256: "sha256:" + evidencefile.SHA256(sourceBytes),
		}},
	}
	candidateBytes, err := json.Marshal(candidateRequest)
	if err != nil {
		t.Fatal(err)
	}
	var candidateOutput bytes.Buffer
	if code := runStepCandidatesCommand([]string{"--example", root, "--request", "-"}, bytes.NewReader(candidateBytes), &candidateOutput, &bytes.Buffer{}); code != 0 {
		t.Fatalf("step candidates returned %d: %s", code, candidateOutput.String())
	}
	var candidates stepauthoring.CandidatesWireResult
	if err := json.Unmarshal(candidateOutput.Bytes(), &candidates); err != nil {
		t.Fatal(err)
	}
	var selected *stepauthoring.StepCandidate
	for i := range candidates.Result.Candidates {
		if candidates.Result.Candidates[i].OperationRef.OperationID == "listProjects" {
			selected = &candidates.Result.Candidates[i]
			break
		}
	}
	if selected == nil {
		t.Fatalf("list-projects operation missing from candidates: %s", candidateOutput.String())
	}
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intentBytes, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	authAlternative := 0
	if selected.Authentication.Status != "known" || len(selected.Authentication.Alternatives) == 0 ||
		len(selected.Authentication.Alternatives[0].Requirements) != 1 {
		t.Fatalf("candidate authentication evidence = %#v", selected.Authentication)
	}
	credentialSlot := selected.Authentication.Alternatives[0].Requirements[0].CredentialSlot
	if credentialSlot == "" {
		t.Fatalf("candidate omitted the bindable symbolic credential slot: %#v", selected.Authentication)
	}
	bindRequest := stepauthoring.BindRequest{
		Version: stepauthoring.WireVersion, Kind: "request", Command: stepauthoring.BindCommand,
		StepID: "list_projects", Contract: contract, OperationRef: selected.OperationRef,
		RequestMappings:           map[string]string{"page_size": "inputs.page_size"},
		OutputMappings:            map[string]string{"projects": "received_body.projects"},
		AuthenticationAlternative: &authAlternative,
		CredentialBindings:        map[string]string{credentialSlot: "project_api_key"},
		IntentRevision:            stepauthoring.IntentRevision{State: "present", SHA256: "sha256:" + evidencefile.SHA256(intentBytes)},
	}
	bindBytes, err := json.Marshal(bindRequest)
	if err != nil {
		t.Fatal(err)
	}
	var bindOutput bytes.Buffer
	if code := runStepBindCommand([]string{"--example", root, "--request", "-"}, bytes.NewReader(bindBytes), &bindOutput, &bytes.Buffer{}); code != 0 {
		t.Fatalf("step bind returned %d: %s; selected candidate: %#v", code, bindOutput.String(), selected)
	}
	var bindResult stepauthoring.BindWireResult
	if err := json.Unmarshal(bindOutput.Bytes(), &bindResult); err != nil {
		t.Fatal(err)
	}
	if bindResult.Result == nil || bindResult.Result.WriteOutcome != "written" {
		t.Fatalf("bind result = %#v", bindResult)
	}
	intentBytes, err = os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	checkRequest := stepauthoring.CheckRequest{
		Version: stepauthoring.WireVersion, Kind: "request", Command: stepauthoring.CheckCommand,
		StepID: "list_projects", Contract: contract, OperationRef: selected.OperationRef,
		IntentSHA256: "sha256:" + evidencefile.SHA256(intentBytes),
	}
	checkBytes, err := json.Marshal(checkRequest)
	if err != nil {
		t.Fatal(err)
	}
	var checkOutput bytes.Buffer
	if code := runStepCheckCommand([]string{"--example", root, "--request", "-"}, bytes.NewReader(checkBytes), &checkOutput, &bytes.Buffer{}); code != 0 {
		t.Fatalf("step check returned %d: %s", code, checkOutput.String())
	}
	var checked stepauthoring.Result
	if err := json.Unmarshal(checkOutput.Bytes(), &checked); err != nil {
		t.Fatal(err)
	}
	if checked.Result == nil || checked.Result.Assessment == "incompatible" {
		t.Fatalf("step check result = %#v", checked)
	}
	reviewRequest := stepauthoring.FlowReviewRequest{
		Version: stepauthoring.WireVersion, Kind: "request", Command: stepauthoring.FlowReviewCommand,
		IntentSHA256: checkRequest.IntentSHA256, ModelReview: &stepauthoring.FlowReviewModelRequest{Enabled: false},
	}
	reviewBytes, err := json.Marshal(reviewRequest)
	if err != nil {
		t.Fatal(err)
	}
	var reviewOutput bytes.Buffer
	if code := runFlowReviewCommand([]string{"--example", root, "--request", "-"}, bytes.NewReader(reviewBytes), &reviewOutput, &bytes.Buffer{}); code != 0 {
		t.Fatalf("flow review returned %d: %s", code, reviewOutput.String())
	}
	var reviewed stepauthoring.FlowReviewWireResult
	if err := json.Unmarshal(reviewOutput.Bytes(), &reviewed); err != nil {
		t.Fatal(err)
	}
	if reviewed.Result == nil || reviewed.Result.ModelReview.Status != "skipped" {
		t.Fatalf("flow review result = %#v", reviewed)
	}
	if _, err := synthesize.Build(t.Context(), synthesize.Options{ExampleDir: root}); err != nil {
		t.Fatalf("build failed after candidate/bind/check/review: %v", err)
	}
	quality, err := synthesize.Assess(synthesize.Options{ExampleDir: root})
	if err != nil || !quality.Passed() {
		t.Fatalf("assessment failed after local step authoring: report=%#v err=%v", quality, err)
	}
}

func TestStepCheckRejectsLocationThatBuildWouldPutInBody(t *testing.T) {
	root := copyCandidateExampleForBuild(t)
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	data, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "page_size = \"inputs.page_size\"", "\"body.page_size\" = \"inputs.page_size\"", 1)
	if updated == string(data) {
		t.Fatal("wrong-location intent was not changed")
	}
	if err := os.WriteFile(intentPath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	build, err := synthesize.Build(t.Context(), synthesize.Options{ExampleDir: root})
	if err != nil {
		t.Fatal(err)
	}
	uwsBytes, err := os.ReadFile(build.UWSPath)
	if err != nil {
		t.Fatal(err)
	}
	var document uws1.Document
	if err := convert.UnmarshalYAML(uwsBytes, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Operations) != 1 || document.Operations[0].Request["body"] == nil || document.Operations[0].Request["query"] != nil {
		t.Fatalf("the misplaced value was not reproduced in generated UWS: %#v", document.Operations)
	}
	fixture := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "requests", "step-check-runnable.json")
	requestBytes, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var request stepauthoring.CheckRequest
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	currentIntent, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(currentIntent), "body.page_size") {
		t.Fatalf("build removed the wrong-location mapping: %s", currentIntent)
	}
	request.IntentSHA256 = "sha256:" + evidencefile.SHA256(currentIntent)
	checked := stepauthoring.Check(t.Context(), root, request)
	if checked.Result.Result == nil || checked.Result.Result.Assessment != "incompatible" {
		t.Fatalf("generated wrong-location UWS passed step.check: %#v", checked)
	}
}

func copyCandidateExampleForBuild(t *testing.T) string {
	t.Helper()
	sourceRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "example")
	root := t.TempDir()
	for _, relative := range []string{
		"workflows/intent.hcl",
		"openapi/project-api.yaml",
	} {
		data, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(relative)))
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
	project := `# Project Inventory

## Goal

List projects visible to the configured account.

## Inputs

- page_size: required integer; maximum number of projects to return.

## Outputs

- projects from list_projects.received_body.projects.

## External Systems and OpenAPI

- Project API: openapi/project-api.yaml.

## Runtime Policy

- Allowed runtimes: openapi and http.
- cmd and ssh are not allowed.

## Data Flow

- list_projects.page_size comes from inputs.page_size.

## Credentials and Secrets

- Use only the symbolic credential binding project_api_key.
- Do not include credential values.

## Safety and Approval Boundary

Generate and validate artifacts only. Do not call the API or execute the workflow.

## Fallback Behavior

Stop if the local Project API description or the required credential binding is unavailable.
`
	if err := os.WriteFile(filepath.Join(root, "project.md"), []byte(project), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestStepCheckCommandReportsMalformedJSONOnStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runStepCheckCommand([]string{"--example", "example", "--request", "-"}, strings.NewReader("{broken"), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr was not empty: %q", stderr.String())
	}
	var result struct {
		Status      string `json:"status"`
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if result.Status != "failed" || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "request.invalid_json" {
		t.Fatalf("result = %#v", result)
	}
}

func TestStepBindCommandEmitsOneJSONResult(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	source, err := os.ReadFile(filepath.Join(root, "bind-example", "openapi", "project-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	example := t.TempDir()
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "project-api.yaml"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	request, err := os.ReadFile(filepath.Join(root, "requests", "step-bind.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStepBindCommand([]string{"--example", example, "--request", "-"}, bytes.NewReader(request), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code %d; stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	if stderr.Len() != 0 || strings.Count(stdout.String(), "\n") != 1 {
		t.Fatalf("unexpected command output: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if result["command"] != "step.bind" || result["status"] != "completed" {
		t.Fatalf("result = %#v", result)
	}
}

func TestStepBindNeedsInputUsesPublishedExitCode(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	requestBytes, err := os.ReadFile(filepath.Join(fixtureRoot, "requests", "step-bind.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request stepauthoring.BindRequest
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	request.RequestMappings = map[string]string{}
	requestBytes, err = json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(fixtureRoot, "bind-example", "openapi", "project-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	example := t.TempDir()
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(example, "openapi", "project-api.yaml"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStepBindCommand([]string{"--example", example, "--request", "-"}, bytes.NewReader(requestBytes), &stdout, &stderr)
	if code != 4 || stderr.Len() != 0 {
		t.Fatalf("needs-input exit code=%d stderr=%q stdout=%q", code, stderr.String(), stdout.String())
	}
	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if result["command"] != "step.bind" || result["status"] != "needs_input" {
		t.Fatalf("result = %#v", result)
	}
	if _, err := os.Lstat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("needs-input command created intent: %v", err)
	}
}

func TestFlowReviewCommandEmitsPublishedFixtureAsOneJSONLine(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request, err := os.ReadFile(filepath.Join(root, "requests", "flow-review.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runFlowReviewCommand([]string{"--example", filepath.Join(root, "example"), "--request", "-"}, bytes.NewReader(request), &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || strings.Count(stdout.String(), "\n") != 1 {
		t.Fatalf("unexpected command output: code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var got, want any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	wantBytes, err := os.ReadFile(filepath.Join(root, "results", "flow-review.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wantBytes, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flow-review result differs from fixture\ngot: %#v\nwant: %#v", got, want)
	}
}

func TestFlowReviewCommandReportsMalformedRequestAsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runFlowReviewCommand([]string{"--example", "example", "--request", "-"}, strings.NewReader("{broken"), &stdout, &stderr)
	if code != 2 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var result map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if result["command"] != "flow-review" || result["status"] != "failed" {
		t.Fatalf("result = %#v", result)
	}
}
