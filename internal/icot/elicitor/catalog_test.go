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
	if len(result.Copied) != 2 {
		t.Fatalf("copied = %#v, want source plus security overlay", result.Copied)
	}
	if got, want := result.Copied[0].RelativePath, "google-discovery/gmail-discovery-v1.json"; got != want {
		t.Fatalf("relative path = %q, want %q", got, want)
	}
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "gmail-discovery-v1.json")); err != nil {
		t.Fatalf("migrated discovery artifact missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "gmail-discovery-v1.security-overlay.json")); err != nil {
		t.Fatalf("migrated discovery security overlay missing: %v", err)
	}
}

func TestMigrationResultFromExportCopiesSecurityOverlaySidecar(t *testing.T) {
	example := t.TempDir()
	exportRoot := t.TempDir()
	sourcePath := filepath.Join(exportRoot, "gmail", "google-discovery", "gmail-discovery-v1.json")
	overlayPath := filepath.Join(exportRoot, "gmail", "security-overlays", "gmail-discovery-auth-overlay.json")
	mustWriteCatalogTestFile(t, sourcePath, []byte("{}\n"))
	mustWriteCatalogTestFile(t, overlayPath, []byte(`{
  "id": "gmail-discovery-auth-overlay",
  "provider_id": "gmail",
  "spec_ref_id": "gmail-discovery-v1",
  "status": "overlay-required",
  "security_schemes": [{"name":"googleOAuth2","type":"oauth2"}],
  "root_security": [{"scheme":"googleOAuth2"}]
}`))

	result, err := migrationResultFromExport(catalog.ExportReport{
		Providers: []catalog.MaterializationReport{{
			ProviderID:  "gmail",
			DisplayName: "Gmail",
			Artifacts: []catalog.MaterializedArtifact{{
				ArtifactID:    "gmail-discovery-v1",
				SpecRefID:     "gmail-discovery-v1",
				Kind:          "google-discovery",
				Protocol:      catalog.SpecProtocolGoogleDiscovery,
				UWSSourceType: "google-discovery",
				SourcePath:    sourcePath,
				TargetPath:    sourcePath,
				SourceURL:     "https://gmail.googleapis.com/$discovery/rest?version=v1",
			}},
			SecurityOverlays: []catalog.MaterializedSecurityOverlay{{
				OverlayID:  "gmail-discovery-auth-overlay",
				SpecRefID:  "gmail-discovery-v1",
				Status:     catalog.AuthStatusOverlayRequired,
				TargetPath: overlayPath,
			}},
		}},
	}, nil, example)
	if err != nil {
		t.Fatalf("migrationResultFromExport failed: %v", err)
	}
	if len(result.Copied) != 2 {
		t.Fatalf("copied = %#v, want source and sidecar", result.Copied)
	}
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "gmail-discovery-v1.security-overlay.json")); err != nil {
		t.Fatalf("exported security overlay sidecar missing: %v", err)
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
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "gmail-discovery-v1.json")); err != nil {
		t.Fatalf("gmail discovery was not retrieved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "gmail-discovery-v1.security-overlay.json")); err != nil {
		t.Fatalf("gmail discovery security overlay was not retrieved: %v", err)
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

func TestCatalogSecurityOverlayTargetRelativePathSupportsSmithy(t *testing.T) {
	got := catalogSecurityOverlayTargetRelativePath("aws-smithy/s3.json")
	if want := "aws-smithy/s3.security-overlay.json"; got != want {
		t.Fatalf("security overlay sidecar path = %q, want %q", got, want)
	}
	got = catalogArtifactTargetRelativePath(catalog.SpecReference{ID: "s3", Kind: catalog.SpecKindSmithyJSON}, filepath.Join(t.TempDir(), "s3.json"))
	if want := "aws-smithy/s3.json"; got != want {
		t.Fatalf("smithy target path = %q, want %q", got, want)
	}
}

func TestSiblingCatalogSpecArtifactCandidatesUseSourceFamilyExtensions(t *testing.T) {
	cacheRoot := t.TempDir()
	for _, tc := range []struct {
		name string
		ref  catalog.SpecReference
		file string
	}{
		{name: "graphql", ref: catalog.SpecReference{ID: "schema", Kind: catalog.SpecKindGraphQL}, file: "graphql/schema.graphql"},
		{name: "openrpc", ref: catalog.SpecReference{ID: "math", Kind: catalog.SpecKindOpenRPC}, file: "openrpc/math.json"},
		{name: "grpc protobuf", ref: catalog.SpecReference{ID: "trace", Kind: catalog.SpecKindGRPCProtobuf}, file: "grpc-protobuf/trace.proto"},
		{name: "odata xml", ref: catalog.SpecReference{ID: "service", Kind: catalog.SpecKindOData}, file: "odata/service.xml"},
		{name: "odata json", ref: catalog.SpecReference{ID: "csdl", Kind: catalog.SpecKindOData}, file: "odata/csdl.json"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			target := filepath.Join(cacheRoot, filepath.FromSlash(tc.file))
			writeCatalogArtifact(t, cacheRoot, tc.file)
			if got := siblingCatalogSpecArtifactPath(cacheRoot, tc.ref); got != filepath.ToSlash(target) {
				t.Fatalf("sibling artifact path = %q, want %q", got, filepath.ToSlash(target))
			}
		})
	}
}

func TestMaterializeSecurityOverlaySidecarsMatchesSourceURL(t *testing.T) {
	example := t.TempDir()
	sourceA := CatalogMigrationCandidate{
		ProviderID:   "demo",
		ProviderName: "Demo",
		SpecRefID:    "a",
		SourceURL:    "https://example.test/a.json",
		Kind:         catalog.SpecKindOpenAPI,
		Protocol:     string(catalog.SpecProtocolOpenAPI),
		RelativePath: "openapi/a.json",
		TargetPath:   filepath.Join(example, "openapi", "a.json"),
	}
	sourceB := CatalogMigrationCandidate{
		ProviderID:   "demo",
		ProviderName: "Demo",
		SpecRefID:    "b",
		SourceURL:    "https://example.test/b.json",
		Kind:         catalog.SpecKindGoogleDiscovery,
		Protocol:     string(catalog.SpecProtocolGoogleDiscovery),
		RelativePath: "google-discovery/b.json",
		TargetPath:   filepath.Join(example, "google-discovery", "b.json"),
	}
	overlay := catalog.SecurityOverlay{
		ID:              "demo-url-overlay",
		ProviderID:      "demo",
		Status:          catalog.AuthStatusOverlayRequired,
		SecuritySchemes: []catalog.SecurityScheme{{Name: "DemoAuth", Type: "apiKey"}},
		RootSecurity:    []catalog.SecurityRequirement{{Scheme: "DemoAuth"}},
		SourceRefs:      []string{"https://example.test/b.json"},
		SourceNote:      "test",
	}

	sidecars, notes, err := materializeSecurityOverlaySidecars([]catalogSecurityOverlayCandidate{{
		ProviderID:   "demo",
		ProviderName: "Demo",
		OverlayID:    overlay.ID,
		SourceRefs:   overlay.SourceRefs,
		Overlay:      &overlay,
	}}, []CatalogMigrationCandidate{sourceA, sourceB}, example)
	if err != nil {
		t.Fatalf("materializeSecurityOverlaySidecars failed: %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("notes = %#v, want none", notes)
	}
	if len(sidecars) != 1 || sidecars[0].RelativePath != "google-discovery/b.security-overlay.json" {
		t.Fatalf("sidecars = %#v, want b sidecar", sidecars)
	}
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "b.security-overlay.json")); err != nil {
		t.Fatalf("source URL matched sidecar missing: %v", err)
	}
}

