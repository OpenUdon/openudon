package runevidence_test

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/runevidence"
	"github.com/OpenUdon/openudon/udonreport"
)

type browserBindingFixture struct {
	OwnerID          string                               `json:"owner_id"`
	AgentID          string                               `json:"agent_id"`
	RunID            string                               `json:"run_id"`
	WorkflowID       string                               `json:"workflow_id"`
	WorkflowSHA256   string                               `json:"workflow_sha256"`
	PackageSHA256    string                               `json:"package_sha256"`
	InputsSHA256     string                               `json:"inputs_sha256"`
	PlanSHA256       string                               `json:"plan_sha256"`
	ApprovalSHA256   string                               `json:"approval_sha256"`
	Calls            []browsercontract.BrowserCallV1      `json:"approved_inventory"`
	AdmittedDeadline string                               `json:"admitted_deadline"`
	Credentials      []browsercontract.CredentialRevision `json:"credential_revisions"`
	Session          *browsercontract.BrowserSessionV1    `json:"session"`
}
type browserReportFixture struct {
	Name          string                `json:"name"`
	Expected      browserBindingFixture `json:"expected_browser"`
	SubmittedRefs []string              `json:"submitted_host_fact_refs"`
	Host          struct {
		CredentialSlots     []browsercontract.CredentialSlotBinding   `json:"credential_slots"`
		CredentialRevisions []browsercontract.CredentialRevision      `json:"credential_revisions"`
		Binding             browserBindingFixture                     `json:"binding"`
		Launch              runevidence.BrowserLaunchWitness          `json:"launch"`
		Join                runevidence.BrowserJoinWitness            `json:"join"`
		Save                *runevidence.BrowserSavePermissionWitness `json:"save_permission"`
		NonDispatch         *struct {
			RunID      string                          `json:"run_id"`
			PlanSHA256 string                          `json:"plan_sha256"`
			Calls      []browsercontract.BrowserCallV1 `json:"approved_inventory"`
			Complete   bool                            `json:"all_initial_and_continuation_messages_covered"`
			None       bool                            `json:"no_message_transmitted"`
			Joined     runevidence.BrowserJoinWitness  `json:"joined"`
		} `json:"non_dispatch"`
	} `json:"trusted_host_witnesses"`
	Trace        []runevidence.BrowserTraceEvent `json:"trace"`
	Report       json.RawMessage                 `json:"report"`
	ShapeValid   bool                            `json:"expected_report_shape_valid"`
	Valid        bool                            `json:"expected_browser_contract_valid"`
	Successor    bool                            `json:"expected_successor_eligible"`
	DurableCount *int                            `json:"submitted_durable_binding_count"`
}

func browserReportFixtures(t *testing.T) []browserReportFixture {
	t.Helper()
	raw, err := os.ReadFile("../packagev3/testdata/browser/m51-report-v5-cases.json")
	if err != nil || browsercontract.SHA256(raw) != "219f2700e34516a7f64d48392e184ffafabce6138c336742c20a0cee6ed62b35" {
		t.Fatal("frozen fixture drift", err)
	}
	var f struct {
		Cases []browserReportFixture `json:"cases"`
	}
	if json.Unmarshal(raw, &f) != nil || len(f.Cases) != 34 {
		t.Fatal("frozen corpus inventory")
	}
	return f.Cases
}

