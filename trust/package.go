// Package trust verifies caller-supplied immutable review-package snapshots.
// It performs no discovery, synthesis, assessment, simulation, approval or I/O.
package trust

import (
	"context"
	"errors"
	"strings"

	"github.com/OpenUdon/evidence/artifact"
	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/wire"
)

const MaxFiles = 1024
const MaxFileBytes = 8 << 20
const MaxTotalBytes = 64 << 20
const PackageDigestVersion = "openudon.handoff-package-digest.v1"

var ErrInvalidPackage = errors.New("invalid review package snapshot")

// Request is an explicit snapshot of the required manifest inventory, including
// the exact manifest bytes at ManifestPath. The host supplies RequiredPaths
// from its format-specific inventory policy. It must keep all bytes stable
// during inspection, and own regular-file/symlink and isolation checks.
type Request struct {
	Scope         string
	ManifestPath  string
	Files         map[string][]byte
	RequiredPaths []string
}

// Inspection binds exact bytes. It proves inventory/digests and neutral policy,
// not source semantics, assessment success, grants or executable capability.
type Inspection struct {
	Scope         string                `json:"scope"`
	PackageSHA256 string                `json:"package_sha256"`
	HandoffSHA256 string                `json:"handoff_sha256"`
	Manifest      handoff.ReviewHandoff `json:"manifest"`
}

func Inspect(ctx context.Context, request Request) (Inspection, error) {
	if ctx == nil {
		return Inspection{}, ErrInvalidPackage
	}
	if err := ctx.Err(); err != nil {
		return Inspection{}, err
	}
	if !canonicalPath(request.Scope) || !canonicalPath(request.ManifestPath) || len(request.Scope) > 2048 || len(request.Files) == 0 || len(request.Files) > MaxFiles || len(request.RequiredPaths) > MaxFiles {
		return Inspection{}, ErrInvalidPackage
	}
	total := 0
	for path, data := range request.Files {
		if err := ctx.Err(); err != nil {
			return Inspection{}, err
		}
		if !canonicalPath(path) || len(data) > MaxFileBytes {
			return Inspection{}, ErrInvalidPackage
		}
		total += len(data)
		if total > MaxTotalBytes {
			return Inspection{}, ErrInvalidPackage
		}
	}
	manifestBytes, ok := request.Files[request.ManifestPath]
	if !ok {
		return Inspection{}, ErrInvalidPackage
	}
	var manifest handoff.ReviewHandoff
	if wire.DecodeStrict(manifestBytes, &manifest) != nil || len(manifest.HandoffInputs) > MaxFiles || len(handoff.ValidateReviewHandoff(manifest)) != 0 {
		return Inspection{}, ErrInvalidPackage
	}
	required := map[string]bool{}
	var files []handoff.DigestFile
	for _, input := range manifest.HandoffInputs {
		if err := ctx.Err(); err != nil {
			return Inspection{}, err
		}
		if !input.Required {
			continue
		}
		if !canonicalPath(input.Path) {
			return Inspection{}, ErrInvalidPackage
		}
		data, ok := request.Files[input.Path]
		if !ok {
			return Inspection{}, ErrInvalidPackage
		}
		required[input.Path] = true
		actual := digest.SHA256(data)
		declared := actual
		if input.Path == request.ManifestPath {
			var err error
			declared, err = handoff.ReviewHandoffSelfDigest(manifest, request.ManifestPath)
			if err != nil {
				return Inspection{}, ErrInvalidPackage
			}
		}
		if declared != strings.ToLower(strings.TrimSpace(input.SHA256)) {
			return Inspection{}, ErrInvalidPackage
		}
		files = append(files, handoff.DigestFile{Path: input.Path, SHA256: actual})
	}
	if !required[request.ManifestPath] || len(required) != len(request.Files) {
		return Inspection{}, ErrInvalidPackage
	}
	for _, path := range request.RequiredPaths {
		if !canonicalPath(path) || !required[path] {
			return Inspection{}, ErrInvalidPackage
		}
	}
	packageDigest, err := handoff.DigestFiles(request.Scope, PackageDigestVersion, files)
	if err != nil {
		return Inspection{}, ErrInvalidPackage
	}
	if err := ctx.Err(); err != nil {
		return Inspection{}, err
	}
	return Inspection{Scope: request.Scope, PackageSHA256: packageDigest, HandoffSHA256: digest.SHA256(manifestBytes), Manifest: manifest}, nil
}

func canonicalPath(path string) bool {
	clean, err := artifact.CleanRelativePath(path, artifact.Options{})
	return err == nil && clean == path && len(path) <= 2048
}
