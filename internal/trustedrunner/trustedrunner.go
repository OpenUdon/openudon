package trustedrunner

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/genelet/ramen/internal/synthesize"
)

const (
	ApprovalVersion        = "ramen.approval.v1"
	SymphonyHandoffVersion = "ramen.symphony-handoff.v1"

	StateApprovedForSandbox    = "approved_for_sandbox"
	StateApprovedForProduction = "approved_for_production"

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
	Scope         string
	Tier          string
	PackageSHA256 string
	WorkflowPath  string
	WorkDir       string
	DryRun        bool
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

type handoffManifest struct {
	Version        string `json:"version"`
	GeneratedState string `json:"generated_state"`
	HandoffInputs  []struct {
		Path     string `json:"path"`
		Required bool   `json:"required"`
	} `json:"handoff_inputs"`
	ExecutionPolicy struct {
		DirectProductionExecution bool `json:"direct_production_execution"`
	} `json:"execution_policy"`
	CredentialBindings struct {
		ValuesAllowedInArtifacts bool `json:"values_allowed_in_artifacts"`
	} `json:"credential_bindings"`
}

type packageFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type packageDigest struct {
	Version string        `json:"version"`
	Scope   string        `json:"scope"`
	Files   []packageFile `json:"files"`
}

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

	result := &RunResult{
		Scope:         p.scope,
		Tier:          opts.Tier,
		PackageSHA256: digest,
		WorkflowPath:  p.workflow,
		WorkDir:       strings.TrimSpace(opts.WorkDir),
		DryRun:        opts.DryRun,
	}
	if result.WorkDir == "" {
		result.WorkDir = p.defaultWorkDir
	}
	if opts.DryRun {
		return result, nil
	}

	runnerPath := strings.TrimSpace(opts.RunnerPath)
	if runnerPath == "" {
		runnerPath = filepath.Join(p.repoRoot, "scripts", "run-udon.sh")
	}
	args := []string{p.workflow}
	if result.WorkDir != "" {
		args = append(args, result.WorkDir)
	}
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
		return nil, fmt.Errorf("run udon: %w", err)
	}
	return result, nil
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
	scope := filepath.ToSlash(filepath.Clean(rel))
	return paths{
		repoRoot:       repoAbs,
		exampleAbs:     exampleAbs,
		scope:          scope,
		project:        filepath.Join(exampleAbs, "project.md"),
		workflow:       filepath.Join(exampleAbs, "workflows", "workflow.hcl"),
		quality:        filepath.Join(exampleAbs, "expected", "quality.json"),
		handoff:        filepath.Join(exampleAbs, "expected", "symphony-handoff.json"),
		defaultWorkDir: filepath.Join(repoAbs, ".ramen-run", strings.ReplaceAll(scope, "/", "-")),
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
	if manifest.Version != SymphonyHandoffVersion || manifest.GeneratedState != "generated" {
		return handoffManifest{}, fmt.Errorf("handoff manifest must use version %s and generated state", SymphonyHandoffVersion)
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
	required := map[string]bool{
		"project.md":                     false,
		"workflows/intent.hcl":           false,
		"workflows/workflow.hcl":         false,
		"workflows/workflow.uws.yaml":    false,
		"expected/plan.json":             false,
		"expected/quality.json":          false,
		"expected/refinement.json":       false,
		"expected/review.md":             false,
		"expected/symphony-handoff.json": false,
	}
	for _, input := range manifest.HandoffInputs {
		if !input.Required {
			continue
		}
		clean, err := cleanManifestPath(input.Path)
		if err != nil {
			return err
		}
		if _, ok := required[clean]; ok {
			required[clean] = true
		}
	}
	var missing []string
	for path, found := range required {
		if !found {
			missing = append(missing, path)
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		return fmt.Errorf("handoff manifest missing required input(s): %s", strings.Join(missing, ", "))
	}
	for path := range required {
		full := filepath.Join(p.exampleAbs, filepath.FromSlash(path))
		if _, err := os.Stat(full); err != nil {
			return fmt.Errorf("required handoff input %s: %w", path, err)
		}
	}
	return nil
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
	fileSet := map[string]bool{}
	for _, input := range manifest.HandoffInputs {
		if !input.Required {
			continue
		}
		clean, err := cleanManifestPath(input.Path)
		if err != nil {
			return "", err
		}
		fileSet[clean] = true
	}
	var files []string
	for path := range fileSet {
		files = append(files, path)
	}
	sort.Strings(files)
	digest := packageDigest{
		Version: "ramen.handoff-package-digest.v1",
		Scope:   p.scope,
	}
	for _, path := range files {
		full := filepath.Join(p.exampleAbs, filepath.FromSlash(path))
		data, err := os.ReadFile(full)
		if err != nil {
			return "", fmt.Errorf("read handoff input %s: %w", path, err)
		}
		sum := sha256.Sum256(data)
		digest.Files = append(digest.Files, packageFile{
			Path:   filepath.ToSlash(filepath.Join(p.scope, path)),
			SHA256: hex.EncodeToString(sum[:]),
		})
	}
	canonical, err := json.Marshal(digest)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

func cleanManifestPath(path string) (string, error) {
	path = strings.TrimSpace(filepath.ToSlash(path))
	if path == "" || strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("handoff input path must be repo-local: %q", path)
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("handoff input path escapes example: %q", path)
	}
	return clean, nil
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
