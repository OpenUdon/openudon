package trustedrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

type ArchiveOptions struct {
	RunEvidencePath string
	ArchiveDir      string
}

type ArchiveResult struct {
	ArchiveDir       string
	RunEvidencePath  string
	AsyncEvidence    []string
	ExecutorReport   string
	VerifiedSidecars int
}

type ReleaseNotesOptions struct {
	RepoRoot           string
	RunEvidencePath    string
	OutPath            string
	Gates              []string
	VerifierOutputPath string
	Now                func() time.Time
	RunCommand         func(context.Context, string, ...string) ([]byte, error)
}

type ReleaseNotesResult struct {
	Path   string
	Commit string
}

func ArchiveRunEvidence(opts ArchiveOptions) (ArchiveResult, error) {
	if strings.TrimSpace(opts.RunEvidencePath) == "" {
		return ArchiveResult{}, fmt.Errorf("run evidence path is required")
	}
	if strings.TrimSpace(opts.ArchiveDir) == "" {
		return ArchiveResult{}, fmt.Errorf("archive dir is required")
	}
	verified, err := VerifyRunEvidenceFile(opts.RunEvidencePath)
	if err != nil {
		return ArchiveResult{}, err
	}
	archiveDir, err := filepath.Abs(opts.ArchiveDir)
	if err != nil {
		return ArchiveResult{}, err
	}
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return ArchiveResult{}, err
	}
	runDst := filepath.Join(archiveDir, "run-evidence.json")
	if err := copyFileForArchive(opts.RunEvidencePath, runDst, 0o600); err != nil {
		return ArchiveResult{}, err
	}
	workdir := filepath.Dir(opts.RunEvidencePath)
	var asyncPaths []string
	for _, ref := range verified.AsyncEvidenceFiles {
		clean, err := packageartifacts.CleanRelativePath(ref.Path)
		if err != nil || clean != ref.Path {
			return ArchiveResult{}, fmt.Errorf("async evidence path must be safe workdir-relative path: %q", ref.Path)
		}
		src := filepath.Join(workdir, filepath.FromSlash(ref.Path))
		dst := filepath.Join(archiveDir, filepath.FromSlash(ref.Path))
		if err := copyFileForArchive(src, dst, 0o600); err != nil {
			return ArchiveResult{}, err
		}
		asyncPaths = append(asyncPaths, dst)
	}
	evidence, err := readRunEvidenceStrict(opts.RunEvidencePath)
	if err != nil {
		return ArchiveResult{}, err
	}
	reportDst := ""
	if strings.TrimSpace(evidence.Executor.ReportPath) != "" {
		if info, err := os.Stat(evidence.Executor.ReportPath); err == nil && info.Mode().IsRegular() {
			reportDst = filepath.Join(archiveDir, "executor-report.json")
			if err := copyFileForArchive(evidence.Executor.ReportPath, reportDst, 0o600); err != nil {
				return ArchiveResult{}, err
			}
		}
	}
	archived, err := VerifyRunEvidenceFile(runDst)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("verify archived run evidence: %w", err)
	}
	sort.Strings(asyncPaths)
	return ArchiveResult{
		ArchiveDir:       archiveDir,
		RunEvidencePath:  runDst,
		AsyncEvidence:    asyncPaths,
		ExecutorReport:   reportDst,
		VerifiedSidecars: len(archived.AsyncEvidenceFiles),
	}, nil
}

