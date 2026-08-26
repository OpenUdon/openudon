package trustedrunner

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	asyncevidence "github.com/OpenUdon/evidence/async"
	evdigest "github.com/OpenUdon/evidence/digest"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/executablefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/processgroup"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

const (
	ApprovalVersion            = "openudon.approval.v1"
	AsyncEvidenceVersion       = "openudon.async-evidence-bundle.v1"
	RunConfigVersion           = udonrunner.RunConfigVersion
	RunEvidenceVersion         = "openudon.run-evidence.v2"
	LegacyRunEvidenceVersion   = "openudon.run-evidence.v1"
	UdonExecutionReportVersion = udonreport.Version
	ReviewHandoffVersion       = authoring.ReviewHandoffVersion

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
	RepoRoot                    string
	ExampleDir                  string
	Tier                        string
	ApprovalPath                string
	WorkDir                     string
	DryRun                      bool
	RunnerPath                  string
	Stdout                      io.Writer
	Stderr                      io.Writer
	Now                         func() time.Time
	Env                         []string
	Assess                      func(context.Context, synthesize.Options) (*synthesize.QualityReport, error)
	Invoke                      udonrunner.InvokeFunc
	SigningKey                  string
	BrowserDriver               string
	BrowserDriverArgs           []string
	RegistrationAttestationPath string
	RegistrationSubmitApproval  string
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

// PackageInspection is a non-writing summary of an exact, currently passing
// review package. It exposes the information needed for a handoff screen
// without creating an approval or accepting runtime credentials.
type PackageInspection struct {
	Scope              string                             `json:"scope"`
	PackageSHA256      string                             `json:"package_sha256"`
	HandoffSHA256      string                             `json:"handoff_sha256"`
	ExecutionPolicy    authoring.ReviewExecutionPolicy    `json:"execution_policy"`
	CredentialBindings authoring.ReviewCredentialBindings `json:"credential_bindings"`
	ApprovalStates     []authoring.ReviewApprovalState    `json:"approval_states"`
}

// InspectPackage validates and hashes the current package without writing an
// approval, run configuration, evidence file, or execution artifact.
func InspectPackage(ctx context.Context, opts TemplateOptions) (PackageInspection, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	validated, err := validatePackage(ctx, packageOptions{
		RepoRoot: opts.RepoRoot, ExampleDir: opts.ExampleDir, Assess: opts.Assess,
	})
	if err != nil {
		return PackageInspection{}, err
	}
	if err := validateManifestPolicy(validated.manifest); err != nil {
		return PackageInspection{}, err
	}
	return PackageInspection{
		Scope: validated.paths.scope, PackageSHA256: validated.packageSHA256, HandoffSHA256: validated.handoffSHA256,
		ExecutionPolicy: validated.manifest.ExecutionPolicy, CredentialBindings: validated.manifest.CredentialBindings,
		ApprovalStates: append([]authoring.ReviewApprovalState(nil), validated.manifest.ApprovalStates...),
	}, nil
}

// RevalidatePackageBytes rederives the complete manifest-bound package from
// current bytes and compares it with a prior passing inspection. It performs
// no assessment or write: an unchanged digest-bound package necessarily
// retains the exact quality and handoff bytes that were assessed previously.
func RevalidatePackageBytes(ctx context.Context, opts TemplateOptions, expected PackageInspection) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := resolveAndValidatePackageBytes(opts.RepoRoot, opts.ExampleDir)
	if err != nil {
		return err
	}
	if expected.Scope != current.paths.scope {
		return fmt.Errorf("reviewed package scope changed")
	}
	if current.packageSHA256 != expected.PackageSHA256 {
		return fmt.Errorf("reviewed package SHA-256 changed")
	}
	if current.handoffSHA256 != expected.HandoffSHA256 {
		return fmt.Errorf("review handoff SHA-256 changed")
	}
	return nil
}

func resolveAndValidatePackageBytes(repoRoot, exampleDir string) (validatedPackage, error) {
	p, err := resolvePaths(repoRoot, exampleDir)
	if err != nil {
		return validatedPackage{}, err
	}
	manifest, handoffBytes, err := readHandoff(p.handoff)
	if err != nil {
		return validatedPackage{}, err
	}
	snapshot, err := readPackageSnapshot(p, manifest, handoffBytes)
	if err != nil {
		return validatedPackage{}, err
	}
	if err := validateRequiredSnapshotInputs(manifest, snapshot); err != nil {
		return validatedPackage{}, err
	}
	qualityBytes, err := snapshot.read("expected/quality.json")
	if err != nil {
		return validatedPackage{}, err
	}
	if err := validateStoredQualityBytes(qualityBytes); err != nil {
		return validatedPackage{}, err
	}
	if err := validateManifestPolicy(manifest); err != nil {
		return validatedPackage{}, err
	}
	packageSHA256, err := computePackageSnapshotDigest(p, manifest, snapshot)
	if err != nil {
		return validatedPackage{}, err
	}
	return validatedPackage{
		paths: p, manifest: manifest, snapshot: snapshot,
		packageSHA256: packageSHA256, handoffSHA256: evidencefile.SHA256(handoffBytes),
	}, nil
}

type RunResult struct {
	RunID             string
	Scope             string
	Tier              string
	PackageSHA256     string
	WorkflowPath      string
	RunConfigPath     string
	RunEvidencePath   string
	AsyncEvidencePath string
	WorkDir           string
	StagePath         string
	DryRun            bool
}

type VerifyRunEvidenceResult struct {
	RunEvidencePath    string
	AsyncEvidenceFiles []RunEvidenceAsyncFile
}

type RunConfig = udonrunner.Config

type RunEvidence struct {
	Version            string                    `json:"version"`
	RunID              string                    `json:"run_id"`
	CreatedAt          string                    `json:"created_at"`
	Scope              string                    `json:"scope"`
	Tier               string                    `json:"tier"`
	DryRun             bool                      `json:"dry_run"`
	ApprovalState      string                    `json:"approval_state"`
	PackageSHA256      string                    `json:"package_sha256"`
	HandoffSHA256      string                    `json:"handoff_sha256"`
	ApprovalSHA256     string                    `json:"approval_sha256"`
	RunConfigSHA256    string                    `json:"run_config_sha256"`
	RunConfigPath      string                    `json:"run_config_path"`
	PackageRoot        string                    `json:"package_root"`
	WorkDir            string                    `json:"workdir"`
	StageKind          string                    `json:"stage_kind"`
	StagePath          string                    `json:"stage_path"`
	WorkflowPath       string                    `json:"workflow_path"`
	PackagePaths       []string                  `json:"package_paths"`
	APISourcePaths     []string                  `json:"api_source_paths,omitempty"`
	CredentialBindings []string                  `json:"credential_bindings,omitempty"`
	CredentialEnvNames []string                  `json:"credential_env_names,omitempty"`
	Browser            *udonrunner.BrowserConfig `json:"browser,omitempty"`
	Gates              []RunEvidenceGate         `json:"gates"`
	Executor           RunEvidenceExecutor       `json:"executor"`
	AsyncEvidenceFiles []RunEvidenceAsyncFile    `json:"async_evidence_files,omitempty"`
}

