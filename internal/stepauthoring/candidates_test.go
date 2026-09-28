package stepauthoring

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/uws/uws1"
)

func TestCandidatesMatchesPublishedFixtureAndKeepsResultPathFree(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readCandidatesRequest(t, filepath.Join(root, "requests", "step-candidates.json"))
	outcome := Candidates(context.Background(), filepath.Join(root, "example"), request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil {
		t.Fatalf("candidate outcome = %#v", outcome)
	}
	got, err := json.Marshal(outcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "results", "step-candidates.json"))
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
		t.Fatalf("candidate result differs from published fixture\ngot:  %s\nwant: %s", got, want)
	}
	for _, forbidden := range []string{"project-api.yaml", "api.example.test", "project_api_key"} {
		if strings.Contains(string(got), forbidden) {
			t.Errorf("result disclosed %q", forbidden)
		}
	}
}

func TestCandidatesDistinguishesMissingAndUnsupportedVersions(t *testing.T) {
	request := CandidatesRequest{Kind: "request", Command: CandidatesCommand}
	missing := Candidates(nil, t.TempDir(), request)
	if missing.Result.Diagnostics[0].Code != "request.invalid" {
		t.Fatalf("missing version diagnostic = %#v", missing.Result.Diagnostics)
	}
	request.Version = "openudon.step-authoring.v99"
	unsupported := Candidates(nil, t.TempDir(), request)
	if unsupported.Result.Diagnostics[0].Code != "request.unsupported_version" {
		t.Fatalf("unsupported version diagnostic = %#v", unsupported.Result.Diagnostics)
	}
}