// The frozen vectors contain host-neutral bindings rather than real worker
// closures. This adapter supplies explicit FIXTURE-ONLY worker/handoff identities
// identically to the expected and independent trusted input. It claims no native
// runtime, credential, browser or containment qualification.
func configFromFixture(b browserBindingFixture, launch runevidence.BrowserLaunchWitness, refs []string) browsercontract.BrowserConfigV1 {
	origins := []string{}
	for _, call := range b.Calls {
		for _, origin := range call.Origins {
			found := false
			for _, old := range origins {
				found = found || old == origin
			}
			if !found {
				origins = append(origins, origin)
			}
		}
	}
	return browsercontract.BrowserConfigV1{Version: browsercontract.ConfigVersion, OwnerID: b.OwnerID, AgentID: b.AgentID, RunID: b.RunID, PackageSHA256: b.PackageSHA256, HandoffSHA256: browsercontract.SHA256([]byte("FIXTURE-only-handoff")), InputsSHA256: b.InputsSHA256, ApprovalSHA256: b.ApprovalSHA256, PlanSHA256: b.PlanSHA256, WorkflowSHA256: b.WorkflowSHA256, SupplementSHA256: browsercontract.SHA256([]byte("FIXTURE-only-supplement")), Worker: browsercontract.BrowserWorkerV1{Profile: "fixture.exec.browser.v1", BinarySHA256: browsercontract.SHA256([]byte("FIXTURE-only-worker")), ClosureSHA256: browsercontract.SHA256([]byte("FIXTURE-only-worker-closure")), RuntimeRevision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}, DriverClosureSHA256: browsercontract.SHA256([]byte("FIXTURE-only-driver")), LaunchNonce: launch.LeaseID, ContainmentLeaseID: launch.ContainmentLeaseID, AdmittedDeadline: b.AdmittedDeadline, Origins: origins, ApprovedCalls: b.Calls, CredentialRevisions: b.Credentials, Session: b.Session, HostFactRefs: refs}
}
func inventoryFromFixture(b browserBindingFixture) udonreport.InventoryV5 {
	i := udonreport.InventoryV5{RunID: b.RunID, WorkflowID: b.WorkflowID, WorkflowDigest: "sha256:" + b.WorkflowSHA256, Steps: []udonreport.StepV5{}}
	for _, call := range b.Calls {
		i.Steps = append(i.Steps, udonreport.StepV5{StepID: call.StepID, OperationID: call.OperationID, InvocationID: call.InvocationID, Outcome: "not_started"})
	}
	return i
}
func fixtureInputs(c browserReportFixture) (browsercontract.BrowserConfigV1, udonreport.InventoryV5, runevidence.BrowserHostWitnesses) {
	expected := configFromFixture(c.Expected, c.Host.Launch, c.SubmittedRefs)
	inventory := inventoryFromFixture(c.Expected)
	host := runevidence.BrowserHostWitnesses{Config: configFromFixture(c.Host.Binding, c.Host.Launch, c.SubmittedRefs), Inventory: inventoryFromFixture(c.Host.Binding), FactRefs: c.SubmittedRefs, Launch: c.Host.Launch, Join: c.Host.Join, Trace: c.Trace, SavePermission: c.Host.Save, DurableBindingCount: 0}
	// The frozen launch vector lacks real closure/nonce projections. Supply
	// explicit FIXTURE-ONLY observed identities independently from Config;
	// production adapters must project the actual native containment instead.
	host.Launch.LaunchNonce = c.Host.Launch.LeaseID
	host.Launch.Worker = browsercontract.BrowserWorkerV1{Profile: "fixture.exec.browser.v1", BinarySHA256: browsercontract.SHA256([]byte("FIXTURE-only-worker")), ClosureSHA256: browsercontract.SHA256([]byte("FIXTURE-only-worker-closure")), RuntimeRevision: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	host.Launch.DriverClosureSHA256 = browsercontract.SHA256([]byte("FIXTURE-only-driver"))
	if c.Host.CredentialSlots != nil {
		host.CredentialLease = &runevidence.BrowserCredentialLeaseWitness{Slots: c.Host.CredentialSlots, Revisions: c.Host.CredentialRevisions}
	}
	if c.Host.Binding.Session != nil {
		host.DurableBindingCount = 1
		host.SessionAccess = &runevidence.BrowserSessionAccessWitness{LeaseID: c.Host.Join.SessionAccessLeaseID, Outcome: "fresh", Session: *c.Host.Binding.Session}
	}
	if c.DurableCount != nil {
		host.DurableBindingCount = *c.DurableCount
	}
	if p := c.Host.NonDispatch; p != nil {
		host.NonDispatch = &runevidence.BrowserNonDispatchWitness{RunID: p.RunID, PlanSHA256: p.PlanSHA256, ApprovedCalls: p.Calls, AllInitialAndContinuationMessagesCovered: p.Complete, NoMessageTransmitted: p.None, Joined: p.Joined}
	}
	return expected, inventory, host
}
func TestBrowserReportReproducesAllFrozenM51CasesIndependently(t *testing.T) {
	for _, c := range browserReportFixtures(t) {
		t.Run(c.Name, func(t *testing.T) {
			_, shapeErr := udonreport.DecodeV5(c.Report)
			if (shapeErr == nil) != c.ShapeValid {
				t.Fatal("report-v5 shape diverged", shapeErr)
			}
			expected, inventory, host := fixtureInputs(c)
			result, err := runevidence.ObserveBrowserReport(expected, inventory, c.Report, host)
			if (err == nil) != c.Valid || result.SuccessorEligible != c.Successor {
				t.Fatalf("contract valid=%v want=%v successor=%v want=%v: %v", err == nil, c.Valid, result.SuccessorEligible, c.Successor, err)
			}
			if !c.Valid {
				if result.Observation.State != "invalid" || result.SuccessorEligible {
					t.Fatal("rejected evidence supplied facts")
				}
				return
			}
			if result.Observation.State != "validated" {
				t.Fatal("lost accepted observation")
			}
			if c.Name == "write-extraction-failure" || c.Name == "lost-response" || c.Name == "unknown-leaf-terminal-error-valid" {
				last := result.Observation.Steps[len(result.Observation.Steps)-1]
				if last.Outcome != "unknown" || last.FinishedAt != "" || last.ErrorCode != "" {
					t.Fatal("possible write relabeled as safe")
				}
			}
		})
	}
}
func validBrowserReportInputs(t *testing.T) (browsercontract.BrowserConfigV1, udonreport.InventoryV5, []byte, runevidence.BrowserHostWitnesses) {
	t.Helper()
	c := browserReportFixtures(t)[2]
	expected, inventory, host := fixtureInputs(c)
	return expected, inventory, c.Report, host
}
func TestBrowserEvidenceNewWireBindsIndependentAttemptAndReport(t *testing.T) {
	c, i, report, host := validBrowserReportInputs(t)
	result, err := runevidence.ObserveBrowserReport(c, i, report, host)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := c.Digest()
	if err != nil {
		t.Fatal(err)
	}
	authority := browsercontract.BrowserAuthorityV1{Version: browsercontract.AuthorityVersion, ConfigSHA256: digest, Config: c}
	e := runevidence.BrowserRunEvidenceV1{Version: browsercontract.RunEvidenceVersion, RunID: c.RunID, CreatedAt: "2026-10-09T00:00:03Z", Scope: "workflows/W01-fixture", Tier: "sandbox", ApprovalState: "approved_for_sandbox", Browser: c, Executor: runevidence.BrowserExecutorV1{Invoked: true, Mode: "native", ReportSHA256: browsercontract.SHA256(report), ReportSize: int64(len(report))}, StepExecution: &result.Observation}
	raw, err := runevidence.MarshalBrowserRunEvidence(e)
	if err != nil {
		t.Fatal(err)
	}
	o := runevidence.BrowserVerifyOptions{ExpectedConfig: c, Authority: authority, Inventory: i, Scope: e.Scope, ReportJSON: report, Host: host}
	decoded, observed, err := runevidence.VerifyBrowserRunEvidence(raw, o)
	if err != nil || !reflect.DeepEqual(observed, result) || decoded.Version != e.Version {
		t.Fatal("new browser evidence", err)
	}
	for name, mutate := range map[string]func(*runevidence.BrowserVerifyOptions){
		"foreign attempt": func(o *runevidence.BrowserVerifyOptions) { o.ExpectedConfig.RunID = "other" },
		"foreign plan": func(o *runevidence.BrowserVerifyOptions) {
			o.ExpectedConfig.PlanSHA256 = browsercontract.SHA256([]byte("foreign"))
		},
		"report changed":                func(o *runevidence.BrowserVerifyOptions) { o.ReportJSON = append(o.ReportJSON, ' ') },
		"missing report":                func(o *runevidence.BrowserVerifyOptions) { o.ReportJSON = nil },
		"missing independent witnesses": func(o *runevidence.BrowserVerifyOptions) { o.Host = runevidence.BrowserHostWitnesses{} },
		"changed original deadline":     func(o *runevidence.BrowserVerifyOptions) { o.Host.Config.AdmittedDeadline = "2026-10-09T00:06:00Z" },
		"unjoined Chromium":             func(o *runevidence.BrowserVerifyOptions) { o.Host.Join.ChromiumJoined = false },
		"unapproved scope":              func(o *runevidence.BrowserVerifyOptions) { o.Scope = "other" },
		"fake dry run":                  func(o *runevidence.BrowserVerifyOptions) { o.DryRun = true },
	} {
		t.Run(name, func(t *testing.T) {
			clone := o
			clone.Host.Config = c
			mutate(&clone)
			if _, result, err := runevidence.VerifyBrowserRunEvidence(raw, clone); err == nil || result.SuccessorEligible {
				t.Fatal("foreign evidence accepted")
			}
		})
	}
	for _, field := range []string{"credential_values", "registration_snapshot", "driver_path", "safe_retry"} {
		var object map[string]json.RawMessage
		json.Unmarshal(raw, &object)
		object[field] = json.RawMessage(`"private"`)
		bad, _ := json.Marshal(object)
		if _, _, err := runevidence.VerifyBrowserRunEvidence(bad, o); err == nil {
			t.Fatal("private/authority field accepted", field)
		}
	}
}
func TestBrowserPositiveNonDispatchProofIsCompleteAndCannotTransferOldAuthority(t *testing.T) {
	var f browserReportFixture
	for _, c := range browserReportFixtures(t) {
		if c.Name == "complete-non-dispatch-proof" {
			f = c
			break
		}
	}
	expected, inventory, host := fixtureInputs(f)
	result, err := runevidence.ObserveBrowserReport(expected, inventory, f.Report, host)
	if err != nil || !result.SuccessorEligible {
		t.Fatal("complete positive witness", err)
	}
	for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
		"absent": func(w *runevidence.BrowserHostWitnesses) { w.NonDispatch = nil },
		"partial messages": func(w *runevidence.BrowserHostWitnesses) {
			w.NonDispatch.AllInitialAndContinuationMessagesCovered = false
		},
		"possible transmission": func(w *runevidence.BrowserHostWitnesses) { w.NonDispatch.NoMessageTransmitted = false },
		"old attempt":           func(w *runevidence.BrowserHostWitnesses) { w.NonDispatch.RunID = "old" },
		"unjoined":              func(w *runevidence.BrowserHostWitnesses) { w.NonDispatch.Joined.CallbacksJoined = false },
	} {
		t.Run(name, func(t *testing.T) {
			clone := host
			p := *host.NonDispatch
			clone.NonDispatch = &p
			mutate(&clone)
			r, err := runevidence.ObserveBrowserReport(expected, inventory, f.Report, clone)
			if err == nil || r.SuccessorEligible {
				t.Fatal("partial non-dispatch proof supplied successor authority")
			}
		})
	}
}
func TestBrowserKnownCompletedLeafSurvivesLaterCheckpointFailure(t *testing.T) {
	c, i, report, host := validBrowserReportInputs(t)
	var r udonreport.ReportV5
	if json.Unmarshal(report, &r) != nil {
		t.Fatal("report")
	}
	r.Status = "error"
	r.ErrorCode = "checkpoint_failed"
	changed, _ := json.Marshal(r)
	// The separate checkpoint has no driver message and cannot erase leaf proof.
	index := 0
	for n, e := range host.Trace {
		if e.Event == "join" {
			index = n
			break
		}
	}
	failure := host.Trace[index]
	failure.Event, failure.Outcome = "failure", "checkpoint_failed"
	host.Trace = append(host.Trace[:index], append([]runevidence.BrowserTraceEvent{failure}, host.Trace[index:]...)...)
	result, err := runevidence.ObserveBrowserReport(c, i, changed, host)
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range result.Observation.Steps {
		if step.Outcome != "succeeded" {
			t.Fatal("known completion erased")
		}
	}
	// A lost/malformed response is never a no-effect proof.
	bad := bytes.Replace(changed, []byte(`"workflow_id":"fixture-workflow"`), []byte(`"workflow_id":"foreign"`), 1)
	result, err = runevidence.ObserveBrowserReport(c, i, bad, host)
	if err == nil || result.SuccessorEligible {
		t.Fatal("foreign report supplied authority")
	}
}

