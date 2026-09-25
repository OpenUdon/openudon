package browserscenario

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestBrowser110CampaignCountFreshLoopbackSmoke is the focused affected-stage
// browser smoke. It uses only the test-owned topics fixture and clean local
// source heads containing the pinned Browserdriver/Udon implementation commits.
func TestBrowser110CampaignCountFreshLoopbackSmoke(t *testing.T) {
	if os.Getenv("OPENUDON_BROWSER110_SMOKE") != "1" {
		t.Skip("set OPENUDON_BROWSER110_SMOKE=1 to run the fresh Browser 1.10 loopback smoke")
	}
	sandbox := strings.TrimSpace(os.Getenv("CHROME_DEVEL_SANDBOX"))
	if sandbox == "" {
		for _, candidate := range []string{"/opt/google/chrome/chrome-sandbox", "/usr/lib/chromium/chrome-sandbox"} {
			if info, err := os.Stat(candidate); err == nil && info.Mode()&os.ModeSetuid != 0 {
				sandbox = candidate
				break
			}
		}
	}
	if sandbox != "" {
		info, err := os.Stat(sandbox)
		if err != nil || info.Mode()&os.ModeSetuid == 0 {
			t.Fatal("CHROME_DEVEL_SANDBOX must name an existing setuid helper")
		}
		t.Setenv("CHROME_DEVEL_SANDBOX", sandbox)
	}

	modules := strings.TrimSpace(os.Getenv("OPENUDON_BROWSER_SYSTEM_BROWSERDRIVER_NODE_MODULES"))
	if !filepath.IsAbs(modules) {
		t.Fatal("OPENUDON_BROWSER_SYSTEM_BROWSERDRIVER_NODE_MODULES must name a separate absolute node_modules directory")
	}
	filename, ok := sourceFilename(t)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	openUdonRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	suppliedRoot := filepath.Clean(filepath.Join(openUdonRoot, ".."))
	lock, err := LoadCurrentCompatibilityLockV4()
	if err != nil {
		t.Fatal(err)
	}
	buildInputs, err := LoadCurrentQualificationBuildInputLockV4(lock)
	if err != nil {
		t.Fatal(err)
	}
	workRoot := t.TempDir()
	lockedRoot := filepath.Join(workRoot, "locked-sources")
	if err := os.Mkdir(lockedRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var suppliedUdon, suppliedBrowserdriver string
	for _, component := range lock.Components {
		switch component.Name {
		case "udon":
			suppliedUdon = filepath.Join(suppliedRoot, "udon")
		case "browserdriver":
			suppliedBrowserdriver = filepath.Join(suppliedRoot, "browserdriver")
		}
	}
	udonSource := cloneLockedSource(t, suppliedUdon, filepath.Join(lockedRoot, "udon"), componentCommit(lock, "udon"))
	for _, input := range buildInputs.Components {
		source := filepath.Join(suppliedRoot, input.Name)
		cloneLockedSource(t, source, filepath.Join(lockedRoot, input.Name), input.Commit)
	}
	browserdriverSource := cloneLockedSource(t, suppliedBrowserdriver, filepath.Join(lockedRoot, "browserdriver"), componentCommit(lock, "browserdriver"))
	if err := validateQualificationBuildInputsForLock(context.Background(), udonSource, lock, buildInputs); err != nil {
		t.Fatalf("staged Udon source closure: %v", err)
	}
	if err := ValidateGoModulePins(openUdonRoot, filepath.Join(lockedRoot, "browsertools"), lock); err != nil {
		t.Fatalf("Browser 1.10 Go module pins: %v", err)
	}
	if err := ValidateBrowserdriverNodeModules(browserdriverSource, modules); err != nil {
		t.Fatalf("Browserdriver modules: %v", err)
	}

	goTool, err := exec.LookPath("go")
	if err != nil || !commandOutputContains(context.Background(), udonSource, []string{goTool, "version"}, "go"+lock.GoVersion) {
		t.Fatal("the pinned Go toolchain is unavailable")
	}
	nodeTool, err := exec.LookPath("node")
	if err != nil || !commandOutputContains(context.Background(), browserdriverSource, []string{nodeTool, "--version"}, "v"+lock.NodeVersion+".") {
		t.Fatal("the pinned Node toolchain is unavailable")
	}
	nodeTool, err = filepath.Abs(nodeTool)
	if err != nil {
		t.Fatal(err)
	}
	executorRoot := filepath.Join(workRoot, "executor")
	if err := os.Mkdir(executorRoot, 0700); err != nil {
		t.Fatal(err)
	}
	stagedBrowserdriver := filepath.Join(executorRoot, "browserdriver")
	if err := StageBrowserdriverWithNodeModules(context.Background(), browserdriverSource, modules, stagedBrowserdriver); err != nil {
		t.Fatalf("stage exact Browserdriver: %v", err)
	}
	udonBinary := filepath.Join(executorRoot, "udon")
	if !runSilent(context.Background(), buildDeadline, udonSource,
		[]string{goTool, "build", "-o", udonBinary, "./cmd/udon"}, qualificationGoBuildEnvironment()) {
		t.Fatal("build staged exact Udon M43 source failed")
	}

	manifests, err := LoadCurrentManifestsV4(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	var selected *Manifest
	for index := range manifests {
		if manifests[index].ID == "campaign-count-browser110-multiple" {
			selected = &manifests[index]
			break
		}
	}
	if selected == nil {
		t.Fatal("multiple-row Browser 1.10 smoke manifest is missing")
	}
	executor := &realExecutor{
		root: executorRoot, udon: udonBinary, node: nodeTool,
		driverEntry: filepath.Join(stagedBrowserdriver, "dist", "src", "index.js"),
	}
	t.Cleanup(func() {
		if err := executor.Close(); err != nil {
			t.Errorf("remove smoke runtime: %v", err)
		}
	})
	result := executor.executeModernJourney(context.Background(), *selected, Environment{Lock: lock, Now: time.Now().UTC()})
	if result.Status != StatusPass || result.Detail != "ok" || !journeyContainsString(result.Assertions, "browser110_count") || !journeyContainsString(result.Assertions, "udon_v11_replay") {
		t.Fatalf("Browser 1.10 loopback smoke result = %#v", result)
	}
}

func sourceFilename(t *testing.T) (string, bool) {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	return filename, ok
}

func smokeGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"--no-replace-objects"}, args...)...)
	command.Dir = root
	data, err := command.Output()
	if err != nil {
		t.Fatalf("git %v in %s: %v", args, root, err)
	}
	return strings.TrimSpace(string(data))
}