type RunEvidenceGate struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type RunEvidenceExecutor struct {
	Invoked      bool     `json:"invoked"`
	Mode         string   `json:"mode"`
	RunnerPath   string   `json:"runner_path,omitempty"`
	Argv         []string `json:"argv,omitempty"`
	ReportPath   string   `json:"report_path,omitempty"`
	ReportSHA256 string   `json:"report_sha256,omitempty"`
	ReportSize   int64    `json:"report_size,omitempty"`
}

type RunEvidenceAsyncFile struct {
	Path    string `json:"path"`
	Digest  string `json:"digest"`
	Records int    `json:"records"`
	Purpose string `json:"purpose"`
}

type AsyncEvidenceBundle struct {
	Version string                `json:"version"`
	Records []AsyncEvidenceRecord `json:"records"`
}

type AsyncEvidenceRecord struct {
	Kind                        string                                     `json:"kind"`
	ExecutionRequest            *asyncevidence.ExecutionRequest            `json:"execution_request,omitempty"`
	ExecutionResponse           *asyncevidence.ExecutionResponse           `json:"execution_response,omitempty"`
	StatusObservation           *asyncevidence.StatusObservation           `json:"status_observation,omitempty"`
	ConfirmationReadObservation *asyncevidence.ConfirmationReadObservation `json:"confirmation_read_observation,omitempty"`
}

type UdonExecutionReport = udonreport.Report

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
	signingKeyPEM, err := loadSigningKeyPEM(opts.SigningKey)
	if err != nil {
		return nil, fmt.Errorf("validate signing key before execution: %w", err)
	}
	validated, err := validatePackage(ctx, packageOptions{
		RepoRoot:   opts.RepoRoot,
		ExampleDir: opts.ExampleDir,
		Assess:     opts.Assess,
	})
	if err != nil {
		return nil, err
	}
	p, manifest, digest := validated.paths, validated.manifest, validated.packageSHA256
	if err := validateManifestPolicy(manifest); err != nil {
		return nil, err
	}
	approval, approvalBytes, err := readApprovalDocument(opts.ApprovalPath)
	if err != nil {
		return nil, err
	}
	now := resolveNow(opts.Now)
	if err := validateApproval(approval, p.scope, digest, opts.Tier, now); err != nil {
		return nil, err
	}

	runID, err := newRunID()
	if err != nil {
		return nil, fmt.Errorf("create run ID: %w", err)
	}
	workdir, err := resolveRunWorkDir(p, opts.WorkDir, runID)
	if err != nil {
		return nil, err
	}
	result := &RunResult{
		RunID:         runID,
		Scope:         p.scope,
		Tier:          opts.Tier,
		PackageSHA256: digest,
		WorkflowPath:  filepath.Join(p.exampleAbs, "workflows", "workflow.uws.yaml"),
		WorkDir:       workdir,
		DryRun:        opts.DryRun,
	}
	browserConfig, err := buildBrowserRunConfigFromSnapshot(validated.snapshot, opts.BrowserDriver, opts.BrowserDriverArgs, opts.Env, opts.DryRun)
	if err != nil {
		return nil, err
	}
	if err := authorizeBrowserRegistration(validated.snapshot, browserConfig, digest, opts.RegistrationAttestationPath, opts.RegistrationSubmitApproval, p.repoRoot, now); err != nil {
		return nil, err
	}
	runConfig, err := buildRunConfig(p, manifest, validated.snapshot, digest, opts.Tier, result.WorkDir, runID, validated.handoffSHA256, evidencefile.SHA256(approvalBytes), browserConfig)
	if err != nil {
		return nil, err
	}
	runConfigPath, runConfigBytes, err := writeRunConfig(runConfig)
	if err != nil {
		return nil, err
	}
	result.RunConfigPath = runConfigPath
	runConfigDigest := evidencefile.SHA256(runConfigBytes)
	if opts.DryRun {
		prepared, err := udonrunner.Prepare(ctx, runConfig, udonrunner.Options{
			RepoRoot: p.repoRoot,
			Env:      opts.Env,
		})
		if err != nil {
			return nil, fmt.Errorf("prepare trusted executor dry-run: %w", err)
		}
		result.StagePath = prepared.StagePath
		evidencePath, asyncEvidencePath, err := writeRunEvidenceWithAsync(result.WorkDir, runEvidenceOptions{
			Config:          runConfig,
			Approval:        approval,
			Prepared:        prepared,
			Result:          result,
			Mode:            "dry-run",
			StageKind:       "dry-run",
			ExecutorStatus:  "",
			Now:             now,
			RunConfigSHA256: runConfigDigest,
			SigningKeyPEM:   signingKeyPEM,
		})
		if err != nil {
			return result, fmt.Errorf("write dry-run evidence: %w", err)
		}
		result.RunEvidencePath = evidencePath
		result.AsyncEvidencePath = asyncEvidencePath
		return result, nil
	}

	runnerPath := strings.TrimSpace(opts.RunnerPath)
	if runnerPath != "" {
		if err := validateRunnerPath("OPENUDON_UDON_RUNNER", runnerPath); err != nil {
			return nil, err
		}
		prepared, err := udonrunner.Prepare(ctx, runConfig, udonrunner.Options{
			RepoRoot:                p.repoRoot,
			Env:                     opts.Env,
			RequireCredentialValues: true,
		})
		if err != nil {
			return nil, fmt.Errorf("prepare trusted executor: %w", err)
		}
		result.StagePath = prepared.StagePath
		prepared.ExecutorReportPath, err = externalExecutorReportPath(runConfig)
		if err != nil {
			return nil, fmt.Errorf("prepare external executor report path: %w", err)
		}
		args := []string{"--config", runConfigPath, "--config-sha256", runConfigDigest, "--approval", opts.ApprovalPath}
		executorArgv := append([]string{runnerPath}, args...)
		invocation := udonrunner.Invocation{Argv: executorArgv, Dir: p.repoRoot, Env: outerRunnerEnvironment(opts.Env, runConfig, opts.RegistrationAttestationPath, opts.RegistrationSubmitApproval)}
		invoke := opts.Invoke
		if invoke == nil {
			invoke = func(ctx context.Context, invocation udonrunner.Invocation) error {
				return processgroup.RunContext(ctx, processgroup.Invocation{
					Args: invocation.Argv, Dir: invocation.Dir, Env: invocation.Env,
					Stdout: opts.Stdout, Stderr: opts.Stderr,
				})
			}
		}
		if err := invoke(ctx, invocation); err != nil {
			evidencePath, asyncEvidencePath, evidenceErr := writeRunEvidenceWithAsync(result.WorkDir, runEvidenceOptions{
				Config:          runConfig,
				Approval:        approval,
				Prepared:        prepared,
				Result:          result,
				Invoked:         true,
				Mode:            "external-runner",
				RunnerPath:      runnerPath,
				ExecutorArgv:    executorArgv,
				StageKind:       "preflight",
				ExecutorStatus:  "fail",
				Now:             now,
				RunConfigSHA256: runConfigDigest,
				SigningKeyPEM:   signingKeyPEM,
			})
			if evidenceErr != nil {
				return result, fmt.Errorf("run trusted executor: %w; write run evidence: %v", err, evidenceErr)
			}
			result.RunEvidencePath = evidencePath
			result.AsyncEvidencePath = asyncEvidencePath
			return result, fmt.Errorf("run trusted executor: %w", err)
		}
		evidencePath, asyncEvidencePath, err := writeRunEvidenceWithAsync(result.WorkDir, runEvidenceOptions{
			Config:          runConfig,
			Approval:        approval,
			Prepared:        prepared,
			Result:          result,
			Invoked:         true,
			Mode:            "external-runner",
			RunnerPath:      runnerPath,
			ExecutorArgv:    executorArgv,
			StageKind:       "preflight",
			ExecutorStatus:  "pass",
			Now:             now,
			RunConfigSHA256: runConfigDigest,
			SigningKeyPEM:   signingKeyPEM,
		})
		if err != nil {
			return result, fmt.Errorf("write run evidence after successful external runner invocation: %w", err)
		}
		result.RunEvidencePath = evidencePath
		result.AsyncEvidencePath = asyncEvidencePath
		return result, nil
	}
	prepared, err := udonrunner.Run(ctx, runConfig, udonrunner.Options{
		RepoRoot: p.repoRoot,
		Env:      opts.Env,
		Stdout:   opts.Stdout,
		Stderr:   opts.Stderr,
		Invoke:   opts.Invoke,
	})
	if err != nil {
		result.StagePath = prepared.StagePath
		evidencePath, asyncEvidencePath, evidenceErr := writeRunEvidenceWithAsync(result.WorkDir, runEvidenceOptions{
			Config:          runConfig,
			Approval:        approval,
			Prepared:        prepared,
			Result:          result,
			Invoked:         true,
			Mode:            "internal-runner",
			StageKind:       "executor",
			ExecutorStatus:  "fail",
			Now:             now,
			RunConfigSHA256: runConfigDigest,
			SigningKeyPEM:   signingKeyPEM,
		})
		if evidenceErr != nil {
			return result, fmt.Errorf("run trusted executor: %w; write run evidence: %v", err, evidenceErr)
		}
		result.RunEvidencePath = evidencePath
		result.AsyncEvidencePath = asyncEvidencePath
		return result, fmt.Errorf("run trusted executor: %w", err)
	}
	result.StagePath = prepared.StagePath
	evidencePath, asyncEvidencePath, err := writeRunEvidenceWithAsync(result.WorkDir, runEvidenceOptions{
		Config:          runConfig,
		Approval:        approval,
		Prepared:        prepared,
		Result:          result,
		Invoked:         true,
		Mode:            "internal-runner",
		StageKind:       "executor",
		ExecutorStatus:  "pass",
		Now:             now,
		RunConfigSHA256: runConfigDigest,
		SigningKeyPEM:   signingKeyPEM,
	})
	if err != nil {
		return result, fmt.Errorf("write run evidence after successful executor invocation: %w", err)
	}
	result.RunEvidencePath = evidencePath
	result.AsyncEvidencePath = asyncEvidencePath
	return result, nil
}

