package synthesize

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

func TestReviewHandoffInputsIncludesAPIProvenanceManifest(t *testing.T) {
	example := t.TempDir()
	manifest := filepath.Join(example, filepath.FromSlash(packageartifacts.APISourceManifestPath))
	if err := os.MkdirAll(filepath.Dir(manifest), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"version":"openudon.api-source-manifest.v1","sources":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	result := Result{ExampleDir: example,
		ProjectPath:        filepath.Join(example, "project.md"),
		IntentPath:         filepath.Join(example, "workflows", "intent.hcl"),
		WorkflowPath:       filepath.Join(example, "workflows", "workflow.hcl"),
		UWSPath:            filepath.Join(example, "workflows", "workflow.uws.yaml"),
		PlanJSONPath:       filepath.Join(example, "expected", "plan.json"),
		DataPath:           filepath.Join(example, "expected", "data.hcl"),
		RefinementJSONPath: filepath.Join(example, "expected", "refinement.json"),
		ReviewPath:         filepath.Join(example, "expected", "review.md"),
		ReviewHandoffPath:  filepath.Join(example, "expected", "review-handoff.json"),
		QualityJSONPath:    filepath.Join(example, "expected", "quality.json"),
	}
	inputs, err := reviewHandoffInputs(result)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(inputs, func(input ReviewHandoffInput) bool {
		return input.Path == packageartifacts.APISourceManifestPath && input.Required
	}) {
		t.Fatalf("review handoff omitted the required API provenance manifest: %#v", inputs)
	}
}
