package trustedrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

const (
	ApprovalVersion      = "openudon.approval.v1"
	RunConfigVersion     = udonrunner.RunConfigVersion
	RunEvidenceVersion   = "openudon.run-evidence.v1"
	ReviewHandoffVersion = authoring.ReviewHandoffVersion

	StateApprovedForSandbox    = string(authoring.ReviewStateApprovedForSandbox)
	StateApprovedForProduction = string(authoring.ReviewStateApprovedForProduction)

	TierSandbox    = "sandbox"
	TierProduction = "production"
)

type Approval struct {
	Version       string `json:"version"`
	Scope         string `json:"scope"`
	State         string `json:"state"`
	Reviewer      string `json:"reviewer"`
	ApprovedAt    string `json:"approved_at"`
	ExpiresAt     string `json:"expires_at,omitempty"`
	PackageSHA256 string `json:"package_sha256"`
	Notes         string `json:"notes,omitempty"`
}

type Options struct {
	RepoRoot     string
	ExampleDir   string
	Tier         string
	ApprovalPath string
	WorkDir      string
	DryRun       bool
	RunnerPath   string
	Stdout       io.Writer
	Stderr       io.Writer
	Now          func() time.Time
	Assess       func(context.Context, synthesize.Options) (*synthesize.QualityReport, error)
	RunCommand   func(context.Context, string, ...string) error
}

type TemplateOptions struct {
	RepoRoot   string
	ExampleDir string
	State      string
	Reviewer   string
	Notes      string
	Now        func() time.Time
	Assess     func(context.Context, synthesize.Options) (*synthesize.QualityReport, error)
}

type RunResult struct {
	Scope           string
	Tier            string
	PackageSHA256   string
	WorkflowPath    string
	RunConfigPath   string
	RunEvidencePath string
	WorkDir         string
	StagePath       string
	DryRun          bool
}

type RunConfig = udonrunner.Config

type RunEvidence struct {
	Version            string              `json:"version"`
	CreatedAt          string              `json:"created_at"`
	Scope              string              `json:"scope"`
	Tier               string              `json:"tier"`
	DryRun             bool                `json:"dry_run"`
	ApprovalState      string              `json:"approval_state"`
	PackageSHA256      string              `json:"package_sha256"`
	RunConfigPath      string              `json:"run_config_path"`
	PackageRoot        string              `json:"package_root"`
	WorkDir            string              `json:"workdir"`
	StageKind          string              `json:"stage_kind"`
	StagePath          string              `json:"stage_path"`
	WorkflowPath       string              `json:"workflow_path"`
	PackagePaths       []string            `json:"package_paths"`
	APISourcePaths     []string            `json:"api_source_paths,omitempty"`
	CredentialBindings []string            `json:"credential_bindings,omitempty"`
	CredentialEnvNames []string            `json:"credential_env_names,omitempty"`
	Gates              []RunEvidenceGate   `json:"gates"`
	Executor           RunEvidenceExecutor `json:"executor"`
}

type RunEvidenceGate struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type RunEvidenceExecutor struct {
	Invoked    bool     `json:"invoked"`
	Mode       string   `json:"mode"`
	RunnerPath string   `json:"runner_path,omitempty"`
	Argv       []string `json:"argv,omitempty"`
}

type paths struct {
	repoRoot       string
	exampleAbs     string
	scope          string
	project        string
	workflow       string
	quality        string
	handoff        string
	defaultWorkDir string
}

type handoffManifest = authoring.ReviewHandoff

