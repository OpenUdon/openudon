package packagev3_test

import (
	"encoding/json"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/wire"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicV3SchemasAreClosedAndMatchRecords(t *testing.T) {
	for _, version := range []string{packagev3.PackageVersion, packagev3.HandoffVersion, packagev3.AssessmentVersion} {
		data, err := os.ReadFile(filepath.Join("..", "docs", "schemas", version+".schema.json"))
		if err != nil {
			t.Fatal(err)
		}
		var document any
		if json.Unmarshal(data, &document) != nil {
			t.Fatal("schema JSON")
		}
		compiler := jsonschema.NewCompiler()
		uri := "https://openudon.test/" + version
		if compiler.AddResource(uri, document) != nil {
			t.Fatal("schema resource")
		}
		schema, err := compiler.Compile(uri)
		if err != nil {
			t.Fatal(err)
		}
		if version == packagev3.PackageVersion {
			data, err := manifest().Marshal()
			if err != nil {
				t.Fatal(err)
			}
			var record any
			_ = json.Unmarshal(data, &record)
			if schema.Validate(record) != nil {
				t.Fatal("valid record disagrees with schema")
			}
			record.(map[string]any)["sources"] = nil
			if schema.Validate(record) == nil {
				t.Fatal("null source inventory admitted")
			}
		}
		if schema.Validate(map[string]any{"version": version, "unknown": "private-value"}) == nil {
			t.Fatal("open/incomplete schema admitted")
		}
	}
}

func TestPackageAndPlanGoldenSchemasAgreeWithClosedRecords(t *testing.T) {
	for _, fixture := range []struct{ schema, file string }{
		{packagev3.PackageVersion, "package/expected/package.json"},
		{packagev3.HandoffVersion, "package/expected/review-handoff.json"},
		{packagev3.AssessmentVersion, "package/expected/assessment.json"},
		{packagev3.ExecutionPlanVersion, "plan.json"},
	} {
		schemaBytes, err := os.ReadFile(filepath.Join("..", "docs", "schemas", fixture.schema+".schema.json"))
		if err != nil {
			t.Fatal(err)
		}
		var document any
		if json.Unmarshal(schemaBytes, &document) != nil {
			t.Fatal("schema")
		}
		compiler := jsonschema.NewCompiler()
		uri := "https://openudon.test/" + fixture.schema
		if compiler.AddResource(uri, document) != nil {
			t.Fatal("resource")
		}
		schema, err := compiler.Compile(uri)
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join("..", "docs", "fixtures", "package-v3-v1", filepath.FromSlash(fixture.file)))
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if wire.DecodeStrictNumbers(data, &value) != nil || schema.Validate(value) != nil {
			t.Fatal("emitted record disagrees with schema", fixture.schema)
		}
		value.(map[string]any)["private_unreviewed_field"] = "synthetic"
		if schema.Validate(value) == nil {
			t.Fatal("open schema admitted unknown field", fixture.schema)
		}
	}
}
