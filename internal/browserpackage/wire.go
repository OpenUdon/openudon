// Package browserpackage exposes reviewed capture adoption over OpenUdon's
// neutral authoring implementation. It never starts a browser or executor.
package browserpackage

import (
	"encoding/json"
	"errors"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"

	engine "github.com/OpenUdon/openudon/internal/authoringengine"
	"github.com/OpenUdon/openudon/internal/browsercapture"
	"github.com/OpenUdon/openudon/internal/elicitor"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const (
	Version         = "openudon.browser-author.v1"
	MaxRequestBytes = 256 << 10
	MaxReportBytes  = 2 << 20
)

var identifier = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

// Request binds a public imported receipt to its exact approved start. Flow
// and Action are native operation names, never newly authored browser recipes.
// Input values are symbolic references only; this wire accepts no credentials.
type Request struct {
	Version               string            `json:"version"`
	Kind                  string            `json:"kind"`
	RequestID             string            `json:"request_id"`
	Start                 json.RawMessage   `json:"start"`
	ReceiptPath           string            `json:"receipt_path"`
	ReceiptSHA256         string            `json:"receipt_sha256"`
	TransactionSHA256     string            `json:"transaction_sha256"`
	InputSHA256           string            `json:"input_sha256,omitempty"`
	ExpectedTOTP          *bool             `json:"expected_totp"`
	RegistrationAuthority string            `json:"registration_authority,omitempty"`
	WorkflowName          string            `json:"workflow_name"`
	AllowOverwrite        bool              `json:"allow_overwrite,omitempty"`
	Flow                  string            `json:"flow,omitempty"`
	Action                string            `json:"action,omitempty"`
	CleanupDisposition    string            `json:"cleanup_disposition,omitempty"`
	Inputs                []Input           `json:"inputs,omitempty"`
	InputBindings         map[string]string `json:"input_bindings,omitempty"`
}

type Input struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

// Operation is a native candidate operation. Inputs list symbolic parameter
// names, not selectors, URLs, values, or reconstructed runtime commands.
type Operation struct {
	CandidateID string   `json:"candidate_id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Inputs      []string `json:"inputs,omitempty"`
}

// Plan is an immutable read-only proposal. Its digest includes the current
// package inventory and exact authored preview, with PlanSHA256 empty.
type Plan struct {
	Version           string                             `json:"version"`
	Kind              string                             `json:"kind"`
	RequestID         string                             `json:"request_id"`
	RequestSHA256     string                             `json:"request_sha256"`
	InputSHA256       string                             `json:"input_sha256"`
	ReceiptSHA256     string                             `json:"receipt_sha256"`
	TransactionSHA256 string                             `json:"transaction_sha256"`
	Candidates        []elicitor.VirtualBrowserCandidate `json:"candidates"`
	Operations        []Operation                        `json:"operations"`
	Ready             bool                               `json:"ready"`
	Blockers          []string                           `json:"blockers"`
	Preview           *engine.Preview                    `json:"preview,omitempty"`
	WriteConflicts    []engine.WriteConflict             `json:"write_conflicts"`
	PlanSHA256        string                             `json:"plan_sha256"`
}

// Result reports authoring, not promotion or execution. build_failed means
// reviewed authoring files were committed but native package build failed;
// losing output requires inspection, never blindly repeating apply.
type Result struct {
	Version       string   `json:"version"`
	Kind          string   `json:"kind"`
	RequestID     string   `json:"request_id"`
	RequestSHA256 string   `json:"request_sha256"`
	PlanSHA256    string   `json:"plan_sha256"`
	Outcome       string   `json:"outcome"`
	Written       []string `json:"written"`
	QualityStatus string   `json:"quality_status"`
}

func DecodeRequest(data []byte) (Request, error) {
	var r Request
	invalid := errors.New("browser author request invalid")
	if len(data) == 0 || len(data) > MaxRequestBytes || !utf8.Valid(data) || evidencefile.DecodeStrict(data, &r) != nil {
		return Request{}, invalid
	}
	if r.Version != Version || r.Kind != "request" || !identifier.MatchString(r.RequestID) || !identifier.MatchString(r.WorkflowName) || r.ExpectedTOTP == nil {
		return Request{}, invalid
	}
	if !safeRelative(r.ReceiptPath) || !strings.HasPrefix(r.ReceiptPath, "expected/browser-capture/") || !strings.HasSuffix(r.ReceiptPath, ".json") || !evidencefile.ValidSHA256(r.ReceiptSHA256) || !validDigest(r.TransactionSHA256) || r.InputSHA256 != "" && !validDigest(r.InputSHA256) {
		return Request{}, invalid
	}
	start, err := browsercapture.DecodeStart(r.Start)
	if err != nil {
		return Request{}, invalid
	}
	if start.Mode == browsercapture.Authenticated {
		if r.RegistrationAuthority != "" || r.CleanupDisposition != "" {
			return Request{}, invalid
		}
	} else if start.Mode == browsercapture.Registration {
		if *r.ExpectedTOTP || !identifier.MatchString(r.RegistrationAuthority) || r.Action != "" || r.CleanupDisposition != "delete_separately" && r.CleanupDisposition != "retain_dedicated_test_identity" {
			return Request{}, invalid
		}
	} else {
		return Request{}, invalid
	}
	for _, name := range []string{r.Flow, r.Action} {
		if name != "" && !identifier.MatchString(name) {
			return Request{}, invalid
		}
	}
	if len(r.Inputs) > 32 || len(r.InputBindings) > 32 {
		return Request{}, invalid
	}
	names := map[string]bool{}
	for _, input := range r.Inputs {
		if !identifier.MatchString(input.Name) || names[input.Name] {
			return Request{}, invalid
		}
		switch input.Type {
		case "string", "number", "boolean", "object", "array":
		default:
			return Request{}, invalid
		}
		names[input.Name] = true
	}
	for name, reference := range r.InputBindings {
		if !identifier.MatchString(name) || !strings.HasPrefix(reference, "inputs.") || !names[strings.TrimPrefix(reference, "inputs.")] {
			return Request{}, invalid
		}
	}
	return r, nil
}

func Digest(data []byte) string { return "sha256:" + evidencefile.SHA256(data) }

func SealPlan(plan Plan) (Plan, error) {
	plan.PlanSHA256 = ""
	data, err := json.Marshal(plan)
	if err != nil || len(data) > MaxReportBytes {
		return Plan{}, errors.New("browser author plan invalid")
	}
	plan.PlanSHA256 = Digest(data)
	return plan, nil
}

func validDigest(value string) bool {
	return strings.HasPrefix(value, "sha256:") && evidencefile.ValidSHA256(strings.TrimPrefix(value, "sha256:"))
}

func safeRelative(value string) bool {
	return value != "" && value != "." && value != ".." && !strings.HasPrefix(value, "../") && !strings.HasPrefix(value, "/") && !strings.ContainsAny(value, "\\\x00\r\n") && path.Clean(value) == value
}