func TestBrowserEveryInitialAndContinuationClaimAndHostIdentityIsMandatory(t *testing.T) {
	for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
		"missing initial claim": func(h *runevidence.BrowserHostWitnesses) {
			for i, e := range h.Trace {
				if e.Event == "claim_dispatch" {
					h.Trace = append(h.Trace[:i], h.Trace[i+1:]...)
					break
				}
			}
		},
		"stale message ordinal": func(h *runevidence.BrowserHostWitnesses) {
			for i := range h.Trace {
				if h.Trace[i].Event == "send" {
					h.Trace[i].MessageOrdinal++
					break
				}
			}
		},
		"foreign native request": func(h *runevidence.BrowserHostWitnesses) {
			for i := range h.Trace {
				if h.Trace[i].Event == "send" {
					h.Trace[i].ProtocolRequestID = "other"
					break
				}
			}
		},
		"wrong outer protocol": func(h *runevidence.BrowserHostWitnesses) {
			for i := range h.Trace {
				if h.Trace[i].Event == "send" {
					h.Trace[i].OuterProtocol = "udon.browser-driver.v6"
					break
				}
			}
		},
		"foreign containment": func(h *runevidence.BrowserHostWitnesses) { h.Launch.ContainmentLeaseID = "other" },
		"unjoined callback":   func(h *runevidence.BrowserHostWitnesses) { h.Join.CallbacksJoined = false },
		"fake host refs":      func(h *runevidence.BrowserHostWitnesses) { h.FactRefs = []string{"other"} },
		"retry under old authority": func(h *runevidence.BrowserHostWitnesses) {
			for i, e := range h.Trace {
				if e.Event == "claim_dispatch" {
					denial := e
					denial.Outcome = "refused"
					denial.MessageKind = "refused"
					h.Trace = append(h.Trace[:i], append([]runevidence.BrowserTraceEvent{denial}, h.Trace[i:]...)...)
					break
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, i, report, host := validBrowserReportInputs(t)
			mutate(&host)
			if r, err := runevidence.ObserveBrowserReport(c, i, report, host); err == nil || r.SuccessorEligible {
				t.Fatal("unproved host dispatch accepted")
			}
		})
	}
	var f browserReportFixture
	for _, c := range browserReportFixtures(t) {
		if c.Name == "claimed-push-continuation" {
			f = c
			break
		}
	}
	for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
		"unclaimed continuation": func(h *runevidence.BrowserHostWitnesses) {
			for i, e := range h.Trace {
				if e.Event == "claim_dispatch" && e.MessageKind == "challenge_response" {
					h.Trace = append(h.Trace[:i], h.Trace[i+1:]...)
					break
				}
			}
		},
		"stale issued question": func(h *runevidence.BrowserHostWitnesses) {
			for i, e := range h.Trace {
				if e.Event == "send" && e.MessageKind == "challenge_response" {
					copy := *e.Question
					copy.Revision++
					h.Trace[i].Question = &copy
				}
			}
		},
		"missing current authority recheck": func(h *runevidence.BrowserHostWitnesses) {
			for i, e := range h.Trace {
				if e.Event == "authority_recheck" {
					h.Trace = append(h.Trace[:i], h.Trace[i+1:]...)
					break
				}
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			c, i, host := fixtureInputs(f)
			host.Trace = append([]runevidence.BrowserTraceEvent{}, host.Trace...)
			mutate(&host)
			if r, err := runevidence.ObserveBrowserReport(c, i, f.Report, host); err == nil || r.SuccessorEligible {
				t.Fatal("unproved continuation accepted")
			}
		})
	}
}

