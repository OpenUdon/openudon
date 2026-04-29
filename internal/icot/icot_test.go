package icot

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMainPreviewEOFCancelsWithoutWriting(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example}, strings.NewReader(testProjectInput(false)), &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Main succeeded with EOF at preview\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(example); !os.IsNotExist(err) {
		t.Fatalf("EOF at preview created files or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unexpected EOF") {
		t.Fatalf("stderr missing unexpected EOF:\n%s", stderr.String())
	}
}

func TestBackupProjectCreatesDistinctBackups(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.md")
	if err := os.WriteFile(projectPath, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old project: %v", err)
	}
	if err := backupProject(projectPath); err != nil {
		t.Fatalf("first backup failed: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte("new\n"), 0o644); err != nil {
		t.Fatalf("write new project: %v", err)
	}
	if err := backupProject(projectPath); err != nil {
		t.Fatalf("second backup failed: %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "project.md.bak.*"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("backup count = %d, want 2: %v", len(backups), backups)
	}
	contents := map[string]bool{}
	for _, path := range backups {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read backup %s: %v", path, err)
		}
		contents[string(data)] = true
	}
	if !contents["old\n"] || !contents["new\n"] {
		t.Fatalf("backup contents = %#v, want old and new", contents)
	}
}

func testProjectInput(withOpenAPI bool) string {
	answers := []string{
		"Guided Project",
		"Fetch a ticket and prepare a draft",
		"`ticket_id`: required string",
		"Stored draft reply",
		"Pass ticket body to draft writer",
		"`write_draft`: inputs ticket body; outputs draft id",
	}
	if withOpenAPI {
		answers = append(answers, "yes", "Support API: use `openapi/support.yaml`")
	} else {
		answers = append(answers, "")
	}
	answers = append(answers,
		"",
		"",
		"support_api_token",
		"Sandbox proof runs only",
		"Stop if required services are unavailable",
	)
	return strings.Join(answers, "\n") + "\n"
}
