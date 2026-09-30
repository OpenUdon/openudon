// Report-v5 wire validation is independent of the private executor.
// Contract frozen from Udon M44 1a5e9aa2045e3d875da2e18aab2d6db869ac5223.
package udonreport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"reflect"
	"regexp"
	"strings"
	"time"
)

const VersionV5 = "udon.execution-report.v5"
const MaxSteps = 256

var identifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)
var digest = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)
var timestampV5 = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?(Z|[+-]([01][0-9]|2[0-3]):[0-5][0-9])$`)

// Step identifies exactly one invocation in a supported straight-line plan.
// Outcomes describe the leaf operation, not downstream expression evaluation.
type StepV5 struct {
	StepID       string `json:"step_id"`
	OperationID  string `json:"operation_id"`
	InvocationID string `json:"invocation_id"`
	Outcome      string `json:"outcome"`
	StartedAt    string `json:"started_at,omitempty"`
	FinishedAt   string `json:"finished_at,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
}

type ReportV5 struct {
	Version           string   `json:"version"`
	RunID             string   `json:"run_id"`
	WorkflowID        string   `json:"workflow_id"`
	WorkflowDigest    string   `json:"workflow_digest"`
	InventoryComplete bool     `json:"inventory_complete"`
	Status            string   `json:"status"`
	StartedAt         string   `json:"started_at"`
	FinishedAt        string   `json:"finished_at,omitempty"`
	ErrorCode         string   `json:"error_code,omitempty"`
	Steps             []StepV5 `json:"steps"`
}

func ValidIdentifier(s string) bool { return identifier.MatchString(s) }

// Validate checks shape, identity uniqueness, causal ordering and timestamp
// consistency. Consumers must additionally compare the inventory and identities
// with their approved plan and exact attempt. A valid stale report proves nothing.
func (r ReportV5) Validate() error {
	if r.Version != VersionV5 || !ValidIdentifier(r.RunID) || !ValidIdentifier(r.WorkflowID) || !digest.MatchString(r.WorkflowDigest) || !r.InventoryComplete || len(r.Steps) == 0 || len(r.Steps) > MaxSteps {
		return fmt.Errorf("invalid report identity or inventory")
	}
	start, err := time.Parse(time.RFC3339Nano, r.StartedAt)
	if err != nil || !timestampV5.MatchString(r.StartedAt) {
		return fmt.Errorf("invalid report start")
	}
	var finish time.Time
	switch r.Status {
	case "incomplete":
		if r.FinishedAt != "" || r.ErrorCode != "" {
			return fmt.Errorf("incomplete report has terminal fields")
		}
	case "success", "error":
		finish, err = time.Parse(time.RFC3339Nano, r.FinishedAt)
		if err != nil || !timestampV5.MatchString(r.FinishedAt) || finish.Before(start) {
			return fmt.Errorf("invalid report finish")
		}
		if r.Status == "success" && r.ErrorCode != "" {
			return fmt.Errorf("successful report has error code")
		}
		if r.Status == "error" && r.ErrorCode != "execution_failed" && r.ErrorCode != "checkpoint_failed" && r.ErrorCode != "cancelled" {
			return fmt.Errorf("invalid report error code")
		}
	default:
		return fmt.Errorf("invalid report status")
	}
	steps, operations := map[string]bool{}, map[string]bool{}
	previous := start
	stopped := false
	for _, s := range r.Steps {
		if !ValidIdentifier(s.StepID) || !ValidIdentifier(s.OperationID) || s.InvocationID != s.StepID || steps[s.StepID] || operations[s.OperationID] {
			return fmt.Errorf("invalid or repeated invocation")
		}
		steps[s.StepID], operations[s.OperationID] = true, true
		if s.Outcome == "not_started" {
			if s.StartedAt != "" || s.FinishedAt != "" || s.ErrorCode != "" {
				return fmt.Errorf("unstarted invocation has execution fields")
			}
			stopped = true
		} else {
			if stopped {
				return fmt.Errorf("invocation started after sequence stopped")
			}
			t, e := time.Parse(time.RFC3339Nano, s.StartedAt)
			if e != nil || !timestampV5.MatchString(s.StartedAt) || t.Before(previous) || (!finish.IsZero() && t.After(finish)) {
				return fmt.Errorf("invalid invocation start")
			}
			previous = t
			switch s.Outcome {
			case "unknown":
				if s.FinishedAt != "" || s.ErrorCode != "" {
					return fmt.Errorf("unknown invocation has terminal fields")
				}
				stopped = true
			case "succeeded", "failed":
				end, e := time.Parse(time.RFC3339Nano, s.FinishedAt)
				if e != nil || !timestampV5.MatchString(s.FinishedAt) || end.Before(t) || (!finish.IsZero() && end.After(finish)) {
					return fmt.Errorf("invalid invocation finish")
				}
				previous = end
				if s.Outcome == "succeeded" && s.ErrorCode != "" {
					return fmt.Errorf("successful invocation has error code")
				}
				if s.Outcome == "failed" {
					if s.ErrorCode != "operation_failed" {
						return fmt.Errorf("invalid invocation error code")
					}
					stopped = true
				}
			default:
				return fmt.Errorf("invalid invocation outcome")
			}
		}
		if r.Status == "success" && s.Outcome != "succeeded" {
			return fmt.Errorf("successful report has unfinished invocation")
		}
	}
	return nil
}

// DecodeV5 enforces bounded strict JSON before semantic validation.
func DecodeV5(data []byte) (*ReportV5, error) {
	if len(data) > 256<<10 {
		return nil, fmt.Errorf("report v5 exceeds limit")
	}
	var r ReportV5
	if err := evidencefile.DecodeStrict(data, &r); err != nil {
		return nil, fmt.Errorf("invalid report v5 JSON")
	}
	// encoding/json accepts case aliases and null for strings; the v5 wire
	// schema permits only canonical property names and non-null typed fields.
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, fmt.Errorf("invalid report v5 JSON")
	}
	allowed := map[string]bool{}
	typ := reflect.TypeOf(ReportV5{})
	for n := 0; n < typ.NumField(); n++ {
		allowed[strings.Split(typ.Field(n).Tag.Get("json"), ",")[0]] = true
	}
	for k, v := range object {
		if !allowed[k] || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
			return nil, fmt.Errorf("invalid report v5 field")
		}
	}
	var steps []map[string]json.RawMessage
	if err := json.Unmarshal(object["steps"], &steps); err != nil {
		return nil, fmt.Errorf("invalid v5 steps")
	}
	allowed = map[string]bool{}
	typ = reflect.TypeOf(StepV5{})
	for n := 0; n < typ.NumField(); n++ {
		allowed[strings.Split(typ.Field(n).Tag.Get("json"), ",")[0]] = true
	}
	if (r.Status == "incomplete" && (object["finished_at"] != nil || object["error_code"] != nil)) || (r.Status == "success" && object["error_code"] != nil) {
		return nil, fmt.Errorf("unexpected v5 terminal field")
	}
	for n, s := range steps {
		outcome := r.Steps[n].Outcome
		if (outcome == "not_started" && (s["started_at"] != nil || s["finished_at"] != nil || s["error_code"] != nil)) || (outcome == "unknown" && (s["finished_at"] != nil || s["error_code"] != nil)) || (outcome == "succeeded" && s["error_code"] != nil) {
			return nil, fmt.Errorf("unexpected v5 invocation field")
		}
		for k, v := range s {
			if !allowed[k] || bytes.Equal(bytes.TrimSpace(v), []byte("null")) {
				return nil, fmt.Errorf("invalid v5 step field")
			}
		}
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}
