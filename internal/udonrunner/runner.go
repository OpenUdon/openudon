package udonrunner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/executablefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/processgroup"
)

const (
	RunConfigVersion        = "openudon.executor-run.v2"
	LegacyRunConfigVersion  = "openudon.executor-run.v1"
	dockerBrowserDriverPath = "/openudon/browser-driver"
)
const dockerExecutorPrefix = "docker://"

type Config struct {
	Version               string         `json:"version"`
	RunID                 string         `json:"run_id"`
	Scope                 string         `json:"scope"`
	Tier                  string         `json:"tier"`
	PackageRoot           string         `json:"package_root"`
	WorkDir               string         `json:"workdir"`
	WorkflowPath          string         `json:"workflow_path"`
	WorkflowFormat        string         `json:"workflow_format"`
	DataFiles             []string       `json:"data_files,omitempty"`
	APISourcePaths        []string       `json:"api_source_paths,omitempty"`
	OpenAPIPaths          []string       `json:"openapi_paths,omitempty"`
	PackagePaths          []string       `json:"package_paths"`
	PackageSHA256         string         `json:"package_sha256"`
	HandoffSHA256         string         `json:"handoff_sha256"`
	ApprovalSHA256        string         `json:"approval_sha256"`
	ExecutorReportVersion string         `json:"executor_report_version,omitempty"`
	CredentialBindings    []string       `json:"credential_bindings,omitempty"`
	Browser               *BrowserConfig `json:"browser,omitempty"`
	DirectProductionRun   bool           `json:"direct_production_run"`
}

// BrowserConfig is the complete value-free browser replay contract. Secret
// and session values stay in the named environment variables.
type BrowserConfig struct {
	DriverPath                    string               `json:"driver_path,omitempty"`
	DriverArgs                    []string             `json:"driver_args,omitempty"`
	DriverEnvironment             []string             `json:"driver_environment,omitempty"`
	Protocol                      string               `json:"protocol"`
	CredentialEnvironment         []EnvironmentBinding `json:"credential_environment,omitempty"`
	SessionEnvironment            []EnvironmentBinding `json:"session_environment,omitempty"`
	ApprovedOperations            []string             `json:"approved_operations,omitempty"`
	ApprovedAuthentication        []string             `json:"approved_authentication,omitempty"`
	ApprovedRegistration          []string             `json:"approved_registration,omitempty"`
	AttestedRegistration          []string             `json:"attested_registration,omitempty"`
	RegistrationAttestationSHA256 string               `json:"registration_attestation_sha256,omitempty"`
}

// EnvironmentBinding maps a reviewed symbolic runtime name to its canonical
// allowlisted environment-variable name; it never contains a value.
type EnvironmentBinding struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
}

// Invocation is the complete, auditable process boundary used by trusted
// runner callers and tests. Argv always includes argv[0].
type Invocation struct {
	Argv []string
	Dir  string
	Env  []string
}

type InvokeFunc func(context.Context, Invocation) error

type Options struct {
	ConfigPath              string
	RepoRoot                string
	Env                     []string
	RequireCredentialValues bool
	Stdout                  io.Writer
	Stderr                  io.Writer
	Invoke                  InvokeFunc
}

type Result struct {
	StagePath          string
	WorkflowPath       string
	ExecutorReportPath string
	Argv               []string
	PackageRoot        string
	WorkDir            string
	APISourcePaths     []string
	OpenAPIPaths       []string
	DataFiles          []string
	PackagePaths       []string
	PackageSHA256      string
	CredentialEnvNames []string
	SessionEnvNames    []string
	BrowserEnvNames    []string
}

