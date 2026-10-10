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
		Binding     browserBindingFixture                     `json:"binding"`
		Launch      runevidence.BrowserLaunchWitness          `json:"launch"`
		Join        runevidence.BrowserJoinWitness            `json:"join"`
		Save        *runevidence.BrowserSavePermissionWitness `json:"save_permission"`
		NonDispatch *struct {
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
	if c.Host.Binding.Session != nil {
		host.DurableBindingCount = 1
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
