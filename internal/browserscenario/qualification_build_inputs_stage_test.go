package browserscenario

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCloneQualificationSourcePreservesSuppliedCheckout(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	target := filepath.Join(root, "stage", "fixture")
	if err := os.MkdirAll(source, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module example.invalid/fixture\n\ngo 1.26.6\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, ".gitignore"), []byte("spider/tmp/\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "go.mod", ".gitignore"}, {"commit", "-q", "-m", "fixture"}} {
		command := exec.Command("git", args...)
		command.Dir = source
		command.Env = append(os.Environ(), "GIT_AUTHOR_NAME=OpenUdon Test", "GIT_AUTHOR_EMAIL=test@example.invalid", "GIT_COMMITTER_NAME=OpenUdon Test", "GIT_COMMITTER_EMAIL=test@example.invalid")
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	command := exec.Command("git", "rev-parse", "HEAD")
	command.Dir = source
	commit, err := command.Output()
	if err != nil {
		t.Fatal(err)
	}
	expected := strings.TrimSpace(string(commit))
	if err := cloneQualificationSource(context.Background(), source, target, expected); err != nil {
		t.Fatalf("clone exact source: %v", err)
	}
	generated := filepath.Join(target, "spider", "tmp", "TestQualification")
	if err := os.MkdirAll(generated, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(generated, "fixture"), []byte("test output"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(source, "spider", "tmp")); !os.IsNotExist(err) {
		t.Fatalf("isolated test output appeared in supplied source tree: %v", err)
	}
	for _, path := range []string{source, target} {
		status := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all")
		status.Dir = path
		if output, err := status.CombinedOutput(); err != nil || len(output) != 0 {
			t.Fatalf("source status at %s = %v: %s", path, err, output)
		}
		command := exec.Command("git", "rev-parse", "HEAD")
		command.Dir = path
		actual, err := command.Output()
		if err != nil || strings.TrimSpace(string(actual)) != expected {
			t.Fatalf("source commit at %s = %s, expected %s: %v", path, strings.TrimSpace(string(actual)), expected, err)
		}
	}
}