func cloneBrowserHost(t *testing.T, h runevidence.BrowserHostWitnesses) runevidence.BrowserHostWitnesses {
	t.Helper()
	raw, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	var copy runevidence.BrowserHostWitnesses
	if json.Unmarshal(raw, &copy) != nil {
		t.Fatal("clone")
	}
	return copy
}
func TestBrowserTerminalUncertaintyCannotBeResurrectedByLaterSuccess(t *testing.T) {
	for _, kind := range []string{"typed_driver_error", "extraction", "response"} {
		t.Run(kind, func(t *testing.T) {
			c, i, report, host := validBrowserReportInputs(t)
			host = cloneBrowserHost(t, host)
			index := -1
			for n, e := range host.Trace {
				if e.Event == "response" && e.OperationID == c.ApprovedCalls[len(c.ApprovedCalls)-1].OperationID {
					index = n
				}
			}
			failure := host.Trace[index]
			failure.Event = kind
			switch kind {
			case "typed_driver_error":
				failure.Outcome = "invalid_response"
			case "extraction":
				failure.Outcome = "failed"
			case "response":
				failure.Outcome = "unknown"
			}
			host.Trace = append(host.Trace[:index], append([]runevidence.BrowserTraceEvent{failure}, host.Trace[index:]...)...)
			if r, err := runevidence.ObserveBrowserReport(c, i, report, host); err == nil || r.SuccessorEligible {
				t.Fatal("terminal uncertainty relabeled succeeded")
			}
		})
	}
}
func TestBrowserOriginalDeadlineCoversEveryStartedLeaf(t *testing.T) {
	for _, at := range []string{"2026-10-09T00:05:00Z", "2026-10-09T00:06:00Z"} {
		t.Run(at, func(t *testing.T) {
			c, i, report, h := validBrowserReportInputs(t)
			var r udonreport.ReportV5
			json.Unmarshal(report, &r)
			last := len(r.Steps) - 1
			r.Steps[last].StartedAt = at
			r.Steps[last].FinishedAt = "2026-10-09T00:06:01Z"
			r.FinishedAt = "2026-10-09T00:06:02Z"
			raw, _ := json.Marshal(r)
			if _, err := udonreport.DecodeV5(raw); err != nil {
				t.Fatal("fixture malformed", err)
			}
			if _, err := runevidence.ObserveBrowserReport(c, i, raw, h); err == nil {
				t.Fatal("late leaf accepted")
			}
		})
	}
	c, i, report, h := validBrowserReportInputs(t)
	var r udonreport.ReportV5
	json.Unmarshal(report, &r)
	r.FinishedAt = "2026-10-09T00:06:00Z"
	raw, _ := json.Marshal(r)
	if _, err := runevidence.ObserveBrowserReport(c, i, raw, h); err != nil {
		t.Fatal("separate later finalization refused", err)
	}
}
func withRuntimeConfirmation(t *testing.T, host runevidence.BrowserHostWitnesses) (runevidence.BrowserHostWitnesses, int) {
	t.Helper()
	host = cloneBrowserHost(t, host)
	index := -1
	for n, e := range host.Trace {
		if e.Event == "claim_dispatch" && e.MessageKind == "action" {
			index = n
			break
		}
	}
	if index < 0 {
		t.Fatal("no action")
	}
	q := &runevidence.BrowserQuestionWitness{ID: "runtime-confirmation", Revision: 1, IssuedSHA256: browsercontract.SHA256([]byte("exact issued confirmation"))}
	event := host.Trace[index]
	event.ProtocolRequestID = ""
	event.MessageOrdinal = 0
	event.MessageKind = ""
	event.OuterProtocol = ""
	event.Question = q
	question := event
	question.Event, question.Outcome = "question", "runtime_confirmation"
	answer := event
	answer.Event, answer.Outcome = "interact", "approve"
	current := event
	current.Event, current.Outcome = "authority_recheck", "current"
	host.Trace[index].Question = q
	host.Trace[index+1].Question = q
	host.Trace = append(host.Trace[:index], append([]runevidence.BrowserTraceEvent{question, answer, current}, host.Trace[index:]...)...)
	return host, index
}
func TestBrowserRuntimeConfirmationIsBoundBeforeInitialActionSend(t *testing.T) {
	c, i, report, host := validBrowserReportInputs(t)
	host, index := withRuntimeConfirmation(t, host)
	if _, err := runevidence.ObserveBrowserReport(c, i, report, host); err != nil {
		t.Fatal("pre-send confirmation refused", err)
	}
	for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
		"changed answer identity": func(h *runevidence.BrowserHostWitnesses) {
			q := *h.Trace[index+1].Question
			q.ID = "other"
			h.Trace[index+1].Question = &q
		},
		"changed recheck revision": func(h *runevidence.BrowserHostWitnesses) {
			q := *h.Trace[index+2].Question
			q.Revision++
			h.Trace[index+2].Question = &q
		},
		"changed claim digest": func(h *runevidence.BrowserHostWitnesses) {
			q := *h.Trace[index+3].Question
			q.IssuedSHA256 = browsercontract.SHA256([]byte("other"))
			h.Trace[index+3].Question = &q
		},
		"deny then old approve": func(h *runevidence.BrowserHostWitnesses) {
			denial := h.Trace[index+1]
			denial.Outcome = "deny"
			h.Trace = append(h.Trace[:index+1], append([]runevidence.BrowserTraceEvent{denial}, h.Trace[index+1:]...)...)
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := cloneBrowserHost(t, host)
			mutate(&copy)
			if _, err := runevidence.ObserveBrowserReport(c, i, report, copy); err == nil {
				t.Fatal("unbound/denied confirmation accepted")
			}
		})
	}
	// Denial stops this attempt but preserves completed authentication. It is not
	// non-dispatch for the whole attempt and supplies no successor authority.
	denied := cloneBrowserHost(t, host)
	denied.Trace[index+1].Outcome = "deny"
	tail := []runevidence.BrowserTraceEvent{}
	for _, e := range denied.Trace[index+2:] {
		if e.Event == "join" || e.Event == "session_release" {
			tail = append(tail, e)
		}
	}
	denied.Trace = append(denied.Trace[:index+2], tail...)
	var r udonreport.ReportV5
	json.Unmarshal(report, &r)
	r.Status = "error"
	r.ErrorCode = "execution_failed"
	last := len(r.Steps) - 1
	r.Steps[last] = udonreport.StepV5{StepID: r.Steps[last].StepID, OperationID: r.Steps[last].OperationID, InvocationID: r.Steps[last].InvocationID, Outcome: "not_started"}
	raw, _ := json.Marshal(r)
	result, err := runevidence.ObserveBrowserReport(c, i, raw, denied)
	if err != nil || result.SuccessorEligible || result.Observation.Steps[0].Outcome != "succeeded" {
		t.Fatal("denial lost known facts", err)
	}
}
func TestBrowserCheckpointFailureStopsLaterDispatch(t *testing.T) {
	c, i, report, host := validBrowserReportInputs(t)
	host = cloneBrowserHost(t, host)
	index := 0
	for n, e := range host.Trace {
		if e.Event == "response" && e.OperationID == c.ApprovedCalls[0].OperationID {
			index = n + 1
			break
		}
	}
	failure := host.Trace[index-1]
	failure.Event, failure.Outcome = "failure", "checkpoint_failed"
	host.Trace = append(host.Trace[:index], append([]runevidence.BrowserTraceEvent{failure}, host.Trace[index:]...)...)
	if _, err := runevidence.ObserveBrowserReport(c, i, report, host); err == nil {
		t.Fatal("checkpoint failure allowed later action")
	}
}
func TestBrowserCredentialLeaseMustBePositiveAndComplete(t *testing.T) {
	var f browserReportFixture
	for _, c := range browserReportFixtures(t) {
		if c.Name == "automatic-totp-seed-environment" {
			f = c
			break
		}
	}
	c, i, h := fixtureInputs(f)
	if _, err := runevidence.ObserveBrowserReport(c, i, f.Report, h); err != nil {
		t.Fatal(err)
	}
	for _, outcome := range []string{"refused", "unknown", "", "partial"} {
		copy := cloneBrowserHost(t, h)
		copy.Trace[0].Outcome = outcome
		if _, err := runevidence.ObserveBrowserReport(c, i, f.Report, copy); err == nil {
			t.Fatal("nonpositive credential lease accepted", outcome)
		}
	}
	for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
		"missing complete lease": func(h *runevidence.BrowserHostWitnesses) { h.CredentialLease = nil },
		"missing slot":           func(h *runevidence.BrowserHostWitnesses) { h.CredentialLease.Slots = nil },
		"foreign revision":       func(h *runevidence.BrowserHostWitnesses) { h.CredentialLease.Revisions[0].Revision = "foreign" },
	} {
		t.Run(name, func(t *testing.T) {
			copy := cloneBrowserHost(t, h)
			mutate(&copy)
			if _, err := runevidence.ObserveBrowserReport(c, i, f.Report, copy); err == nil {
				t.Fatal("incomplete credential union accepted")
			}
		})
	}
}
func TestBrowserOptionalFreshAccessHasNoInventedDurableLease(t *testing.T) {
	for _, name := range []string{"authentication-success", "action-success", "registration-success"} {
		t.Run(name, func(t *testing.T) {
			var f browserReportFixture
			for _, c := range browserReportFixtures(t) {
				if c.Name == name {
					f = c
					break
				}
			}
			c, i, h := fixtureInputs(f)
			c.Session = nil
			h.Config.Session = nil
			h.DurableBindingCount = 0
			h.Join.SessionAccessLeaseID = ""
			trace := []runevidence.BrowserTraceEvent{}
			for _, e := range h.Trace {
				if e.Event == "session_acquire" {
					continue
				}
				e.SessionAccessLeaseID = ""
				trace = append(trace, e)
			}
			h.Trace = trace
			if _, err := runevidence.ObserveBrowserReport(c, i, f.Report, h); err != nil {
				t.Fatal("fresh no-session access refused", err)
			}
		})
	}
}
func TestBrowserSuppliedContinuationAnswerIdentityMustMatchCurrentQuestion(t *testing.T) {
	for _, name := range []string{"claimed-push-continuation", "claimed-private-registration-input-and-submit"} {
		t.Run(name, func(t *testing.T) {
			var f browserReportFixture
			for _, c := range browserReportFixtures(t) {
				if c.Name == name {
					f = c
					break
				}
			}
			c, i, h := fixtureInputs(f)
			for _, kind := range []string{"interact", "authority_recheck", "registration_input"} {
				copy := cloneBrowserHost(t, h)
				found := false
				for n, e := range copy.Trace {
					if e.Event == kind && (kind != "registration_input" || e.Outcome == "checkpoint-private") {
						copy.Trace[n].Question = &runevidence.BrowserQuestionWitness{ID: "foreign", Revision: 2, IssuedSHA256: browsercontract.SHA256([]byte("foreign"))}
						found = true
						break
					}
				}
				if !found {
					continue
				}
				if _, err := runevidence.ObserveBrowserReport(c, i, f.Report, copy); err == nil {
					t.Fatal("foreign supplied answer identity accepted", kind)
				}
			}
		})
	}
}

