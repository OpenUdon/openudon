package stepauthoring

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/openudon/docs/schemas"
	"github.com/OpenUdon/openudon/internal/artifactwriter"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const CatalogSourceWireVersion = "openudon.step-source-catalog.v1"

type CatalogSourceEntry struct {
	SourceID   string                              `json:"source_id"`
	References []apitools.CatalogArtifactReference `json:"references"`
}

// Confirmed represents the caller's exact source-proposal decision; discovery
// and source scores alone cannot supply it. References include the full catalog
// identity, binding the selected advisory overlays as well as the raw source.
type CatalogSourceRequest struct {
	Version          string                 `json:"version"`
	Kind             string                 `json:"kind"`
	Command          string                 `json:"command"`
	Confirmed        bool                   `json:"confirmed"`
	ManifestRevision SourceManifestRevision `json:"manifest_revision"`
	Sources          []CatalogSourceEntry   `json:"sources"`
}

var catalogSourceSchema = sync.OnceValues(func() (*jsonschema.Schema, error) {
	data, err := schemas.CatalogSourceResources.ReadFile(CatalogSourceWireVersion + ".schema.json")
	if err != nil {
		return nil, err
	}
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	compiler := jsonschema.NewCompiler()
	compiler.UseLoader(jsonschema.SchemeURLLoader{})
	url := "https://openudon.dev/schemas/" + CatalogSourceWireVersion + ".schema.json"
	if err := compiler.AddResource(url, document); err != nil {
		return nil, err
	}
	return compiler.Compile(url)
})