func TestMaterializeSecurityOverlaySidecarsDoesNotGuessAcrossMultipleSources(t *testing.T) {
	example := t.TempDir()
	sources := []CatalogMigrationCandidate{
		{
			ProviderID:   "demo",
			ProviderName: "Demo",
			SpecRefID:    "a",
			Kind:         catalog.SpecKindOpenAPI,
			Protocol:     string(catalog.SpecProtocolOpenAPI),
			RelativePath: "openapi/a.json",
			TargetPath:   filepath.Join(example, "openapi", "a.json"),
		},
		{
			ProviderID:   "demo",
			ProviderName: "Demo",
			SpecRefID:    "b",
			Kind:         catalog.SpecKindGoogleDiscovery,
			Protocol:     string(catalog.SpecProtocolGoogleDiscovery),
			RelativePath: "google-discovery/b.json",
			TargetPath:   filepath.Join(example, "google-discovery", "b.json"),
		},
	}
	overlay := catalog.SecurityOverlay{
		ID:              "demo-provider-overlay",
		ProviderID:      "demo",
		Status:          catalog.AuthStatusOverlayRequired,
		SecuritySchemes: []catalog.SecurityScheme{{Name: "DemoAuth", Type: "apiKey"}},
		RootSecurity:    []catalog.SecurityRequirement{{Scheme: "DemoAuth"}},
		SourceNote:      "test",
	}

	sidecars, notes, err := materializeSecurityOverlaySidecars([]catalogSecurityOverlayCandidate{{
		ProviderID:   "demo",
		ProviderName: "Demo",
		OverlayID:    overlay.ID,
		Overlay:      &overlay,
	}}, sources, example)
	if err != nil {
		t.Fatalf("materializeSecurityOverlaySidecars failed: %v", err)
	}
	if len(sidecars) != 0 {
		t.Fatalf("sidecars = %#v, want none", sidecars)
	}
	if len(notes) != 1 || !strings.Contains(notes[0], "no exact source match among multiple migrated demo sources") {
		t.Fatalf("notes = %#v, want no-guess note", notes)
	}
	if _, err := os.Stat(filepath.Join(example, "openapi", "a.security-overlay.json")); !os.IsNotExist(err) {
		t.Fatalf("unexpected guessed sidecar: %v", err)
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

func TestApplyCatalogPlanResponseMigratesValidatedSelectionAndUsesProposedSteps(t *testing.T) {
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
			Name:     "call_test",
			Type:     "http",
			Provider: "test",
			OpenAPI:  "openapi/test-api.json",
			Do:       "Perform the catalog-planned test capability.",
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
	if step.Name != "call_test" || step.Do != "Perform the catalog-planned test capability." {
		t.Fatalf("catalog proposed step was not preserved: %#v", step)
	}
	if step.Operation != "" || len(step.With) != 0 || step.Provider != "test" || step.OpenAPI != "openapi/test-api.json" {
		t.Fatalf("unsafe or missing provider placeholder fields: %#v", step)
	}
	if !hasAssumption(session.Assumptions, "catalog_plan_api_docs_migrated") {
		t.Fatalf("missing catalog plan assumption: %#v", session.Assumptions)
	}
	if !strings.Contains(out.String(), "selected Test API API document from catalog plan") {
		t.Fatalf("missing selected artifact output:\n%s", out.String())
	}
}

func TestApplyCatalogPlanResponsePreservesProposedOrderAndDependsOn(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "openapi/gmail.json")
	writeCatalogArtifact(t, cacheRoot, "openapi/openweathermap.json")
	hints := []CatalogHint{
		{
			Provider: catalog.Provider{ID: "gmail", DisplayName: "Gmail"},
			SpecArtifacts: []CatalogSpecArtifactHint{{
				SpecRef: catalog.SpecReference{ID: "gmail", Kind: catalog.SpecKindOpenAPI},
				Path:    filepath.Join(cacheRoot, "openapi", "gmail.json"),
			}},
		},
		{
			Provider: catalog.Provider{ID: "openweathermap", DisplayName: "OpenWeatherMap"},
			SpecArtifacts: []CatalogSpecArtifactHint{{
				SpecRef: catalog.SpecReference{ID: "openweathermap", Kind: catalog.SpecKindOpenAPI},
				Path:    filepath.Join(cacheRoot, "openapi", "openweathermap.json"),
			}},
		},
	}
	request := BuildCatalogPlanRequest("get weather and gmail the report", Session{}, hints, example)
	keys := map[string]string{}
	for _, candidate := range request.Candidates {
		keys[candidate.ProviderID] = candidate.ArtifactKey
	}
	session := Session{}
	var out strings.Builder

	_, err := applyCatalogPlanResponse(&out, &session, hints, example, CatalogPlanResponse{
		SelectedArtifacts: []CatalogPlanArtifactSelection{
			{ProviderID: "gmail", ArtifactKey: keys["gmail"]},
			{ProviderID: "openweathermap", ArtifactKey: keys["openweathermap"]},
		},
		ProposedSteps: []CatalogPlanStep{
			{Name: "openweathermap", Type: "http", Provider: "openweathermap", OpenAPI: "openapi/openweathermap.json", Do: "Fetch weather."},
			{Name: "gmail", Type: "http", Provider: "gmail", OpenAPI: "openapi/gmail.json", Do: "Send the weather report.", DependsOn: flexibleStringList{"openweathermap"}},
		},
	})
	if err != nil {
		t.Fatalf("applyCatalogPlanResponse failed: %v", err)
	}
	if got := stepNames(session.Intent.Steps); strings.Join(got, ",") != "openweathermap,gmail" {
		t.Fatalf("step order = %v, want openweathermap,gmail", got)
	}
	gmail := stepByName(session.Intent.Steps, "gmail")
	if gmail == nil || len(gmail.DependsOn) != 1 || gmail.DependsOn[0] != "openweathermap" {
		t.Fatalf("gmail depends_on = %#v", gmail)
	}
}

func TestApplyCatalogPlanResponseRejectsUnselectedProposedProvider(t *testing.T) {
	cacheRoot := t.TempDir()
	example := t.TempDir()
	writeCatalogArtifact(t, cacheRoot, "openapi/gmail.json")
	hints := []CatalogHint{{
		Provider: catalog.Provider{ID: "gmail", DisplayName: "Gmail"},
		SpecArtifacts: []CatalogSpecArtifactHint{{
			SpecRef: catalog.SpecReference{ID: "gmail", Kind: catalog.SpecKindOpenAPI},
			Path:    filepath.Join(cacheRoot, "openapi", "gmail.json"),
		}},
	}}
	request := BuildCatalogPlanRequest("gmail the report", Session{}, hints, example)
	session := Session{}
	var out strings.Builder

	_, err := applyCatalogPlanResponse(&out, &session, hints, example, CatalogPlanResponse{
		SelectedArtifacts: []CatalogPlanArtifactSelection{{ProviderID: "gmail", ArtifactKey: request.Candidates[0].ArtifactKey}},
		ProposedSteps: []CatalogPlanStep{
			{Name: "openweathermap", Type: "http", Provider: "openweathermap", OpenAPI: "openapi/openweathermap.json"},
			{Name: "gmail", Type: "http", Provider: "gmail", OpenAPI: "openapi/gmail.json"},
		},
	})
	if err != nil {
		t.Fatalf("applyCatalogPlanResponse failed: %v", err)
	}
	if got := stepNames(session.Intent.Steps); strings.Join(got, ",") != "gmail" {
		t.Fatalf("accepted proposed steps = %v, want gmail only", got)
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

func TestCatalogPlanResponseToleratesNumericDependsOn(t *testing.T) {
	var response CatalogPlanResponse
	if err := json.Unmarshal([]byte(`{
	  "selected_artifacts": [{"provider_id":"gmail","artifact_key":"gmail:discovery/gmail.json"}],
	  "proposed_steps": [{"name":"send","provider":"gmail","depends_on":[1,"prepare"]}]
	}`), &response); err != nil {
		t.Fatalf("unmarshal catalog plan response: %v", err)
	}
	if len(response.ProposedSteps) != 1 {
		t.Fatalf("proposed steps = %#v", response.ProposedSteps)
	}
	if got := []string(response.ProposedSteps[0].DependsOn); !slices.Contains(got, "1") || !slices.Contains(got, "prepare") {
		t.Fatalf("depends_on = %#v", got)
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
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "gmail-discovery-v1.json")); !os.IsNotExist(err) {
		t.Fatalf("catalog plan error should not copy before fallback: %v", err)
	}

	if _, err := MigrateCatalogArtifacts(catalogQueryForSession(session), example, CatalogHintOptions{CacheRoot: cacheRoot}); err != nil {
		t.Fatalf("deterministic fallback migration failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "google-discovery", "gmail-discovery-v1.json")); err != nil {
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
	mustWriteCatalogTestFile(t, path, []byte("{}\n"))
}

func mustWriteCatalogTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
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