func LoadConfig(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, fmt.Errorf("--config is required")
	}
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		return Config{}, fmt.Errorf("read run config: %w", err)
	}
	var config Config
	if err := evidencefile.DecodeStrict(data, &config); err != nil {
		return Config{}, fmt.Errorf("run config must be valid JSON: %w", err)
	}
	if config.Version != RunConfigVersion {
		if config.Version == LegacyRunConfigVersion {
			return Config{}, fmt.Errorf("legacy run config %s is read-only and cannot execute; regenerate the package with openudon build", config.Version)
		}
		return Config{}, fmt.Errorf("unsupported run config version: %s", config.Version)
	}
	if config.OpenAPIPaths == nil {
		config.OpenAPIPaths = []string{}
	}
	if config.APISourcePaths == nil {
		config.APISourcePaths = []string{}
	}
	if config.DataFiles == nil {
		config.DataFiles = []string{}
	}
	if config.PackagePaths == nil {
		config.PackagePaths = []string{}
	}
	if config.CredentialBindings == nil {
		config.CredentialBindings = []string{}
	}
	if strings.TrimSpace(config.ExecutorReportVersion) == "" {
		config.ExecutorReportVersion = "udon.execution-report.v2"
	}
	if config.Browser != nil {
		normalizeBrowserConfig(config.Browser)
	}
	return config, nil
}

func Prepare(ctx context.Context, config Config, opts Options) (Result, error) {
	result, _, _, err := prepare(ctx, config, opts, opts.RequireCredentialValues, false)
	return result, err
}

func Run(ctx context.Context, config Config, opts Options) (Result, error) {
	result, env, repoRootAbs, err := prepare(ctx, config, opts, true, true)
	if err != nil {
		return result, err
	}
	invocation := Invocation{Argv: append([]string(nil), result.Argv...), Dir: repoRootAbs, Env: append([]string(nil), env...)}
	invoke := opts.Invoke
	if invoke == nil {
		invoke = func(ctx context.Context, invocation Invocation) error {
			return processgroup.RunContext(ctx, processgroup.Invocation{
				Args: invocation.Argv, Dir: invocation.Dir, Env: invocation.Env,
				Stdout: opts.Stdout, Stderr: opts.Stderr,
			})
		}
	}
	if err := invoke(ctx, invocation); err != nil {
		return result, fmt.Errorf("invoke trusted executor: %w", err)
	}
	return result, nil
}

