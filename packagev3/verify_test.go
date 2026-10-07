package packagev3_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/openudon/trust"
)

func packageDigest(t *testing.T, scope string, files map[string][]byte) string {
	t.Helper()
	artifacts := []handoff.DigestFile{}
	for path, data := range files {
		artifacts = append(artifacts, handoff.DigestFile{Path: path, SHA256: sha(data)})
	}
	sum, err := handoff.DigestFiles(scope, trust.PackageDigestVersion, artifacts)
	if err != nil {
		t.Fatal(err)
	}
	return sum
}
func TestVerifyOwnsExactIndependentSnapshot(t *testing.T) {
	p, err := packagev3.Build(context.Background(), buildOptions())
	if err != nil {
		t.Fatal(err)
	}
	v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
	if err != nil {
		t.Fatal(err)
	}
	p.Files[packagev3.WorkflowPath][0] = '!'
	snapshot := v.Snapshot()
	snapshot[packagev3.WorkflowPath][0] = '!'
	a := v.Assessment()
	a.Findings = append(a.Findings, packagev3.Finding{Code: "fake", Outcome: "incompatible"})
	if v.Snapshot()[packagev3.WorkflowPath][0] != 'u' || v.SHA256() != p.SHA256 || len(v.Assessment().Findings) == len(a.Findings) {
		t.Fatal("mutable proof snapshot")
	}
}
func TestVerifyRejectsRehashedRecordAndInventoryForgery(t *testing.T) {
	for name, mutate := range map[string]func(*packagev3.Package){
		"forged credentials": func(p *packagev3.Package) {
			p.Handoff.Credentials = []string{"undeclared_key"}
			p.Files[packagev3.HandoffPath], _ = p.Handoff.Marshal()
		},
		"unknown artifact": func(p *packagev3.Package) { p.Files["expected/private.json"] = []byte(`{}`) },
		"missing data":     func(p *packagev3.Package) { delete(p.Files, packagev3.DataPath) },
		"forged assessment": func(p *packagev3.Package) {
			p.Files[packagev3.AssessmentPath] = []byte(strings.Replace(string(p.Files[packagev3.AssessmentPath]), `"compatible"`, `"incompatible"`, 1))
		},
		"stale sources": func(p *packagev3.Package) { p.Files["sources/openapi/api.yaml"][0] = '!' },
		"forged shape": func(p *packagev3.Package) {
			p.Files[packagev3.ShapesPath] = []byte(strings.Replace(string(p.Files[packagev3.ShapesPath]), `"GET"`, `"DELETE"`, 1))
		},
		"scope": func(p *packagev3.Package) {
			p.Manifest.Scope = "workflows/W02-other"
			p.Files[packagev3.ManifestPath], _ = p.Manifest.Marshal()
		},
		"noncanonical manifest": func(p *packagev3.Package) {
			p.Files[packagev3.ManifestPath] = append(p.Files[packagev3.ManifestPath], '\n')
		},
		"legacy hcl": func(p *packagev3.Package) { p.Files["workflows/workflow.hcl"] = []byte(`uws="1.13.0"`) },
	} {
		t.Run(name, func(t *testing.T) {
			p, err := packagev3.Build(context.Background(), buildOptions())
			if err != nil {
				t.Fatal(err)
			}
			mutate(&p)
			// Even an attacker recomputing every advertised artifact/digest cannot
			// replace independent assessment/source proof or extend the closed inventory.
			for i := range p.Handoff.Artifacts {
				if data, ok := p.Files[p.Handoff.Artifacts[i].Path]; ok {
					p.Handoff.Artifacts[i].SHA256 = sha(data)
				}
			}
			p.Handoff.ManifestSHA256 = sha(p.Files[packagev3.ManifestPath])
			p.Handoff.AssessmentSHA256 = sha(p.Files[packagev3.AssessmentPath])
			if bytes, err := p.Handoff.Marshal(); err == nil {
				p.Files[packagev3.HandoffPath] = bytes
			}
			sum := packageDigest(t, p.Manifest.Scope, p.Files)
			v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: sum, Files: p.Files})
			if !errors.Is(err, packagev3.ErrPackage) || v.Snapshot() != nil {
				t.Fatal("forged verification", err)
			}
		})
	}
}
