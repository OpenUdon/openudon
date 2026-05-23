package trustedrunner

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

func TestRunValidSandboxApprovalPassesDryRun(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("dry-run invoked runner")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !result.DryRun || result.Scope != "examples/support-email" || result.PackageSHA256 == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestRunLegacyHandoffVersionPassesDryRun(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{handoffVersion: legacyHandoffVersion})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)

	if _, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRunValidProductionApprovalPassesDryRun(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForProduction, now)

	if _, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierProduction,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
}

func TestRunProductionWithSandboxApprovalFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierProduction,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "not valid for production") {
		t.Fatalf("expected tier/state failure, got %v", err)
	}
}

func TestRunMissingApprovalFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "approvals", "missing.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "read approval") {
		t.Fatalf("expected missing approval failure, got %v", err)
	}
}

func TestRunExpiredApprovalFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	approval := readApprovalFile(t, approvalPath)
	approval.ExpiresAt = now().Add(-time.Hour).UTC().Format(time.RFC3339)
	writeApprovalFile(t, approvalPath, approval)

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired approval failure, got %v", err)
	}
}

func TestRunScopeMismatchFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	approval := readApprovalFile(t, approvalPath)
	approval.Scope = "examples/other"
	writeApprovalFile(t, approvalPath, approval)

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "scope") {
		t.Fatalf("expected scope mismatch failure, got %v", err)
	}
}

func TestRunDigestMismatchFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	mustWriteFile(t, filepath.Join(example, "project.md"), []byte("changed\n"))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "package_sha256") {
		t.Fatalf("expected digest mismatch failure, got %v", err)
	}
}

func TestRunFailedQualityReportFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{qualityStatus: "fail"})
	now := fixedNow()
	approvalPath := filepath.Join(root, "approval.json")
	mustWriteFile(t, approvalPath, []byte(`{"version":"openudon.approval.v1"}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "stored quality") {
		t.Fatalf("expected failed quality failure, got %v", err)
	}

	root, example = writeFixture(t, fixtureOptions{})
	approvalPath = writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	_, err = Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess: func(context.Context, synthesize.Options) (*synthesize.QualityReport, error) {
			return &synthesize.QualityReport{Status: "fail"}, nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "current quality") {
		t.Fatalf("expected current quality failure, got %v", err)
	}
}

func TestRunMalformedHandoffManifestFails(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{malformedHandoff: true})
	approvalPath := filepath.Join(root, "approval.json")
	mustWriteFile(t, approvalPath, []byte(`{"version":"openudon.approval.v1"}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("expected malformed handoff failure, got %v", err)
	}
}

func TestRunUnsafeManifestFails(t *testing.T) {
	for name, opts := range map[string]fixtureOptions{
		"credential values": {valuesAllowed: true},
		"direct production": {directProduction: true},
	} {
		t.Run(name, func(t *testing.T) {
			root, example := writeFixture(t, opts)
			now := fixedNow()
			approvalPath := writeApprovalTemplateWithoutPolicyCheck(t, root, example, StateApprovedForSandbox, now)
			_, err := Run(context.Background(), Options{
				RepoRoot:     root,
				ExampleDir:   example,
				Tier:         TierSandbox,
				ApprovalPath: approvalPath,
				DryRun:       true,
				Now:          now,
				Assess:       passAssess,
			})
			if err == nil || !strings.Contains(err.Error(), "manifest") {
				t.Fatalf("expected unsafe manifest failure, got %v", err)
			}
		})
	}
}

