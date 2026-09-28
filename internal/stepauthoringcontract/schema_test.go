package stepauthoringcontract

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
	"github.com/OpenUdon/uws/uws1"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestPublishedFixturesConformToSchema(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	schemaPath := filepath.Join("..", "..", "docs", "schemas", "openudon.step-authoring.v1.schema.json")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatal(err)
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	const resource = "openudon.step-authoring.v1.schema.json"
	if err := compiler.AddResource(resource, document); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile(resource)
	if err != nil {
		t.Fatal(err)
	}

	for _, group := range []struct {
		name  string
		valid bool
	}{
		{name: "requests", valid: true},
		{name: "results", valid: true},
		{name: "invalid", valid: false},
	} {
		t.Run(group.name, func(t *testing.T) {
			entries, err := os.ReadDir(filepath.Join(root, group.name))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) == 0 {
				t.Fatal("fixture directory is empty")
			}
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				t.Run(entry.Name(), func(t *testing.T) {
					data, err := os.ReadFile(filepath.Join(root, group.name, entry.Name()))
					if err != nil {
						t.Fatal(err)
					}
					instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
					if err == nil {
						err = schema.Validate(instance)
					}
					if (err == nil) != group.valid {
						t.Fatalf("valid=%t, parse/validation error=%v", group.valid, err)
					}
				})
			}
		})
	}

	requestBytes, err := os.ReadFile(filepath.Join(root, "requests", "step-candidates.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request stepauthoring.CandidatesRequest
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	outcome := stepauthoring.Candidates(context.Background(), filepath.Join(root, "example"), request)
	if outcome.Result.Status != "completed" {
		t.Fatalf("candidate result did not complete: %#v", outcome.Result)
	}
	runtimeBytes, err := json.Marshal(outcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	runtimeInstance, err := jsonschema.UnmarshalJSON(bytes.NewReader(runtimeBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(runtimeInstance); err != nil {
		t.Fatalf("runtime step-candidates output does not conform to the published schema: %v", err)
	}

	sourceRoot := t.TempDir()
	sourceContent := []byte("openapi: 3.0.3\ninfo: {title: Weather, version: '1'}\npaths: {}\n")
	sourcePath := filepath.Join(t.TempDir(), "weather.yaml")
	if err := os.WriteFile(sourcePath, sourceContent, 0o600); err != nil {
		t.Fatal(err)
	}
	sourceOutcome := stepauthoring.AddSources(context.Background(), sourceRoot, stepauthoring.SourceAddRequest{
		Version: stepauthoring.WireVersion, Kind: "request", Command: stepauthoring.SourceAddCommand,
		ManifestRevision: stepauthoring.SourceManifestRevision{State: "missing"},
		Sources: []stepauthoring.SourceAddEntry{{
			SourceKind: "openapi", SourceID: "weather", SourcePath: sourcePath,
			SourceSHA256: "sha256:" + evidencefile.SHA256(sourceContent),
		}},
	})
	if sourceOutcome.Result.Status != "completed" {
		t.Fatalf("runtime step-source-add output did not complete: %#v", sourceOutcome.Result)
	}
	sourceResultBytes, err := json.Marshal(sourceOutcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	sourceResultInstance, err := jsonschema.UnmarshalJSON(bytes.NewReader(sourceResultBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(sourceResultInstance); err != nil {
		t.Fatalf("runtime step-source-add output does not conform to the published schema: %v", err)
	}
	indeterminateResult := stepauthoring.SourceAddWireResult{
		Version: stepauthoring.WireVersion, Kind: "result", Command: stepauthoring.SourceAddCommand,
		Status: "failed", WriteOutcome: "indeterminate", AffectedPaths: []string{"expected/api-source-manifest.json", "openapi/weather.yaml"},
		Diagnostics: []stepauthoring.Diagnostic{{Code: "write.indeterminate", Severity: "error", Message: "Inspect the listed paths before retrying."}},
	}
	indeterminateBytes, err := json.Marshal(indeterminateResult)
	if err != nil {
		t.Fatal(err)
	}
	indeterminateInstance, err := jsonschema.UnmarshalJSON(bytes.NewReader(indeterminateBytes))
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.Validate(indeterminateInstance); err != nil {
		t.Fatalf("indeterminate step-source-add output does not conform to the published schema: %v", err)
	}
}

func TestSharedFieldFixtureMatchesUWParamSchema(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	requestBytes, err := os.ReadFile(filepath.Join(root, "requests", "step-candidates.json"))
	if err != nil {
		t.Fatal(err)
	}
	pendingBytes, err := os.ReadFile(filepath.Join(root, "uws-c07-2-pending-fields-draft.json"))
	if err != nil {
		t.Fatal(err)
	}

	type sharedFields struct {
		Purpose string            `json:"purpose"`
		Inputs  *uws1.ParamSchema `json:"inputs"`
		Outputs *uws1.ParamSchema `json:"outputs"`
		Effect  string            `json:"effect"`
	}
	var request struct {
		Contract sharedFields `json:"contract"`
	}
	var pending sharedFields
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(pendingBytes, &pending); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request.Contract, pending) {
		t.Fatal("OpenUdon shared fields do not losslessly match the provisional UWS pending-step fields")
	}
	if request.Contract.Inputs == nil || request.Contract.Inputs.Type != "object" ||
		request.Contract.Outputs == nil || request.Contract.Outputs.Type != "object" {
		t.Fatal("shared input and output ParamSchema roots must be objects")
	}
	for name, value := range map[string]*uws1.ParamSchema{
		"inputs":  request.Contract.Inputs,
		"outputs": request.Contract.Outputs,
	} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatalf("marshal %s ParamSchema: %v", name, err)
		}
		var roundTrip uws1.ParamSchema
		if err := json.Unmarshal(encoded, &roundTrip); err != nil {
			t.Fatalf("unmarshal %s ParamSchema: %v", name, err)
		}
		if !reflect.DeepEqual(value, &roundTrip) {
			t.Fatalf("%s ParamSchema changed during UWS JSON round trip", name)
		}
	}
}

func TestSelectedOperationReferenceStaysBoundAcrossCommands(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	type operationRef struct {
		SourceKind   string `json:"source_kind"`
		SourceID     string `json:"source_id"`
		SourceSHA256 string `json:"source_sha256"`
		Selector     string `json:"native_selector"`
		OperationKey string `json:"operation_key"`
		OperationID  string `json:"operation_id,omitempty"`
	}
	var candidates struct {
		Result struct {
			Candidates []struct {
				OperationRef operationRef `json:"operation_ref"`
			} `json:"candidates"`
		} `json:"result"`
	}
	if err := readJSONFixture(filepath.Join(root, "results", "step-candidates.json"), &candidates); err != nil {
		t.Fatal(err)
	}
	if len(candidates.Result.Candidates) != 1 {
		t.Fatalf("candidate fixture has %d candidates, want one", len(candidates.Result.Candidates))
	}
	want := candidates.Result.Candidates[0].OperationRef
	for _, path := range []string{
		filepath.Join(root, "requests", "step-check.json"),
		filepath.Join(root, "requests", "step-bind.json"),
		filepath.Join(root, "results", "step-check.json"),
	} {
		var message struct {
			OperationRef operationRef `json:"operation_ref"`
			Result       struct {
				OperationRef operationRef `json:"operation_ref"`
			} `json:"result"`
		}
		if err := readJSONFixture(path, &message); err != nil {
			t.Fatal(err)
		}
		got := message.OperationRef
		if got == (operationRef{}) {
			got = message.Result.OperationRef
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s operation_ref = %#v, want candidate reference %#v", path, got, want)
		}
	}
}

func readJSONFixture(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}
