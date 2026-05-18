package n8nbridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateAcceptsAuthoringAssistanceSummary(t *testing.T) {
	summary := Summary{
		Version:  SummaryVersion,
		Fixture:  "n8n-slack-message-post",
		Boundary: "authoring_assistance_only",
		Source:   SourceEvidence{Kind: "n8n_workflow_fixture", Paths: []string{"reference/n8n.json"}},
		Services: []Service{{Name: "Slack", Operations: []string{"postMessage"}}},
		Nodes: []Node{{
			Name:               "Slack",
			Type:               "n8n-nodes-base.slack",
			Resource:           "message",
			Operation:          "post",
			OpenUdonStep:       "post_message",
			OpenAPIOperationID: "postMessage",
			MappingStatus:      "advisory",
		}},
		OpenAPICandidates: []OpenAPICandidate{{Path: "openapi/slack.json", OperationID: "postMessage", Status: "fixture-local"}},
		CredentialBindings: []CredentialBinding{{
			Name:    "slack_bot_token",
			Service: "Slack",
		}},
		UnsupportedSemantics: []UnsupportedDiagnostic{{
			Semantic: "manual_trigger",
			Handling: "diagnostic",
		}},
		GeneratedCandidates: GeneratedCandidates{
			ProjectPath: "project.md",
			IntentPath:  "reference/intent.hcl",
			Promoted:    false,
		},
		Validation: ValidationEvidence{Status: "advisory", Gates: []string{"icot lint"}},
	}
	if err := Validate(summary); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRejectsExecutableImportBoundary(t *testing.T) {
	err := Validate(Summary{
		Version:  SummaryVersion,
		Fixture:  "bad",
		Boundary: "executable_import",
		Source:   SourceEvidence{Kind: "n8n_workflow_fixture"},
		Services: []Service{{Name: "Slack"}},
		Nodes:    []Node{{Name: "Slack", Type: "n8n-nodes-base.slack", MappingStatus: "mapped"}},
		Validation: ValidationEvidence{
			Status: "advisory",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "authoring_assistance_only") {
		t.Fatalf("expected boundary failure, got %v", err)
	}
}

func TestValidateRootFindsReferenceBridgeSummaries(t *testing.T) {
	root := t.TempDir()
	writeSummaryFile(t, filepath.Join(root, "one", "project.md"), "# one\n")
	writeSummaryFile(t, filepath.Join(root, "one", "reference", "intent.hcl"), `workflow { name = "one" }`)
	writeSummaryFile(t, filepath.Join(root, "one", "reference", "n8n-bridge.json"), `{
  "version": "openudon.n8n-pattern-summary.v1",
  "fixture": "one",
  "boundary": "authoring_assistance_only",
  "source": {"kind": "n8n_workflow_fixture"},
  "services": [{"name": "Slack"}],
  "nodes": [{"name": "Slack", "type": "n8n-nodes-base.slack", "mapping_status": "advisory"}],
  "generated_candidates": {"project_path": "project.md", "intent_path": "reference/intent.hcl", "promoted": false},
  "validation": {"status": "advisory"}
}`)
	writeSummaryFile(t, filepath.Join(root, "two", "reference", "ignored.json"), `{}`)
	results, err := ValidateRoot(root)
	if err != nil {
		t.Fatalf("ValidateRoot() error = %v", err)
	}
	if len(results) != 1 || results[0].Summary.Fixture != "one" {
		t.Fatalf("results = %#v, want one summary", results)
	}
}

func TestValidateFileRejectsMissingCandidatePath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "one", "reference", "n8n-bridge.json")
	writeSummaryFile(t, path, `{
  "version": "openudon.n8n-pattern-summary.v1",
  "fixture": "one",
  "boundary": "authoring_assistance_only",
  "source": {"kind": "n8n_workflow_fixture"},
  "services": [{"name": "Slack"}],
  "nodes": [{"name": "Slack", "type": "n8n-nodes-base.slack", "mapping_status": "advisory"}],
  "generated_candidates": {"project_path": "project.md", "promoted": false},
  "validation": {"status": "advisory"}
}`)
	_, err := ValidateFile(path)
	if err == nil || !strings.Contains(err.Error(), "generated_candidates.project_path does not exist") {
		t.Fatalf("expected missing candidate path failure, got %v", err)
	}
}

func TestValidateFileRejectsEscapedCandidatePath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "one", "reference", "n8n-bridge.json")
	writeSummaryFile(t, path, `{
  "version": "openudon.n8n-pattern-summary.v1",
  "fixture": "one",
  "boundary": "authoring_assistance_only",
  "source": {"kind": "n8n_workflow_fixture"},
  "services": [{"name": "Slack"}],
  "nodes": [{"name": "Slack", "type": "n8n-nodes-base.slack", "mapping_status": "advisory"}],
  "generated_candidates": {"project_path": "..", "promoted": false},
  "validation": {"status": "advisory"}
}`)
	_, err := ValidateFile(path)
	if err == nil || !strings.Contains(err.Error(), "generated_candidates.project_path must stay inside the fixture") {
		t.Fatalf("expected escaped candidate path failure, got %v", err)
	}
}

func TestValidateFileRejectsCandidateDirectory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "one", "reference", "n8n-bridge.json")
	writeSummaryFile(t, filepath.Join(root, "one", "project.md"), "# one\n")
	writeSummaryFile(t, path, `{
  "version": "openudon.n8n-pattern-summary.v1",
  "fixture": "one",
  "boundary": "authoring_assistance_only",
  "source": {"kind": "n8n_workflow_fixture"},
  "services": [{"name": "Slack"}],
  "nodes": [{"name": "Slack", "type": "n8n-nodes-base.slack", "mapping_status": "advisory"}],
  "generated_candidates": {"project_path": "reference", "promoted": false},
  "validation": {"status": "advisory"}
}`)
	_, err := ValidateFile(path)
	if err == nil || !strings.Contains(err.Error(), "generated_candidates.project_path must be a regular file") {
		t.Fatalf("expected candidate directory failure, got %v", err)
	}
}

func writeSummaryFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
