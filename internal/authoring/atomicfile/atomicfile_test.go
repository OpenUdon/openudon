package atomicfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAtomicallyReplacesAndAppliesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "evidence.json")
	if err := Write(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("new"), 0o640); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "new" {
		t.Fatalf("content = %q, err=%v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, err=%v", info.Mode(), err)
	}
	assertNoTemps(t, filepath.Dir(path))
}

func TestWriteCleansTemporaryFileWhenRenameFails(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "occupied")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Write(target, []byte("data"), 0o600); err == nil {
		t.Fatal("Write() succeeded over a directory")
	}
	assertNoTemps(t, root)
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		t.Fatalf("failed write changed target: info=%v err=%v", info, err)
	}
}

func TestWriteEmptyPathFails(t *testing.T) {
	if err := Write("  ", []byte("ignored"), 0o600); err == nil {
		t.Fatal("Write accepted an empty path")
	}
	if err := WriteNew("  ", []byte("ignored"), 0o600); err == nil {
		t.Fatal("WriteNew accepted an empty path")
	}
}

func TestWriteNewCreatesWithoutReplacing(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "key.pem")
	if err := WriteNew(path, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteNew(path, []byte("second"), 0o644); err == nil {
		t.Fatal("WriteNew() replaced an existing file")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "first" {
		t.Fatalf("content = %q, err=%v", data, err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, err=%v", info.Mode(), err)
	}
	assertNoTemps(t, filepath.Dir(path))
}

func assertNoTemps(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp.") {
			t.Fatalf("temporary file was not cleaned: %s", entry.Name())
		}
	}
}
