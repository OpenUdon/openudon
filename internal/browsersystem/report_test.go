package browsersystem

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/browserscenario"
)

func offlineReport(t *testing.T) *Report {
	t.Helper()
	lock, err := browserscenario.LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	r := &Report{Toolchains: Toolchains{Go: lock.GoVersion, Node: "24.13.0"}, Version: Version, Suite: "offline", Status: "pass", FailureStage: "none", Baseline: lock, PlaywrightGo: "v0.6201.0", NetworkClaim: "application_request_allowlists_not_network_wide_containment"}
	for _, name := range []string{"openudon", "browsertools", "uws", "udon", "browserdriver"} {
		commit := strings.Repeat("a", 40)
		for _, c := range lock.Components {
			if c.Name == name {
				commit = c.Commit
			}
		}
		r.Sources = append(r.Sources, Source{Name: name, Commit: commit, SHA256: strings.Repeat("b", 64)})
	}
	pass := Pass{Number: 1}
	for _, id := range offlineStages {
		pass.Stages = append(pass.Stages, proof(id, Tests{Passed: 2, InventorySHA256: strings.Repeat("c", 64)}))
	}
	r.Passes = []Pass{pass}
	return r
}

func TestVersionedInventoriesPreserveLegacyMeaning(t *testing.T) {
	r := offlineReport(t)
	r.Version = legacyVersion
	if err := Validate(r); err != nil {
		t.Fatal(err)
	}
	if len(legacyLoopbackStages) != 11 || len(loopbackStages) != 13 || loopbackStages[11] != "supervised_registration_package" || loopbackStages[12] != "supervised_authenticated_package" {
		t.Fatal("versioned qualification inventory changed")
	}
	r.Version = "openudon.browser-system-qualification.v99"
	if Validate(r) == nil {
		t.Fatal("unknown inventory version accepted")
	}
}
func TestReportRejectsMissingReorderedTamperedAndExtraEvidence(t *testing.T) {
	for name, mutate := range map[string]func(*Report){
		"missing": func(r *Report) { r.Passes[0].Stages = r.Passes[0].Stages[:1] },
		"order": func(r *Report) {
			r.Passes[0].Stages[0], r.Passes[0].Stages[1] = r.Passes[0].Stages[1], r.Passes[0].Stages[0]
		},
		"digest": func(r *Report) { r.Passes[0].Stages[0].SHA256 = strings.Repeat("d", 64) },
		"empty_tests": func(r *Report) {
			r.Passes[0].Stages[0] = proof(offlineStages[0], Tests{InventorySHA256: strings.Repeat("c", 64)})
		},
		"runtime":             func(r *Report) { r.Toolchains.Node = "25.0.0" },
		"upstream":            func(r *Report) { r.Baseline.Playwright = "1.63.0" },
		"false_repeatability": func(r *Report) { r.Suite = "loopback" },
		"extra_private_field": func(r *Report) {
			s := &r.Passes[0].Stages[0]
			s.Evidence = json.RawMessage(`{"passed":2,"skipped":0,"inventory_sha256":"` + strings.Repeat("c", 64) + `","password":"synthetic"}`)
			s.SHA256 = hash(s.Evidence)
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := offlineReport(t)
			mutate(r)
			if Validate(r) == nil {
				t.Fatal("invalid report accepted")
			}
		})
	}
}
func TestReportRoundTripRejectsUnknownDuplicateAndNoncanonicalJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	r := offlineReport(t)
	if err := Write(path, r); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(path); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{append([]byte(" "), canonical(r)...), bytes.Replace(canonical(r), []byte(`"suite": "offline"`), []byte(`"suite": "offline", "suite": "offline"`), 1), bytes.Replace(canonical(r), []byte(`"suite": "offline"`), []byte(`"suite": "offline", "secret": "synthetic"`), 1)} {
		if os.WriteFile(path, data, 0600) != nil {
			t.Fatal("write")
		}
		if _, err := Verify(path); err == nil {
			t.Fatal("invalid file accepted")
		}
	}
}
func TestFailureReportCannotClaimSuccessOrResumeAfterFailure(t *testing.T) {
	r := offlineReport(t)
	r.Status = "fail"
	r.FailureStage = offlineStages[1]
	r.Passes[0].Stages = r.Passes[0].Stages[:2]
	r.Passes[0].Stages[1] = Stage{ID: offlineStages[1], Status: "fail"}
	if err := Validate(r); err != nil {
		t.Fatal(err)
	}
	r.Status = "pass"
	if Validate(r) == nil {
		t.Fatal("failure accepted as success")
	}
}
func TestOfflineCommandEnvironmentDoesNotInheritAuthority(t *testing.T) {
	for _, key := range []string{"NODE_OPTIONS", "GOFLAGS", "BROWSERDRIVER_REGISTRATION_LIVE_TEST", "BROWSERTOOLS_REGISTRATION_LIVE_TEST", "AWS_SECRET_ACCESS_KEY"} {
		t.Setenv(key, "synthetic")
	}
	for _, entry := range environment() {
		if strings.Contains(entry, "synthetic") {
			t.Fatal("ambient authority leaked")
		}
	}
}
func TestInvalidSuiteHasNoQualificationSideEffects(t *testing.T) {
	_, err := Run(context.Background(), Options{Root: t.TempDir(), Suite: "public", Out: filepath.Join(t.TempDir(), "report.json")})
	if err == nil {
		t.Fatal("public suite accepted")
	}
}

func TestComponentProofRejectsMissingZeroValuedFields(t *testing.T) {
	r := offlineReport(t)
	stage := &r.Passes[0].Stages[0]
	stage.Evidence = json.RawMessage(`{"passed":2,"inventory_sha256":"` + strings.Repeat("c", 64) + `"}`)
	stage.SHA256 = hash(stage.Evidence)
	if Validate(r) == nil {
		t.Fatal("missing skip evidence accepted")
	}
}

func TestLoopbackReportBindsEveryAuxiliarySource(t *testing.T) {
	r := offlineReport(t)
	r.Suite, r.Status, r.FailureStage = "loopback", "fail", loopbackStages[0]
	r.Passes = []Pass{{Number: 1, Stages: []Stage{{ID: loopbackStages[0], Status: "fail"}}}}
	lock, err := browserscenario.LoadQualificationBuildInputLock(r.Baseline)
	if err != nil {
		t.Fatal(err)
	}
	for _, component := range lock.Components {
		r.Sources = append(r.Sources, Source{Name: "udon_build_" + component.Name, Commit: component.Commit, SHA256: strings.Repeat("c", 64)})
	}
	if Validate(r) != nil {
		t.Fatal("valid failure evidence rejected")
	}
	r.Sources[len(r.Sources)-1].Commit = strings.Repeat("d", 40)
	if Validate(r) == nil {
		t.Fatal("substituted auxiliary source accepted")
	}
	r.Sources = r.Sources[:5]
	if Validate(r) == nil {
		t.Fatal("missing auxiliary sources accepted")
	}
}

func TestOutputCannotEnterWorkspaceThroughRootAlias(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "workspace", "openudon")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(parent, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	_, err := Run(context.Background(), Options{Root: alias, Suite: "offline", Out: filepath.Join(root, "report.json")})
	if err == nil || err.Error() != "report_must_be_outside_workspace" {
		t.Fatal("source alias bypassed output boundary")
	}
}