func Run(ctx context.Context, opts Options) (*RunResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	p, manifest, digest, err := validatePackage(ctx, packageOptions{
		RepoRoot:   opts.RepoRoot,
		ExampleDir: opts.ExampleDir,
		Assess:     opts.Assess,
	})
	if err != nil {
		return nil, err
	}
	if err := validateManifestPolicy(manifest); err != nil {
		return nil, err
	}
	approval, err := readApproval(opts.ApprovalPath)
	if err != nil {
		return nil, err
	}
	now := resolveNow(opts.Now)
	if err := validateApproval(approval, p.scope, digest, opts.Tier, now); err != nil {
		return nil, err
	}

	workdir, err := resolveRunWorkDir(p, opts.WorkDir)
	if err != nil {
		return nil, err
	}
	result := &RunResult{
		Scope:         p.scope,
		Tier:          opts.Tier,
		PackageSHA256: digest,
		WorkflowPath:  filepath.Join(p.exampleAbs, "workflows", "workflow.uws.yaml"),
		WorkDir:       workdir,
		DryRun:        opts.DryRun,
	}
	runConfig, err := buildRunConfig(p, manifest, digest, opts.Tier, result.WorkDir)
	if err != nil {
		return nil, err
	}
	runConfigPath, err := writeRunConfig(runConfig)
	if err != nil {
		return nil, err
	}
	result.RunConfigPath = runConfigPath
	if opts.DryRun {
		prepared, err := udonrunner.PrepareConfig(ctx, udonrunner.Options{
			ConfigPath: runConfigPath,
			RepoRoot:   p.repoRoot,
		})
		if err != nil {
			return nil, fmt.Errorf("prepare trusted executor dry-run: %w", err)
		}
		result.StagePath = prepared.StagePath
		evidencePath, err := writeRunEvidence(result.WorkDir, buildRunEvidence(runEvidenceOptions{
			Config:         runConfig,
			Approval:       approval,
			Prepared:       prepared,
			Result:         result,
			Mode:           "dry-run",
			StageKind:      "dry-run",
			ExecutorStatus: "",
			Now:            now,
		}))
		if err != nil {
			return nil, err
		}
		result.RunEvidencePath = evidencePath
		return result, nil
	}

	runnerPath := strings.TrimSpace(opts.RunnerPath)
	if runnerPath != "" {
		if err := validateRunnerPath("OPENUDON_UDON_RUNNER", runnerPath); err != nil {
			return nil, err
		}
		prepared, err := udonrunner.PrepareConfig(ctx, udonrunner.Options{
			ConfigPath:              runConfigPath,
			RepoRoot:                p.repoRoot,
			RequireCredentialValues: true,
		})
		if err != nil {
			return nil, fmt.Errorf("prepare trusted executor: %w", err)
		}
		result.StagePath = prepared.StagePath
		args := []string{"--config", runConfigPath}
		executorArgv := append([]string{runnerPath}, args...)
		runCommand := opts.RunCommand
		if runCommand == nil {
			runCommand = func(ctx context.Context, name string, args ...string) error {
				cmd := exec.CommandContext(ctx, name, args...)
				cmd.Dir = p.repoRoot
				cmd.Stdout = opts.Stdout
				cmd.Stderr = opts.Stderr
				return cmd.Run()
			}
		}
		if err := runCommand(ctx, runnerPath, args...); err != nil {
			evidencePath, evidenceErr := writeRunEvidence(result.WorkDir, buildRunEvidence(runEvidenceOptions{
				Config:         runConfig,
				Approval:       approval,
				Prepared:       prepared,
				Result:         result,
				Invoked:        true,
				Mode:           "external-runner",
				RunnerPath:     runnerPath,
				ExecutorArgv:   executorArgv,
				StageKind:      "preflight",
				ExecutorStatus: "fail",
				Now:            now,
			}))
			if evidenceErr != nil {
				return result, fmt.Errorf("run trusted executor: %w; write run evidence: %v", err, evidenceErr)
			}
			result.RunEvidencePath = evidencePath
			return result, fmt.Errorf("run trusted executor: %w", err)
		}
		evidencePath, err := writeRunEvidence(result.WorkDir, buildRunEvidence(runEvidenceOptions{
			Config:         runConfig,
			Approval:       approval,
			Prepared:       prepared,
			Result:         result,
			Invoked:        true,
			Mode:           "external-runner",
			RunnerPath:     runnerPath,
			ExecutorArgv:   executorArgv,
			StageKind:      "preflight",
			ExecutorStatus: "pass",
			Now:            now,
		}))
		if err != nil {
			return nil, err
		}
		result.RunEvidencePath = evidencePath
		return result, nil
	}
	prepared, err := udonrunner.RunConfig(ctx, udonrunner.Options{
		ConfigPath: runConfigPath,
		RepoRoot:   p.repoRoot,
		Stdout:     opts.Stdout,
		Stderr:     opts.Stderr,
		RunCommand: opts.RunCommand,
	})
	if err != nil {
		result.StagePath = prepared.StagePath
		evidencePath, evidenceErr := writeRunEvidence(result.WorkDir, buildRunEvidence(runEvidenceOptions{
			Config:         runConfig,
			Approval:       approval,
			Prepared:       prepared,
			Result:         result,
			Invoked:        true,
			Mode:           "internal-runner",
			StageKind:      "executor",
			ExecutorStatus: "fail",
			Now:            now,
		}))
		if evidenceErr != nil {
			return result, fmt.Errorf("run trusted executor: %w; write run evidence: %v", err, evidenceErr)
		}
		result.RunEvidencePath = evidencePath
		return result, fmt.Errorf("run trusted executor: %w", err)
	}
	result.StagePath = prepared.StagePath
	evidencePath, err := writeRunEvidence(result.WorkDir, buildRunEvidence(runEvidenceOptions{
		Config:         runConfig,
		Approval:       approval,
		Prepared:       prepared,
		Result:         result,
		Invoked:        true,
		Mode:           "internal-runner",
		StageKind:      "executor",
		ExecutorStatus: "pass",
		Now:            now,
	}))
	if err != nil {
		return nil, err
	}
	result.RunEvidencePath = evidencePath
	return result, nil
}

