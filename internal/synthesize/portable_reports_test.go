package synthesize

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/openapidisco"
)

func TestPrepareRefinementRejectsUnsafeOrOversizedProjectBrief(t *testing.T) {
	for _, test := range []struct {
		name  string
		setup func(*testing.T, string)
	}{
		{name: "symlink", setup: func(t *testing.T, path string) {
			target := filepath.Join(t.TempDir(), "outside-project.md")
			if err := os.WriteFile(target, []byte("# Outside\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, path); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
		}},
		{name: "oversized", setup: func(t *testing.T, path string) {
			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
			if err != nil {
				t.Fatal(err)
			}
			if err := file.Truncate(8<<20 + 1); err != nil {
				_ = file.Close()
				t.Fatal(err)
			}
			if err := file.Close(); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			example := t.TempDir()
			test.setup(t, filepath.Join(example, "project.md"))
			if _, err := prepareRefinement(context.Background(), Options{ExampleDir: example}); err == nil {
				t.Fatalf("prepareRefinement accepted %s project.md", test.name)
			}
		})
	}
}

func TestPersistedReportsContainOnlyPackageRelativePaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "stable-example")
	result := resultPaths(root)
	result.OpenAPICandidates = []openapidisco.Candidate{{
		Path: filepath.Join(root, "openapi", "service.yaml"), RelativePath: "openapi/service.yaml",
		Title: "Service under " + root, Description: "loaded from " + root,
		Source: "url:https://operator:secret@example.com/openapi.json?api_key=secret-value#fragment",
	}}
	report := &QualityReport{
		Status: "fail", Example: root, Artifacts: result,
		Checks: []QualityCheck{{Code: "test", Status: "fail", Message: "failed under " + root, Detail: filepath.Join(root, "project.md")}},
	}
	refinement := &RefinementReport{
		Status: "fail", Example: root, MaxAttempts: 1, PromptVersion: "test",
		Attempts: []RefinementAttempt{{Number: 1, Status: "fail", Detail: "read " + filepath.Join(root, "project.md")}},
	}
	if err := writeRefinementReport(result, refinement); err != nil {
		t.Fatal(err)
	}
	if err := writeQualityFiles(result, report); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{result.QualityJSONPath, result.QualityMDPath, result.RefinementJSONPath, result.RefinementMDPath} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), root) || strings.Contains(string(data), filepath.Dir(root)) ||
			strings.Contains(string(data), "operator:secret") || strings.Contains(string(data), "secret-value") {
			t.Fatalf("%s contains checkout or temporary root:\n%s", path, data)
		}
	}
	data, err := os.ReadFile(result.QualityJSONPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"source": "url:https://example.com/openapi.json"`) {
		t.Fatalf("quality report does not retain a safe source label:\n%s", data)
	}
}

func TestDiscoveryReportAndReviewUsePortableSourcePaths(t *testing.T) {
	root := filepath.Join(t.TempDir(), "portable-example")
	result := resultPaths(root)
	report := portableDiscoveryReport(result, openapidisco.DiscoveryReport{Attempts: []openapidisco.DiscoveryAttempt{{
		Kind: "local", Source: filepath.Join(root, "openapi"), Status: "pass", Detail: "scanned " + root,
	}, {
		Kind: "url", Source: "https://operator:secret@example.com/openapi.json?api_key=secret#fragment", Status: "pass",
	}}})
	if report.Attempts[0].Source != "openapi" || strings.Contains(report.Attempts[0].Detail, root) {
		t.Fatalf("local discovery attempt was not made portable: %#v", report.Attempts[0])
	}
	if got := report.Attempts[1].Source; got != "https://example.com/openapi.json" {
		t.Fatalf("remote discovery source = %q", got)
	}
}

func TestFailedRefinementDoesNotRebindPassingHandoff(t *testing.T) {
	root := t.TempDir()
	result := resultPaths(root)
	if err := os.MkdirAll(filepath.Dir(result.RefinementJSONPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.QualityJSONPath, []byte("old quality\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(result.ReviewHandoffPath, []byte("old handoff\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := newRefinementReport(result, 1)
	report.addAttempt(1, "generate_intent", nil, errors.New("provider failed"), "intent generation failed")
	if err := writeRefinementReport(result, report); err != nil {
		t.Fatal(err)
	}
	handoff, err := os.ReadFile(result.ReviewHandoffPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(handoff) != "old handoff\n" {
		t.Fatalf("failed refinement rewrote existing handoff: %q", handoff)
	}
}
