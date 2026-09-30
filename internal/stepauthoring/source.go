package stepauthoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/artifactwriter"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

const (
	SourceAddCommand       = "step.source.add"
	APISourceManifestPath  = packageartifacts.APISourceManifestPath
	apiSourceManifestV1    = "openudon.api-source-manifest.v1"
	maxSourceAddEntries    = 16
	maxSourceManifestBytes = 1 << 20
)

// SourceManifestRevision binds a package write to the exact manifest revision
// rendered in Kinet's approved proposal.
type SourceManifestRevision struct {
	State  string `json:"state"`
	SHA256 string `json:"sha256,omitempty"`
}

// SourceAddEntry names one explicitly selected local source file and its
// proposal-approved digest. Output paths are derived from kind and ID.
type SourceAddEntry struct {
	SourceKind   string `json:"source_kind"`
	SourceID     string `json:"source_id"`
	SourcePath   string `json:"source_path"`
	SourceSHA256 string `json:"source_sha256"`
}

type SourceAddRequest struct {
	Version          string                 `json:"version"`
	Kind             string                 `json:"kind"`
	Command          string                 `json:"command"`
	ManifestRevision SourceManifestRevision `json:"manifest_revision"`
	Sources          []SourceAddEntry       `json:"sources"`
}

type APISourceManifest struct {
	Version string                   `json:"version"`
	Sources []APISourceManifestEntry `json:"sources"`
}

type APISourceManifestEntry struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	PackagePath string `json:"package_path"`
	SHA256      string `json:"sha256"`
}

type SourceAddResultEntry struct {
	SourceID          string `json:"source_id"`
	CandidateSourceID string `json:"candidate_source_id"`
	SourceKind        string `json:"source_kind"`
	PackagePath       string `json:"package_path"`
	SHA256            string `json:"sha256"`
}

type SourceAddData struct {
	Sources        []SourceAddResultEntry `json:"sources"`
	ManifestPath   string                 `json:"manifest_path"`
	ManifestSHA256 string                 `json:"manifest_sha256"`
}

type SourceAddWireResult struct {
	Version       string         `json:"version"`
	Kind          string         `json:"kind"`
	Command       string         `json:"command"`
	Status        string         `json:"status"`
	Diagnostics   []Diagnostic   `json:"diagnostics"`
	WriteOutcome  string         `json:"write_outcome,omitempty"`
	AffectedPaths []string       `json:"affected_paths,omitempty"`
	Result        *SourceAddData `json:"result,omitempty"`
}

type SourceAddOutcome struct {
	Result   SourceAddWireResult
	ExitCode int
}

type preparedSource struct {
	entry   APISourceManifestEntry
	content []byte
}

