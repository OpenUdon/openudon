package runevidence

import (
	"encoding/json"
	"reflect"
	"strconv"
	"time"

	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/udonreport"
)

type BrowserConfigV1 = browsercontract.BrowserConfigV1
type BrowserRunEvidenceV1 = browsercontract.BrowserRunEvidenceV1
type BrowserExecutorV1 = browsercontract.BrowserExecutorV1

// These witnesses are independent trusted-host inputs, never fields decoded
// from the submitted evidence/report. Booleans here do not attest an actual
// browser/container: the consumer must establish their origin and current scope.
type BrowserLaunchWitness struct {
	LeaseID            string   `json:"lease_id"`
	ContainmentLeaseID string   `json:"containment_lease_id"`
	AllTraffic         bool     `json:"all_traffic"`
	Sandbox            bool     `json:"sandbox"`
	Origins            []string `json:"origins"`
}
type BrowserJoinWitness struct {
	LaunchLeaseID        string `json:"launch_lease_id"`
	ContainmentLeaseID   string `json:"containment_lease_id"`
	SessionAccessLeaseID string `json:"session_access_lease_id"`
	TransportJoined      bool   `json:"transport_joined"`
	DriverJoined         bool   `json:"driver_joined"`
	ChromiumJoined       bool   `json:"chromium_joined"`
	CallbacksJoined      bool   `json:"callbacks_joined"`
}
type BrowserQuestionWitness struct {
	ID           string `json:"id"`
	Revision     uint64 `json:"revision"`
	IssuedSHA256 string `json:"issued_sha256"`
}
type BrowserTraceEvent struct {
	Event                string                  `json:"event"`
	Outcome              string                  `json:"outcome"`
	RunID                string                  `json:"run_id"`
	PlanSHA256           string                  `json:"plan_sha256"`
	LaunchLeaseID        string                  `json:"launch_lease_id,omitempty"`
	ContainmentLeaseID   string                  `json:"containment_lease_id,omitempty"`
	SessionAccessLeaseID string                  `json:"session_access_lease_id,omitempty"`
	StepID               string                  `json:"step_id,omitempty"`
	OperationID          string                  `json:"operation_id,omitempty"`
	InvocationID         string                  `json:"invocation_id,omitempty"`
	ProtocolRequestID    string                  `json:"protocol_request_id,omitempty"`
	MessageOrdinal       uint64                  `json:"message_ordinal,omitempty"`
	MessageKind          string                  `json:"message_kind,omitempty"`
	OuterProtocol        string                  `json:"outer_protocol,omitempty"`
	Question             *BrowserQuestionWitness `json:"question,omitempty"`
}
type BrowserNonDispatchWitness struct {
	RunID                                    string
	PlanSHA256                               string
	ApprovedCalls                            []browsercontract.BrowserCallV1
	AllInitialAndContinuationMessagesCovered bool
	NoMessageTransmitted                     bool
	Joined                                   BrowserJoinWitness
}
type BrowserSavePermissionWitness struct {
	PermissionID            string `json:"permission_id"`
	SessionName             string `json:"session_name"`
	BindingSHA256           string `json:"binding_sha256"`
	Generation              uint64 `json:"generation"`
	AuthenticateOperationID string `json:"authenticate_operation_id"`
	OneUse                  bool   `json:"one_use"`
	SaveAllowed             bool   `json:"save_allowed"`
}
type BrowserHostWitnesses struct {
	Config              BrowserConfigV1
	Inventory           udonreport.InventoryV5
	FactRefs            []string
	Launch              BrowserLaunchWitness
	Join                BrowserJoinWitness
	Trace               []BrowserTraceEvent
	NonDispatch         *BrowserNonDispatchWitness
	SavePermission      *BrowserSavePermissionWitness
	DurableBindingCount int
}
type BrowserReportResult struct {
	Observation       udonreport.ObservationV5
	SuccessorEligible bool // evidence for a distinct freshly confirmed proposal only
}

