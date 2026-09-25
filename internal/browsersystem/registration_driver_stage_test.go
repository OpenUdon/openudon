package browsersystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// This opt-in runs the exact synthetic Browserdriver live-test stage used by
// current native qualification. It protects ESM resolution through the
// separate read-only node_modules link without making ordinary Go tests launch
// a browser.
func TestBrowserdriverRegistrationDriverStageOptIn(t *testing.T) {
	if os.Getenv("OPENUDON_BROWSER_SYSTEM_TEST_REGISTRATION_DRIVER") != "1" {
		t.Skip("set OPENUDON_BROWSER_SYSTEM_TEST_REGISTRATION_DRIVER=1 for the isolated Browserdriver stage")
	}
	modules := os.Getenv("OPENUDON_BROWSER_SYSTEM_BROWSERDRIVER_NODE_MODULES")
	if modules == "" {
		t.Fatal("OPENUDON_BROWSER_SYSTEM_BROWSERDRIVER_NODE_MODULES is required")
	}
	workingDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal("OpenUdon test directory is unavailable")
	}
	root := filepath.Clean(filepath.Join(workingDirectory, "..", ".."))
	browserdriverRoot := filepath.Join(filepath.Dir(root), "browserdriver")
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	result, err := nodeTests(ctx, browserdriverRoot, true, modules)
	if err != nil {
		t.Fatal("Browserdriver registration driver stage failed")
	}
	if result.Passed != 5 || result.Skipped != 0 {
		t.Fatalf("Browserdriver registration driver stage = %d passed, %d skipped; want 5 passed and no skips", result.Passed, result.Skipped)
	}
}
