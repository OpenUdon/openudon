package packagev3

import (
	"bytes"
	"context"
	"github.com/OpenUdon/uws/binding"
	"reflect"

	"github.com/OpenUdon/openudon/credentialpolicy"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/trust"
)

type VerifyOptions struct {
	Scope           string
	ExpectedSHA256  string
	Files           map[string][]byte
	Private         func([]byte) bool
	RuntimeVerifier RuntimeVerifier
}

// VerifiedPackage owns a private independent snapshot. Its zero value is not
// verified; public metadata and producer flags cannot manufacture this value.
// Verification establishes reviewed bytes/shape correspondence, never a grant.
type VerifiedPackage struct {
	files      map[string][]byte
	manifest   Manifest
	assessment Assessment
	handoff    Handoff
	sha256     string
}

func (v VerifiedPackage) SHA256() string { return v.sha256 }
func (v VerifiedPackage) Assessment() Assessment {
	a := v.assessment
	a.Findings = append([]Finding{}, a.Findings...)
	return a
}

// Snapshot returns independent copies. A caller changing them must verify the
// changed bytes again; authority derivation uses the retained private snapshot.
func (v VerifiedPackage) Snapshot() map[string][]byte {
	if v.files == nil {
		return nil
	}
	out := make(map[string][]byte, len(v.files))
	for path, data := range v.files {
		out[path] = append([]byte(nil), data...)
	}
	return out
}

// Verify checks a complete closed package inventory and independently reruns
// source-shape/assessment checks against copied exact bytes. The trusted host
// supplies the expected scope/digest and owns isolation and current approval.
func Verify(ctx context.Context, options VerifyOptions) (VerifiedPackage, error) {
	if ctx == nil || !scopeValid(options.Scope) || !digestValid(options.ExpectedSHA256) || len(options.Files) < 6 || len(options.Files) > MaxFiles {
		return VerifiedPackage{}, ErrPackage
	}
	files := make(map[string][]byte, len(options.Files))
	total := 0
	for path, data := range options.Files {
		if ctx.Err() != nil {
			return VerifiedPackage{}, ctx.Err()
		}
		if !publicPath(path) || len(data) == 0 || len(data) > MaxFileBytes || len(data) > MaxTotalBytes-total {
			return VerifiedPackage{}, ErrPackage
		}
		if credentialpolicy.ContainsLikelyValue(data) || options.Private != nil && options.Private(data) {
			return VerifiedPackage{}, ErrPackage
		}
		files[path] = append([]byte(nil), data...)
		total += len(data)
	}
	manifest, err := ParseManifest(files[ManifestPath])
	if err != nil || manifest.Scope != options.Scope {
		return VerifiedPackage{}, ErrPackage
	}
	canonical, err := manifest.Marshal()
	if err != nil || !bytes.Equal(canonical, files[ManifestPath]) {
		return VerifiedPackage{}, ErrPackage
	}
	inputs, err := manifest.InputArtifacts()
	if err != nil || len(files) != len(inputs)+3 {
		return VerifiedPackage{}, ErrPackage
	}
	expected := map[string]bool{ManifestPath: true, AssessmentPath: true, HandoffPath: true}
	for _, input := range inputs {
		expected[input.Path] = true
	}
	for path := range files {
		if !expected[path] {
			return VerifiedPackage{}, ErrPackage
		}
	}
	review, err := ParseHandoff(files[HandoffPath])
	if err != nil || review.Scope != options.Scope || len(review.Artifacts) != len(files)-1 {
		return VerifiedPackage{}, ErrPackage
	}
	canonical, err = review.Marshal()
	if err != nil || !bytes.Equal(canonical, files[HandoffPath]) {
		return VerifiedPackage{}, ErrPackage
	}
	for _, artifact := range review.Artifacts {
		data, exists := files[artifact.Path]
		if !exists || hashBytes(data) != artifact.SHA256 {
			return VerifiedPackage{}, ErrPackage
		}
	}
	if review.ManifestSHA256 != hashBytes(files[ManifestPath]) || review.AssessmentSHA256 != hashBytes(files[AssessmentPath]) {
		return VerifiedPackage{}, ErrPackage
	}
	document, _, err := DecodeWorkflow(ctx, files[WorkflowPath])
	if err != nil {
		return VerifiedPackage{}, ErrPackage
	}
	table, err := binding.ParseTable(files[ShapesPath])
	if err != nil {
		return VerifiedPackage{}, ErrPackage
	}
	credentials, err := declaredCredentials(ctx, document, manifest.Sources, table)
	if err != nil || !reflect.DeepEqual(review.Credentials, credentials) {
		return VerifiedPackage{}, ErrPackage
	}
	assessment, err := ParseAssessment(files[AssessmentPath])
	if err != nil {
		return VerifiedPackage{}, ErrPackage
	}
	fresh, err := Assess(ctx, manifest, files, options.RuntimeVerifier)
	if err != nil {
		return VerifiedPackage{}, err
	}
	canonical, err = fresh.Marshal()
	if err != nil || !bytes.Equal(canonical, files[AssessmentPath]) || assessment.InputsSHA256 != review.InputsSHA256 {
		return VerifiedPackage{}, ErrPackage
	}
	artifacts := make([]handoff.DigestFile, 0, len(files))
	for path, data := range files {
		artifacts = append(artifacts, handoff.DigestFile{Path: path, SHA256: hashBytes(data)})
	}
	sum, err := handoff.DigestFiles(options.Scope, trust.PackageDigestVersion, artifacts)
	if err != nil || sum != options.ExpectedSHA256 {
		return VerifiedPackage{}, ErrPackage
	}
	if ctx.Err() != nil {
		return VerifiedPackage{}, ctx.Err()
	}
	return VerifiedPackage{files: files, manifest: manifest, assessment: assessment, handoff: review, sha256: sum}, nil
}
