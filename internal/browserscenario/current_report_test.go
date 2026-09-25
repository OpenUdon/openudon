package browserscenario

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestM86CurrentLockSnapshotRetainsPublishedPins(t *testing.T) {
	lock, err := LoadCurrentCompatibilityLockV2()
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Components) != 4 {
		t.Fatalf("M86 current lock has %d components", len(lock.Components))
	}
	for _, component := range lock.Components {
		if component.Name == "udon" && component.Commit != "080b8282e2b8f7ca7a9994b6d9f0e3d2891d853f" {
			t.Fatalf("M86 Udon pin changed: %s", component.Commit)
		}
	}
}

func TestCurrentV3LockSnapshotUsesUdon6dAndSeparateFourteenInputClosure(t *testing.T) {
	lock, err := LoadCurrentCompatibilityLockV3()
	if err != nil {
		t.Fatal(err)
	}
	udon := ""
	for _, component := range lock.Components {
		if component.Name == "udon" {
			udon = component.Commit
		}
	}
	if udon != "6d32d4967469c579d35adcf47eaddb76a225dbae" {
		t.Fatalf("current Udon pin = %s", udon)
	}
	build, err := LoadCurrentQualificationBuildInputLockV3(lock)
	if err != nil || len(build.Components) != 14 {
		t.Fatalf("current build closure = %d components, err = %v", len(build.Components), err)
	}
	for _, component := range build.Components {
		if component.Name == "browsertools" && component.Commit != "9333a9f25dbb17551998a429e123e7a9ba976648" {
			t.Fatalf("current Browsertools build input = %s", component.Commit)
		}
		if component.Name == "uws" && component.Commit != "e9b6181be0abb7f683fdb624d4dba282a59991d1" {
			t.Fatalf("current UWS build input = %s", component.Commit)
		}
	}
}

func TestCurrentV4LockPinsPublishedBrowser110DependencyChain(t *testing.T) {
	lock, err := LoadCurrentCompatibilityLockV4()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"browserdriver": "1f0e0d8c3bf16861f72938fa456035f1226da547",
		"browsertools":  "3abe70efc03d9ccb97b8b30e5e86328f60a70c64",
		"udon":          "4266ac99610a6fe39e363c75068e8256bdd821f5",
		"uws":           "80ee9bfb24a688b5e875dadf9ecacdc65398f1ff",
	}
	for _, component := range lock.Components {
		if component.Commit != want[component.Name] {
			t.Fatalf("Browser 1.10 %s pin = %s, want %s", component.Name, component.Commit, want[component.Name])
		}
		if component.Name == "browsertools" && component.Version != "v0.0.0-20260925161530-3abe70efc03d" {
			t.Fatalf("Browsertools module version = %s", component.Version)
		}
		if component.Name == "uws" && component.Version != "v0.0.0-20260925154821-80ee9bfb24a6" {
			t.Fatalf("UWS module version = %s", component.Version)
		}
	}
	build, err := LoadCurrentQualificationBuildInputLockV4(lock)
	if err != nil || len(build.Components) != 14 {
		t.Fatalf("Browser 1.10 build closure = %d components, err = %v", len(build.Components), err)
	}
	for _, component := range build.Components {
		if component.Name == "browsertools" && component.Commit != want["browsertools"] {
			t.Fatalf("Browsertools build input = %s", component.Commit)
		}
		if component.Name == "uws" && component.Commit != want["uws"] {
			t.Fatalf("UWS build input = %s", component.Commit)
		}
	}
}

