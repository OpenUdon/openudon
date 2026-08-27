package packagepipeline_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/OpenUdon/openudon/internal/packagepipeline"
	"github.com/OpenUdon/openudon/internal/synthesize"
	"github.com/OpenUdon/openudon/internal/trustedrunner"
)

func TestPreparedDigestMatchesTrustedPackageInspection(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test source")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	example := filepath.Join(root, "examples", "slack-message-audit-log")
	prepared, err := packagepipeline.PrepareCurrent(context.Background(), packagepipeline.PrepareOptions{
		ExampleDir: example, Scope: "examples/slack-message-audit-log",
	})
	if err != nil {
		t.Fatal(err)
	}
	inspection, err := trustedrunner.InspectPackage(context.Background(), trustedrunner.TemplateOptions{
		RepoRoot: root, ExampleDir: example,
		Assess: func(context.Context, synthesize.Options) (*synthesize.QualityReport, error) {
			quality := prepared.Quality()
			return &quality, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := prepared.Manifest()
	if manifest.PackageSHA256 != inspection.PackageSHA256 || manifest.HandoffSHA256 != inspection.HandoffSHA256 {
		t.Fatalf("prepared digest does not match current package inspection: prepared=%#v inspected=%#v", manifest, inspection)
	}
}