func TestRunNonDryRunInvokesRunner(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	runnerPath := filepath.Join(root, "fake-runner")
	mustWriteFile(t, runnerPath, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(runnerPath, 0o755); err != nil {
		t.Fatal(err)
	}
	var gotName string
	var gotArgs []string

	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		RunnerPath:   runnerPath,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
		RunCommand: func(_ context.Context, name string, args ...string) error {
			gotName = name
			gotArgs = args
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotName != runnerPath {
		t.Fatalf("runner path = %q", gotName)
	}
	if len(gotArgs) != 2 || gotArgs[0] != "--config" || gotArgs[1] != result.RunConfigPath {
		t.Fatalf("runner args = %#v", gotArgs)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"version": "openudon.executor-run.v1"`) || !strings.Contains(string(data), `"workflow_path": "workflows/workflow.uws.yaml"`) {
		t.Fatalf("unexpected run config:\n%s", data)
	}
}

func TestRunRejectsUnsafeRunnerPathOverride(t *testing.T) {
	for _, tc := range []struct {
		name       string
		runnerPath string
		want       string
	}{
		{name: "relative", runnerPath: "fake-runner", want: "absolute path"},
		{name: "missing", runnerPath: filepath.Join(t.TempDir(), "missing-runner"), want: "executable file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, example := writeFixture(t, fixtureOptions{})
			now := fixedNow()
			approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
			_, err := Run(context.Background(), Options{
				RepoRoot:     root,
				ExampleDir:   example,
				Tier:         TierSandbox,
				ApprovalPath: approvalPath,
				RunnerPath:   tc.runnerPath,
				WorkDir:      filepath.Join(root, "work"),
				Now:          now,
				Assess:       passAssess,
				RunCommand: func(context.Context, string, ...string) error {
					t.Fatal("runner should not be invoked")
					return nil
				},
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestRunNonDryRunUsesDefaultGoRunner(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	fakeExecutor := filepath.Join(root, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("OPENUDON_EXECUTOR", fakeExecutor)

	var gotName string
	var gotArgs []string
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
		RunCommand: func(_ context.Context, name string, args ...string) error {
			gotName = name
			gotArgs = append([]string(nil), args...)
			return nil
		},
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if gotName != fakeExecutor {
		t.Fatalf("executor path = %q, want %q", gotName, fakeExecutor)
	}
	stagedWorkdir := argValue(t, gotArgs, "--workdir")
	if !strings.HasPrefix(stagedWorkdir, filepath.Join(root, "work")+string(os.PathSeparator)+"stage.") {
		t.Fatalf("executor workdir = %q, want fresh stage under work; args=%#v", stagedWorkdir, gotArgs)
	}
	if gotWorkflow := argValue(t, gotArgs, "--workflow"); gotWorkflow != filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml") {
		t.Fatalf("executor workflow = %q, want staged workflow under %q; args=%#v", gotWorkflow, stagedWorkdir, gotArgs)
	}
	if result.RunConfigPath == "" {
		t.Fatalf("missing run config path in result: %+v", result)
	}
}

func TestRunConfigIncludesNestedOpenAPIPaths(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"openapi/nested/support.yaml"}})
	if err := os.MkdirAll(filepath.Join(example, "openapi", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(example, "openapi", "nested", "support.yaml"), []byte("openapi: 3.0.0\ninfo: {title: Support, version: 1.0.0}\npaths: {}\n"))
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"openapi/nested/support.yaml"`) {
		t.Fatalf("run config missing nested OpenAPI path:\n%s", data)
	}
	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"expected/plan.json",
		"expected/quality.json",
		"expected/refinement.json",
		"expected/review.md",
		"expected/symphony-handoff.json",
		"openapi/nested/support.yaml",
		"project.md",
		"workflows/intent.hcl",
		"workflows/workflow.hcl",
		"workflows/workflow.uws.yaml",
	}
	if strings.Join(config.PackagePaths, "\n") != strings.Join(want, "\n") {
		t.Fatalf("package paths = %#v, want %#v", config.PackagePaths, want)
	}
}

func TestRunConfigIncludesAdvisorySecuritySidecarPackagePath(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{
		"google-discovery/gmail.json",
		"google-discovery/gmail.security.json",
	}})
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(`{"discoveryVersion":"v1"}`))
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.security.json"), []byte(`{"security_schemes":[]}`))
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	result, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		WorkDir:      filepath.Join(root, "work"),
		Now:          now,
		Assess:       passAssess,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var config RunConfig
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatal(err)
	}
	if !stringSliceContains(config.PackagePaths, "google-discovery/gmail.security.json") {
		t.Fatalf("package paths missing advisory security sidecar: %#v", config.PackagePaths)
	}
	if !stringSliceContains(config.OpenAPIPaths, "google-discovery/gmail.json") {
		t.Fatalf("API source paths missing source: %#v", config.OpenAPIPaths)
	}
	if stringSliceContains(config.OpenAPIPaths, "google-discovery/gmail.security.json") {
		t.Fatalf("API source paths included advisory security sidecar: %#v", config.OpenAPIPaths)
	}
}

func TestRunRejectsOpenAPIFileMissingFromHandoffInputs(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	if err := os.MkdirAll(filepath.Join(example, "openapi", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(example, "openapi", "nested", "support.yaml"), []byte("openapi: 3.0.0\ninfo: {title: Support, version: 1.0.0}\npaths: {}\n"))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "openapi/nested/support.yaml") {
		t.Fatalf("expected missing OpenAPI handoff input error, got %v", err)
	}
}

