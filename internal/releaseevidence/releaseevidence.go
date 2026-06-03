package releaseevidence

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/smokematrix"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

const SummaryVersion = "openudon.release-evidence-summary.v1"

type Options struct {
	RepoRoot     string
	UdonRepo     string
	WorkDir      string
	ArchiveDir   string
	ReleaseNotes string
	SummaryJSON  string
	SummaryMD    string
	Gates        []string
	Now          func() time.Time

	RunCommand   func(context.Context, string, ...string) error
	BuildCommand func(context.Context, string, string) error
	GitCommand   func(context.Context, string, ...string) ([]byte, error)
}

type Summary struct {
	Version        string            `json:"version"`
	CreatedAt      string            `json:"created_at"`
	Status         string            `json:"status"`
	Commit         string            `json:"commit"`
	WorkDir        string            `json:"workdir"`
	SmokeSummary   string            `json:"smoke_summary"`
	RunEvidence    string            `json:"run_evidence"`
	ArchiveDir     string            `json:"archive_dir"`
	ArchivedRun    string            `json:"archived_run_evidence"`
	AsyncEvidence  []string          `json:"async_evidence,omitempty"`
	ExecutorReport string            `json:"executor_report,omitempty"`
	ReleaseNotes   string            `json:"release_notes"`
	SummaryJSON    string            `json:"summary_json"`
	SummaryMD      string            `json:"summary_md"`
	Gates          []string          `json:"gates"`
	VerifierOutput string            `json:"verifier_output"`
	Counts         map[string]int    `json:"counts,omitempty"`
	Errors         []string          `json:"errors,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

func Run(ctx context.Context, opts Options) (*Summary, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	repoRoot, err := filepath.Abs(defaultString(opts.RepoRoot, "."))
	if err != nil {
		return nil, err
	}
	workdir, err := filepath.Abs(defaultString(opts.WorkDir, filepath.Join(repoRoot, ".openudon-run", "release-evidence")))
	if err != nil {
		return nil, err
	}
	smokeSummary := filepath.Join(workdir, "local-udon-smoke.json")
	report, err := smokematrix.RunLocalUdonSmoke(ctx, smokematrix.LocalUdonSmokeOptions{
		RepoRoot:     repoRoot,
		UdonRepo:     opts.UdonRepo,
		WorkDir:      filepath.Join(workdir, "smoke"),
		OutPath:      smokeSummary,
		Now:          opts.Now,
		RunCommand:   opts.RunCommand,
		BuildCommand: opts.BuildCommand,
	})
	if err != nil {
		summary := baseSummary(opts, workdir)
		summary.Status = "fail"
		summary.Errors = append(summary.Errors, err.Error())
		_ = writeSummaryArtifacts(summary)
		return summary, err
	}
	if report == nil || len(report.Scenarios) == 0 || strings.TrimSpace(report.Scenarios[0].RunEvidencePath) == "" {
		err := fmt.Errorf("local udon smoke did not return run evidence")
		summary := baseSummary(opts, workdir)
		summary.Status = "fail"
		summary.Errors = append(summary.Errors, err.Error())
		_ = writeSummaryArtifacts(summary)
		return summary, err
	}
	runEvidence := filepath.FromSlash(report.Scenarios[0].RunEvidencePath)
	archiveDir := defaultString(opts.ArchiveDir, filepath.Join(workdir, "archive"))
	archive, err := trustedrunner.ArchiveRunEvidence(trustedrunner.ArchiveOptions{
		RunEvidencePath: runEvidence,
		ArchiveDir:      archiveDir,
	})
	if err != nil {
		summary := baseSummary(opts, workdir)
		summary.Status = "fail"
		summary.SmokeSummary = smokeSummary
		summary.RunEvidence = runEvidence
		summary.Errors = append(summary.Errors, err.Error())
		_ = writeSummaryArtifacts(summary)
		return summary, err
	}
	verifierOutput := fmt.Sprintf("openudon run-evidence verify: pass %s (%d async sidecar file(s))", archive.RunEvidencePath, archive.VerifiedSidecars)
	gates := releaseEvidenceGates(opts.Gates)
	notesPath := defaultString(opts.ReleaseNotes, filepath.Join(workdir, "release-notes.md"))
	notes, err := trustedrunner.WriteReleaseNotesDraft(ctx, trustedrunner.ReleaseNotesOptions{
		RepoRoot:        repoRoot,
		RunEvidencePath: archive.RunEvidencePath,
		OutPath:         notesPath,
		Gates:           gates,
		Now:             opts.Now,
		RunCommand:      opts.GitCommand,
	})
	if err != nil {
		summary := baseSummary(opts, workdir)
		summary.Status = "fail"
		summary.SmokeSummary = smokeSummary
		summary.RunEvidence = runEvidence
		summary.ArchiveDir = archive.ArchiveDir
		summary.ArchivedRun = archive.RunEvidencePath
		summary.AsyncEvidence = archive.AsyncEvidence
		summary.ExecutorReport = archive.ExecutorReport
		summary.VerifierOutput = verifierOutput
		summary.Gates = gates
		summary.Errors = append(summary.Errors, err.Error())
		_ = writeSummaryArtifacts(summary)
		return summary, err
	}
	summary := baseSummary(opts, workdir)
	summary.Status = "pass"
	summary.Commit = notes.Commit
	summary.SmokeSummary = smokeSummary
	summary.RunEvidence = runEvidence
	summary.ArchiveDir = archive.ArchiveDir
	summary.ArchivedRun = archive.RunEvidencePath
	summary.AsyncEvidence = archive.AsyncEvidence
	summary.ExecutorReport = archive.ExecutorReport
	summary.ReleaseNotes = notes.Path
	summary.Gates = gates
	summary.VerifierOutput = verifierOutput
	summary.Counts = map[string]int{
		"async_sidecars":  archive.VerifiedSidecars,
		"smoke_scenarios": len(report.Scenarios),
	}
	if err := writeSummaryArtifacts(summary); err != nil {
		summary.Status = "fail"
		summary.Errors = append(summary.Errors, err.Error())
		return summary, err
	}
	return summary, nil
}

func baseSummary(opts Options, workdir string) *Summary {
	now := time.Now().UTC()
	if opts.Now != nil {
		now = opts.Now().UTC()
	}
	summaryJSON := defaultString(opts.SummaryJSON, filepath.Join(workdir, "summary.json"))
	summaryMD := defaultString(opts.SummaryMD, filepath.Join(workdir, "summary.md"))
	return &Summary{
		Version:     SummaryVersion,
		CreatedAt:   now.Format(time.RFC3339),
		Status:      "pending",
		WorkDir:     workdir,
		SummaryJSON: summaryJSON,
		SummaryMD:   summaryMD,
		Metadata: map[string]string{
			"local_only": "true",
		},
	}
}

func writeSummaryArtifacts(summary *Summary) error {
	if summary == nil {
		return fmt.Errorf("release evidence summary is nil")
	}
	if strings.TrimSpace(summary.SummaryJSON) != "" {
		if err := os.MkdirAll(filepath.Dir(summary.SummaryJSON), 0o755); err != nil {
			return err
		}
		data, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(summary.SummaryJSON, append(data, '\n'), 0o644); err != nil {
			return err
		}
	}
	if strings.TrimSpace(summary.SummaryMD) != "" {
		if err := os.MkdirAll(filepath.Dir(summary.SummaryMD), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(summary.SummaryMD, []byte(formatMarkdown(summary)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func formatMarkdown(summary *Summary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# OpenUdon Release Evidence Summary\n\n")
	fmt.Fprintf(&b, "- Status: %s\n", summary.Status)
	fmt.Fprintf(&b, "- Created at: %s\n", summary.CreatedAt)
	if strings.TrimSpace(summary.Commit) != "" {
		fmt.Fprintf(&b, "- Commit: %s\n", summary.Commit)
	}
	fmt.Fprintf(&b, "- Workdir: %s\n", summary.WorkDir)
	fmt.Fprintf(&b, "\n## Evidence\n\n")
	writePathLine(&b, "Smoke summary", summary.SmokeSummary)
	writePathLine(&b, "Run evidence", summary.RunEvidence)
	writePathLine(&b, "Archive", summary.ArchiveDir)
	writePathLine(&b, "Archived run evidence", summary.ArchivedRun)
	for _, path := range summary.AsyncEvidence {
		writePathLine(&b, "Async evidence", path)
	}
	writePathLine(&b, "Executor report", summary.ExecutorReport)
	writePathLine(&b, "Release notes", summary.ReleaseNotes)
	fmt.Fprintf(&b, "\n## Gates\n\n")
	for _, gate := range summary.Gates {
		fmt.Fprintf(&b, "- %s\n", gate)
	}
	if strings.TrimSpace(summary.VerifierOutput) != "" {
		fmt.Fprintf(&b, "\n## Verifier Output\n\n```text\n%s\n```\n", summary.VerifierOutput)
	}
	if len(summary.Errors) > 0 {
		fmt.Fprintf(&b, "\n## Errors\n\n")
		for _, err := range summary.Errors {
			fmt.Fprintf(&b, "- %s\n", err)
		}
	}
	return b.String()
}

func writePathLine(b *strings.Builder, label, path string) {
	if strings.TrimSpace(path) != "" {
		fmt.Fprintf(b, "- %s: %s\n", label, path)
	}
}

func releaseEvidenceGates(input []string) []string {
	gates := []string{
		"local-udon-smoke=pass",
		"run-evidence archive=pass",
		"run-evidence verify=pass",
	}
	for _, gate := range input {
		gate = strings.TrimSpace(gate)
		if gate != "" {
			gates = append(gates, gate)
		}
	}
	sort.Strings(gates)
	return gates
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
