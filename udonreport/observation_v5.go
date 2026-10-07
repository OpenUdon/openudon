package udonreport

import (
	"fmt"
	"reflect"
)

// InventoryV5 is derived independently from the reviewed staged workflow.
type InventoryV5 struct {
	RunID          string   `json:"run_id"`
	WorkflowID     string   `json:"workflow_id"`
	WorkflowDigest string   `json:"workflow_digest"`
	Steps          []StepV5 `json:"steps"`
}

// ObservationV5 contains only validated outcomes or conservative uncertainty.
type ObservationV5 struct {
	ReportVersion     string   `json:"report_version"`
	RunID             string   `json:"run_id"`
	WorkflowID        string   `json:"workflow_id"`
	WorkflowDigest    string   `json:"workflow_digest"`
	State             string   `json:"state"`
	InventoryComplete bool     `json:"inventory_complete"`
	ReportStatus      string   `json:"report_status,omitempty"`
	StartedAt         string   `json:"started_at,omitempty"`
	FinishedAt        string   `json:"finished_at,omitempty"`
	ErrorCode         string   `json:"error_code,omitempty"`
	Steps             []StepV5 `json:"steps"`
}

func (i InventoryV5) Validate() error {
	r := ReportV5{Version: VersionV5, RunID: i.RunID, WorkflowID: i.WorkflowID,
		WorkflowDigest: i.WorkflowDigest, InventoryComplete: true, Status: "incomplete",
		StartedAt: "2000-01-01T00:00:00Z", Steps: i.Steps}
	for _, s := range i.Steps {
		if s.Outcome != "not_started" {
			return fmt.Errorf("expected inventory has observed outcomes")
		}
	}
	return r.Validate()
}

func UnknownV5(i InventoryV5, state string) ObservationV5 {
	o := ObservationV5{ReportVersion: VersionV5, RunID: i.RunID, WorkflowID: i.WorkflowID,
		WorkflowDigest: i.WorkflowDigest, State: state, Steps: append([]StepV5(nil), i.Steps...)}
	for n := range o.Steps {
		o.Steps[n].Outcome = "unknown"
	}
	return o
}

// ObserveV5 rejects a report without retaining any of its arbitrary bytes.
func ObserveV5(i InventoryV5, data []byte) ObservationV5 {
	if i.Validate() != nil {
		return UnknownV5(i, "invalid")
	}
	r, err := DecodeV5(data)
	if err != nil {
		return UnknownV5(i, "invalid")
	}
	if r.RunID != i.RunID || r.WorkflowID != i.WorkflowID || r.WorkflowDigest != i.WorkflowDigest || len(r.Steps) != len(i.Steps) {
		return UnknownV5(i, "mismatched")
	}
	for n, s := range r.Steps {
		e := i.Steps[n]
		if s.StepID != e.StepID || s.OperationID != e.OperationID || s.InvocationID != e.InvocationID {
			return UnknownV5(i, "mismatched")
		}
	}
	return ObservationV5{ReportVersion: VersionV5, RunID: i.RunID, WorkflowID: i.WorkflowID,
		WorkflowDigest: i.WorkflowDigest, State: "validated", InventoryComplete: true,
		ReportStatus: r.Status, StartedAt: r.StartedAt, FinishedAt: r.FinishedAt, ErrorCode: r.ErrorCode, Steps: r.Steps}
}

func (o ObservationV5) Inventory() InventoryV5 {
	i := InventoryV5{RunID: o.RunID, WorkflowID: o.WorkflowID, WorkflowDigest: o.WorkflowDigest, Steps: make([]StepV5, len(o.Steps))}
	for n, s := range o.Steps {
		i.Steps[n] = StepV5{StepID: s.StepID, OperationID: s.OperationID, InvocationID: s.InvocationID, Outcome: "not_started"}
	}
	return i
}

func (o ObservationV5) Validate() error {
	if o.ReportVersion != VersionV5 {
		return fmt.Errorf("invalid observation version")
	}
	i := o.Inventory()
	if err := i.Validate(); err != nil {
		return err
	}
	if o.State == "validated" {
		if !o.InventoryComplete {
			return fmt.Errorf("validated observation needs complete inventory")
		}
		r := ReportV5{Version: VersionV5, RunID: o.RunID, WorkflowID: o.WorkflowID, WorkflowDigest: o.WorkflowDigest,
			InventoryComplete: true, Status: o.ReportStatus, StartedAt: o.StartedAt, FinishedAt: o.FinishedAt, ErrorCode: o.ErrorCode, Steps: o.Steps}
		return r.Validate()
	}
	switch o.State {
	case "dry_run", "missing", "invalid", "mismatched":
	default:
		return fmt.Errorf("invalid observation state")
	}
	if !reflect.DeepEqual(o, UnknownV5(i, o.State)) {
		return fmt.Errorf("uncertain observation has unsupported facts")
	}
	return nil
}
