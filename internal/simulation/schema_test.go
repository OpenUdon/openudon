package simulation

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func simulationSchema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	for _, resource := range []string{"openudon.step-authoring.v1", "openudon.simulate-input.v1", "openudon.simulate.v1"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "docs", "schemas", resource+".schema.json"))
		if err != nil {
			t.Fatal(err)
		}
		document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if err := compiler.AddResource("https://openudon.dev/schemas/"+resource+".schema.json", document); err != nil {
			t.Fatal(err)
		}
	}
	schema, err := compiler.Compile("https://openudon.dev/schemas/" + name + ".schema.json")
	if err != nil {
		t.Fatal(err)
	}
	return schema
}
func validateSimulationJSON(t *testing.T, schema *jsonschema.Schema, data []byte) {
	t.Helper()
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err == nil {
		err = schema.Validate(instance)
	}
	if err != nil {
		t.Fatal(err)
	}
}

func TestPublishedSimulationFixturesAndRuntimeConform(t *testing.T) {
	schema := simulationSchema(t, Version)
	root := filepath.Join("..", "..", "docs", "examples", "simulation", "v1")
	for _, name := range []string{"pending.json", "blocked.json"} {
		data, err := os.ReadFile(filepath.Join(root, "results", name))
		if err != nil {
			t.Fatal(err)
		}
		validateSimulationJSON(t, schema, data)
	}
	input, err := os.ReadFile(filepath.Join(root, "inputs", "example.json"))
	if err != nil {
		t.Fatal(err)
	}
	validateSimulationJSON(t, simulationSchema(t, InputVersion), input)
	repo := t.TempDir()
	relative := filepath.Join("docs", "examples", "simulation", "v1", "example")
	destination := filepath.Join(repo, relative)
	if err := filepath.WalkDir(filepath.Join(root, "example"), func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(filepath.Join(root, "example"), path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		write(t, filepath.Join(destination, relative), data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	report := Run(context.Background(), Options{RepoRoot: repo, ExampleDir: destination})
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	validateSimulationJSON(t, schema, encoded)
	expected, err := os.ReadFile(filepath.Join(root, "results", "pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	var normalized bytes.Buffer
	if err := json.Compact(&normalized, expected); err != nil {
		t.Fatal(err)
	}
	if report.Status != "completed" || string(encoded) != normalized.String() {
		t.Fatalf("published fixture drift: %s", encoded)
	}
}

func TestSimulationSchemaRejectsAuthorityAndPendingEndpointDrift(t *testing.T) {
	schema := simulationSchema(t, Version)
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "examples", "simulation", "v1", "results", "pending.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, changed := range []string{strings.Replace(string(data), `"simulation"`, `"sandbox"`, 1), strings.Replace(string(data), `"unresolved"`, `"bound"`, 1), strings.Replace(string(data), `"synthesized"`, `"live-read"`, -1), strings.Replace(string(data), `"package_unchanged": true`, `"package_unchanged": false`, 1)} {
		instance, err := jsonschema.UnmarshalJSON(strings.NewReader(changed))
		if err == nil {
			err = schema.Validate(instance)
		}
		if err == nil {
			t.Fatal("invalid simulation authority accepted")
		}
	}
}

func TestPublishedSimulationRefusalVectors(t *testing.T) {
	schema := simulationSchema(t, Version)
	root := filepath.Join("..", "..", "docs", "examples", "simulation", "v1", "invalid")
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
		if err == nil {
			err = schema.Validate(instance)
		}
		if err == nil {
			t.Fatal("invalid published vector accepted:", entry.Name())
		}
	}
}