func browserFailure(inventory udonreport.InventoryV5) (BrowserReportResult, error) {
	return BrowserReportResult{Observation: udonreport.UnknownV5(inventory, "invalid")}, browsercontract.ErrContract
}
func sameCanonical(a, b any) bool {
	left, e1 := browsercontract.CanonicalDigest(a)
	right, e2 := browsercontract.CanonicalDigest(b)
	return e1 == nil && e2 == nil && left == right
}
func joined(w BrowserJoinWitness, launch BrowserLaunchWitness) bool {
	return w.LaunchLeaseID == launch.LeaseID && w.ContainmentLeaseID == launch.ContainmentLeaseID && browsercontract.Identifier(w.SessionAccessLeaseID) && w.TransportJoined && w.DriverJoined && w.ChromiumJoined && w.CallbacksJoined
}
func traceKey(e BrowserTraceEvent) string {
	return e.StepID + "\x00" + e.OperationID + "\x00" + e.InvocationID + "\x00" + e.ProtocolRequestID + "\x00" + strconv.FormatUint(e.MessageOrdinal, 10) + "\x00" + e.MessageKind + "\x00" + e.OuterProtocol
}
func initialKind(kind string) string {
	switch kind {
	case "authentication":
		return "authenticate"
	case "registration":
		return "register"
	case "action":
		return "action"
	}
	return ""
}
func questionValid(q *BrowserQuestionWitness) bool {
	return q != nil && browsercontract.Identifier(q.ID) && q.Revision > 0 && q.Revision <= 9007199254740991 && browsercontract.DigestValid(q.IssuedSHA256)
}

