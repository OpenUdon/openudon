package browserintegrationeval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browserscenario"
)

func TestM86CurrentLockSnapshotRetainsPublishedPins(t *testing.T) {
	lock, gates, err := contractForVersion(M86ReportVersion)
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Components) != 4 || len(gates) != 19 {
		t.Fatalf("M86 current contract has %d components and %d gates", len(lock.Components), len(gates))
	}
	for _, component := range lock.Components {
		if component.Name == "udon" && component.Commit != "080b8282e2b8f7ca7a9994b6d9f0e3d2891d853f" {
			t.Fatalf("M86 Udon pin changed: %s", component.Commit)
		}
	}
}

func TestCurrentReportVersionUsesRepairedLockAndBuildClosure(t *testing.T) {
	lock, gates, err := contractForVersion(ReportVersion)
	if err != nil {
		t.Fatal(err)
	}
	if len(lock.Components) != 4 || len(gates) != 19 {
		t.Fatalf("current contract has %d components and %d gates", len(lock.Components), len(gates))
	}
	scenarioLock, err := browserscenario.LoadCurrentCompatibilityLock()
	if err != nil || !reflect.DeepEqual(lock, scenarioLock) {
		t.Fatalf("integration and scenario current locks differ: %v", err)
	}
	for _, component := range lock.Components {
		if component.Name == "udon" && component.Commit != "6d32d4967469c579d35adcf47eaddb76a225dbae" {
			t.Fatalf("current Udon pin = %s", component.Commit)
		}
	}
	build, err := browserscenario.LoadCurrentQualificationBuildInputLock(lock)
	if err != nil || len(build.Components) != 14 {
		t.Fatalf("current build closure = %d components, err = %v", len(build.Components), err)
	}
}

