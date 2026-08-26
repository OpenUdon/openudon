// Package packagepipeline owns prepare, qualification, and promotion of
// complete OpenUdon review-package generations. Preparation is byte-only and
// never mutates the source package.
package packagepipeline

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

const PreparationVersion = "openudon.package-preparation.v1"

const (
	qualityJSONPath      = "expected/quality.json"
	maxPreparedFiles     = 1024
	maxPreparedTotalSize = 64 << 20
)

var prepareReadHook func()

// PrepareOptions binds preparation to one package root, portable scope, and
// optional previously observed input digest.
type PrepareOptions struct {
	ExampleDir          string
	Scope               string
	ExpectedInputSHA256 string
}

// FileIdentity is one canonical package-relative prepared file.
type FileIdentity struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// Manifest is the value-free identity and review summary of a prepared byte
// generation. ManifestSHA256 hashes this structure with that field empty.
type Manifest struct {
	Version         string                          `json:"version"`
	Scope           string                          `json:"scope"`
	InputSHA256     string                          `json:"input_sha256"`
	PackageSHA256   string                          `json:"package_sha256"`
	HandoffSHA256   string                          `json:"handoff_sha256"`
	QualitySHA256   string                          `json:"quality_sha256"`
	QualityStatus   string                          `json:"quality_status"`
	ApprovalStates  []string                        `json:"approval_states"`
	ExecutionPolicy authoring.ReviewExecutionPolicy `json:"execution_policy"`
	CredentialNames []string                        `json:"credential_names,omitempty"`
	Files           []FileIdentity                  `json:"files"`
	ManifestSHA256  string                          `json:"manifest_sha256"`
}

// Prepared is an immutable-by-copy proposed package byte generation.
type Prepared struct {
	root     string
	manifest Manifest
	files    map[string][]byte
	quality  synthesize.QualityReport
	handoff  authoring.ReviewHandoff
}

// PrepareCurrent reads and validates one complete current package generation
// into memory. It writes no package, approval, report, pointer, or runtime
// state and rejects a generation that changes during the bounded read.
func PrepareCurrent(ctx context.Context, opts PrepareOptions) (Prepared, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Prepared{}, err
	}
	root, err := canonicalPackageRoot(opts.ExampleDir)
	if err != nil {
		return Prepared{}, err
	}
	scope, err := canonicalScope(opts.Scope)
	if err != nil {
		return Prepared{}, err
	}
	files, handoff, quality, err := readGeneration(ctx, root)
	if err != nil {
		return Prepared{}, err
	}
	if prepareReadHook != nil {
		prepareReadHook()
	}
	if err := verifyGenerationUnchanged(ctx, root, handoff, files); err != nil {
		return Prepared{}, err
	}
	manifest, err := buildManifest(ctx, scope, files, handoff, quality)
	if err != nil {
		return Prepared{}, err
	}
	if expected := strings.ToLower(strings.TrimSpace(opts.ExpectedInputSHA256)); expected != "" && expected != manifest.InputSHA256 {
		return Prepared{}, fmt.Errorf("package input generation is %s, not expected %s", manifest.InputSHA256, expected)
	}
	return Prepared{root: root, manifest: manifest, files: cloneFiles(files), quality: quality, handoff: handoff}, nil
}

// Manifest returns a defensive copy of the value-free preparation manifest.
func (prepared Prepared) Manifest() Manifest { return cloneManifest(prepared.manifest) }

// Files returns defensive copies of every prepared byte body.
func (prepared Prepared) Files() map[string][]byte { return cloneFiles(prepared.files) }

// Quality returns a defensive copy of the stored passing quality report.
func (prepared Prepared) Quality() synthesize.QualityReport {
	data, _ := json.Marshal(prepared.quality)
	var result synthesize.QualityReport
	_ = json.Unmarshal(data, &result)
	return result
}

// Handoff returns a defensive copy of the reviewed approval and execution
// contract included in this generation.
func (prepared Prepared) Handoff() authoring.ReviewHandoff {
	data, _ := json.Marshal(prepared.handoff)
	var result authoring.ReviewHandoff
	_ = json.Unmarshal(data, &result)
	return result
}