func validateRunnerPath(name, path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("%s must be an absolute path: %s", name, path)
	}
	if !executablefile.Is(path) {
		return fmt.Errorf("%s does not point to an executable file: %s", name, path)
	}
	return nil
}

func buildRunConfig(p paths, manifest handoffManifest, snapshot packageSnapshot, digest, tier, workdir, runID, handoffDigest, approvalDigest string, browser *udonrunner.BrowserConfig) (RunConfig, error) {
	var relOpenAPI []string
	for _, path := range snapshot.paths {
		if sourcecatalog.IsAPIPath(path) && !packageartifacts.IsAdvisorySecuritySidecarPath(path) {
			relOpenAPI = append(relOpenAPI, path)
		}
	}
	packagePaths := append([]string(nil), snapshot.paths...)
	config := RunConfig{
		Version:               RunConfigVersion,
		RunID:                 runID,
		Scope:                 p.scope,
		Tier:                  tier,
		PackageRoot:           p.exampleAbs,
		WorkDir:               workdir,
		WorkflowPath:          filepath.ToSlash(filepath.Join("workflows", "workflow.uws.yaml")),
		WorkflowFormat:        "uws-yaml",
		DataFiles:             runConfigDataFiles(snapshot),
		APISourcePaths:        relOpenAPI,
		OpenAPIPaths:          relOpenAPI,
		PackagePaths:          packagePaths,
		PackageSHA256:         digest,
		HandoffSHA256:         handoffDigest,
		ApprovalSHA256:        approvalDigest,
		ExecutorReportVersion: udonreport.VersionV3,
		CredentialBindings:    sortedCredentialBindings(manifest),
		Browser:               cloneBrowserConfig(browser),
		DirectProductionRun:   false,
	}
	if config.WorkDir == "" {
		config.WorkDir = p.defaultWorkDir
	}
	return config, nil
}

func runConfigDataFiles(snapshot packageSnapshot) []string {
	if _, ok := snapshot.files[packageartifacts.RuntimeDataPath]; ok {
		return []string{packageartifacts.RuntimeDataPath}
	}
	return nil
}

func writeRunConfig(config RunConfig) (string, []byte, error) {
	if strings.TrimSpace(config.WorkDir) == "" {
		return "", nil, fmt.Errorf("run config workdir is required")
	}
	if err := os.MkdirAll(config.WorkDir, 0o755); err != nil {
		return "", nil, err
	}
	path := filepath.Join(config.WorkDir, "run-config.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", nil, err
	}
	data = append(data, '\n')
	if err := atomicfile.Write(path, data, 0o600); err != nil {
		return "", nil, err
	}
	return path, data, nil
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
	if err := atomicfile.Write(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func VerifyRunEvidenceFile(path string) (VerifyRunEvidenceResult, error) {
	return VerifyRunEvidenceFileWithOptions(path, VerifyRunEvidenceOptions{})
}

func VerifyRunEvidenceFileWithOptions(path string, opts VerifyRunEvidenceOptions) (VerifyRunEvidenceResult, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return VerifyRunEvidenceResult{}, fmt.Errorf("--file is required")
	}
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		return VerifyRunEvidenceResult{}, fmt.Errorf("read run evidence: %w", err)
	}
	var evidence RunEvidence
	if err := evidencefile.DecodeStrict(data, &evidence); err != nil {
		return VerifyRunEvidenceResult{}, fmt.Errorf("run evidence must be valid JSON: %w", err)
	}
	if err := validateRunEvidenceForVerify(evidence); err != nil {
		return VerifyRunEvidenceResult{}, err
	}
	if evidence.Version == LegacyRunEvidenceVersion && (opts.RequireSignature || strings.TrimSpace(opts.TrustedPublicKey) != "") {
		return VerifyRunEvidenceResult{}, fmt.Errorf("legacy run evidence cannot carry a v0.2 signature")
	}
	if evidence.Version == RunEvidenceVersion {
		if err := verifyRunEvidenceSignature(path, data, opts); err != nil {
			return VerifyRunEvidenceResult{}, err
		}
	}
	seenAsyncPaths := map[string]bool{}
	workdir := filepath.Dir(path)
	for _, ref := range evidence.AsyncEvidenceFiles {
		if seenAsyncPaths[ref.Path] {
			return VerifyRunEvidenceResult{}, fmt.Errorf("duplicate async evidence path: %s", ref.Path)
		}
		seenAsyncPaths[ref.Path] = true
		if err := verifyAsyncEvidenceFile(workdir, ref); err != nil {
			return VerifyRunEvidenceResult{}, err
		}
	}
	if evidence.Version == RunEvidenceVersion {
		requireSuccessfulReport := !evidence.DryRun &&
			(evidence.Executor.Mode == "internal-runner" || evidence.Executor.Mode == "external-runner") &&
			evidenceGateStatus(evidence, "executor_invocation") == "pass"
		if err := verifyExecutorReport(workdir, evidence.Executor, evidence.Browser, requireSuccessfulReport); err != nil {
			return VerifyRunEvidenceResult{}, err
		}
	}
	return VerifyRunEvidenceResult{
		RunEvidencePath:    path,
		AsyncEvidenceFiles: append([]RunEvidenceAsyncFile(nil), evidence.AsyncEvidenceFiles...),
	}, nil
}