func validateRunnerPath(name, path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be an absolute path: %s", name, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s does not point to an executable file: %s", name, path)
	}
	if info.IsDir() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("%s does not point to an executable file: %s", name, path)
	}
	return nil
}

func buildRunConfig(p paths, manifest handoffManifest, digest, tier, workdir string) (RunConfig, error) {
	relOpenAPI, err := packageartifacts.CollectAPISourcePaths(p.exampleAbs)
	if err != nil {
		return RunConfig{}, err
	}
	packagePaths, err := packagePathsForRunConfig(p, manifest)
	if err != nil {
		return RunConfig{}, err
	}
	config := RunConfig{
		Version:             RunConfigVersion,
		Scope:               p.scope,
		Tier:                tier,
		PackageRoot:         p.exampleAbs,
		WorkDir:             workdir,
		WorkflowPath:        filepath.ToSlash(filepath.Join("workflows", "workflow.uws.yaml")),
		WorkflowFormat:      "uws-yaml",
		DataFiles:           runConfigDataFiles(p.exampleAbs),
		OpenAPIPaths:        relOpenAPI,
		PackagePaths:        packagePaths,
		PackageSHA256:       digest,
		CredentialBindings:  sortedCredentialBindings(manifest),
		DirectProductionRun: false,
	}
	if config.WorkDir == "" {
		config.WorkDir = p.defaultWorkDir
	}
	return config, nil
}

func runConfigDataFiles(exampleRoot string) []string {
	info, err := os.Lstat(filepath.Join(exampleRoot, filepath.FromSlash(packageartifacts.RuntimeDataPath)))
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	return []string{packageartifacts.RuntimeDataPath}
}

func packagePathsForRunConfig(p paths, manifest handoffManifest) ([]string, error) {
	paths, err := packageartifacts.RequiredManifestPaths(p.exampleAbs, packageManifestInputs(manifest))
	if err != nil {
		return nil, err
	}
	return append([]string(nil), paths...), nil
}