// DecodeCatalogSourceRequest validates the exact published schema, including
// casing/null/presence, rather than permitting encoding/json field aliases.
func DecodeCatalogSourceRequest(data []byte) (CatalogSourceRequest, error) {
	var request CatalogSourceRequest
	if len(data) > MaxRequestBytes || !ValidUTF8Request(data) || evidencefile.DecodeStrict(data, &request) != nil {
		return request, errors.New("invalid catalog source request")
	}
	schema, err := catalogSourceSchema()
	if err != nil {
		return request, errors.New("catalog source schema unavailable")
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil || schema.Validate(instance) != nil {
		return request, errors.New("catalog source request does not match its published schema")
	}
	return request, nil
}

type CatalogSourceArtifact struct {
	Reference      apitools.CatalogArtifactReference `json:"reference"`
	RegisteredKind string                            `json:"registered_kind"`
	Advisory       bool                              `json:"advisory,omitempty"`
	SourceURL      string                            `json:"source_url,omitempty"`
	SourceID       string                            `json:"source_id"`
	PackagePath    string                            `json:"package_path"`
}

type CatalogSourceProvenance struct {
	Version          string                                  `json:"version"`
	RequestSHA256    string                                  `json:"request_sha256"`
	CatalogSHA256    string                                  `json:"catalog_sha256"`
	Artifacts        []CatalogSourceArtifact                 `json:"artifacts"`
	SecurityOverlays []apitools.CatalogArtifactExportOverlay `json:"security_overlays,omitempty"`
}

// AddCatalogSources materializes only selected native references privately,
// then validates selector identity and commits all package files atomically.
// Native export does not validate selectors, so this owner must check them.
func AddCatalogSources(ctx context.Context, example string, installation CatalogInstallation, request CatalogSourceRequest) SourceAddOutcome {
	if ctx == nil {
		ctx = context.Background()
	}
	fail := func(status, code, message string, exit int) SourceAddOutcome {
		result := sourceAddFailure(status, code, message, exit)
		result.Result.Version = CatalogSourceWireVersion
		return result
	}
	if request.Version != CatalogSourceWireVersion || request.Kind != "request" || request.Command != SourceAddCommand || !request.Confirmed || len(request.Sources) == 0 || len(request.Sources) > maxSourceAddEntries {
		return fail("failed", "request.invalid", "The catalog-source request requires an exact confirmed selection.", 2)
	}
	seen := map[string]bool{}
	refs := []apitools.CatalogArtifactReference{}
	var totalBytes int64
	for _, source := range request.Sources {
		if !sourceIDPattern.MatchString(source.SourceID) || seen[source.SourceID] || len(source.References) == 0 || len(source.References) > 64 {
			return fail("failed", "request.invalid", "Catalog source IDs and reference counts are invalid.", 2)
		}
		seen[source.SourceID] = true
		first := source.References[0]
		if !sourceKindSupported(string(first.Kind)) || first.Bytes <= 0 || first.Bytes > MaxSourceBytes || first.Selector == "" {
			return fail("failed", "request.invalid", "A selected catalog source lacks a supported identity or native selector.", 2)
		}
		totalBytes += first.Bytes
		for _, ref := range source.References {
			if ref.Kind != first.Kind || ref.SHA256 != first.SHA256 || ref.Bytes != first.Bytes || ref.Selector != first.Selector || ref.CatalogSHA256 != first.CatalogSHA256 {
				return fail("failed", "request.invalid", "Provider references for one source must bind the same exact native operation.", 2)
			}
			refs = append(refs, ref)
		}
	}
	if len(refs) > 64 || totalBytes > MaxCandidateSourceBytesTotal {
		return fail("blocked", "source.byte_limit", "The selected catalog sources exceed their supported budget.", 4)
	}
	if err := ctx.Err(); err != nil {
		return fail("blocked", "request.cancelled", "Catalog provisioning was cancelled before validation.", 4)
	}
	root, err := resolveExampleRoot(example)
	if err != nil {
		return fail("blocked", "example.invalid", "The package is unavailable or unsafe.", 4)
	}
	index, err := installation.IndexOptions()
	if err != nil {
		return fail("blocked", "catalog.invalid", "Catalog installation metadata is unavailable or invalid.", 4)
	}
	paths, err := catalog.ResolveRoot(index.Root)
	if err != nil || paths.Directory == "" {
		return fail("blocked", "catalog.invalid", "An explicit prepared catalog root is required.", 4)
	}
	if root == paths.Directory || strings.HasPrefix(root, paths.Directory+string(filepath.Separator)) || strings.HasPrefix(paths.Directory, root+string(filepath.Separator)) {
		return fail("blocked", "catalog.source_overlap", "The package and catalog source root must be disjoint.", 4)
	}
	// Native export owns its disposable transaction. The package itself is not
	// changed until all native export, selector and source checks pass.
	stage, err := os.MkdirTemp(filepath.Dir(root), ".openudon-catalog-stage-")
	if err != nil {
		return fail("blocked", "catalog.staging_unavailable", "Private catalog staging is unavailable.", 4)
	}
	defer os.RemoveAll(stage)
	export, err := apitools.ExportCatalogArtifacts(ctx, apitools.CatalogArtifactExportOptions{Index: index, References: refs, WorkflowDir: stage})
	if err != nil {
		return fail("blocked", "catalog.export_refused", "The selected catalog artifact identities could not be verified.", 4)
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return fail("failed", "request.invalid", "The selected catalog request cannot be encoded.", 2)
	}
	requestDigest := evidencefile.SHA256(requestBytes)
	provenanceDirectory := "expected/catalog-sources/" + requestDigest
	provenancePath := provenanceDirectory + "/provenance.json"
	provenance := CatalogSourceProvenance{Version: "openudon.catalog-source-provenance.v1", RequestSHA256: requestDigest, CatalogSHA256: export.CatalogSHA256, Artifacts: []CatalogSourceArtifact{}}
	local := SourceAddRequest{Version: WireVersion, Kind: "request", Command: SourceAddCommand, ManifestRevision: request.ManifestRevision}
	exportRoot := filepath.Join(stage, "api-artifacts")
	for _, source := range request.Sources {
		first := source.References[0]
		var selected *apitools.CatalogArtifactExportEntry
		for i := range export.Artifacts {
			if export.Artifacts[i].Reference == first {
				selected = &export.Artifacts[i]
				break
			}
		}
		if selected == nil {
			return fail("blocked", "catalog.reference_mismatch", "Native export omitted a selected reference.", 4)
		}
		content, err := readWithin(exportRoot, selected.Path, MaxSourceBytes)
		if err != nil || int64(len(content)) != first.Bytes || evidencefile.SHA256(content) != first.SHA256 {
			return fail("blocked", "catalog.digest_mismatch", "An exported artifact changed before package validation.", 4)
		}
		if authoring.ContainsLikelyCredentialValue(content) {
			return fail("blocked", "source.private_value", "Selected raw source bytes contain a likely concrete credential.", 4)
		}
		candidates, err := apitools.BuildOperationCandidates(ctx, apitools.OperationCandidateOptions{Sources: []apitools.OperationSourceInput{{Kind: first.Kind, Name: source.SourceID, Path: selected.Path, Content: content}}, MaxBytes: MaxSourceBytes, MaxOperations: MaxSourceOps, MaxCandidates: MaxSourceOps})
		found := false
		for _, candidate := range candidates.Candidates {
			if candidate.Source.Selector == first.Selector && candidate.Source.SHA256 == first.SHA256 && candidate.Source.Kind == first.Kind {
				found = true
			}
		}
		if err != nil || !found {
			return fail("blocked", "catalog.selector_mismatch", "The selected native operation does not bind to the exported raw artifact.", 4)
		}
		inputPath := filepath.Join(exportRoot, filepath.FromSlash(selected.Path))
		packagePath, _, valid := sourcePackagePath(string(first.Kind), source.SourceID, inputPath)
		if !valid {
			return fail("blocked", "source.format_unsupported", "The native source format is unsupported by package source provisioning.", 4)
		}
		local.Sources = append(local.Sources, SourceAddEntry{SourceKind: string(first.Kind), SourceID: source.SourceID, SourcePath: inputPath, SourceSHA256: "sha256:" + first.SHA256})
		for _, ref := range source.References {
			matched := false
			for _, artifact := range export.Artifacts {
				if artifact.Reference == ref {
					provenance.Artifacts = append(provenance.Artifacts, CatalogSourceArtifact{Reference: ref, RegisteredKind: artifact.RegisteredKind, Advisory: artifact.Advisory, SourceURL: artifact.SourceURL, SourceID: source.SourceID, PackagePath: packagePath})
					matched = true
					break
				}
			}
			if !matched {
				return fail("blocked", "catalog.reference_mismatch", "Native export omitted selected provider provenance.", 4)
			}
		}
	}
	additional := []artifactwriter.GeneratedFile{}
	if len(export.SecurityOverlays) > 64 {
		return fail("blocked", "catalog.overlay_limit", "Selected security provenance exceeds its supported count.", 4)
	}
	for _, overlay := range export.SecurityOverlays {
		content, err := readWithin(exportRoot, overlay.Path, 2<<20)
		if err != nil || evidencefile.SHA256(content) != overlay.SHA256 || authoring.ContainsLikelyCredentialValue(content) {
			return fail("blocked", "catalog.overlay_mismatch", "Selected security provenance failed integrity validation.", 4)
		}
		totalBytes += int64(len(content))
		if totalBytes > MaxCandidateSourceBytesTotal {
			return fail("blocked", "source.byte_limit", "Selected catalog provenance exceeds its supported byte budget.", 4)
		}
		overlay.Path = provenanceDirectory + "/" + overlay.Path
		provenance.SecurityOverlays = append(provenance.SecurityOverlays, overlay)
		additional = append(additional, artifactwriter.GeneratedFile{Path: filepath.Join(root, filepath.FromSlash(overlay.Path)), Content: string(content), Action: "write", Reason: "preserve selected advisory catalog security provenance"})
	}
	provenanceBytes, err := json.MarshalIndent(provenance, "", "  ")
	if err != nil || len(provenanceBytes)+1 > maxSourceManifestBytes || authoring.ContainsLikelyCredentialValue(provenanceBytes) {
		return fail("blocked", "catalog.provenance_limit", "Selected catalog provenance exceeds its supported bound.", 4)
	}
	additional = append(additional, artifactwriter.GeneratedFile{Path: filepath.Join(root, filepath.FromSlash(provenancePath)), Content: string(provenanceBytes) + "\n", Action: "write", Reason: "record exact confirmed catalog source and native selector provenance"})
	outcome := addSources(ctx, root, local, additional)
	outcome.Result.Version = CatalogSourceWireVersion
	if outcome.Result.Result != nil {
		outcome.Result.Result.ProvenancePath = provenancePath
	}
	return outcome
}