func validateRunEvidenceForVerify(evidence RunEvidence) error {
	if evidence.Version != RunEvidenceVersion && evidence.Version != LegacyRunEvidenceVersion {
		return fmt.Errorf("run evidence version must be %s or read-only legacy %s", RunEvidenceVersion, LegacyRunEvidenceVersion)
	}
	if strings.TrimSpace(evidence.Scope) == "" {
		return fmt.Errorf("run evidence scope is required")
	}
	if strings.TrimSpace(evidence.PackageSHA256) == "" {
		return fmt.Errorf("run evidence package_sha256 is required")
	}
	if strings.TrimSpace(evidence.RunConfigPath) == "" {
		return fmt.Errorf("run evidence run_config_path is required")
	}
	if strings.TrimSpace(evidence.WorkDir) == "" {
		return fmt.Errorf("run evidence workdir is required")
	}
	if evidence.Version == RunEvidenceVersion {
		if strings.TrimSpace(evidence.RunID) == "" || !evidencefile.ValidSHA256(evidence.PackageSHA256) ||
			!evidencefile.ValidSHA256(evidence.HandoffSHA256) || !evidencefile.ValidSHA256(evidence.ApprovalSHA256) ||
			!evidencefile.ValidSHA256(evidence.RunConfigSHA256) {
			return fmt.Errorf("run evidence v2 requires run_id and full package, handoff, approval, and config SHA-256 digests")
		}
		if err := validateRunEvidenceGates(evidence); err != nil {
			return err
		}
		if err := udonrunner.ValidateBrowserEvidenceConfig(evidence.Browser, evidence.CredentialBindings); err != nil {
			return fmt.Errorf("run evidence browser contract is invalid: %w", err)
		}
	}
	return nil
}

func validateRunEvidenceGates(evidence RunEvidence) error {
	statuses := make(map[string]string, len(evidence.Gates))
	for _, gate := range evidence.Gates {
		name := strings.TrimSpace(gate.Name)
		status := strings.TrimSpace(gate.Status)
		if name == "" || (status != "pass" && status != "fail") {
			return fmt.Errorf("run evidence gate names are required and statuses must be pass or fail")
		}
		if _, exists := statuses[name]; exists {
			return fmt.Errorf("duplicate run evidence gate: %s", name)
		}
		statuses[name] = status
	}
	for _, name := range []string{"handoff_package", "manifest_policy", "stored_quality", "current_quality", "approval", "run_config", "staged_digest"} {
		if statuses[name] != "pass" {
			return fmt.Errorf("run evidence requires passing %s gate", name)
		}
	}
	executorStatus, hasExecutorGate := statuses["executor_invocation"]
	if evidence.DryRun {
		if hasExecutorGate || evidence.Executor.Invoked || evidence.Executor.Mode != "dry-run" || evidence.StageKind != "dry-run" {
			return fmt.Errorf("dry-run evidence must use the non-invoked dry-run execution posture")
		}
		return nil
	}
	if !hasExecutorGate || (executorStatus != "pass" && executorStatus != "fail") {
		return fmt.Errorf("non-dry-run evidence requires an executor_invocation gate")
	}
	if !evidence.Executor.Invoked {
		return fmt.Errorf("non-dry-run evidence must record an invoked executor")
	}
	switch evidence.Executor.Mode {
	case "internal-runner":
		if evidence.StageKind != "executor" {
			return fmt.Errorf("internal-runner evidence must use the executor stage")
		}
	case "external-runner":
		if evidence.StageKind != "preflight" {
			return fmt.Errorf("external-runner evidence must use the preflight stage")
		}
	default:
		return fmt.Errorf("non-dry-run evidence has unsupported executor mode %q", evidence.Executor.Mode)
	}
	return nil
}

func verifyExecutorReport(workdir string, executor RunEvidenceExecutor, browser *udonrunner.BrowserConfig, requiredSuccess bool) error {
	if strings.TrimSpace(executor.ReportPath) == "" {
		if executor.ReportSHA256 != "" || executor.ReportSize != 0 {
			return fmt.Errorf("executor report digest and size require report_path")
		}
		if requiredSuccess {
			return fmt.Errorf("successful executor evidence requires an executor report")
		}
		return nil
	}
	clean, err := packageartifacts.CleanRelativePath(executor.ReportPath)
	if err != nil || clean != executor.ReportPath {
		return fmt.Errorf("executor report path must be safe workdir-relative path: %q", executor.ReportPath)
	}
	data, info, err := evidencefile.ReadRegular(filepath.Join(workdir, filepath.FromSlash(clean)), evidencefile.DefaultMaxBytes)
	if err != nil {
		return fmt.Errorf("read executor report %s: %w", clean, err)
	}
	if info.Size() != executor.ReportSize {
		return fmt.Errorf("executor report size mismatch for %s", clean)
	}
	if evidencefile.SHA256(data) != executor.ReportSHA256 {
		return fmt.Errorf("executor report digest mismatch for %s", clean)
	}
	report, err := decodeUdonExecutionReport(data)
	if err != nil {
		return fmt.Errorf("executor report %s is invalid: %w", clean, err)
	}
	if requiredSuccess && !strings.EqualFold(report.Status, "success") {
		return fmt.Errorf("successful executor evidence requires a success report status")
	}
	if browser != nil && browser.Protocol == "v4" && report.Version != udonreport.VersionV3 {
		return fmt.Errorf("browser registration evidence requires udon.execution-report.v3")
	}
	return nil
}

func evidenceGateStatus(evidence RunEvidence, name string) string {
	for _, gate := range evidence.Gates {
		if gate.Name == name {
			return gate.Status
		}
	}
	return ""
}