func cloneLockedSource(t *testing.T, source, target, commit string) string {
	t.Helper()
	if !filepath.IsAbs(source) || !filepath.IsAbs(target) || !commitPattern.MatchString(commit) {
		t.Fatal("locked source clone inputs are invalid")
	}
	clone := exec.Command("git", "--no-replace-objects", "clone", "--shared", "--no-checkout", "--quiet", source, target)
	if output, err := clone.CombinedOutput(); err != nil {
		t.Fatalf("clone locked source %s: %v: %s", source, err, strings.TrimSpace(string(output)))
	}
	checkout := exec.Command("git", "--no-replace-objects", "checkout", "--quiet", "--detach", commit)
	checkout.Dir = target
	if output, err := checkout.CombinedOutput(); err != nil {
		t.Fatalf("checkout locked source %s at %s: %v: %s", source, commit, err, strings.TrimSpace(string(output)))
	}
	if actual := smokeGit(t, target, "rev-parse", "HEAD"); actual != commit {
		t.Fatalf("locked source %s checked out %s, want %s", source, actual, commit)
	}
	if status := smokeGit(t, target, "status", "--porcelain=v1", "--ignored=matching", "--untracked-files=all", "--", "."); status != "" {
		t.Fatalf("locked source %s is dirty after checkout", source)
	}
	return target
}

func componentCommit(lock CompatibilityLock, name string) string {
	for _, component := range lock.Components {
		if component.Name == name {
			return component.Commit
		}
	}
	return ""
}