func WriteReleaseNotesDraft(ctx context.Context, opts ReleaseNotesOptions) (ReleaseNotesResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(opts.RunEvidencePath) == "" {
		return ReleaseNotesResult{}, fmt.Errorf("run evidence path is required")
	}
	if strings.TrimSpace(opts.OutPath) == "" {
		return ReleaseNotesResult{}, fmt.Errorf("release-note output path is required")
	}
	verified, err := VerifyRunEvidenceFile(opts.RunEvidencePath)
	if err != nil {
		return ReleaseNotesResult{}, err
	}
	evidence, err := readRunEvidenceStrict(opts.RunEvidencePath)
	if err != nil {
		return ReleaseNotesResult{}, err
	}
	repoRoot := strings.TrimSpace(opts.RepoRoot)
	if repoRoot == "" {
		repoRoot = "."
	}
	commit, err := currentCommit(ctx, repoRoot, opts.RunCommand)
	if err != nil {
		return ReleaseNotesResult{}, err
	}
	verifierOutput := strings.TrimSpace(opts.VerifierOutputPath)
	if verifierOutput == "" {
		verifierOutput = fmt.Sprintf("openudon run-evidence verify: pass %s (%d async sidecar file(s))", verified.RunEvidencePath, len(verified.AsyncEvidenceFiles))
	} else {
		data, err := os.ReadFile(verifierOutput)
		if err != nil {
			return ReleaseNotesResult{}, fmt.Errorf("read verifier output: %w", err)
		}
		verifierOutput = strings.TrimSpace(string(data))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# OpenUdon Release Evidence Draft\n\n")
	fmt.Fprintf(&b, "- Created at: %s\n", resolveNow(opts.Now).UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- Commit: %s\n", commit)
	fmt.Fprintf(&b, "- Scope: %s\n", evidence.Scope)
	fmt.Fprintf(&b, "- Tier: %s\n", evidence.Tier)
	fmt.Fprintf(&b, "- Package SHA-256: %s\n", evidence.PackageSHA256)
	fmt.Fprintf(&b, "\n## Gate Results\n\n")
	for _, gate := range releaseNoteGates(opts.Gates) {
		fmt.Fprintf(&b, "- %s\n", gate)
	}
	fmt.Fprintf(&b, "\n## Verifier Output\n\n```text\n%s\n```\n\n", verifierOutput)
	fmt.Fprintf(&b, "## Evidence Paths\n\n")
	fmt.Fprintf(&b, "- Run evidence: %s\n", verified.RunEvidencePath)
	for _, ref := range verified.AsyncEvidenceFiles {
		fmt.Fprintf(&b, "- Async sidecar: %s (records: %d, digest: %s)\n", filepath.Join(filepath.Dir(verified.RunEvidencePath), filepath.FromSlash(ref.Path)), ref.Records, ref.Digest)
	}
	if strings.TrimSpace(evidence.Executor.ReportPath) != "" {
		fmt.Fprintf(&b, "- Executor report: %s\n", evidence.Executor.ReportPath)
	}
	if err := os.MkdirAll(filepath.Dir(opts.OutPath), 0o755); err != nil {
		return ReleaseNotesResult{}, err
	}
	if err := os.WriteFile(opts.OutPath, []byte(b.String()), 0o644); err != nil {
		return ReleaseNotesResult{}, err
	}
	return ReleaseNotesResult{Path: opts.OutPath, Commit: commit}, nil
}

func readRunEvidenceStrict(path string) (RunEvidence, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunEvidence{}, err
	}
	var evidence RunEvidence
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&evidence); err != nil {
		return RunEvidence{}, err
	}
	return evidence, nil
}

func copyFileForArchive(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read archive source %s: %w", src, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, mode); err != nil {
		return fmt.Errorf("write archive file %s: %w", dst, err)
	}
	return nil
}

func currentCommit(ctx context.Context, repoRoot string, run func(context.Context, string, ...string) ([]byte, error)) (string, error) {
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Dir = repoRoot
			return cmd.Output()
		}
	}
	out, err := run(ctx, "git", "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve current commit: %w", err)
	}
	commit := strings.TrimSpace(string(out))
	if commit == "" {
		return "", fmt.Errorf("resolve current commit: empty output")
	}
	return commit, nil
}

func releaseNoteGates(input []string) []string {
	var out []string
	for _, gate := range input {
		gate = strings.TrimSpace(gate)
		if gate != "" {
			out = append(out, gate)
		}
	}
	if len(out) == 0 {
		out = []string{"run-evidence verify=pass"}
	}
	sort.Strings(out)
	return out
}
