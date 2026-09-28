package stepauthoring

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

const sourceAddOpenAPIFixture = `openapi: 3.0.3
info: {title: Weather, version: '1'}
paths:
  /weather:
    get:
      operationId: getWeather
      summary: Get weather
      responses:
        '200': {description: Current weather}
`

func TestAddSourcesWritesValidatedSourceAndProvenanceAtomically(t *testing.T) {
	root := t.TempDir()
	input := writeSourceAddInput(t, "weather.yaml", sourceAddOpenAPIFixture)
	request := sourceAddRequest(input, []byte(sourceAddOpenAPIFixture), SourceManifestRevision{State: "missing"})
	outcome := AddSources(context.Background(), root, request)
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil {
		t.Fatalf("AddSources() = %#v", outcome)
	}
	if len(outcome.Result.Result.Sources) != 1 {
		t.Fatalf("sources = %#v", outcome.Result.Result.Sources)
	}
	entry := outcome.Result.Result.Sources[0]
	if entry.PackagePath != "openapi/weather.yaml" || entry.SHA256 != request.Sources[0].SourceSHA256 || entry.CandidateSourceID != sourceIDForPath(entry.PackagePath) {
		t.Fatalf("source result = %#v", entry)
	}
	got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.PackagePath)))
	if err != nil || string(got) != sourceAddOpenAPIFixture {
		t.Fatalf("copied source = %q, err = %v", got, err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(APISourceManifestPath)))
	if err != nil || "sha256:"+evidencefile.SHA256(manifestBytes) != outcome.Result.Result.ManifestSHA256 {
		t.Fatalf("manifest digest = %q, err = %v", outcome.Result.Result.ManifestSHA256, err)
	}
	var manifest APISourceManifest
	if err := evidencefile.DecodeStrict(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != apiSourceManifestV1 || len(manifest.Sources) != 1 || manifest.Sources[0].PackagePath != entry.PackagePath {
		t.Fatalf("manifest = %#v", manifest)
	}
	encoded, _ := json.Marshal(outcome.Result)
	if strings.Contains(string(encoded), input) {
		t.Fatalf("result disclosed local input path: %s", encoded)
	}
	paths, err := packageartifacts.RequiredPackagePaths(root)
	if err != nil {
		t.Fatal(err)
	}
	if !containsPath(paths, APISourceManifestPath) || !containsPath(paths, entry.PackagePath) {
		t.Fatalf("required package paths = %#v", paths)
	}
}

func TestAddSourcesAppendsWithExactManifestRevision(t *testing.T) {
	root := t.TempDir()
	firstPath := writeSourceAddInput(t, "weather.yaml", sourceAddOpenAPIFixture)
	first := AddSources(nil, root, sourceAddRequest(firstPath, []byte(sourceAddOpenAPIFixture), SourceManifestRevision{State: "missing"}))
	if first.ExitCode != 0 || first.Result.Result == nil {
		t.Fatalf("first AddSources() = %#v", first)
	}

	secondContent := `openapi: 3.0.3
info: {title: Forecast, version: '1'}
paths: {}
`
	secondPath := writeSourceAddInput(t, "forecast.yaml", secondContent)
	revision := SourceManifestRevision{State: "present", SHA256: first.Result.Result.ManifestSHA256}
	second := AddSources(nil, root, sourceAddRequestWithID(secondPath, []byte(secondContent), revision, "forecast"))
	if second.ExitCode != 0 || second.Result.Result == nil {
		t.Fatalf("second AddSources() = %#v", second)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(APISourceManifestPath)))
	if err != nil {
		t.Fatal(err)
	}
	var manifest APISourceManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if len(manifest.Sources) != 2 || manifest.Sources[0].PackagePath != "openapi/forecast.yaml" || manifest.Sources[1].PackagePath != "openapi/weather.yaml" {
		t.Fatalf("manifest entries are not sorted or complete: %#v", manifest.Sources)
	}
}

func TestAddSourcesRejectsStaleRevisionsAndSourceDigestsWithoutWriting(t *testing.T) {
	root := t.TempDir()
	input := writeSourceAddInput(t, "weather.yaml", sourceAddOpenAPIFixture)
	request := sourceAddRequest(input, []byte(sourceAddOpenAPIFixture), SourceManifestRevision{State: "missing"})
	request.Sources[0].SourceSHA256 = "sha256:" + strings.Repeat("0", 64)
	staleDigest := AddSources(nil, root, request)
	if staleDigest.Result.Status != "conflict" || staleDigest.ExitCode != 3 {
		t.Fatalf("stale source digest outcome = %#v", staleDigest)
	}
	if _, err := os.Lstat(filepath.Join(root, "openapi")); !os.IsNotExist(err) {
		t.Fatalf("digest failure wrote source directory: %v", err)
	}

	staleManifest := sourceAddRequest(input, []byte(sourceAddOpenAPIFixture), SourceManifestRevision{State: "present", SHA256: "sha256:" + strings.Repeat("0", 64)})
	outcome := AddSources(nil, root, staleManifest)
	if outcome.Result.Status != "conflict" || outcome.ExitCode != 3 {
		t.Fatalf("stale manifest outcome = %#v", outcome)
	}
	if _, err := os.Lstat(filepath.Join(root, "openapi")); !os.IsNotExist(err) {
		t.Fatalf("manifest conflict wrote source directory: %v", err)
	}
}

