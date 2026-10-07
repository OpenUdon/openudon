package packagev3_test

import (
	"encoding/json"
	"github.com/OpenUdon/openudon/packagev3"
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
