package elicitor

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/OpenUdon/apitools/catalog"
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
