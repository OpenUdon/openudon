package authoringcli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpertSurfaceRefusesInteractiveAndWorkerInvocations(t *testing.T) {
	for _, command := range []string{"ui", "control", "browser-author", "browser-transaction", "__browsertools-worker", "--agent"} {
		t.Run(command, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if code := RunExpert([]string{command}, strings.NewReader(""), &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "unsupported expert command") {
				t.Fatalf("code=%d out=%s err=%s", code, out.String(), errOut.String())
			}
		})
	}
}

func TestExpertLintPreservesStructuredContract(t *testing.T) {
	var out, errOut bytes.Buffer
	if code := RunExpert([]string{"lint", "--example", filepath.Join("..", "..", "examples", "eval", "runtime-only-render"), "--json"}, strings.NewReader(""), &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	var report lintReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.Version != lintReportVersion || report.Status != statusPass || len(report.ProjectChecks) == 0 {
		t.Fatalf("report=%#v", report)
	}
}