func TestCurrentSelectorAdvancesToV4WhileV3SnapshotRemainsVerifiable(t *testing.T) {
	current, err := LoadCurrentCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	v4, err := LoadCurrentCompatibilityLockV4()
	if err != nil || !reflect.DeepEqual(current, v4) {
		t.Fatalf("current selector differs from v4 lock: %v", err)
	}
	v3, err := LoadCurrentCompatibilityLockV3()
	if err != nil {
		t.Fatal(err)
	}
	v3Build, err := LoadCurrentQualificationBuildInputLockV3(v3)
	if err != nil || len(v3Build.Components) != 14 {
		t.Fatalf("frozen v3 build closure = %d components, err = %v", len(v3Build.Components), err)
	}
	currentBuild, err := LoadCurrentQualificationBuildInputLock(current)
	if err != nil || !reflect.DeepEqual(currentBuild, mustCurrentV4Build(t, current)) {
		t.Fatalf("current selector differs from v4 build closure: %v", err)
	}
	versions := map[string]string{}
	commits := map[string]string{}
	for _, component := range v3.Components {
		versions[component.Name], commits[component.Name] = component.Version, component.Commit
	}
	rootCommit := strings.Repeat("a", 40)
	legacy := NewReportForStack(SuiteLoopback, StackCurrent, time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC), []RepositoryRevision{
		{Name: "openudon", Commit: rootCommit}, {Name: "browsertools", Commit: commits["browsertools"]},
		{Name: "uws", Commit: commits["uws"]}, {Name: "udon", Commit: commits["udon"]},
		{Name: "browserdriver", Commit: commits["browserdriver"]},
	}, []DependencyRevision{
		{Module: "github.com/OpenUdon/browsertools", Version: versions["browsertools"]},
		{Module: "github.com/OpenUdon/uws", Version: versions["uws"]},
	}, []ScenarioResult{{ID: "password-main", Status: StatusPass, Attempts: 1, Detail: "ok", Phases: []PhaseResult{{ID: "fixture_ready", Status: StatusPass, Detail: "ok"}}, Assertions: []string{"author_session_v2"}}})
	legacy.Version = CurrentV3ReportVersion
	if err := ValidateReport(legacy); err != nil {
		t.Fatalf("retained v3 scenario report no longer verifies: %v", err)
	}
	legacy.Version = CurrentReportVersion
	if err := ValidateReport(legacy); err == nil {
		t.Fatal("v4 report accepted v3 source bindings")
	}
}

func mustCurrentV4Build(t *testing.T, lock CompatibilityLock) QualificationBuildInputLock {
	t.Helper()
	build, err := LoadCurrentQualificationBuildInputLockV4(lock)
	if err != nil {
		t.Fatal(err)
	}
	return build
}

func TestM86CurrentReportUsesFrozenLockAfterCurrentLockAdvances(t *testing.T) {
	lock, err := LoadCurrentCompatibilityLockV2()
	if err != nil {
		t.Fatal(err)
	}
	commits := map[string]string{}
	versions := map[string]string{}
	for _, component := range lock.Components {
		commits[component.Name], versions[component.Name] = component.Commit, component.Version
	}
	rootCommit := strings.Repeat("a", 40)
	report := NewReportForStack(SuiteLoopback, StackCurrent, time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC), []RepositoryRevision{
		{Name: "openudon", Commit: rootCommit}, {Name: "browsertools", Commit: commits["browsertools"]},
		{Name: "uws", Commit: commits["uws"]}, {Name: "udon", Commit: commits["udon"]},
		{Name: "browserdriver", Commit: commits["browserdriver"]},
	}, []DependencyRevision{
		{Module: "github.com/OpenUdon/browsertools", Version: versions["browsertools"]},
		{Module: "github.com/OpenUdon/uws", Version: versions["uws"]},
	}, []ScenarioResult{{ID: "password-main", Status: StatusPass, Attempts: 1, Detail: "ok", Phases: []PhaseResult{{ID: "fixture_ready", Status: StatusPass, Detail: "ok"}}, Assertions: []string{"author_session_v2"}}})
	report.Version = M86CurrentReportVersion
	if err := ValidateReport(report); err != nil {
		t.Fatalf("M86 current report no longer verifies: %v", err)
	}
	report.Version = M86CurrentJourneyVersion
	if err := ValidateReport(report); err == nil {
		t.Fatal("cross-suite M86 report version was accepted")
	}
}