func TestBrowserObservedLaunchNonceAndClosureCannotFollowExpectedMetadata(t *testing.T) {
	c, i, report, host := validBrowserReportInputs(t)
	for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
		"nonce":          func(h *runevidence.BrowserHostWitnesses) { h.Launch.LaunchNonce = "foreign" },
		"worker profile": func(h *runevidence.BrowserHostWitnesses) { h.Launch.Worker.Profile = "foreign" },
		"worker binary": func(h *runevidence.BrowserHostWitnesses) {
			h.Launch.Worker.BinarySHA256 = browsercontract.SHA256([]byte("foreign"))
		},
		"worker closure": func(h *runevidence.BrowserHostWitnesses) {
			h.Launch.Worker.ClosureSHA256 = browsercontract.SHA256([]byte("foreign"))
		},
		"runtime revision": func(h *runevidence.BrowserHostWitnesses) {
			h.Launch.Worker.RuntimeRevision = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		},
		"driver closure": func(h *runevidence.BrowserHostWitnesses) {
			h.Launch.DriverClosureSHA256 = browsercontract.SHA256([]byte("foreign"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := host
			mutate(&copy)
			if _, err := runevidence.ObserveBrowserReport(c, i, report, copy); err == nil {
				t.Fatal("foreign actual launch accepted")
			}
		})
	}
	c.LaunchNonce = "foreign-requested-nonce"
	host.Config.LaunchNonce = c.LaunchNonce
	if _, err := runevidence.ObserveBrowserReport(c, i, report, host); err == nil {
		t.Fatal("expected metadata replaced independent launch facts")
	}
}

