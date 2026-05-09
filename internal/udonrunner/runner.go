package udonrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

const RunConfigVersion = "openudon.executor-run.v1"

type Config struct {
	Version             string   `json:"version"`
	Scope               string   `json:"scope"`
	Tier                string   `json:"tier"`
	PackageRoot         string   `json:"package_root"`
	WorkDir             string   `json:"workdir"`
	WorkflowPath        string   `json:"workflow_path"`
	WorkflowFormat      string   `json:"workflow_format"`
	OpenAPIPaths        []string `json:"openapi_paths,omitempty"`
	PackagePaths        []string `json:"package_paths"`
	PackageSHA256       string   `json:"package_sha256"`
	CredentialBindings  []string `json:"credential_bindings,omitempty"`
	DirectProductionRun bool     `json:"direct_production_run"`
}

type Options struct {
	ConfigPath string
	RepoRoot   string
	Env        []string
	Stdout     io.Writer
	Stderr     io.Writer
	RunCommand func(context.Context, string, ...string) error
}

type Result struct {
	StagePath    string
	WorkflowPath string
	Argv         []string
}

func RunConfig(ctx context.Context, opts Options) (Result, error) {
	config, err := LoadConfig(opts.ConfigPath)
	if err != nil {
		return Result{}, err
	}
	return Run(ctx, config, opts)
}

func LoadConfig(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, fmt.Errorf("--config is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read run config: %w", err)
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return Config{}, fmt.Errorf("run config must be valid JSON: %w", err)
	}
	if config.Version != RunConfigVersion {
		return Config{}, fmt.Errorf("unsupported run config version: %s", config.Version)
	}
	if config.OpenAPIPaths == nil {
		config.OpenAPIPaths = []string{}
	}
	if config.PackagePaths == nil {
		config.PackagePaths = []string{}
	}
	if config.CredentialBindings == nil {
		config.CredentialBindings = []string{}
	}
	return config, nil
}

func Run(ctx context.Context, config Config, opts Options) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if config.DirectProductionRun {
		return Result{}, fmt.Errorf("run config direct_production_run must be false")
	}
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	repoRootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return Result{}, fmt.Errorf("resolve repo root: %w", err)
	}

	packageRoot, err := requireAbsDir(config.PackageRoot, "package_root")
	if err != nil {
		return Result{}, err
	}
	if err := packageartifacts.ValidatePackageRoot(packageRoot); err != nil {
		return Result{}, err
	}
	workdir, err := requireAbsPath(config.WorkDir, "workdir")
	if err != nil {
		return Result{}, err
	}
	workflowFormat := strings.TrimSpace(config.WorkflowFormat)
	if workflowFormat == "" {
		workflowFormat = "uws-yaml"
	}
	if err := rejectControlChars("workflow_format", workflowFormat); err != nil {
		return Result{}, err
	}

	workflowRaw, err := requireString(config.WorkflowPath, "workflow_path")
	if err != nil {
		return Result{}, err
	}
	workflowRel, workflowPath, err := packageRelativePath(packageRoot, "workflow_path", workflowRaw)
	if err != nil {
		return Result{}, err
	}
	if err := validateRegularPackageFile(packageRoot, workflowRel, workflowPath, "workflow"); err != nil {
		return Result{}, err
	}
	openAPIFiles, err := validateOpenAPIPaths(packageRoot, config.OpenAPIPaths)
	if err != nil {
		return Result{}, err
	}
	packageFiles, err := validatePackagePaths(packageRoot, config.PackagePaths)
	if err != nil {
		return Result{}, err
	}
	approvedDigest, err := requirePackageSHA256(config.PackageSHA256)
	if err != nil {
		return Result{}, err
	}
	if err := validateDigestInventory(workflowRel, openAPIFiles, packageFiles); err != nil {
		return Result{}, err
	}
	credentialEnvNames, err := credentialEnvNames(config.CredentialBindings)
	if err != nil {
		return Result{}, err
	}
	env := opts.Env
	if env == nil {
		env = os.Environ()
	}
	envByName := environmentMap(env)
	for _, name := range credentialEnvNames {
		if strings.TrimSpace(envByName[name]) == "" {
			return Result{}, fmt.Errorf("required credential env var is not set: %s", name)
		}
	}

	stage, stagedWorkflow, err := stagePackage(workdir, workflowRel, workflowPath, openAPIFiles, packageFiles)
	if err != nil {
		return Result{}, err
	}
	if err := verifyStagedPackageDigest(stage, config.Scope, approvedDigest, packageFiles); err != nil {
		return Result{}, err
	}
	argv, err := executorArgv(repoRootAbs, stage, stagedWorkflow, workflowFormat, credentialEnvNames, envByName)
	if err != nil {
		return Result{}, err
	}
	result := Result{StagePath: stage, WorkflowPath: stagedWorkflow, Argv: append([]string(nil), argv...)}
	runCommand := opts.RunCommand
	if runCommand == nil {
		runCommand = func(ctx context.Context, name string, args ...string) error {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Dir = repoRootAbs
			cmd.Env = env
			cmd.Stdout = opts.Stdout
			cmd.Stderr = opts.Stderr
			return cmd.Run()
		}
	}
	if err := runCommand(ctx, argv[0], argv[1:]...); err != nil {
		return result, fmt.Errorf("invoke trusted executor: %w", err)
	}
	return result, nil
}

