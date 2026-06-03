package releaseevidence

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunWritesReleaseEvidenceSummary(t *testing.T) {
	repoRoot := repoRootForTest(t)
	runRoot := filepath.Join(repoRoot, ".openudon-run", "test-release-evidence-"+strings.ReplaceAll(t.Name(), "/", "-"))
	t.Cleanup(func() {
		_ = os.RemoveAll(runRoot)
	})
	summary, err := Run(context.Background(), Options{
		RepoRoot:     repoRoot,
		UdonRepo:     filepath.Join(runRoot, "fake-udon-repo"),
		WorkDir:      runRoot,
		ArchiveDir:   filepath.Join(runRoot, "archive"),
		ReleaseNotes: filepath.Join(runRoot, "notes.md"),
		SummaryJSON:  filepath.Join(runRoot, "summary.json"),
		SummaryMD:    filepath.Join(runRoot, "summary.md"),
		Gates:        []string{"go test ./...=pass"},
		Now:          fixedNow,
		BuildCommand: fakeBuildCommand,
		RunCommand:   fakeUdonRunCommand,
		GitCommand:   fakeGitCommand,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v; summary=%#v", err, summary)
	}
	if summary.Status != "pass" || summary.Commit != "abc1234" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	for _, path := range []string{
		summary.SummaryJSON,
		summary.SummaryMD,
		summary.ReleaseNotes,
		summary.ArchivedRun,
		filepath.Join(summary.ArchiveDir, "async-evidence.json"),
		filepath.Join(summary.ArchiveDir, "executor-report.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", path, err)
		}
	}
	data, err := os.ReadFile(summary.SummaryJSON)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Summary
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("summary JSON invalid: %v\n%s", err, data)
	}
	if decoded.Version != SummaryVersion || decoded.Counts["async_sidecars"] != 1 {
		t.Fatalf("decoded summary = %#v", decoded)
	}
	markdown, err := os.ReadFile(summary.SummaryMD)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"OpenUdon Release Evidence Summary",
		"go test ./...=pass",
		"run-evidence verify: pass",
		"executor-report.json",
	} {
		if !strings.Contains(string(markdown), expected) {
			t.Fatalf("summary markdown missing %q:\n%s", expected, markdown)
		}
	}
	notes, err := os.ReadFile(summary.ReleaseNotes)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(notes), "go test ./...=pass") || !strings.Contains(string(notes), "async-evidence.json") {
		t.Fatalf("release notes missing evidence details:\n%s", notes)
	}
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.Abs(filepath.Join(cwd, "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func fixedNow() time.Time {
	return time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
}

func fakeBuildCommand(_ context.Context, _ string, out string) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	return os.WriteFile(out, []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755)
}

func fakeUdonRunCommand(_ context.Context, _ string, args ...string) error {
	reportPath := argValue(args, "--execution-report")
	data := `{"version":"udon.execution-report.v1","status":"success","started_at":"2026-06-03T12:00:00Z","finished_at":"2026-06-03T12:00:00Z","workflow_path":"workflow.uws.yaml","workflow_format":"uws-yaml","workdir":".","output_path":"output.hcl","output_digest":"sha256:` + strings.Repeat("a", 64) + `"}` + "\n"
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(reportPath, []byte(data), 0o600)
}

func fakeGitCommand(_ context.Context, name string, args ...string) ([]byte, error) {
	return []byte("abc1234\n"), nil
}

func argValue(args []string, name string) string {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}
