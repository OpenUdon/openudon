package packagev3_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/approval"
	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/wire"
)

func TestBrowserNamedOutputP10HistoryRemainsReadOnly(t *testing.T) {
	raw, err := os.ReadFile("testdata/browser/p11-p10-named-output-package.json")
	if err != nil || browsercontract.SHA256(raw) != "b23a5965c92a807b42f4bfcb309bb970e1d3398859fe2233d3c04ab27021490a" {
		t.Fatal("frozen P10 package fixture drift", err)
	}
	var old struct {
		ProducerRevision string            `json:"producer_revision"`
		SourceSHA256     string            `json:"source_sha256"`
		Scope            string            `json:"scope"`
		PackageSHA256    string            `json:"package_sha256"`
		Files            map[string][]byte `json:"files"`
	}
	if err := wire.DecodeStrictNumbers(raw, &old); err != nil || old.ProducerRevision != "78f8609d2e21976fa1311aa914cf2e4170b895ef" || old.SourceSHA256 != "56b48d53a4f1e6834bccab3bb4c1027b852b46fdd5b48986642c1791890bed87" || old.PackageSHA256 != "17eb76ce14ef6f616534b6fe66195e1daaa69f4f40eda98fd0a3caf6c2768051" || len(old.Files) != 8 {
		t.Fatal("frozen P10 provenance", err)
	}
	before := map[string][]byte{}
	for path, content := range old.Files {
		before[path] = append([]byte(nil), content...)
	}
	inspection, err := packagev3.InspectHistory(context.Background(), packagev3.HistoryRequest{FormatVersion: packagev3.PackageVersion, Scope: old.Scope, ExpectedSHA256: old.PackageSHA256, Files: old.Files})
	if err != nil || !inspection.ReadOnly || inspection.SourceProof || inspection.PackageSHA256 != old.PackageSHA256 {
		t.Fatal("P10 history lost read-only identity", err)
	}
	if _, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: old.Scope, ExpectedSHA256: old.PackageSHA256, Files: old.Files}); !errors.Is(err, packagev3.ErrPackage) {
		t.Fatal("stale output assessment acquired current verification", err)
	}
	o := browserOutputOptions(t, "count: '$response.body#/result_count'", "count: '$response.body#/result_count'", "count: '$steps.leaf.outputs.count'", "", "", false)
	p, err := packagev3.Build(context.Background(), o)
	if err != nil || p.Assessment.Outcome != "compatible" || p.SHA256 == old.PackageSHA256 {
		t.Fatal("corrected rebuild did not produce a new proof identity", err)
	}
	for path, content := range old.Files {
		if path == packagev3.AssessmentPath || path == packagev3.HandoffPath {
			if bytes.Equal(content, p.Files[path]) {
				t.Fatal("corrected proof artifact retained its old bytes", path)
			}
		} else if !bytes.Equal(content, p.Files[path]) {
			t.Fatal("correction rewrote an input or manifest", path)
		}
	}
	v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
	if err != nil || v.Assessment().Outcome != "compatible" {
		t.Fatal("corrected proof did not independently verify", err)
	}
	// A retained approval wire remains readable for its recorded identity. It
	// never migrates to the changed package or supplies browser authority.
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	oldApproval := approval.Approval{Version: approval.Version, Scope: old.Scope, State: approval.StateApprovedForSandbox, Reviewer: "fixture", ApprovedAt: now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Minute).Format(time.RFC3339), PackageSHA256: old.PackageSHA256}
	if approval.Validate(oldApproval, old.Scope, old.PackageSHA256, "sandbox", now) != nil || approval.Validate(oldApproval, old.Scope, p.SHA256, "sandbox", now) == nil {
		t.Fatal("retained approval changed or transferred identity")
	}
	if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, executionOptions()); err == nil {
		t.Fatal("compatible output assessment supplied native runtime admission")
	}
	if !reflect.DeepEqual(before, old.Files) {
		t.Fatal("historical package bytes were rewritten")
	}
}
