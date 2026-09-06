package authoring

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArtifactScanUsesOnlyValidHandoffDeclarationsForBindingPositions(t *testing.T) {
	name := "w8m_dedicated_password"
	manifest := NewReviewHandoff(ReviewHandoffOptions{
		HandoffInputs:      []ReviewHandoffInput{{Path: "project.md", Purpose: "brief", Required: true, SHA256: strings.Repeat("0", 64)}},
		OwnerSplit:         ReviewOwnerSplit{"openudon": {"review package"}},
		ExecutionPolicy:    DefaultReviewExecutionPolicy(true),
		CredentialBindings: ReviewCredentialBindings{Declared: []string{name}},
	})
	if diagnostics := ValidateReviewHandoff(manifest); len(diagnostics) != 0 {
		t.Fatal("fixture handoff")
	}
	data, _ := json.Marshal(manifest)
	artifacts := []Artifact{{Path: "expected/review-handoff.json", Content: data}, {Path: "workflows/workflow.uws.yaml", Content: []byte("credentialBindings:\n  password: w8m_dedicated_password\n")}}
	if hits := ScanCredentialValues(artifacts); len(hits) != 0 {
		t.Fatal("declared binding rejected")
	}
	if hits := ScanCredentialValues(artifacts[1:]); len(hits) == 0 {
		t.Fatal("declaration requirement bypassed")
	}
	artifacts[1].Content = []byte(`{"password":"w8m_dedicated_password"}`)
	if hits := ScanCredentialValues(artifacts); len(hits) == 0 {
		t.Fatal("ordinary password value exempted")
	}
	manifest.CredentialBindings.ValuesAllowedInArtifacts = true
	artifacts[0].Content, _ = json.Marshal(manifest)
	artifacts[1].Content = []byte("credentialBindings:\n  password: w8m_dedicated_password\n")
	if hits := ScanCredentialValues(artifacts); len(hits) == 0 {
		t.Fatal("invalid handoff allowed exemption")
	}
}

func TestValidateReviewHandoffRequiresSafePackage(t *testing.T) {
	manifest := NewReviewHandoff(ReviewHandoffOptions{
		HandoffInputs: []ReviewHandoffInput{{Path: "project.md", Purpose: "brief", Required: true, SHA256: strings.Repeat("0", 64)}},
		OwnerSplit: ReviewOwnerSplit{
			"openudon": {"review package"},
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
	manifest.HandoffInputs = append(manifest.HandoffInputs, ReviewHandoffInput{Path: "../secret.txt", Required: true, SHA256: strings.Repeat("0", 64)})
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
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  inputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := ComputeReviewHandoffDigest(ReviewHandoffDigestOptions{
		Root:    root,
		Scope:   "examples/demo",
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  []ReviewHandoffInput{inputs[1], inputs[0]},
	})
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("digest should be stable across input order: %s != %s", first, second)
	}
}

func TestComputeReviewHandoffDigestRejectsSymlinkInput(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.md")
	mustWrite(t, target, []byte("brief\n"))
	if err := os.Symlink(target, filepath.Join(root, "project.md")); err != nil {
		t.Fatal(err)
	}

	_, err := ComputeReviewHandoffDigest(ReviewHandoffDigestOptions{
		Root:    root,
		Scope:   "examples/demo",
		Version: "openudon.handoff-package-digest.v1",
		Inputs: []ReviewHandoffInput{
			{Path: "project.md", Required: true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink digest input rejection, got %v", err)
	}
}

func TestComputeReviewHandoffDigestIsBoundedAndCancelable(t *testing.T) {
	root := t.TempDir()
	oversize := filepath.Join(root, "project.md")
	file, err := os.Create(oversize)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(8<<20 + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	opts := ReviewHandoffDigestOptions{Root: root, Inputs: []ReviewHandoffInput{{Path: "project.md", Required: true}}}
	if _, err := ComputeReviewHandoffDigest(opts); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("expected bounded-read failure, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	opts.Context = ctx
	if _, err := ComputeReviewHandoffDigest(opts); err != context.Canceled {
		t.Fatalf("expected context cancellation, got %v", err)
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