// AddSources validates explicitly named local API documents using APItools,
// then atomically installs those documents and a package-relative provenance
// manifest. It never fetches URLs or writes outside the selected package.
func AddSources(ctx context.Context, exampleDir string, request SourceAddRequest) SourceAddOutcome {
	if err := validateSourceAddRequest(request); err != nil {
		return sourceAddFailure("failed", "request.invalid", "The step-source-add request is invalid.", 2)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return sourceAddFailure("blocked", "request.cancelled", "Source provisioning was cancelled before validation completed.", 4)
	}
	root, err := resolveExampleRoot(exampleDir)
	if err != nil {
		return sourceAddFailure("blocked", "example.invalid", "The selected workflow package is unavailable or unsafe.", 4)
	}

	manifest, conflict := loadSourceManifest(root, request.ManifestRevision)
	if conflict {
		return sourceAddFailure("conflict", "manifest.stale", "The package source manifest changed after proposal review.", 3)
	}
	if manifest.Version != "" && manifest.Version != apiSourceManifestV1 {
		return sourceAddFailure("blocked", "manifest.unsupported", "The package source manifest uses an unsupported version.", 4)
	}
	manifest, err = normalizeAndValidateSourceManifest(manifest)
	if err != nil {
		return sourceAddFailure("blocked", "manifest.invalid", "The package source manifest is invalid or ambiguous.", 4)
	}
	for _, source := range manifest.Sources {
		content, err := readWithin(root, source.PackagePath, MaxSourceBytes)
		if err != nil || "sha256:"+evidencefile.SHA256(content) != source.SHA256 {
			return sourceAddFailure("blocked", "manifest.invalid", "A recorded source no longer matches its provenance manifest.", 4)
		}
	}

	knownIDs := make(map[string]struct{}, len(manifest.Sources))
	knownPaths := make(map[string]struct{}, len(manifest.Sources))
	for _, source := range manifest.Sources {
		knownIDs[source.ID] = struct{}{}
		knownPaths[source.PackagePath] = struct{}{}
	}

	prepared := make([]preparedSource, 0, len(request.Sources))
	apiDocuments := make([]apitools.APISourceDocument, 0, len(request.Sources))
	seenInputPaths := make(map[string]struct{}, len(request.Sources))
	var totalBytes int64
	for _, source := range request.Sources {
		if err := ctx.Err(); err != nil {
			return sourceAddFailure("blocked", "request.cancelled", "Source provisioning was cancelled before validation completed.", 4)
		}
		if _, exists := knownIDs[source.SourceID]; exists {
			return sourceAddFailure("conflict", "source.id_exists", "A source with the requested ID is already recorded in this package.", 3)
		}
		packagePath, extension, ok := sourcePackagePath(source.SourceKind, source.SourceID, source.SourcePath)
		if !ok {
			return sourceAddFailure("failed", "source.invalid", "A requested source kind, ID, or file type is unsupported.", 2)
		}
		if _, exists := knownPaths[packagePath]; exists {
			return sourceAddFailure("conflict", "source.path_exists", "A source with the requested package path is already recorded.", 3)
		}
		if err := ensureSourceIDAvailable(root, source.SourceKind, source.SourceID); err != nil {
			return sourceAddFailure("conflict", "source.id_exists", "A source with the requested ID is already present in this package.", 3)
		}
		inputPath, content, err := readExplicitSource(source.SourcePath)
		if err != nil {
			return sourceAddFailure("blocked", "source.unavailable", "A requested local source could not be read as a safe regular file.", 4)
		}
		if filepath.Ext(inputPath) != extension || len(content) == 0 || !utf8.Valid(content) {
			return sourceAddFailure("blocked", "source.format_unsupported", "A requested local source has an unsupported extension or encoding.", 4)
		}
		if _, exists := seenInputPaths[inputPath]; exists {
			return sourceAddFailure("failed", "source.duplicate", "The same local source file was supplied more than once.", 2)
		}
		seenInputPaths[inputPath] = struct{}{}
		totalBytes += int64(len(content))
		if totalBytes > MaxCandidateSourceBytesTotal {
			return sourceAddFailure("blocked", "source.byte_limit", "Selected local sources exceed the combined byte limit.", 4)
		}
		digest := "sha256:" + evidencefile.SHA256(content)
		if digest != source.SourceSHA256 {
			return sourceAddFailure("conflict", "source.digest_mismatch", "A local source changed after proposal review.", 3)
		}
		if err := ensurePackageTargetAbsent(root, packagePath); err != nil {
			return sourceAddFailure("conflict", "source.path_exists", "A requested source package path is already present or unsafe.", 3)
		}
		knownIDs[source.SourceID] = struct{}{}
		knownPaths[packagePath] = struct{}{}
		item := preparedSource{
			entry:   APISourceManifestEntry{ID: source.SourceID, Kind: source.SourceKind, PackagePath: packagePath, SHA256: digest},
			content: content,
		}
		prepared = append(prepared, item)
		apiDocuments = append(apiDocuments, apitools.APISourceDocument{
			Kind: source.SourceKind, Name: source.SourceID, Path: packagePath,
			RelativePath: packagePath, Content: content,
		})
	}

	inventory, err := apitools.BuildAPISourceOperationInventory(ctx, apitools.APISourceInventoryOptions{
		Documents: apiDocuments, MaxBytes: MaxSourceBytes, MaxOperations: MaxSourceOps,
	})
	if err != nil || inventory.Truncated || hasAPISourceInventoryErrors(inventory.Diagnostics) || len(inventory.Documents) != len(prepared) {
		return sourceAddFailure("blocked", "source.validation_failed", "APItools could not completely validate the selected local source documents.", 4)
	}
	if err := ctx.Err(); err != nil {
		return sourceAddFailure("blocked", "request.cancelled", "Source provisioning was cancelled before the package write.", 4)
	}

	for _, item := range prepared {
		manifest.Sources = append(manifest.Sources, item.entry)
	}
	sort.Slice(manifest.Sources, func(i, j int) bool {
		return manifest.Sources[i].PackagePath < manifest.Sources[j].PackagePath
	})
	manifest.Version = apiSourceManifestV1
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil || len(manifestBytes)+1 > maxSourceManifestBytes {
		return sourceAddFailure("blocked", "manifest.invalid", "The resulting source manifest exceeds its supported bound.", 4)
	}
	manifestBytes = append(manifestBytes, '\n')

	files := make([]artifactwriter.GeneratedFile, 0, len(prepared)+1)
	resultSources := make([]SourceAddResultEntry, 0, len(prepared))
	for _, item := range prepared {
		files = append(files, artifactwriter.GeneratedFile{
			Path: filepath.Join(root, filepath.FromSlash(item.entry.PackagePath)), Content: string(item.content),
			Action: "write", Reason: "install explicitly approved local API source document",
		})
		resultSources = append(resultSources, SourceAddResultEntry{
			SourceID: item.entry.ID, CandidateSourceID: sourceIDForPath(item.entry.PackagePath), SourceKind: item.entry.Kind,
			PackagePath: item.entry.PackagePath, SHA256: item.entry.SHA256,
		})
	}
	manifestFile := artifactwriter.GeneratedFile{
		Path: filepath.Join(root, filepath.FromSlash(APISourceManifestPath)), Content: string(manifestBytes),
		Action: "write", Reason: "record exact local source provenance for package handoff",
	}
	if request.ManifestRevision.State == "present" {
		manifestFile.AllowOverwrite = true
		manifestFile.ExpectedCurrentSHA256 = request.ManifestRevision.SHA256
	}
	files = append(files, manifestFile)
	if _, err := artifactwriter.CommitChecked(artifactwriter.Prepared{ExampleRoot: root, Files: files}, false, nil); err != nil {
		var transactionErr *artifactwriter.TransactionError
		if errors.As(err, &transactionErr) {
			paths := make([]string, 0, len(files))
			for _, file := range files {
				relative, relErr := filepath.Rel(root, file.Path)
				if relErr != nil {
					continue
				}
				paths = append(paths, filepath.ToSlash(relative))
			}
			sort.Strings(paths)
			return SourceAddOutcome{Result: SourceAddWireResult{
				Version: WireVersion, Kind: "result", Command: SourceAddCommand, Status: "failed",
				Diagnostics:  []Diagnostic{{Code: "write.indeterminate", Severity: "error", Message: "The atomic source write outcome is indeterminate; inspect the listed package paths before retrying."}},
				WriteOutcome: "indeterminate", AffectedPaths: paths,
			}, ExitCode: 1}
		}
		return sourceAddFailure("failed", "package.write_failed", "The package source files were not committed because the atomic write failed.", 1)
	}
	return sourceAddComplete(SourceAddData{
		Sources: resultSources, ManifestPath: APISourceManifestPath,
		ManifestSHA256: "sha256:" + evidencefile.SHA256(manifestBytes),
	})
}

