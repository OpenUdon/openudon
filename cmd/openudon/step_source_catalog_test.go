package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/openudon/internal/stepauthoring"
	"github.com/OpenUdon/uws/uws1"
)

func catalogSourceSelection(t *testing.T, installation stepauthoring.CatalogInstallation) stepauthoring.CatalogSourceRequest {
	t.Helper()
	options, err := installation.IndexOptions()
	if err != nil {
		t.Fatal(err)
	}
	report, err := apitools.DiscoverCatalogOperations(context.Background(), apitools.CatalogDiscoveryOptions{Index: options, Request: discoveryRequest()})
	if err != nil || report.Outcome != apitools.CatalogDiscoveryMatch {
		t.Fatal("discovery did not produce an exact selectable operation", err, report.Outcome)
	}
	for _, candidate := range report.Candidates {
		if candidate.Qualified {
			return stepauthoring.CatalogSourceRequest{Version: stepauthoring.CatalogSourceWireVersion, Kind: "request", Command: stepauthoring.SourceAddCommand, Confirmed: true, ManifestRevision: stepauthoring.SourceManifestRevision{State: "missing"}, Sources: []stepauthoring.CatalogSourceEntry{{SourceID: "notes", References: candidate.References}}}
		}
	}
	t.Fatal("no qualified native reference")
	return stepauthoring.CatalogSourceRequest{}
}

func packageSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if entry.IsDir() {
			result[rel] = "directory"
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash := sha256.Sum256(data)
		result[rel] = hex.EncodeToString(hash[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func runCatalogSourceFixture(t *testing.T, installation stepauthoring.CatalogInstallation, root string, request stepauthoring.CatalogSourceRequest) (int, stepauthoring.SourceAddWireResult) {
	t.Helper()
	data, _ := json.Marshal(request)
	args := []string{"add", "--catalog", "--catalog-root", installation.Root.Directory, "--catalog-metadata", installation.MetadataPath, "--example", root, "--request", "-"}
	var stdout, stderr bytes.Buffer
	code := runStepSourceCommand(args, bytes.NewReader(data), &stdout, &stderr)
	if stderr.Len() != 0 || strings.Contains(stdout.String(), root) || strings.Contains(stdout.String(), installation.Root.Directory) {
		t.Fatal("source command disclosed private installation/package paths", stderr.String())
	}
	var result stepauthoring.SourceAddWireResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err, stdout.String())
	}
	return code, result
}

func TestCatalogSourceRoundTripPreservesRawSelectorsProviderLinksAndScope(t *testing.T) {
	installation := preparedDiscoveryCatalog(t, false)
	options, _ := installation.IndexOptions()
	cat := *options.Catalog
	other := cat.Providers[0].SpecReferences[0]
	other.ID = "unselected-notes"
	cat.Providers[0].SpecReferences = append(cat.Providers[0].SpecReferences, other)
	for _, row := range []struct{ id, provider, spec string }{
		{"provider-auth", cat.Providers[0].ID, ""},
		{"selected-auth", cat.Providers[0].ID, cat.Providers[0].SpecReferences[0].ID},
		{"unselected-auth", cat.Providers[0].ID, other.ID},
	} {
		cat.SecurityOverlays = append(cat.SecurityOverlays, catalog.SecurityOverlay{ID: row.id, ProviderID: row.provider, SpecRefID: row.spec, Status: catalog.AuthStatusUnknown, SourceNote: "Synthetic advisory metadata.", SourceRefs: []string{"https://example.invalid/auth"}})
	}
	data, _ := json.Marshal(cat)
	if err := os.WriteFile(installation.MetadataPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	options, _ = installation.IndexOptions()
	index, err := apitools.BuildCatalogOperationIndex(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if err := apitools.WriteCatalogOperationIndex(context.Background(), options, index); err != nil {
		t.Fatal(err)
	}
	request := catalogSourceSelection(t, installation)
	root := t.TempDir()
	before := packageSnapshot(t, installation.Root.Directory)
	code, result := runCatalogSourceFixture(t, installation, root, request)
	if code != 0 || result.Status != "completed" || result.Version != stepauthoring.CatalogSourceWireVersion || result.Result == nil {
		t.Fatalf("source provisioning failed: %d %+v", code, result)
	}
	if !reflect.DeepEqual(before, packageSnapshot(t, installation.Root.Directory)) {
		t.Fatal("source root/index/registry changed")
	}
	original, _ := os.ReadFile(filepath.Join(installation.Root.Directory, "openapi/notes.json"))
	saved, _ := os.ReadFile(filepath.Join(root, result.Result.Sources[0].PackagePath))
	if !bytes.Equal(original, saved) {
		t.Fatal("native raw bytes changed")
	}
	data, err = os.ReadFile(filepath.Join(root, result.Result.ProvenancePath))
	if err != nil {
		t.Fatal(err)
	}
	var provenance stepauthoring.CatalogSourceProvenance
	if err := json.Unmarshal(data, &provenance); err != nil {
		t.Fatal(err)
	}
	if len(provenance.Artifacts) != len(request.Sources[0].References) || len(provenance.SecurityOverlays) != 2 {
		t.Fatal("provider links lost or unselected overlays copied", provenance)
	}
	for _, artifact := range provenance.Artifacts {
		if artifact.Reference.Selector != request.Sources[0].References[0].Selector || artifact.PackagePath != result.Result.Sources[0].PackagePath {
			t.Fatal("selector or package identity changed")
		}
	}
	for _, overlay := range provenance.SecurityOverlays {
		data, err := os.ReadFile(filepath.Join(root, overlay.Path))
		hash := sha256.Sum256(data)
		if err != nil || hex.EncodeToString(hash[:]) != overlay.SHA256 || overlay.OverlayID == "unselected-auth" {
			t.Fatal("security provenance not digest-bound and scoped")
		}
	}
	// Source candidates still resolve through the unchanged legacy package path.
	candidates := stepauthoring.Candidates(context.Background(), root, stepauthoring.CandidatesRequest{Version: stepauthoring.WireVersion, Kind: "request", Command: stepauthoring.CandidatesCommand, Contract: stepauthoring.StepContract{ID: "list_notes", Purpose: "list notes records", Effect: "read", Inputs: &uws1.ParamSchema{Type: "object"}, Outputs: &uws1.ParamSchema{Type: "object"}}})
	if candidates.ExitCode != 0 || candidates.Result.Result == nil || len(candidates.Result.Result.Candidates) == 0 {
		t.Fatal("provisioned raw source is not usable by legacy candidate commands", candidates)
	}
}

func TestCatalogSourceRefusalDriftSelectorCollisionAndCancellationNeverPublish(t *testing.T) {
	for _, mode := range []string{"unconfirmed", "catalog-drift", "raw-drift", "wrong-selector", "manifest-conflict", "collision", "provenance-collision", "source-overlap", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			installation := preparedDiscoveryCatalog(t, false)
			request := catalogSourceSelection(t, installation)
			root := t.TempDir()
			ctx := context.Background()
			switch mode {
			case "unconfirmed":
				request.Confirmed = false
			case "catalog-drift":
				for i := range request.Sources[0].References {
					request.Sources[0].References[i].CatalogSHA256 = strings.Repeat("0", 64)
				}
			case "raw-drift":
				if err := os.WriteFile(filepath.Join(installation.Root.Directory, "openapi/notes.json"), []byte("{}"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "wrong-selector":
				for i := range request.Sources[0].References {
					request.Sources[0].References[i].Selector = "#/paths/~1does-not-exist/get"
				}
			case "manifest-conflict":
				request.ManifestRevision = stepauthoring.SourceManifestRevision{State: "present", SHA256: "sha256:" + strings.Repeat("0", 64)}
			case "collision":
				if err := os.Mkdir(filepath.Join(root, "openapi"), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, "openapi/notes.json"), []byte("existing"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "source-overlap":
				root = installation.Root.Directory
			case "provenance-collision":
				data, _ := json.Marshal(request)
				hash := sha256.Sum256(data)
				path := filepath.Join(root, "expected/catalog-sources", hex.EncodeToString(hash[:]), "provenance.json")
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			before := packageSnapshot(t, root)
			outcome := stepauthoring.AddCatalogSources(ctx, root, installation, request)
			if outcome.ExitCode == 0 || outcome.Result.Result != nil {
				t.Fatal("unsafe source selection published", outcome)
			}
			if mode == "wrong-selector" && outcome.Result.Diagnostics[0].Code != "catalog.selector_mismatch" {
				t.Fatal("did not independently check the selected native operation", outcome)
			}
			if !reflect.DeepEqual(before, packageSnapshot(t, root)) {
				t.Fatal("refusal left package changes")
			}
			left, err := filepath.Glob(filepath.Join(filepath.Dir(root), ".openudon-catalog-stage-*"))
			if err != nil || len(left) != 0 {
				t.Fatal("private staging was not cleaned up")
			}
		})
	}
}

func TestCatalogSourcePublishedSchemaRejectsAliasesNullsAndUnknownAuthority(t *testing.T) {
	installation := preparedDiscoveryCatalog(t, false)
	request := catalogSourceSelection(t, installation)
	data, _ := json.Marshal(request)
	if _, err := stepauthoring.DecodeCatalogSourceRequest(data); err != nil {
		t.Fatal("real native selection does not conform", err)
	}
	for _, input := range []string{
		strings.Replace(string(data), `"confirmed":true`, `"Confirmed":true`, 1),
		strings.Replace(string(data), `"confirmed":true`, `"confirmed":null`, 1),
		strings.Replace(string(data), `"state":"missing"`, `"state":"missing","sha256":null`, 1),
		strings.Replace(string(data), `"kind":"request"`, `"kind":"request","root":"PRIVATE_CANARY"`, 1),
		strings.Replace(string(data), `"confirmed":true`, `"confirmed":false,"confirmed":true`, 1),
	} {
		root := t.TempDir()
		var stdout, stderr bytes.Buffer
		code := runStepSourceCommand([]string{"add", "--catalog", "--catalog-root", installation.Root.Directory, "--catalog-metadata", installation.MetadataPath, "--example", root, "--request", "-"}, strings.NewReader(input), &stdout, &stderr)
		if code != 2 || len(packageSnapshot(t, root)) != 1 || strings.Contains(stdout.String()+stderr.String(), "PRIVATE_CANARY") {
			t.Fatal("malformed authority accepted, package changed or raw input disclosed", code)
		}
		var result stepauthoring.SourceAddWireResult
		if json.Unmarshal(stdout.Bytes(), &result) != nil || result.Version != stepauthoring.CatalogSourceWireVersion {
			t.Fatal("catalog refusal lost its additive wire identity")
		}
	}
}

func TestCatalogSourceRefusesCredentialLikeAdvisoryWithoutPublishing(t *testing.T) {
	installation := preparedDiscoveryCatalog(t, false)
	options, _ := installation.IndexOptions()
	cat := *options.Catalog
	// Synthetic non-secret canary intentionally matching the existing scanner.
	canary := "sk-" + strings.Repeat("synthetic", 4)
	cat.SecurityOverlays = append(cat.SecurityOverlays, catalog.SecurityOverlay{ID: "private-advisory", ProviderID: cat.Providers[0].ID, Status: catalog.AuthStatusUnknown, SourceNote: canary, SourceRefs: []string{"https://example.invalid/auth"}})
	data, _ := json.Marshal(cat)
	if err := os.WriteFile(installation.MetadataPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	options, _ = installation.IndexOptions()
	index, err := apitools.BuildCatalogOperationIndex(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	if err := apitools.WriteCatalogOperationIndex(context.Background(), options, index); err != nil {
		t.Fatal(err)
	}
	request := catalogSourceSelection(t, installation)
	root := t.TempDir()
	before := packageSnapshot(t, root)
	code, result := runCatalogSourceFixture(t, installation, root, request)
	encoded, _ := json.Marshal(result)
	if code != 4 || result.Diagnostics[0].Code != "catalog.overlay_mismatch" || strings.Contains(string(encoded), canary) || !reflect.DeepEqual(before, packageSnapshot(t, root)) {
		t.Fatal("credential-like advisory was published or disclosed", code)
	}
}
