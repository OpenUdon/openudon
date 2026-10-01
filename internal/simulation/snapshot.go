package simulation

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/uwsexec"
	"github.com/OpenUdon/uws/uws1"
)

type packageSnapshot struct {
	root, scope, digest string
	files               map[string][]byte
	document            *uws1.Document
}

// capture accepts unresolved review packages without interpreting stored
// quality as authority. It consumes bounded captured bytes, never live APIs.
func capture(ctx context.Context, repoRoot, example string) (packageSnapshot, error) {
	var snapshot packageSnapshot
	root, err := filepath.Abs(example)
	if err != nil {
		return snapshot, err
	}
	repo, err := filepath.Abs(repoRoot)
	if err != nil {
		return snapshot, err
	}
	scope, err := filepath.Rel(repo, root)
	if err != nil || scope == ".." || strings.HasPrefix(scope, ".."+string(filepath.Separator)) {
		return snapshot, errors.New("package must be inside the selected repository root")
	}
	if packageartifacts.ValidatePackageRoot(root) != nil {
		return snapshot, errors.New("unsafe package root")
	}
	handoff, _, err := evidencefile.ReadRegular(filepath.Join(root, packageartifacts.ReviewHandoffPath), evidencefile.DefaultMaxBytes)
	if err != nil {
		return snapshot, err
	}
	var manifest authoring.ReviewHandoff
	if evidencefile.DecodeStrict(handoff, &manifest) != nil || len(authoring.ValidateReviewHandoff(manifest, authoring.ReviewHandoffValidationOptions{AllowedVersions: []string{authoring.ReviewHandoffVersion, authoring.LegacyReviewHandoffVersion}})) != 0 {
		return snapshot, errors.New("invalid review handoff")
	}
	inputs := make([]packageartifacts.ManifestInput, 0, len(manifest.HandoffInputs))
	for _, input := range manifest.HandoffInputs {
		inputs = append(inputs, packageartifacts.ManifestInput{Path: input.Path, Required: input.Required})
	}
	paths, err := packageartifacts.RequiredManifestPaths(root, inputs)
	if err != nil || len(paths) > 1024 {
		return snapshot, errors.New("invalid or oversized package inventory")
	}
	if packageartifacts.ValidateRegularPackageFiles(root, paths) != nil {
		return snapshot, errors.New("unsafe package files")
	}
	files := make(map[string][]byte, len(paths))
	total := 0
	for _, path := range paths {
		if ctx.Err() != nil {
			return snapshot, ctx.Err()
		}
		data, _, e := evidencefile.ReadRegular(filepath.Join(root, filepath.FromSlash(path)), evidencefile.DefaultMaxBytes)
		if e != nil {
			return snapshot, e
		}
		total += len(data)
		if total > 32<<20 {
			return snapshot, errors.New("simulation package exceeds capture bound")
		}
		files[path] = data
	}
	digest, err := authoring.ComputeReviewHandoffDigest(authoring.ReviewHandoffDigestOptions{Context: ctx, Root: root, Scope: filepath.ToSlash(scope), Version: "openudon.handoff-package-digest.v1", Inputs: manifest.HandoffInputs, InputBytes: files})
	if err != nil {
		return snapshot, err
	}
	doc, err := uwsexec.DecodeDocument(files["workflows/workflow.uws.yaml"], uwsexec.DocumentFormatYAML)
	if err != nil || doc.Validate() != nil {
		return snapshot, errors.New("invalid public UWS document")
	}
	hcl, err := uwsexec.DecodeDocument(files["workflows/workflow.hcl"], uwsexec.DocumentFormatHCL)
	if err != nil || hcl.Validate() != nil {
		return snapshot, errors.New("invalid public UWS HCL document")
	}
	// Both captured representations must have the same semantics, including
	// unresolved contracts; disagreement is evidence corruption, not approval.
	// The public HCL and YAML decoders differ only in nil versus empty
	// operation inventories for pending-only documents. Normalize that empty
	// inventory without removing any contract or executable content.
	if len(doc.Operations) == 0 {
		doc.Operations = []*uws1.Operation{}
	}
	if len(hcl.Operations) == 0 {
		hcl.Operations = []*uws1.Operation{}
	}
	left, err := uwsexec.MarshalDocument(doc, uwsexec.DocumentFormatJSON)
	if err != nil {
		return snapshot, err
	}
	right, err := uwsexec.MarshalDocument(hcl, uwsexec.DocumentFormatJSON)
	if err != nil || string(left) != string(right) {
		return snapshot, errors.New("public UWS representations disagree")
	}
	return packageSnapshot{root: root, scope: scope, digest: digest, files: files, document: doc}, nil
}
