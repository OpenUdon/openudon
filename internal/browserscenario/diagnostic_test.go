package browserscenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/internal/icot"
)

func failedAuthoringReport(t *testing.T) *Report {
	t.Helper()
	report := sampleReport(t)
	scenario := ScenarioResult{ID: "outputs-sixteen", Attempts: 1, Phases: []PhaseResult{{ID: "fixture_ready", Status: StatusPass, Detail: "ok"}}}
	scenario = appendAuthoringFailure(scenario, errors.New("credential-token-page-canary"))
	if scenario.AuthoringDiagnostic.Phase != "unknown" || scenario.AuthoringDiagnostic.Code != "operation_failed" {
		t.Fatal("untyped error was not reduced to a closed fallback")
	}
	scenario.AuthoringDiagnostic = &icot.BrowserScenarioAuthorDiagnostic{Phase: "controller", Code: "worker_protocol"}
	report.Scenarios = cloneScenarioResults([]ScenarioResult{scenario})
	report.Status, report.Summary = StatusFail, Summary{Total: 1, Failed: 1}
	if err := ValidateReport(report); err != nil {
		t.Fatal(err)
	}
	return report
}

func TestAuthoringDiagnosticSurvivesCleanupWithoutChangingReportWire(t *testing.T) {
	report := failedAuthoringReport(t)
	original := report.Scenarios[0].AuthoringDiagnostic
	copy := cloneScenarioResults(report.Scenarios)
	original.Code = "worker_exit"
	if copy[0].AuthoringDiagnostic.Code != "worker_protocol" {
		t.Fatal("diagnostic snapshot aliases executor state")
	}
	report.Scenarios = copy
	data, err := AuthoringFailureDiagnostic(report)
	if err != nil || ValidateAuthoringFailureDiagnostic(data, report) != nil || !bytes.Contains(data, []byte("worker_protocol")) {
		t.Fatal("closed authoring cause was lost")
	}
	wire, err := json.Marshal(report)
	if err != nil || bytes.Contains(wire, []byte("worker_protocol")) || bytes.Contains(wire, []byte("AuthoringDiagnostic")) || bytes.Contains(data, []byte("canary")) {
		t.Fatal("private metadata changed report wire or leaked a private value")
	}
	var restored Report
	if json.Unmarshal(wire, &restored) != nil || ValidateAuthoringFailureDiagnostic(data, &restored) != nil {
		t.Fatal("diagnostic did not bind the preserved report after cleanup")
	}
	if restored.Status != StatusFail {
		t.Fatal("failure diagnostic established success")
	}
	if data, err := AuthoringFailureDiagnostic(sampleReport(t)); err != nil || len(data) != 0 {
		t.Fatal("success acquired a failure diagnostic")
	}
}

func TestAuthoringDiagnosticRejectsTamperingAndPrivateVocabulary(t *testing.T) {
	report := failedAuthoringReport(t)
	data, err := AuthoringFailureDiagnostic(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*authoringFailureDiagnostic){
		func(d *authoringFailureDiagnostic) { d.Version = "unsupported" },
		func(d *authoringFailureDiagnostic) { d.ReportSHA256 = "bad" },
		func(d *authoringFailureDiagnostic) { d.Failures[0].Scenario = "password-main" },
		func(d *authoringFailureDiagnostic) { d.Failures[0].Authoring.Code = "credential_canary" },
		func(d *authoringFailureDiagnostic) { d.Failures[0].Authoring.Phase = "page-canary" },
		func(d *authoringFailureDiagnostic) { d.Failures = append(d.Failures, d.Failures[0]) },
	} {
		var record authoringFailureDiagnostic
		if json.Unmarshal(data, &record) != nil {
			t.Fatal("fixture decode failed")
		}
		mutate(&record)
		changed, _ := json.Marshal(record)
		if ValidateAuthoringFailureDiagnostic(changed, report) == nil {
			t.Fatal("tampered diagnostic accepted")
		}
	}
	unknown := append([]byte(`{"unknown":true,`), data[1:]...)
	if ValidateAuthoringFailureDiagnostic(unknown, report) == nil || ValidateAuthoringFailureDiagnostic(data, sampleReport(t)) == nil {
		t.Fatal("unknown field or unrelated passing report accepted")
	}
}

func TestAuthoringDiagnosticCanonicalizesReportScenarioOrder(t *testing.T) {
	report := failedAuthoringReport(t)
	other := cloneScenarioResults(report.Scenarios)[0]
	other.ID = "password-main"
	report.Scenarios = append([]ScenarioResult{other}, report.Scenarios...)
	report.Summary = Summary{Total: 2, Failed: 2}
	data, err := AuthoringFailureDiagnostic(report)
	if err != nil || ValidateAuthoringFailureDiagnostic(data, report) != nil {
		t.Fatal("valid report ordering changed diagnostic admissibility")
	}
	var record authoringFailureDiagnostic
	if json.Unmarshal(data, &record) != nil || record.Failures[0].Scenario != "outputs-sixteen" {
		t.Fatal("diagnostic entries were not canonicalized")
	}
}

func TestAuthoringDiagnosticWritesPrivatelyAndRefusesOverwrite(t *testing.T) {
	report := failedAuthoringReport(t)
	path := filepath.Join(t.TempDir(), "report.json")
	if err := writeAuthoringFailureDiagnostic(path, report); err != nil {
		t.Fatal(err)
	}
	diagnosticPath := path + ".authoring-diagnostic.json"
	before, _ := os.ReadFile(diagnosticPath)
	info, err := os.Lstat(diagnosticPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("diagnostic is not private")
	}
	if writeAuthoringFailureDiagnostic(path, report) == nil {
		t.Fatal("existing diagnostic was overwritten")
	}
	other := filepath.Join(t.TempDir(), "other.json")
	if err := os.Symlink(diagnosticPath, other+".authoring-diagnostic.json"); err != nil {
		t.Fatal(err)
	}
	if writeAuthoringFailureDiagnostic(other, report) == nil {
		t.Fatal("final diagnostic symlink was followed")
	}
	after, _ := os.ReadFile(diagnosticPath)
	if !bytes.Equal(before, after) {
		t.Fatal("preserved diagnostic changed")
	}
}
