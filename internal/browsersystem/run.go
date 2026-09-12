package browsersystem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/openudon/internal/browserscenario"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

type Options struct {
	Root     string
	Suite    string
	Out      string
	UdonRepo string
	Progress io.Writer
}

func environment(extra ...string) []string {
	var env []string
	for _, key := range []string{"CHROME_DEVEL_SANDBOX", "HOME", "PATH", "DISPLAY", "XAUTHORITY", "XDG_RUNTIME_DIR", "LANG", "LC_ALL"} {
		if value := os.Getenv(key); value != "" {
			env = append(env, key+"="+value)
		}
	}
	env = append(env, "GOWORK=off", "GOPROXY=off", "GOSUMDB=off", "GOENV=off", "GOTOOLCHAIN=local")
	return append(env, extra...)
}

type boundedBuffer struct {
	buffer   bytes.Buffer
	mu       sync.Mutex
	exceeded bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.buffer.Len()+len(p) > 32<<20 {
		b.exceeded = true
		return len(p), nil
	}
	return b.buffer.Write(p)
}
func (b *boundedBuffer) snapshot() ([]byte, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buffer.Bytes()...), b.exceeded
}
func command(ctx context.Context, root string, args, extra []string) ([]byte, error) {
	var output, diagnostic boundedBuffer
	err := processgroup.Run(ctx, 15*time.Minute, processgroup.Invocation{Args: args, Dir: root, Env: environment(extra...), Stdout: &output, Stderr: &diagnostic})
	stdout, outputExceeded := output.snapshot()
	stderr, diagnosticExceeded := diagnostic.snapshot()
	if err != nil || outputExceeded || diagnosticExceeded {
		reason := "subprocess"
		switch {
		case errors.Is(err, processgroup.ErrTerminationTimeout):
			reason = "teardown"
		case ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
			reason = "deadline_or_cancellation"
		case outputExceeded || diagnosticExceeded:
			reason = "output_limit"
		}
		return nil, &commandFailure{reason: reason, stdout: stdout, stderr: stderr, truncated: outputExceeded || diagnosticExceeded}
	}
	return stdout, nil
}
func toolchains(ctx context.Context, root string) (Toolchains, error) {
	goOutput, err := command(ctx, root, []string{"go", "version"}, nil)
	if err != nil {
		return Toolchains{}, err
	}
	parts := strings.Fields(string(goOutput))
	if len(parts) != 4 || parts[0] != "go" || parts[1] != "version" || !strings.HasPrefix(parts[2], "go") {
		return Toolchains{}, errors.New("runtime_baseline")
	}
	nodeOutput, err := command(ctx, root, []string{"node", "--version"}, nil)
	if err != nil {
		return Toolchains{}, err
	}
	return Toolchains{Go: strings.TrimPrefix(parts[2], "go"), Node: strings.TrimPrefix(strings.TrimSpace(string(nodeOutput)), "v")}, nil
}

