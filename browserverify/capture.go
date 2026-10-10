package browserverify

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/OpenUdon/evidence/artifact"
	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/browsertransaction"
)

const CaptureImportVersion = "openudon.browser-capture-import.v1"

type CaptureFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// CaptureReceipt preserves the retained byte-bound import wire. A receipt is
// evidence only; it is neither an attestation nor a fresh capture/run approval.
type CaptureReceipt struct {
	Version     string                         `json:"version"`
	StartSHA256 string                         `json:"start_sha256"`
	Transaction browsertransaction.Transaction `json:"transaction"`
	Effect      string                         `json:"effect"`
	Files       []CaptureFile                  `json:"files"`
}
type CaptureVerifyOptions struct {
	ExpectedReceiptSHA256     string
	ExpectedStartSHA256       string            // independently retained exact reviewed start
	ExpectedTransactionSHA256 string            // tagged native transaction identity
	Files                     map[string][]byte // exact bounded public source/review bytes
	At                        time.Time         // optional historical inspection; current use requires host time
}

func DecodeCaptureReceipt(data []byte) (CaptureReceipt, error) {
	var r CaptureReceipt
	if browsercontract.Decode(data, &r) != nil || r.Version != CaptureImportVersion || r.Effect != "write" || len(r.StartSHA256) != 64 || !sha256Plain(r.StartSHA256) || r.Transaction.Validate() != nil || r.Transaction.State != browsertransaction.StateReviewed || len(r.Files) == 0 || len(r.Files) > 128 {
		return CaptureReceipt{}, browsercontract.ErrContract
	}
	seen := map[string]bool{}
	for _, f := range r.Files {
		clean, err := artifact.CleanRelativePath(f.Path, artifact.Options{})
		if err != nil || clean != f.Path || len(f.Path) > 256 || !sha256Plain(f.SHA256) || seen[f.Path] {
			return CaptureReceipt{}, browsercontract.ErrContract
		}
		seen[f.Path] = true
	}
	return r, nil
}
func sha256Plain(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'f' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// VerifyCaptureReceipt independently binds original receipt/start/transaction
// identities and every declared source/review byte. It does not infer that a
// browser ran or promote claimed capture provenance into host authority.
func VerifyCaptureReceipt(data []byte, o CaptureVerifyOptions) (CaptureReceipt, error) {
	if len(o.Files) > 512 {
		return CaptureReceipt{}, browsercontract.ErrContract
	}
	r, err := DecodeCaptureReceipt(data)
	if err != nil || !sha256Plain(o.ExpectedReceiptSHA256) || !sha256Plain(o.ExpectedStartSHA256) || browsercontract.SHA256(data) != o.ExpectedReceiptSHA256 || r.StartSHA256 != o.ExpectedStartSHA256 {
		return CaptureReceipt{}, browsercontract.ErrContract
	}
	transaction, err := browsertransaction.Digest(r.Transaction)
	if err != nil || transaction != o.ExpectedTransactionSHA256 {
		return CaptureReceipt{}, browsercontract.ErrContract
	}
	if !o.At.IsZero() {
		observed, _ := time.Parse(time.RFC3339Nano, r.Transaction.Provenance.ObservedAt)
		expires, _ := time.Parse(time.RFC3339Nano, r.Transaction.Provenance.ExpiresAt)
		if o.At.Before(observed) || !o.At.Before(expires) {
			return CaptureReceipt{}, browsercontract.ErrContract
		}
	}
	hashes := map[string]bool{}
	total := 0
	for _, f := range r.Files {
		raw, ok := o.Files[f.Path]
		if !ok || len(raw) == 0 || len(raw) > 8<<20 || len(raw) > (32<<20)-total || browsercontract.SHA256(raw) != f.SHA256 {
			return CaptureReceipt{}, browsercontract.ErrContract
		}
		total += len(raw)
		hashes["sha256:"+f.SHA256] = true
	}
	for _, candidate := range r.Transaction.Candidates {
		if !hashes[candidate.SourceSHA256] || !hashes[candidate.ReviewSHA256] {
			return CaptureReceipt{}, browsercontract.ErrContract
		}
	}
	// Return a private copied wire snapshot, never caller-owned maps or raw values.
	raw, err := json.Marshal(r)
	if err != nil {
		return CaptureReceipt{}, browsercontract.ErrContract
	}
	var copy CaptureReceipt
	if browsercontract.Decode(raw, &copy) != nil {
		return CaptureReceipt{}, browsercontract.ErrContract
	}
	return copy, nil
}

// CaptureArtifactsMatch checks only package reference correspondence. The host
// must use VerifyCaptureReceipt with independently retained start/receipt IDs
// before adopting a capture; package bytes cannot attest their own provenance.
func CaptureArtifactsMatch(r CaptureReceipt, files map[string][]byte) error {
	if len(files) > 512 {
		return browsercontract.ErrContract
	}
	hashes := map[string]bool{}
	total := 0
	for _, raw := range files {
		if len(raw) == 0 || len(raw) > 8<<20 || len(raw) > (32<<20)-total {
			return browsercontract.ErrContract
		}
		total += len(raw)
		hashes[browsercontract.SHA256(raw)] = true
	}
	for _, f := range r.Files {
		if !hashes[f.SHA256] {
			return browsercontract.ErrContract
		}
	}
	for _, candidate := range r.Transaction.Candidates {
		for _, h := range []string{candidate.SourceSHA256, candidate.ReviewSHA256} {
			if !hashes[strings.TrimPrefix(h, "sha256:")] {
				return browsercontract.ErrContract
			}
		}
	}
	return nil
}
