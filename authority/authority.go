// Package authority owns the public reviewed-package broker handoff.
// It describes value-free authority; the trusted host enforces grants and I/O.
package authority

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	trustdigest "github.com/OpenUdon/openudon/digest"
)

const Version = "openudon.broker-authority.v1"
const TransportVersion = "udon.http-broker.v1"
const MaxOperations = 100

// Authority is a concrete one-run approval, including occurrences derived from
// a recurring grant. Possessing a grant ID alone never authorizes execution.
// PolicySHA256 hashes these standard encoding/json bytes with that field empty.
type Authority struct {
	Version             string      `json:"version"`
	RunID               string      `json:"run_id"`
	OwnerID             string      `json:"owner_id"`
	AgentID             string      `json:"agent_id"`
	GrantID             string      `json:"grant_id"`
	GrantRevisionSHA256 string      `json:"grant_revision_sha256"`
	OccurrenceID        string      `json:"occurrence_id"`
	PackageSHA256       string      `json:"package_sha256"`
	HandoffSHA256       string      `json:"handoff_sha256"`
	InputsSHA256        string      `json:"inputs_sha256"`
	ExecutorSHA256      string      `json:"executor_sha256"`
	PolicySHA256        string      `json:"policy_sha256"`
	ApprovedAt          string      `json:"approved_at"`
	ExpiresAt           string      `json:"expires_at"`
	Operations          []Operation `json:"operations"`
}

type Operation struct {
	StepID            string    `json:"step_id"`
	OperationID       string    `json:"operation_id"`
	InvocationID      string    `json:"invocation_id"`
	Method            string    `json:"method"`
	Origin            string    `json:"origin"`
	ConstraintsSHA256 string    `json:"constraints_sha256"`
	Bindings          []Binding `json:"bindings,omitempty"`
}

// Binding mirrors the published Udon v1 symbolic wire, not credential values.
type Binding struct {
	Name      string `json:"name"`
	Revision  string `json:"revision"`
	Kind      string `json:"kind"`
	In        string `json:"in"`
	Parameter string `json:"parameter"`
}

func Identifier(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return true
}

func (a Authority) Digest() string {
	a.PolicySHA256 = ""
	data, _ := json.Marshal(a)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func (a Authority) Validate() error {
	if a.Version != Version || !Identifier(a.OwnerID) || !Identifier(a.AgentID) || !Identifier(a.GrantID) || !Identifier(a.OccurrenceID) || len(a.RunID) < 16 || len(a.RunID) > 64 {
		return errors.New("invalid broker authority identity")
	}
	for _, c := range a.RunID {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return errors.New("invalid broker run identity")
		}
	}
	for _, digest := range []string{a.GrantRevisionSHA256, a.PackageSHA256, a.HandoffSHA256, a.InputsSHA256, a.ExecutorSHA256, a.PolicySHA256} {
		if !trustdigest.ValidSHA256(digest) || strings.ToLower(digest) != digest {
			return errors.New("invalid broker authority digest")
		}
	}
	if a.PolicySHA256 != a.Digest() {
		return errors.New("broker authority digest mismatch")
	}
	approved, e1 := time.Parse(time.RFC3339, a.ApprovedAt)
	expires, e2 := time.Parse(time.RFC3339, a.ExpiresAt)
	if e1 != nil || e2 != nil || !expires.After(approved) {
		return errors.New("invalid broker authority deadline")
	}
	if len(a.Operations) == 0 || len(a.Operations) > MaxOperations {
		return errors.New("invalid broker operation count")
	}
	steps, operations, invocations := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, op := range a.Operations {
		if !Identifier(op.StepID) || !Identifier(op.OperationID) || !Identifier(op.InvocationID) || steps[op.StepID] || operations[op.OperationID] || invocations[op.InvocationID] || !trustdigest.ValidSHA256(op.ConstraintsSHA256) || strings.ToLower(op.ConstraintsSHA256) != op.ConstraintsSHA256 {
			return errors.New("invalid broker operation inventory")
		}
		steps[op.StepID], operations[op.OperationID], invocations[op.InvocationID] = true, true, true
		switch op.Method {
		case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		default:
			return errors.New("unsupported broker operation method")
		}
		u, err := url.Parse(op.Origin)
		if err != nil || len(op.Origin) > 2048 || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil || u.Path != "" || u.RawPath != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.ContainsAny(op.Origin, "\r\n\x00") {
			return errors.New("invalid reviewed broker origin")
		}
		if len(op.Bindings) > 16 {
			return errors.New("too many broker bindings")
		}
		names, locations := map[string]bool{}, map[string]bool{}
		for _, b := range op.Bindings {
			location := b.In + ":" + strings.ToLower(b.Parameter)
			if b.Validate() != nil || names[b.Name] || locations[location] {
				return errors.New("invalid broker binding inventory")
			}
			names[b.Name], locations[location] = true, true
		}
	}
	return nil
}

func (a Authority) ValidateAt(now time.Time) error {
	if err := a.Validate(); err != nil {
		return err
	}
	approved, _ := time.Parse(time.RFC3339, a.ApprovedAt)
	expires, _ := time.Parse(time.RFC3339, a.ExpiresAt)
	if now.Before(approved) || !now.Before(expires) {
		return errors.New("broker authority is not current")
	}
	return nil
}

func (b Binding) Validate() error {
	if !Identifier(b.Name) || !trustdigest.ValidSHA256(b.Revision) || strings.ToLower(b.Revision) != b.Revision {
		return errors.New("invalid symbolic broker binding")
	}
	if b.Kind == "bearer" && b.In == "header" && b.Parameter == "Authorization" {
		return nil
	}
	if b.Kind != "api_key" || (b.In != "header" && b.In != "query") || !Identifier(b.Parameter) {
		return errors.New("unsupported symbolic broker binding")
	}
	if b.In == "header" {
		if http.CanonicalHeaderKey(b.Parameter) != b.Parameter {
			return errors.New("noncanonical broker header binding")
		}
		switch strings.ToLower(b.Parameter) {
		case "authorization", "proxy-authorization", "cookie", "set-cookie", "connection", "proxy-connection", "transfer-encoding", "upgrade", "trailer", "te", "host", "content-length":
			return errors.New("reserved broker header binding")
		}
	}
	return nil
}