func validateSourceAddRequest(request SourceAddRequest) error {
	if request.Version != WireVersion || request.Kind != "request" || request.Command != SourceAddCommand ||
		len(request.Sources) == 0 || len(request.Sources) > maxSourceAddEntries {
		return fmt.Errorf("wire envelope or source count is invalid")
	}
	switch request.ManifestRevision.State {
	case "missing":
		if request.ManifestRevision.SHA256 != "" {
			return fmt.Errorf("missing manifest revision cannot have a digest")
		}
	case "present":
		if !sha256Pattern.MatchString(request.ManifestRevision.SHA256) {
			return fmt.Errorf("present manifest revision requires a digest")
		}
	default:
		return fmt.Errorf("manifest revision state is invalid")
	}
	seen := make(map[string]struct{}, len(request.Sources))
	for _, source := range request.Sources {
		if !sourceKindSupported(source.SourceKind) || !sourceIDPattern.MatchString(source.SourceID) ||
			!filepath.IsAbs(source.SourcePath) || strings.TrimSpace(source.SourcePath) != source.SourcePath ||
			len(source.SourcePath) > 4096 ||
			!sha256Pattern.MatchString(source.SourceSHA256) {
			return fmt.Errorf("source entry is invalid")
		}
		if _, exists := seen[source.SourceID]; exists {
			return fmt.Errorf("source IDs must be unique")
		}
		seen[source.SourceID] = struct{}{}
	}
	return nil
}

