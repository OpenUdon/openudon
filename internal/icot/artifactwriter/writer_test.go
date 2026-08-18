package artifactwriter

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
)

const testIntent = `workflow {
  name = "safe"
  description = "safe"
}

step "run" {
  type = "fnct"
  do = "Run safely."
}
`

func TestProposedFileActionsMatchPreparedTransaction(t *testing.T) {
	now := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)

	t.Run("complete-removals", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "complete")
		prepared, err := Prepare(example, elicitor.Artifacts{ProjectMD: "project\n", IntentHCL: "intent\n"}, true, now)
		if err != nil {
			t.Fatal(err)
		}
		actions := ProposedFileActions(prepared)
		assertActionsCoverPreparedFiles(t, prepared, actions)
		for _, relative := range []string{".icot/browser-authentication.json", ".icot/browser-sources.json", ".icot/readiness.json", ".icot/session.yaml", "workflows/intent.draft.hcl"} {
			assertAction(t, actions, "remove_if_present", filepath.Join(example, filepath.FromSlash(relative)))
		}
	})

	t.Run("incomplete-authentication-writes", func(t *testing.T) {
		example := filepath.Join(t.TempDir(), "incomplete")
		materialized := []byte("reviewed authentication profile\n")
		digest := fmt.Sprintf("%x", sha256.Sum256(materialized))
		session := elicitor.Session{SourcePlan: []elicitor.SourceMaterialization{{
			Kind: "browser-authentication", ID: "member", SourcePath: "reviewed-auth.json", TargetPath: "browser-authentication/member.json", SHA256: digest,
			Title: "Member authentication", Flows: []string{"sign_in"}, FlowCredentialSlots: map[string][]string{"sign_in": {"username", "password"}},
			Origins: []string{"https://example.test"}, Lifecycle: "active", ExpiresAt: "2126-08-18T12:00:00Z", Provenance: "reviewed test fixture",
			MaterializedContent: materialized,
		}}}
		prepared, err := Prepare(example, elicitor.Artifacts{ProjectMD: "project\n", IntentHCL: "draft intent\n", Incomplete: true, Session: session}, true, now)
		if err != nil {
			t.Fatal(err)
		}
		actions := ProposedFileActions(prepared)
		assertActionsCoverPreparedFiles(t, prepared, actions)
		assertAction(t, actions, "write", filepath.Join(example, ".icot", "browser-authentication.json"))
		assertAction(t, actions, "write", filepath.Join(example, ".icot", "session.yaml"))
		assertAction(t, actions, "write", filepath.Join(example, ".icot", "readiness.json"))
		assertAction(t, actions, "copy", filepath.Join(example, "browser-authentication", "member.json"))
	})
}

