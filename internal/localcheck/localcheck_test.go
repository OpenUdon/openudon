package localcheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckAPIToolsBoundaryRejectsRemovedLifecycleImport(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "bad.go"), []byte(`package bad

import "github.com/OpenUdon/apitools/llm"

var _ = llm.Client{}
`))

	err := CheckAPIToolsBoundary(root)
	if err == nil || !strings.Contains(err.Error(), "apitools/llm") {
		t.Fatalf("expected blocked import failure, got %v", err)
	}
}

func TestCheckAPIToolsBoundaryIgnoresTests(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "bad_test.go"), []byte(`package bad

import "github.com/OpenUdon/apitools/llm"
`))

	if err := CheckAPIToolsBoundary(root); err != nil {
		t.Fatalf("CheckAPIToolsBoundary returned error for test file: %v", err)
	}
}

func TestCheckAPIToolsBoundaryRejectsOpenTofuInternals(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "bad.go"), []byte(`package bad

import "github.com/opentofu/opentofu/internal/configs"

var _ configs.Config
`))

	err := CheckAPIToolsBoundary(root)
	if err == nil || !strings.Contains(err.Error(), "opentofu/opentofu/internal/configs") {
		t.Fatalf("expected blocked OpenTofu import failure, got %v", err)
	}
}

func TestCheckAPIToolsBoundaryRejectsPrivateExecutorImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "bad.go"), []byte(`package bad

import "github.com/OpenUdon/udon/runtime"

var _ = runtime.Executor{}
`))
	writeFile(t, filepath.Join(root, "internal", "also_bad.go"), []byte(`package bad

import "github.com/genelet/udon/arazzo"

var _ = arazzo.Flow{}
`))

	err := CheckAPIToolsBoundary(root)
	if err == nil || !strings.Contains(err.Error(), "github.com/OpenUdon/udon/runtime") || !strings.Contains(err.Error(), "github.com/genelet/udon/arazzo") {
		t.Fatalf("expected private executor import failure, got %v", err)
	}
}

func TestCheckAPIToolsBoundaryAllowsPublicIndirectDependencyImports(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "ok.go"), []byte(`package ok

import "github.com/genelet/horizon/dethcl"

var _ = dethcl.Body{}
`))

	if err := CheckAPIToolsBoundary(root); err != nil {
		t.Fatalf("CheckAPIToolsBoundary returned error for public indirect dependency import: %v", err)
	}
}

func TestCheckAPIToolsBoundaryRejectsTFConfigPackage(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "internal", "bad.go"), []byte(`package bad

import "github.com/OpenUdon/tfconfig"

var _ tfconfig.Document
`))

	err := CheckAPIToolsBoundary(root)
	if err == nil || !strings.Contains(err.Error(), "github.com/OpenUdon/tfconfig") {
		t.Fatalf("expected tfconfig import failure, got %v", err)
	}
}

func TestCheckAPIToolsBoundaryIgnoresGitIgnoredFiles(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init")
	writeFile(t, filepath.Join(root, ".gitignore"), []byte(".openudon-run/\n"))
	writeFile(t, filepath.Join(root, ".openudon-run", "bad.go"), []byte(`package bad

import "github.com/OpenUdon/apitools/llm"
`))

	if err := CheckAPIToolsBoundary(root); err != nil {
		t.Fatalf("CheckAPIToolsBoundary returned error for ignored file: %v", err)
	}
}

func TestCheckDocMemoryRequiresMemoryBankFiles(t *testing.T) {
	root := t.TempDir()
	_, err := CheckDocMemory(root)
	if err == nil || !strings.Contains(err.Error(), "memory-bank/product.md") {
		t.Fatalf("expected missing memory bank file, got %v", err)
	}
}

func TestCheckDocMemoryRejectsStaleReferences(t *testing.T) {
	root := t.TempDir()
	writeRequiredMemoryFiles(t, root)
	stale := "TODO" + ".md"
	writeFile(t, filepath.Join(root, "docs", "note.md"), []byte("See "+stale+"\n"))

	_, err := CheckDocMemory(root)
	if err == nil || !strings.Contains(err.Error(), stale) {
		t.Fatalf("expected stale reference failure, got %v", err)
	}
}

func TestCheckDocMemoryPassesWithRequiredFiles(t *testing.T) {
	root := t.TempDir()
	writeRequiredMemoryFiles(t, root)

	result, err := CheckDocMemory(root)
	if err != nil {
		t.Fatalf("CheckDocMemory returned error: %v", err)
	}
	if len(result.CheckedFiles) != len(RequiredMemoryFiles) {
		t.Fatalf("checked files = %#v", result.CheckedFiles)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func writeRequiredMemoryFiles(t *testing.T, root string) {
	t.Helper()
	for _, rel := range RequiredMemoryFiles {
		writeFile(t, filepath.Join(root, filepath.FromSlash(rel)), []byte(rel+"\n"))
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
