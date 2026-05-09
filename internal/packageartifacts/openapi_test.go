package packageartifacts

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
)

func TestRequiredPackagePathsIncludesFixedAndNestedOpenAPI(t *testing.T) {
	root := t.TempDir()
	writeRequiredPackageFiles(t, root)
	mustWrite(t, filepath.Join(root, "openapi", "nested", "support.yaml"), []byte("openapi: 3.0.0\n"))

	paths, err := RequiredPackagePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"project.md",
		"workflows/intent.hcl",
		"workflows/workflow.hcl",
		"workflows/workflow.uws.yaml",
		"expected/plan.json",
		"expected/quality.json",
		"expected/refinement.json",
		"expected/review.md",
		"expected/symphony-handoff.json",
		"openapi/nested/support.yaml",
	} {
		if !stringSliceContains(paths, want) {
			t.Fatalf("RequiredPackagePaths missing %q in %#v", want, paths)
		}
	}
}

func TestCleanRelativePathRejectsUnsafePaths(t *testing.T) {
	for _, input := range []string{
		"",
		"/project.md",
		"../project.md",
		"workflows/../project.md",
		`workflows\workflow.uws.yaml`,
		"C:/package/project.md",
		"project.md\n",
		"project\x00.md",
	} {
		t.Run(input, func(t *testing.T) {
			if _, err := CleanRelativePath(input); err == nil {
				t.Fatalf("expected %q to be rejected", input)
			}
		})
	}
}

func TestValidateRegularPackageFilesRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.md")
	mustWrite(t, target, []byte("brief\n"))
	if err := os.Symlink(target, filepath.Join(root, "project.md")); err != nil {
		t.Fatal(err)
	}

	err := ValidateRegularPackageFiles(root, []string{"project.md"})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

func TestValidateRegularPackageFilesRejectsSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "workflows")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(outside, "workflow.uws.yaml"), []byte("version: outside\n"))
	if err := os.Symlink(outside, filepath.Join(root, "workflows")); err != nil {
		t.Fatal(err)
	}

	err := ValidateRegularPackageFiles(root, []string{"workflows/workflow.uws.yaml"})
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected parent symlink rejection, got %v", err)
	}
}

func TestValidateRegularPackageFilesRejectsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "project.md"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := ValidateRegularPackageFiles(root, []string{"project.md"})
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected directory rejection, got %v", err)
	}
}

func TestValidateRegularPackageFilesRejectsSpecialFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mkfifo is not portable to windows")
	}
	root := t.TempDir()
	path := filepath.Join(root, "project.md")
	if err := syscall.Mkfifo(path, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}

	err := ValidateRegularPackageFiles(root, []string{"project.md"})
	if err == nil || !strings.Contains(err.Error(), "regular file") {
		t.Fatalf("expected special-file rejection, got %v", err)
	}
}

func TestRequiredManifestPathsRequiresEveryPackagePath(t *testing.T) {
	root := t.TempDir()
	writeRequiredPackageFiles(t, root)
	mustWrite(t, filepath.Join(root, "openapi", "support.yaml"), []byte("openapi: 3.0.0\n"))
	inputs := []ManifestInput{
		{Path: "project.md", Required: true},
		{Path: "workflows/intent.hcl", Required: true},
		{Path: "workflows/workflow.hcl", Required: true},
		{Path: "workflows/workflow.uws.yaml", Required: true},
		{Path: "expected/plan.json", Required: true},
		{Path: "expected/quality.json", Required: true},
		{Path: "expected/refinement.json", Required: true},
		{Path: "expected/review.md", Required: true},
		{Path: "expected/symphony-handoff.json", Required: true},
	}

	_, err := RequiredManifestPaths(root, inputs)
	if err == nil || !strings.Contains(err.Error(), "openapi/support.yaml") {
		t.Fatalf("expected missing OpenAPI manifest input, got %v", err)
	}

	inputs = append(inputs, ManifestInput{Path: "openapi/support.yaml", Required: true})
	paths, err := RequiredManifestPaths(root, inputs)
	if err != nil {
		t.Fatal(err)
	}
	if !stringSliceContains(paths, "openapi/support.yaml") {
		t.Fatalf("manifest paths missing OpenAPI path: %#v", paths)
	}
}

func writeRequiredPackageFiles(t *testing.T, root string) {
	t.Helper()
	for _, rel := range fixedRequiredPackagePaths {
		mustWrite(t, filepath.Join(root, filepath.FromSlash(rel)), []byte(rel+"\n"))
	}
}

func mustWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func stringSliceContains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
