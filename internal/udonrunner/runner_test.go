package udonrunner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPackageRelativePathRejectsEscapesAndUnsafeText(t *testing.T) {
	root := t.TempDir()
	for _, tc := range []struct {
		name  string
		value string
		want  string
	}{
		{name: "parent escape", value: "../outside.yaml", want: "escapes package_root"},
		{name: "root itself", value: ".", want: "escapes package_root"},
		{name: "backslash", value: `workflows\workflow.uws.yaml`, want: "slash separators"},
		{name: "control", value: "workflows/\nworkflow.uws.yaml", want: "control characters"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := packageRelativePath(root, "workflow_path", tc.value)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestPackageRelativePathAcceptsAbsolutePathInsideRoot(t *testing.T) {
	root := t.TempDir()
	absolute := filepath.Join(root, "workflows", "workflow.uws.yaml")
	rel, gotAbs, err := packageRelativePath(root, "workflow_path", absolute)
	if err != nil {
		t.Fatalf("packageRelativePath returned error: %v", err)
	}
	if filepath.ToSlash(rel) != "workflows/workflow.uws.yaml" {
		t.Fatalf("rel = %q", rel)
	}
	if gotAbs != absolute {
		t.Fatalf("absolute = %q, want %q", gotAbs, absolute)
	}
}

func TestValidateRegularPackageFileRejectsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests require Unix-style symlink permissions")
	}
	root := t.TempDir()
	outside := t.TempDir()

	targetFile := filepath.Join(outside, "workflow.uws.yaml")
	mustWriteRunnerTestFile(t, targetFile, []byte("uws: 1.0.0\n"))
	workflowDir := filepath.Join(root, "workflows")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetFile, filepath.Join(workflowDir, "leaf.uws.yaml")); err != nil {
		t.Fatal(err)
	}
	if err := validateRegularPackageFile(root, filepath.Join("workflows", "leaf.uws.yaml"), filepath.Join(root, "workflows", "leaf.uws.yaml"), "workflow"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected leaf symlink error, got %v", err)
	}

	linkRoot := t.TempDir()
	realWorkflows := filepath.Join(outside, "real-workflows")
	mustWriteRunnerTestFile(t, filepath.Join(realWorkflows, "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	if err := os.Symlink(realWorkflows, filepath.Join(linkRoot, "workflows")); err != nil {
		t.Fatal(err)
	}
	if err := validateRegularPackageFile(linkRoot, filepath.Join("workflows", "workflow.uws.yaml"), filepath.Join(linkRoot, "workflows", "workflow.uws.yaml"), "workflow"); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected parent symlink error, got %v", err)
	}
}

func TestValidateRegularPackageFileRejectsNonRegularFile(t *testing.T) {
	root := t.TempDir()
	dirPath := filepath.Join(root, "workflows", "workflow.uws.yaml")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	err := validateRegularPackageFile(root, filepath.Join("workflows", "workflow.uws.yaml"), dirPath, "workflow")
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected regular-file error, got %v", err)
	}
}

func TestExecutorArgvPrecedenceAndValidation(t *testing.T) {
	root := t.TempDir()
	stage := filepath.Join(root, "stage")
	workflow := filepath.Join(stage, "workflows", "workflow.uws.yaml")
	executor := filepath.Join(root, "openudon-executor")
	udonBin := filepath.Join(root, "udon-bin")
	mustWriteExecutable(t, executor)
	mustWriteExecutable(t, udonBin)

	argv, err := executorArgv(root, stage, workflow, "uws-yaml", []string{"UDON_CREDENTIAL_API_KEY"}, map[string]string{
		"OPENUDON_UDON_IMAGE": "openudon/udon:test",
		"OPENUDON_EXECUTOR":   executor,
	})
	if err != nil {
		t.Fatalf("docker executorArgv returned error: %v", err)
	}
	if argv[0] != "docker" || !containsArg(argv, "openudon/udon:test") || !containsArg(argv, "UDON_CREDENTIAL_API_KEY") {
		t.Fatalf("unexpected docker argv: %#v", argv)
	}

	argv, err = executorArgv(root, stage, workflow, "uws-yaml", nil, map[string]string{
		"OPENUDON_EXECUTOR": executor,
		"OPENUDON_UDON_BIN": udonBin,
	})
	if err != nil {
		t.Fatalf("binary executorArgv returned error: %v", err)
	}
	if argv[0] != executor {
		t.Fatalf("executor precedence picked %q, want %q", argv[0], executor)
	}

	if _, err := executorArgv(root, stage, workflow, "uws-yaml", nil, map[string]string{"OPENUDON_EXECUTOR": "relative"}); err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("expected relative executor rejection, got %v", err)
	}
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", nil, map[string]string{"OPENUDON_EXECUTOR": filepath.Join(root, "missing")}); err == nil || !strings.Contains(err.Error(), "executable file") {
		t.Fatalf("expected missing executor rejection, got %v", err)
	}
	nonExecutable := filepath.Join(root, "not-executable")
	mustWriteRunnerTestFile(t, nonExecutable, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", nil, map[string]string{"OPENUDON_EXECUTOR": nonExecutable}); err == nil || !strings.Contains(err.Error(), "executable file") {
		t.Fatalf("expected non-executable rejection, got %v", err)
	}
}

func mustWriteExecutable(t *testing.T, path string) {
	t.Helper()
	mustWriteRunnerTestFile(t, path, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteRunnerTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
