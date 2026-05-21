package elicitor

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestBuildCatalogHintsMatchesWeatherAndGmail(t *testing.T) {
	hints, err := BuildCatalogHints("get weather in Toronto and gmail the report to me", CatalogHintOptions{
		CacheRoot: t.TempDir(),
	})
	if err != nil {
		t.Fatalf("BuildCatalogHints failed: %v", err)
	}
	var ids []string
	for _, hint := range hints {
		ids = append(ids, hint.Provider.ID)
	}
	for _, want := range []string{"gmail", "openweathermap"} {
		if !slices.Contains(ids, want) {
			t.Fatalf("matched providers = %v, want %s", ids, want)
		}
	}
	if got := CatalogProviderPlan(hints); len(got) < 2 || got[0] != "OpenWeatherMap" || got[1] != "Gmail" {
		t.Fatalf("catalog provider plan = %v, want weather before gmail", got)
	}
}

func TestBuildCatalogHintsFindsSiblingArtifacts(t *testing.T) {
	cacheRoot := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "google-discovery/gmail-discovery-v1.json")
	writeCatalogArtifact(t, cacheRoot, "advisory-overlays/openweathermap-one-call-3-overlay.json")

	hints, err := BuildCatalogHints("gmail weather report", CatalogHintOptions{CacheRoot: cacheRoot})
	if err != nil {
		t.Fatalf("BuildCatalogHints failed: %v", err)
	}
	gmail := findCatalogHint(t, hints, "gmail")
	if len(gmail.SpecArtifacts) == 0 || gmail.SpecArtifacts[0].SpecRef.Kind != catalog.SpecKindGoogleDiscovery {
		t.Fatalf("gmail spec artifacts = %#v", gmail.SpecArtifacts)
	}
	if gmail.SpecArtifacts[0].Path == "" {
		t.Fatalf("gmail discovery artifact path was not detected")
	}
	if len(gmail.FollowUps) == 0 {
		t.Fatalf("gmail follow-ups missing")
	}
	weather := findCatalogHint(t, hints, "openweathermap")
	if len(weather.OverlayArtifacts) != 1 {
		t.Fatalf("openweathermap overlay artifacts = %#v", weather.OverlayArtifacts)
	}
}

func TestMigrateCatalogArtifactsCopiesDiscoveryIntoWorkflow(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "google-discovery/gmail-discovery-v1.json")

	result, err := MigrateCatalogArtifacts("gmail a report", example, CatalogHintOptions{CacheRoot: cacheRoot})
	if err != nil {
		t.Fatalf("MigrateCatalogArtifacts failed: %v", err)
	}
	if len(result.Copied) != 1 {
		t.Fatalf("copied = %#v, want one artifact", result.Copied)
	}
	if got, want := result.Copied[0].RelativePath, "discovery/gmail-discovery-v1.json"; got != want {
		t.Fatalf("relative path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(example, "discovery", "gmail-discovery-v1.json")); err != nil {
		t.Fatalf("migrated discovery artifact missing: %v", err)
	}
}

func TestMigrateCatalogArtifactsReportsMissingProviders(t *testing.T) {
	cacheRoot := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "google-discovery/gmail-discovery-v1.json")

	result, err := MigrateCatalogArtifacts("get weather and gmail me", t.TempDir(), CatalogHintOptions{CacheRoot: cacheRoot})
	if err != nil {
		t.Fatalf("MigrateCatalogArtifacts failed: %v", err)
	}
	var missing []string
	for _, hint := range result.Missing {
		missing = append(missing, hint.Provider.ID)
	}
	if !slices.Contains(missing, "openweathermap") {
		t.Fatalf("missing providers = %v, want openweathermap", missing)
	}
}

func TestMigrateCatalogArtifactsCopiesAdvisoryOverlayIntoWorkflow(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "advisory-overlays/openweathermap-one-call-3-overlay.json")

	result, err := MigrateCatalogArtifacts("get weather", example, CatalogHintOptions{CacheRoot: cacheRoot})
	if err != nil {
		t.Fatalf("MigrateCatalogArtifacts failed: %v", err)
	}
	if len(result.Copied) != 1 {
		t.Fatalf("copied = %#v, want one artifact", result.Copied)
	}
	if got, want := result.Copied[0].RelativePath, "openapi/openweathermap-one-call-3-overlay.json"; got != want {
		t.Fatalf("relative path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(example, "openapi", "openweathermap-one-call-3-overlay.json")); err != nil {
		t.Fatalf("migrated advisory overlay missing: %v", err)
	}
}