// ObserveBrowserReport binds unchanged report-v5 bytes to the exact independently
// approved attempt/inventory and trusted dispatch/teardown witnesses. Missing,
// malformed, foreign or dishonest evidence returns conservative uncertainty.
// A typed error is never a positive non-dispatch or retry/continuation proof.
func ObserveBrowserReport(expected BrowserConfigV1, inventory udonreport.InventoryV5, report []byte, host BrowserHostWitnesses) (BrowserReportResult, error) {
	fail := func() (BrowserReportResult, error) { return browserFailure(inventory) }
	if expected.Validate() != nil || inventory.Validate() != nil || host.DurableBindingCount < 0 || host.DurableBindingCount > 1 || !sameCanonical(expected, host.Config) || !reflect.DeepEqual(inventory, host.Inventory) || !reflect.DeepEqual(expected.HostFactRefs, host.FactRefs) || len(host.Trace) == 0 || len(host.Trace) > 4096 {
		return fail()
	}
	if inventory.RunID != expected.RunID || inventory.WorkflowDigest != "sha256:"+expected.WorkflowSHA256 || len(inventory.Steps) != len(expected.ApprovedCalls) {
		return fail()
	}
	observation := udonreport.ObserveV5(inventory, report)
	if observation.State != "validated" {
		return fail()
	}
	for i, call := range expected.ApprovedCalls {
		step := inventory.Steps[i]
		if step.StepID != call.StepID || step.OperationID != call.OperationID || step.InvocationID != call.InvocationID {
			return fail()
		}
	}
	if !host.Launch.AllTraffic || !host.Launch.Sandbox || !browsercontract.Identifier(host.Launch.LeaseID) || host.Launch.ContainmentLeaseID != expected.ContainmentLeaseID || !reflect.DeepEqual(host.Launch.Origins, expected.Origins) || !joined(host.Join, host.Launch) {
		return fail()
	}
	deadline, _ := time.Parse(time.RFC3339Nano, expected.AdmittedDeadline)
	reportStart, err := time.Parse(time.RFC3339Nano, observation.StartedAt)
	if err != nil || !reportStart.Before(deadline) {
		return fail()
	}
	outcomes := make([]string, len(inventory.Steps))
	for i := range outcomes {
		outcomes[i] = "not_started"
	}
	sent := make([]bool, len(outcomes))
	requests := make([]string, len(outcomes))
	lastMessage := make([]string, len(outcomes))
	closedRegistration := make([]bool, len(outcomes))
	launchDone, joinDone, released, anySent, nonDispatch, candidate, generationCurrent, saveDone := false, false, false, false, false, false, false, false
	credentialsLeased := len(expected.CredentialRevisions) == 0
	stopped := false
	lastStarted := -1
	ordinal := uint64(0)
	pending := ""
	pendingQuestion := ""
	var question *BrowserQuestionWitness
	questionKind := ""
	questionLeaf := -1
	answered, rechecked := false, false
	for _, e := range host.Trace {
		if e.RunID != expected.RunID || e.PlanSHA256 != expected.PlanSHA256 {
			return fail()
		}
		if e.Event == "credential_lease" {
			if launchDone || anySent || e.Outcome == "" {
				return fail()
			}
			credentialsLeased = true
			continue
		}
		if e.LaunchLeaseID != host.Launch.LeaseID || e.ContainmentLeaseID != host.Launch.ContainmentLeaseID || e.SessionAccessLeaseID != host.Join.SessionAccessLeaseID {
			return fail()
		}
		index := -1
		for i, call := range expected.ApprovedCalls {
			if call.StepID == e.StepID && call.OperationID == e.OperationID && call.InvocationID == e.InvocationID {
				index = i
				break
			}
		}
		if index < 0 {
			return fail()
		}
		call := expected.ApprovedCalls[index]
		if pending != "" && e.Event != "send" {
			return fail()
		}
		if released {
			return fail()
		}
		switch e.Event {
		case "session_acquire":
			if anySent || joinDone {
				return fail()
			}
			switch e.Outcome {
			case "fresh", "missing", "expired":
			case "reuse":
				if expected.Session == nil || !expected.Session.ReuseAllowed {
					return fail()
				}
			default:
				return fail()
			}
		case "launch":
			if launchDone || joinDone || !credentialsLeased || e.Outcome != "contained" {
				return fail()
			}
			launchDone = true
		case "registration_input":
			if call.Kind != "registration" || joinDone {
				return fail()
			}
			if e.Outcome == "initial-private" {
				if anySent {
					return fail()
				}
			} else if e.Outcome == "checkpoint-private" {
				if questionLeaf != index || questionKind != "registration_input" || question == nil {
					return fail()
				}
				answered = true
			} else {
				return fail()
			}
		case "question":
			if !launchDone || joinDone || !sent[index] || outcomes[index] != "unknown" || !questionValid(e.Question) {
				return fail()
			}
			switch e.Outcome {
			case "mfa_push", "mfa_number_match":
				if call.Kind != "authentication" {
					return fail()
				}
			case "registration_input", "registration_checkpoint", "registration_submit":
				if call.Kind != "registration" {
					return fail()
				}
			case "runtime_confirmation":
				if call.Kind != "action" {
					return fail()
				}
			default:
				return fail()
			}
			question, questionKind, questionLeaf = e.Question, e.Outcome, index
			answered, rechecked = false, false
		case "interact":
			if question == nil || questionLeaf != index || joinDone {
				return fail()
			}
			switch questionKind {
			case "mfa_push", "mfa_number_match", "runtime_confirmation":
				if e.Outcome != "approve" && e.Outcome != "deny" {
					return fail()
				}
			case "registration_checkpoint", "registration_submit":
				if e.Outcome != "continue" && e.Outcome != "deny" {
					return fail()
				}
			default:
				return fail()
			}
			answered = e.Outcome != "deny"
		case "authority_recheck":
			if question == nil || !answered || questionLeaf != index || e.Outcome != "current" || joinDone {
				return fail()
			}
			rechecked = true
		case "claim_dispatch":
			if !launchDone || joinDone || stopped {
				return fail()
			}
			if e.Outcome == "refused" {
				stopped = true
				continue
			}
			if !browsercontract.Identifier(e.ProtocolRequestID) || e.MessageOrdinal != ordinal+1 || e.MessageOrdinal > 4096 || e.OuterProtocol != call.OuterProtocol {
				return fail()
			}
			if !sent[index] {
				if e.MessageKind != initialKind(call.Kind) || e.Outcome != call.Kind || index != lastStarted+1 || e.Question != nil {
					return fail()
				}
				for i := 0; i < index; i++ {
					if outcomes[i] != "succeeded" || expected.ApprovedCalls[i].Kind == "registration" && !closedRegistration[i] {
						return fail()
					}
				}
			} else {
				if outcomes[index] != "unknown" || questionLeaf != index || !answered || !rechecked || !sameCanonical(question, e.Question) || e.Outcome != questionKind || requests[index] != e.ProtocolRequestID {
					return fail()
				}
				switch questionKind {
				case "mfa_push", "mfa_number_match":
					if e.MessageKind != "challenge_response" {
						return fail()
					}
				case "registration_input":
					if e.MessageKind != "registration_input_response" {
						return fail()
					}
				case "registration_checkpoint", "registration_submit":
					if e.MessageKind != "registration_checkpoint_response" {
						return fail()
					}
				default:
					return fail()
				}
			}
			pending = traceKey(e)
			pendingQuestion = questionKind
		case "send":
			if !launchDone || joinDone || stopped || pending == "" || pending != traceKey(e) {
				return fail()
			}
			if sent[index] {
				if !sameCanonical(question, e.Question) || e.Outcome != pendingQuestion {
					return fail()
				}
			} else if e.Outcome != call.Kind {
				return fail()
			}
			pending = ""
			pendingQuestion = ""
			ordinal = e.MessageOrdinal
			anySent = true
			sent[index] = true
			requests[index] = e.ProtocolRequestID
			lastMessage[index] = traceKey(e)
			outcomes[index] = "unknown"
			if index > lastStarted {
				lastStarted = index
			}
			question = nil
			questionKind = ""
			questionLeaf = -1
			answered, rechecked = false, false
		case "response":
			if !sent[index] || joinDone || lastMessage[index] != traceKey(e) || outcomes[index] != "unknown" {
				return fail()
			}
			switch e.Outcome {
			case "success":
				outcomes[index] = "succeeded"
			case "definite_failure":
				outcomes[index] = "failed"
				stopped = true
			case "unknown":
				stopped = true
			default:
				return fail()
			}
		case "write_completed", "extraction", "typed_driver_error":
			if !sent[index] || joinDone || outcomes[index] == "succeeded" || outcomes[index] == "failed" {
				return fail()
			}
			outcomes[index] = "unknown"
			stopped = true
		case "registration_context_close":
			if call.Kind != "registration" || !sent[index] || e.Outcome != "proved" || joinDone {
				return fail()
			}
			closedRegistration[index] = true
		case "candidate":
			if call.Kind != "authentication" || !call.SaveAllowed || outcomes[index] != "succeeded" || candidate || joinDone || e.Outcome != "staged-before-close" {
				return fail()
			}
			candidate = true
		case "join":
			if !launchDone || joinDone || e.Outcome != "proved" {
				return fail()
			}
			joinDone = true
		case "generation_recheck":
			if !joinDone || expected.Session == nil || e.Outcome != "current-"+strconv.FormatUint(expected.Session.Generation, 10) {
				return fail()
			}
			generationCurrent = true
		case "session_save":
			p := host.SavePermission
			s := expected.Session
			if !joinDone || !candidate || !generationCurrent || saveDone || p == nil || s == nil || !p.OneUse || !p.SaveAllowed || !browsercontract.Identifier(p.PermissionID) || p.SessionName != s.Name || p.BindingSHA256 != s.BindingSHA256 || p.Generation != s.Generation || p.AuthenticateOperationID != call.OperationID || !call.SaveAllowed || e.Outcome != "encrypted-accepted" {
				return fail()
			}
			saveDone = true
		case "session_release":
			if !joinDone || e.Outcome != "proved" {
				return fail()
			}
			released = true
		case "non_dispatch_check":
			if anySent || e.Outcome != "none" {
				return fail()
			}
		case "non_dispatch":
			p := host.NonDispatch
			if anySent || !joinDone || e.Outcome != "complete" || p == nil || p.RunID != expected.RunID || p.PlanSHA256 != expected.PlanSHA256 || !p.AllInitialAndContinuationMessagesCovered || !p.NoMessageTransmitted || !sameCanonical(p.ApprovedCalls, expected.ApprovedCalls) || !reflect.DeepEqual(p.Joined, host.Join) {
				return fail()
			}
			nonDispatch = true
		case "failure":
			if e.Outcome != "checkpoint_failed" || outcomes[index] != "succeeded" || joinDone {
				return fail()
			}
		default:
			return fail()
		}
	}
	if pending != "" || !joinDone || !released {
		return fail()
	}
	for i, step := range observation.Steps {
		if step.Outcome != outcomes[i] || sent[i] && expected.ApprovedCalls[i].Kind == "registration" && !closedRegistration[i] {
			return fail()
		}
	}
	if host.NonDispatch != nil && !nonDispatch {
		return fail()
	}
	return BrowserReportResult{Observation: observation, SuccessorEligible: nonDispatch && !anySent}, nil
}

