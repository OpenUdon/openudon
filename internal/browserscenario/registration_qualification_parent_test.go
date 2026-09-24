package browserscenario

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRegistrationQualificationExampleParentCleanup(t *testing.T) {
	t.Run("removes created parents after example cleanup", func(t *testing.T) {
		root := t.TempDir()
		parent, cleanup, err := createRegistrationQualificationExampleParent(root)
		if err != nil {
			t.Fatal(err)
		}
		example, err := os.MkdirTemp(parent, ".e11-brp-")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(example); err != nil {
			t.Fatal(err)
		}
		if err := cleanup(); err != nil {
			t.Fatal(err)
		}
		for _, directory := range []string{filepath.Join(root, "eval", "runs"), filepath.Join(root, "eval")} {
			if _, err := os.Lstat(directory); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("created directory remains: %s (err %v)", directory, err)
			}
		}
	})

	t.Run("preserves preexisting parent and contents", func(t *testing.T) {
		root := t.TempDir()
		runs := filepath.Join(root, "eval", "runs")
		if err := os.MkdirAll(runs, 0o700); err != nil {
			t.Fatal(err)
		}
		sentinel := filepath.Join(runs, "keep")
		if err := os.WriteFile(sentinel, []byte("fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, cleanup, err := createRegistrationQualificationExampleParent(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := cleanup(); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(sentinel); err != nil {
			t.Fatalf("preexisting content was removed: %v", err)
		}
	})

	t.Run("removes only newly created child", func(t *testing.T) {
		root := t.TempDir()
		evalDir := filepath.Join(root, "eval")
		if err := os.Mkdir(evalDir, 0o700); err != nil {
			t.Fatal(err)
		}
		_, cleanup, err := createRegistrationQualificationExampleParent(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := cleanup(); err != nil {
			t.Fatal(err)
		}
		if info, err := os.Stat(evalDir); err != nil || !info.IsDir() {
			t.Fatalf("preexisting eval directory changed: info=%v err=%v", info, err)
		}
		if _, err := os.Lstat(filepath.Join(evalDir, "runs")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("new runs directory remains: %v", err)
		}
	})

	t.Run("rejects symlink parent", func(t *testing.T) {
		root := t.TempDir()
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(root, "eval")); err != nil {
			t.Skipf("symlink unavailable: %v", err)
		}
		if _, _, err := createRegistrationQualificationExampleParent(root); err == nil {
			t.Fatal("symlink parent was accepted")
		}
		if _, err := os.Stat(outside); err != nil {
			t.Fatalf("symlink target was affected: %v", err)
		}
	})

	t.Run("preserves unexpected nonempty created parents", func(t *testing.T) {
		root := t.TempDir()
		parent, cleanup, err := createRegistrationQualificationExampleParent(root)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(parent, "unrelated"), []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := cleanup(); err == nil {
			t.Fatal("nonempty created parent was silently accepted")
		}
		if _, err := os.Stat(filepath.Join(parent, "unrelated")); err != nil {
			t.Fatalf("unexpected content was removed: %v", err)
		}
	})
}
