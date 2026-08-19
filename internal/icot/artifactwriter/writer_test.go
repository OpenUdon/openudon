package artifactwriter

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
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
		prepared, err := Prepare(example, elicitor.Artifacts{ProjectMD: "project\n", IntentHCL: testIntent}, true, now)
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

func TestWriteConflictsMatchCommitOverwritePreflight(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project.md")
	intent := filepath.Join(root, "workflows", "intent.hcl")
	source := filepath.Join(root, "browser-profiles", "status.json")
	metadata := filepath.Join(root, ".icot", "session.yaml")
	for _, path := range []string{project, intent, source, metadata} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("existing\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	prepared := Prepared{ExampleRoot: root, Files: []GeneratedFile{
		{Path: source, Content: "replacement\n", Action: "copy"},
		{Path: metadata, Content: "replacement\n", Action: "write", AllowOverwrite: true},
		{Path: filepath.Join(root, ".icot", "readiness.json"), Remove: true, Action: "remove_if_present"},
		{Path: intent, Content: testIntent, Action: "write"},
		{Path: project, Content: "replacement\n", Action: "write"},
	}}
	conflicts, err := WriteConflicts(prepared)
	if err != nil {
		t.Fatal(err)
	}
	want := []WriteConflict{
		{Code: "overwrite_required", Action: "copy", Path: source},
		{Code: "overwrite_required", Action: "write", Path: project},
		{Code: "overwrite_required", Action: "write", Path: intent},
	}
	sort.SliceStable(want, func(i, j int) bool {
		if want[i].Path != want[j].Path {
			return want[i].Path < want[j].Path
		}
		return want[i].Action < want[j].Action
	})
	if !reflect.DeepEqual(conflicts, want) {
		t.Fatalf("write conflicts = %#v, want %#v", conflicts, want)
	}
	if err := writeFilesAtomic(root, prepared.Files, false, nil); err == nil || !strings.Contains(err.Error(), "pass --force") {
		t.Fatalf("commit without overwrite authority = %v", err)
	}
}

func TestWriteConflictsIsReadOnlyAndRejectsUnsafePaths(t *testing.T) {
	root := t.TempDir()
	missing := filepath.Join(root, "new", "nested", "file.txt")
	conflicts, err := WriteConflicts(Prepared{ExampleRoot: root, Files: []GeneratedFile{{Path: missing, Content: "new\n", Action: "write"}}})
	if err != nil || len(conflicts) != 0 {
		t.Fatalf("missing output conflicts = %#v, %v", conflicts, err)
	}
	if _, err := os.Stat(filepath.Dir(missing)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("conflict inspection created directories: %v", err)
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "unsafe")); err != nil {
		t.Fatal(err)
	}
	_, err = WriteConflicts(Prepared{ExampleRoot: root, Files: []GeneratedFile{{Path: filepath.Join(root, "unsafe", "file.txt"), Content: "new\n"}}})
	if err == nil || !strings.Contains(err.Error(), "safe directory") {
		t.Fatalf("unsafe conflict path error = %v", err)
	}

	directLink := filepath.Join(root, "allowed-link.json")
	if err := os.Symlink(filepath.Join(outside, "target.json"), directLink); err != nil {
		t.Fatal(err)
	}
	_, err = WriteConflicts(Prepared{ExampleRoot: root, Files: []GeneratedFile{{
		Path: directLink, Content: "replacement\n", AllowOverwrite: true,
	}}})
	if err == nil || !strings.Contains(err.Error(), "is a symlink") {
		t.Fatalf("allow-overwrite symlink error = %v", err)
	}

	removeThroughLink := filepath.Join(root, "unsafe", "obsolete.json")
	_, err = WriteConflicts(Prepared{ExampleRoot: root, Files: []GeneratedFile{{
		Path: removeThroughLink, Remove: true, AllowOverwrite: true,
	}}})
	if err == nil || !strings.Contains(err.Error(), "safe directory") {
		t.Fatalf("removal through symlink error = %v", err)
	}

	directoryTarget := filepath.Join(root, "metadata")
	if err := os.Mkdir(directoryTarget, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err = WriteConflicts(Prepared{ExampleRoot: root, Files: []GeneratedFile{{
		Path: directoryTarget, Content: "replacement\n", AllowOverwrite: true,
	}}})
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("allow-overwrite directory error = %v", err)
	}
}