func TestRunRejectsAdvisorySecuritySidecarMissingFromHandoffInputs(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"google-discovery/gmail.json"}})
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(`{"discoveryVersion":"v1"}`))
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.security.json"), []byte(`{"security_schemes":[]}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "google-discovery/gmail.security.json") {
		t.Fatalf("expected missing advisory security sidecar handoff input error, got %v", err)
	}
}

func TestRunRejectsListedAdvisorySecuritySidecarMissingFromPackage(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{
		"google-discovery/gmail.json",
		"google-discovery/gmail.security.json",
	}})
	mustWriteFile(t, filepath.Join(example, "google-discovery", "gmail.json"), []byte(`{"discoveryVersion":"v1"}`))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "google-discovery/gmail.security.json") {
		t.Fatalf("expected missing listed advisory security sidecar error, got %v", err)
	}
}

func TestRunRejectsOpenAPISymlink(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"openapi/support.yaml"}})
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "outside.yaml")
	mustWriteFile(t, target, []byte("openapi: 3.0.0\n"))
	if err := os.Symlink(target, filepath.Join(example, "openapi", "support.yaml")); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected OpenAPI symlink error, got %v", err)
	}
}

func TestRunRejectsSymlinkedProjectBeforeApproval(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	projectPath := filepath.Join(example, "project.md")
	target := filepath.Join(root, "outside-project.md")
	mustWriteFile(t, target, []byte("# Outside\n"))
	if err := os.Remove(projectPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, projectPath); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected project symlink rejection, got %v", err)
	}
}

func TestRunRejectsSymlinkedWorkflowBeforeExecutorInvocation(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	workflowPath := filepath.Join(example, "workflows", "workflow.uws.yaml")
	target := filepath.Join(root, "outside-workflow.yaml")
	mustWriteFile(t, target, []byte("version: outside\n"))
	if err := os.Remove(workflowPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, workflowPath); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		RunnerPath:   filepath.Join(root, "fake-runner"),
		Now:          now,
		Assess:       passAssess,
		RunCommand: func(context.Context, string, ...string) error {
			t.Fatal("runner should not be invoked for symlinked workflow")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected workflow symlink rejection, got %v", err)
	}
}

func TestRunRejectsSymlinkedExampleDirBeforeApproval(t *testing.T) {
	root, realExample := writeFixture(t, fixtureOptions{})
	linkExample := filepath.Join(root, "examples", "support-email-link")
	if err := os.Symlink(realExample, linkExample); err != nil {
		t.Fatal(err)
	}

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   linkExample,
		Tier:         TierSandbox,
		ApprovalPath: filepath.Join(root, "missing-approval.json"),
		DryRun:       true,
		Now:          fixedNow(),
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "package root must not be a symlink") {
		t.Fatalf("expected package root symlink rejection, got %v", err)
	}
}

func TestRunPackageDigestChangesWhenOpenAPIChanges(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"openapi/support.yaml"}})
	if err := os.MkdirAll(filepath.Join(example, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	openAPIPath := filepath.Join(example, "openapi", "support.yaml")
	mustWriteFile(t, openAPIPath, []byte("openapi: 3.0.0\ninfo: {title: Support, version: 1.0.0}\npaths: {}\n"))
	now := fixedNow()
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, now)
	mustWriteFile(t, openAPIPath, []byte("openapi: 3.0.0\ninfo: {title: Changed, version: 1.0.0}\npaths: {}\n"))

	_, err := Run(context.Background(), Options{
		RepoRoot:     root,
		ExampleDir:   example,
		Tier:         TierSandbox,
		ApprovalPath: approvalPath,
		DryRun:       true,
		Now:          now,
		Assess:       passAssess,
	})
	if err == nil || !strings.Contains(err.Error(), "package_sha256") {
		t.Fatalf("expected package digest mismatch, got %v", err)
	}
}