func canonicalPackageRoot(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", errors.New("package example directory is required")
	}
	root, err := filepath.Abs(value)
	if err != nil {
		return "", err
	}
	root = filepath.Clean(root)
	if err := packageartifacts.ValidatePackageRoot(root); err != nil {
		return "", err
	}
	return root, nil
}

func canonicalScope(value string) (string, error) {
	scope := strings.Trim(strings.TrimSpace(filepath.ToSlash(value)), "/")
	if scope == "" {
		return "", errors.New("portable package scope is required")
	}
	clean, err := packageartifacts.CleanRelativePath(scope)
	if err != nil {
		return "", fmt.Errorf("package scope is invalid: %w", err)
	}
	return clean, nil
}

func readGeneration(ctx context.Context, root string) (map[string][]byte, authoring.ReviewHandoff, synthesize.QualityReport, error) {
	handoffPath := filepath.Join(root, filepath.FromSlash(packageartifacts.ReviewHandoffPath))
	handoffBytes, _, err := evidencefile.ReadRegular(handoffPath, evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("read review handoff: %w", err)
	}
	var handoff authoring.ReviewHandoff
	if err := evidencefile.DecodeStrict(handoffBytes, &handoff); err != nil {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("decode review handoff: %w", err)
	}
	if diagnostics := authoring.ValidateReviewHandoff(handoff, authoring.ReviewHandoffValidationOptions{AllowedVersions: []string{authoring.ReviewHandoffVersion}}); len(diagnostics) > 0 {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("review handoff is invalid: %s", diagnostics[0].Message)
	}
	if handoff.CredentialBindings.ValuesAllowedInArtifacts || handoff.ExecutionPolicy.DirectProductionExecution {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, errors.New("review handoff grants forbidden artifact values or direct production execution")
	}
	inputs := make([]packageartifacts.ManifestInput, 0, len(handoff.HandoffInputs))
	for _, input := range handoff.HandoffInputs {
		inputs = append(inputs, packageartifacts.ManifestInput{Path: input.Path, Required: input.Required})
	}
	paths, err := packageartifacts.RequiredManifestPaths(root, inputs)
	if err != nil {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, err
	}
	if len(paths) == 0 || len(paths) > maxPreparedFiles {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("prepared package file count must be between 1 and %d", maxPreparedFiles)
	}
	if err := packageartifacts.ValidateRegularPackageFiles(root, paths); err != nil {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, err
	}
	files := make(map[string][]byte, len(paths))
	var totalBytes int64
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, err
		}
		data, _, err := evidencefile.ReadRegular(filepath.Join(root, filepath.FromSlash(path)), evidencefile.DefaultMaxBytes)
		if err != nil {
			return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("read prepared input %s: %w", path, err)
		}
		totalBytes += int64(len(data))
		if totalBytes > maxPreparedTotalSize {
			return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("prepared package exceeds %d bytes", maxPreparedTotalSize)
		}
		files[path] = data
	}
	if err := validateHandoffDigests(handoff, files); err != nil {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, err
	}
	qualityBytes, ok := files[qualityJSONPath]
	if !ok {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, errors.New("prepared package omits quality report")
	}
	var quality synthesize.QualityReport
	if err := evidencefile.DecodeStrict(qualityBytes, &quality); err != nil {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("decode quality report: %w", err)
	}
	if !quality.Passed() {
		return nil, authoring.ReviewHandoff{}, synthesize.QualityReport{}, fmt.Errorf("prepared quality status is %q", quality.Status)
	}
	return files, handoff, quality, nil
}

func validateHandoffDigests(handoff authoring.ReviewHandoff, files map[string][]byte) error {
	for _, input := range handoff.HandoffInputs {
		if !input.Required {
			continue
		}
		path, err := packageartifacts.CleanRelativePath(input.Path)
		if err != nil {
			return err
		}
		var got string
		if path == packageartifacts.ReviewHandoffPath {
			got, err = authoring.ReviewHandoffSelfDigest(handoff, path)
		} else if data, ok := files[path]; ok {
			got = evidencefile.SHA256(data)
		} else {
			err = fmt.Errorf("snapshot missing handoff input %s", path)
		}
		if err != nil {
			return err
		}
		if got != strings.ToLower(strings.TrimSpace(input.SHA256)) {
			return fmt.Errorf("handoff input SHA-256 mismatch for %s", path)
		}
	}
	return nil
}