func verifyAsyncEvidenceFile(workdir string, ref RunEvidenceAsyncFile) error {
	clean, err := packageartifacts.CleanRelativePath(ref.Path)
	if err != nil || clean != ref.Path {
		return fmt.Errorf("async evidence path must be safe workdir-relative path: %q", ref.Path)
	}
	if ref.Records < 1 {
		return fmt.Errorf("async evidence record count must be positive for %s", ref.Path)
	}
	if strings.TrimSpace(ref.Digest) == "" {
		return fmt.Errorf("async evidence digest is required for %s", ref.Path)
	}
	if strings.TrimSpace(ref.Purpose) != "openudon_run_async_execution_forwarding" {
		return fmt.Errorf("async evidence purpose is invalid for %s", ref.Path)
	}
	path := filepath.Join(workdir, filepath.FromSlash(ref.Path))
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		return fmt.Errorf("read async evidence %s: %w", ref.Path, err)
	}
	if got := evdigest.SHA256Bytes(data).String(); got != ref.Digest {
		return fmt.Errorf("async evidence digest mismatch for %s: got %s want %s", ref.Path, got, ref.Digest)
	}
	var bundle AsyncEvidenceBundle
	if err := evidencefile.DecodeStrict(data, &bundle); err != nil {
		return fmt.Errorf("async evidence %s must be valid JSON: %w", ref.Path, err)
	}
	if len(bundle.Records) != ref.Records {
		return fmt.Errorf("async evidence record count mismatch for %s: got %d want %d", ref.Path, len(bundle.Records), ref.Records)
	}
	if err := validateAsyncEvidenceBundle(bundle); err != nil {
		return fmt.Errorf("async evidence %s is invalid: %w", ref.Path, err)
	}
	return nil
}

func writeRunEvidenceWithAsync(workdir string, opts runEvidenceOptions) (string, string, error) {
	evidence, err := buildRunEvidence(opts)
	if err != nil {
		return "", "", err
	}
	bundle, err := buildAsyncEvidenceBundle(opts)
	if err != nil {
		return "", "", err
	}
	ref, err := writeAsyncEvidenceBundle(workdir, bundle)
	if err != nil {
		return "", "", err
	}
	evidence.AsyncEvidenceFiles = []RunEvidenceAsyncFile{ref}
	evidencePath, err := writeRunEvidence(workdir, evidence)
	if err != nil {
		return "", "", err
	}
	if len(opts.SigningKeyPEM) != 0 {
		if _, err := signRunEvidence(evidencePath, opts.SigningKeyPEM); err != nil {
			return "", "", fmt.Errorf("sign run evidence: %w", err)
		}
	}
	return evidencePath, filepath.Join(workdir, ref.Path), nil
}

func writeAsyncEvidenceBundle(workdir string, bundle AsyncEvidenceBundle) (RunEvidenceAsyncFile, error) {
	if strings.TrimSpace(workdir) == "" {
		return RunEvidenceAsyncFile{}, fmt.Errorf("async evidence workdir is required")
	}
	if err := validateAsyncEvidenceBundle(bundle); err != nil {
		return RunEvidenceAsyncFile{}, err
	}
	if err := os.MkdirAll(workdir, 0o755); err != nil {
		return RunEvidenceAsyncFile{}, err
	}
	path := filepath.Join(workdir, "async-evidence.json")
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return RunEvidenceAsyncFile{}, err
	}
	data = append(data, '\n')
	if err := atomicfile.Write(path, data, 0o600); err != nil {
		return RunEvidenceAsyncFile{}, err
	}
	return RunEvidenceAsyncFile{
		Path:    filepath.Base(path),
		Digest:  evdigest.SHA256Bytes(data).String(),
		Records: len(bundle.Records),
		Purpose: "openudon_run_async_execution_forwarding",
	}, nil
}

type runEvidenceOptions struct {
	Config          RunConfig
	Approval        Approval
	Prepared        udonrunner.Result
	Result          *RunResult
	Invoked         bool
	Mode            string
	RunnerPath      string
	ExecutorArgv    []string
	StageKind       string
	ExecutorStatus  string
	Now             time.Time
	RunConfigSHA256 string
	SigningKeyPEM   []byte
}

func buildRunEvidence(opts runEvidenceOptions) (RunEvidence, error) {
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
	executor, err := buildRunEvidenceExecutor(opts, executorArgv)
	if err != nil {
		return RunEvidence{}, err
	}
	return RunEvidence{
		Version:            RunEvidenceVersion,
		RunID:              opts.Result.RunID,
		CreatedAt:          opts.Now.UTC().Format(time.RFC3339),
		Scope:              opts.Result.Scope,
		Tier:               opts.Result.Tier,
		DryRun:             opts.Result.DryRun,
		ApprovalState:      opts.Approval.State,
		PackageSHA256:      opts.Result.PackageSHA256,
		HandoffSHA256:      opts.Config.HandoffSHA256,
		ApprovalSHA256:     opts.Config.ApprovalSHA256,
		RunConfigSHA256:    opts.RunConfigSHA256,
		RunConfigPath:      opts.Result.RunConfigPath,
		PackageRoot:        opts.Config.PackageRoot,
		WorkDir:            opts.Result.WorkDir,
		StageKind:          opts.StageKind,
		StagePath:          opts.Prepared.StagePath,
		WorkflowPath:       opts.Prepared.WorkflowPath,
		PackagePaths:       append([]string(nil), opts.Prepared.PackagePaths...),
		APISourcePaths:     append([]string(nil), opts.Prepared.APISourcePaths...),
		CredentialBindings: append([]string(nil), opts.Config.CredentialBindings...),
		CredentialEnvNames: append([]string(nil), opts.Prepared.CredentialEnvNames...),
		Browser:            cloneBrowserConfig(opts.Config.Browser),
		Gates:              gates,
		Executor:           executor,
	}, nil
}

func buildRunEvidenceExecutor(opts runEvidenceOptions, executorArgv []string) (RunEvidenceExecutor, error) {
	executor := RunEvidenceExecutor{
		Invoked:    opts.Invoked,
		Mode:       opts.Mode,
		RunnerPath: opts.RunnerPath,
		Argv:       executorArgv,
	}
	requiredSuccess := opts.Invoked && opts.ExecutorStatus == "pass" &&
		(opts.Mode == "internal-runner" || opts.Mode == "external-runner") && opts.Result != nil && !opts.Result.DryRun
	reportPath := strings.TrimSpace(opts.Prepared.ExecutorReportPath)
	if reportPath == "" {
		if requiredSuccess {
			return RunEvidenceExecutor{}, fmt.Errorf("successful executor did not provide an executor report path")
		}
		return executor, nil
	}
	rel, err := filepath.Rel(opts.Result.WorkDir, reportPath)
	if err != nil || rel == "." || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		if requiredSuccess {
			return RunEvidenceExecutor{}, fmt.Errorf("successful executor report path is not workdir-relative")
		}
		return executor, nil
	}
	data, info, err := evidencefile.ReadRegular(reportPath, evidencefile.DefaultMaxBytes)
	if err != nil {
		if requiredSuccess {
			return RunEvidenceExecutor{}, fmt.Errorf("read successful executor report: %w", err)
		}
		return executor, nil
	}
	if requiredSuccess {
		report, err := decodeUdonExecutionReport(data)
		if err != nil {
			return RunEvidenceExecutor{}, fmt.Errorf("validate successful executor report: %w", err)
		}
		if !strings.EqualFold(report.Status, "success") {
			return RunEvidenceExecutor{}, fmt.Errorf("successful executor report status must be success")
		}
		if opts.Config.Browser != nil && opts.Config.Browser.Protocol == "v4" && report.Version != opts.Config.ExecutorReportVersion {
			return RunEvidenceExecutor{}, fmt.Errorf("successful executor report version does not match the run config")
		}
	}
	executor.ReportPath = filepath.ToSlash(rel)
	executor.ReportSHA256 = evidencefile.SHA256(data)
	executor.ReportSize = info.Size()
	return executor, nil
}

