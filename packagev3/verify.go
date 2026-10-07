package packagev3

import (
	"bytes"
	"context"
	"github.com/OpenUdon/uws/binding"
	"reflect"
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
	snapshot, err := inspectSnapshot(ctx, options)
	if err != nil {
		return VerifiedPackage{}, err
	}
	files, manifest, review, assessment := snapshot.files, snapshot.manifest, snapshot.handoff, snapshot.assessment
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
	fresh, err := Assess(ctx, manifest, files, options.RuntimeVerifier)
	if err != nil {
		return VerifiedPackage{}, err
	}
	canonical, err := fresh.Marshal()
	if err != nil || !bytes.Equal(canonical, files[AssessmentPath]) {
		return VerifiedPackage{}, ErrPackage
	}
	if ctx.Err() != nil {
		return VerifiedPackage{}, ctx.Err()
	}
	return VerifiedPackage{files: files, manifest: manifest, assessment: assessment, handoff: review, sha256: snapshot.sha256}, nil
}
