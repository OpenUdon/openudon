package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	evdigest "github.com/OpenUdon/evidence/digest"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

// ReviewArtifactInput is caller-bound artifact metadata used to assemble a
// runtime-neutral review package or handoff manifest.
type ReviewArtifactInput struct {
	Path      string `json:"path"`
	Kind      string `json:"kind,omitempty"`
	MediaType string `json:"media_type,omitempty"`
	Purpose   string `json:"purpose,omitempty"`
	SizeBytes int    `json:"size_bytes,omitempty"`
	Required  bool   `json:"required,omitempty"`
}

// ReviewPackageInput builds prompt-safe review metadata without artifact
// content.
type ReviewPackageInput struct {
	Name                    string
	Source                  string
	Artifacts               []ReviewArtifactInput
	Diagnostics             []Diagnostic
	ReadinessIssues         []ReadinessIssue
	SymbolicBindings        []SymbolicBinding
	BindingNames            []string
	Assumptions             []Assumption
	QuestionPlan            QuestionPlan
	Transcript              *Transcript
	BindingContract         BindingContract
	DeferredExecutionPolicy DeferredExecutionPolicy
}

// ReviewHandoffInputsFromArtifacts converts artifact metadata to stable,
// deduplicated handoff inputs and appends stable extra inputs.
func ReviewHandoffInputsFromArtifacts(artifacts []ReviewArtifactInput, extra ...ReviewHandoffInput) []ReviewHandoffInput {
	seen := map[string]struct{}{}
	var inputs []ReviewHandoffInput
	add := func(input ReviewHandoffInput) {
		input.Path = strings.TrimSpace(input.Path)
		input.Purpose = strings.TrimSpace(input.Purpose)
		if input.Path == "" {
			return
		}
		clean, ok := cleanReviewHandoffInputPath(input.Path)
		if !ok {
			input.Path = strings.TrimSpace(input.Path)
		} else {
			input.Path = clean
		}
		if _, ok := seen[input.Path]; ok {
			return
		}
		seen[input.Path] = struct{}{}
		inputs = append(inputs, input)
	}
	for _, artifact := range artifacts {
		required := artifact.Required
		if !required {
			required = true
		}
		add(ReviewHandoffInput{
			Path:     artifact.Path,
			Purpose:  firstNonEmpty(artifact.Purpose, "Reviewable artifact."),
			Required: required,
		})
	}
	for _, input := range extra {
		add(input)
	}
	sort.SliceStable(inputs, func(i, j int) bool { return inputs[i].Path < inputs[j].Path })
	return inputs
}

// ReviewHandoffDigestOptions configures a deterministic digest over required
// handoff input files.
type ReviewHandoffDigestOptions struct {
	Context context.Context
	Root    string
	Scope   string
	Version string
	Inputs  []ReviewHandoffInput
}

type reviewHandoffDigestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type reviewHandoffDigest struct {
	Version string                    `json:"version"`
	Scope   string                    `json:"scope"`
	Files   []reviewHandoffDigestFile `json:"files"`
}

// ComputeReviewHandoffDigest hashes the required files referenced by a handoff
// manifest. The digest includes file paths and file SHA-256s, not file content.
func ComputeReviewHandoffDigest(opts ReviewHandoffDigestOptions) (string, error) {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root := strings.TrimSpace(opts.Root)
	if root == "" {
		root = "."
	}
	scope := strings.Trim(strings.TrimSpace(filepath.ToSlash(opts.Scope)), "/")
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "openudon.review-handoff-digest.v1"
	}
	fileSet := map[string]struct{}{}
	for _, input := range opts.Inputs {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if !input.Required {
			continue
		}
		clean, err := packageartifacts.CleanRelativePath(input.Path)
		if err != nil {
			return "", fmt.Errorf("handoff input path must be safe relative path: %q", input.Path)
		}
		fileSet[clean] = struct{}{}
	}
	files := make([]string, 0, len(fileSet))
	for path := range fileSet {
		files = append(files, path)
	}
	sort.Strings(files)
	if err := packageartifacts.ValidateRegularPackageFiles(root, files); err != nil {
		return "", err
	}
	digest := reviewHandoffDigest{
		Version: version,
		Scope:   scope,
	}
	for _, path := range files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		full := filepath.Join(root, filepath.FromSlash(path))
		data, _, err := evidencefile.ReadRegular(full, evidencefile.DefaultMaxBytes)
		if err != nil {
			return "", fmt.Errorf("read handoff input %s: %w", path, err)
		}
		reportPath := path
		if scope != "" {
			reportPath = filepath.ToSlash(filepath.Join(scope, path))
		}
		digest.Files = append(digest.Files, reviewHandoffDigestFile{
			Path:   reportPath,
			SHA256: evdigest.SHA256Bytes(data).Value,
		})
	}
	canonical, err := json.Marshal(digest)
	if err != nil {
		return "", err
	}
	return evdigest.SHA256Bytes(canonical).Value, nil
}
