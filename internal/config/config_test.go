package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSiblingsPassesWhenRequiredDirsExist(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "ramen")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range requiredSiblings {
		if err := os.Mkdir(filepath.Join(parent, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckSiblings(root); err != nil {
		t.Fatalf("CheckSiblings returned error: %v", err)
	}
}

func TestCheckSiblingsReportsMissingSibling(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "ramen")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := CheckSiblings(root); err == nil || !strings.Contains(err.Error(), "required sibling") {
		t.Fatalf("expected required sibling error, got %v", err)
	}
}

func TestCheckSiblingsRejectsFile(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "ramen")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range requiredSiblings {
		path := filepath.Join(parent, name)
		if name == requiredSiblings[0] {
			if err := os.WriteFile(path, []byte("not a directory"), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := CheckSiblings(root); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not a directory error, got %v", err)
	}
}