func TestUdonRunnerStagesPackageAndUsesConfiguredWorkdir(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	for _, dir := range []string{
		filepath.Join(packageRoot, "workflows"),
		filepath.Join(packageRoot, "openapi", "nested"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	mustWriteFile(t, filepath.Join(packageRoot, "openapi", "nested", "support.yaml"), []byte("openapi: 3.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:        RunConfigVersion,
		Scope:          "examples/test",
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
		OpenAPIPaths:   []string{"openapi/nested/support.yaml"},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	capture := filepath.Join(tmp, "args.txt")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$CAPTURE_ARGS\"\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor, "CAPTURE_ARGS="+capture)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	stagedWorkdir := capturedArgValue(t, string(args), "--workdir")
	if stagedWorkdir == workdir || !strings.HasPrefix(stagedWorkdir, workdir+string(os.PathSeparator)+"stage.") {
		t.Fatalf("executor workdir = %q, want fresh stage under %q\nargs:\n%s", stagedWorkdir, workdir, args)
	}
	if gotWorkflow := capturedArgValue(t, string(args), "--workflow"); gotWorkflow != filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml") {
		t.Fatalf("executor workflow = %q, want staged workflow under %q\nargs:\n%s", gotWorkflow, stagedWorkdir, args)
	}
	if strings.Contains(string(args), "--workdir\n"+packageRoot) {
		t.Fatalf("executor args did not use staged workdir:\n%s", args)
	}
	for _, path := range []string{
		filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml"),
		filepath.Join(stagedWorkdir, "openapi", "nested", "support.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("staged path missing %s: %v", path, err)
		}
	}
	if _, err := os.Stat(filepath.Join(workdir, "workflows", "workflow.uws.yaml")); !os.IsNotExist(err) {
		t.Fatalf("workflow was staged in persistent workdir root, err=%v", err)
	}
}

func TestUdonRunnerRejectsSymlinkedPackageRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	realPackageRoot := filepath.Join(tmp, "real-package")
	linkPackageRoot := filepath.Join(tmp, "package-link")
	workdir := filepath.Join(tmp, "work")
	mustWriteFile(t, filepath.Join(realPackageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	if err := os.Symlink(realPackageRoot, linkPackageRoot); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    linkPackageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	capture := filepath.Join(tmp, "args.txt")
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$CAPTURE_ARGS\"\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor, "CAPTURE_ARGS="+capture)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "package root must not be a symlink") {
		t.Fatalf("expected package root symlink rejection, err=%v out=%s", err, out)
	}
	if _, statErr := os.Stat(capture); !os.IsNotExist(statErr) {
		t.Fatalf("executor was invoked for symlinked package root, stat err=%v", statErr)
	}
}

func TestUdonRunnerRejectsSymlinkedWorkflow(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(tmp, "outside-workflow.yaml")
	mustWriteFile(t, target, []byte("uws: outside\n"))
	if err := os.Symlink(target, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "workflow file must not be a symlink") {
		t.Fatalf("expected workflow symlink rejection, err=%v out=%s", err, out)
	}
}

func TestUdonRunnerRejectsSymlinkedWorkflowParent(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	realWorkflows := filepath.Join(tmp, "real-workflows")
	if err := os.MkdirAll(realWorkflows, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(realWorkflows, "workflow.uws.yaml"), []byte("uws: outside\n"))
	if err := os.MkdirAll(packageRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realWorkflows, filepath.Join(packageRoot, "workflows")); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "workflow file must not be a symlink") {
		t.Fatalf("expected workflow parent symlink rejection, err=%v out=%s", err, out)
	}
}

func TestUdonRunnerRejectsUnsafeWorkflowTypes(t *testing.T) {
	for _, tc := range []struct {
		name       string
		setup      func(t *testing.T, path string)
		wantOutput string
	}{
		{
			name: "directory",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if err := os.MkdirAll(path, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "workflow file must be a regular file",
		},
		{
			name: "non-regular",
			setup: func(t *testing.T, path string) {
				t.Helper()
				if runtime.GOOS == "windows" {
					t.Skip("fifo test requires Unix")
				}
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Skipf("fifo unavailable: %v", err)
				}
			},
			wantOutput: "workflow file must be a regular file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			workflowPath := filepath.Join(packageRoot, "workflows", "workflow.uws.yaml")
			tc.setup(t, workflowPath)
			configPath := filepath.Join(tmp, "run-config.json")
			data, err := json.Marshal(RunConfig{
				Version:        RunConfigVersion,
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   "workflows/workflow.uws.yaml",
				WorkflowFormat: "uws-yaml",
			})
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.wantOutput) {
				t.Fatalf("expected %q, err=%v out=%s", tc.wantOutput, err, out)
			}
		})
	}
}

func TestUdonRunnerRejectsUnsafeWorkflowPathFields(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		wantOutput string
	}{
		{
			name:       "control-character",
			path:       "workflows/workflow.uws.yaml\n2",
			wantOutput: "control characters",
		},
		{
			name:       "absolute-outside-package",
			path:       "",
			wantOutput: "workflow_path escapes package_root",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
			workflowPath := tc.path
			if tc.name == "absolute-outside-package" {
				outside := filepath.Join(tmp, "outside-workflow.yaml")
				mustWriteFile(t, outside, []byte("uws: outside\n"))
				workflowPath = outside
			}
			configPath := filepath.Join(tmp, "run-config.json")
			data, err := json.Marshal(RunConfig{
				Version:        RunConfigVersion,
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   workflowPath,
				WorkflowFormat: "uws-yaml",
			})
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.wantOutput) {
				t.Fatalf("expected %q, err=%v out=%s", tc.wantOutput, err, out)
			}
		})
	}
}

