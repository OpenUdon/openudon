package browserintegrationeval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunWritesAndVerifiesValueFreeProviderFreeMatrix(t *testing.T) {
	repos := makeTestRepos(t)
	out := filepath.Join(t.TempDir(), "matrix", "report.json")
	runner := &fakeRunner{t: t}
	report, err := Run(context.Background(), testOptions(repos, out, runner.Run))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Status != StatusPass || report.Summary != (Summary{Total: 13, Passed: 11, Skipped: 2}) {
		t.Fatalf("report summary = %#v", report)
	}
	if report.BrowserLaunchedByDefault || report.TargetContactedByICoT || report.CredentialEnvironmentReadByICoT || report.PlanningDeliverablesWritten {
		t.Fatalf("authority widened: %#v", report)
	}
	verified, err := VerifyFile(out)
	if err != nil {
		t.Fatalf("VerifyFile: %v", err)
	}
	if verified.Status != StatusPass || verified.Commit != "0123456789ab" {
		t.Fatalf("verified report = %#v", verified)
	}
	if len(verified.Repositories) != 5 || verified.Repositories[0].Name != "openudon" || verified.Repositories[4].Name != "browserdriver" {
		t.Fatalf("repository provenance = %#v", verified.Repositories)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"member-password-value", "PASSWORD", "raw browser stdout", "cookie", "storage_state"} {
		if strings.Contains(string(data), forbidden) {
			t.Fatalf("report retained private subprocess output %q: %s", forbidden, data)
		}
	}
	if runner.calls["installed-headless-opt-in"] != 0 || runner.calls["headed-auth-opt-in"] != 0 {
		t.Fatalf("default run invoked opt-in browser gates: %#v", runner.calls)
	}
}

func TestRunOptInsPassOrHonestlySkipUnavailableComponents(t *testing.T) {
	repos := makeTestRepos(t)
	for _, test := range []struct {
		name        string
		doctorReady bool
		skipOptIns  bool
		wantPass    int
		wantSkip    int
		wantCalls   int
	}{
		{name: "installed components pass", doctorReady: true, wantPass: 13, wantCalls: 1},
		{name: "missing components skip", wantPass: 11, wantSkip: 2},
		{name: "named tests skip", doctorReady: true, skipOptIns: true, wantPass: 11, wantSkip: 2, wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &fakeRunner{t: t, doctorReady: test.doctorReady, skipOptIns: test.skipOptIns}
			opts := testOptions(repos, "", runner.Run)
			opts.InstalledEngines = true
			opts.HeadedAuth = true
			report, err := Run(context.Background(), opts)
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if report.Summary.Passed != test.wantPass || report.Summary.Skipped != test.wantSkip {
				t.Fatalf("summary = %#v", report.Summary)
			}
			if runner.calls["installed-headless-opt-in"] != test.wantCalls || runner.calls["headed-auth-opt-in"] != test.wantCalls {
				t.Fatalf("opt-in calls = %#v", runner.calls)
			}
		})
	}
}

func TestRunFailureReportDoesNotRetainPrivateDiagnostics(t *testing.T) {
	repos := makeTestRepos(t)
	out := filepath.Join(t.TempDir(), "report.json")
	runner := &fakeRunner{t: t, failID: "openudon-authoring"}
	report, err := Run(context.Background(), testOptions(repos, out, runner.Run))
	if err == nil || report == nil || report.Status != StatusFail || report.Summary.Failed != 1 {
		t.Fatalf("Run report=%#v err=%v", report, err)
	}
	verified, verifyErr := VerifyFile(out)
	if verifyErr != nil || verified.Status != StatusFail {
		t.Fatalf("VerifyFile report=%#v err=%v", verified, verifyErr)
	}
	if _, verifyErr := VerifyPassingFile(out); verifyErr == nil || !strings.Contains(verifyErr.Error(), "status is fail") {
		t.Fatalf("VerifyPassingFile error = %v", verifyErr)
	}
	data, readErr := os.ReadFile(out)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(data), "member-password-value") || strings.Contains(string(data), "raw browser stdout") {
		t.Fatalf("failure report retained subprocess diagnostics: %s", data)
	}
}

