package browsersystem

import (
	"context"
	"encoding/json"
	"github.com/OpenUdon/openudon/internal/browsercheck"
	"os"
	"path/filepath"
	"testing"
)

func currentModulesFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join(root, "browserdriver")
	modules := filepath.Join(root, "modules")
	for _, p := range []string{source, filepath.Join(modules, ".bin")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	versions := map[string]string{"@types/node": "24.5.2", "playwright": "1.62.1", "playwright-core": "1.62.1", "typescript": "5.9.2"}
	packages := map[string]any{}
	for name, version := range versions {
		packages["node_modules/"+name] = map[string]string{"version": version}
		p := filepath.Join(modules, name)
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(p, "package.json"), []byte(`{"version":"`+version+`"}`), 0600); err != nil {
			t.Fatal(err)
		}
	}
	data, _ := json.Marshal(map[string]any{"packages": packages})
	if err := os.WriteFile(filepath.Join(source, "package-lock.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tsc", "tsserver", "playwright", "playwright-core"} {
		target := filepath.Join(modules, "bin-"+name)
		if err := os.WriteFile(target, []byte("#!/usr/bin/node\n"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../bin-"+name, filepath.Join(modules, ".bin", name)); err != nil {
			t.Fatal(err)
		}
	}
	return source, modules
}
func TestCurrentInputModulesRejectUnsafeOrUnreadyDependencies(t *testing.T) {
	for _, variant := range []string{"valid", "relative", "alias", "missing_bin", "nonexecutable", "escaped_bin", "version"} {
		t.Run(variant, func(t *testing.T) {
			source, modules := currentModulesFixture(t)
			switch variant {
			case "relative":
				modules = "relative"
			case "alias":
				alias := filepath.Join(t.TempDir(), "alias")
				if err := os.Symlink(modules, alias); err != nil {
					t.Fatal(err)
				}
				modules = alias
			case "missing_bin":
				if err := os.Remove(filepath.Join(modules, ".bin", "tsc")); err != nil {
					t.Fatal(err)
				}
			case "nonexecutable":
				if err := os.Chmod(filepath.Join(modules, "bin-tsc"), 0600); err != nil {
					t.Fatal(err)
				}
			case "escaped_bin":
				if err := os.Remove(filepath.Join(modules, ".bin", "tsc")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("/usr/bin/node", filepath.Join(modules, ".bin", "tsc")); err != nil {
					t.Fatal(err)
				}
			case "version":
				if err := os.WriteFile(filepath.Join(modules, "typescript", "package.json"), []byte(`{"version":"0.0.0"}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := validateCurrentInputModules(source, modules); (err == nil) != (variant == "valid") {
				t.Fatalf("readiness = %v", err)
			}
		})
	}
}
func TestCurrentInputModuleInventoryBindsBytesAndExecutableModes(t *testing.T) {
	source, modules := currentModulesFixture(t)
	if err := validateCurrentInputModules(source, modules); err != nil {
		t.Fatal(err)
	}
	before, err := browsercheck.TreeDigest(modules, true)
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(modules, "bin-tsc")
	if err := os.Chmod(p, 0500); err != nil {
		t.Fatal(err)
	}
	after, err := browsercheck.TreeDigest(modules, true)
	if err != nil || before == after {
		t.Fatal("executable mode change omitted", err)
	}
	if err := os.WriteFile(filepath.Join(modules, "typescript", "package.json"), []byte(`{"version":"5.9.2","extra":"synthetic"}`), 0600); err != nil {
		t.Fatal(err)
	}
	changed, err := browsercheck.TreeDigest(modules, true)
	if err != nil || changed == after {
		t.Fatal("module byte change omitted", err)
	}
}
func TestCurrentQualificationInputRejectsMissingUnsafeAndCancelledInputs(t *testing.T) {
	root := t.TempDir()
	for _, modules := range []string{"", "relative", filepath.Join(root, "missing")} {
		if got, err := CurrentQualificationInput(context.Background(), root, root, modules); err == nil || got != (InputIdentity{}) {
			t.Fatal("unready inventory accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CurrentQualificationInput(ctx, root, root, filepath.Join(root, "missing")); err == nil {
		t.Fatal("cancelled inventory accepted")
	}
}