func sourcePackagePath(kind, id, sourcePath string) (string, string, bool) {
	if !sourceKindSupported(kind) || !sourceIDPattern.MatchString(id) || strings.Contains(sourcePath, "://") {
		return "", "", false
	}
	parsed, err := url.Parse(sourcePath)
	if err != nil || parsed.Scheme != "" || !filepath.IsAbs(sourcePath) {
		return "", "", false
	}
	ext := strings.ToLower(filepath.Ext(sourcePath))
	allowed := false
	switch kind {
	case "openapi", "asyncapi":
		allowed = extIs(ext, ".json", ".yaml", ".yml")
	case "google-discovery", "aws-smithy", "openrpc":
		allowed = ext == ".json"
	case "graphql":
		allowed = extIs(ext, ".graphql", ".gql")
	case "grpc-protobuf":
		allowed = ext == ".proto"
	case "odata":
		allowed = ext == ".xml" || ext == ".json"
	}
	if !allowed {
		return "", "", false
	}
	if ext == ".yml" {
		ext = ".yaml"
	}
	dir := kind
	if kind == "google-discovery" {
		dir = "google-discovery"
	}
	return filepath.ToSlash(filepath.Join(dir, id+ext)), filepath.Ext(sourcePath), true
}

func extIs(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}

func readExplicitSource(path string) (string, []byte, error) {
	if !filepath.IsAbs(path) || strings.ContainsRune(path, '\x00') || strings.Contains(path, "://") {
		return "", nil, fmt.Errorf("source path is not an absolute local path")
	}
	clean := filepath.Clean(path)
	volume := filepath.VolumeName(clean)
	root := volume + string(filepath.Separator)
	rel := strings.TrimPrefix(clean, root)
	if rel == clean || rel == "" || rel == "." {
		return "", nil, fmt.Errorf("source path is invalid")
	}
	current := root
	parts := strings.Split(rel, string(filepath.Separator))
	for index, part := range parts {
		if part == "" || part == "." || part == ".." {
			return "", nil, fmt.Errorf("source path contains an unsafe component")
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", nil, fmt.Errorf("source path is unavailable or traverses a symlink")
		}
		if index < len(parts)-1 && !info.IsDir() {
			return "", nil, fmt.Errorf("source path parent is not a directory")
		}
	}
	content, info, err := evidencefile.ReadRegular(current, MaxSourceBytes)
	if err != nil || !info.Mode().IsRegular() {
		return "", nil, fmt.Errorf("source file is unavailable")
	}
	return clean, content, nil
}

