package icot

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

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBuildBrowserAuthoringPlanIsValueFreeAndNonExecuting(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMBER_PASSWORD", "must-not-appear-in-plan")
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	plan, err := buildBrowserAuthoringPlan(browserAuthoringPlanInput{
		ExampleDir: filepath.Join(root, "example"), TargetURL: "https://example.test/member",
		Origins: []string{"https://example.test"}, ProfileID: "member-status",
		ActionHint: "read_status", LoginState: "not-required",
		PrivateRoot: privateRoot,
	})
	if err != nil {
		t.Fatalf("build browser authoring plan: %v", err)
	}
	if plan.Version != browserAuthoringPlanVersion || plan.Status != "ready" || plan.Resume == nil {
		t.Fatalf("plan = %#v", plan)
	}
	if plan.Authority.ICoTLaunchesBrowser || plan.Authority.ICoTRunsBrowsertools || plan.Authority.ICoTReadsCredentialEnvironment || plan.Authority.HandoffCarriesCredentialValues || plan.Authority.HandoffCarriesSessionState || !plan.Authority.ExternalRunMayLaunchBrowser {
		t.Fatalf("plan authority widened: %#v", plan.Authority)
	}
	if strings.Join(plan.Authority.ExternalCaptureAllowedNetworkMethods, ",") != "GET,HEAD" {
		t.Fatalf("network methods = %#v", plan.Authority.ExternalCaptureAllowedNetworkMethods)
	}
	stages := map[string]browserAuthoringStage{}
	for _, stage := range plan.Stages {
		stages[stage.ID] = stage
	}
	if !argvHasPair(stages["capture"].Argv, "--action-hint", "read_status") || !argvHasPair(stages["normalize"].Argv, "--action-hint", "read_status") {
		t.Fatalf("action hint was not preserved in capture/normalization: %#v", stages)
	}
	if !argvContains(stages["normalize"].Argv, "<reviewed-redaction-argv>") || len(stages["normalize"].Placeholders) != 1 {
		t.Fatalf("redaction argv is not explicitly declared: %#v", stages["normalize"])
	}
	if len(plan.Resume.Argv) != 5 || plan.Resume.Argv[0] != "icot" || plan.Resume.Argv[3] != "--browser-profile" || !strings.HasPrefix(plan.Resume.Argv[4], "member-status=") {
		t.Fatalf("resume argv = %#v", plan.Resume.Argv)
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("must-not-appear-in-plan")) || bytes.Contains(data, []byte("MEMBER_PASSWORD")) {
		t.Fatalf("plan copied credential environment: %s", data)
	}
}

func TestBuildBrowserAuthoringPlanFailsClosedForAuthenticatedCapture(t *testing.T) {
	root := t.TempDir()
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	plan, err := buildBrowserAuthoringPlan(browserAuthoringPlanInput{
		ExampleDir: filepath.Join(root, "example"), TargetURL: "https://members.example.test/dashboard",
		Origins:   []string{"https://members.example.test", "https://login.example.test"},
		ProfileID: "member-dashboard", ActionHint: "read_dashboard", LoginState: "required",
		PrivateRoot: privateRoot,
	})
	if err != nil {
		t.Fatalf("build login-required plan: %v", err)
	}
	if plan.Status != "needs_reviewed_profiles" || plan.Resume != nil || len(plan.Authority.ExternalCaptureAllowedNetworkMethods) != 0 {
		t.Fatalf("login-required plan = %#v", plan)
	}
	for _, stage := range plan.Stages {
		if len(stage.Argv) != 0 || stage.ID == "capture" {
			t.Fatalf("login-required plan exposed executable capture stage: %#v", stage)
		}
	}
	if len(plan.Diagnostics) != 2 || plan.Diagnostics[0].Code != "authenticated_capability_observation_not_supported" {
		t.Fatalf("login diagnostics = %#v", plan.Diagnostics)
	}
}

