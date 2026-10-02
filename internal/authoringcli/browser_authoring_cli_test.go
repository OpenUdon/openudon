package authoringcli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/OpenUdon/openudon/internal/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBrowserAuthoringPlanCLIAndAgentReportDoNotWriteDeliverables(t *testing.T) {
	var targetHits atomic.Int64
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		targetHits.Add(1)
	}))
	defer target.Close()
	t.Setenv("MEMBER_PASSWORD", "must-not-appear-in-cli-or-agent-report")

	root := t.TempDir()
	example := filepath.Join(root, "example")
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(privateRoot, "handoff.json")
	args := []string{
		"browser-plan", "--example", example,
		"--url", target.URL + "/member", "--origin", target.URL,
		"--profile-id", "member", "--action-hint", "read_member",
		"--login-state", "not-required", "--private-root", privateRoot, "--out", outPath,
	}
	var stdout, stderr bytes.Buffer
	if code := RunExpert(args, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("plan code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("plan mode = %o", info.Mode().Perm())
	}
	if code := RunExpert(args, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Fatalf("plan overwrite code=%d, want 1", code)
	}
	outsideArgs := append([]string(nil), args...)
	outsideArgs[len(outsideArgs)-1] = filepath.Join(root, "outside.json")
	stderr.Reset()
	if code := RunExpert(outsideArgs, strings.NewReader(""), &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "inside the private root") {
		t.Fatalf("outside output code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("plan wrote project deliverable: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := RunExpert([]string{"draft", "--print", "--example", example, "--browser-authoring-url", "not-a-url"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "only with --agent") {
		t.Fatalf("interactive handoff code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	session, err := elicitor.SessionFromIntent(&rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_status", Description: "Read member status from the reviewed website UI"},
		Steps:    []*rollout.Step{{Name: "read_member", Type: "browser"}},
	}, projectwizard.Answers{
		ProjectName: "Member status", Goal: "Read member status from the reviewed website UI",
		SideEffectScope: projectwizard.SideEffectReadOnly, Safety: "Read only", Fallback: "Stop if the reviewed UI is unavailable",
	})
	if err != nil {
		t.Fatal(err)
	}
	session.BrowserRoute = "browser"
	session.BrowserSession = "none"
	sessionPath := writeSessionJSON(t, root, session)
	agentArgs := []string{
		"--example", example, "--answers", sessionPath, "--agent", "--json",
		"--browser-authoring-url", target.URL + "/member",
		"--browser-authoring-origin", target.URL,
		"--browser-authoring-id", "member", "--browser-authoring-action", "read_member",
		"--browser-authoring-login", "not-required", "--browser-authoring-private-root", privateRoot,
	}
	if code := RunExpert(append([]string{"draft"}, agentArgs...), strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("agent code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	var report authorReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode agent report: %v\n%s", err, stdout.String())
	}
	if report.BrowserAuthoring == nil || report.BrowserAuthoring.Status != "ready" {
		t.Fatalf("agent browser handoff = %#v", report.BrowserAuthoring)
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("agent wrote project deliverable: %v", err)
	}
	if got := targetHits.Load(); got != 0 {
		t.Fatalf("planning contacted the browser target %d time(s)", got)
	}
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "must-not-appear-in-cli-or-agent-report") || strings.Contains(combined, "MEMBER_PASSWORD") {
		t.Fatalf("planning output copied credential environment metadata: %s", combined)
	}
}

func TestBrowserAuthoringSourceGapPreservesAPIFirstSelection(t *testing.T) {
	if browserAuthoringSourceGap([]elicitor.ReadinessIssue{{Code: "missing_operation", Severity: "blocking"}}) {
		t.Fatal("browser handoff was attached without a missing source")
	}
	if !browserAuthoringSourceGap([]elicitor.ReadinessIssue{{Code: "missing_api_doc", Severity: "blocking"}}) {
		t.Fatal("browser handoff was not attached at the missing-source boundary")
	}
}
