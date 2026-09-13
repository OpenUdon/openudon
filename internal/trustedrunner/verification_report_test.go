package trustedrunner

import (
	"encoding/json"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	"os"
	"path/filepath"
	"testing"
)

func TestVerificationFailureReportRemainsBoundAndCannotProveSuccess(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0700); err != nil {
		t.Fatal(err)
	}
	report := udonreport.Report{Version: udonreport.VersionV4, Status: "error", StartedAt: "2026-09-13T12:00:00Z", FinishedAt: "2026-09-13T12:00:01Z", WorkflowPath: "workflow.uws.yaml", WorkflowFormat: "uws-yaml", WorkDir: stage, ErrorCode: "verification_not_ready", ErrorSummary: "Verification was not ready."}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(stage, "executor-report.json")
	if err := os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	config := RunConfig{RunID: "0123456789abcdef", WorkDir: root, ExecutorReportVersion: udonreport.VersionV4, Browser: &udonrunner.BrowserConfig{Protocol: "v6"}}
	prepared, err := publishExternalExecutorReportOutcome(config, udonrunner.Result{ExecutorReportPath: source}, false)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(prepared.ExecutorReportPath) != root {
		t.Fatal("failure report not published to evidence directory")
	}
	opts := runEvidenceOptions{Config: config, Prepared: prepared, Result: &RunResult{WorkDir: root}, Invoked: true, Mode: "external-runner", ExecutorStatus: "fail"}
	evidence, err := buildRunEvidenceExecutor(opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if evidence.ReportPath == "" || evidence.ReportSHA256 == "" || evidence.ReportSize != int64(len(data)) {
		t.Fatal("failure reference absent")
	}
	if err := verifyExecutorReport(root, evidence, config.Browser, false); err != nil {
		t.Fatal(err)
	}
	if verifyExecutorReport(root, evidence, config.Browser, true) == nil {
		t.Fatal("failure established success")
	}
	if _, err := publishExternalExecutorReport(config, udonrunner.Result{ExecutorReportPath: source}); err == nil {
		t.Fatal("nonzero outcome accepted as success")
	}
	if err := os.WriteFile(prepared.ExecutorReportPath, append(data, ' '), 0600); err != nil {
		t.Fatal(err)
	}
	if verifyExecutorReport(root, evidence, config.Browser, false) == nil {
		t.Fatal("tampered failure accepted")
	}
	opts.Prepared.ExecutorReportPath = filepath.Join(root, "absent-crash-report.json")
	absent, err := buildRunEvidenceExecutor(opts, nil)
	if err != nil || absent.ReportPath != "" {
		t.Fatal("absent crash report was fabricated or rejected")
	}
	if err := os.WriteFile(opts.Prepared.ExecutorReportPath, []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := buildRunEvidenceExecutor(opts, nil); err == nil {
		t.Fatal("malformed failure silently omitted")
	}
}