func TestRetrieveCatalogArtifactsCopiesWeatherOverlay(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "google-discovery/gmail-discovery-v1.json")
	writeCatalogArtifact(t, cacheRoot, "advisory-overlays/openweathermap-one-call-3-overlay.json")
	session := Session{}
	session.Project.Goal = "get weather and gmail me"
	session.Intent.Workflow = &rollout.WorkflowMeta{Description: session.Project.Goal}
	var out strings.Builder

	if err := retrieveCatalogArtifactsForSession(&out, session, example, CatalogHintOptions{CacheRoot: cacheRoot}); err != nil {
		t.Fatalf("retrieveCatalogArtifactsForSession failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "discovery", "gmail-discovery-v1.json")); err != nil {
		t.Fatalf("gmail discovery was not retrieved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "openapi", "openweathermap-one-call-3-overlay.json")); err != nil {
		t.Fatalf("OpenWeatherMap overlay was not retrieved: %v", err)
	}
	if !strings.Contains(out.String(), "retrieved Gmail API document") {
		t.Fatalf("retrieve output missing Gmail retrieval:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "retrieved OpenWeatherMap advisory OpenAPI overlay") {
		t.Fatalf("retrieve output missing OpenWeatherMap retrieval:\n%s", out.String())
	}
}

func TestBuildCatalogPlanRequestUsesCompactArtifactMetadata(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	sourceRel := "advisory-overlays/openweathermap-one-call-3-overlay.json"
	sourcePath := filepath.Join(cacheRoot, filepath.FromSlash(sourceRel))
	if err := os.MkdirAll(filepath.Dir(sourcePath), 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	fullContent := `{"openapi":"3.0.0","paths":{"/secret/full/content":{}}}`
	if err := os.WriteFile(sourcePath, []byte(fullContent), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
	hints, err := BuildCatalogHints("get weather", CatalogHintOptions{CacheRoot: cacheRoot})
	if err != nil {
		t.Fatalf("BuildCatalogHints failed: %v", err)
	}

	request := BuildCatalogPlanRequest("get weather", Session{}, hints, example)
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	text := string(encoded)
	for _, want := range []string{"artifact_key", "relative_path", "provider_id", "openweathermap"} {
		if !strings.Contains(text, want) {
			t.Fatalf("catalog plan request missing %q: %s", want, text)
		}
	}
	for _, forbidden := range []string{sourcePath, "/secret/full/content", `"paths"`} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("catalog plan request leaked %q: %s", forbidden, text)
		}
	}
}

func TestApplyCatalogPlanResponseMigratesValidatedSelectionAndSeedsRoughSteps(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "openapi/test-api.json")
	writeCatalogArtifact(t, cacheRoot, "advisory-overlays/test-overlay.json")
	hints := []CatalogHint{{
		Provider: catalog.Provider{ID: "test", DisplayName: "Test API"},
		SpecArtifacts: []CatalogSpecArtifactHint{{
			SpecRef: catalog.SpecReference{ID: "test-api", Kind: catalog.SpecKindOpenAPI},
			Path:    filepath.Join(cacheRoot, "openapi", "test-api.json"),
		}},
		OverlayArtifacts: []string{filepath.Join(cacheRoot, "advisory-overlays", "test-overlay.json")},
	}}
	request := BuildCatalogPlanRequest("use test api then notify", Session{}, hints, example)
	key := request.Candidates[0].ArtifactKey
	session := Session{}
	var out strings.Builder

	applied, err := applyCatalogPlanResponse(&out, &session, hints, example, CatalogPlanResponse{
		SelectedArtifacts: []CatalogPlanArtifactSelection{{ProviderID: "test", ArtifactKey: key}},
		ProposedSteps: []CatalogPlanStep{{
			Name:     "test_lookup",
			Type:     "http",
			Provider: "test",
			OpenAPI:  "openapi/test-api.json",
			Do:       "Use Test API for the first capability.",
		}},
	})
	if err != nil {
		t.Fatalf("applyCatalogPlanResponse failed: %v", err)
	}
	if !applied.Applied {
		t.Fatalf("catalog plan was not applied: %#v", applied)
	}
	for _, rel := range []string{"openapi/test-api.json", "openapi/test-overlay.json"} {
		if _, err := os.Stat(filepath.Join(example, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("expected migrated artifact %s: %v", rel, err)
		}
	}
	if len(session.Intent.Steps) != 1 {
		t.Fatalf("steps = %#v", session.Intent.Steps)
	}
	step := session.Intent.Steps[0]
	if step.Operation != "" || len(step.With) != 0 || step.Provider != "test" || step.OpenAPI != "openapi/test-api.json" {
		t.Fatalf("unsafe or missing rough step fields: %#v", step)
	}
	if !hasAssumption(session.Assumptions, "catalog_plan_api_docs_migrated") {
		t.Fatalf("missing catalog plan assumption: %#v", session.Assumptions)
	}
	if !strings.Contains(out.String(), "selected Test API API document from catalog plan") {
		t.Fatalf("missing selected artifact output:\n%s", out.String())
	}
}

func TestApplyCatalogPlanResponseRejectsUnknownArtifact(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "openapi/test-api.json")
	hints := []CatalogHint{{
		Provider: catalog.Provider{ID: "test", DisplayName: "Test API"},
		SpecArtifacts: []CatalogSpecArtifactHint{{
			SpecRef: catalog.SpecReference{ID: "test-api", Kind: catalog.SpecKindOpenAPI},
			Path:    filepath.Join(cacheRoot, "openapi", "test-api.json"),
		}},
	}}
	session := Session{}
	var out strings.Builder

	applied, err := applyCatalogPlanResponse(&out, &session, hints, example, CatalogPlanResponse{
		SelectedArtifacts: []CatalogPlanArtifactSelection{{ProviderID: "test", ArtifactKey: "test:openapi/invented.yaml"}},
		ProposedSteps:     []CatalogPlanStep{{Name: "invented", Provider: "test", OpenAPI: "openapi/invented.yaml"}},
	})
	if err != nil {
		t.Fatalf("applyCatalogPlanResponse failed: %v", err)
	}
	if applied.Applied {
		t.Fatalf("invalid catalog plan was applied: %#v", applied)
	}
	if len(session.Intent.Steps) != 0 {
		t.Fatalf("invalid catalog plan seeded steps: %#v", session.Intent.Steps)
	}
	if _, err := os.Stat(filepath.Join(example, "openapi", "invented.yaml")); !os.IsNotExist(err) {
		t.Fatalf("invented artifact exists or stat failed unexpectedly: %v", err)
	}
	if !hasAssumption(session.Assumptions, "catalog_plan_rejected") {
		t.Fatalf("missing rejection assumption: %#v", session.Assumptions)
	}
	if !strings.Contains(out.String(), "rejected unknown catalog artifact test:openapi/invented.yaml") {
		t.Fatalf("missing rejection output:\n%s", out.String())
	}
}

func TestCatalogPlanErrorFallsBackToDeterministicMigration(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "google-discovery/gmail-discovery-v1.json")
	hints, err := BuildCatalogHints("gmail me a report", CatalogHintOptions{CacheRoot: cacheRoot})
	if err != nil {
		t.Fatalf("BuildCatalogHints failed: %v", err)
	}
	session := Session{}
	session.Project.Goal = "gmail me a report"
	session.Intent.Workflow = &rollout.WorkflowMeta{Description: session.Project.Goal}
	var out strings.Builder

	applied, err := planOpeningCatalogArtifacts(context.Background(), &out, errorCatalogPlanExtractor{}, &session, "gmail me a report", hints, Options{
		ExampleDir:         example,
		CatalogHintOptions: CatalogHintOptions{CacheRoot: cacheRoot},
	})
	if err != nil {
		t.Fatalf("planOpeningCatalogArtifacts returned error: %v", err)
	}
	if applied {
		t.Fatalf("errored catalog plan was applied")
	}
	if _, err := os.Stat(filepath.Join(example, "discovery", "gmail-discovery-v1.json")); !os.IsNotExist(err) {
		t.Fatalf("catalog plan error should not copy before fallback: %v", err)
	}

	if _, err := MigrateCatalogArtifacts(catalogQueryForSession(session), example, CatalogHintOptions{CacheRoot: cacheRoot}); err != nil {
		t.Fatalf("deterministic fallback migration failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "discovery", "gmail-discovery-v1.json")); err != nil {
		t.Fatalf("fallback did not migrate discovery artifact: %v", err)
	}
}

type errorCatalogPlanExtractor struct {
	noopExtractor
}

func (errorCatalogPlanExtractor) CatalogPlan(context.Context, CatalogPlanRequest) (CatalogPlanResponse, error) {
	return CatalogPlanResponse{}, errors.New("catalog planner unavailable")
}

func writeCatalogArtifact(t *testing.T, root, rel string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("write artifact: %v", err)
	}
}

func findCatalogHint(t *testing.T, hints []CatalogHint, providerID string) CatalogHint {
	t.Helper()
	for _, hint := range hints {
		if hint.Provider.ID == providerID {
			return hint
		}
	}
	t.Fatalf("provider %s not found in hints %#v", providerID, hints)
	return CatalogHint{}
}
