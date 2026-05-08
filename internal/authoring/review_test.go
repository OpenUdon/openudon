package authoring

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateReviewHandoffRequiresSafePackage(t *testing.T) {
	manifest := NewReviewHandoff(ReviewHandoffOptions{
		HandoffInputs: []ReviewHandoffInput{{Path: "project.md", Purpose: "brief", Required: true}},
		OwnerSplit: ReviewOwnerSplit{
			"ramen": {"review package"},
		},
		ExecutionPolicy: ReviewExecutionPolicy{
			SideEffectful:             true,
			RequiredNextState:         string(ReviewStateReviewRequired),
			SandboxProofRunState:      string(ReviewStateApprovedForSandbox),
			ProductionExecutionState:  string(ReviewStateApprovedForProduction),
			DirectProductionExecution: false,
		},
	})
	if diagnostics := ValidateReviewHandoff(manifest); len(diagnostics) != 0 {
		t.Fatalf("expected valid handoff, got %#v", diagnostics)
	}
	manifest.HandoffInputs = append(manifest.HandoffInputs, ReviewHandoffInput{Path: "../secret.txt", Required: true})
	if diagnostics := ValidateReviewHandoff(manifest); len(diagnostics) == 0 {
		t.Fatalf("expected unsafe input path diagnostic")
	}
}

func TestComputeReviewHandoffDigestIsStable(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "project.md"), []byte("brief\n"))
	mustWrite(t, filepath.Join(root, "expected", "quality.json"), []byte("{}\n"))
	inputs := []ReviewHandoffInput{
		{Path: "expected/quality.json", Required: true},
		{Path: "project.md", Required: true},
	}
	first, err := ComputeReviewHandoffDigest(ReviewHandoffDigestOptions{
		Root:    root,
		Scope:   "examples/demo",
		Version: "ramen.handoff-package-digest.v1",
		Inputs:  inputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ComputeReviewHandoffDigest(ReviewHandoffDigestOptions{
		Root:    root,
		Scope:   "examples/demo",
		Version: "ramen.handoff-package-digest.v1",
		Inputs:  []ReviewHandoffInput{inputs[1], inputs[0]},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("digest should be stable across input order: %s != %s", first, second)
	}
}

func TestScanCredentialValuesFlagsLiteralSecrets(t *testing.T) {
	diagnostics := ScanCredentialValues([]Artifact{{
		Path:    "project.md",
		Content: []byte(`api_key = "sk-proj-1234567890abcdef1234567890"`),
	}})
	if len(diagnostics) != 1 {
		t.Fatalf("expected credential diagnostic, got %#v", diagnostics)
	}
	if !strings.Contains(diagnostics[0].Remediation, "symbolic binding") {
		t.Fatalf("expected symbolic binding remediation, got %q", diagnostics[0].Remediation)
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
