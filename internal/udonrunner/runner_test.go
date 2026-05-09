package udonrunner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/authoring"
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
		"OPENUDON_EXECUTOR": "docker://openudon/udon:test",
	})
	if err != nil {
		t.Fatalf("canonical docker executorArgv returned error: %v", err)
	}
	if argv[0] != "docker" || !containsArg(argv, "openudon/udon:test") || !containsArg(argv, "UDON_CREDENTIAL_API_KEY") {
		t.Fatalf("unexpected docker argv: %#v", argv)
	}

	argv, err = executorArgv(root, stage, workflow, "uws-yaml", nil, map[string]string{
		"OPENUDON_UDON_IMAGE": "openudon/udon:test",
		"OPENUDON_EXECUTOR":   executor,
	})
	if err != nil {
		t.Fatalf("binary executorArgv returned error: %v", err)
	}
	if argv[0] != executor {
		t.Fatalf("OPENUDON_EXECUTOR should override compatibility aliases, got %#v", argv)
	}

	argv, err = executorArgv(root, stage, workflow, "uws-yaml", []string{"UDON_CREDENTIAL_API_KEY"}, map[string]string{
		"OPENUDON_UDON_IMAGE": "openudon/udon:test",
	})
	if err != nil {
		t.Fatalf("compat docker executorArgv returned error: %v", err)
	}
	if argv[0] != "docker" || !containsArg(argv, "openudon/udon:test") || !containsArg(argv, "UDON_CREDENTIAL_API_KEY") {
		t.Fatalf("unexpected compat docker argv: %#v", argv)
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
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", nil, map[string]string{"OPENUDON_EXECUTOR": "docker://"}); err == nil || !strings.Contains(err.Error(), "docker image") {
		t.Fatalf("expected empty docker image rejection, got %v", err)
	}
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", nil, map[string]string{"OPENUDON_EXECUTOR": "docker://bad image"}); err == nil || !strings.Contains(err.Error(), "whitespace") {
		t.Fatalf("expected whitespace docker image rejection, got %v", err)
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

func TestRunRejectsEmptyPackageSHA256(t *testing.T) {
	config := validRunnerConfig(t)
	config.PackageSHA256 = ""
	_, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("executor should not be invoked")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "package_sha256") {
		t.Fatalf("expected package_sha256 rejection, got %v", err)
	}
}

func TestRunRejectsEmptyPackagePaths(t *testing.T) {
	config := validRunnerConfig(t)
	config.PackagePaths = nil
	_, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("executor should not be invoked")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "package_paths") {
		t.Fatalf("expected package_paths rejection, got %v", err)
	}
}

func TestRunRejectsDirectProductionRun(t *testing.T) {
	config := Config{DirectProductionRun: true}
	_, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("executor should not be invoked")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "direct_production_run") {
		t.Fatalf("expected direct_production_run rejection, got %v", err)
	}
}

func TestRunAcceptsValidDigestCoveredConfigAndInvokesExecutor(t *testing.T) {
	config := validRunnerConfig(t)
	var gotName string
	var gotArgs []string
	result, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
		RunCommand: func(_ context.Context, name string, args ...string) error {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotName != "/bin/true" {
		t.Fatalf("executor name = %q, want /bin/true", gotName)
	}
	if result.StagePath == "" || result.WorkflowPath == "" {
		t.Fatalf("result missing staged paths: %#v", result)
	}
	if !containsArg(gotArgs, result.StagePath) || !containsArg(gotArgs, result.WorkflowPath) {
		t.Fatalf("executor args do not reference staged paths: %#v result=%#v", gotArgs, result)
	}
	for _, rel := range []string{"workflows/workflow.uws.yaml", "openapi/support.yaml"} {
		if _, err := os.Stat(filepath.Join(result.StagePath, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("staged file missing %s: %v", rel, err)
		}
	}
}

func validRunnerConfig(t *testing.T) Config {
	t.Helper()
	root := t.TempDir()
	workdir := filepath.Join(t.TempDir(), "work")
	workflowRel := "workflows/workflow.uws.yaml"
	openAPIRel := "openapi/support.yaml"
	mustWriteRunnerTestFile(t, filepath.Join(root, filepath.FromSlash(workflowRel)), []byte("uws: 1.0.0\n"))
	mustWriteRunnerTestFile(t, filepath.Join(root, filepath.FromSlash(openAPIRel)), []byte("openapi: 3.0.0\n"))
	scope := "examples/test"
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:    root,
		Scope:   scope,
		Version: "openudon.handoff-package-digest.v1",
		Inputs: []authoring.ReviewHandoffInput{
			{Path: workflowRel, Required: true},
			{Path: openAPIRel, Required: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return Config{
		Version:        RunConfigVersion,
		Scope:          scope,
		PackageRoot:    root,
		WorkDir:        workdir,
		WorkflowPath:   workflowRel,
		WorkflowFormat: "uws-yaml",
		OpenAPIPaths:   []string{openAPIRel},
		PackagePaths:   []string{workflowRel, openAPIRel},
		PackageSHA256:  digest,
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
