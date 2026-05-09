package uwsexec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDocumentFileReadsYAMLAndJSON(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "workflow.uws.yaml")
	jsonPath := filepath.Join(dir, "workflow.uws.json")
	mustWriteUWSExecTestFile(t, yamlPath, []byte("uws: 1.0.0\ninfo:\n  title: Test\n  version: 1.0.0\noperations: []\n"))
	mustWriteUWSExecTestFile(t, jsonPath, []byte(`{"uws":"1.1.0","info":{"title":"Test","version":"1.0.0"},"operations":[]}`))

	doc, err := LoadDocumentFile(yamlPath, DocumentFormatAuto)
	if err != nil {
		t.Fatalf("LoadDocumentFile YAML returned error: %v", err)
	}
	if doc.UWS != "1.0.0" {
		t.Fatalf("YAML UWS = %q", doc.UWS)
	}

	doc, err = LoadDocumentFile(jsonPath, DocumentFormatAuto)
	if err != nil {
		t.Fatalf("LoadDocumentFile JSON returned error: %v", err)
	}
	if doc.UWS != "1.1.0" {
		t.Fatalf("JSON UWS = %q", doc.UWS)
	}
}

func TestLoadDocumentFileReportsFormatAndReadErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "workflow.uws.yaml")
	mustWriteUWSExecTestFile(t, path, []byte("uws: 1.0.0\n"))

	if _, err := LoadDocumentFile(path, "toml"); err == nil || !strings.Contains(err.Error(), "unsupported UWS document format") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
	if _, err := LoadDocumentFile(filepath.Join(dir, "missing.uws.yaml"), DocumentFormatAuto); err == nil {
		t.Fatalf("expected missing file error")
	}
	badJSON := filepath.Join(dir, "bad.uws.json")
	mustWriteUWSExecTestFile(t, badJSON, []byte(`{"uws":`))
	if _, err := LoadDocumentFile(badJSON, DocumentFormatAuto); err == nil {
		t.Fatalf("expected malformed JSON error")
	}
}

func TestResolveFormatDefaultsUnknownExtensionToYAML(t *testing.T) {
	if got := resolveFormat("workflow.uws", DocumentFormatAuto); got != DocumentFormatYAML {
		t.Fatalf("resolveFormat unknown extension = %q", got)
	}
	if got := resolveFormat("workflow.uws.json", " JSON "); got != DocumentFormatJSON {
		t.Fatalf("resolveFormat explicit JSON = %q", got)
	}
}

func mustWriteUWSExecTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