func source(ctx context.Context, name, root string) (Source, error) {
	fail := errors.New("source_state")
	revision, err := command(ctx, root, []string{"git", "--no-replace-objects", "rev-parse", "HEAD"}, nil)
	if err != nil {
		return Source{}, fail
	}
	names, err := command(ctx, root, []string{"git", "ls-files", "-z", "--cached", "--others", "--exclude-standard"}, nil)
	if err != nil {
		return Source{}, fail
	}
	files := strings.Split(strings.TrimSuffix(string(names), "\x00"), "\x00")
	sort.Strings(files)
	var inventory bytes.Buffer
	previous := ""
	for _, name := range files {
		if name == previous {
			continue
		}
		previous = name
		if name == "" || filepath.IsAbs(name) || strings.HasPrefix(filepath.Clean(name), "../") {
			return Source{}, fail
		}
		path := filepath.Join(root, name)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			fmt.Fprintf(&inventory, "%s\x00deleted\x00", name)
			continue
		}
		if err != nil {
			return Source{}, fail
		}
		var data []byte
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return Source{}, fail
			}
			data = []byte(target)
		} else {
			if !info.Mode().IsRegular() || info.Size() > 64<<20 {
				return Source{}, fail
			}
			data, err = os.ReadFile(path)
			if err != nil {
				return Source{}, fail
			}
		}
		fmt.Fprintf(&inventory, "%s\x00%o\x00%s\x00", name, info.Mode(), hash(data))
	}
	return Source{Name: name, Commit: strings.TrimSpace(string(revision)), SHA256: hash(inventory.Bytes())}, nil
}
func sources(ctx context.Context, root, udonRoot string, includeBuild bool) ([]Source, error) {
	var result []Source
	for _, name := range []string{"openudon", "browsertools", "uws", "udon", "browserdriver"} {
		path := filepath.Join(filepath.Dir(root), name)
		if name == "openudon" {
			path = root
		}
		if name == "udon" {
			path = udonRoot
		}
		s, err := source(ctx, name, path)
		if err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	if includeBuild {
		baseline, err := browserscenario.LoadCompatibilityLock()
		if err != nil {
			return nil, err
		}
		lock, err := browserscenario.LoadQualificationBuildInputLock(baseline)
		if err != nil {
			return nil, err
		}
		for _, component := range lock.Components {
			value, err := source(ctx, "udon_build_"+component.Name, filepath.Join(filepath.Dir(udonRoot), component.Name))
			if err != nil || value.Commit != component.Commit {
				return nil, errors.New("build_input_source")
			}
			result = append(result, value)
		}
	}
	return result, nil
}
func goTests(ctx context.Context, root string, args, extra []string, noSkips bool) (Tests, error) {
	data, err := command(ctx, root, append([]string{"go", "test", "-json", "-count=1"}, args...), extra)
	if err != nil {
		return Tests{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var result Tests
	var names []string
	for {
		var event struct{ Action, Package, Test string }
		err := decoder.Decode(&event)
		if err == io.EOF {
			break
		}
		if err != nil {
			return Tests{}, errors.New("test_protocol")
		}
		if event.Action == "skip" {
			result.Skipped++
		}
		if event.Action == "fail" {
			return Tests{}, errors.New("test_failed")
		}
		if event.Action == "pass" && strings.HasPrefix(event.Test, "Test") && !strings.Contains(event.Test, "/") {
			result.Passed++
			names = append(names, event.Package+"/"+event.Test)
		}
	}
	sort.Strings(names)
	result.InventorySHA256 = hash([]byte(strings.Join(names, "\n")))
	if result.Passed == 0 || noSkips && result.Skipped != 0 {
		return result, errors.New("required_tests_missing")
	}
	return result, nil
}
func nodeTests(ctx context.Context, root string, live bool) (Tests, error) {
	temp, err := os.MkdirTemp("", "openudon-driver-tests-")
	if err != nil {
		return Tests{}, errors.New("driver_build")
	}
	defer os.RemoveAll(temp)
	staged := filepath.Join(temp, "driver")
	if err := browserscenario.StageBrowserdriver(ctx, root, staged); err != nil {
		return Tests{}, err
	}
	root = staged
	args := []string{"node", "--test", "--test-reporter=tap"}
	var extra []string
	if live {
		args = append(args, "dist/test/registration-live.test.js", "dist/test/registration-inputs-live.test.js")
		extra = []string{"BROWSERDRIVER_REGISTRATION_LIVE_TEST=1"}
	} else {
		entries, err := filepath.Glob(filepath.Join(root, "dist/test/*.test.js"))
		if err != nil {
			return Tests{}, err
		}
		args = append(args, entries...)
	}
	data, err := command(ctx, root, args, extra)
	if err != nil {
		return Tests{}, err
	}
	var result Tests
	var names []string
	failures := -1
	for _, line := range strings.Split(string(data), "\n") {
		switch {
		case strings.HasPrefix(line, "# pass "):
			fmt.Sscanf(line, "# pass %d", &result.Passed)
		case strings.HasPrefix(line, "# fail "):
			fmt.Sscanf(line, "# fail %d", &failures)
		case strings.HasPrefix(line, "# skipped "):
			fmt.Sscanf(line, "# skipped %d", &result.Skipped)
		case strings.HasPrefix(line, "# Subtest: "):
			names = append(names, line)
		}
	}
	sort.Strings(names)
	result.InventorySHA256 = hash([]byte(strings.Join(names, "\n")))
	if failures != 0 || result.Passed == 0 || live && result.Skipped != 0 {
		return Tests{}, errors.New("required_tests_missing")
	}
	return result, nil
}

func Run(ctx context.Context, o Options) (*Report, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Hour)
	defer cancel()
	if o.Suite != "offline" && o.Suite != "loopback" || o.Out == "" {
		return nil, errors.New("suite_and_output_required")
	}
	root, err := filepath.Abs(o.Root)
	if err == nil {
		root, err = filepath.EvalSymlinks(root)
	}
	if err != nil {
		return nil, errors.New("source_state")
	}
	// Reports must stay outside source trees so source binding remains immutable.
	out, err := filepath.Abs(o.Out)
	if err != nil {
		return nil, errors.New("output")
	}
	resolvedParent, resolveErr := filepath.EvalSymlinks(filepath.Dir(out))
	if resolveErr != nil {
		return nil, errors.New("output_parent")
	}
	out = filepath.Join(resolvedParent, filepath.Base(out))
	parent := filepath.Dir(root)
	rel, err := filepath.Rel(parent, out)
	if err != nil || rel != ".." && !strings.HasPrefix(rel, "../") {
		return nil, errors.New("report_must_be_outside_workspace")
	}
	baseline, err := browserscenario.LoadCompatibilityLock()
	if err != nil {
		return nil, err
	}
	udonRoot := o.UdonRepo
	if udonRoot == "" {
		udonRoot = filepath.Join(filepath.Dir(root), "udon")
	}
	udonRoot, err = filepath.Abs(udonRoot)
	if err == nil {
		udonRoot, err = filepath.EvalSymlinks(udonRoot)
	}
	if err != nil {
		return nil, errors.New("source_state")
	}
	rel, err = filepath.Rel(filepath.Dir(udonRoot), out)
	if err != nil || rel != ".." && !strings.HasPrefix(rel, "../") {
		return nil, errors.New("report_must_be_outside_workspace")
	}
	before, err := sources(ctx, root, udonRoot, o.Suite == "loopback")
	if err != nil {
		return nil, err
	}
	for _, component := range baseline.Components {
		for _, source := range before {
			if source.Name == component.Name && source.Commit != component.Commit {
				return nil, errors.New("compatibility_baseline")
			}
		}
	}
	if err := browserscenario.ValidateGoModulePins(root, filepath.Join(filepath.Dir(root), "browsertools"), baseline); err != nil {
		return nil, errors.New("module_baseline")
	}
	runtimes, err := toolchains(ctx, root)
	if err != nil || !validToolchains(runtimes, baseline) {
		return nil, errors.New("runtime_baseline")
	}
	report := &Report{Toolchains: runtimes, Version: Version, Suite: o.Suite, Status: "pass", FailureStage: "none", Sources: before, Baseline: baseline, PlaywrightGo: "v0.6201.0", NetworkClaim: "application_request_allowlists_not_network_wide_containment"}
	count := 1
	if o.Suite == "loopback" {
		count = 3
	}
	for pass := 1; pass <= count; pass++ {
		row := Pass{Number: pass}
		for _, id := range inventory(o.Suite) {
			if o.Progress != nil {
				fmt.Fprintf(o.Progress, "browser-system-eval: pass %d stage %s started\n", pass, id)
			}
			value, err := runStage(ctx, root, udonRoot, id)
			after, sourceErr := sources(ctx, root, udonRoot, o.Suite == "loopback")
			currentRuntimes, runtimeErr := toolchains(ctx, root)
			if err != nil || sourceErr != nil || runtimeErr != nil || currentRuntimes != runtimes || !reflect.DeepEqual(before, after) {
				retainFailureDiagnostic(out, id, err, o.Progress)
				row.Stages = append(row.Stages, Stage{ID: id, Status: "fail"})
				report.Passes = append(report.Passes, row)
				report.Status = "fail"
				report.FailureStage = id
				if writeErr := Write(out, report); writeErr != nil {
					return report, writeErr
				}
				return report, fmt.Errorf("browser system failed at %s", id)
			}
			row.Stages = append(row.Stages, proof(id, value))
			if o.Progress != nil {
				fmt.Fprintf(o.Progress, "browser-system-eval: pass %d stage %s passed\n", pass, id)
			}
		}
		report.Passes = append(report.Passes, row)
	}
	if err := Write(out, report); err != nil {
		return report, err
	}
	return report, nil
}
func runStage(ctx context.Context, root, udonRoot, id string) (any, error) {
	sibling := func(name string) string { return filepath.Join(filepath.Dir(root), name) }
	options := browserscenario.Options{RepoRoot: root, BrowsertoolsRepo: sibling("browsertools"), UWSRepo: sibling("uws"), UdonRepo: udonRoot, BrowserdriverRepo: sibling("browserdriver"), RequireReady: true}
	switch id {
	case "openudon_unit":
		return goTests(ctx, root, []string{"./..."}, nil, false)
	case "browsertools_unit":
		return goTests(ctx, sibling("browsertools"), []string{"./..."}, nil, false)
	case "driver_unit":
		return nodeTests(ctx, sibling("browserdriver"), false)
	case "application_lifecycle":
		return goTests(ctx, root, []string{"-race", "./internal/icot/ui", "./internal/icot/browserauthor", "./internal/processgroup"}, nil, false)
	case "ui_browser":
		return goTests(ctx, root, []string{"-tags=icot_ui_browser", "./internal/icot/ui", "-run", "^TestPhaseCBrowser", "-timeout=5m"}, []string{"OPENUDON_ICOT_UI_BROWSER_SANDBOX_REQUIRED=1"}, true)
	case "registration_ui":
		return goTests(ctx, root, []string{"-tags=browser_system_qualification", "./internal/icot/ui", "-run", "^TestBrowserSystemRealRegistrationUI$", "-timeout=6m"}, nil, true)
	case "supervised_control":
		return goTests(ctx, root, []string{"-tags=browser_system_qualification", "./internal/icot/ui", "-run", "^TestBrowserSystemSupervisedControl$", "-timeout=6m"}, nil, true)
	case "supervised_registration_package":
		return goTests(ctx, root, []string{"-tags=browser_system_qualification", "./internal/icot/ui", "-run", "^TestBrowserSystemSupervisedRegistrationPackage$", "-timeout=6m"}, nil, true)
	case "supervised_authenticated_package":
		return goTests(ctx, root, []string{"-tags=browser_system_qualification", "./internal/icot/ui", "-run", "^TestBrowserSystemSupervisedAuthenticatedPackage$", "-timeout=6m"}, nil, true)
	case "udon_browser_contract":
		return goTests(ctx, udonRoot, []string{"-race", "./pkg/browserdriver", "./pkg/uwsprofile", "./pkg/registrationinput", "./internal/sourceloader", "-skip", "TestPrivateFormLiveUIStartApplyAndSubmit"}, nil, true)
	case "udon_browser_cli":
		return goTests(ctx, udonRoot, []string{"-race", "./cmd/udon", "-run", "Browser|Registration|ReadPrivateLine|ExecutionReportRedactsDriverErrors"}, nil, true)
	case "registration_driver":
		return nodeTests(ctx, sibling("browserdriver"), true)
	case "build_inputs":
		lock, _ := browserscenario.LoadCompatibilityLock()
		if err := browserscenario.ValidateQualificationBuildInputs(ctx, udonRoot, lock); err != nil {
			return nil, err
		}
		return browserscenario.LoadQualificationBuildInputLock(lock)
	case "bap_bcp_transaction", "registration_ui_handoff":
		executable, err := os.Executable()
		if err != nil {
			return nil, errors.New("component_executable")
		}
		data, err := command(ctx, root, []string{executable, "browser-system-component", "--component", id, "--repo-root", root, "--udon-repo", udonRoot}, nil)
		if err != nil {
			return nil, err
		}
		if id == "bap_bcp_transaction" {
			var e browserscenario.BAPBCPQualificationEvidence
			if evidencefile.DecodeStrict(data, &e) != nil {
				return nil, errors.New("component_protocol")
			}
			return e, browserscenario.ValidateBAPBCPQualificationEvidence(e)
		}
		var e browserscenario.BRPQualificationEvidence
		if evidencefile.DecodeStrict(data, &e) != nil {
			return nil, errors.New("component_protocol")
		}
		return e, browserscenario.ValidateBRPQualificationEvidence(e)
	case "loopback_scenarios", "journey_scenarios":
		directory, err := os.MkdirTemp("", "browser-system-components-")
		if err != nil {
			return nil, errors.New("output")
		}
		defer os.RemoveAll(directory)
		options.OutPath = filepath.Join(directory, "report.json")
		options.Suite = browserscenario.SuiteLoopback
		if id == "journey_scenarios" {
			options.Suite = browserscenario.SuiteJourney
		}
		report, err := browserscenario.RunLocalQualification(ctx, options)
		return report, scenarioFailure(report, err)
	}
	return nil, errors.New("unknown_stage")
}

// RunComponent is a closed synthetic child mode. The aggregate parent owns
// its process tree, deadline, protocol stream and verified teardown.
func RunComponent(ctx context.Context, root, udonRoot, id string) (any, error) {
	options := browserscenario.Options{RepoRoot: root, UdonRepo: udonRoot, RequireReady: true}
	switch id {
	case "bap_bcp_transaction":
		return browserscenario.RunBAPBCPQualification(ctx, options)
	case "registration_ui_handoff":
		return browserscenario.RunBRPQualification(ctx, options)
	}
	return nil, errors.New("unknown_component")
}
