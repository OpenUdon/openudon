package trustedrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/processgroup"
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
	Commit             string
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
	evidence, err := readRunEvidenceStrict(opts.RunEvidencePath)
	if err != nil {
		return ArchiveResult{}, err
	}
	if evidence.Version != RunEvidenceVersion {
		return ArchiveResult{}, fmt.Errorf("legacy run evidence %s is read-only and cannot be archived because report ownership is not provable", evidence.Version)
	}
	archiveDir, err := filepath.Abs(opts.ArchiveDir)
	if err != nil {
		return ArchiveResult{}, err
	}
	if err := prepareArchiveDirectory(archiveDir); err != nil {
		return ArchiveResult{}, err
	}
	runDst := filepath.Join(archiveDir, "run-evidence.json")
	if err := copyFileForArchive(archiveDir, opts.RunEvidencePath, runDst, 0o600); err != nil {
		return ArchiveResult{}, err
	}
	if _, err := os.Lstat(SignaturePath(opts.RunEvidencePath)); err == nil {
		if err := copyFileForArchive(archiveDir, SignaturePath(opts.RunEvidencePath), SignaturePath(runDst), 0o600); err != nil {
			return ArchiveResult{}, err
		}
	} else if !os.IsNotExist(err) {
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
		if err := copyFileForArchive(archiveDir, src, dst, 0o600); err != nil {
			return ArchiveResult{}, err
		}
		asyncPaths = append(asyncPaths, dst)
	}
	reportDst := ""
	if strings.TrimSpace(evidence.Executor.ReportPath) != "" {
		clean, err := packageartifacts.CleanRelativePath(evidence.Executor.ReportPath)
		if err != nil || clean != evidence.Executor.ReportPath {
			return ArchiveResult{}, fmt.Errorf("executor report path must be safe workdir-relative path: %q", evidence.Executor.ReportPath)
		}
		src := filepath.Join(workdir, filepath.FromSlash(clean))
		reportDst = filepath.Join(archiveDir, filepath.FromSlash(clean))
		if err := copyFileForArchive(archiveDir, src, reportDst, 0o600); err != nil {
			return ArchiveResult{}, err
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
	commit := strings.TrimSpace(opts.Commit)
	if commit == "" {
		commit, err = currentCommit(ctx, repoRoot, opts.RunCommand)
		if err != nil {
			return ReleaseNotesResult{}, err
		}
	}
	if err := validateCommitRevision(commit); err != nil {
		return ReleaseNotesResult{}, err
	}
	verifierOutput := strings.TrimSpace(opts.VerifierOutputPath)
	if verifierOutput == "" {
		verifierOutput = fmt.Sprintf("openudon run-evidence verify: pass %s (%d async sidecar file(s))", verified.RunEvidencePath, len(verified.AsyncEvidenceFiles))
	} else {
		data, _, err := evidencefile.ReadRegular(verifierOutput, 1<<20)
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
	if err := atomicfile.Write(opts.OutPath, []byte(b.String()), 0o644); err != nil {
		return ReleaseNotesResult{}, err
	}
	return ReleaseNotesResult{Path: opts.OutPath, Commit: commit}, nil
}

func readRunEvidenceStrict(path string) (RunEvidence, error) {
	data, _, err := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
	if err != nil {
		return RunEvidence{}, err
	}
	var evidence RunEvidence
	if err := evidencefile.DecodeStrict(data, &evidence); err != nil {
		return RunEvidence{}, err
	}
	return evidence, nil
}

func prepareArchiveDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			return fmt.Errorf("create exclusive archive directory: %w", err)
		}
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("archive path must be a real directory: %s", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("archive directory is not empty: %s", path)
	}
	return nil
}

func copyFileForArchive(root, src, dst string, mode os.FileMode) error {
	data, _, err := evidencefile.ReadRegular(src, evidencefile.DefaultMaxBytes)
	if err != nil {
		return fmt.Errorf("read archive source %s: %w", src, err)
	}
	if err := createArchiveParents(root, filepath.Dir(dst)); err != nil {
		return err
	}
	if err := atomicfile.WriteNew(dst, data, mode); err != nil {
		return fmt.Errorf("write archive file %s: %w", dst, err)
	}
	return nil
}

func createArchiveParents(root, target string) error {
	root = filepath.Clean(root)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(root, target)
	if err != nil || rel == ".." || filepath.IsAbs(rel) || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("archive destination escapes archive directory: %s", target)
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 || !rootInfo.IsDir() {
		return fmt.Errorf("archive path must remain a real directory: %s", root)
	}
	current := root
	if rel == "." {
		return nil
	}
	for _, component := range strings.Split(rel, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			if err := os.Mkdir(current, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
			info, err = os.Lstat(current)
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("archive destination ancestor is not a real directory: %s", current)
		}
	}
	return nil
}

func currentCommit(ctx context.Context, repoRoot string, run func(context.Context, string, ...string) ([]byte, error)) (string, error) {
	if run == nil {
		run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
			var out bytes.Buffer
			err := processgroup.Run(ctx, 30*time.Second, processgroup.Invocation{
				Args:   append([]string{name}, args...),
				Dir:    repoRoot,
				Env:    os.Environ(),
				Stdout: &out,
			})
			return out.Bytes(), err
		}
	}
	out, err := run(ctx, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("resolve current commit: %w", err)
	}
	commit := strings.TrimSpace(string(out))
	if commit == "" {
		return "", fmt.Errorf("resolve current commit: empty output")
	}
	return commit, nil
}

func validateCommitRevision(commit string) error {
	if !evidencefile.ValidGitObject(commit) {
		return fmt.Errorf("commit revision must be a full 40- or 64-character hexadecimal Git object ID")
	}
	return nil
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
