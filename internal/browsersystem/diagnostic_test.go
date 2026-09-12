package browsersystem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/browserscenario"
)

func TestDiagnosticChild(t *testing.T) {
	if os.Getenv("OPENUDON_DIAGNOSTIC_FIXTURE") != "1" {
		return
	}
	fmt.Fprintln(os.Stdout, "synthetic-private-test-assertion")
	fmt.Fprintln(os.Stderr, "synthetic-private-child-error")
	os.Exit(7)
}

func TestFailedCommandRetainsPrivateDiagnosticOutsideReport(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	output, err := command(context.Background(), root, []string{executable, "-test.run=^TestDiagnosticChild$"}, []string{"OPENUDON_DIAGNOSTIC_FIXTURE=1"})
	if err == nil || err.Error() != "component_failed" || len(output) != 0 {
		t.Fatal("private command failure escaped through ordinary output/error")
	}
	out := filepath.Join(root, "report.json")
	var progress bytes.Buffer
	retainFailureDiagnostic(out, "ui_browser", err, &progress)
	path := out + ".diagnostic.json"
	data, readErr := os.ReadFile(path)
	info, statErr := os.Stat(path)
	if readErr != nil || statErr != nil || info.Mode().Perm() != 0600 {
		t.Fatal("private diagnostic was not retained with owner-only permissions")
	}
	var record map[string]any
	if json.Unmarshal(data, &record) != nil || record["reason"] != "subprocess" || record["stage"] != "ui_browser" || !strings.Contains(record["private_stdout"].(string), "synthetic-private-test-assertion") || !strings.Contains(record["private_stderr"].(string), "synthetic-private-child-error") {
		t.Fatal("failed child streams missing")
	}
	if strings.Contains(progress.String(), "synthetic-private") || !strings.Contains(progress.String(), filepath.Base(path)) {
		t.Fatal("raw child output leaked into progress")
	}
	if _, statErr := os.Lstat(out); !os.IsNotExist(statErr) {
		t.Fatal("private diagnostics substituted for report evidence")
	}
	retainFailureDiagnostic(out, "registration_ui", err, &progress)
	after, _ := os.ReadFile(path)
	if !bytes.Equal(data, after) || !strings.Contains(progress.String(), "diagnostic unavailable") {
		t.Fatal("earlier diagnostic overwritten")
	}
}

func TestPrivateDiagnosticBoundsAndSymlinkRefusal(t *testing.T) {
	out := filepath.Join(t.TempDir(), "report.json")
	failure := &commandFailure{reason: "output_limit", stdout: bytes.Repeat([]byte{0}, diagnosticStreamLimit+8), stderr: []byte("last-error"), truncated: true}
	retainFailureDiagnostic(out, "ui_browser", failure, nil)
	data, err := os.ReadFile(out + ".diagnostic.json")
	var record map[string]any
	if err != nil || len(data) > 16<<20 || json.Unmarshal(data, &record) != nil || record["truncated"] != true || len(record["private_stdout"].(string)) != diagnosticStreamLimit || record["private_stderr"] != "last-error" {
		t.Fatal("private stream bound or truncation marker invalid")
	}
	other := filepath.Join(t.TempDir(), "report.json")
	if err := os.Symlink(out+".diagnostic.json", other+".diagnostic.json"); err != nil {
		t.Fatal(err)
	}
	retainFailureDiagnostic(other, "registration_ui", failure, nil)
	after, _ := os.ReadFile(out + ".diagnostic.json")
	if !bytes.Equal(data, after) {
		t.Fatal("diagnostic followed final symlink")
	}
}

func TestFailedScenarioReportSurvivesPrivateDiagnostic(t *testing.T) {
	report := &browserscenario.Report{Status: browserscenario.StatusFail, Scenarios: []browserscenario.ScenarioResult{{ID: "synthetic-case", Status: browserscenario.StatusFail, Phases: []browserscenario.PhaseResult{{ID: "teardown", Status: browserscenario.StatusFail, Detail: "teardown_failed"}}}}}
	if scenarioFailure(report, nil) != nil {
		t.Fatal("successful evaluation acquired a failure")
	}
	cause := scenarioFailure(report, errors.New("synthetic-private-scenario-cause"))
	if cause.Error() != "component_failed" {
		t.Fatal("scenario detail leaked into ordinary error")
	}
	out := filepath.Join(t.TempDir(), "report.json")
	var progress bytes.Buffer
	retainFailureDiagnostic(out, "loopback_scenarios", cause, &progress)
	data, err := os.ReadFile(out + ".diagnostic.json")
	var diagnostic struct {
		Reason string `json:"reason"`
		Stdout string `json:"private_stdout"`
		Stderr string `json:"private_stderr"`
	}
	if err != nil || json.Unmarshal(data, &diagnostic) != nil || diagnostic.Reason != "scenario_evaluation" || diagnostic.Stderr != "synthetic-private-scenario-cause" {
		t.Fatal("scenario cause was lost")
	}
	var retained browserscenario.Report
	if json.Unmarshal([]byte(diagnostic.Stdout), &retained) != nil || len(retained.Scenarios) != 1 || retained.Scenarios[0].ID != "synthetic-case" || len(retained.Scenarios[0].Phases) != 1 || retained.Scenarios[0].Phases[0].Detail != "teardown_failed" {
		t.Fatal("failed scenario and phase were lost")
	}
	if strings.Contains(progress.String(), "synthetic-") {
		t.Fatal("private scenario detail leaked into progress")
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatal("failed diagnostic replaced aggregate evidence")
	}
}

func TestDiagnosticSnapshotWhileChildWriterRemainsActive(t *testing.T) {
	var buffer boundedBuffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 500 {
			_, _ = buffer.Write([]byte("synthetic-output"))
		}
	}()
	for range 500 {
		data, _ := buffer.snapshot()
		// A snapshot is caller-owned even while the child writer is active.
		if len(data) != 0 {
			data[0] = 'x'
		}
	}
	<-done
	data, exceeded := buffer.snapshot()
	if exceeded || bytes.Contains(data, []byte("x")) {
		t.Fatal("diagnostic snapshot aliases the active capture buffer")
	}
}
