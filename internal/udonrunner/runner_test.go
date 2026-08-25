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

func TestPrepareHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Prepare(ctx, Config{}, Options{}); err != context.Canceled {
		t.Fatalf("Prepare canceled-context error = %v", err)
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
	reportPath := filepath.Join(stage, "executor-report.json")
	executor := filepath.Join(root, "openudon-executor")
	udonBin := filepath.Join(root, "udon-bin")
	mustWriteExecutable(t, executor)
	mustWriteExecutable(t, udonBin)

	argv, err := executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, []string{"UDON_CREDENTIAL_API_KEY"}, map[string]string{
		"OPENUDON_EXECUTOR": "docker://openudon/udon:test",
	})
	if err != nil {
		t.Fatalf("canonical docker executorArgv returned error: %v", err)
	}
	if argv[0] != "docker" || !containsArg(argv, "openudon/udon:test") || !containsArg(argv, "UDON_CREDENTIAL_API_KEY") {
		t.Fatalf("unexpected docker argv: %#v", argv)
	}
	if !containsArg(argv, "/workspace/executor-report.json") {
		t.Fatalf("docker argv missing execution report path: %#v", argv)
	}
	dataFile := filepath.Join(stage, "expected", "data.hcl")
	argv, err = executorArgv(root, stage, workflow, "uws-yaml", reportPath, []string{dataFile}, nil, map[string]string{
		"OPENUDON_EXECUTOR": executor,
	})
	if err != nil {
		t.Fatalf("binary executorArgv with datafile returned error: %v", err)
	}
	if !containsArg(argv, "--datafile") || !containsArg(argv, dataFile) {
		t.Fatalf("binary argv missing datafile: %#v", argv)
	}
	if !containsArg(argv, "--execution-report") || !containsArg(argv, reportPath) {
		t.Fatalf("binary argv missing execution report path: %#v", argv)
	}

	argv, err = executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, nil, map[string]string{
		"OPENUDON_UDON_IMAGE": "openudon/udon:test",
		"OPENUDON_EXECUTOR":   executor,
	})
	if err != nil {
		t.Fatalf("binary executorArgv returned error: %v", err)
	}
	if argv[0] != executor {
		t.Fatalf("OPENUDON_EXECUTOR should override compatibility aliases, got %#v", argv)
	}

	argv, err = executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, []string{"UDON_CREDENTIAL_API_KEY"}, map[string]string{
		"OPENUDON_UDON_IMAGE": "openudon/udon:test",
	})
	if err != nil {
		t.Fatalf("compat docker executorArgv returned error: %v", err)
	}
	if argv[0] != "docker" || !containsArg(argv, "openudon/udon:test") || !containsArg(argv, "UDON_CREDENTIAL_API_KEY") {
		t.Fatalf("unexpected compat docker argv: %#v", argv)
	}

	argv, err = executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, nil, map[string]string{
		"OPENUDON_EXECUTOR": executor,
		"OPENUDON_UDON_BIN": udonBin,
	})
	if err != nil {
		t.Fatalf("binary executorArgv returned error: %v", err)
	}
	if argv[0] != executor {
		t.Fatalf("executor precedence picked %q, want %q", argv[0], executor)
	}

	if _, err := executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, nil, map[string]string{"OPENUDON_EXECUTOR": "relative"}); err == nil || !strings.Contains(err.Error(), "absolute path") {
		t.Fatalf("expected relative executor rejection, got %v", err)
	}
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, nil, map[string]string{"OPENUDON_EXECUTOR": "docker://"}); err == nil || !strings.Contains(err.Error(), "docker image") {
		t.Fatalf("expected empty docker image rejection, got %v", err)
	}
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, nil, map[string]string{"OPENUDON_EXECUTOR": "docker://bad image"}); err == nil || !strings.Contains(err.Error(), "whitespace") {
		t.Fatalf("expected whitespace docker image rejection, got %v", err)
	}
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, nil, map[string]string{"OPENUDON_EXECUTOR": filepath.Join(root, "missing")}); err == nil || !strings.Contains(err.Error(), "executable file") {
		t.Fatalf("expected missing executor rejection, got %v", err)
	}
	delimiterDriver := filepath.Join(root, "browser,driver")
	mustWriteExecutable(t, delimiterDriver)
	if _, err := executorArgvWithBrowser(root, stage, workflow, "uws-yaml", reportPath, nil, nil, &BrowserConfig{DriverPath: delimiterDriver, Protocol: "v1"}, nil, map[string]string{"OPENUDON_EXECUTOR": "docker://udon:test"}); err == nil || !strings.Contains(err.Error(), "must not contain a comma") {
		t.Fatalf("expected Docker mount-delimiter rejection, got %v", err)
	}
	nonExecutable := filepath.Join(root, "not-executable")
	mustWriteRunnerTestFile(t, nonExecutable, []byte("#!/usr/bin/env bash\nexit 0\n"))
	if _, err := executorArgv(root, stage, workflow, "uws-yaml", reportPath, nil, nil, map[string]string{"OPENUDON_EXECUTOR": nonExecutable}); err == nil || !strings.Contains(err.Error(), "executable file") {
		t.Fatalf("expected non-executable rejection, got %v", err)
	}
}

