package trustedrunner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/trust"
)

func TestPublicSnapshotInspectionMatchesPrivateLegacyIdentity(t *testing.T) {
	root, example := writeFixture(t, fixtureOptions{})
	legacy, err := resolveAndValidatePackageBytes(root, example)
	if err != nil {
		t.Fatal(err)
	}
	request := trust.Request{Scope: legacy.paths.scope, ManifestPath: "expected/review-handoff.json", Files: map[string][]byte{}, RequiredPaths: legacy.snapshot.paths}
	for _, path := range legacy.snapshot.paths {
		data, err := os.ReadFile(filepath.Join(example, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		request.Files[path] = data
	}
	inspection, err := trust.Inspect(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.PackageSHA256 != legacy.packageSHA256 || inspection.HandoffSHA256 != legacy.handoffSHA256 {
		t.Fatal("legacy canonical identity changed at public boundary")
	}
}