func TestCurrentScenarioReportRequiresExactLockAndCompleteSuite(t *testing.T) {
	lock, err := LoadCurrentCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	commits := map[string]string{}
	versions := map[string]string{}
	for _, component := range lock.Components {
		commits[component.Name], versions[component.Name] = component.Commit, component.Version
	}
	rootCommit := strings.Repeat("a", 40)
	report := NewReportForStack(SuiteLoopback, StackCurrent, time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC), []RepositoryRevision{
		{Name: "openudon", Commit: rootCommit}, {Name: "browsertools", Commit: commits["browsertools"]},
		{Name: "uws", Commit: commits["uws"]}, {Name: "udon", Commit: commits["udon"]},
		{Name: "browserdriver", Commit: commits["browserdriver"]},
	}, []DependencyRevision{
		{Module: "github.com/OpenUdon/browsertools", Version: versions["browsertools"]},
		{Module: "github.com/OpenUdon/uws", Version: versions["uws"]},
	}, []ScenarioResult{{ID: "password-main", Status: StatusPass, Attempts: 1, Detail: "ok", Phases: []PhaseResult{{ID: "fixture_ready", Status: StatusPass, Detail: "ok"}}, Assertions: []string{"author_session_v2"}}})
	path := filepath.Join(t.TempDir(), "report.json")
	if err := WriteReport(path, report); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyReportFile(path, true); err == nil || !strings.Contains(err.Error(), "complete suite") {
		t.Fatalf("partial current report verified: %v", err)
	}
	report.Repositories[3].Commit = strings.Repeat("b", 40)
	if err := ValidateReport(report); err == nil {
		t.Fatal("current report accepted wrong Udon revision")
	}
	report.Repositories[3].Commit = commits["udon"]
	report.Dependencies[0].Version = "v0.0.0-wrong"
	if err := ValidateReport(report); err == nil {
		t.Fatal("current report accepted wrong module pin")
	}
}

func TestBrowser110CurrentReportRequiresVersionedCountEvidence(t *testing.T) {
	lock, err := LoadCurrentCompatibilityLockV4()
	if err != nil {
		t.Fatal(err)
	}
	commits, versions := map[string]string{}, map[string]string{}
	for _, component := range lock.Components {
		commits[component.Name], versions[component.Name] = component.Commit, component.Version
	}
	repositories := []RepositoryRevision{
		{Name: "openudon", Commit: strings.Repeat("a", 40)},
		{Name: "browsertools", Commit: commits["browsertools"]},
		{Name: "uws", Commit: commits["uws"]},
		{Name: "udon", Commit: commits["udon"]},
		{Name: "browserdriver", Commit: commits["browserdriver"]},
	}
	result := ScenarioResult{ID: "campaign-count-browser110-multiple", Status: StatusPass, Attempts: 1, Detail: "ok",
		Phases:     []PhaseResult{{ID: "fixture_ready", Status: StatusPass, Detail: "ok"}},
		Assertions: []string{"author_session_v2"}}
	report := NewReportForStack(SuiteJourney, StackCurrent, time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC), repositories, []DependencyRevision{
		{Module: "github.com/OpenUdon/browsertools", Version: versions["browsertools"]},
		{Module: "github.com/OpenUdon/uws", Version: versions["uws"]},
	}, []ScenarioResult{result})
	if err := ValidateReport(report); err == nil {
		t.Fatal("Browser 1.10 count report without v11 evidence was accepted")
	}
	report.Scenarios[0].Assertions = append(report.Scenarios[0].Assertions, "browser110_count", "udon_v11_replay")
	report.Scenarios[0].Phases = append(report.Scenarios[0].Phases, PhaseResult{ID: "udon_v11", Status: StatusPass, Detail: "ok"})
	if err := ValidateReport(report); err != nil {
		t.Fatalf("Browser 1.10 count evidence rejected: %v", err)
	}
	report.Scenarios[0].ID = "template-browser19"
	if err := ValidateReport(report); err == nil {
		t.Fatal("non-count Browser 1.9 scenario accepted Browser 1.10 count evidence")
	}
}
