package uwsvalidate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/uwsschema"
)

func TestValidateFileAcceptsJSONAndYAML(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(schema, []byte(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "required": ["name"],
  "properties": {"name": {"type": "string"}}
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	jsonDoc := filepath.Join(dir, "doc.json")
	if err := os.WriteFile(jsonDoc, []byte(`{"name":"openudon"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	yamlDoc := filepath.Join(dir, "doc.yaml")
	if err := os.WriteFile(yamlDoc, []byte("name: openudon\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, doc := range []string{jsonDoc, yamlDoc} {
		if err := ValidateFile(schema, doc); err != nil {
			t.Fatalf("ValidateFile(%s) returned error: %v", doc, err)
		}
	}
}

func TestValidateFileRejectsInvalidAndUnsupportedDocuments(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(schema, []byte(`{"type":"object","required":["name"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	invalid := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalid, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFile(schema, invalid); err == nil {
		t.Fatalf("expected schema validation failure")
	}
	unsupported := filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(unsupported, []byte(`{"name":"openudon"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFile(schema, unsupported); err == nil || !strings.Contains(err.Error(), "unsupported document extension") {
		t.Fatalf("expected unsupported extension error, got %v", err)
	}
}

func TestValidateFileAcceptsUWS12TypedSourceDocument(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "workflow.uws.json")
	if err := os.WriteFile(doc, []byte(`{
  "uws": "1.2.0",
  "info": {"title": "typed source", "version": "1.0.0"},
  "sourceDescriptions": [
    {"name": "gmail", "url": "google-discovery/gmail.json", "type": "google-discovery"}
  ],
  "operations": [
    {"operationId": "send", "sourceDescription": "gmail", "sourceOperationId": "gmail_users_messages_send"}
  ],
  "workflows": [
    {"workflowId": "main", "type": "sequence", "steps": [{"stepId": "send", "operationRef": "send"}]}
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	schema := uwsschema.PathForVersion(dir, "1.2.0")
	if err := ValidateFile(schema, doc); err != nil {
		t.Fatalf("ValidateFile returned error: %v", err)
	}
}
