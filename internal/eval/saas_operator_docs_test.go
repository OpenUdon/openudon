package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaaSOperatorReleaseDocsNameDemoAndBoundaries(t *testing.T) {
	root := filepath.Join("..", "..")
	doc := readRepoFile(t, root, "docs", "saas-operator-release.md")
	for _, want := range []string{
		"gmail-send-audit-receipt",
		"order-fulfillment-chain",
		"go run ./cmd/openudon approval-template",
		"go run ./cmd/openudon run",
		"--dry-run",
		"go run ./cmd/openudon n8n-bridge validate --root examples/eval",
		"run n8n workflows",
		"execute Terraform/OpenTofu",
		"implement Symphony reviewer identity",
		"compile or execute generic UWS/OpenAPI workflows itself",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("saas operator release doc missing %q", want)
		}
	}
}

func TestReleaseNoteTemplateCapturesSaaSOperatorEvidence(t *testing.T) {
	root := filepath.Join("..", "..")
	doc := readRepoFile(t, root, "docs", "release-note-template.md")
	for _, want := range []string{
		"SaaS operator demo fixtures",
		"SaaS operator demo dry-run result",
		"n8n bridge validation result",
		"Selected SaaS fixture lint",
		"SaaS demo trusted dry-run",
		"Provider Drift Watch",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("release note template missing %q", want)
		}
	}
}

func readRepoFile(t *testing.T, root string, elems ...string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(append([]string{root}, elems...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
