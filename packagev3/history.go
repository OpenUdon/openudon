package packagev3

import (
	"context"
	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/trust"
)

type HistoryRequest struct {
	FormatVersion       string
	Scope               string
	ExpectedSHA256      string
	Files               map[string][]byte
	LegacyManifestPath  string
	LegacyRequiredPaths []string
	Private             func([]byte) bool
}
type HistoryInspection struct {
	FormatVersion string `json:"format_version"`
	Scope         string `json:"scope"`
	PackageSHA256 string `json:"package_sha256"`
	HandoffSHA256 string `json:"handoff_sha256"`
	ReadOnly      bool   `json:"read_only"`
	SourceProof   bool   `json:"source_proof"`
}

// InspectHistory explicitly dispatches a selected format's byte/digest reader.
// It never calls runtime/source verifiers, constructs a successor or derives
// authority. V2 and unsupported-execution v3 history remain readable; source
// proof is always false and all outputs are read-only observations.
func InspectHistory(ctx context.Context, request HistoryRequest) (HistoryInspection, error) {
	if ctx == nil || !digestValid(request.ExpectedSHA256) {
		return HistoryInspection{}, ErrPackage
	}
	switch request.FormatVersion {
	case PackageVersion:
		if request.LegacyManifestPath != "" || len(request.LegacyRequiredPaths) != 0 {
			return HistoryInspection{}, ErrPackage
		}
		s, err := inspectSnapshot(ctx, VerifyOptions{Scope: request.Scope, ExpectedSHA256: request.ExpectedSHA256, Files: request.Files, Private: request.Private})
		if err != nil {
			return HistoryInspection{}, err
		}
		return HistoryInspection{FormatVersion: PackageVersion, Scope: s.manifest.Scope, PackageSHA256: s.sha256, HandoffSHA256: digest.SHA256(s.files[HandoffPath]), ReadOnly: true, SourceProof: false}, nil
	case handoff.ReviewHandoffVersion:
		if request.LegacyManifestPath == "" || len(request.LegacyRequiredPaths) == 0 {
			return HistoryInspection{}, ErrPackage
		}
		// Neutral legacy trust policy and bounds remain unchanged. Private filtering
		// is caller policy; contents are never echoed in a reader error or response.
		files := map[string][]byte{}
		total := 0
		for path, data := range request.Files {
			if ctx.Err() != nil {
				return HistoryInspection{}, ctx.Err()
			}
			if len(data) > trust.MaxFileBytes || len(data) > trust.MaxTotalBytes-total || len(files) >= trust.MaxFiles {
				return HistoryInspection{}, ErrPackage
			}
			if request.Private != nil && request.Private(data) {
				return HistoryInspection{}, ErrPackage
			}
			files[path] = append([]byte(nil), data...)
			total += len(data)
		}
		result, err := trust.Inspect(ctx, trust.Request{Scope: request.Scope, ManifestPath: request.LegacyManifestPath, Files: files, RequiredPaths: request.LegacyRequiredPaths})
		if err != nil {
			if ctx.Err() != nil {
				return HistoryInspection{}, ctx.Err()
			}
			return HistoryInspection{}, ErrPackage
		}
		if result.Manifest.Version != request.FormatVersion || result.PackageSHA256 != request.ExpectedSHA256 {
			return HistoryInspection{}, ErrPackage
		}
		return HistoryInspection{FormatVersion: result.Manifest.Version, Scope: result.Scope, PackageSHA256: result.PackageSHA256, HandoffSHA256: result.HandoffSHA256, ReadOnly: true, SourceProof: false}, nil
	}
	return HistoryInspection{}, ErrPackage
}
