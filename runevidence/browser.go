package runevidence

import (
	"encoding/json"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/OpenUdon/evidence/artifact"
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
	LeaseID             string                          `json:"lease_id"`
	LaunchNonce         string                          `json:"launch_nonce"`
	Worker              browsercontract.BrowserWorkerV1 `json:"worker"`
	DriverClosureSHA256 string                          `json:"driver_closure_sha256"`
	ContainmentLeaseID  string                          `json:"containment_lease_id"`
	AllTraffic          bool                            `json:"all_traffic"`
	Sandbox             bool                            `json:"sandbox"`
	Origins             []string                        `json:"origins"`
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
type BrowserCredentialLeaseWitness struct {
	Slots     []browsercontract.CredentialSlotBinding
	Revisions []browsercontract.CredentialRevision
}

// SessionAccess is observed native access metadata, not the requested Config.
// Fresh-only executions need no durable access witness or host acquisition.
type BrowserSessionAccessWitness struct {
	LeaseID string
	Outcome string
	Session browsercontract.BrowserSessionV1
	Binding BrowserSessionBindingWitness
}

// Full value-free native binding facts. BindingSHA256 is an opaque native
// identity; this package neither decodes it nor invents its hash algorithm.
type BrowserSessionBindingWitness struct {
	OwnerID, AgentID, Name, BindingSHA256 string
	ProfileSHA256, AuthenticationSHA256   string
	CredentialRevisions                   []browsercontract.CredentialRevision
	Origins                               []string
	Generation                            uint64
	CreatedAt, ExpiresAt                  string
}

type BrowserHostWitnesses struct {
	CredentialLease        *BrowserCredentialLeaseWitness
	SessionAccess          *BrowserSessionAccessWitness
	ExpectedSessionBinding *BrowserSessionBindingWitness // independent host expectation, anchored to Config's opaque binding
	Config                 BrowserConfigV1
	Inventory              udonreport.InventoryV5
	FactRefs               []string
	Launch                 BrowserLaunchWitness
	Join                   BrowserJoinWitness
	Trace                  []BrowserTraceEvent
	NonDispatch            *BrowserNonDispatchWitness
	SavePermission         *BrowserSavePermissionWitness
	DurableBindingCount    int
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
func joined(w BrowserJoinWitness, launch BrowserLaunchWitness, requireAccess bool) bool {
	return w.LaunchLeaseID == launch.LeaseID && w.ContainmentLeaseID == launch.ContainmentLeaseID && (browsercontract.Identifier(w.SessionAccessLeaseID) || !requireAccess && w.SessionAccessLeaseID == "") && w.TransportJoined && w.DriverJoined && w.ChromiumJoined && w.CallbacksJoined
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
	requiresAccess := expected.Session != nil && (expected.Session.ReuseAllowed || expected.Session.SaveAllowed)
	if !host.Launch.AllTraffic || !host.Launch.Sandbox || !browsercontract.Identifier(host.Launch.LeaseID) || host.Launch.LaunchNonce != expected.LaunchNonce || host.Launch.Worker != expected.Worker || host.Launch.DriverClosureSHA256 != expected.DriverClosureSHA256 || host.Launch.ContainmentLeaseID != expected.ContainmentLeaseID || !reflect.DeepEqual(host.Launch.Origins, expected.Origins) || !joined(host.Join, host.Launch, requiresAccess) {
		return fail()
	}
	if host.SessionAccess == nil {
		if requiresAccess || host.Join.SessionAccessLeaseID != "" {
			return fail()
		}
	} else if host.DurableBindingCount != 1 || !sessionAccessMatches(expected, host.Join.SessionAccessLeaseID, host.SessionAccess, host.ExpectedSessionBinding) {
		return fail()
	}
	deadline, _ := time.Parse(time.RFC3339Nano, expected.AdmittedDeadline)
	reportStart, err := time.Parse(time.RFC3339Nano, observation.StartedAt)
	if err != nil || !reportStart.Before(deadline) {
		return fail()
	}
	if host.SessionAccess != nil && host.SessionAccess.Outcome == "reuse" {
		created, _ := time.Parse(time.RFC3339Nano, host.SessionAccess.Session.CreatedAt)
		expires, _ := time.Parse(time.RFC3339Nano, host.SessionAccess.Session.ExpiresAt)
		if reportStart.Before(created) || !reportStart.Before(expires) {
			return fail()
		}
		for i, call := range expected.ApprovedCalls {
			if call.SessionName == host.SessionAccess.Session.Name && observation.Steps[i].StartedAt != "" {
				at, err := time.Parse(time.RFC3339Nano, observation.Steps[i].StartedAt)
				if err != nil || at.Before(created) || !at.Before(expires) {
					return fail()
				}
			}
		}
	}
	for _, step := range observation.Steps {
		if step.StartedAt != "" {
			at, err := time.Parse(time.RFC3339Nano, step.StartedAt)
			if err != nil || !at.Before(deadline) {
				return fail()
			}
		}
	}
	outcomes := make([]string, len(inventory.Steps))
	for i := range outcomes {
		outcomes[i] = "not_started"
	}
	terminal := make([]bool, len(outcomes))
	sent := make([]bool, len(outcomes))
	requests := make([]string, len(outcomes))
	lastMessage := make([]string, len(outcomes))
	closedRegistration := make([]bool, len(outcomes))
	launchDone, joinDone, released, anySent, nonDispatch, candidate, generationCurrent, saveDone := false, false, false, false, false, false, false, false
	credentialsLeased := len(expected.CredentialRevisions) == 0
	stopped := false
	humanDenied := false
	stopExecution := func() {
		stopped = true
		for i := range outcomes {
			if sent[i] && outcomes[i] == "unknown" {
				terminal[i] = true
			}
		}
	}
	lastStarted := -1
	ordinal := uint64(0)
	pending := ""
	pendingQuestion := ""
	var question *BrowserQuestionWitness
	questionKind := ""
	questionLeaf := -1
	answered, rechecked := false, false
	acquired := ""
	for _, e := range host.Trace {
		if e.RunID != expected.RunID || e.PlanSHA256 != expected.PlanSHA256 {
			return fail()
		}
		if e.Event == "credential_lease" {
			if launchDone || anySent || (e.Outcome != "whole-plan-private" && e.Outcome != "whole-plan-private-totp-seed") || !credentialLeaseMatches(expected, host.CredentialLease) {
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
			if expected.Session == nil || launchDone || anySent || joinDone || acquired != "" && !((acquired == "missing" || acquired == "expired") && e.Outcome == "fresh") {
				return fail()
			}
			if access := host.SessionAccess; access != nil && (access.Session.ReuseAllowed && !call.ReuseAllowed || access.Session.SaveAllowed && !call.SaveAllowed || call.SessionName != access.Session.Name || call.Kind == "authentication" && call.SourceSHA256 != access.Binding.AuthenticationSHA256 || call.Kind == "action" && call.SourceSHA256 != access.Binding.ProfileSHA256) {
				return fail()
			}
			switch e.Outcome {
			case "fresh", "missing", "expired":
			case "reuse":
				if expected.Session == nil || !expected.Session.ReuseAllowed || !call.ReuseAllowed || call.SessionName != expected.Session.Name {
					return fail()
				}
			default:
				return fail()
			}
			if host.SessionAccess != nil && e.Outcome != "missing" && e.Outcome != "expired" && e.Outcome != host.SessionAccess.Outcome {
				return fail()
			}
			acquired = e.Outcome
		case "launch":
			if launchDone || joinDone || !credentialsLeased || e.Outcome != "contained" || acquired != "" && acquired != "fresh" && acquired != "reuse" || host.SessionAccess != nil && acquired != host.SessionAccess.Outcome {
				return fail()
			}
			launchDone = true
		case "registration_input":
			if call.Kind != "registration" || joinDone || stopped || e.Question != nil && !sameCanonical(question, e.Question) {
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
			if !launchDone || joinDone || stopped || terminal[index] || !questionValid(e.Question) || e.Outcome != "runtime_confirmation" && (!sent[index] || outcomes[index] != "unknown") {
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
				if call.Kind != "action" || sent[index] {
					return fail()
				}
			default:
				return fail()
			}
			question, questionKind, questionLeaf = e.Question, e.Outcome, index
			answered, rechecked = false, false
		case "interact":
			if question == nil || questionLeaf != index || joinDone || stopped || e.Question != nil && !sameCanonical(question, e.Question) || questionKind == "runtime_confirmation" && !sameCanonical(question, e.Question) {
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
			if e.Outcome == "deny" {
				stopExecution()
				humanDenied = true
			}
		case "authority_recheck":
			if question == nil || !answered || questionLeaf != index || e.Outcome != "current" || joinDone || stopped || e.Question != nil && !sameCanonical(question, e.Question) || questionKind == "runtime_confirmation" && !sameCanonical(question, e.Question) {
				return fail()
			}
			rechecked = true
		case "claim_dispatch":
			if !launchDone || joinDone || stopped {
				return fail()
			}
			if e.Outcome == "refused" {
				stopExecution()
				continue
			}
			if !browsercontract.Identifier(e.ProtocolRequestID) || e.MessageOrdinal != ordinal+1 || e.MessageOrdinal > 4096 || e.OuterProtocol != call.OuterProtocol {
				return fail()
			}
			if !sent[index] {
				if e.MessageKind != initialKind(call.Kind) || e.Outcome != call.Kind || index != lastStarted+1 {
					return fail()
				}
				if questionKind == "runtime_confirmation" {
					if questionLeaf != index || !answered || !rechecked || !sameCanonical(question, e.Question) {
						return fail()
					}
				} else if e.Question != nil {
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
			} else if e.Outcome != call.Kind || questionKind == "runtime_confirmation" && !sameCanonical(question, e.Question) {
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
			if !sent[index] || joinDone || terminal[index] || lastMessage[index] != traceKey(e) || outcomes[index] != "unknown" {
				return fail()
			}
			terminal[index] = true
			switch e.Outcome {
			case "success":
				outcomes[index] = "succeeded"
			case "definite_failure":
				outcomes[index] = "failed"
				stopExecution()
			case "unknown":
				stopExecution()
			default:
				return fail()
			}
		case "write_completed":
			if !sent[index] || joinDone || terminal[index] || e.Outcome != "proved" {
				return fail()
			}
		case "extraction", "typed_driver_error":
			if !sent[index] || joinDone || outcomes[index] == "succeeded" || outcomes[index] == "failed" {
				return fail()
			}
			outcomes[index] = "unknown"
			terminal[index] = true
			stopExecution()
		case "registration_context_close":
			if call.Kind != "registration" || !sent[index] || e.Outcome != "proved" || joinDone {
				return fail()
			}
			closedRegistration[index] = true
		case "candidate":
			if humanDenied || call.Kind != "authentication" || !call.SaveAllowed || host.SessionAccess == nil || !host.SessionAccess.Session.SaveAllowed || outcomes[index] != "succeeded" || candidate || joinDone || e.Outcome != "staged-before-close" {
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
			if humanDenied || !joinDone || !candidate || !generationCurrent || saveDone || p == nil || s == nil || host.SessionAccess == nil || !host.SessionAccess.Session.SaveAllowed || !p.OneUse || !p.SaveAllowed || !browsercontract.Identifier(p.PermissionID) || p.SessionName != s.Name || p.BindingSHA256 != s.BindingSHA256 || p.Generation != s.Generation || p.AuthenticateOperationID != call.OperationID || !call.SaveAllowed || e.Outcome != "encrypted-accepted" {
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
			stopExecution()
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
	if o.Inventory.Validate() != nil || o.Inventory.RunID != o.ExpectedConfig.RunID || o.Inventory.WorkflowDigest != "sha256:"+o.ExpectedConfig.WorkflowSHA256 || len(o.Inventory.Steps) != len(o.ExpectedConfig.ApprovedCalls) || browsercontract.Decode(data, &e) != nil || !browserEvidenceMetadataValid(e) || e.RunID != o.ExpectedConfig.RunID || e.Scope != o.Scope || e.DryRun != o.DryRun || o.Authority.Validate() != nil || !sameCanonical(o.Authority.Config, o.ExpectedConfig) || !sameCanonical(e.Browser, o.ExpectedConfig) || o.ExpectedConfig.Validate() != nil {
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
	if !browserEvidenceMetadataValid(e) {
		return nil, browsercontract.ErrContract
	}
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

func browserEvidenceMetadataValid(e BrowserRunEvidenceV1) bool {
	scope, err := artifact.CleanRelativePath(e.Scope, artifact.Options{})
	if err != nil || scope != e.Scope || len(scope) > 2048 || e.Version != browsercontract.RunEvidenceVersion || e.Browser.Validate() != nil || e.RunID != e.Browser.RunID || e.Tier != "sandbox" || e.ApprovalState != "approved_for_sandbox" || e.StepExecution == nil || e.StepExecution.Validate() != nil {
		return false
	}
	created, err := time.Parse(time.RFC3339Nano, e.CreatedAt)
	if err != nil || e.StepExecution.RunID != e.RunID || e.StepExecution.WorkflowDigest != "sha256:"+e.Browser.WorkflowSHA256 || len(e.StepExecution.Steps) != len(e.Browser.ApprovedCalls) {
		return false
	}
	for i, call := range e.Browser.ApprovedCalls {
		step := e.StepExecution.Steps[i]
		if step.StepID != call.StepID || step.OperationID != call.OperationID || step.InvocationID != call.InvocationID {
			return false
		}
	}
	if e.DryRun {
		return !e.Executor.Invoked && e.Executor.Mode == "dry_run" && e.Executor.ReportSHA256 == "" && e.Executor.ReportSize == 0 && e.StepExecution.State == "dry_run"
	}
	last := e.StepExecution.FinishedAt
	if last == "" {
		last = e.StepExecution.StartedAt
	}
	finished, err := time.Parse(time.RFC3339Nano, last)
	return err == nil && !created.Before(finished) && e.StepExecution.State == "validated" && e.Executor.Invoked && e.Executor.Mode == "native" && browsercontract.DigestValid(e.Executor.ReportSHA256) && e.Executor.ReportSize > 0 && e.Executor.ReportSize <= browsercontract.MaxBytes
}

func credentialLeaseMatches(c BrowserConfigV1, w *BrowserCredentialLeaseWitness) bool {
	if w == nil || !reflect.DeepEqual(c.CredentialRevisions, w.Revisions) {
		return false
	}
	expected := map[string]bool{}
	for _, call := range c.ApprovedCalls {
		for _, slot := range call.CredentialSlots {
			expected[slot.Slot+"\x00"+slot.Name] = true
		}
	}
	if len(w.Slots) != len(expected) {
		return false
	}
	seen := map[string]bool{}
	for _, slot := range w.Slots {
		key := slot.Slot + "\x00" + slot.Name
		if !expected[key] || seen[key] {
			return false
		}
		seen[key] = true
	}
	return true
}

func sessionAccessMatches(config BrowserConfigV1, lease string, observed *BrowserSessionAccessWitness, independentlyExpected *BrowserSessionBindingWitness) bool {
	expected := config.Session
	if expected == nil || observed == nil || !browsercontract.Identifier(observed.LeaseID) || observed.LeaseID != lease || observed.Outcome != "fresh" && observed.Outcome != "reuse" {
		return false
	}
	s := observed.Session
	if s.ReuseAllowed && !expected.ReuseAllowed || s.SaveAllowed && !expected.SaveAllowed || observed.Outcome == "reuse" && !s.ReuseAllowed {
		return false
	}
	// Observed permissions may narrow approved rights on missing/expired->fresh;
	// all immutable binding identities and timestamps still match exactly.
	s.ReuseAllowed, s.SaveAllowed = expected.ReuseAllowed, expected.SaveAllowed
	if !sameCanonical(s, expected) {
		return false
	}
	bound, ok := expectedSessionBinding(config, independentlyExpected)
	return ok && reflect.DeepEqual(observed.Binding, bound)
}

func expectedSessionBinding(c BrowserConfigV1, independent *BrowserSessionBindingWitness) (BrowserSessionBindingWitness, bool) {
	if c.Session == nil {
		return BrowserSessionBindingWitness{}, false
	}
	s := c.Session
	b := BrowserSessionBindingWitness{OwnerID: c.OwnerID, AgentID: c.AgentID, Name: s.Name, BindingSHA256: s.BindingSHA256, Generation: s.Generation, CreatedAt: s.CreatedAt, ExpiresAt: s.ExpiresAt, CredentialRevisions: []browsercontract.CredentialRevision{}, Origins: []string{}}
	authID := ""
	profiles, credentials, origins := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, call := range c.ApprovedCalls {
		if call.SessionName != s.Name {
			continue
		}
		for _, o := range call.Origins {
			origins[o] = true
		}
		for _, slot := range call.CredentialSlots {
			credentials[slot.Name] = true
		}
		if call.Kind == "authentication" {
			if authID != "" {
				return BrowserSessionBindingWitness{}, false
			}
			authID, b.AuthenticationSHA256 = call.SourceID, call.SourceSHA256
		}
		if call.Kind == "action" {
			profiles[call.SourceSHA256] = true
		}
	}
	for _, call := range c.ApprovedCalls {
		if call.Kind == "action" && call.SessionName == s.Name && call.AuthenticationSourceID != authID {
			return BrowserSessionBindingWitness{}, false
		}
	}
	if len(profiles) > 1 {
		return BrowserSessionBindingWitness{}, false
	}
	for sha := range profiles {
		b.ProfileSHA256 = sha
	}
	if len(origins) == 0 {
		for _, o := range c.Origins {
			origins[o] = true
		}
		for _, r := range c.CredentialRevisions {
			credentials[r.Name] = true
		}
	}
	for _, r := range c.CredentialRevisions {
		if credentials[r.Name] {
			b.CredentialRevisions = append(b.CredentialRevisions, r)
		}
	}
	if len(b.CredentialRevisions) != len(credentials) {
		return BrowserSessionBindingWitness{}, false
	}
	for o := range origins {
		b.Origins = append(b.Origins, o)
	}
	sort.Strings(b.Origins)
	// An auth-only plan has no action-profile digest to derive. Its expected
	// native binding must come separately from the host, anchored to Config.
	// Fresh-only legacy metadata may likewise lack an establishing auth source.
	if b.ProfileSHA256 == "" || b.AuthenticationSHA256 == "" {
		if independent == nil {
			return BrowserSessionBindingWitness{}, false
		}
		if b.ProfileSHA256 == "" {
			b.ProfileSHA256 = independent.ProfileSHA256
		}
		if b.AuthenticationSHA256 == "" {
			b.AuthenticationSHA256 = independent.AuthenticationSHA256
		}
	}
	if !browsercontract.DigestValid(b.ProfileSHA256) || !browsercontract.DigestValid(b.AuthenticationSHA256) || independent != nil && !reflect.DeepEqual(*independent, b) {
		return BrowserSessionBindingWitness{}, false
	}
	return b, true
}
