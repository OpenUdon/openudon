package browsersystem

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func currentLoopbackFailureReport(t *testing.T, version string) *Report {
	t.Helper()
	lock, err := browserscenario.LoadCurrentCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	if version == CurrentV3Version {
		lock, err = browserscenario.LoadCurrentCompatibilityLockV3()
		if err != nil {
			t.Fatal(err)
		}
	}
	r := &Report{
		Toolchains: Toolchains{Go: lock.GoVersion, Node: "24.13.0"}, Version: version,
		Suite: "loopback", Status: "fail", FailureStage: loopbackStages[0], Baseline: lock,
		PlaywrightGo: "v0.6201.0", NetworkClaim: "application_request_allowlists_not_network_wide_containment",
	}
	for _, name := range []string{"openudon", "browsertools", "uws", "udon", "browserdriver"} {
		commit := strings.Repeat("a", 40)
		for _, component := range lock.Components {
			if component.Name == name {
				commit = component.Commit
			}
		}
		r.Sources = append(r.Sources, Source{Name: name, Commit: commit, SHA256: strings.Repeat("b", 64)})
	}
	build, err := browserscenario.LoadCurrentQualificationBuildInputLock(lock)
	if version == CurrentV3Version {
		build, err = browserscenario.LoadCurrentQualificationBuildInputLockV3(lock)
	}
	if err != nil {
		t.Fatal(err)
	}
	for _, component := range build.Components {
		r.Sources = append(r.Sources, Source{Name: "udon_build_" + component.Name, Commit: component.Commit, SHA256: strings.Repeat("c", 64)})
	}
	r.Passes = []Pass{{Number: 1, Stages: []Stage{{ID: loopbackStages[0], Status: "fail"}}}}
	return r
}

func TestCurrentV3ReportSelectsFrozenE21LockAndBuildClosure(t *testing.T) {
	r := currentLoopbackFailureReport(t, CurrentV3Version)
	if err := Validate(r); err != nil {
		t.Fatalf("current report rejected: %v", err)
	}
	r.Sources[3].Commit = strings.Repeat("d", 40)
	if Validate(r) == nil {
		t.Fatal("current report accepted a mismatched Udon revision")
	}
	r = currentLoopbackFailureReport(t, CurrentV3Version)
	r.Sources[len(r.Sources)-1].Commit = strings.Repeat("d", 40)
	if Validate(r) == nil {
		t.Fatal("current report accepted a mismatched auxiliary build source")
	}
	r = currentLoopbackFailureReport(t, CurrentV3Version)
	r.Version = Version
	if Validate(r) == nil {
		t.Fatal("historical v2 report accepted the current baseline")
	}
	r = currentLoopbackFailureReport(t, CurrentV3Version)
	r.Suite = "offline"
	if Validate(r) == nil {
		t.Fatal("current v3 report accepted an offline inventory")
	}
}

func TestCurrentV4ReportSelectsBrowser110LockAndBuildClosure(t *testing.T) {
	r := currentLoopbackFailureReport(t, CurrentVersion)
	if err := Validate(r); err != nil {
		t.Fatalf("current v4 report rejected: %v", err)
	}
	r.Sources[3].Commit = strings.Repeat("d", 40)
	if Validate(r) == nil {
		t.Fatal("current v4 report accepted a mismatched Udon revision")
	}
}

