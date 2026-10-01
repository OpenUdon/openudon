package stepauthoringcontract

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/internal/stepauthoring"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestPublishedPendingProtocolConformsWithoutNetwork(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	for _, name := range []string{"openudon.step-authoring.v1", "openudon.step-pending.v1"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "docs", "schemas", name+".schema.json"))
		if err != nil {
			t.Fatal(err)
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if err := compiler.AddResource("https://openudon.dev/schemas/"+name+".schema.json", document); err != nil {
			t.Fatal(err)
		}
	}
	schema, err := compiler.Compile("https://openudon.dev/schemas/openudon.step-pending.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Join("..", "..", "docs", "examples", "step-pending", "v1")
	var request stepauthoring.PendingRequest
	for _, group := range []string{"requests", "results"} {
		data, err := os.ReadFile(filepath.Join(root, group, "pending.json"))
		if err != nil {
			t.Fatal(err)
		}
		instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err == nil {
			err = schema.Validate(instance)
		}
		if err != nil {
			t.Fatal(err)
		}
		if group == "requests" && json.Unmarshal(data, &request) != nil {
			t.Fatal("invalid pending fixture")
		}
	}
	outcome := stepauthoring.Pending(context.Background(), t.TempDir(), request)
	if outcome.ExitCode != 0 {
		t.Fatalf("fixture execution failed: %+v", outcome)
	}
	data, err := json.Marshal(outcome.Result)
	if err != nil {
		t.Fatal(err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err == nil {
		err = schema.Validate(instance)
	}
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join(root, "results", "pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, expected); err != nil {
		t.Fatal(err)
	}
	if string(data) != compact.String() {
		t.Fatalf("pending fixture result/digest drift: %s", data)
	}
}

func TestPublishedPendingInvalidEffectRefusesBeforeWrites(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "step-pending", "v1", "invalid", "invalid-effect.json"))
	if err != nil {
		t.Fatal(err)
	}
	var request stepauthoring.PendingRequest
	if json.Unmarshal(data, &request) != nil {
		t.Fatal("malformed refusal vector")
	}
	root := t.TempDir()
	outcome := stepauthoring.Pending(context.Background(), root, request)
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.ExitCode == 0 || len(entries) != 0 {
		t.Fatal("invalid-effect vector wrote package content")
	}
}