func TestAtomicWriterCleansBackupsAndRollsBackDeterministically(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project.md")
	intent := filepath.Join(root, "workflows", "intent.hcl")
	if err := os.MkdirAll(filepath.Dir(intent), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(project, []byte("old project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(intent, []byte("workflow \"old\" {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []GeneratedFile{
		{Path: project, Content: "new project\n"},
		{Path: intent, Content: testIntent},
	}
	originalRename := renameFile
	defer func() { renameFile = originalRename }()
	renameFile = func(oldPath, newPath string) error {
		if newPath == intent && strings.Contains(filepath.Base(oldPath), ".intent.hcl.tmp.") {
			return errors.New("injected replacement failure")
		}
		return originalRename(oldPath, newPath)
	}
	if err := writeFilesAtomic(root, files, true, nil); err == nil {
		t.Fatal("expected injected transaction failure")
	} else {
		var indeterminate *TransactionError
		if errors.As(err, &indeterminate) {
			t.Fatalf("successful rollback classified indeterminate: %v", err)
		}
	}
	for path, want := range map[string]string{project: "old project\n", intent: "workflow \"old\" {}\n"} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != want {
			t.Fatalf("rolled-back %s = %q, %v", path, data, err)
		}
	}
	assertNoTransactionBackups(t, root)

	renameFile = originalRename
	if err := writeFilesAtomic(root, files, true, nil); err != nil {
		t.Fatal(err)
	}
	assertNoTransactionBackups(t, root)
}

func TestAtomicWriterReportsPostCommitCleanupFailureWithoutFailingCommit(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project.md")
	if err := os.WriteFile(project, []byte("old project\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	originalRemove := removeFile
	defer func() { removeFile = originalRemove }()
	removeFile = func(path string) error {
		if strings.Contains(filepath.Base(path), ".project.md.backup.") {
			return errors.New("injected committed-backup cleanup failure")
		}
		return originalRemove(path)
	}
	result, err := Commit(Prepared{
		ExampleRoot: root,
		Files:       []GeneratedFile{{Path: project, Content: "new project\n"}},
	}, true)
	if err != nil {
		t.Fatalf("committed transaction reported failure: %v", err)
	}
	if len(result.CleanupWarnings) != 1 || !strings.Contains(result.CleanupWarnings[0], "committed-backup cleanup failure") {
		t.Fatalf("cleanup warnings = %#v", result.CleanupWarnings)
	}
	data, err := os.ReadFile(project)
	if err != nil || string(data) != "new project\n" {
		t.Fatalf("committed project = %q, %v", data, err)
	}
}

func TestAtomicWriterClassifiesRollbackFailureIndeterminate(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project.md")
	intent := filepath.Join(root, "workflows", "intent.hcl")
	if err := os.MkdirAll(filepath.Dir(intent), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{project: "old\n", intent: "workflow \"old\" {}\n"} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	originalRename := renameFile
	defer func() { renameFile = originalRename }()
	renameFile = func(oldPath, newPath string) error {
		base := filepath.Base(oldPath)
		if newPath == intent && strings.Contains(base, ".intent.hcl.tmp.") {
			return errors.New("injected replacement failure")
		}
		if newPath == project && strings.Contains(base, ".project.md.backup.") {
			return errors.New("injected rollback failure")
		}
		return originalRename(oldPath, newPath)
	}
	err := writeFilesAtomic(root, []GeneratedFile{
		{Path: project, Content: "new\n"},
		{Path: intent, Content: testIntent},
	}, true, nil)
	var indeterminate *TransactionError
	if !errors.As(err, &indeterminate) {
		t.Fatalf("rollback failure = %T %v, want TransactionError", err, err)
	}
}

func TestAtomicWriterRejectsSymlinksSwapsAndOutsidePaths(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "workflows")); err != nil {
		t.Fatal(err)
	}
	err := writeFilesAtomic(root, []GeneratedFile{{Path: filepath.Join(root, "workflows", "intent.hcl"), Content: testIntent}}, true, nil)
	if err == nil || !strings.Contains(err.Error(), "safe directory") {
		t.Fatalf("descendant symlink error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "intent.hcl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("writer reached outside target: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "workflows")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := []GeneratedFile{
		{Path: filepath.Join(root, "project.md"), Content: "project\n"},
		{Path: filepath.Join(root, "workflows", "intent.hcl"), Content: testIntent},
	}
	err = writeFilesAtomic(root, files, true, func() error {
		if err := os.Rename(filepath.Join(root, "workflows"), filepath.Join(root, "workflows-swapped")); err != nil {
			return err
		}
		return os.Symlink(outside, filepath.Join(root, "workflows"))
	})
	if err == nil || !strings.Contains(err.Error(), "safe directory") {
		t.Fatalf("symlink swap error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "project.md")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial project write survived rollback: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "intent.hcl")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("symlink swap escaped root: %v", err)
	}

	cleanRoot := t.TempDir()
	_, err = Commit(Prepared{ExampleRoot: cleanRoot, Files: []GeneratedFile{{Path: filepath.Join(outside, "escaped.txt"), Content: "escape"}}}, true)
	if err == nil || !strings.Contains(err.Error(), "outside") {
		t.Fatalf("outside output error = %v", err)
	}
}

func assertNoTransactionBackups(t *testing.T, root string) {
	t.Helper()
	var backups []string
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && strings.Contains(entry.Name(), ".backup.") {
			backups = append(backups, path)
		}
		return nil
	})
	if len(backups) != 0 {
		t.Fatalf("transaction backups remain: %v", backups)
	}
}

func assertActionsCoverPreparedFiles(t *testing.T, prepared Prepared, actions []elicitor.FileAction) {
	t.Helper()
	if len(actions) != len(prepared.Files) {
		t.Fatalf("actions cover %d files, prepared transaction has %d: %#v", len(actions), len(prepared.Files), actions)
	}
	for _, file := range prepared.Files {
		action := file.Action
		if action == "" {
			action = "write"
			if file.Remove {
				action = "remove_if_present"
			}
		}
		assertAction(t, actions, action, file.Path)
	}
}

func assertAction(t *testing.T, actions []elicitor.FileAction, action, path string) {
	t.Helper()
	for _, candidate := range actions {
		if candidate.Action == action && candidate.Path == path {
			return
		}
	}
	t.Fatalf("action %s %s not found in %#v", action, path, actions)
}
