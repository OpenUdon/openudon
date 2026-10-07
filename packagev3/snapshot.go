package packagev3

import (
	"bytes"
	"context"
	"github.com/OpenUdon/openudon/credentialpolicy"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/trust"
)

type snapshot struct {
	files      map[string][]byte
	manifest   Manifest
	assessment Assessment
	handoff    Handoff
	sha256     string
}

// inspectSnapshot checks byte inventory only. It is private and deliberately
// cannot return VerifiedPackage or supply source proof/execution authority.
func inspectSnapshot(ctx context.Context, options VerifyOptions) (snapshot, error) {
	if ctx == nil || !scopeValid(options.Scope) || !digestValid(options.ExpectedSHA256) || len(options.Files) < 6 || len(options.Files) > MaxFiles {
		return snapshot{}, ErrPackage
	}
	files := make(map[string][]byte, len(options.Files))
	total := 0
	for path, data := range options.Files {
		if ctx.Err() != nil {
			return snapshot{}, ctx.Err()
		}
		if !publicPath(path) || len(data) == 0 || len(data) > MaxFileBytes || len(data) > MaxTotalBytes-total {
			return snapshot{}, ErrPackage
		}
		if credentialpolicy.ContainsLikelyValue(data) || options.Private != nil && options.Private(data) {
			return snapshot{}, ErrPackage
		}
		files[path] = append([]byte(nil), data...)
		total += len(data)
	}
	manifest, err := ParseManifest(files[ManifestPath])
	if err != nil || manifest.Scope != options.Scope {
		return snapshot{}, ErrPackage
	}
	canonical, err := manifest.Marshal()
	if err != nil || !bytes.Equal(canonical, files[ManifestPath]) {
		return snapshot{}, ErrPackage
	}
	inputs, err := manifest.InputArtifacts()
	if err != nil || len(files) != len(inputs)+3 {
		return snapshot{}, ErrPackage
	}
	expected := map[string]bool{ManifestPath: true, AssessmentPath: true, HandoffPath: true}
	for _, input := range inputs {
		expected[input.Path] = true
		if hashBytes(files[input.Path]) != input.SHA256 {
			return snapshot{}, ErrPackage
		}
	}
	for path := range files {
		if !expected[path] {
			return snapshot{}, ErrPackage
		}
	}
	review, err := ParseHandoff(files[HandoffPath])
	if err != nil || review.Scope != options.Scope || len(review.Artifacts) != len(files)-1 {
		return snapshot{}, ErrPackage
	}
	canonical, err = review.Marshal()
	if err != nil || !bytes.Equal(canonical, files[HandoffPath]) {
		return snapshot{}, ErrPackage
	}
	for _, artifact := range review.Artifacts {
		data, exists := files[artifact.Path]
		if !exists || hashBytes(data) != artifact.SHA256 {
			return snapshot{}, ErrPackage
		}
	}
	assessment, err := ParseAssessment(files[AssessmentPath])
	if err != nil {
		return snapshot{}, ErrPackage
	}
	canonical, err = assessment.Marshal()
	if err != nil || !bytes.Equal(canonical, files[AssessmentPath]) {
		return snapshot{}, ErrPackage
	}
	inputDigest, err := manifest.InputDigest()
	if err != nil || review.InputsSHA256 != inputDigest || assessment.InputsSHA256 != inputDigest || assessment.Scope != options.Scope || review.ManifestSHA256 != hashBytes(files[ManifestPath]) || review.AssessmentSHA256 != hashBytes(files[AssessmentPath]) || assessment.ManifestSHA256 != review.ManifestSHA256 {
		return snapshot{}, ErrPackage
	}
	artifacts := make([]handoff.DigestFile, 0, len(files))
	for path, data := range files {
		artifacts = append(artifacts, handoff.DigestFile{Path: path, SHA256: hashBytes(data)})
	}
	sum, err := handoff.DigestFiles(options.Scope, trust.PackageDigestVersion, artifacts)
	if err != nil || sum != options.ExpectedSHA256 {
		return snapshot{}, ErrPackage
	}
	if ctx.Err() != nil {
		return snapshot{}, ctx.Err()
	}
	return snapshot{files: files, manifest: manifest, assessment: assessment, handoff: review, sha256: sum}, nil
}