func TestBrowserdriverNPMTestBuildsDisposablePinnedSourceCopy(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "browserdriver")
	modules := filepath.Join(root, "node-modules")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	versions := map[string]string{"@types/node": "24.5.2", "playwright": "1.62.1", "playwright-core": "1.62.1", "typescript": "5.9.2"}
	packages := make(map[string]map[string]string, len(versions))
	for name, version := range versions {
		packages["node_modules/"+name] = map[string]string{"version": version}
		packageRoot := filepath.Join(modules, filepath.FromSlash(name))
		if err := os.MkdirAll(packageRoot, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(packageRoot, "package.json"), []byte(`{"version":"`+version+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	lock, err := json.Marshal(map[string]any{"lockfileVersion": 3, "packages": packages})
	if err != nil {
		t.Fatal(err)
	}
	write := func(name, data string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(source, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("package.json", `{"name":"disposable-browserdriver-test","version":"1.0.0","scripts":{"test":"node test.js"}}`)
	write("package-lock.json", string(lock))
	write("test.js", `const fs = require("node:fs"); fs.mkdirSync("dist", {recursive:true}); fs.writeFileSync("dist/probe", "ok"); console.log("staged-npm-pass");`)
	for _, args := range [][]string{{"init", "-q"}, {"add", "package.json", "package-lock.json", "test.js"}, {"commit", "-q", "-m", "fixture"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = source
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=OpenUdon Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=OpenUdon Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	commitCommand := exec.Command("git", "rev-parse", "HEAD")
	commitCommand.Dir = source
	commitBytes, err := commitCommand.Output()
	if err != nil {
		t.Fatal(err)
	}
	command := Command{
		Repository: "browserdriver", Kind: "npm_test", Dir: source,
		Args: []string{"npm", "test"}, Timeout: time.Minute,
		ExpectedCommit: strings.TrimSpace(string(commitBytes)),
		Env:            map[string]string{"BROWSERDRIVER_NODE_MODULES": modules},
	}
	result := runIsolatedBrowserdriverNPMTest(context.Background(), command)
	if result.Err != nil || !strings.Contains(result.Stdout, "staged-npm-pass") {
		t.Fatalf("disposable npm test result = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(source, "dist")); !os.IsNotExist(err) {
		t.Fatalf("npm test mutated supplied Browserdriver source: %v", err)
	}
	status := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all")
	status.Dir = source
	if output, err := status.CombinedOutput(); err != nil || len(output) != 0 {
		t.Fatalf("source tree changed after disposable npm test: %v: %s", err, output)
	}
}

func TestRunWritesAndVerifiesValueFreeProviderFreeMatrix(t *testing.T) {
	repos := makeTestRepos(t)
	out := filepath.Join(t.TempDir(), "matrix", "report.json")
	runner := &fakeRunner{t: t}
	report, err := Run(context.Background(), testOptions(repos, out, runner.Run))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Status != StatusPass || report.Summary != (Summary{Total: 19, Passed: 16, Skipped: 3}) {
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
	if runner.calls["installed-headless-opt-in"] != 0 || runner.calls["headed-auth-opt-in"] != 0 || runner.calls["headed-author-opt-in"] != 0 {
		t.Fatalf("default run invoked opt-in browser gates: %#v", runner.calls)
	}
}

func TestLegacyReportRemainsVerifiableAgainstHistoricalLock(t *testing.T) {
	lock, specs, err := contractForVersion(LegacyReportVersion)
	if err != nil {
		t.Fatal(err)
	}
	report := &Report{
		Version: LegacyReportVersion, Status: StatusPass,
		GeneratedAt: "2026-09-13T00:00:00Z", Commit: "0123456789ab",
		Command: "openudon browser-integration-eval", RetentionClass: "release_evidence",
		SafeToArchive: true, ProviderFree: true,
		Repositories: []RepositoryRevision{{Name: "openudon", Commit: "0123456789ab"}},
	}
	for _, name := range []string{"browsertools", "uws", "udon", "browserdriver"} {
		for _, component := range lock.Components {
			if component.Name == name {
				report.Repositories = append(report.Repositories, RepositoryRevision{Name: name, Commit: component.Commit})
			}
		}
	}
	for _, spec := range specs {
		result := GateResult{ID: spec.ID, Repository: spec.Repository, Kind: spec.Kind,
			Command: append([]string(nil), spec.Args...), Assertions: append([]string(nil), spec.Assertions...)}
		if spec.OptIn != "" {
			result.Status, result.Detail = StatusSkipped, optInSkipDetail(spec.OptIn)
		} else {
			result.Status = StatusPass
			switch spec.Kind {
			case "go_test":
				result.EvidenceCount = len(spec.RequiredPasses)
				result.Detail = fmt.Sprintf("%d named provider-free test(s) passed", result.EvidenceCount)
			case "npm_test":
				result.EvidenceCount, result.Detail = 20, "20 Browserdriver test(s) passed"
			case "dependency_scan":
				result.EvidenceCount, result.Detail = 3, "3 dependency path(s) scanned"
			case "command":
				result.EvidenceCount = len(spec.RequiredLines)
				result.Detail = fmt.Sprintf("%d value-free command marker(s) observed", result.EvidenceCount)
			case "doctor":
				result.EvidenceCount, result.Detail = 1, doctorPassDetail(doctorEngine(spec), false)
			}
		}
		report.Results = append(report.Results, result)
	}
	report.Summary = summarize(report.Results)
	path := filepath.Join(t.TempDir(), "legacy.json")
	if err := Write(path, report); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyPassingFile(path); err != nil {
		t.Fatal(err)
	}
	report.Repositories[1].Commit = "9333a9f25dbb17551998a429e123e7a9ba976648"
	if err := Validate(report); err == nil {
		t.Fatal("historical report accepted a current-stack commit")
	}
}

func TestCurrentDependencyBoundarySeparatesEngineFromUIQualification(t *testing.T) {
	lock, err := loadCurrentCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range currentGates() {
		switch spec.ID {
		case "icot-dependency-boundary":
			if !equalStrings(spec.Args, []string{"go", "list", "-deps", "./internal/icot/engine"}) {
				t.Fatalf("engine scan widened: %#v", spec.Args)
			}
			if got := evaluateGate(spec, CommandOutput{Stdout: "github.com/mxschmitt/playwright-go\n"}, lock); got.Status != StatusFail {
				t.Fatal("engine accepted Playwright implementation dependency")
			}
		case "icot-ui-capture-boundary":
			if !equalStrings(spec.Args, []string{"go", "list", "-deps", "./internal/icot/ui"}) {
				t.Fatalf("UI scan narrowed: %#v", spec.Args)
			}
			if got := evaluateGate(spec, CommandOutput{Stdout: "github.com/mxschmitt/playwright-go\n"}, lock); got.Status != StatusPass {
				t.Fatal("explicit UI qualification adapter was rejected")
			}
			if got := evaluateGate(spec, CommandOutput{Stdout: "github.com/OpenUdon/browsertools/capture\n"}, lock); got.Status != StatusFail {
				t.Fatal("UI accepted Browsertools capture implementation dependency")
			}
		}
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
		{name: "installed components pass", doctorReady: true, wantPass: 19, wantCalls: 1},
		{name: "missing components skip", wantPass: 16, wantSkip: 3},
		{name: "named tests skip", doctorReady: true, skipOptIns: true, wantPass: 16, wantSkip: 3, wantCalls: 1},
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
			if runner.calls["installed-headless-opt-in"] != test.wantCalls || runner.calls["headed-auth-opt-in"] != test.wantCalls || runner.calls["headed-author-opt-in"] != test.wantCalls {
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
		{name: "dirty sibling", mutate: func(report *Report) { report.Repositories[1].Dirty = true }},
		{name: "dirty openudon", mutate: func(report *Report) { report.Repositories[0].Dirty = true }},
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
		if command.Repository == "openudon" {
			return CommandOutput{Stdout: "0123456789ab\n"}
		}
		lock, err := loadCurrentCompatibilityLock()
		if err != nil {
			runner.t.Fatal(err)
		}
		for _, component := range lock.Components {
			if component.Name == command.Repository {
				return CommandOutput{Stdout: component.Commit + "\n"}
			}
		}
		runner.t.Fatalf("missing locked revision for %s", command.Repository)
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
		var output strings.Builder
		for _, name := range spec.RequiredPasses {
			fmt.Fprintf(&output, "✔ %s (0.1ms)\n", name)
		}
		output.WriteString("ℹ tests 20\nℹ pass 20\nℹ fail 0\n")
		return CommandOutput{Stdout: output.String()}
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
	lock, err := loadCurrentCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	locked := map[string]browserscenario.LockedRevision{}
	for _, component := range lock.Components {
		locked[component.Name] = component
	}
	openudonMod := fmt.Sprintf("module example.test/openudon\n\nrequire (\n\t%s %s\n\t%s %s\n)\n",
		locked["browsertools"].Module, locked["browsertools"].Version,
		locked["uws"].Module, locked["uws"].Version)
	browsertoolsMod := fmt.Sprintf("module example.test/browsertools\n\nrequire %s %s\n", locked["uws"].Module, locked["uws"].Version)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(openudonMod), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repos["browsertools"], "go.mod"), []byte(browsertoolsMod), 0o600); err != nil {
		t.Fatal(err)
	}
	return repos
}

func testOptions(repos map[string]string, out string, runner Runner) Options {
	return Options{
		RepoRoot: repos["openudon"], BrowsertoolsRepo: repos["browsertools"],
		UWSRepo: repos["uws"], UdonRepo: repos["udon"], BrowserdriverRepo: repos["browserdriver"],
		OutPath: out, Runner: runner,
		Now:                 func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC) },
		validateBuildInputs: func(context.Context, string, string) error { return nil },
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