type BrowserVerifyOptions struct {
	ExpectedConfig BrowserConfigV1
	Authority      browsercontract.BrowserAuthorityV1
	Inventory      udonreport.InventoryV5
	Scope          string
	DryRun         bool
	ReportJSON     []byte
	Host           BrowserHostWitnesses
}

// VerifyBrowserRunEvidence verifies a separate new closed wire; all legacy
// RunEvidence/BrowserConfig and report-v5 wire bytes retain their meanings.
func VerifyBrowserRunEvidence(data []byte, o BrowserVerifyOptions) (BrowserRunEvidenceV1, BrowserReportResult, error) {
	fail := func() (BrowserRunEvidenceV1, BrowserReportResult, error) {
		r, e := browserFailure(o.Inventory)
		return BrowserRunEvidenceV1{}, r, e
	}
	var e BrowserRunEvidenceV1
	if o.Inventory.Validate() != nil || o.Inventory.RunID != o.ExpectedConfig.RunID || o.Inventory.WorkflowDigest != "sha256:"+o.ExpectedConfig.WorkflowSHA256 || len(o.Inventory.Steps) != len(o.ExpectedConfig.ApprovedCalls) || browsercontract.Decode(data, &e) != nil || e.Version != browsercontract.RunEvidenceVersion || e.RunID != o.ExpectedConfig.RunID || e.Scope != o.Scope || e.Tier != "sandbox" || e.ApprovalState != "approved_for_sandbox" || e.DryRun != o.DryRun || o.Authority.Validate() != nil || !sameCanonical(o.Authority.Config, o.ExpectedConfig) || !sameCanonical(e.Browser, o.ExpectedConfig) || o.ExpectedConfig.Validate() != nil {
		return fail()
	}
	for i, c := range o.ExpectedConfig.ApprovedCalls {
		step := o.Inventory.Steps[i]
		if step.StepID != c.StepID || step.OperationID != c.OperationID || step.InvocationID != c.InvocationID {
			return fail()
		}
	}
	created, err := time.Parse(time.RFC3339Nano, e.CreatedAt)
	if err != nil {
		return fail()
	}
	if e.DryRun {
		expected := udonreport.UnknownV5(o.Inventory, "dry_run")
		if e.Executor.Invoked || e.Executor.Mode != "dry_run" || e.Executor.ReportSHA256 != "" || e.Executor.ReportSize != 0 || len(o.ReportJSON) != 0 || e.StepExecution == nil || !reflect.DeepEqual(*e.StepExecution, expected) {
			return fail()
		}
		return e, BrowserReportResult{Observation: expected}, nil
	}
	if !e.Executor.Invoked || e.Executor.Mode != "native" || e.Executor.ReportSize != int64(len(o.ReportJSON)) || !browsercontract.DigestValid(e.Executor.ReportSHA256) || e.Executor.ReportSHA256 != browsercontract.SHA256(o.ReportJSON) || e.StepExecution == nil {
		return fail()
	}
	result, err := ObserveBrowserReport(o.ExpectedConfig, o.Inventory, o.ReportJSON, o.Host)
	if err != nil || !reflect.DeepEqual(*e.StepExecution, result.Observation) {
		return fail()
	}
	last := result.Observation.FinishedAt
	if last == "" {
		last = result.Observation.StartedAt
	}
	at, err := time.Parse(time.RFC3339Nano, last)
	if err != nil || created.Before(at) {
		return fail()
	}
	return e, result, nil
}

// MarshalBrowserRunEvidence retains only bounded closed metadata. Verification
// still requires independently expected Config/Authority and host witnesses.
func MarshalBrowserRunEvidence(e BrowserRunEvidenceV1) ([]byte, error) {
	raw, err := json.Marshal(e)
	if err != nil || len(raw) > browsercontract.MaxBytes {
		return nil, browsercontract.ErrContract
	}
	var check BrowserRunEvidenceV1
	if browsercontract.Decode(raw, &check) != nil {
		return nil, browsercontract.ErrContract
	}
	return browsercontract.CanonicalJSON(raw)
}