func buildAsyncEvidenceBundle(opts runEvidenceOptions) (AsyncEvidenceBundle, error) {
	requestEvidenceID := asyncEvidenceID(opts, "request")
	responseEvidenceID := asyncEvidenceID(opts, "response")
	attemptID := asyncAttemptID(opts)
	operation := asyncOperationRef(opts)
	argv := asyncExecutorArgv(opts)
	metadata := map[string]string{
		"approval_state":     opts.Approval.State,
		"run_config_version": opts.Config.Version,
	}
	if len(argv) > 0 {
		if data, err := json.Marshal(argv); err == nil {
			metadata["argv_json"] = string(data)
		}
	}
	if strings.TrimSpace(opts.RunnerPath) != "" {
		metadata["runner_path"] = strings.TrimSpace(opts.RunnerPath)
	}
	if strings.TrimSpace(opts.Prepared.StagePath) != "" {
		metadata["stage_path"] = strings.TrimSpace(opts.Prepared.StagePath)
	}
	outcome := "accepted"
	errorSummary := ""
	if opts.ExecutorStatus == "fail" {
		outcome = "fatal_failure"
		errorSummary = strings.TrimSpace(opts.Mode + " invocation failed")
	}
	recordedAt := opts.Now.UTC()
	request := asyncevidence.NormalizeExecutionRequest(asyncevidence.ExecutionRequest{
		Version: asyncevidence.ExecutionRequestVersion,
		Attempt: asyncevidence.AttemptMetadata{
			EvidenceID: requestEvidenceID,
			AttemptID:  attemptID,
			Sequence:   1,
			Actor:      opts.Approval.Reviewer,
			Source:     "openudon.trustedrunner",
			RecordedAt: recordedAt,
		},
		RequestID: opts.Result.PackageSHA256,
		Operation: operation,
		Transport: map[string]string{
			"runner_mode":     opts.Mode,
			"stage_kind":      opts.StageKind,
			"tier":            opts.Result.Tier,
			"dry_run":         fmt.Sprint(opts.Result.DryRun),
			"run_config_path": opts.Result.RunConfigPath,
		},
		Digests: []evdigest.Record{
			{Algorithm: evdigest.AlgorithmSHA256, Value: opts.Result.PackageSHA256},
		},
		Metadata: metadata,
	})
	response := asyncevidence.NormalizeExecutionResponse(asyncevidence.ExecutionResponse{
		Version: asyncevidence.ExecutionResponseVersion,
		Attempt: asyncevidence.AttemptMetadata{
			EvidenceID: responseEvidenceID,
			AttemptID:  attemptID,
			Sequence:   2,
			Actor:      opts.Approval.Reviewer,
			Source:     "openudon.trustedrunner",
			RecordedAt: recordedAt,
		},
		RequestEvidenceID: requestEvidenceID,
		ResponseID:        asyncEvidenceID(opts, "executor-result"),
		Operation:         operation,
		Outcome:           outcome,
		ErrorSummary:      errorSummary,
		FinishedAt:        recordedAt,
	})
	records := []AsyncEvidenceRecord{
		{Kind: "execution_request", ExecutionRequest: &request},
		{Kind: "execution_response", ExecutionResponse: &response},
	}
	observations, err := asyncExecutionReportRecords(opts, requestEvidenceID, attemptID, operation, len(records)+1)
	if err != nil {
		return AsyncEvidenceBundle{}, err
	}
	records = append(records, observations...)
	return AsyncEvidenceBundle{
		Version: AsyncEvidenceVersion,
		Records: records,
	}, nil
}

func validateAsyncEvidenceBundle(bundle AsyncEvidenceBundle) error {
	if bundle.Version != AsyncEvidenceVersion {
		return fmt.Errorf("async evidence bundle version must be %s", AsyncEvidenceVersion)
	}
	if len(bundle.Records) == 0 {
		return fmt.Errorf("async evidence bundle requires records")
	}
	for i, record := range bundle.Records {
		switch record.Kind {
		case "execution_request":
			if record.ExecutionRequest == nil {
				return fmt.Errorf("async evidence record %d missing execution_request", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for execution_request", i)
			}
			if diagnostics := asyncevidence.ValidateExecutionRequest(*record.ExecutionRequest); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence request record %d is invalid: %s", i, diagnostics[0].Code)
			}
		case "execution_response":
			if record.ExecutionResponse == nil {
				return fmt.Errorf("async evidence record %d missing execution_response", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for execution_response", i)
			}
			if diagnostics := asyncevidence.ValidateExecutionResponse(*record.ExecutionResponse); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence response record %d is invalid: %s", i, diagnostics[0].Code)
			}
		case "status_observation":
			if record.StatusObservation == nil {
				return fmt.Errorf("async evidence record %d missing status_observation", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for status_observation", i)
			}
			if diagnostics := asyncevidence.ValidateStatusObservation(*record.StatusObservation); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence status record %d is invalid: %s", i, diagnostics[0].Code)
			}
		case "confirmation_read_observation":
			if record.ConfirmationReadObservation == nil {
				return fmt.Errorf("async evidence record %d missing confirmation_read_observation", i)
			}
			if asyncRecordPayloadCount(record) != 1 {
				return fmt.Errorf("async evidence record %d has unexpected payload for confirmation_read_observation", i)
			}
			if diagnostics := asyncevidence.ValidateConfirmationReadObservation(*record.ConfirmationReadObservation); len(diagnostics) != 0 {
				return fmt.Errorf("async evidence confirmation-read record %d is invalid: %s", i, diagnostics[0].Code)
			}
		default:
			return fmt.Errorf("unsupported async evidence record kind %q", record.Kind)
		}
	}
	return nil
}

func asyncRecordPayloadCount(record AsyncEvidenceRecord) int {
	count := 0
	if record.ExecutionRequest != nil {
		count++
	}
	if record.ExecutionResponse != nil {
		count++
	}
	if record.StatusObservation != nil {
		count++
	}
	if record.ConfirmationReadObservation != nil {
		count++
	}
	return count
}