func prepare(ctx context.Context, config Config, opts Options, requireCredentialValues, buildExecutorArgv bool) (Result, []string, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Result{}, nil, "", err
	}
	if config.DirectProductionRun {
		return Result{}, nil, "", fmt.Errorf("run config direct_production_run must be false")
	}
	if config.Version != RunConfigVersion {
		if config.Version == LegacyRunConfigVersion {
			return Result{}, nil, "", fmt.Errorf("legacy run config %s is read-only and cannot execute; regenerate the package with openudon build", config.Version)
		}
		return Result{}, nil, "", fmt.Errorf("run config version must be %s", RunConfigVersion)
	}
	reportVersion := strings.TrimSpace(config.ExecutorReportVersion)
	if reportVersion == "" {
		reportVersion = "udon.execution-report.v2"
	}
	if reportVersion != "udon.execution-report.v2" && reportVersion != "udon.execution-report.v3" {
		return Result{}, nil, "", fmt.Errorf("run config executor_report_version must be udon.execution-report.v2 or v3")
	}
	if config.Browser != nil && strings.EqualFold(strings.TrimSpace(config.Browser.Protocol), "v4") && reportVersion != "udon.execution-report.v3" {
		return Result{}, nil, "", fmt.Errorf("browser registration protocol v4 requires udon.execution-report.v3")
	}
	if err := ValidateRunID(config.RunID); err != nil {
		return Result{}, nil, "", err
	}
	if _, err := requirePackageSHA256(config.HandoffSHA256); err != nil {
		return Result{}, nil, "", fmt.Errorf("run config handoff_sha256: %w", err)
	}
	if _, err := requirePackageSHA256(config.ApprovalSHA256); err != nil {
		return Result{}, nil, "", fmt.Errorf("run config approval_sha256: %w", err)
	}
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	repoRootAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return Result{}, nil, "", fmt.Errorf("resolve repo root: %w", err)
	}

	packageRoot, err := requireAbsDir(config.PackageRoot, "package_root")
	if err != nil {
		return Result{}, nil, "", err
	}
	if err := packageartifacts.ValidatePackageRoot(packageRoot); err != nil {
		return Result{}, nil, "", err
	}
	workdir, err := requireAbsPath(config.WorkDir, "workdir")
	if err != nil {
		return Result{}, nil, "", err
	}
	workflowFormat := strings.TrimSpace(config.WorkflowFormat)
	if workflowFormat == "" {
		workflowFormat = "uws-yaml"
	}
	if err := rejectControlChars("workflow_format", workflowFormat); err != nil {
		return Result{}, nil, "", err
	}

	workflowRaw, err := requireString(config.WorkflowPath, "workflow_path")
	if err != nil {
		return Result{}, nil, "", err
	}
	workflowRel, workflowPath, err := packageRelativePath(packageRoot, "workflow_path", workflowRaw)
	if err != nil {
		return Result{}, nil, "", err
	}
	if err := validateRegularPackageFile(packageRoot, workflowRel, workflowPath, "workflow"); err != nil {
		return Result{}, nil, "", err
	}
	apiSourceFiles, err := validateAPISourcePaths(ctx, packageRoot, runConfigAPISourcePaths(config))
	if err != nil {
		return Result{}, nil, "", err
	}
	dataFiles, err := validateDataFilePaths(ctx, packageRoot, config.DataFiles)
	if err != nil {
		return Result{}, nil, "", err
	}
	packageFiles, err := validatePackagePaths(ctx, packageRoot, config.PackagePaths)
	if err != nil {
		return Result{}, nil, "", err
	}
	if err := validateDataFilesInPackagePaths(dataFiles, packageFiles); err != nil {
		return Result{}, nil, "", err
	}
	approvedDigest, err := requirePackageSHA256(config.PackageSHA256)
	if err != nil {
		return Result{}, nil, "", err
	}
	if err := validateDigestInventory(workflowRel, apiSourceFiles, packageFiles); err != nil {
		return Result{}, nil, "", err
	}
	credentialEnvNames, err := credentialEnvNames(config.CredentialBindings)
	if err != nil {
		return Result{}, nil, "", err
	}
	sourceEnv := opts.Env
	if sourceEnv == nil {
		sourceEnv = os.Environ()
	}
	envByName := environmentMap(sourceEnv)
	if requireCredentialValues {
		for _, name := range credentialEnvNames {
			if strings.TrimSpace(envByName[name]) == "" {
				return Result{}, nil, "", fmt.Errorf("required credential env var is not set: %s", name)
			}
		}
	}
	browser, err := validateBrowserConfig(config.Browser, config.CredentialBindings, envByName, requireCredentialValues, buildExecutorArgv)
	if err != nil {
		return Result{}, nil, "", err
	}

	stage, stagedWorkflow, err := stagePackage(ctx, workdir, workflowRel, workflowPath, apiSourceFiles, packageFiles)
	if err != nil {
		return Result{}, nil, "", err
	}
	if err := verifyStagedPackageDigest(ctx, stage, config.Scope, approvedDigest, packageFiles); err != nil {
		return Result{}, nil, "", err
	}
	result := Result{
		StagePath:          stage,
		WorkflowPath:       stagedWorkflow,
		PackageRoot:        packageRoot,
		WorkDir:            workdir,
		APISourcePaths:     relPaths(apiSourceFiles),
		OpenAPIPaths:       relPaths(apiSourceFiles),
		DataFiles:          relPaths(dataFiles),
		PackagePaths:       relPaths(packageFiles),
		PackageSHA256:      approvedDigest,
		CredentialEnvNames: append([]string(nil), credentialEnvNames...),
		SessionEnvNames:    append([]string(nil), browser.sessionEnv...),
		BrowserEnvNames:    append([]string(nil), browser.driverEnv...),
	}
	if buildExecutorArgv {
		result.ExecutorReportPath = filepath.Join(stage, "executor-report-"+config.RunID+".json")
		argv, err := executorArgvWithBrowser(repoRootAbs, stage, stagedWorkflow, workflowFormat, result.ExecutorReportPath, stagedDataFilePaths(stage, dataFiles), credentialEnvNames, config.Browser, browser.driverEnv, envByName)
		if err != nil {
			return result, nil, "", err
		}
		result.Argv = append([]string(nil), argv...)
	}
	executorNames := append(append([]string(nil), credentialEnvNames...), browser.sessionEnv...)
	if len(result.Argv) == 0 || filepath.Base(result.Argv[0]) != "docker" {
		executorNames = append(executorNames, browser.driverEnv...)
	}
	sort.Strings(executorNames)
	executorEnv := credentialEnvironment(executorNames, envByName)
	if len(result.Argv) > 0 && filepath.Base(result.Argv[0]) == "docker" {
		executorEnv = append(executorEnv, launcherEnvironment(envByName)...)
		sort.Strings(executorEnv)
	}
	return result, executorEnv, repoRootAbs, nil
}