func TestBuildBrowserAuthoringPlanRejectsUnsafeAuthority(t *testing.T) {
	root := t.TempDir()
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	base := browserAuthoringPlanInput{
		ExampleDir: filepath.Join(root, "example"), TargetURL: "https://example.test/member",
		Origins: []string{"https://example.test"}, ProfileID: "member",
		ActionHint: "read_member", LoginState: "not-required", PrivateRoot: privateRoot,
	}
	tests := []struct {
		name   string
		mutate func(*browserAuthoringPlanInput)
		want   string
	}{
		{name: "query", mutate: func(v *browserAuthoringPlanInput) { v.TargetURL += "?token=value" }, want: "must not contain"},
		{name: "userinfo", mutate: func(v *browserAuthoringPlanInput) { v.TargetURL = "https://user:pass@example.test/member" }, want: "must not contain"},
		{name: "missing target origin", mutate: func(v *browserAuthoringPlanInput) { v.Origins = []string{"https://other.example.test"} }, want: "must include target origin"},
		{name: "package private root", mutate: func(v *browserAuthoringPlanInput) {
			v.PrivateRoot = filepath.Join(v.ExampleDir, ".private")
			if err := os.MkdirAll(v.PrivateRoot, 0o700); err != nil {
				t.Fatal(err)
			}
		}, want: "must be disjoint"},
		{name: "permissive private root", mutate: func(v *browserAuthoringPlanInput) {
			v.PrivateRoot = filepath.Join(root, "permissive")
			if err := os.Mkdir(v.PrivateRoot, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(v.PrivateRoot, 0o755); err != nil {
				t.Fatal(err)
			}
		}, want: "permissions"},
		{name: "symlink private root", mutate: func(v *browserAuthoringPlanInput) {
			v.PrivateRoot = filepath.Join(root, "private-link")
			if err := os.Symlink(privateRoot, v.PrivateRoot); err != nil {
				t.Fatal(err)
			}
		}, want: "non-symlink"},
		{name: "broad private root", mutate: func(v *browserAuthoringPlanInput) { v.PrivateRoot = string(filepath.Separator) }, want: "filesystem root"},
		{name: "reserved path delimiter", mutate: func(v *browserAuthoringPlanInput) { v.TargetURL = "https://example.test/<capture-id>" }, want: "reserved delimiter"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			input.Origins = append([]string(nil), base.Origins...)
			test.mutate(&input)
			if _, err := buildBrowserAuthoringPlan(input); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
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
		"browser-authoring", "plan", "--example", example,
		"--url", target.URL + "/member", "--origin", target.URL,
		"--profile-id", "member", "--action-hint", "read_member",
		"--login-state", "not-required", "--private-root", privateRoot, "--out", outPath,
	}
	var stdout, stderr bytes.Buffer
	if code := Main(args, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("plan code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	info, err := os.Stat(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("plan mode = %o", info.Mode().Perm())
	}
	if code := Main(args, strings.NewReader(""), &stdout, &stderr); code != 1 {
		t.Fatalf("plan overwrite code=%d, want 1", code)
	}
	outsideArgs := append([]string(nil), args...)
	outsideArgs[len(outsideArgs)-1] = filepath.Join(root, "outside.json")
	stderr.Reset()
	if code := Main(outsideArgs, strings.NewReader(""), &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "inside the private root") {
		t.Fatalf("outside output code=%d stderr=%s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("plan wrote project deliverable: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := Main([]string{"--example", example, "--browser-authoring-url", "not-a-url"}, strings.NewReader(""), &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "only with --agent") {
		t.Fatalf("interactive handoff code=%d stderr=%s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	session := elicitor.SessionFromIntent(&rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "member_status", Description: "Read member status from the reviewed website UI"},
		Steps:    []*rollout.Step{{Name: "read_member", Type: "browser"}},
	}, projectwizard.Answers{
		ProjectName: "Member status", Goal: "Read member status from the reviewed website UI",
		SideEffectScope: projectwizard.SideEffectReadOnly, Safety: "Read only", Fallback: "Stop if the reviewed UI is unavailable",
	})
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
	if code := Main(agentArgs, strings.NewReader(""), &stdout, &stderr); code != 0 {
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

func argvContains(argv []string, wanted string) bool {
	for _, value := range argv {
		if value == wanted {
			return true
		}
	}
	return false
}

func argvHasPair(argv []string, key, value string) bool {
	for index := 0; index+1 < len(argv); index++ {
		if argv[index] == key && argv[index+1] == value {
			return true
		}
	}
	return false
}