func asyncExecutionReportRecords(opts runEvidenceOptions, requestEvidenceID, attemptID string, operation asyncevidence.OperationRef, sequenceStart int) ([]AsyncEvidenceRecord, error) {
	report, err := readUdonExecutionReport(opts.Prepared.ExecutorReportPath)
	if err != nil || report == nil {
		return nil, err
	}
	observedAt, err := reportTime(report.FinishedAt, opts.Now)
	if err != nil {
		return nil, err
	}
	digests, err := reportOutputDigests(report.OutputDigest)
	if err != nil {
		return nil, err
	}
	status := asyncevidence.NormalizeStatusObservation(asyncevidence.StatusObservation{
		Version: asyncevidence.StatusObservationVersion,
		Attempt: asyncevidence.AttemptMetadata{
			EvidenceID: asyncEvidenceID(opts, "executor-status"),
			AttemptID:  attemptID,
			Sequence:   int64(sequenceStart),
			Actor:      opts.Approval.Reviewer,
			Source:     "udon.execution-report",
			RecordedAt: observedAt,
		},
		RequestEvidenceID: requestEvidenceID,
		Operation:         operation,
		Status:            report.Status,
		TerminalityHint:   "terminal",
		PayloadDigests:    digests,
		ObservedAt:        observedAt,
	})
	records := []AsyncEvidenceRecord{{Kind: "status_observation", StatusObservation: &status}}
	if strings.EqualFold(report.Status, "success") && len(digests) > 0 {
		read := asyncevidence.NormalizeConfirmationReadObservation(asyncevidence.ConfirmationReadObservation{
			Version: asyncevidence.ConfirmationReadObservationVersion,
			Attempt: asyncevidence.AttemptMetadata{
				EvidenceID: asyncEvidenceID(opts, "executor-confirmation-read"),
				AttemptID:  attemptID,
				Sequence:   int64(sequenceStart + 1),
				Actor:      opts.Approval.Reviewer,
				Source:     "udon.execution-report",
				RecordedAt: observedAt,
			},
			RequestEvidenceID: requestEvidenceID,
			Operation:         operation,
			Outcome:           "confirmed",
			ProjectedDigests:  digests,
			ObservedAt:        observedAt,
		})
		records = append(records, AsyncEvidenceRecord{Kind: "confirmation_read_observation", ConfirmationReadObservation: &read})
	}
	return records, nil
}

func readUdonExecutionReport(path string) (*UdonExecutionReport, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read udon execution report: %w", err)
	}
	return decodeUdonExecutionReport(data)
}

func decodeUdonExecutionReport(data []byte) (*UdonExecutionReport, error) {
	report, err := udonreport.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("udon execution report is invalid: %w", err)
	}
	return report, nil
}

func reportTime(value string, fallback time.Time) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback.UTC(), nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("udon execution report finished_at must be RFC3339: %w", err)
	}
	return t.UTC(), nil
}

func reportOutputDigests(value string) ([]evdigest.Record, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	algorithm, digestValue, ok := strings.Cut(value, ":")
	digestValue = strings.TrimSpace(digestValue)
	if !ok || strings.TrimSpace(algorithm) != string(evdigest.AlgorithmSHA256) || digestValue != strings.ToLower(digestValue) || !evidencefile.ValidSHA256(digestValue) {
		return nil, fmt.Errorf("udon execution report output_digest must be sha256:<hex>")
	}
	return []evdigest.Record{{Algorithm: evdigest.AlgorithmSHA256, Value: digestValue}}, nil
}

func asyncOperationRef(opts runEvidenceOptions) asyncevidence.OperationRef {
	sourcePath := strings.TrimSpace(opts.Prepared.WorkflowPath)
	operationID := sourcePath
	if operationID == "" {
		operationID = strings.TrimSpace(opts.Config.WorkflowPath)
	}
	if operationID == "" && opts.Result != nil {
		operationID = strings.TrimSpace(opts.Result.Scope)
	}
	subjectID := ""
	if opts.Result != nil {
		subjectID = opts.Result.Scope
	}
	return asyncevidence.NormalizeOperation(asyncevidence.OperationRef{
		SubjectKind: "openudon_package",
		SubjectID:   subjectID,
		Action:      "run",
		SourceKind:  "uws",
		SourcePath:  sourcePath,
		OperationID: operationID,
	})
}

func asyncAttemptID(opts runEvidenceOptions) string {
	scope := strings.ReplaceAll(strings.TrimSpace(opts.Result.Scope), "/", ".")
	if scope == "" {
		scope = "openudon_package"
	}
	suffix := strings.TrimSpace(opts.Result.PackageSHA256)
	if len(suffix) > 12 {
		suffix = suffix[:12]
	}
	if suffix == "" {
		suffix = evdigest.SHA256Bytes([]byte(opts.Result.RunConfigPath)).Value[:12]
	}
	return scope + "." + suffix
}

func asyncEvidenceID(opts runEvidenceOptions, kind string) string {
	return asyncAttemptID(opts) + "." + kind
}

func asyncExecutorArgv(opts runEvidenceOptions) []string {
	argv := append([]string(nil), opts.ExecutorArgv...)
	if len(argv) == 0 {
		argv = append(argv, opts.Prepared.Argv...)
	}
	return argv
}

func resolveRunWorkDir(p paths, input, runID string) (string, error) {
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
	return filepath.Join(filepath.Clean(workdir), "run-"+runID), nil
}

func newRunID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", value[:]), nil
}

func outerRunnerEnvironment(source []string, config RunConfig, registrationAttestationPath, registrationSubmitApproval string) []string {
	if source == nil {
		source = os.Environ()
	}
	values := map[string]string{}
	for _, item := range source {
		name, value, ok := strings.Cut(item, "=")
		if ok {
			values[name] = value
		}
	}
	allowed := map[string]bool{
		"PATH": true, "PATHEXT": true, "SystemRoot": true, "SYSTEMROOT": true,
		"WINDIR": true, "COMSPEC": true, "TMP": true, "TEMP": true,
		"OPENUDON_EXECUTOR": true, "OPENUDON_UDON_BIN": true, "OPENUDON_UDON_IMAGE": true,
	}
	for _, binding := range config.CredentialBindings {
		allowed[udonrunner.CredentialEnvironmentName(binding)] = true
	}
	if config.Browser != nil {
		for _, binding := range config.Browser.SessionEnvironment {
			allowed[binding.Environment] = true
		}
		for _, name := range config.Browser.DriverEnvironment {
			allowed[name] = true
		}
	}
	if strings.TrimSpace(registrationAttestationPath) != "" {
		values["OPENUDON_BROWSER_REGISTRATION_ATTESTATION"] = registrationAttestationPath
		allowed["OPENUDON_BROWSER_REGISTRATION_ATTESTATION"] = true
	}
	if strings.TrimSpace(registrationSubmitApproval) != "" {
		values["OPENUDON_BROWSER_REGISTRATION_SUBMIT_APPROVAL"] = registrationSubmitApproval
		allowed["OPENUDON_BROWSER_REGISTRATION_SUBMIT_APPROVAL"] = true
	}
	var out []string
	for name := range allowed {
		if value, ok := values[name]; ok {
			out = append(out, name+"="+value)
		}
	}
	sort.Strings(out)
	return out
}