func TestVerifyFileRejectsTamperDuplicateUnknownSymlinkAndOversize(t *testing.T) {
	repos := makeTestRepos(t)
	out := filepath.Join(t.TempDir(), "report.json")
	if _, err := Run(context.Background(), testOptions(repos, out, (&fakeRunner{t: t}).Run)); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(out, append(append([]byte(nil), original...), ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFile(out); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("tamper error = %v", err)
	}

	for _, test := range []struct {
		name   string
		mutate func([]byte) []byte
		want   string
	}{
		{name: "duplicate", mutate: func(data []byte) []byte {
			return []byte(strings.Replace(string(data), `"version": "`+ReportVersion+`"`, `"version": "`+ReportVersion+`", "version": "`+ReportVersion+`"`, 1))
		}, want: "duplicate JSON field"},
		{name: "unknown", mutate: func(data []byte) []byte {
			return []byte(strings.Replace(string(data), "{\n", "{\n  \"page_value\": \"private\",\n", 1))
		}, want: "unknown field"},
		{name: "missing false authority", mutate: func(data []byte) []byte {
			return []byte(strings.Replace(string(data), "  \"target_contacted_by_icot\": false,\n", "", 1))
		}, want: `requires non-null field "target_contacted_by_icot"`},
		{name: "missing clean state", mutate: func(data []byte) []byte {
			return []byte(strings.Replace(string(data), "      \"dirty\": false\n", "      \"dirty\": null\n", 1))
		}, want: `requires non-null field "dirty"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "report.json")
			writeReportAndDigest(t, path, test.mutate(original))
			if _, err := VerifyFile(path); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("VerifyFile error = %v, want %q", err, test.want)
			}
		})
	}

	realPath := filepath.Join(t.TempDir(), "real.json")
	writeReportAndDigest(t, realPath, original)
	symlink := filepath.Join(t.TempDir(), "report.json")
	if err := os.Symlink(realPath, symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFile(symlink); err == nil || !strings.Contains(err.Error(), "regular non-symlink") {
		t.Fatalf("symlink error = %v", err)
	}

	oversized := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(oversized, []byte(strings.Repeat("x", maxOutputBytes+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFile(oversized); err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("oversize error = %v", err)
	}
}

func TestValidateRejectsContractAndAuthorityDrift(t *testing.T) {
	repos := makeTestRepos(t)
	base, err := Run(context.Background(), testOptions(repos, "", (&fakeRunner{t: t}).Run))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*Report)
	}{
		{name: "result order", mutate: func(report *Report) { report.Results[0], report.Results[1] = report.Results[1], report.Results[0] }},
		{name: "authority", mutate: func(report *Report) { report.TargetContactedByICoT = true }},
		{name: "private detail", mutate: func(report *Report) { report.Results[0].Detail = "member-password-value" }},
		{name: "required skip", mutate: func(report *Report) {
			report.Results[0].Status = StatusSkipped
			report.Results[0].Detail = "not run"
			report.Summary = summarize(report.Results)
		}},
		{name: "named evidence count", mutate: func(report *Report) {
			report.Results[0].EvidenceCount--
			report.Results[0].Detail = fmt.Sprintf("%d named provider-free test(s) passed", report.Results[0].EvidenceCount)
		}},
		{name: "summary", mutate: func(report *Report) { report.Summary.Passed-- }},
		{name: "repository order", mutate: func(report *Report) {
			report.Repositories[0], report.Repositories[1] = report.Repositories[1], report.Repositories[0]
		}},
		{name: "repository commit", mutate: func(report *Report) { report.Repositories[2].Commit = "unknown" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := cloneReport(t, base)
			test.mutate(report)
			if err := Validate(report); err == nil {
				t.Fatalf("Validate accepted %#v", report)
			}
		})
	}
}

type fakeRunner struct {
	t           *testing.T
	failID      string
	doctorReady bool
	skipOptIns  bool
	calls       map[string]int
}

func (runner *fakeRunner) Run(_ context.Context, command Command) CommandOutput {
	runner.t.Helper()
	if runner.calls == nil {
		runner.calls = map[string]int{}
	}
	if len(command.Args) >= 2 && command.Args[0] == "git" && command.Args[1] == "rev-parse" {
		return CommandOutput{Stdout: "0123456789ab\n"}
	}
	if len(command.Args) >= 2 && command.Args[0] == "git" && command.Args[1] == "status" {
		return CommandOutput{}
	}
	spec, ok := gateForCommand(command)
	if !ok {
		runner.t.Fatalf("unexpected command in %s: %#v", command.Repository, command.Args)
	}
	runner.calls[spec.ID]++
	if spec.ID == runner.failID {
		return CommandOutput{Stdout: "raw browser stdout", Stderr: "PASSWORD=member-password-value", Err: errors.New("private failure")}
	}
	switch spec.Kind {
	case "go_test":
		var output strings.Builder
		verb := "PASS"
		if runner.skipOptIns && spec.OptIn != "" {
			verb = "SKIP"
		}
		for _, name := range spec.RequiredPasses {
			fmt.Fprintf(&output, "--- %s: %s (0.00s)\n", verb, name)
		}
		return CommandOutput{Stdout: output.String(), Stderr: "PASSWORD=member-password-value"}
	case "dependency_scan":
		return CommandOutput{Stdout: "runtime\nfmt\ngithub.com/OpenUdon/openudon/internal/icot\n"}
	case "command":
		return CommandOutput{Stdout: strings.Join(spec.RequiredLines, "\n") + "\n"}
	case "npm_test":
		return CommandOutput{Stdout: "ℹ tests 20\nℹ pass 20\nℹ fail 0\n"}
	case "doctor":
		engine := doctorEngine(spec)
		if runner.doctorReady {
			data := fmt.Sprintf(`{"version":"browsertools.playwright-doctor.v1","engine":%q,"playwright_go_version":"v0.6201.0","playwright_version":"1.62.1","driver_ready":true,"browser_ready":true,"browser_executable":%q,"capability_policy":[{"name":"isolated_browser_context","disposition":"adopted","reason":"ephemeral"}]}`, engine, "/pinned/"+engine)
			return CommandOutput{Stdout: data}
		}
		data := fmt.Sprintf(`{"version":"browsertools.playwright-doctor.v1","engine":%q,"playwright_go_version":"v0.6201.0","playwright_version":"1.62.1","driver_ready":false,"browser_ready":false,"capability_policy":[{"name":"isolated_browser_context","disposition":"adopted","reason":"ephemeral"}],"error":"driver unavailable"}`, engine)
		return CommandOutput{Stdout: data, Err: errors.New("exit status 1")}
	default:
		runner.t.Fatalf("unexpected gate kind %q", spec.Kind)
		return CommandOutput{}
	}
}

func TestEnvironmentWithOverridesReplacesRatherThanDuplicates(t *testing.T) {
	got := environmentWithOverrides([]string{"KEEP=one", "FLAG=old", "FLAG=older", "EMPTY"}, map[string]string{"FLAG": "new", "SECOND": "two"})
	want := []string{"KEEP=one", "EMPTY", "FLAG=new", "SECOND=two"}
	if !equalStrings(got, want) {
		t.Fatalf("environment = %#v, want %#v", got, want)
	}
}

func gateForCommand(command Command) (gate, bool) {
	for _, spec := range defaultGates() {
		if spec.Repository == command.Repository && equalStrings(spec.Args, command.Args) {
			return spec, true
		}
	}
	return gate{}, false
}

func makeTestRepos(t *testing.T) map[string]string {
	t.Helper()
	root := t.TempDir()
	repos := map[string]string{"openudon": root}
	for _, name := range []string{"browsertools", "uws", "udon", "browserdriver"} {
		path := filepath.Join(root, name)
		if err := os.Mkdir(path, 0o700); err != nil {
			t.Fatal(err)
		}
		repos[name] = path
	}
	return repos
}

func testOptions(repos map[string]string, out string, runner Runner) Options {
	return Options{
		RepoRoot: repos["openudon"], BrowsertoolsRepo: repos["browsertools"],
		UWSRepo: repos["uws"], UdonRepo: repos["udon"], BrowserdriverRepo: repos["browserdriver"],
		OutPath: out, Runner: runner,
		Now: func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) },
	}
}

func cloneReport(t *testing.T, report *Report) *Report {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var cloned Report
	if err := json.Unmarshal(data, &cloned); err != nil {
		t.Fatal(err)
	}
	return &cloned
}

func writeReportAndDigest(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	line := "sha256:" + hex.EncodeToString(sum[:]) + "  " + filepath.Base(path) + "\n"
	if err := os.WriteFile(path+".sha256", []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
}