func TestBrowserDurableAccessIsObservedAndAcquirePrecedesLaunch(t *testing.T) {
	var fixture browserReportFixture
	for _, c := range browserReportFixtures(t) {
		if c.Name == "joined-durable-session-save" {
			fixture = c
		}
	}
	c, i, host := fixtureInputs(fixture)
	if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, host); err != nil {
		t.Fatal("positive observed access", err)
	}
	for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
		"missing": func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess = nil },
		"lease":   func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.LeaseID = "foreign" },
		"binding": func(h *runevidence.BrowserHostWitnesses) {
			h.SessionAccess.Session.BindingSHA256 = browsercontract.SHA256([]byte("foreign"))
		},
		"generation":         func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.Session.Generation++ },
		"name":               func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.Session.Name = "foreign" },
		"creation":           func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.Session.CreatedAt = "2026-10-09T00:01:00Z" },
		"permission":         func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.Session.SaveAllowed = false },
		"refused":            func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.Outcome = "refused" },
		"no durable binding": func(h *runevidence.BrowserHostWitnesses) { h.DurableBindingCount = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			copy := host
			access := *host.SessionAccess
			copy.SessionAccess = &access
			mutate(&copy)
			if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, copy); err == nil {
				t.Fatal("unproved durable access accepted")
			}
		})
	}
	copy := host
	copy.Trace = append([]runevidence.BrowserTraceEvent{}, host.Trace...)
	copy.Trace[0], copy.Trace[1] = copy.Trace[1], copy.Trace[0]
	if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, copy); err == nil {
		t.Fatal("Acquire after Launch accepted")
	}
	copy = host
	copy.Trace = append([]runevidence.BrowserTraceEvent{host.Trace[0]}, host.Trace...)
	if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, copy); err == nil {
		t.Fatal("extra positive access acquisition accepted")
	}
	// A current host may narrow the unchanged approved save permission during
	// missing/expired -> fresh. No candidate or durable save may then occur.
	for _, outcome := range []string{"missing", "expired"} {
		t.Run("narrowed "+outcome+" to fresh", func(t *testing.T) {
			copy := host
			access := *host.SessionAccess
			access.Session.SaveAllowed = false
			copy.SessionAccess = &access
			copy.SavePermission = nil
			before := host.Trace[0]
			before.Outcome = outcome
			copy.Trace = []runevidence.BrowserTraceEvent{before}
			for _, e := range host.Trace {
				if e.Event != "candidate" && e.Event != "generation_recheck" && e.Event != "session_save" {
					copy.Trace = append(copy.Trace, e)
				}
			}
			if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, copy); err != nil {
				t.Fatal("positive permission narrowing/fallback refused", err)
			}
		})
	}
	fresh, inventory, report, freshHost := validBrowserReportInputs(t)
	freshHost.SessionAccess = nil
	if _, err := runevidence.ObserveBrowserReport(fresh, inventory, report, freshHost); err != nil {
		t.Fatal("fresh-only access made mandatory", err)
	}
	freshHost.Trace = append([]runevidence.BrowserTraceEvent{}, freshHost.Trace...)
	freshHost.Trace[0].Outcome = "expired"
	if _, err := runevidence.ObserveBrowserReport(fresh, inventory, report, freshHost); err == nil {
		t.Fatal("unresolved expired acquisition reached Launch")
	}
}