func TestUdonRunnerRejectsOpenAPIUnsafePaths(t *testing.T) {
	for _, tc := range []struct {
		name       string
		path       string
		setup      func(t *testing.T, packageRoot string)
		wantOutput string
	}{
		{
			name: "control-character",
			path: "openapi/support.yaml\n2",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				mustWriteFile(t, filepath.Join(packageRoot, "openapi", "support.yaml"), []byte("openapi: 3.0.0\n"))
			},
			wantOutput: "control characters",
		},
		{
			name:       "absolute-outside-package",
			path:       "",
			setup:      func(t *testing.T, packageRoot string) {},
			wantOutput: "openapi path escapes package_root",
		},
		{
			name: "symlink",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				target := filepath.Join(filepath.Dir(packageRoot), "outside.yaml")
				mustWriteFile(t, target, []byte("openapi: 3.0.0\n"))
				if err := os.MkdirAll(filepath.Join(packageRoot, "openapi"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, filepath.Join(packageRoot, "openapi", "support.yaml")); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "openapi file must not be a symlink",
		},
		{
			name: "symlinked-parent",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				realOpenAPI := filepath.Join(filepath.Dir(packageRoot), "real-openapi")
				if err := os.MkdirAll(realOpenAPI, 0o755); err != nil {
					t.Fatal(err)
				}
				mustWriteFile(t, filepath.Join(realOpenAPI, "support.yaml"), []byte("openapi: 3.0.0\n"))
				if err := os.Symlink(realOpenAPI, filepath.Join(packageRoot, "openapi")); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "openapi file must not be a symlink",
		},
		{
			name: "directory",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(packageRoot, "openapi", "support.yaml"), 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantOutput: "openapi file must be a regular file",
		},
		{
			name: "non-regular",
			path: "openapi/support.yaml",
			setup: func(t *testing.T, packageRoot string) {
				t.Helper()
				if runtime.GOOS == "windows" {
					t.Skip("fifo test requires Unix")
				}
				path := filepath.Join(packageRoot, "openapi", "support.yaml")
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := syscall.Mkfifo(path, 0o600); err != nil {
					t.Skipf("fifo unavailable: %v", err)
				}
			},
			wantOutput: "openapi file must be a regular file",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
			tc.setup(t, packageRoot)
			openAPIPath := tc.path
			if tc.name == "absolute-outside-package" {
				outside := filepath.Join(tmp, "outside.yaml")
				mustWriteFile(t, outside, []byte("openapi: 3.0.0\n"))
				openAPIPath = outside
			}
			configPath := filepath.Join(tmp, "run-config.json")
			data, err := json.Marshal(RunConfig{
				Version:        RunConfigVersion,
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   "workflows/workflow.uws.yaml",
				WorkflowFormat: "uws-yaml",
				OpenAPIPaths:   []string{openAPIPath},
			})
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR=/bin/true")
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), tc.wantOutput) {
				t.Fatalf("expected %q, err=%v out=%s", tc.wantOutput, err, out)
			}
		})
	}
}