func requireString(value, name string) (string, error) {
	if err := rejectControlChars(name, value); err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("run config requires %s", name)
	}
	return value, nil
}

func requireAbsDir(value, name string) (string, error) {
	value, err := requireString(value, name)
	if err != nil {
		return "", err
	}
	return filepath.Abs(value)
}

func requireAbsPath(value, name string) (string, error) {
	value, err := requireString(value, name)
	if err != nil {
		return "", err
	}
	return filepath.Abs(value)
}

func rejectControlChars(name, value string) error {
	for _, ch := range value {
		if ch < 0x20 || ch == 0x7f {
			return fmt.Errorf("%s must not contain control characters", name)
		}
	}
	return nil
}

func rejectBackslash(name, value string) error {
	if strings.Contains(value, `\`) {
		return fmt.Errorf("%s must use slash separators: %s", name, value)
	}
	return nil
}

func packageRelativePath(packageRoot, name, value string) (string, string, error) {
	if err := rejectControlChars(name, value); err != nil {
		return "", "", err
	}
	if err := rejectBackslash(name, value); err != nil {
		return "", "", err
	}
	var abs string
	var err error
	if filepath.IsAbs(value) {
		abs, err = filepath.Abs(value)
	} else {
		abs, err = filepath.Abs(filepath.Join(packageRoot, filepath.FromSlash(filepath.Clean(value))))
	}
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(packageRoot, abs)
	if err != nil {
		return "", "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", "", fmt.Errorf("%s escapes package_root: %s", name, value)
	}
	rel = filepath.Clean(rel)
	for _, segment := range strings.Split(rel, string(filepath.Separator)) {
		if segment == "" || segment == "." {
			return "", "", fmt.Errorf("%s path is invalid: %s", name, filepath.ToSlash(rel))
		}
		if segment == ".." {
			return "", "", fmt.Errorf("%s escapes package_root: %s", name, value)
		}
	}
	return rel, abs, nil
}

func validateRegularPackageFile(packageRoot, rel, absolute, label string) error {
	current := packageRoot
	segments := strings.Split(rel, string(filepath.Separator))
	for index, segment := range segments {
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			return fmt.Errorf("%s file not found: %s: %w", label, absolute, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s file must not be a symlink: %s", label, absolute)
		}
		last := index == len(segments)-1
		if last {
			if !info.Mode().IsRegular() {
				return fmt.Errorf("%s file must be a regular file: %s", label, absolute)
			}
			continue
		}
		if !info.IsDir() {
			return fmt.Errorf("%s parent must be a directory: %s", label, absolute)
		}
	}
	return nil
}

func validateOpenAPIPaths(packageRoot string, paths []string) ([][2]string, error) {
	out := make([][2]string, 0, len(paths))
	for _, raw := range paths {
		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("openapi path must be non-empty")
		}
		rel, src, err := packageRelativePath(packageRoot, "openapi path", raw)
		if err != nil {
			return nil, err
		}
		if err := validateRegularPackageFile(packageRoot, rel, src, "openapi"); err != nil {
			return nil, err
		}
		out = append(out, [2]string{rel, src})
	}
	return out, nil
}

func validatePackagePaths(packageRoot string, paths []string) ([][2]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("run config requires package_paths")
	}
	out := make([][2]string, 0, len(paths))
	seen := map[string]bool{}
	for _, raw := range paths {
		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("package path must be non-empty")
		}
		rel, src, err := packageRelativePath(packageRoot, "package path", raw)
		if err != nil {
			return nil, err
		}
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			continue
		}
		if err := validateRegularPackageFile(packageRoot, filepath.FromSlash(rel), src, "package"); err != nil {
			return nil, err
		}
		seen[rel] = true
		out = append(out, [2]string{rel, src})
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out, nil
}

func requirePackageSHA256(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("run config requires package_sha256")
	}
	return value, nil
}

func validateDigestInventory(workflowRel string, openAPIFiles, packageFiles [][2]string) error {
	covered := map[string]bool{}
	for _, pair := range packageFiles {
		covered[filepath.ToSlash(pair[0])] = true
	}
	var missing []string
	if !covered[filepath.ToSlash(workflowRel)] {
		missing = append(missing, filepath.ToSlash(workflowRel))
	}
	for _, pair := range openAPIFiles {
		rel := filepath.ToSlash(pair[0])
		if !covered[rel] {
			missing = append(missing, rel)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("package_paths must include digest-covered executor input(s): %s", strings.Join(missing, ", "))
	}
	return nil
}

func stagePackage(workdir, workflowRel, workflowPath string, openAPIFiles, packageFiles [][2]string) (string, string, error) {
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return "", "", err
	}
	stage, err := os.MkdirTemp(workdir, "stage.")
	if err != nil {
		return "", "", err
	}
	stagedWorkflow := filepath.Join(stage, workflowRel)
	if err := copyRegularFile(workflowPath, stagedWorkflow); err != nil {
		return "", "", err
	}
	for _, pair := range openAPIFiles {
		if err := copyRegularFile(pair[1], filepath.Join(stage, pair[0])); err != nil {
			return "", "", err
		}
	}
	for _, pair := range packageFiles {
		if err := copyRegularFile(pair[1], filepath.Join(stage, filepath.FromSlash(pair[0]))); err != nil {
			return "", "", err
		}
	}
	return stage, stagedWorkflow, nil
}

func verifyStagedPackageDigest(stage, scope, approvedDigest string, packageFiles [][2]string) error {
	approvedDigest = strings.TrimSpace(approvedDigest)
	inputs := make([]authoring.ReviewHandoffInput, 0, len(packageFiles))
	for _, pair := range packageFiles {
		inputs = append(inputs, authoring.ReviewHandoffInput{
			Path:     pair[0],
			Required: true,
		})
	}
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:    stage,
		Scope:   scope,
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  inputs,
	})
	if err != nil {
		return fmt.Errorf("verify staged package digest: %w", err)
	}
	if digest != approvedDigest {
		return fmt.Errorf("staged package_sha256 does not match approved handoff package")
	}
	return nil
}

func copyRegularFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func credentialEnvNames(bindings []string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, binding := range bindings {
		binding = strings.TrimSpace(binding)
		if binding == "" {
			continue
		}
		if err := rejectControlChars("credential binding", binding); err != nil {
			return nil, err
		}
		name := credentialEnvName(binding)
		if name == "UDON_CREDENTIAL" {
			return nil, fmt.Errorf("credential binding does not produce a valid env var: %s", binding)
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

func credentialEnvName(binding string) string {
	var b strings.Builder
	b.WriteString("UDON_CREDENTIAL_")
	lastUnderscore := false
	for _, ch := range strings.TrimSpace(binding) {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			b.WriteByte(byte(ch))
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.TrimRight(strings.ToUpper(b.String()), "_")
}

func environmentMap(env []string) map[string]string {
	out := map[string]string{}
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if ok {
			out[key] = value
		}
	}
	return out
}

func executorArgv(repoRoot, stage, stagedWorkflow, workflowFormat string, credentialNames []string, env map[string]string) ([]string, error) {
	if image := strings.TrimSpace(env["OPENUDON_UDON_IMAGE"]); image != "" {
		argv := []string{"docker", "run", "--rm", "-v", stage + ":/workspace", "-w", "/workspace"}
		for _, name := range credentialNames {
			argv = append(argv, "-e", name)
		}
		rel, err := filepath.Rel(stage, stagedWorkflow)
		if err != nil {
			return nil, err
		}
		argv = append(argv, image, "--workdir", "/workspace", "--workflow", "/workspace/"+filepath.ToSlash(rel), "--workflow-format", workflowFormat)
		return argv, nil
	}
	if executor := strings.TrimSpace(env["OPENUDON_EXECUTOR"]); executor != "" {
		return executorPathArgv("OPENUDON_EXECUTOR", executor, stage, stagedWorkflow, workflowFormat)
	}
	if executor := strings.TrimSpace(env["OPENUDON_UDON_BIN"]); executor != "" {
		return executorPathArgv("OPENUDON_UDON_BIN", executor, stage, stagedWorkflow, workflowFormat)
	}
	executor := filepath.Join(repoRoot, "..", "udon", "dist", "udon-linux-amd64")
	if !isExecutable(executor) {
		executor = filepath.Join(repoRoot, "..", "udon", "udon")
	}
	if !isExecutable(executor) {
		return nil, fmt.Errorf("trusted executor not found. Set OPENUDON_EXECUTOR, OPENUDON_UDON_BIN, OPENUDON_UDON_IMAGE, or build ../udon")
	}
	return []string{executor, "--workdir", stage, "--workflow", stagedWorkflow, "--workflow-format", workflowFormat}, nil
}

func executorPathArgv(envName, executor, stage, stagedWorkflow, workflowFormat string) ([]string, error) {
	if !filepath.IsAbs(executor) {
		return nil, fmt.Errorf("%s must be an absolute path: %s", envName, executor)
	}
	if !isExecutable(executor) {
		return nil, fmt.Errorf("%s does not point to an executable file: %s", envName, executor)
	}
	return []string{executor, "--workdir", stage, "--workflow", stagedWorkflow, "--workflow-format", workflowFormat}, nil
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