func TestBrowserDeniedInflightLeafCannotUpgradeToSuccess(t *testing.T) {
	var fixture browserReportFixture
	for _, c := range browserReportFixtures(t) {
		if c.Name == "claimed-push-continuation" {
			fixture = c
		}
	}
	c, i, host := fixtureInputs(fixture)
	trace := []runevidence.BrowserTraceEvent{}
	var initial runevidence.BrowserTraceEvent
	for _, e := range host.Trace {
		if e.Event == "send" && e.Outcome == "authentication" {
			initial = e
		}
		if e.Event == "authority_recheck" || (e.Event == "claim_dispatch" || e.Event == "send") && e.Outcome == "mfa_push" {
			continue
		}
		if e.Event == "interact" {
			e.Outcome = "deny"
		}
		if e.Event == "response" {
			e = initial
			e.Event, e.Outcome = "response", "success"
		}
		trace = append(trace, e)
	}
	host.Trace = trace
	if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, host); err == nil {
		t.Fatal("denial upgraded unknown in-flight leaf to success")
	}
	withoutResponse := []runevidence.BrowserTraceEvent{}
	for _, e := range trace {
		if e.Event != "response" {
			withoutResponse = append(withoutResponse, e)
		}
	}
	host.Trace = withoutResponse
	var report udonreport.ReportV5
	json.Unmarshal(fixture.Report, &report)
	report.Status, report.FinishedAt = "incomplete", ""
	report.Steps[0].Outcome, report.Steps[0].FinishedAt = "unknown", ""
	raw, _ := json.Marshal(report)
	result, err := runevidence.ObserveBrowserReport(c, i, raw, host)
	if err != nil || result.Observation.Steps[0].Outcome != "unknown" || result.SuccessorEligible {
		t.Fatal("denied in-flight uncertainty control", err)
	}
}