func loadSourceManifest(root string, expected SourceManifestRevision) (APISourceManifest, bool) {
	data, err := readWithin(root, APISourceManifestPath, maxSourceManifestBytes)
	if os.IsNotExist(err) {
		if expected.State != "missing" {
			return APISourceManifest{}, true
		}
		return APISourceManifest{Sources: []APISourceManifestEntry{}}, false
	}
	if err != nil || expected.State != "present" ||
		"sha256:"+evidencefile.SHA256(data) != expected.SHA256 {
		return APISourceManifest{}, true
	}
	var manifest APISourceManifest
	if err := evidencefile.DecodeStrict(data, &manifest); err != nil {
		return APISourceManifest{}, true
	}
	return manifest, false
}

func normalizeAndValidateSourceManifest(manifest APISourceManifest) (APISourceManifest, error) {
	if manifest.Version == "" && len(manifest.Sources) > 0 || (manifest.Version != "" && manifest.Version != apiSourceManifestV1) {
		return APISourceManifest{}, fmt.Errorf("manifest version is invalid")
	}
	seenIDs := make(map[string]struct{}, len(manifest.Sources))
	seenPaths := make(map[string]struct{}, len(manifest.Sources))
	for _, entry := range manifest.Sources {
		if !sourceIDPattern.MatchString(entry.ID) || !sourceKindSupported(entry.Kind) || !sha256Pattern.MatchString(entry.SHA256) {
			return APISourceManifest{}, fmt.Errorf("manifest entry is invalid")
		}
		clean, err := packageartifacts.CleanRelativePath(entry.PackagePath)
		if err != nil || clean != entry.PackagePath || !strings.HasPrefix(clean, entry.Kind+"/") {
			return APISourceManifest{}, fmt.Errorf("manifest path is invalid")
		}
		expectedPath, _, ok := sourcePackagePath(entry.Kind, entry.ID, filepath.Join(string(filepath.Separator), filepath.Base(clean)))
		if !ok || expectedPath != clean {
			return APISourceManifest{}, fmt.Errorf("manifest path does not match its source identity")
		}
		if _, exists := seenIDs[entry.ID]; exists {
			return APISourceManifest{}, fmt.Errorf("manifest source IDs are ambiguous")
		}
		if _, exists := seenPaths[entry.PackagePath]; exists {
			return APISourceManifest{}, fmt.Errorf("manifest source paths are ambiguous")
		}
		seenIDs[entry.ID] = struct{}{}
		seenPaths[entry.PackagePath] = struct{}{}
	}
	return manifest, nil
}

func ensurePackageTargetAbsent(root, relative string) error {
	clean, err := packageartifacts.CleanRelativePath(relative)
	if err != nil || clean != relative {
		return fmt.Errorf("package target is unsafe")
	}
	path := filepath.Join(root, filepath.FromSlash(relative))
	if _, err := os.Lstat(path); err == nil {
		return fmt.Errorf("package target exists")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("package target cannot be inspected")
	}
	return nil
}

func ensureSourceIDAvailable(root, kind, id string) error {
	paths, err := packageartifacts.CollectAPISourcePaths(root)
	if err != nil {
		return err
	}
	for _, path := range paths {
		directory := filepath.ToSlash(filepath.Dir(path))
		if directory == "discovery" {
			directory = "google-discovery"
		}
		if directory == kind && strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)) == id {
			return fmt.Errorf("source identity already exists")
		}
	}
	return nil
}

func hasAPISourceInventoryErrors(diagnostics []apitools.Diagnostic) bool {
	for _, diagnostic := range diagnostics {
		if strings.EqualFold(diagnostic.Severity, "error") {
			return true
		}
	}
	return false
}

func sourceAddFailure(status, code, message string, exitCode int) SourceAddOutcome {
	return SourceAddOutcome{Result: SourceAddWireResult{
		Version: WireVersion, Kind: "result", Command: SourceAddCommand, Status: status,
		Diagnostics: []Diagnostic{{Code: code, Severity: "error", Message: message}},
	}, ExitCode: exitCode}
}

func sourceAddComplete(data SourceAddData) SourceAddOutcome {
	return SourceAddOutcome{Result: SourceAddWireResult{
		Version: WireVersion, Kind: "result", Command: SourceAddCommand, Status: "completed",
		Diagnostics: []Diagnostic{}, Result: &data,
	}}
}
