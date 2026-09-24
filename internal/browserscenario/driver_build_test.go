package browserscenario

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCloneBrowserdriverSourceUsesExactCommitAndLeavesSourceClean(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "clone")
	if err := os.Mkdir(source, 0700); err != nil {
		t.Fatal(err)
	}
	git := func(dir string, args ...string) string {
		t.Helper()
		command := exec.Command("git", args...)
		command.Dir = dir
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
		return string(output)
	}
	git(source, "init", "--quiet")
	git(source, "config", "user.name", "OpenUdon Test")
	git(source, "config", "user.email", "openudon-test@example.invalid")
	if err := os.WriteFile(filepath.Join(source, "index.ts"), []byte("export const answer = 42;\n"), 0600); err != nil {
		t.Fatal(err)
	}
	git(source, "add", "index.ts")
	git(source, "commit", "--quiet", "-m", "fixture")
	want := git(source, "rev-parse", "HEAD")
	if err := cloneBrowserdriverSource(context.Background(), source, target); err != nil {
		t.Fatalf("cloneBrowserdriverSource: %v", err)
	}
	if got := git(target, "rev-parse", "HEAD"); got != want {
		t.Fatalf("clone revision = %q, want %q", got, want)
	}
	if got, err := os.ReadFile(filepath.Join(target, "index.ts")); err != nil || string(got) != "export const answer = 42;\n" {
		t.Fatalf("cloned source = %q, %v", got, err)
	}
	if got := git(source, "status", "--porcelain", "--untracked-files=all"); got != "" {
		t.Fatalf("supplied source changed: %q", got)
	}
}

func TestValidateBrowserdriverNodeModulesBindsPinnedBuildAndRuntimeVersions(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "browserdriver")
	modules := filepath.Join(root, "supplied-modules")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(modules, 0700); err != nil {
		t.Fatal(err)
	}
	versions := map[string]string{
		"@types/node": "24.5.2", "playwright": "1.62.1",
		"playwright-core": "1.62.1", "typescript": "5.9.2",
	}
	lock := "{\"packages\":{"
	first := true
	for name, version := range versions {
		if !first {
			lock += ","
		}
		first = false
		lock += `"node_modules/` + name + `":{"version":"` + version + `"}`
		packageRoot := filepath.Join(modules, filepath.FromSlash(name))
		if err := os.MkdirAll(packageRoot, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(packageRoot, "package.json"), []byte(`{"version":"`+version+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	lock += "}}"
	if err := os.WriteFile(filepath.Join(source, "package-lock.json"), []byte(lock), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBrowserdriverNodeModules(source, modules); err != nil {
		t.Fatalf("locked module set rejected: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modules, "typescript", "package.json"), []byte(`{"version":"5.9.3"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBrowserdriverNodeModules(source, modules); err == nil {
		t.Fatal("mismatched TypeScript version accepted")
	}
	inside := filepath.Join(source, "node_modules", "typescript")
	if err := os.MkdirAll(inside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(inside, "package.json"), []byte(`{"version":"5.9.2"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(modules, "typescript", "package.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(modules, "typescript")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(inside, filepath.Join(modules, "typescript")); err != nil {
		t.Fatal(err)
	}
	if err := ValidateBrowserdriverNodeModules(source, modules); err == nil {
		t.Fatal("dependency symlink into source checkout accepted")
	}
}