// ValidateRunID validates the identifier used to derive unique executor and
// report paths across the parent and external trusted-runner processes.
func ValidateRunID(value string) error {
	value = strings.TrimSpace(value)
	if len(value) < 16 || len(value) > 64 {
		return fmt.Errorf("run config run_id must be 16 to 64 lowercase hexadecimal characters")
	}
	for _, ch := range value {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			return fmt.Errorf("run config run_id must be 16 to 64 lowercase hexadecimal characters")
		}
	}
	return nil
}

func credentialEnvironment(names []string, values map[string]string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		if value, ok := values[name]; ok {
			out = append(out, name+"="+value)
		}
	}
	sort.Strings(out)
	return out
}

// launcherEnvironment is the documented minimum needed to locate and launch
// Docker (and its Windows process helpers). Cloud, proxy, agent, and unrelated
// variables are intentionally excluded.
func launcherEnvironment(values map[string]string) []string {
	names := []string{"PATH", "PATHEXT", "SystemRoot", "SYSTEMROOT", "WINDIR", "COMSPEC", "TMP", "TEMP"}
	var out []string
	seen := map[string]bool{}
	for _, name := range names {
		if seen[name] {
			continue
		}
		if value, ok := values[name]; ok && value != "" {
			out = append(out, name+"="+value)
			seen[name] = true
		}
	}
	return out
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

func runConfigAPISourcePaths(config Config) []string {
	seen := map[string]bool{}
	var out []string
	for _, raw := range append(append([]string(nil), config.APISourcePaths...), config.OpenAPIPaths...) {
		clean := filepath.ToSlash(strings.TrimSpace(raw))
		if clean == "" {
			out = append(out, raw)
			continue
		}
		if seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, raw)
	}
	sort.Strings(out)
	return out
}

func validateAPISourcePaths(ctx context.Context, packageRoot string, paths []string) ([][2]string, error) {
	out := make([][2]string, 0, len(paths))
	for _, raw := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.TrimSpace(raw) == "" {
			return nil, fmt.Errorf("api source path must be non-empty")
		}
		rel, src, err := packageRelativePath(packageRoot, "api source path", raw)
		if err != nil {
			return nil, err
		}
		if err := validateRegularPackageFile(packageRoot, rel, src, "api source"); err != nil {
			return nil, err
		}
		out = append(out, [2]string{rel, src})
	}
	return out, nil
}

func validatePackagePaths(ctx context.Context, packageRoot string, paths []string) ([][2]string, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("run config requires package_paths")
	}
	out := make([][2]string, 0, len(paths))
	seen := map[string]bool{}
	for _, raw := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
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
	if len(value) != 64 {
		return "", fmt.Errorf("run config package_sha256 must be a 64-character hex SHA-256 digest")
	}
	for _, ch := range value {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return "", fmt.Errorf("run config package_sha256 must be a 64-character hex SHA-256 digest")
		}
	}
	return value, nil
}

