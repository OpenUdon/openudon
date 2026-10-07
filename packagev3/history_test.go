package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/OpenUdon/openudon/approval"
	"github.com/OpenUdon/openudon/wire"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/trust"
)

func legacyHistory(t *testing.T) packagev3.HistoryRequest {
	t.Helper()
	path := "expected/review-handoff.json"
	workflow := []byte(`uws = "1.12.0"\n# old packaged HCL stays opaque\n`)
	manifest := handoff.NewReviewHandoff(handoff.ReviewHandoffOptions{HandoffInputs: []handoff.ReviewHandoffInput{{Path: "workflows/workflow.hcl", Required: true, SHA256: digest.SHA256(workflow)}, {Path: path, Required: true, SHA256: strings.Repeat("0", 64)}}, OwnerSplit: handoff.ReviewOwnerSplit{"host": []string{"authority"}}})
	self, err := handoff.ReviewHandoffSelfDigest(manifest, path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range manifest.HandoffInputs {
		if manifest.HandoffInputs[i].Path == path {
			manifest.HandoffInputs[i].SHA256 = self
		}
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{path: raw, "workflows/workflow.hcl": workflow}
	result, err := trust.Inspect(context.Background(), trust.Request{Scope: "workflows/W01-legacy", ManifestPath: path, Files: files, RequiredPaths: []string{"workflows/workflow.hcl"}})
	if err != nil {
		t.Fatal(err)
	}
	return packagev3.HistoryRequest{FormatVersion: handoff.ReviewHandoffVersion, Scope: result.Scope, ExpectedSHA256: result.PackageSHA256, Files: files, LegacyManifestPath: path, LegacyRequiredPaths: []string{"workflows/workflow.hcl"}}
}
func TestExplicitHistoryDispatchPreservesBytesWithoutProofOrUpgrade(t *testing.T) {
	legacy := legacyHistory(t)
	before := append([]byte(nil), legacy.Files[legacy.LegacyManifestPath]...)
	inspected, err := packagev3.InspectHistory(context.Background(), legacy)
	if err != nil {
		t.Fatal(err)
	}
	if !inspected.ReadOnly || inspected.SourceProof || inspected.FormatVersion != handoff.ReviewHandoffVersion || inspected.PackageSHA256 != legacy.ExpectedSHA256 || !bytes.Equal(before, legacy.Files[legacy.LegacyManifestPath]) {
		t.Fatal("legacy inspection upgraded or manufactured proof")
	}
	// New-stack source verification cannot consume the legacy HCL snapshot.
	if _, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: legacy.Scope, ExpectedSHA256: legacy.ExpectedSHA256, Files: legacy.Files}); err == nil {
		t.Fatal("legacy implicitly upgraded")
	}
	options := runtimeOptions(t)
	p, err := packagev3.Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	request := packagev3.HistoryRequest{FormatVersion: packagev3.PackageVersion, Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files}
	// History needs no private runtime or source-semantic verifier. Its result
	// remains explicitly read-only and cannot replace VerifiedPackage.
	result, err := packagev3.InspectHistory(context.Background(), request)
	if err != nil || !result.ReadOnly || result.SourceProof || result.PackageSHA256 != p.SHA256 {
		t.Fatal("new history proof boundary", err)
	}
}
func TestHistoryRefusesAmbiguousVersionDriftAndPrivateValues(t *testing.T) {
	for _, mutate := range []func(*packagev3.HistoryRequest){
		func(r *packagev3.HistoryRequest) { r.FormatVersion = "" },
		func(r *packagev3.HistoryRequest) { r.FormatVersion = "openudon.package.v99" },
		func(r *packagev3.HistoryRequest) { r.FormatVersion = packagev3.PackageVersion },
		func(r *packagev3.HistoryRequest) { r.Scope = "workflows/W02-other" },
		func(r *packagev3.HistoryRequest) { r.ExpectedSHA256 = strings.Repeat("a", 64) },
		func(r *packagev3.HistoryRequest) { r.Files["workflows/workflow.hcl"][0] = '!' },
		func(r *packagev3.HistoryRequest) { r.LegacyRequiredPaths = nil },
		func(r *packagev3.HistoryRequest) { r.Private = func([]byte) bool { return true } },
	} {
		r := legacyHistory(t)
		mutate(&r)
		_, err := packagev3.InspectHistory(context.Background(), r)
		if !errors.Is(err, packagev3.ErrPackage) {
			t.Fatal("unselected or drifting history", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := packagev3.InspectHistory(ctx, legacyHistory(t)); !errors.Is(err, context.Canceled) {
		t.Fatal("history cancellation")
	}
}

func TestLegacyApprovalCannotAuthorizeV3Successor(t *testing.T) {
	legacy := legacyHistory(t)
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	old := approval.Approval{Version: approval.Version, Scope: legacy.Scope, State: approval.StateApprovedForSandbox, Reviewer: "owner", ApprovedAt: now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Minute).Format(time.RFC3339), PackageSHA256: legacy.ExpectedSHA256}
	if approval.Validate(old, legacy.Scope, legacy.ExpectedSHA256, "sandbox", now) != nil {
		t.Fatal("old approval is unreadable")
	}
	options := buildOptions()
	options.Scope = legacy.Scope
	v := verifiedPackage(t, options)
	if v.SHA256() == old.PackageSHA256 || approval.Validate(old, legacy.Scope, v.SHA256(), "sandbox", now) == nil {
		t.Fatal("legacy approval carried into v3")
	}
	// Retained old report/evidence readers validate old identities through their
	// own versioned public APIs; they never upgrade this approval or package.
	manifestBytes := legacy.Files[legacy.LegacyManifestPath]
	var oldManifest handoff.ReviewHandoff
	if wire.DecodeStrictNumbers(manifestBytes, &oldManifest) != nil || len(handoff.ValidateReviewHandoff(oldManifest)) != 0 {
		t.Fatal("old handoff wire unreadable")
	}
	if _, err := packagev3.ParseHandoff(manifestBytes); err == nil {
		t.Fatal("v2 became v3 handoff")
	}
}