func TestCandidatesRejectsMissingAndChangedFilteredSources(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	root := copyRunnableExample(t)

	request.SourceFilters[0].SourceID = "src-not-present"
	missing := Candidates(context.Background(), root, request)
	if missing.Result.Status != "blocked" || firstCandidateDiagnostic(missing).Code != "source.unavailable" {
		t.Fatalf("missing source outcome = %#v", missing)
	}

	request = readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	if err := os.WriteFile(filepath.Join(root, "openapi", "project-api.yaml"), []byte("openapi: 3.0.0\ninfo: {title: changed, version: '1'}\npaths: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed := Candidates(context.Background(), root, request)
	if changed.Result.Status != "conflict" || firstCandidateDiagnostic(changed).Code != "source.digest_mismatch" {
		t.Fatalf("changed source outcome = %#v", changed)
	}
}

func TestCandidatesPreservesKnownConflictWhenSchemaHasUnsupportedConstructs(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	request.Contract.Inputs.Ref = "#/components/schemas/Inputs"
	request.Contract.Inputs.Properties["page_size"].Type = "string"
	outcome := Candidates(context.Background(), filepath.Join(fixtureRoot, "example"), request)
	if outcome.Result.Status != "completed" || outcome.Result.Result == nil || len(outcome.Result.Result.Candidates) == 0 {
		t.Fatalf("mixed supported/unsupported contract result = %#v", outcome)
	}
	for _, candidate := range outcome.Result.Result.Candidates {
		if candidate.Match.Inputs.Status == "incompatible" {
			if len(candidate.Match.Inputs.Gaps) == 0 {
				t.Fatalf("unsupported schema gap was omitted: %#v", candidate.Match.Inputs)
			}
			return
		}
	}
	t.Fatalf("known input mismatch was hidden by the unsupported schema construct: %#v", outcome.Result.Result.Candidates)
}

func TestCandidatesKeepNullableOutputIndeterminate(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root := copyRunnableExample(t)
	sourcePath := filepath.Join(root, "openapi", "project-api.yaml")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "                  projects:\n                    type: array", "                  projects:\n                    type: array\n                    nullable: true", 1)
	if updated == string(data) {
		t.Fatal("nullable fixture was not changed")
	}
	if err := os.WriteFile(sourcePath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	request := readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	request.SourceFilters[0].SourceSHA256 = "sha256:" + evidencefile.SHA256([]byte(updated))
	outcome := Candidates(context.Background(), root, request)
	if outcome.ExitCode != 0 || outcome.Result.Result == nil || len(outcome.Result.Result.Candidates) != 1 {
		t.Fatalf("nullable candidate outcome = %#v", outcome)
	}
	match := outcome.Result.Result.Candidates[0].Match.Outputs
	if match.Status != "indeterminate" || match.Score != 0 || !strings.Contains(strings.Join(match.Gaps, " "), "null") {
		t.Fatalf("nullable output was scored compatible: %#v", match)
	}
}

func TestCandidatesIdentifyUnsupportedRootExtensionField(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	request.Contract.Inputs.Extensions = map[string]any{"x-openudon-contract": "sensitive extension value"}
	outcome := Candidates(context.Background(), filepath.Join(fixtureRoot, "example"), request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" {
		t.Fatalf("root extension candidate result = %#v", outcome.Result)
	}
	found := false
	for _, diagnostic := range outcome.Result.Diagnostics {
		if diagnostic.Code == "contract.root_extension_unsupported" && strings.Contains(diagnostic.Message, "x-openudon-contract") {
			found = true
		}
		if strings.Contains(diagnostic.Message, "sensitive extension value") {
			t.Fatal("candidate diagnostics disclosed the root extension value")
		}
	}
	if !found {
		t.Fatalf("root extension diagnostic did not name the field: %#v", outcome.Result.Diagnostics)
	}
}

func TestCandidatesIgnoreUnselectedNullableOutputSibling(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root := copyRunnableExample(t)
	sourcePath := filepath.Join(root, "openapi", "project-api.yaml")
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), "                  projects:\n                    type: array", "                  projects:\n                    type: array\n                  debug_label:\n                    type: string\n                    nullable: true", 1)
	if updated == string(data) {
		t.Fatal("nullable sibling fixture was not changed")
	}
	if err := os.WriteFile(sourcePath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	request := readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	request.SourceFilters[0].SourceSHA256 = "sha256:" + evidencefile.SHA256([]byte(updated))
	outcome := Candidates(context.Background(), root, request)
	if outcome.ExitCode != 0 || outcome.Result.Result == nil || len(outcome.Result.Result.Candidates) != 1 {
		t.Fatalf("nullable-sibling candidate outcome = %#v", outcome)
	}
	match := outcome.Result.Result.Candidates[0].Match.Outputs
	if match.Status != "compatible" || match.Score != 30 || strings.Contains(strings.Join(match.Gaps, " "), "null") {
		t.Fatalf("unselected nullable sibling changed selected-output compatibility: %#v", match)
	}
}

func TestCandidatesSkipsSecuritySidecarsAndDoesNotExposePaths(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(fixtureRoot, "example", "openapi", "project-api.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "openapi", "project-api.yaml"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, sidecar := range []string{"project-api.yaml.security.json", "project-api.security.yaml", "project-api.security-overlay.json"} {
		if err := os.WriteFile(filepath.Join(root, "openapi", sidecar), []byte(`{"security": {"type":"apiKey"}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	request := readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	request.SourceFilters = nil
	outcome := Candidates(context.Background(), root, request)
	if outcome.ExitCode != 0 || outcome.Result.Result == nil || len(outcome.Result.Result.Sources) != 1 {
		t.Fatalf("sidecar discovery outcome = %#v", outcome)
	}
	if outcome.Result.Result.Sources[0].OperationCount != 1 {
		t.Fatalf("sidecars changed operation count: %#v", outcome.Result.Result.Sources)
	}
}

func TestCandidatesBlocksUnsafeAndCancelledDiscovery(t *testing.T) {
	fixtureRoot := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	request := readCandidatesRequest(t, filepath.Join(fixtureRoot, "requests", "step-candidates.json"))
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(fixtureRoot, "example", "openapi", "project-api.yaml"), filepath.Join(root, "openapi", "project-api.yaml")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	unsafe := Candidates(context.Background(), root, request)
	if unsafe.Result.Status != "blocked" || firstCandidateDiagnostic(unsafe).Code != "source.unavailable" {
		t.Fatalf("symlink outcome = %#v", unsafe)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled := Candidates(ctx, root, request)
	if cancelled.Result.Status != "blocked" || firstCandidateDiagnostic(cancelled).Code != "request.cancelled" {
		t.Fatalf("cancelled outcome = %#v", cancelled)
	}
}

func TestCandidatesEnforcesSourceBudgets(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "openapi"), 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxCandidateSources; i++ {
		path := filepath.Join(root, "openapi", fmt.Sprintf("source-%02d.json", i))
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Truncate(path, MaxSourceBytes); err != nil {
			t.Fatal(err)
		}
	}
	files, failure := loadCandidateSources(context.Background(), root, nil)
	if failure == nil || failure.code != "source.byte_limit" || files != nil {
		t.Fatalf("combined byte-limit result files=%d failure=%#v", len(files), failure)
	}
}

func TestCandidatesMapsEverySupportedSourceFamily(t *testing.T) {
	sources := map[string]struct {
		name    string
		content string
	}{
		"openapi": {"service.yaml", `openapi: 3.0.3
info: {title: Pets API, version: '1'}
paths:
  /pets/{petId}:
    get:
      operationId: getPet
      summary: Get a pet
      responses: {"200": {description: Pet}}
`},
		"google-discovery": {"service.json", `{"discoveryVersion":"v1","name":"pets","version":"v1","title":"Pets API","rootUrl":"https://example.invalid/","servicePath":"pets/v1/","resources":{"pets":{"methods":{"get":{"id":"pets.pets.get","path":"pets/{petId}","httpMethod":"GET","parameters":{"petId":{"type":"string","required":true,"location":"path"}},"response":{"type":"object","properties":{"name":{"type":"string"}}}}}}}}`},
		"aws-smithy":       {"service.json", `{"smithy":"2.0","shapes":{"example.pets#Pets":{"type":"service","version":"2026-01-01","operations":[{"target":"example.pets#GetPet"}],"traits":{"aws.api#service":{"sdkId":"Pets","endpointPrefix":"pets"},"aws.protocols#restJson1":{}}},"example.pets#GetPet":{"type":"operation","input":{"target":"example.pets#GetPetInput"},"output":{"target":"example.pets#GetPetOutput"},"traits":{"smithy.api#readonly":{},"smithy.api#http":{"method":"GET","uri":"/pets/{petId}","code":200}}},"example.pets#GetPetInput":{"type":"structure","members":{"petId":{"target":"smithy.api#String","traits":{"smithy.api#required":{}}}}},"example.pets#GetPetOutput":{"type":"structure","members":{"name":{"target":"smithy.api#String"}}}}}`},
		"asyncapi": {"service.yaml", `asyncapi: 3.0.0
info: {title: Pet Events, version: '1'}
operations:
  publishPet:
    action: send
    summary: Publish a pet event
    channel: {$ref: '#/channels/pets'}
    messages: [{$ref: '#/channels/pets/messages/petChanged'}]
channels:
  pets:
    address: pets.events
    messages:
      petChanged:
        payload:
          type: object
          properties: {petId: {type: string}}
`},
		"graphql": {"service.graphql", `type Query { pet(id: ID!): Pet }
type Pet { id: ID!, name: String }
`},
		"openrpc": {"service.json", `{"openrpc":"1.3.2","info":{"title":"Pets RPC","version":"1"},"methods":[{"name":"pets.get","summary":"Get a pet","params":[{"name":"petId","required":true,"schema":{"type":"string"}}],"result":{"name":"pet","schema":{"type":"object","properties":{"id":{"type":"string"}}}}}]}`},
		"grpc-protobuf": {"service.proto", `syntax = "proto3";
package pets.v1;
message GetPetRequest { string pet_id = 1; }
message Pet { string id = 1; }
service Pets { rpc GetPet(GetPetRequest) returns (Pet); }
`},
		"odata": {"service.xml", `<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices><Schema Namespace="Demo" xmlns="http://docs.oasis-open.org/odata/ns/edm">
    <EntityType Name="Pet"><Property Name="ID" Type="Edm.String" Nullable="false"/></EntityType>
    <Function Name="GetPet"><Parameter Name="ID" Type="Edm.String" Nullable="false"/><ReturnType Type="Demo.Pet" Nullable="false"/></Function>
    <EntityContainer Name="Container"><EntitySet Name="Pets" EntityType="Demo.Pet"/></EntityContainer>
  </Schema></edmx:DataServices>
</edmx:Edmx>
`},
	}
	root := t.TempDir()
	for kind, source := range sources {
		directory := filepath.Join(root, kind)
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, source.name), []byte(source.content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	request := CandidatesRequest{
		Version: WireVersion, Kind: "request", Command: CandidatesCommand,
		Contract: StepContract{
			ID: "get_pet", Purpose: "Get a pet",
			Inputs:  &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{}},
			Outputs: &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{}},
			Effect:  "read",
		},
	}
	outcome := Candidates(context.Background(), root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil {
		t.Fatalf("mixed source-family outcome = %#v", outcome)
	}
	if len(outcome.Result.Result.Sources) != len(sources) {
		t.Fatalf("source report count = %d, want %d: %#v", len(outcome.Result.Result.Sources), len(sources), outcome.Result.Result.Sources)
	}
	seen := make(map[string]bool, len(sources))
	for _, candidate := range outcome.Result.Result.Candidates {
		if candidate.OperationRef.NativeSelector == "" || candidate.OperationRef.SourceSHA256 == "" || candidate.Summary.Description == "" {
			t.Errorf("candidate lost identity or summary while adapting: %#v", candidate)
		}
		seen[candidate.OperationRef.SourceKind] = true
		selectedSource := sources[candidate.OperationRef.SourceKind]
		sourcePath := filepath.Join(candidate.OperationRef.SourceKind, selectedSource.name)
		operation := candidate.OperationRef.OperationID
		if operation == "" {
			operation = candidate.OperationRef.OperationKey
		}
		quote := func(value string) string {
			encoded, _ := json.Marshal(value)
			return string(encoded)
		}

		checkRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(checkRoot, candidate.OperationRef.SourceKind), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(checkRoot, filepath.FromSlash(sourcePath)), []byte(selectedSource.content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(checkRoot, "workflows"), 0o700); err != nil {
			t.Fatal(err)
		}
		intent := "workflow {\n  name = \"Pets\"\n}\nstep \"get_pet\" {\n  type = \"http\"\n  do = \"Get a pet.\"\n  source = " + quote(sourcePath) + "\n  operation = " + quote(operation) + "\n}\n"
		intentBytes := []byte(intent)
		if err := os.WriteFile(filepath.Join(checkRoot, "workflows", "intent.hcl"), intentBytes, 0o600); err != nil {
			t.Fatal(err)
		}
		check := CheckRequest{
			Version: WireVersion, Kind: "request", Command: CheckCommand,
			StepID: "get_pet", Contract: request.Contract, OperationRef: candidate.OperationRef,
			IntentSHA256: sourceDigest(intentBytes),
		}
		checked := Check(context.Background(), checkRoot, check)
		if checked.Result.Result == nil || !hasCheckItem(checked.Result.Result.Checks, "operation.exact_match", "pass") {
			t.Errorf("%s candidate did not resolve through step.check: %#v", candidate.OperationRef.SourceKind, checked)
		}

		bindRoot := t.TempDir()
		if err := os.MkdirAll(filepath.Join(bindRoot, candidate.OperationRef.SourceKind), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(bindRoot, filepath.FromSlash(sourcePath)), []byte(selectedSource.content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(bindRoot, "workflows"), 0o700); err != nil {
			t.Fatal(err)
		}
		bind := Bind(context.Background(), bindRoot, BindRequest{
			Version: WireVersion, Kind: "request", Command: BindCommand, StepID: "get_pet",
			Contract: request.Contract, OperationRef: candidate.OperationRef,
			RequestMappings: map[string]string{}, OutputMappings: map[string]string{}, CredentialBindings: map[string]string{},
			IntentRevision: IntentRevision{State: "missing"},
			Scaffold:       &BindScaffold{Workflow: BindScaffoldWorkflow{Name: "pets", Description: "Test workflow."}},
		})
		if bind.Result.Status != "completed" && bind.Result.Status != "needs_input" {
			t.Errorf("%s candidate did not pass step.bind operation resolution: %#v", candidate.OperationRef.SourceKind, bind)
		}
	}
	for kind := range sources {
		if !seen[kind] {
			t.Errorf("supported source family %q produced no candidate", kind)
		}
	}
}

func hasCheckItem(items []CheckItem, code, status string) bool {
	for _, item := range items {
		if item.Code == code && item.Status == status {
			return true
		}
	}
	return false
}

func TestCandidatesAndBindSupportLegacyDiscoveryDirectory(t *testing.T) {
	content := `{"discoveryVersion":"v1","name":"pets","version":"v1","title":"Pets API","rootUrl":"https://example.invalid/","servicePath":"pets/v1/","resources":{"pets":{"methods":{"get":{"id":"pets.pets.get","path":"pets/{petId}","httpMethod":"GET","parameters":{"petId":{"type":"string","required":true,"location":"path"}},"response":{"type":"object","properties":{"name":{"type":"string"}}}}}}}}`
	root := t.TempDir()
	legacyPath := filepath.Join(root, "discovery", "legacy.json")
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	contract := StepContract{
		ID: "get_pet", Purpose: "Get a pet",
		Inputs:  &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{}},
		Outputs: &uws1.ParamSchema{Type: "object", Properties: map[string]*uws1.ParamSchema{}},
		Effect:  "read",
	}
	request := CandidatesRequest{Version: WireVersion, Kind: "request", Command: CandidatesCommand, Contract: contract}
	candidates := Candidates(context.Background(), root, request)
	if candidates.ExitCode != 0 || candidates.Result.Result == nil || len(candidates.Result.Result.Candidates) == 0 {
		t.Fatalf("legacy discovery candidate result = %#v", candidates)
	}
	var selected *StepCandidate
	for i := range candidates.Result.Result.Candidates {
		candidate := &candidates.Result.Result.Candidates[i]
		if candidate.OperationRef.SourceKind == "google-discovery" && candidate.OperationRef.SourceID == sourceIDForPath("discovery/legacy.json") {
			selected = candidate
			break
		}
	}
	if selected == nil {
		t.Fatalf("legacy discovery source was not exposed with the Google source kind: %#v", candidates.Result.Result)
	}

	workflowDir := filepath.Join(root, "workflows")
	if err := os.MkdirAll(workflowDir, 0o700); err != nil {
		t.Fatal(err)
	}
	operation := selected.OperationRef.OperationID
	if operation == "" {
		operation = selected.OperationRef.OperationKey
	}
	intent := "workflow {\n  name = \"pets\"\n}\nstep \"get_pet\" {\n  type = \"http\"\n  do = \"Get a pet.\"\n  source = \"discovery/legacy.json\"\n  operation = \"" + operation + "\"\n}\n"
	intentBytes := []byte(intent)
	if err := os.WriteFile(filepath.Join(workflowDir, "intent.hcl"), intentBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	checked := Check(context.Background(), root, CheckRequest{
		Version: WireVersion, Kind: "request", Command: CheckCommand,
		StepID: "get_pet", Contract: contract, OperationRef: selected.OperationRef,
		IntentSHA256: sourceDigest(intentBytes),
	})
	if checked.Result.Result == nil || !hasCheckItem(checked.Result.Result.Checks, "operation.exact_match", "pass") {
		t.Fatalf("legacy discovery check result = %#v", checked)
	}

	bindRoot := t.TempDir()
	bindSource := filepath.Join(bindRoot, "discovery", "legacy.json")
	if err := os.MkdirAll(filepath.Dir(bindSource), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bindSource, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(bindRoot, "workflows"), 0o700); err != nil {
		t.Fatal(err)
	}
	bound := Bind(context.Background(), bindRoot, BindRequest{
		Version: WireVersion, Kind: "request", Command: BindCommand, StepID: "get_pet",
		Contract: contract, OperationRef: selected.OperationRef,
		RequestMappings: map[string]string{}, OutputMappings: map[string]string{}, CredentialBindings: map[string]string{},
		IntentRevision: IntentRevision{State: "missing"},
		Scaffold:       &BindScaffold{Workflow: BindScaffoldWorkflow{Name: "pets", Description: "Test workflow."}},
	})
	if bound.Result.Status != "completed" && bound.Result.Status != "needs_input" {
		t.Fatalf("legacy discovery bind result = %#v", bound)
	}

	sidecar := []byte(content)
	sidecarPath := filepath.Join(root, "google-discovery", "legacy.security.json")
	if err := os.MkdirAll(filepath.Dir(sidecarPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sidecarPath, sidecar, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := locateSource(root, OperationRef{
		SourceKind: "google-discovery", SourceID: sourceIDForPath("google-discovery/legacy.security.json"),
		SourceSHA256: sourceDigest(sidecar),
	}); err == nil {
		t.Fatal("bind source lookup accepted a security sidecar that candidates exclude")
	}
}

func TestCandidateAuthenticationPreservesOROfANDAndAnonymousAlternatives(t *testing.T) {
	candidate := apitools.OperationCandidate{
		Capabilities: []apitools.OperationCapability{{Dimension: "auth", Status: apitools.OperationCapabilitySupported}},
		Operation: apitools.OperationSummary{SecurityRequirementSets: []apitools.SecurityRequirementSetSummary{
			{Requirements: []apitools.SecuritySummary{{Name: "key", Type: "apiKey", In: "header", ParameterName: "X-KEY"}, {Name: "oauth", Type: "oauth2", OAuthFlows: []apitools.OAuthFlowSummary{{Name: "clientCredentials"}}, Scopes: []string{"write", "read"}}}},
			{},
		}},
	}
	got := mapCandidateAuthentication(candidate)
	if got.Status != "known" || len(got.Alternatives) != 2 || len(got.Alternatives[0].Requirements) != 2 || len(got.Alternatives[1].Requirements) != 0 {
		t.Fatalf("authentication alternatives = %#v", got)
	}
	if got.Alternatives[0].Requirements[1].Flows[0] != "clientCredentials" || !reflect.DeepEqual(got.Alternatives[0].Requirements[1].Scopes, []string{"read", "write"}) {
		t.Fatalf("OAuth requirements were not normalized: %#v", got.Alternatives[0])
	}
	if got.Alternatives[0].Requirements[0].ParameterName != "X-KEY" || got.Alternatives[0].Requirements[0].CredentialSlot != "key" {
		t.Fatalf("bindable auth metadata was not exposed: %#v", got.Alternatives[0].Requirements[0])
	}
}

func TestMapCandidateMatchDoesNotCallReasonsMissing(t *testing.T) {
	compatible := mapCandidateMatch(apitools.ContractDimensionMatch{
		Status: apitools.ContractMatchCompatible, Reasons: []string{"evidence overlaps purpose"},
	})
	if len(compatible.Missing) != 0 || !reflect.DeepEqual(compatible.Reasons, []string{"evidence overlaps purpose"}) {
		t.Fatalf("compatible match reasons were not kept distinct from missing fields: %#v", compatible)
	}
	indeterminate := mapCandidateMatch(apitools.ContractDimensionMatch{
		Status: apitools.ContractMatchIndeterminate, Reasons: []string{"not enough evidence"},
	})
	if !reflect.DeepEqual(indeterminate.Reasons, []string{"not enough evidence"}) || len(indeterminate.Missing) != 0 {
		t.Fatalf("indeterminate reason was not retained distinctly: %#v", indeterminate)
	}
}

func TestUnsupportedContractDimensionsPreserveKnownIncompatibility(t *testing.T) {
	data := CandidatesData{Candidates: []StepCandidate{{Match: CandidateMatch{
		Inputs: CandidateCompatibility{Status: "incompatible", Missing: []string{"unmapped input"}, Conflicts: []string{"known conflict"}},
	}}}}
	applyUnsupportedDimensions(&data, true, false)
	match := data.Candidates[0].Match.Inputs
	if match.Status != "incompatible" || !reflect.DeepEqual(match.Gaps, []string{"Some declared input schema constructs are not represented by APItools."}) {
		t.Fatalf("unsupported input dimension erased the known incompatibility: %#v", match)
	}
	if !reflect.DeepEqual(match.Missing, []string{"unmapped input"}) {
		t.Fatalf("APItools missing evidence changed: %#v", match)
	}
	if !reflect.DeepEqual(match.Conflicts, []string{"known conflict"}) {
		t.Fatalf("known conflicts were discarded: %#v", match)
	}

	for _, status := range []string{"compatible", "indeterminate"} {
		data := CandidatesData{Candidates: []StepCandidate{{Match: CandidateMatch{
			Inputs: CandidateCompatibility{Status: status},
		}}}}
		applyUnsupportedDimensions(&data, true, false)
		if got := data.Candidates[0].Match.Inputs.Status; got != "indeterminate" {
			t.Errorf("unsupported input dimension changed %q to %q, want indeterminate", status, got)
		}
	}
}

func readCandidatesRequest(t *testing.T, path string) CandidatesRequest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var request CandidatesRequest
	if err := evidencefile.DecodeStrict(data, &request); err != nil {
		t.Fatal(err)
	}
	return request
}

func firstCandidateDiagnostic(outcome CandidatesOutcome) Diagnostic {
	if len(outcome.Result.Diagnostics) == 0 {
		return Diagnostic{}
	}
	return outcome.Result.Diagnostics[0]
}