func TestLoadConfigRejectsUnknownFieldsAndTrailingJSON(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
		want string
	}{
		{
			name: "unknown-field",
			data: `{"version":"openudon.executor-run.v1","extra":true}`,
			want: "unknown field",
		},
		{
			name: "trailing-json",
			data: `{"version":"openudon.executor-run.v1"}{}`,
			want: "single JSON value",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "run-config.json")
			mustWriteRunnerTestFile(t, path, []byte(tc.data))
			_, err := LoadConfig(path)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q error, got %v", tc.want, err)
			}
		})
	}
}

func TestRunRejectsEmptyPackageSHA256(t *testing.T) {
	config := validRunnerConfig(t)
	config.PackageSHA256 = ""
	_, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
		Invoke: func(context.Context, Invocation) error {
			t.Fatal("executor should not be invoked")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "package_sha256") {
		t.Fatalf("expected package_sha256 rejection, got %v", err)
	}
}

func TestRunRejectsInvalidPackageSHA256(t *testing.T) {
	config := validRunnerConfig(t)
	config.PackageSHA256 = "not-a-sha"
	_, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
		Invoke: func(context.Context, Invocation) error {
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
		Invoke: func(context.Context, Invocation) error {
			t.Fatal("executor should not be invoked")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "package_paths") {
		t.Fatalf("expected package_paths rejection, got %v", err)
	}
}

func TestPrepareStagesWithoutCredentialValues(t *testing.T) {
	config := validRunnerConfig(t)
	config.CredentialBindings = []string{"support-api.token"}
	result, err := Prepare(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if result.StagePath == "" || result.WorkflowPath == "" {
		t.Fatalf("prepare result missing staged paths: %#v", result)
	}
	if len(result.Argv) != 0 {
		t.Fatalf("dry preparation should not derive executor argv, got %#v", result.Argv)
	}
	if !containsArg(result.CredentialEnvNames, "UDON_CREDENTIAL_SUPPORT_API_TOKEN") {
		t.Fatalf("credential env names = %#v", result.CredentialEnvNames)
	}
}

func TestPrepareAcceptsAPISourcePathsWithoutLegacyOpenAPIPaths(t *testing.T) {
	config := validRunnerConfig(t)
	config.APISourcePaths = append([]string(nil), config.OpenAPIPaths...)
	config.OpenAPIPaths = nil
	result, err := Prepare(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
	})
	if err != nil {
		t.Fatalf("Prepare returned error: %v", err)
	}
	if !containsArg(result.APISourcePaths, "openapi/support.yaml") || !containsArg(result.OpenAPIPaths, "openapi/support.yaml") {
		t.Fatalf("result missing API source compatibility paths: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(result.StagePath, "openapi", "support.yaml")); err != nil {
		t.Fatalf("staged API source missing: %v", err)
	}
}

func TestPrepareRequiresCredentialValuesWhenRequested(t *testing.T) {
	config := validRunnerConfig(t)
	config.CredentialBindings = []string{"support-api.token"}
	_, err := Prepare(context.Background(), config, Options{
		RepoRoot:                t.TempDir(),
		Env:                     []string{"OPENUDON_EXECUTOR=/bin/true"},
		RequireCredentialValues: true,
	})
	if err == nil || !strings.Contains(err.Error(), "UDON_CREDENTIAL_SUPPORT_API_TOKEN") {
		t.Fatalf("expected missing credential env failure, got %v", err)
	}
}

func TestPrepareRejectsCredentialEnvNameCollision(t *testing.T) {
	config := validRunnerConfig(t)
	config.CredentialBindings = []string{"api-key", "api_key"}
	_, err := Prepare(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
	})
	if err == nil || !strings.Contains(err.Error(), "same env var") {
		t.Fatalf("expected credential env collision failure, got %v", err)
	}
}

func TestRunRejectsDirectProductionRun(t *testing.T) {
	config := Config{DirectProductionRun: true}
	_, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=/bin/true"},
		Invoke: func(context.Context, Invocation) error {
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
		Invoke: func(_ context.Context, invocation Invocation) error {
			gotName = invocation.Argv[0]
			gotArgs = append([]string(nil), invocation.Argv[1:]...)
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

func TestRunInvocationEnvironmentIsAllowlisted(t *testing.T) {
	for _, tc := range []struct {
		name     string
		executor string
		wantPath bool
	}{
		{name: "local", executor: "/bin/true"},
		{name: "docker", executor: "docker://udon:test", wantPath: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := validRunnerConfig(t)
			config.CredentialBindings = []string{"support-api.token"}
			var invocation Invocation
			_, err := Run(context.Background(), config, Options{
				RepoRoot: t.TempDir(),
				Env: []string{
					"OPENUDON_EXECUTOR=" + tc.executor,
					"UDON_CREDENTIAL_SUPPORT_API_TOKEN=declared-secret",
					"PATH=/trusted/bin", "AWS_SECRET_ACCESS_KEY=must-not-pass",
					"HTTP_PROXY=http://must-not-pass", "SSH_AUTH_SOCK=/must-not-pass", "UNRELATED_SENTINEL=must-not-pass",
				},
				Invoke: func(_ context.Context, got Invocation) error {
					invocation = got
					return nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			joined := strings.Join(invocation.Env, "\n")
			if !strings.Contains(joined, "UDON_CREDENTIAL_SUPPORT_API_TOKEN=declared-secret") {
				t.Fatalf("declared credential missing from invocation env: %#v", invocation.Env)
			}
			for _, forbidden := range []string{"AWS_SECRET_ACCESS_KEY", "HTTP_PROXY", "SSH_AUTH_SOCK", "UNRELATED_SENTINEL", "OPENUDON_EXECUTOR"} {
				if strings.Contains(joined, forbidden) {
					t.Fatalf("%s leaked into invocation env: %#v", forbidden, invocation.Env)
				}
			}
			if got := strings.Contains(joined, "PATH=/trusted/bin"); got != tc.wantPath {
				t.Fatalf("PATH presence = %v, want %v; env=%#v", got, tc.wantPath, invocation.Env)
			}
		})
	}
}

func TestRunBrowserInvocationIsExactAndAllowlisted(t *testing.T) {
	for _, tc := range []struct {
		name     string
		executor string
	}{
		{name: "local", executor: "/bin/true"},
		{name: "docker", executor: "docker://udon:test"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := validRunnerConfig(t)
			driver := filepath.Join(t.TempDir(), "browserdriver")
			mustWriteExecutable(t, driver)
			config.CredentialBindings = []string{"member_password"}
			config.Browser = &BrowserConfig{
				DriverPath: driver, DriverArgs: []string{"--headless"}, DriverEnvironment: []string{"HOME", "PATH"}, Protocol: "v3",
				CredentialEnvironment:  []EnvironmentBinding{{Name: "member_password", Environment: "UDON_CREDENTIAL_MEMBER_PASSWORD"}},
				SessionEnvironment:     []EnvironmentBinding{{Name: "existing_member", Environment: "UDON_BROWSER_SESSION_EXISTING_MEMBER"}},
				ApprovedOperations:     []string{"read_dashboard"},
				ApprovedAuthentication: []string{"authenticate_member"},
			}
			var invocation Invocation
			result, err := Run(context.Background(), config, Options{
				RepoRoot: t.TempDir(),
				Env: []string{
					"OPENUDON_EXECUTOR=" + tc.executor,
					"UDON_CREDENTIAL_MEMBER_PASSWORD=credential-value",
					"UDON_BROWSER_SESSION_EXISTING_MEMBER=session-value",
					"HOME=/trusted/home", "PATH=/trusted/bin",
					"HTTP_PROXY=http://must-not-pass", "AWS_SECRET_ACCESS_KEY=must-not-pass", "SSH_AUTH_SOCK=/must-not-pass",
				},
				Invoke: func(_ context.Context, got Invocation) error { invocation = got; return nil },
			})
			if err != nil {
				t.Fatal(err)
			}
			expectedDriver := driver
			if tc.name == "docker" {
				expectedDriver = dockerBrowserDriverPath
				mount := "type=bind,src=" + driver + ",dst=" + dockerBrowserDriverPath + ",readonly"
				if !containsArg(invocation.Argv, mount) {
					t.Fatalf("docker invocation did not mount the configured browser driver read-only: %#v", invocation.Argv)
				}
			}
			for _, value := range []string{
				"--browser-driver", expectedDriver, "--browser-driver-arg", "--headless", "--browser-driver-protocol", "v3",
				"--browser-credential-env", "member_password=UDON_CREDENTIAL_MEMBER_PASSWORD",
				"--browser-session-env", "existing_member=UDON_BROWSER_SESSION_EXISTING_MEMBER",
				"--approve-browser-operation", "read_dashboard",
				"--approve-browser-authentication", "authenticate_member",
			} {
				if !containsArg(invocation.Argv, value) {
					t.Fatalf("browser invocation missing %q: %#v", value, invocation.Argv)
				}
			}
			if !containsArg(result.SessionEnvNames, "UDON_BROWSER_SESSION_EXISTING_MEMBER") || !containsArg(result.BrowserEnvNames, "HOME") {
				t.Fatalf("browser environment metadata = %#v / %#v", result.SessionEnvNames, result.BrowserEnvNames)
			}
			joined := strings.Join(invocation.Env, "\n")
			requiredEnvironment := []string{"UDON_CREDENTIAL_MEMBER_PASSWORD=credential-value", "UDON_BROWSER_SESSION_EXISTING_MEMBER=session-value", "PATH=/trusted/bin"}
			if tc.name == "local" {
				requiredEnvironment = append(requiredEnvironment, "HOME=/trusted/home")
			} else {
				for _, forbiddenArg := range []string{"HOME", "PATH"} {
					if containsAdjacentArgs(invocation.Argv, "-e", forbiddenArg) {
						t.Fatalf("docker forwarded host browser environment %s: %#v", forbiddenArg, invocation.Argv)
					}
				}
				if strings.Contains(joined, "HOME=/trusted/home") {
					t.Fatalf("docker launcher inherited host browser HOME: %#v", invocation.Env)
				}
			}
			for _, required := range requiredEnvironment {
				if !strings.Contains(joined, required) {
					t.Fatalf("browser invocation environment omitted %s: %#v", required, invocation.Env)
				}
			}
			for _, forbidden := range []string{"HTTP_PROXY", "AWS_SECRET_ACCESS_KEY", "SSH_AUTH_SOCK", "OPENUDON_EXECUTOR"} {
				if strings.Contains(joined, forbidden) {
					t.Fatalf("browser invocation leaked %s: %#v", forbidden, invocation.Env)
				}
			}
		})
	}
}

func TestDockerBrowserRejectsDesktopAndSocketEnvironment(t *testing.T) {
	config := validRunnerConfig(t)
	driver := filepath.Join(t.TempDir(), "browserdriver")
	mustWriteExecutable(t, driver)
	config.Browser = &BrowserConfig{DriverPath: driver, DriverEnvironment: []string{"DISPLAY"}, Protocol: "v1"}
	_, err := Run(context.Background(), config, Options{
		RepoRoot: t.TempDir(),
		Env:      []string{"OPENUDON_EXECUTOR=docker://udon:test", "DISPLAY=:99", "PATH=/trusted/bin"},
		Invoke: func(context.Context, Invocation) error {
			t.Fatal("Docker invoked with host display authority")
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported host desktop") {
		t.Fatalf("desktop environment error = %v", err)
	}
}

func containsAdjacentArgs(values []string, first, second string) bool {
	for index := 0; index+1 < len(values); index++ {
		if values[index] == first && values[index+1] == second {
			return true
		}
	}
	return false
}

func TestBrowserConfigRejectsForgedMappingsAndValues(t *testing.T) {
	driver := filepath.Join(t.TempDir(), "browserdriver")
	mustWriteExecutable(t, driver)
	base := &BrowserConfig{
		DriverPath: driver, Protocol: "v3",
		CredentialEnvironment: []EnvironmentBinding{{Name: "member_password", Environment: "UDON_CREDENTIAL_MEMBER_PASSWORD"}},
	}
	tests := []struct {
		name   string
		mutate func(*BrowserConfig)
	}{
		{name: "forged credential env", mutate: func(value *BrowserConfig) { value.CredentialEnvironment[0].Environment = "AWS_SECRET_ACCESS_KEY" }},
		{name: "proxy launcher env", mutate: func(value *BrowserConfig) { value.DriverEnvironment = []string{"HTTP_PROXY"} }},
		{name: "secret driver argument", mutate: func(value *BrowserConfig) { value.DriverArgs = []string{"Bearer abcdefghijklmnopqrstuvwxyz012345"} }},
		{name: "unsorted approvals", mutate: func(value *BrowserConfig) { value.ApprovedOperations = []string{"z", "a"} }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			value := *base
			value.CredentialEnvironment = append([]EnvironmentBinding(nil), base.CredentialEnvironment...)
			tc.mutate(&value)
			if _, err := validateBrowserConfig(&value, []string{"member_password"}, map[string]string{"UDON_CREDENTIAL_MEMBER_PASSWORD": "value"}, true, true); err == nil {
				t.Fatal("forged browser run config was accepted")
			}
		})
	}
}

func TestBrowserRegistrationEvidenceRejectsExecutorConstruction(t *testing.T) {
	config := &BrowserConfig{Protocol: "v3", DriverPath: "/trusted/browserdriver", ApprovedRegistration: []string{"register_test_user"}}
	if err := ValidateBrowserEvidenceConfig(config, nil); err != nil {
		t.Fatalf("dry-run evidence config rejected: %v", err)
	}
	if _, err := validateBrowserConfig(config, nil, map[string]string{}, false, true); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("executor construction error = %v", err)
	}
	config.Protocol = "v2"
	if err := ValidateBrowserEvidenceConfig(config, nil); err == nil || !strings.Contains(err.Error(), "protocol v3") {
		t.Fatalf("registration protocol error = %v", err)
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
		RunID:          "0123456789abcdef0123456789abcdef",
		Scope:          scope,
		PackageRoot:    root,
		WorkDir:        workdir,
		WorkflowPath:   workflowRel,
		WorkflowFormat: "uws-yaml",
		OpenAPIPaths:   []string{openAPIRel},
		PackagePaths:   []string{workflowRel, openAPIRel},
		PackageSHA256:  digest,
		HandoffSHA256:  strings.Repeat("a", 64),
		ApprovalSHA256: strings.Repeat("b", 64),
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
