package trustedrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/runevidence"
)

func TestPublicVerifierAcceptsExactPrivateDryRunEvidence(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, fixedNow())
	result, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath, DryRun: true, Now: fixedNow(), Assess: passAssess})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.RunEvidencePath)
	if err != nil {
		t.Fatal(err)
	}
	e := readRunEvidenceFile(t, result.RunEvidencePath)
	artifacts := map[string][]byte{}
	for _, ref := range e.AsyncEvidenceFiles {
		b, err := os.ReadFile(filepath.Join(result.WorkDir, filepath.FromSlash(ref.Path)))
		if err != nil {
			t.Fatal(err)
		}
		artifacts[ref.Path] = b
	}
	verified, err := runevidence.Verify(context.Background(), runevidence.Request{Evidence: data, Artifacts: artifacts})
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Evidence.DryRun || verified.Evidence.RunID != e.RunID {
		t.Fatal("dry run identity/posture changed")
	}
}