func cloneBrowserConfig(input *udonrunner.BrowserConfig) *udonrunner.BrowserConfig {
	if input == nil {
		return nil
	}
	result := *input
	result.DriverArgs = append([]string(nil), input.DriverArgs...)
	result.DriverEnvironment = append([]string(nil), input.DriverEnvironment...)
	result.CredentialEnvironment = append([]udonrunner.EnvironmentBinding(nil), input.CredentialEnvironment...)
	result.SessionEnvironment = append([]udonrunner.EnvironmentBinding(nil), input.SessionEnvironment...)
	result.ApprovedOperations = append([]string(nil), input.ApprovedOperations...)
	result.ApprovedAuthentication = append([]string(nil), input.ApprovedAuthentication...)
	result.ApprovedRegistration = append([]string(nil), input.ApprovedRegistration...)
	result.AttestedRegistration = append([]string(nil), input.AttestedRegistration...)
	return &result
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

func ApprovalTemplate(ctx context.Context, opts TemplateOptions) (Approval, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	validated, err := validatePackage(ctx, packageOptions{
		RepoRoot:   opts.RepoRoot,
		ExampleDir: opts.ExampleDir,
		Assess:     opts.Assess,
	})
	if err != nil {
		return Approval{}, err
	}
	p, manifest, digest := validated.paths, validated.manifest, validated.packageSHA256
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

type validatedPackage struct {
	paths         paths
	manifest      handoffManifest
	snapshot      packageSnapshot
	packageSHA256 string
	handoffSHA256 string
}

// packageSnapshot is the single manifest-bound byte generation used by all
// post-validation parsing and handoff derivation. Files are read once while
// constructing it and are never refreshed in place.
type packageSnapshot struct {
	paths []string
	files map[string][]byte
}

func (snapshot packageSnapshot) read(path string) ([]byte, error) {
	clean, err := packageartifacts.CleanRelativePath(path)
	if err != nil {
		return nil, err
	}
	data, ok := snapshot.files[clean]
	if !ok {
		return nil, fmt.Errorf("package snapshot missing %s", clean)
	}
	return data, nil
}

func readPackageSnapshot(p paths, manifest handoffManifest, handoffBytes []byte) (packageSnapshot, error) {
	paths, err := packageartifacts.RequiredManifestPaths(p.exampleAbs, packageManifestInputs(manifest))
	if err != nil {
		return packageSnapshot{}, err
	}
	if err := packageartifacts.ValidateRegularPackageFiles(p.exampleAbs, paths); err != nil {
		return packageSnapshot{}, err
	}
	snapshot := packageSnapshot{paths: append([]string(nil), paths...), files: make(map[string][]byte, len(paths))}
	for _, path := range paths {
		if path == packageartifacts.ReviewHandoffPath {
			snapshot.files[path] = append([]byte(nil), handoffBytes...)
			continue
		}
		data, _, err := evidencefile.ReadRegular(filepath.Join(p.exampleAbs, filepath.FromSlash(path)), evidencefile.DefaultMaxBytes)
		if err != nil {
			return packageSnapshot{}, fmt.Errorf("read handoff input %s: %w", path, err)
		}
		snapshot.files[path] = data
	}
	return snapshot, nil
}

func validatePackage(ctx context.Context, opts packageOptions) (validatedPackage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	before, err := resolveAndValidatePackageBytes(opts.RepoRoot, opts.ExampleDir)
	if err != nil {
		return validatedPackage{}, err
	}
	assess := opts.Assess
	if assess == nil {
		assess = synthesize.AssessCurrent
	}
	report, err := assess(ctx, synthesize.Options{ExampleDir: before.paths.exampleAbs})
	if err != nil {
		return validatedPackage{}, fmt.Errorf("assess current quality: %w", err)
	}
	if report == nil {
		return validatedPackage{}, fmt.Errorf("assess current quality: assessment returned no report")
	}
	if !report.Passed() {
		return validatedPackage{}, fmt.Errorf("current quality gate is %q", report.Status)
	}
	if err := ctx.Err(); err != nil {
		return validatedPackage{}, err
	}
	after, err := resolveAndValidatePackageBytes(opts.RepoRoot, opts.ExampleDir)
	if err != nil {
		return validatedPackage{}, fmt.Errorf("revalidate package after assessment: %w", err)
	}
	if before.paths.scope != after.paths.scope || before.packageSHA256 != after.packageSHA256 || before.handoffSHA256 != after.handoffSHA256 {
		return validatedPackage{}, fmt.Errorf("reviewed package changed during current-state assessment")
	}
	return after, nil
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

func readHandoff(path string) (handoffManifest, []byte, error) {
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		return handoffManifest{}, nil, fmt.Errorf("read handoff manifest: %w", err)
	}
	var manifest handoffManifest
	if err := evidencefile.DecodeStrict(data, &manifest); err != nil {
		return handoffManifest{}, nil, fmt.Errorf("handoff manifest must be valid JSON: %w", err)
	}
	if manifest.Version == authoring.LegacyReviewHandoffVersion {
		return handoffManifest{}, nil, fmt.Errorf("legacy handoff %s is read-only and cannot execute; regenerate the package with openudon build", manifest.Version)
	}
	allowedVersions := []string{ReviewHandoffVersion}
	if diagnostics := authoring.ValidateReviewHandoff(manifest, authoring.ReviewHandoffValidationOptions{AllowedVersions: allowedVersions}); len(diagnostics) > 0 {
		return handoffManifest{}, nil, fmt.Errorf("handoff manifest is invalid: %s", diagnostics[0].Message)
	}
	return manifest, data, nil
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

func validateRequiredSnapshotInputs(manifest handoffManifest, snapshot packageSnapshot) error {
	for _, input := range manifest.HandoffInputs {
		if !input.Required {
			continue
		}
		clean, err := packageartifacts.CleanRelativePath(input.Path)
		if err != nil {
			return err
		}
		var got string
		if clean == packageartifacts.ReviewHandoffPath {
			got, err = authoring.ReviewHandoffSelfDigest(manifest, clean)
		} else {
			var data []byte
			data, err = snapshot.read(clean)
			if err == nil {
				got = evidencefile.SHA256(data)
			}
		}
		if err != nil {
			return fmt.Errorf("verify handoff input %s: %w", clean, err)
		}
		if got != strings.ToLower(strings.TrimSpace(input.SHA256)) {
			return fmt.Errorf("handoff input SHA-256 mismatch for %s", clean)
		}
	}
	return nil
}

func validateStoredQualityBytes(data []byte) error {
	var report synthesize.QualityReport
	if err := evidencefile.DecodeStrict(data, &report); err != nil {
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

func computePackageSnapshotDigest(p paths, manifest handoffManifest, snapshot packageSnapshot) (string, error) {
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
	inputs := make([]authoring.ReviewHandoffInput, 0, len(snapshot.paths))
	for _, path := range snapshot.paths {
		input, ok := manifestInputByPath[path]
		if !ok {
			return "", fmt.Errorf("snapshot path is not a required handoff input: %s", path)
		}
		inputs = append(inputs, input)
	}
	return authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Root: p.exampleAbs, Scope: p.scope, Version: "openudon.handoff-package-digest.v1",
		Inputs: inputs, InputBytes: snapshot.files,
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

func readApprovalDocument(path string) (Approval, []byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Approval{}, nil, fmt.Errorf("--approval is required")
	}
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		return Approval{}, nil, fmt.Errorf("read approval: %w", err)
	}
	var approval Approval
	if err := evidencefile.DecodeStrict(data, &approval); err != nil {
		return Approval{}, nil, fmt.Errorf("approval must be valid JSON: %w", err)
	}
	return approval, data, nil
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