func writeRunConfig(config RunConfig) (string, error) {
	if strings.TrimSpace(config.WorkDir) == "" {
		return "", fmt.Errorf("run config workdir is required")
	}
	if err := os.MkdirAll(config.WorkDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(config.WorkDir, "run-config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func writeRunEvidence(workdir string, evidence RunEvidence) (string, error) {
	if strings.TrimSpace(workdir) == "" {
		return "", fmt.Errorf("run evidence workdir is required")
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(workdir, "run-evidence.json")
	data, err := json.MarshalIndent(evidence, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

type runEvidenceOptions struct {
	Config         RunConfig
	Approval       Approval
	Prepared       udonrunner.Result
	Result         *RunResult
	Invoked        bool
	Mode           string
	RunnerPath     string
	ExecutorArgv   []string
	StageKind      string
	ExecutorStatus string
	Now            time.Time
}

func buildRunEvidence(opts runEvidenceOptions) RunEvidence {
	gates := []RunEvidenceGate{
		{Name: "handoff_package", Status: "pass"},
		{Name: "manifest_policy", Status: "pass"},
		{Name: "stored_quality", Status: "pass"},
		{Name: "current_quality", Status: "pass"},
		{Name: "approval", Status: "pass"},
		{Name: "run_config", Status: "pass"},
		{Name: "staged_digest", Status: "pass"},
	}
	if opts.ExecutorStatus != "" {
		gates = append(gates, RunEvidenceGate{Name: "executor_invocation", Status: opts.ExecutorStatus})
	}
	executorArgv := append([]string(nil), opts.ExecutorArgv...)
	if len(executorArgv) == 0 {
		executorArgv = append(executorArgv, opts.Prepared.Argv...)
	}
	return RunEvidence{
		Version:            RunEvidenceVersion,
		CreatedAt:          opts.Now.UTC().Format(time.RFC3339),
		Scope:              opts.Result.Scope,
		Tier:               opts.Result.Tier,
		DryRun:             opts.Result.DryRun,
		ApprovalState:      opts.Approval.State,
		PackageSHA256:      opts.Result.PackageSHA256,
		RunConfigPath:      opts.Result.RunConfigPath,
		PackageRoot:        opts.Config.PackageRoot,
		WorkDir:            opts.Result.WorkDir,
		StageKind:          opts.StageKind,
		StagePath:          opts.Prepared.StagePath,
		WorkflowPath:       opts.Prepared.WorkflowPath,
		PackagePaths:       append([]string(nil), opts.Prepared.PackagePaths...),
		APISourcePaths:     append([]string(nil), opts.Prepared.OpenAPIPaths...),
		CredentialBindings: append([]string(nil), opts.Config.CredentialBindings...),
		CredentialEnvNames: append([]string(nil), opts.Prepared.CredentialEnvNames...),
		Gates:              gates,
		Executor: RunEvidenceExecutor{
			Invoked:    opts.Invoked,
			Mode:       opts.Mode,
			RunnerPath: opts.RunnerPath,
			Argv:       executorArgv,
		},
	}
}

func resolveRunWorkDir(p paths, input string) (string, error) {
	input = strings.TrimSpace(input)
	for _, ch := range input {
		if ch < 0x20 || ch == 0x7f {
			return "", fmt.Errorf("run workdir must not contain control characters")
		}
	}
	if input == "" {
		input = p.defaultWorkDir
	} else if !filepath.IsAbs(input) {
		input = filepath.Join(p.repoRoot, input)
	}
	workdir, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	return filepath.Clean(workdir), nil
}

func sortedCredentialBindings(manifest handoffManifest) []string {
	seen := map[string]bool{}
	for _, binding := range append(append([]string(nil), manifest.CredentialBindings.Declared...), manifest.CredentialBindings.ExpectedFromPlan...) {
		name := strings.TrimSpace(binding)
		if name != "" {
			seen[name] = true
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func relOrAbs(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return rel
}

func ApprovalTemplate(ctx context.Context, opts TemplateOptions) (Approval, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	p, manifest, digest, err := validatePackage(ctx, packageOptions{
		RepoRoot:   opts.RepoRoot,
		ExampleDir: opts.ExampleDir,
		Assess:     opts.Assess,
	})
	if err != nil {
		return Approval{}, err
	}
	if err := validateManifestPolicy(manifest); err != nil {
		return Approval{}, err
	}
	state := strings.TrimSpace(opts.State)
	if state != StateApprovedForSandbox && state != StateApprovedForProduction {
		return Approval{}, fmt.Errorf("--state must be %s or %s", StateApprovedForSandbox, StateApprovedForProduction)
	}
	reviewer := strings.TrimSpace(opts.Reviewer)
	if reviewer == "" {
		return Approval{}, fmt.Errorf("--reviewer is required")
	}
	return Approval{
		Version:       ApprovalVersion,
		Scope:         p.scope,
		State:         state,
		Reviewer:      reviewer,
		ApprovedAt:    resolveNow(opts.Now).UTC().Format(time.RFC3339),
		PackageSHA256: digest,
		Notes:         strings.TrimSpace(opts.Notes),
	}, nil
}

func WriteApproval(w io.Writer, approval Approval) error {
	data, err := json.MarshalIndent(approval, "", "  ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(data, '\n'))
	return err
}

type packageOptions struct {
	RepoRoot   string
	ExampleDir string
	Assess     func(context.Context, synthesize.Options) (*synthesize.QualityReport, error)
}

func validatePackage(ctx context.Context, opts packageOptions) (paths, handoffManifest, string, error) {
	p, err := resolvePaths(opts.RepoRoot, opts.ExampleDir)
	if err != nil {
		return paths{}, handoffManifest{}, "", err
	}
	manifest, err := readHandoff(p.handoff)
	if err != nil {
		return paths{}, handoffManifest{}, "", err
	}
	if err := validateRequiredInputs(p, manifest); err != nil {
		return paths{}, handoffManifest{}, "", err
	}
	if err := validateStoredQuality(p.quality); err != nil {
		return paths{}, handoffManifest{}, "", err
	}
	assess := opts.Assess
	if assess == nil {
		assess = synthesize.AssessCurrent
	}
	report, err := assess(ctx, synthesize.Options{ExampleDir: p.exampleAbs})
	if err != nil {
		return paths{}, handoffManifest{}, "", fmt.Errorf("assess current quality: %w", err)
	}
	if !report.Passed() {
		return paths{}, handoffManifest{}, "", fmt.Errorf("current quality gate is %q", report.Status)
	}
	digest, err := computePackageDigest(p, manifest)
	if err != nil {
		return paths{}, handoffManifest{}, "", err
	}
	return p, manifest, digest, nil
}

func resolvePaths(repoRoot, example string) (paths, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	repoAbs, err := filepath.Abs(repoRoot)
	if err != nil {
		return paths{}, err
	}
	example = strings.TrimSpace(example)
	if example == "" {
		return paths{}, fmt.Errorf("--example is required")
	}
	exampleAbs, err := filepath.Abs(example)
	if err != nil {
		return paths{}, err
	}
	rel, err := filepath.Rel(repoAbs, exampleAbs)
	if err != nil {
		return paths{}, err
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return paths{}, fmt.Errorf("example must be inside repo root: %s", example)
	}
	if err := packageartifacts.ValidatePackageRoot(exampleAbs); err != nil {
		return paths{}, err
	}
	scope := filepath.ToSlash(filepath.Clean(rel))
	return paths{
		repoRoot:       repoAbs,
		exampleAbs:     exampleAbs,
		scope:          scope,
		project:        filepath.Join(exampleAbs, "project.md"),
		workflow:       filepath.Join(exampleAbs, "workflows", "workflow.hcl"),
		quality:        filepath.Join(exampleAbs, "expected", "quality.json"),
		handoff:        filepath.Join(exampleAbs, filepath.FromSlash(packageartifacts.ReviewHandoffPath)),
		defaultWorkDir: filepath.Join(repoAbs, ".openudon-run", strings.ReplaceAll(scope, "/", "-")),
	}, nil
}

func readHandoff(path string) (handoffManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return handoffManifest{}, fmt.Errorf("read handoff manifest: %w", err)
	}
	var manifest handoffManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return handoffManifest{}, fmt.Errorf("handoff manifest must be valid JSON: %w", err)
	}
	allowedVersions := []string{ReviewHandoffVersion}
	if diagnostics := authoring.ValidateReviewHandoff(manifest, authoring.ReviewHandoffValidationOptions{AllowedVersions: allowedVersions}); len(diagnostics) > 0 {
		return handoffManifest{}, fmt.Errorf("handoff manifest is invalid: %s", diagnostics[0].Message)
	}
	return manifest, nil
}

func validateManifestPolicy(manifest handoffManifest) error {
	if manifest.CredentialBindings.ValuesAllowedInArtifacts {
		return fmt.Errorf("handoff manifest allows credential values in artifacts")
	}
	if manifest.ExecutionPolicy.DirectProductionExecution {
		return fmt.Errorf("handoff manifest allows direct production execution")
	}
	return nil
}

func validateRequiredInputs(p paths, manifest handoffManifest) error {
	paths, err := packageartifacts.RequiredManifestPaths(p.exampleAbs, packageManifestInputs(manifest))
	if err != nil {
		return err
	}
	return packageartifacts.ValidateRegularPackageFiles(p.exampleAbs, paths)
}

func validateStoredQuality(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read quality report: %w", err)
	}
	var report synthesize.QualityReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("quality report must be valid JSON: %w", err)
	}
	if !report.Passed() {
		return fmt.Errorf("stored quality report is %q", report.Status)
	}
	return nil
}

func computePackageDigest(p paths, manifest handoffManifest) (string, error) {
	manifestPaths, err := packageartifacts.RequiredManifestPaths(p.exampleAbs, packageManifestInputs(manifest))
	if err != nil {
		return "", err
	}
	manifestInputByPath := map[string]authoring.ReviewHandoffInput{}
	for _, input := range manifest.HandoffInputs {
		if !input.Required {
			continue
		}
		clean, err := packageartifacts.CleanRelativePath(input.Path)
		if err != nil {
			return "", fmt.Errorf("handoff input path must be safe relative path: %q", input.Path)
		}
		input.Path = clean
		manifestInputByPath[clean] = input
	}
	inputs := make([]authoring.ReviewHandoffInput, 0, len(manifestPaths))
	for _, path := range manifestPaths {
		inputs = append(inputs, manifestInputByPath[path])
	}
	return authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root:    p.exampleAbs,
		Scope:   p.scope,
		Version: "openudon.handoff-package-digest.v1",
		Inputs:  inputs,
	})
}

func packageManifestInputs(manifest handoffManifest) []packageartifacts.ManifestInput {
	inputs := make([]packageartifacts.ManifestInput, 0, len(manifest.HandoffInputs))
	for _, input := range manifest.HandoffInputs {
		inputs = append(inputs, packageartifacts.ManifestInput{
			Path:     input.Path,
			Required: input.Required,
		})
	}
	return inputs
}

func readApproval(path string) (Approval, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Approval{}, fmt.Errorf("--approval is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Approval{}, fmt.Errorf("read approval: %w", err)
	}
	var approval Approval
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&approval); err != nil {
		return Approval{}, fmt.Errorf("approval must be valid JSON: %w", err)
	}
	var extra struct{}
	if err := dec.Decode(&extra); err != io.EOF {
		return Approval{}, fmt.Errorf("approval must contain a single JSON object")
	}
	return approval, nil
}

func validateApproval(approval Approval, scope, digest, tier string, now time.Time) error {
	if approval.Version != ApprovalVersion {
		return fmt.Errorf("approval version must be %s", ApprovalVersion)
	}
	if approval.Scope != scope {
		return fmt.Errorf("approval scope %q does not match %q", approval.Scope, scope)
	}
	if strings.TrimSpace(approval.Reviewer) == "" {
		return fmt.Errorf("approval reviewer is required")
	}
	if _, err := time.Parse(time.RFC3339, approval.ApprovedAt); err != nil {
		return fmt.Errorf("approval approved_at must be RFC3339: %w", err)
	}
	if strings.TrimSpace(approval.ExpiresAt) != "" {
		expires, err := time.Parse(time.RFC3339, approval.ExpiresAt)
		if err != nil {
			return fmt.Errorf("approval expires_at must be RFC3339: %w", err)
		}
		if !now.Before(expires) {
			return fmt.Errorf("approval expired at %s", expires.Format(time.RFC3339))
		}
	}
	if approval.PackageSHA256 != digest {
		return fmt.Errorf("approval package_sha256 does not match current handoff package")
	}
	if err := validateTierState(tier, approval.State); err != nil {
		return err
	}
	return nil
}

func validateTierState(tier, state string) error {
	switch tier {
	case TierSandbox:
		if state == StateApprovedForSandbox || state == StateApprovedForProduction {
			return nil
		}
	case TierProduction:
		if state == StateApprovedForProduction {
			return nil
		}
	default:
		return fmt.Errorf("--tier must be %s or %s", TierSandbox, TierProduction)
	}
	return fmt.Errorf("approval state %q is not valid for %s tier", state, tier)
}

func resolveNow(now func() time.Time) time.Time {
	if now == nil {
		return time.Now()
	}
	return now()
}
