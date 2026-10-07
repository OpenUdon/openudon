package runevidence

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"github.com/OpenUdon/evidence/artifact"
	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/udonreport"
	"github.com/OpenUdon/openudon/wire"
)

var ErrInvalidEvidence = errors.New("invalid run evidence snapshot")

// Identity is independent host-held provenance. A valid stale record alone
// cannot establish that it belongs to the host's approved exact attempt.
type Identity struct {
	RunID           string
	PackageSHA256   string
	HandoffSHA256   string
	ApprovalSHA256  string
	RunConfigSHA256 string
}

type Request struct {
	Evidence            []byte
	Artifacts           map[string][]byte // exactly the report/async files referenced by Evidence
	Expected            *Identity
	ExpectedInventory   *udonreport.InventoryV5
	Signature           []byte
	TrustedPublicKeyPEM []byte
	RequireSignature    bool
}

type Result struct {
	Evidence          RunEvidence
	SHA256            string
	SignatureVerified bool
	SignerTrusted     bool
	LegacyReadOnly    bool
}

// Verify checks exact supplied evidence, reports, async references and optional
// signature. It reads no paths from the evidence and supplies no run authority.
// Browser evidence requires the separately pinned legacy CLI verifier.
func Verify(ctx context.Context, request Request) (Result, error) {
	if ctx == nil {
		return Result{}, ErrInvalidEvidence
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if len(request.Evidence) > wire.MaxBytes || len(request.Artifacts) > 1024 || len(request.Signature) > 1<<20 || len(request.TrustedPublicKeyPEM) > 64<<10 {
		return Result{}, ErrInvalidEvidence
	}
	total := len(request.Evidence)
	for path, data := range request.Artifacts {
		if !safePath(path) || len(data) > wire.MaxBytes {
			return Result{}, ErrInvalidEvidence
		}
		total += len(data)
		if total > 64<<20 {
			return Result{}, ErrInvalidEvidence
		}
	}
	var e RunEvidence
	if wire.DecodeStrict(request.Evidence, &e) != nil {
		return Result{}, ErrInvalidEvidence
	}
	if err := Validate(e); err != nil {
		if errors.Is(err, ErrUnsupportedBrowser) {
			return Result{}, err
		}
		return Result{}, ErrInvalidEvidence
	}
	if e.Version == BrokerVersion && validateBrokerWire(request.Evidence) != nil {
		return Result{}, ErrInvalidEvidence
	}
	if request.Expected != nil && *request.Expected != (Identity{e.RunID, e.PackageSHA256, e.HandoffSHA256, e.ApprovalSHA256, e.RunConfigSHA256}) {
		return Result{}, ErrInvalidEvidence
	}
	if request.ExpectedInventory != nil && (request.ExpectedInventory.Validate() != nil || e.StepExecution == nil || !reflect.DeepEqual(*request.ExpectedInventory, e.StepExecution.Inventory())) {
		return Result{}, ErrInvalidEvidence
	}
	legacy := e.Version == LegacyVersion
	if legacy && (len(request.Signature) != 0 || request.RequireSignature || len(request.TrustedPublicKeyPEM) != 0) {
		return Result{}, ErrInvalidEvidence
	}
	if !legacy && (len(request.Signature) != 0 || request.RequireSignature || len(request.TrustedPublicKeyPEM) != 0) {
		if VerifySignature(request.Evidence, request.Signature, request.TrustedPublicKeyPEM) != nil {
			return Result{}, ErrInvalidEvidence
		}
	}
	seen := map[string]bool{}
	for _, ref := range e.AsyncEvidenceFiles {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		if !safePath(ref.Path) || seen[ref.Path] || ref.Records < 1 || ref.Records > wire.MaxNodes || ref.Purpose != "openudon_run_async_execution_forwarding" {
			return Result{}, ErrInvalidEvidence
		}
		seen[ref.Path] = true
		data, ok := request.Artifacts[ref.Path]
		if !ok || "sha256:"+digest.SHA256(data) != ref.Digest {
			return Result{}, ErrInvalidEvidence
		}
		var bundle AsyncEvidenceBundle
		if wire.DecodeStrict(data, &bundle) != nil || len(bundle.Records) != ref.Records || validateAsyncEvidenceBundle(bundle) != nil {
			return Result{}, ErrInvalidEvidence
		}
	}
	if legacy {
		// The legacy v1 profile cannot bind an executor report to an exact attempt.
		if e.Executor.ReportPath != "" || e.Executor.ReportSHA256 != "" || e.Executor.ReportSize != 0 {
			return Result{}, ErrInvalidEvidence
		}
	} else {
		success := !e.DryRun && evidenceGateStatus(e, "executor_invocation") == "pass"
		if e.Executor.ReportPath == "" {
			if e.Executor.ReportSHA256 != "" || e.Executor.ReportSize != 0 || success || (e.StepExecution != nil && e.StepExecution.State == "validated") {
				return Result{}, ErrInvalidEvidence
			}
		} else {
			path := e.Executor.ReportPath
			data, ok := request.Artifacts[path]
			if !safePath(path) || seen[path] || !ok || int64(len(data)) != e.Executor.ReportSize || digest.SHA256(data) != e.Executor.ReportSHA256 {
				return Result{}, ErrInvalidEvidence
			}
			seen[path] = true
			if e.StepExecution != nil {
				observed := udonreport.ObserveV5(e.StepExecution.Inventory(), data)
				if e.StepExecution.State != "validated" || !reflect.DeepEqual(observed, *e.StepExecution) || success && observed.ReportStatus != "success" {
					return Result{}, ErrInvalidEvidence
				}
			} else {
				report, err := udonreport.Decode(data)
				if err != nil || success && report.Status != "success" {
					return Result{}, ErrInvalidEvidence
				}
			}
		}
	}
	if len(seen) != len(request.Artifacts) {
		return Result{}, ErrInvalidEvidence
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	return Result{Evidence: e, SHA256: digest.SHA256(request.Evidence), SignatureVerified: len(request.Signature) != 0, SignerTrusted: len(request.TrustedPublicKeyPEM) != 0, LegacyReadOnly: legacy}, nil
}

func safePath(path string) bool {
	clean, err := artifact.CleanRelativePath(path, artifact.Options{})
	return err == nil && clean == path && len(path) <= 2048 && strings.TrimSpace(path) == path
}

func evidenceGateStatus(e RunEvidence, name string) string {
	for _, gate := range e.Gates {
		if gate.Name == name {
			return gate.Status
		}
	}
	return ""
}