func TestAddSourcesRejectsUnparseableAndSymlinkedSourceFiles(t *testing.T) {
	root := t.TempDir()
	badContent := "openapi: [broken"
	badPath := writeSourceAddInput(t, "invalid.yaml", badContent)
	invalid := AddSources(nil, root, sourceAddRequest(badPath, []byte(badContent), SourceManifestRevision{State: "missing"}))
	if invalid.Result.Status != "blocked" || invalid.ExitCode != 4 {
		t.Fatalf("invalid source outcome = %#v", invalid)
	}
	if _, err := os.Lstat(filepath.Join(root, "openapi")); !os.IsNotExist(err) {
		t.Fatalf("invalid source wrote into package: %v", err)
	}

	actual := writeSourceAddInput(t, "actual.yaml", sourceAddOpenAPIFixture)
	link := filepath.Join(t.TempDir(), "weather.yaml")
	if err := os.Symlink(actual, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	linked := AddSources(nil, root, sourceAddRequest(link, []byte(sourceAddOpenAPIFixture), SourceManifestRevision{State: "missing"}))
	if linked.Result.Status != "blocked" || linked.ExitCode != 4 {
		t.Fatalf("symlink source outcome = %#v", linked)
	}
	if _, err := os.Lstat(filepath.Join(root, "openapi")); !os.IsNotExist(err) {
		t.Fatalf("symlink rejection wrote into package: %v", err)
	}
}

func TestAddSourcesValidatesWholeBatchBeforeWritingAnyFile(t *testing.T) {
	root := t.TempDir()
	goodPath := writeSourceAddInput(t, "good.yaml", sourceAddOpenAPIFixture)
	badContent := "openapi: [broken"
	badPath := writeSourceAddInput(t, "bad.yaml", badContent)
	request := SourceAddRequest{
		Version: WireVersion, Kind: "request", Command: SourceAddCommand,
		ManifestRevision: SourceManifestRevision{State: "missing"},
		Sources: []SourceAddEntry{
			{SourceKind: "openapi", SourceID: "good", SourcePath: goodPath, SourceSHA256: "sha256:" + evidencefile.SHA256([]byte(sourceAddOpenAPIFixture))},
			{SourceKind: "openapi", SourceID: "bad", SourcePath: badPath, SourceSHA256: "sha256:" + evidencefile.SHA256([]byte(badContent))},
		},
	}
	outcome := AddSources(nil, root, request)
	if outcome.Result.Status != "blocked" || outcome.ExitCode != 4 {
		t.Fatalf("batch validation outcome = %#v", outcome)
	}
	for _, path := range []string{"openapi/good.yaml", "openapi/bad.yaml", APISourceManifestPath} {
		if _, err := os.Lstat(filepath.Join(root, filepath.FromSlash(path))); !os.IsNotExist(err) {
			t.Errorf("failed batch wrote %s: %v", path, err)
		}
	}
}

func TestAddSourcesRejectsUnsupportedSourcePathAndKindPairings(t *testing.T) {
	input := filepath.Join(string(os.PathSeparator), "tmp", "source.txt")
	for _, test := range []SourceAddEntry{
		{SourceKind: "openapi", SourceID: "weather", SourcePath: input, SourceSHA256: "sha256:" + strings.Repeat("a", 64)},
		{SourceKind: "unknown", SourceID: "weather", SourcePath: input, SourceSHA256: "sha256:" + strings.Repeat("a", 64)},
		{SourceKind: "openapi", SourceID: "../weather", SourcePath: input, SourceSHA256: "sha256:" + strings.Repeat("a", 64)},
	} {
		request := SourceAddRequest{
			Version: WireVersion, Kind: "request", Command: SourceAddCommand,
			ManifestRevision: SourceManifestRevision{State: "missing"}, Sources: []SourceAddEntry{test},
		}
		if err := validateSourceAddRequest(request); err != nil {
			continue
		}
		if _, _, ok := sourcePackagePath(test.SourceKind, test.SourceID, test.SourcePath); ok {
			t.Fatalf("unsupported entry accepted: %#v", test)
		}
	}
}

func sourceAddRequest(path string, content []byte, revision SourceManifestRevision) SourceAddRequest {
	return sourceAddRequestWithID(path, content, revision, "weather")
}

func sourceAddRequestWithID(path string, content []byte, revision SourceManifestRevision, id string) SourceAddRequest {
	return SourceAddRequest{
		Version: WireVersion, Kind: "request", Command: SourceAddCommand,
		ManifestRevision: revision,
		Sources: []SourceAddEntry{{
			SourceKind: "openapi", SourceID: id, SourcePath: path,
			SourceSHA256: "sha256:" + evidencefile.SHA256(content),
		}},
	}
}

func writeSourceAddInput(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func containsPath(paths []string, wanted string) bool {
	for _, path := range paths {
		if path == wanted {
			return true
		}
	}
	return false
}