func relPaths(files [][2]string) []string {
	out := make([]string, 0, len(files))
	for _, pair := range files {
		out = append(out, filepath.ToSlash(pair[0]))
	}
	sort.Strings(out)
	return out
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

func validateDataFilesInPackagePaths(dataFiles, packageFiles [][2]string) error {
	if len(dataFiles) == 0 {
		return nil
	}
	packageSet := map[string]struct{}{}
	for _, pair := range packageFiles {
		packageSet[pair[0]] = struct{}{}
	}
	var missing []string
	for _, pair := range dataFiles {
		if _, ok := packageSet[pair[0]]; !ok {
			missing = append(missing, pair[0])
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("data_files must be included in package_paths: %s", strings.Join(missing, ", "))
	}
	return nil
}

func validateDataFilePaths(ctx context.Context, packageRoot string, paths []string) ([][2]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	files, err := validatePackagePaths(ctx, packageRoot, paths)
	if err != nil {
		return nil, fmt.Errorf("data_files: %w", err)
	}
	return files, nil
}

func stagedDataFilePaths(stage string, dataFiles [][2]string) []string {
	out := make([]string, 0, len(dataFiles))
	for _, pair := range dataFiles {
		out = append(out, filepath.Join(stage, filepath.FromSlash(pair[0])))
	}
	return out
}

func stagePackage(ctx context.Context, workdir, workflowRel, workflowPath string, openAPIFiles, packageFiles [][2]string) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return "", "", err
	}
	stage, err := os.MkdirTemp(workdir, "stage.")
	if err != nil {
		return "", "", err
	}
	keepStage := false
	defer func() {
		if !keepStage {
			_ = os.RemoveAll(stage)
		}
	}()
	stagedWorkflow := filepath.Join(stage, workflowRel)
	if err := copyRegularFile(ctx, workflowPath, stagedWorkflow); err != nil {
		return "", "", err
	}
	for _, pair := range openAPIFiles {
		if err := copyRegularFile(ctx, pair[1], filepath.Join(stage, pair[0])); err != nil {
			return "", "", err
		}
	}
	for _, pair := range packageFiles {
		if err := copyRegularFile(ctx, pair[1], filepath.Join(stage, filepath.FromSlash(pair[0]))); err != nil {
			return "", "", err
		}
	}
	keepStage = true
	return stage, stagedWorkflow, nil
}

