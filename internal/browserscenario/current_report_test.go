package browserscenario

import (
	"path/filepath"
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

func TestCurrentV3LockUsesUdon6dAndSeparateFourteenInputClosure(t *testing.T) {
	lock, err := LoadCurrentCompatibilityLock()
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
	build, err := LoadCurrentQualificationBuildInputLock(lock)
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