func TestPreparedPlanRejectsAmbiguousPathsBeforeFilesystemMutation(t *testing.T) {
	tests := []struct {
		name  string
		files func(string) []GeneratedFile
		want  string
	}{
		{
			name: "duplicate",
			files: func(root string) []GeneratedFile {
				path := filepath.Join(root, "generated", "same.json")
				return []GeneratedFile{{Path: path, Content: "one\n"}, {Path: path, Content: "two\n"}}
			},
			want: "duplicate or case-insensitive-equivalent",
		},
		{
			name: "case-folded",
			files: func(root string) []GeneratedFile {
				return []GeneratedFile{
					{Path: filepath.Join(root, "generated", "Source.json"), Content: "one\n"},
					{Path: filepath.Join(root, "generated", "source.JSON"), Content: "two\n"},
				}
			},
			want: "duplicate or case-insensitive-equivalent",
		},
		{
			name: "ancestor-descendant",
			files: func(root string) []GeneratedFile {
				return []GeneratedFile{
					{Path: filepath.Join(root, "generated", "source"), Content: "one\n"},
					{Path: filepath.Join(root, "generated", "source", "child.json"), Content: "two\n"},
				}
			},
			want: "overlapping ancestor and descendant",
		},
		{
			name: "remove-write-collision",
			files: func(root string) []GeneratedFile {
				path := filepath.Join(root, "generated", "collision.json")
				return []GeneratedFile{{Path: path, Remove: true}, {Path: path, Content: "replacement\n"}}
			},
			want: "duplicate or case-insensitive-equivalent",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent := t.TempDir()
			root := filepath.Join(parent, "workspace")
			if err := os.Mkdir(root, 0o755); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(root, "keep.txt")
			if err := os.WriteFile(sentinel, []byte("byte-identical sentinel\n"), 0o640); err != nil {
				t.Fatal(err)
			}
			before := snapshotFilesystem(t, root)
			prepared := Prepared{ExampleRoot: root, Files: test.files(root)}

			if _, err := WriteConflicts(prepared); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("WriteConflicts error = %v, want %q", err, test.want)
			}
			if after := snapshotFilesystem(t, root); !reflect.DeepEqual(after, before) {
				t.Fatalf("conflict inspection changed filesystem\nbefore: %#v\nafter: %#v", before, after)
			}
			if _, err := Commit(prepared, true); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Commit error = %v, want %q", err, test.want)
			}
			if after := snapshotFilesystem(t, root); !reflect.DeepEqual(after, before) {
				t.Fatalf("rejected commit changed filesystem\nbefore: %#v\nafter: %#v", before, after)
			}
		})
	}
}

func TestPrepareRejectsReservedAndOverlappingSourceTargets(t *testing.T) {
	content := []byte("reviewed source\n")
	digest := fmt.Sprintf("%x", sha256.Sum256(content))
	source := func(id, target string) elicitor.SourceMaterialization {
		return elicitor.SourceMaterialization{
			Kind: "openapi", ID: id, SourcePath: id + ".json", TargetPath: target,
			SHA256: digest, Provenance: "reviewed test", MaterializedContent: append([]byte(nil), content...),
		}
	}
	tests := []struct {
		name    string
		targets []string
		want    string
	}{
		{name: "icot-state", targets: []string{".icot/copied.json"}, want: "reserved"},
		{name: "case-folded-icot-state", targets: []string{".ICOT/copied.json"}, want: "reserved"},
		{name: "project", targets: []string{"project.md"}, want: "reserved"},
		{name: "case-folded-project", targets: []string{"PROJECT.MD"}, want: "reserved"},
		{name: "final-intent", targets: []string{"workflows/intent.hcl"}, want: "reserved"},
		{name: "draft-intent", targets: []string{"workflows/intent.draft.hcl"}, want: "reserved"},
		{name: "duplicate", targets: []string{"openapi/source.json", "openapi/./source.json"}, want: "duplicate or case-insensitive-equivalent"},
		{name: "case-folded", targets: []string{"openapi/Source.json", "OPENAPI/source.JSON"}, want: "duplicate or case-insensitive-equivalent"},
		{name: "ancestor-descendant", targets: []string{"openapi/source", "openapi/source/schema.json"}, want: "overlapping ancestor and descendant"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "workspace")
			sources := make([]elicitor.SourceMaterialization, 0, len(test.targets))
			for index, target := range test.targets {
				sources = append(sources, source(fmt.Sprintf("source-%d", index), target))
			}
			_, err := Prepare(root, elicitor.Artifacts{
				ProjectMD: "project\n", IntentHCL: testIntent,
				Session: elicitor.Session{SourcePlan: sources},
			}, true, time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Prepare error = %v, want %q", err, test.want)
			}
			if _, statErr := os.Lstat(root); !errors.Is(statErr, os.ErrNotExist) {
				t.Fatalf("rejected source plan left filesystem residue: %v", statErr)
			}
		})
	}
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

type filesystemEntry struct {
	Mode    os.FileMode
	Content string
}

func snapshotFilesystem(t *testing.T, root string) map[string]filesystemEntry {
	t.Helper()
	result := map[string]filesystemEntry{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		value := filesystemEntry{Mode: info.Mode()}
		if info.Mode().IsRegular() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			value.Content = string(data)
		}
		result[filepath.ToSlash(relative)] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
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