func TestCurrentV3NativeVerifierAcceptsOnlyV3ScenarioStages(t *testing.T) {
	lock, err := browserscenario.LoadCurrentCompatibilityLockV3()
	if err != nil {
		t.Fatal(err)
	}
	commits, versions := map[string]string{}, map[string]string{}
	for _, component := range lock.Components {
		commits[component.Name], versions[component.Name] = component.Commit, component.Version
	}
	root := strings.Repeat("a", 40)
	repositories := []browserscenario.RepositoryRevision{{Name: "openudon", Commit: root}}
	for _, name := range []string{"browsertools", "uws", "udon", "browserdriver"} {
		repositories = append(repositories, browserscenario.RepositoryRevision{Name: name, Commit: commits[name]})
	}
	journeys, err := browserscenario.LoadCurrentManifests(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	selected, err := browserscenario.SelectManifests(journeys, browserscenario.SuiteJourney, nil)
	if err != nil {
		t.Fatal(err)
	}
	results := make([]browserscenario.ScenarioResult, 0, len(selected))
	for _, manifest := range selected {
		result := browserscenario.ScenarioResult{ID: manifest.ID, Status: browserscenario.StatusPass, Attempts: 1, Detail: "ok", Phases: []browserscenario.PhaseResult{{ID: "fixture_ready", Status: browserscenario.StatusPass, Detail: "ok"}}, Assertions: []string{"author_session_v2"}}
		if required := map[string]string{"template-browser18": "browser18_template", "template-browser19": "browser19_template", "mixed-legacy-modern": "mixed_profile_session"}[manifest.ID]; required != "" {
			result.Assertions = append(result.Assertions, required, "udon_v10_replay")
			result.Phases = append(result.Phases, browserscenario.PhaseResult{ID: "udon_v10", Status: browserscenario.StatusPass, Detail: "ok"})
		}
		results = append(results, result)
	}
	component := browserscenario.NewReportForStack(browserscenario.SuiteJourney, browserscenario.StackCurrent, time.Now(), repositories, []browserscenario.DependencyRevision{
		{Module: "github.com/OpenUdon/browsertools", Version: versions["browsertools"]},
		{Module: "github.com/OpenUdon/uws", Version: versions["uws"]},
	}, results)
	component.Version = browserscenario.CurrentV3JourneyVersion
	stage := proof("journey_scenarios", component)
	if err := validateProof(stage, "loopback", browserscenario.StackCurrent, CurrentV3Version); err != nil {
		t.Fatalf("current v3 journey stage rejected: %v", err)
	}
	component.Version = browserscenario.M86CurrentJourneyVersion
	stage = proof("journey_scenarios", component)
	if validateProof(stage, "loopback", browserscenario.StackCurrent, CurrentV3Version) == nil {
		t.Fatal("native v3 verifier accepted an M86 v2 scenario stage")
	}
}

func TestCurrentV4NativeVerifierAcceptsOnlyV4CountScenarioStages(t *testing.T) {
	lock, err := browserscenario.LoadCurrentCompatibilityLockV4()
	if err != nil {
		t.Fatal(err)
	}
	commits, versions := map[string]string{}, map[string]string{}
	for _, component := range lock.Components {
		commits[component.Name], versions[component.Name] = component.Commit, component.Version
	}
	repositories := []browserscenario.RepositoryRevision{{Name: "openudon", Commit: strings.Repeat("a", 40)}}
	for _, name := range []string{"browsertools", "uws", "udon", "browserdriver"} {
		repositories = append(repositories, browserscenario.RepositoryRevision{Name: name, Commit: commits[name]})
	}
	journeys, err := browserscenario.LoadCurrentManifestsV4(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	selected, err := browserscenario.SelectManifests(journeys, browserscenario.SuiteJourney, nil)
	if err != nil {
		t.Fatal(err)
	}
	results := make([]browserscenario.ScenarioResult, 0, len(selected))
	for _, manifest := range selected {
		result := browserscenario.ScenarioResult{ID: manifest.ID, Status: browserscenario.StatusPass, Attempts: 1, Detail: "ok", Phases: []browserscenario.PhaseResult{{ID: "fixture_ready", Status: browserscenario.StatusPass, Detail: "ok"}}, Assertions: []string{"author_session_v2"}}
		switch manifest.ID {
		case "template-browser18":
			result.Assertions = append(result.Assertions, "browser18_template", "udon_v10_replay")
			result.Phases = append(result.Phases, browserscenario.PhaseResult{ID: "udon_v10", Status: browserscenario.StatusPass, Detail: "ok"})
		case "template-browser19":
			result.Assertions = append(result.Assertions, "browser19_template", "udon_v10_replay")
			result.Phases = append(result.Phases, browserscenario.PhaseResult{ID: "udon_v10", Status: browserscenario.StatusPass, Detail: "ok"})
		case "mixed-legacy-modern":
			result.Assertions = append(result.Assertions, "mixed_profile_session", "udon_v10_replay")
			result.Phases = append(result.Phases, browserscenario.PhaseResult{ID: "udon_v10", Status: browserscenario.StatusPass, Detail: "ok"})
		default:
			if strings.HasPrefix(manifest.ID, "campaign-count-browser110-") {
				result.Assertions = append(result.Assertions, "browser110_count", "udon_v11_replay")
				result.Phases = append(result.Phases, browserscenario.PhaseResult{ID: "udon_v11", Status: browserscenario.StatusPass, Detail: "ok"})
			}
		}
		results = append(results, result)
	}
	component := browserscenario.NewReportForStack(browserscenario.SuiteJourney, browserscenario.StackCurrent, time.Now(), repositories, []browserscenario.DependencyRevision{
		{Module: "github.com/OpenUdon/browsertools", Version: versions["browsertools"]},
		{Module: "github.com/OpenUdon/uws", Version: versions["uws"]},
	}, results)
	stage := proof("journey_scenarios", component)
	if err := validateProof(stage, "loopback", browserscenario.StackCurrent, CurrentVersion); err != nil {
		t.Fatalf("current v4 journey stage rejected: %v", err)
	}
	component.Version = browserscenario.CurrentV3JourneyVersion
	stage = proof("journey_scenarios", component)
	if validateProof(stage, "loopback", browserscenario.StackCurrent, CurrentVersion) == nil {
		t.Fatal("native v4 verifier accepted a v3 scenario stage")
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

func TestCurrentStackRejectsUnpinnedSourcesBeforeCreatingEvidence(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "openudon")
	udon := filepath.Join(parent, "udon")
	nodeModules := filepath.Join(parent, "node-modules")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(udon, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nodeModules, 0700); err != nil {
		t.Fatal(err)
	}
	outputRoot := t.TempDir()
	out := filepath.Join(outputRoot, "current.json")
	_, err := Run(context.Background(), Options{Root: root, UdonRepo: udon, BrowserdriverNodeModules: nodeModules, Stack: browserscenario.StackCurrent, Suite: "loopback", Out: out})
	if err == nil || err.Error() != "current_stack_source_state" {
		t.Fatalf("current source preflight error = %v", err)
	}
	for _, path := range []string{out, out + ".timing.jsonl"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("current preflight created evidence at %s: %v", path, err)
		}
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