func verifyStagedPackageDigest(ctx context.Context, stage, scope, approvedDigest string, packageFiles [][2]string) error {
	approvedDigest = strings.TrimSpace(approvedDigest)
	inputs := make([]authoring.ReviewHandoffInput, 0, len(packageFiles))
	for _, pair := range packageFiles {
		inputs = append(inputs, authoring.ReviewHandoffInput{
			Path:     pair[0],
			Required: true,
		})
	}
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Context: ctx,
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

func copyRegularFile(ctx context.Context, src, dst string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, _, err := evidencefile.ReadRegular(src, evidencefile.DefaultMaxBytes)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return atomicfile.Write(dst, data, 0o644)
}

func credentialEnvNames(bindings []string) ([]string, error) {
	seen := map[string]string{}
	var out []string
	for _, binding := range bindings {
		binding = strings.TrimSpace(binding)
		if binding == "" {
			continue
		}
		if err := rejectControlChars("credential binding", binding); err != nil {
			return nil, err
		}
		name := CredentialEnvironmentName(binding)
		if name == "UDON_CREDENTIAL" {
			return nil, fmt.Errorf("credential binding does not produce a valid env var: %s", binding)
		}
		if previous, ok := seen[name]; ok {
			if previous != binding {
				return nil, fmt.Errorf("credential bindings %q and %q produce the same env var %s", previous, binding, name)
			}
			continue
		}
		seen[name] = binding
		out = append(out, name)
	}
	sort.Strings(out)
	return out, nil
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

func executorArgv(repoRoot, stage, stagedWorkflow, workflowFormat, executorReportPath string, dataFiles []string, credentialNames []string, env map[string]string) ([]string, error) {
	return executorArgvWithBrowser(repoRoot, stage, stagedWorkflow, workflowFormat, executorReportPath, dataFiles, credentialNames, nil, nil, env)
}

func executorArgvWithBrowser(repoRoot, stage, stagedWorkflow, workflowFormat, executorReportPath string, dataFiles []string, credentialNames []string, browser *BrowserConfig, driverEnvNames []string, env map[string]string) ([]string, error) {
	if executor := strings.TrimSpace(env["OPENUDON_EXECUTOR"]); executor != "" {
		if strings.HasPrefix(executor, dockerExecutorPrefix) {
			image := strings.TrimPrefix(executor, dockerExecutorPrefix)
			return dockerImageArgv("OPENUDON_EXECUTOR", image, stage, stagedWorkflow, workflowFormat, executorReportPath, dataFiles, credentialNames, browser, driverEnvNames)
		}
		return executorPathArgv("OPENUDON_EXECUTOR", executor, stage, stagedWorkflow, workflowFormat, executorReportPath, dataFiles, browser, driverEnvNames)
	}
	if image := strings.TrimSpace(env["OPENUDON_UDON_IMAGE"]); image != "" {
		return dockerImageArgv("OPENUDON_UDON_IMAGE", image, stage, stagedWorkflow, workflowFormat, executorReportPath, dataFiles, credentialNames, browser, driverEnvNames)
	}
	if executor := strings.TrimSpace(env["OPENUDON_UDON_BIN"]); executor != "" {
		return executorPathArgv("OPENUDON_UDON_BIN", executor, stage, stagedWorkflow, workflowFormat, executorReportPath, dataFiles, browser, driverEnvNames)
	}
	executor := filepath.Join(repoRoot, "..", "udon", "dist", "udon-linux-amd64")
	if !executablefile.Is(executor) {
		executor = filepath.Join(repoRoot, "..", "udon", "udon")
	}
	if !executablefile.Is(executor) {
		return nil, fmt.Errorf("trusted executor not found. Set OPENUDON_EXECUTOR to an absolute binary path or docker://image, or build ../udon")
	}
	if browser != nil && browser.DriverPath != "" && !executablefile.Is(browser.DriverPath) {
		return nil, fmt.Errorf("browser driver does not point to an executable file: %s", browser.DriverPath)
	}
	argv := appendDataFileArgs(appendExecutionReportArg([]string{executor, "--workdir", stage, "--workflow", stagedWorkflow, "--workflow-format", workflowFormat}, executorReportPath), dataFiles...)
	return appendBrowserArgs(argv, browser, driverEnvNames), nil
}

func dockerImageArgv(envName, image, stage, stagedWorkflow, workflowFormat, executorReportPath string, dataFiles, credentialNames []string, browser *BrowserConfig, driverEnvNames []string) ([]string, error) {
	if err := validateDockerImage(envName, image); err != nil {
		return nil, err
	}
	argv := []string{"docker", "run", "--rm", "-v", stage + ":/workspace", "-w", "/workspace"}
	containerBrowser := browser
	if browser != nil && strings.TrimSpace(browser.DriverPath) != "" {
		if !executablefile.Is(browser.DriverPath) {
			return nil, fmt.Errorf("browser driver does not point to an executable file: %s", browser.DriverPath)
		}
		if strings.Contains(browser.DriverPath, ",") {
			return nil, fmt.Errorf("browser driver path must not contain a comma for Docker execution: %s", browser.DriverPath)
		}
		argv = append(argv, "--mount", "type=bind,src="+browser.DriverPath+",dst="+dockerBrowserDriverPath+",readonly")
		clone := *browser
		clone.DriverPath = dockerBrowserDriverPath
		containerBrowser = &clone
	}
	passNames := append([]string(nil), credentialNames...)
	if browser != nil {
		for _, binding := range browser.SessionEnvironment {
			passNames = append(passNames, binding.Environment)
		}
		var err error
		driverEnvNames, err = dockerBrowserDriverEnvironment(driverEnvNames)
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(passNames)
	for _, name := range passNames {
		argv = append(argv, "-e", name)
	}
	rel, err := filepath.Rel(stage, stagedWorkflow)
	if err != nil {
		return nil, err
	}
	argv = append(argv, image, "--workdir", "/workspace", "--workflow", "/workspace/"+filepath.ToSlash(rel), "--workflow-format", workflowFormat)
	if strings.TrimSpace(executorReportPath) != "" {
		relReport, err := filepath.Rel(stage, executorReportPath)
		if err != nil {
			return nil, err
		}
		argv = append(argv, "--execution-report", "/workspace/"+filepath.ToSlash(relReport))
	}
	for _, dataFile := range dataFiles {
		relData, err := filepath.Rel(stage, dataFile)
		if err != nil {
			return nil, err
		}
		argv = append(argv, "--datafile", "/workspace/"+filepath.ToSlash(relData))
	}
	return appendBrowserArgs(argv, containerBrowser, driverEnvNames), nil
}

func dockerBrowserDriverEnvironment(names []string) ([]string, error) {
	containerOwned := map[string]bool{
		"HOME": true, "PATH": true, "TMPDIR": true, "TMP": true, "TEMP": true,
		"LANG": true, "LC_ALL": true, "LC_CTYPE": true, "PLAYWRIGHT_BROWSERS_PATH": true,
	}
	result := make([]string, 0, len(names))
	for _, name := range names {
		if !containerOwned[name] {
			return nil, fmt.Errorf("Docker browser driver environment %s requires an unsupported host desktop, socket, or platform binding", name)
		}
		result = append(result, name)
	}
	return result, nil
}

func validateDockerImage(envName, image string) error {
	if strings.TrimSpace(image) == "" {
		return fmt.Errorf("%s docker image must be non-empty", envName)
	}
	if err := rejectControlChars(envName, image); err != nil {
		return err
	}
	if strings.ContainsAny(image, " \t\r\n") {
		return fmt.Errorf("%s docker image must not contain whitespace: %s", envName, image)
	}
	return nil
}

func executorPathArgv(envName, executor, stage, stagedWorkflow, workflowFormat, executorReportPath string, dataFiles []string, browser *BrowserConfig, driverEnvNames []string) ([]string, error) {
	if !filepath.IsAbs(executor) {
		return nil, fmt.Errorf("%s must be an absolute path: %s", envName, executor)
	}
	if !executablefile.Is(executor) {
		return nil, fmt.Errorf("%s does not point to an executable file: %s", envName, executor)
	}
	if browser != nil && browser.DriverPath != "" && !executablefile.Is(browser.DriverPath) {
		return nil, fmt.Errorf("browser driver does not point to an executable file: %s", browser.DriverPath)
	}
	argv := appendDataFileArgs(appendExecutionReportArg([]string{executor, "--workdir", stage, "--workflow", stagedWorkflow, "--workflow-format", workflowFormat}, executorReportPath), dataFiles...)
	return appendBrowserArgs(argv, browser, driverEnvNames), nil
}

func appendBrowserArgs(argv []string, browser *BrowserConfig, driverEnvNames []string) []string {
	if browser == nil {
		return argv
	}
	argv = append(argv, "--browser-driver", browser.DriverPath)
	for _, value := range browser.DriverArgs {
		argv = append(argv, "--browser-driver-arg", value)
	}
	argv = append(argv, "--browser-driver-protocol", browser.Protocol)
	for _, name := range driverEnvNames {
		argv = append(argv, "--browser-driver-env", name)
	}
	for _, binding := range browser.CredentialEnvironment {
		argv = append(argv, "--browser-credential-env", binding.Name+"="+binding.Environment)
	}
	for _, binding := range browser.SessionEnvironment {
		argv = append(argv, "--browser-session-env", binding.Name+"="+binding.Environment)
	}
	for _, operation := range browser.ApprovedOperations {
		argv = append(argv, "--approve-browser-operation", operation)
	}
	for _, operation := range browser.ApprovedAuthentication {
		argv = append(argv, "--approve-browser-authentication", operation)
	}
	for _, operation := range browser.AttestedRegistration {
		argv = append(argv, "--attest-browser-registration", operation)
	}
	for _, operation := range browser.ApprovedRegistration {
		if browser.Protocol == "v4" {
			argv = append(argv, "--approve-browser-registration", operation)
		}
	}
	return argv
}

func appendExecutionReportArg(argv []string, reportPath string) []string {
	reportPath = strings.TrimSpace(reportPath)
	if reportPath == "" {
		return argv
	}
	return append(argv, "--execution-report", reportPath)
}

func appendDataFileArgs(argv []string, dataFiles ...string) []string {
	for _, dataFile := range dataFiles {
		dataFile = strings.TrimSpace(dataFile)
		if dataFile == "" {
			continue
		}
		argv = append(argv, "--datafile", dataFile)
	}
	return argv
}