func verifyGenerationUnchanged(ctx context.Context, root string, handoff authoring.ReviewHandoff, expected map[string][]byte) error {
	inputs := make([]packageartifacts.ManifestInput, 0, len(handoff.HandoffInputs))
	for _, input := range handoff.HandoffInputs {
		inputs = append(inputs, packageartifacts.ManifestInput{Path: input.Path, Required: input.Required})
	}
	paths, err := packageartifacts.RequiredManifestPaths(root, inputs)
	if err != nil || !samePreparedPaths(paths, sortedFilePaths(expected)) {
		return errors.New("package input generation changed during preparation")
	}
	for _, path := range sortedFilePaths(expected) {
		if err := ctx.Err(); err != nil {
			return err
		}
		data, _, err := evidencefile.ReadRegular(filepath.Join(root, filepath.FromSlash(path)), evidencefile.DefaultMaxBytes)
		if err != nil || evidencefile.SHA256(data) != evidencefile.SHA256(expected[path]) {
			return fmt.Errorf("package input generation changed during preparation at %s", path)
		}
	}
	return nil
}

func samePreparedPaths(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := range first {
		if first[index] != second[index] {
			return false
		}
	}
	return true
}

func buildManifest(ctx context.Context, scope string, files map[string][]byte, handoff authoring.ReviewHandoff, quality synthesize.QualityReport) (Manifest, error) {
	packageSHA, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Context: ctx, Scope: scope, Version: "openudon.handoff-package-digest.v1", Inputs: handoff.HandoffInputs, InputBytes: files,
	})
	if err != nil {
		return Manifest{}, err
	}
	inputSHA, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{
		Context: ctx, Scope: scope, Version: "openudon.package-preparation-input.v1", Inputs: handoff.HandoffInputs, InputBytes: files,
	})
	if err != nil {
		return Manifest{}, err
	}
	identities := make([]FileIdentity, 0, len(files))
	for _, path := range sortedFilePaths(files) {
		identities = append(identities, FileIdentity{Path: path, SHA256: evidencefile.SHA256(files[path]), Bytes: int64(len(files[path]))})
	}
	approvalStates := make([]string, 0, len(handoff.ApprovalStates))
	for _, state := range handoff.ApprovalStates {
		approvalStates = append(approvalStates, state.Name)
	}
	credentialNames := append([]string(nil), handoff.CredentialBindings.Declared...)
	sort.Strings(credentialNames)
	manifest := Manifest{
		Version: PreparationVersion, Scope: scope, InputSHA256: inputSHA, PackageSHA256: packageSHA,
		HandoffSHA256: evidencefile.SHA256(files[packageartifacts.ReviewHandoffPath]),
		QualitySHA256: evidencefile.SHA256(files[qualityJSONPath]), QualityStatus: quality.Status,
		ApprovalStates: approvalStates, ExecutionPolicy: handoff.ExecutionPolicy, CredentialNames: credentialNames, Files: identities,
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return Manifest{}, err
	}
	sum := sha256.Sum256(data)
	manifest.ManifestSHA256 = "sha256:" + hex.EncodeToString(sum[:])
	return manifest, nil
}

func sortedFilePaths(files map[string][]byte) []string {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func cloneFiles(files map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(files))
	for path, data := range files {
		result[path] = append([]byte(nil), data...)
	}
	return result
}

func cloneManifest(manifest Manifest) Manifest {
	manifest.Files = append([]FileIdentity(nil), manifest.Files...)
	manifest.ApprovalStates = append([]string(nil), manifest.ApprovalStates...)
	manifest.CredentialNames = append([]string(nil), manifest.CredentialNames...)
	return manifest
}
