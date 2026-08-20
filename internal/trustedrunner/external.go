package trustedrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

// ExternalOptions describes the deliberately narrow udon-runner boundary.
// The config and approval files are both digest-pinned by the parent process,
// then revalidated against current package state before any executor starts.
type ExternalOptions struct {
	ConfigPath   string
	ConfigSHA256 string
	ApprovalPath string
	Env          []string
	Stdout       io.Writer
	Stderr       io.Writer
	Now          func() time.Time
	Assess       func(context.Context, synthesize.Options) (*synthesize.QualityReport, error)
	Invoke       udonrunner.InvokeFunc
}

func RunExternal(ctx context.Context, opts ExternalOptions) (udonrunner.Result, error) {
	configBytes, _, err := evidencefile.ReadRegular(opts.ConfigPath, evidencefile.DefaultMaxBytes)
	if err != nil {
		return udonrunner.Result{}, fmt.Errorf("read run config: %w", err)
	}
	wantConfigDigest := strings.ToLower(strings.TrimSpace(opts.ConfigSHA256))
	if !evidencefile.ValidSHA256(wantConfigDigest) {
		return udonrunner.Result{}, fmt.Errorf("--config-sha256 must be a full SHA-256 digest")
	}
	if got := evidencefile.SHA256(configBytes); got != wantConfigDigest {
		return udonrunner.Result{}, fmt.Errorf("run config SHA-256 mismatch")
	}
	var config RunConfig
	if err := evidencefile.DecodeStrict(configBytes, &config); err != nil {
		return udonrunner.Result{}, fmt.Errorf("run config must be valid JSON: %w", err)
	}
	if config.Version == udonrunner.LegacyRunConfigVersion {
		return udonrunner.Result{}, fmt.Errorf("legacy run config %s cannot execute", config.Version)
	}
	if config.Version != RunConfigVersion {
		return udonrunner.Result{}, fmt.Errorf("run config version must be %s", RunConfigVersion)
	}
	approval, approvalBytes, err := readApprovalDocument(opts.ApprovalPath)
	if err != nil {
		return udonrunner.Result{}, err
	}
	if evidencefile.SHA256(approvalBytes) != config.ApprovalSHA256 {
		return udonrunner.Result{}, fmt.Errorf("approval SHA-256 does not match the validated run config")
	}
	repoRoot, err := repoRootForConfig(config)
	if err != nil {
		return udonrunner.Result{}, err
	}
	p, manifest, packageDigest, err := validatePackage(ctx, packageOptions{
		RepoRoot: repoRoot, ExampleDir: config.PackageRoot, Assess: opts.Assess,
	})
	if err != nil {
		return udonrunner.Result{}, err
	}
	if err := validateManifestPolicy(manifest); err != nil {
		return udonrunner.Result{}, err
	}
	if err := validateApproval(approval, p.scope, packageDigest, config.Tier, resolveNow(opts.Now)); err != nil {
		return udonrunner.Result{}, err
	}
	handoffBytes, _, err := evidencefile.ReadRegular(p.handoff, evidencefile.DefaultMaxBytes)
	if err != nil {
		return udonrunner.Result{}, err
	}
	if evidencefile.SHA256(handoffBytes) != config.HandoffSHA256 {
		return udonrunner.Result{}, fmt.Errorf("handoff SHA-256 does not match the validated run config")
	}
	var browserDriver string
	var browserDriverArgs []string
	if config.Browser != nil {
		browserDriver = config.Browser.DriverPath
		browserDriverArgs = config.Browser.DriverArgs
	}
	expectedBrowser, err := buildBrowserRunConfig(p.exampleAbs, browserDriver, browserDriverArgs, opts.Env, false)
	if err != nil {
		return udonrunner.Result{}, err
	}
	expected, err := buildRunConfig(p, manifest, packageDigest, config.Tier, config.WorkDir, config.RunID, config.HandoffSHA256, config.ApprovalSHA256, expectedBrowser)
	if err != nil {
		return udonrunner.Result{}, err
	}
	canonical, err := json.MarshalIndent(expected, "", "  ")
	if err != nil {
		return udonrunner.Result{}, err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(configBytes, canonical) {
		return udonrunner.Result{}, fmt.Errorf("run config bytes are not the canonical validated encoding")
	}
	result, err := udonrunner.Run(ctx, config, udonrunner.Options{
		RepoRoot: repoRoot, Env: opts.Env, Stdout: opts.Stdout, Stderr: opts.Stderr, Invoke: opts.Invoke,
	})
	if err != nil {
		return result, err
	}
	return publishExternalExecutorReport(config, result)
}

func publishExternalExecutorReport(config RunConfig, result udonrunner.Result) (udonrunner.Result, error) {
	data, _, err := evidencefile.ReadRegular(result.ExecutorReportPath, evidencefile.DefaultMaxBytes)
	if err != nil {
		return result, fmt.Errorf("read successful external executor report: %w", err)
	}
	report, err := decodeUdonExecutionReport(data)
	if err != nil {
		return result, fmt.Errorf("validate successful external executor report: %w", err)
	}
	if !strings.EqualFold(strings.TrimSpace(report.Status), "success") {
		return result, fmt.Errorf("successful external executor report status must be success")
	}
	path, err := externalExecutorReportPath(config)
	if err != nil {
		return result, err
	}
	if err := atomicfile.WriteNew(path, data, 0o600); err != nil {
		return result, fmt.Errorf("publish external executor report: %w", err)
	}
	result.ExecutorReportPath = path
	return result, nil
}

func externalExecutorReportPath(config RunConfig) (string, error) {
	workdir := strings.TrimSpace(config.WorkDir)
	if workdir == "" || !filepath.IsAbs(workdir) {
		return "", fmt.Errorf("external executor report requires an absolute workdir")
	}
	if err := udonrunner.ValidateRunID(config.RunID); err != nil {
		return "", err
	}
	return filepath.Join(filepath.Clean(workdir), "executor-report-"+config.RunID+".json"), nil
}

func repoRootForConfig(config RunConfig) (string, error) {
	root, err := filepath.Abs(config.PackageRoot)
	if err != nil {
		return "", err
	}
	scope := strings.Trim(strings.TrimSpace(filepath.ToSlash(config.Scope)), "/")
	if scope == "" {
		return "", fmt.Errorf("run config scope is required")
	}
	for range strings.Split(scope, "/") {
		root = filepath.Dir(root)
	}
	return root, nil
}