func TestUdonRunnerAcceptsAbsolutePathsInsidePackageRoot(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	workflowPath := filepath.Join(packageRoot, "workflows", "workflow.uws.yaml")
	openAPIPath := filepath.Join(packageRoot, "openapi", "nested", "support.yaml")
	mustWriteFile(t, workflowPath, []byte("uws: 1.0.0\n"))
	mustWriteFile(t, openAPIPath, []byte("openapi: 3.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:        RunConfigVersion,
		Scope:          "examples/test",
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   workflowPath,
		WorkflowFormat: "uws-yaml",
		OpenAPIPaths:   []string{openAPIPath},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	capture := filepath.Join(tmp, "args.txt")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$CAPTURE_ARGS\"\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor, "CAPTURE_ARGS="+capture)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	stagedWorkdir := capturedArgValue(t, string(args), "--workdir")
	for _, path := range []string{
		filepath.Join(stagedWorkdir, "workflows", "workflow.uws.yaml"),
		filepath.Join(stagedWorkdir, "openapi", "nested", "support.yaml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("staged path missing %s: %v", path, err)
		}
	}
}

func TestUdonRunnerFreshStageHidesPersistentStaleFiles(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(workdir, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	mustWriteFile(t, filepath.Join(workdir, "openapi", "stale.yaml"), []byte("openapi: 3.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:        RunConfigVersion,
		Scope:          "examples/test",
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   "workflows/workflow.uws.yaml",
		WorkflowFormat: "uws-yaml",
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	capture := filepath.Join(tmp, "args.txt")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$CAPTURE_ARGS\"\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor, "CAPTURE_ARGS="+capture)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	stagedWorkdir := capturedArgValue(t, string(args), "--workdir")
	if _, err := os.Stat(filepath.Join(stagedWorkdir, "openapi", "stale.yaml")); !os.IsNotExist(err) {
		t.Fatalf("stale OpenAPI file visible in staged workdir, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(workdir, "openapi", "stale.yaml")); err != nil {
		t.Fatalf("persistent stale file should not be deleted: %v", err)
	}
}

func TestUdonRunnerCanInvokeDockerImage(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:            RunConfigVersion,
		Scope:              "examples/test",
		PackageRoot:        packageRoot,
		WorkDir:            workdir,
		WorkflowPath:       "workflows/workflow.uws.yaml",
		WorkflowFormat:     "uws-yaml",
		CredentialBindings: []string{"support-api.token"},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	capture := filepath.Join(tmp, "docker-args.txt")
	fakeDocker := filepath.Join(binDir, "docker")
	mustWriteFile(t, fakeDocker, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$CAPTURE_ARGS\"\n"))
	if err := os.Chmod(fakeDocker, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"OPENUDON_EXECUTOR=docker://udon:test",
		"CAPTURE_ARGS="+capture,
		"UDON_CREDENTIAL_SUPPORT_API_TOKEN=super-secret",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("udon-runner docker failed: %v\n%s", err, out)
	}
	args, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	mountPath := capturedArgValue(t, string(args), "-v")
	if !strings.HasSuffix(mountPath, ":/workspace") || !strings.HasPrefix(mountPath, workdir+string(os.PathSeparator)+"stage.") {
		t.Fatalf("docker mount = %q, want fresh stage under %q\nargs:\n%s", mountPath, workdir, args)
	}
	for _, want := range []string{"run\n", "--rm\n", "-e\nUDON_CREDENTIAL_SUPPORT_API_TOKEN\n", "udon:test\n", "--workdir\n/workspace\n", "--workflow\n/workspace/workflows/workflow.uws.yaml\n"} {
		if !strings.Contains(string(args), want) {
			t.Fatalf("docker args missing %q:\n%s", want, args)
		}
	}
	if strings.Contains(string(args), "super-secret") {
		t.Fatalf("docker args leaked credential value:\n%s", args)
	}
}

func TestUdonRunnerFailsWhenCredentialEnvMissing(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	if err := os.MkdirAll(filepath.Join(packageRoot, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	config := RunConfig{
		Version:            RunConfigVersion,
		Scope:              "examples/test",
		PackageRoot:        packageRoot,
		WorkDir:            workdir,
		WorkflowPath:       "workflows/workflow.uws.yaml",
		WorkflowFormat:     "uws-yaml",
		CredentialBindings: []string{"missing.plan.test"},
	}
	config = withRunnerPackageDigest(t, packageRoot, config)
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = nil
	for _, item := range os.Environ() {
		if strings.HasPrefix(item, "UDON_CREDENTIAL_MISSING_PLAN_TEST=") {
			continue
		}
		cmd.Env = append(cmd.Env, item)
	}
	cmd.Env = append(cmd.Env, "OPENUDON_EXECUTOR=/bin/true")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "UDON_CREDENTIAL_MISSING_PLAN_TEST") {
		t.Fatalf("expected missing credential env failure, err=%v out=%s", err, out)
	}
}

func TestUdonRunnerRejectsRelativeExecutorEnv(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  string
	}{
		{name: "openudon-executor", env: "OPENUDON_EXECUTOR=relative-udon"},
		{name: "openudon-udon-bin", env: "OPENUDON_UDON_BIN=relative-udon"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wd, err := os.Getwd()
			if err != nil {
				t.Fatal(err)
			}
			repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
			tmp := t.TempDir()
			packageRoot := filepath.Join(tmp, "package")
			workdir := filepath.Join(tmp, "work")
			mustWriteFile(t, filepath.Join(packageRoot, "workflows", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
			configPath := filepath.Join(tmp, "run-config.json")
			config := RunConfig{
				Version:        RunConfigVersion,
				Scope:          "examples/test",
				PackageRoot:    packageRoot,
				WorkDir:        workdir,
				WorkflowPath:   "workflows/workflow.uws.yaml",
				WorkflowFormat: "uws-yaml",
			}
			config = withRunnerPackageDigest(t, packageRoot, config)
			data, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, configPath, data)
			cmd := runnerCLICommand(t, repoRoot, configPath)
			cmd.Env = append(os.Environ(), tc.env)
			out, err := cmd.CombinedOutput()
			if err == nil || !strings.Contains(string(out), "must be an absolute path") {
				t.Fatalf("expected absolute-path rejection, err=%v out=%s", err, out)
			}
		})
	}
}

func TestUdonRunnerVerifiesStagedPackageDigestBeforeExecutor(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	tmp := t.TempDir()
	packageRoot := filepath.Join(tmp, "package")
	workdir := filepath.Join(tmp, "work")
	workflowRel := "workflows/workflow.uws.yaml"
	workflowPath := filepath.Join(packageRoot, filepath.FromSlash(workflowRel))
	mustWriteFile(t, workflowPath, []byte("uws: 1.0.0\n"))
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:    packageRoot,
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  []authoring.ReviewHandoffInput{{Path: workflowRel, Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, workflowPath, []byte("uws: changed\n"))
	configPath := filepath.Join(tmp, "run-config.json")
	data, err := json.Marshal(RunConfig{
		Version:        RunConfigVersion,
		PackageRoot:    packageRoot,
		WorkDir:        workdir,
		WorkflowPath:   workflowRel,
		WorkflowFormat: "uws-yaml",
		PackagePaths:   []string{workflowRel},
		PackageSHA256:  digest,
	})
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, configPath, data)
	capture := filepath.Join(tmp, "args.txt")
	fakeExecutor := filepath.Join(tmp, "fake-udon")
	mustWriteFile(t, fakeExecutor, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$CAPTURE_ARGS\"\n"))
	if err := os.Chmod(fakeExecutor, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := runnerCLICommand(t, repoRoot, configPath)
	cmd.Env = append(os.Environ(), "OPENUDON_EXECUTOR="+fakeExecutor, "CAPTURE_ARGS="+capture)
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "staged package_sha256") {
		t.Fatalf("expected staged digest mismatch, err=%v out=%s", err, out)
	}
	if _, statErr := os.Stat(capture); !os.IsNotExist(statErr) {
		t.Fatalf("executor was invoked despite staged digest mismatch, stat err=%v", statErr)
	}
}

func capturedArgValue(t *testing.T, args, flag string) string {
	t.Helper()
	lines := strings.Split(strings.TrimRight(args, "\n"), "\n")
	for i := 0; i < len(lines)-1; i++ {
		if lines[i] == flag {
			return lines[i+1]
		}
	}
	t.Fatalf("args missing %s:\n%s", flag, args)
	return ""
}

func withRunnerPackageDigest(t *testing.T, packageRoot string, config RunConfig) RunConfig {
	t.Helper()
	paths := []string{runnerPackageRel(t, packageRoot, config.WorkflowPath)}
	for _, path := range config.OpenAPIPaths {
		paths = append(paths, runnerPackageRel(t, packageRoot, path))
	}
	config.PackagePaths = paths
	inputs := make([]authoring.ReviewHandoffInput, 0, len(paths))
	for _, path := range paths {
		inputs = append(inputs, authoring.ReviewHandoffInput{Path: path, Required: true})
	}
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:    packageRoot,
		Scope:   config.Scope,
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  inputs,
	})
	if err != nil {
		t.Fatal(err)
	}
	config.PackageSHA256 = digest
	return config
}

func runnerPackageRel(t *testing.T, packageRoot, path string) string {
	t.Helper()
	if filepath.IsAbs(path) {
		rel, err := filepath.Rel(packageRoot, path)
		if err != nil {
			t.Fatal(err)
		}
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(filepath.Clean(path))
}

func argValue(t *testing.T, args []string, flag string) string {
	t.Helper()
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	t.Fatalf("args missing %s: %#v", flag, args)
	return ""
}

func runnerCLICommand(t *testing.T, repoRoot, configPath string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command("go", "run", "./cmd/udon-runner", "--config", configPath)
	cmd.Dir = repoRoot
	return cmd
}

type fixtureOptions struct {
	qualityStatus       string
	malformedHandoff    bool
	valuesAllowed       bool
	directProduction    bool
	handoffVersion      string
	extraRequiredInputs []string
}

func writeFixture(t *testing.T, opts fixtureOptions) (string, string) {
	t.Helper()
	root := t.TempDir()
	example := filepath.Join(root, "examples", "support-email")
	for _, dir := range []string{
		filepath.Join(example, "workflows"),
		filepath.Join(example, "expected"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	status := opts.qualityStatus
	if status == "" {
		status = "pass"
	}
	files := map[string][]byte{
		"project.md":                  []byte("# Project\n"),
		"workflows/intent.hcl":        []byte("intent {}\n"),
		"workflows/workflow.hcl":      []byte("workflow {}\n"),
		"workflows/workflow.uws.yaml": []byte("version: 1.0.0\n"),
		"expected/plan.json":          []byte("{}\n"),
		"expected/quality.json":       []byte(`{"status":"` + status + `"}` + "\n"),
		"expected/refinement.json":    []byte("{}\n"),
		"expected/review.md":          []byte("# Review\n"),
	}
	for rel, data := range files {
		mustWriteFile(t, filepath.Join(example, filepath.FromSlash(rel)), data)
	}
	if opts.malformedHandoff {
		mustWriteFile(t, filepath.Join(example, "expected", "symphony-handoff.json"), []byte("{"))
		return root, example
	}
	version := opts.handoffVersion
	if version == "" {
		version = SymphonyHandoffVersion
	}
	manifest := map[string]any{
		"version":         version,
		"generated_state": string(authoring.ReviewStateGenerated),
		"handoff_inputs": []map[string]any{
			{"path": "project.md", "required": true},
			{"path": "workflows/intent.hcl", "required": true},
			{"path": "workflows/workflow.hcl", "required": true},
			{"path": "workflows/workflow.uws.yaml", "required": true},
			{"path": "expected/plan.json", "required": true},
			{"path": "expected/quality.json", "required": true},
			{"path": "expected/refinement.json", "required": true},
			{"path": "expected/review.md", "required": true},
			{"path": "expected/symphony-handoff.json", "required": true},
		},
		"approval_states": authoring.DefaultReviewStateMachine(),
		"owner_split": map[string]any{
			"openudon": []string{"artifact validation"},
			"symphony": []string{"approval routing"},
		},
		"execution_policy": map[string]any{
			"direct_production_execution": opts.directProduction,
		},
		"credential_bindings": map[string]any{
			"values_allowed_in_artifacts": opts.valuesAllowed,
		},
	}
	inputs := manifest["handoff_inputs"].([]map[string]any)
	for _, path := range opts.extraRequiredInputs {
		inputs = append(inputs, map[string]any{"path": path, "required": true})
	}
	manifest["handoff_inputs"] = inputs
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(example, "expected", "symphony-handoff.json"), append(data, '\n'))
	return root, example
}

func writeApprovalTemplate(t *testing.T, root, example, state string, now func() time.Time) string {
	t.Helper()
	approval, err := ApprovalTemplate(context.Background(), TemplateOptions{
		RepoRoot:   root,
		ExampleDir: example,
		State:      state,
		Reviewer:   "Ada",
		Now:        now,
		Assess:     passAssess,
	})
	if err != nil {
		t.Fatalf("ApprovalTemplate returned error: %v", err)
	}
	path := filepath.Join(root, "approval.json")
	writeApprovalFile(t, path, approval)
	return path
}

func writeApprovalTemplateWithoutPolicyCheck(t *testing.T, root, example, state string, now func() time.Time) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(example, "expected", "symphony-handoff.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest handoffManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	p, err := resolvePaths(root, example)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := computePackageDigest(p, manifest)
	if err != nil {
		t.Fatal(err)
	}
	approval := Approval{
		Version:       ApprovalVersion,
		Scope:         "examples/support-email",
		State:         state,
		Reviewer:      "Ada",
		ApprovedAt:    now().UTC().Format(time.RFC3339),
		PackageSHA256: digest,
	}
	path := filepath.Join(root, "approval.json")
	writeApprovalFile(t, path, approval)
	return path
}

func passAssess(context.Context, synthesize.Options) (*synthesize.QualityReport, error) {
	return &synthesize.QualityReport{Status: "pass"}, nil
}

func fixedNow() func() time.Time {
	return func() time.Time {
		return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	}
}

func readApprovalFile(t *testing.T, path string) Approval {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var approval Approval
	if err := json.Unmarshal(data, &approval); err != nil {
		t.Fatal(err)
	}
	return approval
}

func writeApprovalFile(t *testing.T, path string, approval Approval) {
	t.Helper()
	data, err := json.MarshalIndent(approval, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, append(data, '\n'))
}

func mustWriteFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
