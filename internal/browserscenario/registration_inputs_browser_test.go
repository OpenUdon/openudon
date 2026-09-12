//go:build browser_system_qualification

package browserscenario

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/processgroup"
)

func TestTypedRegistrationUIToTrustedRuntime(t *testing.T) {
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal("source")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()
	root := t.TempDir()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal("node")
	}
	driver := filepath.Join(root, "driver")
	if err := StageBrowserdriver(ctx, filepath.Join(filepath.Dir(repo), "browserdriver"), driver); err != nil {
		t.Fatal(err)
	}
	executor := &realExecutor{root: root, node: node, driverEntry: filepath.Join(driver, "dist", "src", "index.js"), udon: filepath.Join(root, "udon"), browsertools: filepath.Join(root, "browsertools")}
	for _, component := range []struct{ name, path string }{{"browsertools", executor.browsertools}, {"udon", executor.udon}} {
		if err := processgroup.Run(ctx, 2*time.Minute, processgroup.Invocation{Args: []string{"go", "build", "-o", component.path, "./cmd/" + component.name}, Dir: filepath.Join(filepath.Dir(repo), component.name), Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard}); err != nil {
			t.Fatal("build " + component.name)
		}
	}
	evidence, err := executor.runBRPQualification(ctx, Environment{RepoRoot: repo})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBRPQualificationEvidence(evidence); err != nil {
		t.Fatal(err)
	}
}
