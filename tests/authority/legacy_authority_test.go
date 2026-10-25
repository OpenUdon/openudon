package authority_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/wire"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

func authorityFixture(t *testing.T) authority.Authority {
	t.Helper()
	data, err := os.ReadFile("../../docs/fixtures/broker-handoff-v1/authority-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	var a authority.Authority
	if err := wire.DecodeStrict(data, &a); err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestConcreteAuthorityDriftAndBounds(t *testing.T) {
	a := authorityFixture(t)
	now, _ := time.Parse(time.RFC3339, "2026-10-05T00:01:00Z")
	if err := a.ValidateAt(now); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"owner", "agent", "grant", "occurrence", "package", "input", "executor", "operation", "credential"} {
		t.Run(field, func(t *testing.T) {
			changed := authorityFixture(t)
			switch field {
			case "owner":
				changed.OwnerID = "other-owner"
			case "agent":
				changed.AgentID = "other-agent"
			case "grant":
				changed.GrantRevisionSHA256 = strings.Repeat("f", 64)
			case "occurrence":
				changed.OccurrenceID = "next-occurrence"
			case "package":
				changed.PackageSHA256 = strings.Repeat("f", 64)
			case "input":
				changed.InputsSHA256 = strings.Repeat("f", 64)
			case "executor":
				changed.ExecutorSHA256 = strings.Repeat("f", 64)
			case "operation":
				changed.Operations[0].Method = "POST"
			case "credential":
				changed.Operations[0].Bindings[0].Revision = strings.Repeat("f", 64)
			}
			if changed.Validate() == nil {
				t.Fatal("accepted drift under the old policy digest")
			}
		})
	}
	for _, mutate := range []func(*authority.Authority){
		func(a *authority.Authority) { a.Version = "unsupported" },
		func(a *authority.Authority) { a.OccurrenceID = "" },
		func(a *authority.Authority) { a.Operations = append(a.Operations, a.Operations[0]) },
		func(a *authority.Authority) { a.Operations[0].Origin = "https://user:secret@example.test" },
		func(a *authority.Authority) { a.Operations[0].Origin = "https://example.test/path" },
		func(a *authority.Authority) { a.Operations[0].Method = "CONNECT" },
		func(a *authority.Authority) { a.Operations[0].Bindings[0].Kind = "oauth" },
		func(a *authority.Authority) { a.Operations[0].Bindings[0].Parameter = "Host" },
		func(a *authority.Authority) { a.Operations = make([]authority.Operation, authority.MaxOperations+1) },
	} {
		changed := authorityFixture(t)
		mutate(&changed)
		changed.PolicySHA256 = changed.Digest()
		if changed.Validate() == nil {
			t.Fatal("accepted malformed authority")
		}
	}
	if a.ValidateAt(now.Add(24*time.Hour)) == nil || a.ValidateAt(now.Add(-24*time.Hour)) == nil {
		t.Fatal("accepted expired/future approval")
	}
}

func TestPublishedAuthoritySchemaAndManifest(t *testing.T) {
	root := "../../docs/fixtures/broker-handoff-v1"
	data, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version string `json:"version"`
		Files   map[string]struct {
			Valid  bool   `json:"valid"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if wire.DecodeStrict(data, &manifest) != nil || manifest.Version != "openudon.broker-handoff-fixtures.v1" {
		t.Fatal("invalid fixture manifest")
	}
	schemaBytes, err := os.ReadFile("../../docs/schemas/openudon.broker-authority.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
	if err != nil {
		t.Fatal(err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("https://openudon.test/broker-authority", doc); err != nil {
		t.Fatal(err)
	}
	schema, err := compiler.Compile("https://openudon.test/broker-authority")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != len(manifest.Files)+1 {
		t.Fatal("unlisted fixture")
	}
	for name, spec := range manifest.Files {
		t.Run(name, func(t *testing.T) {
			if filepath.Base(name) != name {
				t.Fatal("fixture path")
			}
			data, err := os.ReadFile(filepath.Join(root, name))
			if err != nil || digest.SHA256(data) != spec.SHA256 {
				t.Fatal("fixture bytes changed")
			}
			var a authority.Authority
			err = wire.DecodeStrict(data, &a)
			if err == nil {
				err = a.Validate()
			}
			if (err == nil) != spec.Valid {
				t.Fatalf("valid=%v: %v", spec.Valid, err)
			}
			instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			// Digest mismatch and duplicate inventories require semantic checks;
			// every positive wire must also validate against the published schema.
			if spec.Valid {
				if err := schema.Validate(instance); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
	encoded, _ := json.Marshal(authorityFixture(t))
	var values map[string]any
	json.Unmarshal(encoded, &values)
	values["credential_value"] = "SECRET_CANARY"
	changed, _ := json.Marshal(values)
	var a authority.Authority
	if wire.DecodeStrict(changed, &a) == nil {
		t.Fatal("secret/unknown field admitted")
	}
}
