//go:build browser_transaction_qualification

package browserscenario

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestRealBAPBCPTransactionQualification(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve qualification repository root")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	evidence, err := RunBAPBCPQualification(ctx, Options{
		RepoRoot: root, BrowsertoolsRepo: filepath.Join(root, "..", "browsertools"),
		UWSRepo: filepath.Join(root, "..", "uws"), UdonRepo: filepath.Join(root, "..", "udon"),
		BrowserdriverRepo: filepath.Join(root, "..", "browserdriver"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBAPBCPQualificationEvidence(evidence); err != nil {
		t.Fatal(err)
	}
}
