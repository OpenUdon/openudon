package packagev3_test

import (
	"encoding/json"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/trust"
	"sort"
	"strings"
	"testing"
)

func manifest() packagev3.Manifest {
	return packagev3.Manifest{Version: packagev3.PackageVersion, Scope: "workflows/W01-fixture", Workflow: packagev3.Artifact{Path: packagev3.WorkflowPath, SHA256: strings.Repeat("a", 64)}, Data: packagev3.Artifact{Path: packagev3.DataPath, SHA256: strings.Repeat("b", 64)}, Shapes: packagev3.Artifact{Path: packagev3.ShapesPath, SHA256: strings.Repeat("c", 64)}, ShapeVersion: packagev3.ShapeVersion, Sources: []packagev3.Source{{ID: "api", Kind: "openapi", Artifact: packagev3.Artifact{Path: "sources/openapi/api.json", SHA256: strings.Repeat("d", 64)}}}}
}

func TestManifestRetainsDigestV1AndRejectsHCL(t *testing.T) {
	m := manifest()
	sum, err := m.InputDigest()
	if err != nil {
		t.Fatal(err)
	}
	files, err := m.InputArtifacts()
	if err != nil {
		t.Fatal(err)
	}
	var old []handoff.DigestFile
	for _, file := range files {
		old = append(old, handoff.DigestFile{Path: file.Path, SHA256: file.SHA256})
	}
	expected, err := handoff.DigestFiles(m.Scope, trust.PackageDigestVersion, old)
	if err != nil || sum != expected {
		t.Fatal("digest-v1 algorithm changed")
	}
	for _, mutate := range []func(*packagev3.Manifest){func(m *packagev3.Manifest) { m.Workflow.Path = "workflows/workflow.hcl" }, func(m *packagev3.Manifest) { m.Sources[0].Artifact.Path = "sources/openapi/intent.hcl" }, func(m *packagev3.Manifest) { m.Data.SHA256 = " " + m.Data.SHA256 }, func(m *packagev3.Manifest) { m.Sources = append(m.Sources, m.Sources[0]) }, func(m *packagev3.Manifest) { m.Scope = "../private" }} {
		copy := manifest()
		mutate(&copy)
		if copy.Validate() == nil {
			t.Fatal("invalid v3 identity/path admitted")
		}
	}
	files[0].SHA256 = strings.Repeat("0", 64)
	again, _ := m.InputDigest()
	if again != sum {
		t.Fatal("returned inventory aliases manifest")
	}
}

func TestRecordJSONRejectsMissingNullAliasesAndExtras(t *testing.T) {
	data, err := manifest().Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packagev3.ParseManifest(data); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(string(data), `"scope":`, `"Scope":`, 1), strings.Replace(string(data), `"sources":[`, `"sources":null,"extra":[`, 1), strings.Replace(string(data), `"path":`, `"Path":`, 1), string(data) + "{}", `{"version":"openudon.package.v3","Version":"openudon.package.v3"}`, `{"version":"openudon.package.v3"}`} {
		if _, err := packagev3.ParseManifest([]byte(bad)); err == nil {
			t.Fatal("open/ambiguous record admitted")
		}
	}
}

func TestHandoffAndAssessmentCannotClaimApproval(t *testing.T) {
	artifacts := []packagev3.Artifact{{Path: packagev3.DataPath, SHA256: strings.Repeat("a", 64)}, {Path: packagev3.AssessmentPath, SHA256: strings.Repeat("b", 64)}, {Path: packagev3.ShapesPath, SHA256: strings.Repeat("c", 64)}, {Path: packagev3.ManifestPath, SHA256: strings.Repeat("d", 64)}, {Path: packagev3.WorkflowPath, SHA256: strings.Repeat("e", 64)}}
	// Artifacts are canonical sorted inventory; Handoff never contains itself.
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	h := packagev3.Handoff{Version: packagev3.HandoffVersion, Scope: "workflows/W01-fixture", InputsSHA256: strings.Repeat("f", 64), ManifestSHA256: strings.Repeat("d", 64), AssessmentSHA256: strings.Repeat("b", 64), Artifacts: artifacts, ReviewState: "review_required", Credentials: []string{"symbolic_key"}}
	bytes, err := h.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packagev3.ParseHandoff(bytes); err != nil {
		t.Fatal(err)
	}
	h.ReviewState = "approved_for_production"
	if h.Validate() == nil {
		t.Fatal("record manufactured approval")
	}
	a := packagev3.Assessment{Version: packagev3.AssessmentVersion, Scope: "workflows/W01-fixture", InputsSHA256: strings.Repeat("f", 64), ManifestSHA256: strings.Repeat("d", 64), Outcome: "indeterminate", Findings: []packagev3.Finding{{Code: "shape.unknown", Path: "sources/openapi/api.json", Outcome: "indeterminate"}}}
	bytes, err = json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := packagev3.ParseAssessment(bytes); err != nil {
		t.Fatal(err)
	}
	a.Outcome = "approved"
	if a.Validate() == nil {
		t.Fatal("assessment manufactured authority")
	}
}