func TestBrowserEvidenceSerializerRejectsPrivateAndUnclosedMetadata(t *testing.T) {
	c, i, report, host := validBrowserReportInputs(t)
	result, err := runevidence.ObserveBrowserReport(c, i, report, host)
	if err != nil {
		t.Fatal(err)
	}
	e := runevidence.BrowserRunEvidenceV1{Version: browsercontract.RunEvidenceVersion, RunID: c.RunID, CreatedAt: "2026-10-09T00:00:03Z", Scope: "workflows/W01-fixture", Tier: "sandbox", ApprovalState: "approved_for_sandbox", Browser: c, Executor: runevidence.BrowserExecutorV1{Invoked: true, Mode: "native", ReportSHA256: browsercontract.SHA256(report), ReportSize: int64(len(report))}, StepExecution: &result.Observation}
	original, _ := json.Marshal(e)
	for name, mutate := range map[string]func(*runevidence.BrowserRunEvidenceV1){
		"private effects": func(e *runevidence.BrowserRunEvidenceV1) {
			e.Browser.ApprovedCalls[0].Effects = json.RawMessage(`{"private_registration_input":"raw-private-value"}`)
		},
		"version":       func(e *runevidence.BrowserRunEvidenceV1) { e.Version = "foreign" },
		"run":           func(e *runevidence.BrowserRunEvidenceV1) { e.RunID = "foreign" },
		"scope":         func(e *runevidence.BrowserRunEvidenceV1) { e.Scope = "/private/path" },
		"time":          func(e *runevidence.BrowserRunEvidenceV1) { e.CreatedAt = "private-value" },
		"before report": func(e *runevidence.BrowserRunEvidenceV1) { e.CreatedAt = "2026-10-09T00:00:01Z" },
		"tier":          func(e *runevidence.BrowserRunEvidenceV1) { e.Tier = "production" },
		"approval":      func(e *runevidence.BrowserRunEvidenceV1) { e.ApprovalState = "unapproved" },
		"executor":      func(e *runevidence.BrowserRunEvidenceV1) { e.Executor.Mode = "private-value" },
		"observation":   func(e *runevidence.BrowserRunEvidenceV1) { e.StepExecution.ErrorCode = "private-value" },
	} {
		t.Run(name, func(t *testing.T) {
			var copy runevidence.BrowserRunEvidenceV1
			json.Unmarshal(original, &copy)
			mutate(&copy)
			if raw, err := runevidence.MarshalBrowserRunEvidence(copy); err == nil || len(raw) != 0 {
				t.Fatal("unclosed/private metadata serialized")
			}
		})
	}
	if _, err := runevidence.MarshalBrowserRunEvidence(e); err != nil {
		t.Fatal("native positive serializer", err)
	}
	dry := udonreport.UnknownV5(i, "dry_run")
	e.DryRun, e.StepExecution, e.Executor = true, &dry, runevidence.BrowserExecutorV1{Mode: "dry_run"}
	if _, err := runevidence.MarshalBrowserRunEvidence(e); err != nil {
		t.Fatal("dry positive serializer", err)
	}
}

func TestBrowserObservedReuseAndNarrowedFreshFallback(t *testing.T) {
	fixture := browserReportFixtures(t)[0]
	c, i, host := fixtureInputs(fixture)
	// Explicit revised fixture-only reviewed permission; no actual saved state
	// or human authority is created by this metadata control.
	session := *c.Session
	session.ReuseAllowed = true
	c.Session = &session
	c.ApprovedCalls[0].ReuseAllowed = true
	host.Config = c
	access := *host.SessionAccess
	access.Session, access.Outcome = session, "reuse"
	host.SessionAccess = &access
	host.Trace = append([]runevidence.BrowserTraceEvent{}, host.Trace...)
	host.Trace[0].Outcome = "reuse"
	if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, host); err != nil {
		t.Fatal("positive observed reuse", err)
	}
	bad := host
	badAccess := access
	badAccess.Session.ReuseAllowed = false
	bad.SessionAccess = &badAccess
	if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, bad); err == nil {
		t.Fatal("actual reuse without observed reuse permission")
	}
	access.Session.ReuseAllowed, access.Outcome = false, "fresh"
	missing, fresh := host.Trace[0], host.Trace[0]
	missing.Outcome, fresh.Outcome = "missing", "fresh"
	host.Trace = append([]runevidence.BrowserTraceEvent{missing, fresh}, host.Trace[1:]...)
	if _, err := runevidence.ObserveBrowserReport(c, i, fixture.Report, host); err != nil {
		t.Fatal("narrowed reuse-to-fresh permission", err)
	}
}
