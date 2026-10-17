package runevidence_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/runevidence"
	"github.com/OpenUdon/openudon/udonreport"
)

func savedFallbackFixture(t *testing.T, outcome string) (browsercontract.BrowserConfigV1, udonreport.InventoryV5, []byte, runevidence.BrowserHostWitnesses) {
	t.Helper()
	var fixture browserReportFixture
	for _, f := range browserReportFixtures(t) {
		if f.Name == "joined-durable-session-save" {
			fixture = f
		}
	}
	c, i, h := fixtureInputs(fixture)
	// A fixture-only review permits both rights; actual native fallback must
	// attenuate both without rewriting this original configuration or binding.
	s := *c.Session
	s.ReuseAllowed = true
	c.Session = &s
	c.ApprovedCalls[0].ReuseAllowed = true
	h.Config = c
	setObservedSessionTimes(&c, &h, "2026-10-08T23:59:00Z", "2026-10-09T00:00:00Z")
	h.SessionAccess.Outcome = "fresh"
	h.SessionAccess.Session.ReuseAllowed, h.SessionAccess.Session.SaveAllowed = false, false
	h.SavePermission = nil
	failed := h.Trace[0]
	failed.Outcome = outcome
	trace := []runevidence.BrowserTraceEvent{failed}
	for _, e := range h.Trace {
		if e.Event != "candidate" && e.Event != "generation_recheck" && e.Event != "session_save" {
			trace = append(trace, e)
		}
	}
	h.Trace = trace
	return c, i, fixture.Report, h
}

// Use the unchanged report's observation to construct evidence independently of
// the host under test. Both public entry points must reject dishonest witnesses.
func checkFallbackReportAndEvidence(t *testing.T, c browsercontract.BrowserConfigV1, i udonreport.InventoryV5, report []byte, h runevidence.BrowserHostWitnesses, want bool) {
	t.Helper()
	before, _ := json.Marshal(c)
	observation := udonreport.ObserveV5(i, report)
	if observation.State != "validated" {
		t.Fatal("fixture report is invalid")
	}
	digest, err := c.Digest()
	if err != nil {
		t.Fatal(err)
	}
	a := browsercontract.BrowserAuthorityV1{Version: browsercontract.AuthorityVersion, ConfigSHA256: digest, Config: c}
	e := runevidence.BrowserRunEvidenceV1{Version: browsercontract.RunEvidenceVersion, RunID: c.RunID, CreatedAt: "2026-10-09T00:00:03Z", Scope: "workflows/W01-fixture", Tier: "sandbox", ApprovalState: "approved_for_sandbox", Browser: c, Executor: runevidence.BrowserExecutorV1{Invoked: true, Mode: "native", ReportSHA256: browsercontract.SHA256(report), ReportSize: int64(len(report))}, StepExecution: &observation}
	raw, err := runevidence.MarshalBrowserRunEvidence(e)
	if err != nil {
		t.Fatal(err)
	}
	result, observeErr := runevidence.ObserveBrowserReport(c, i, report, h)
	decoded, verified, verifyErr := runevidence.VerifyBrowserRunEvidence(raw, runevidence.BrowserVerifyOptions{ExpectedConfig: c, Authority: a, Inventory: i, Scope: e.Scope, ReportJSON: report, Host: h})
	if (observeErr == nil) != want || (verifyErr == nil) != want {
		t.Fatalf("Observe valid=%v Verify valid=%v want=%v: %v / %v", observeErr == nil, verifyErr == nil, want, observeErr, verifyErr)
	}
	if want {
		decodedDigest, _ := decoded.Browser.Digest()
		if !reflect.DeepEqual(result.Observation, observation) || !reflect.DeepEqual(verified, result) || decodedDigest != digest || result.SuccessorEligible {
			t.Fatal("fallback changed approved metadata or observed execution")
		}
	} else if result.Observation.State != "invalid" || verified.Observation.State != "invalid" || result.SuccessorEligible || verified.SuccessorEligible {
		t.Fatal("refused fallback supplied facts or successor eligibility")
	}
	after, _ := json.Marshal(c)
	if !bytes.Equal(before, after) {
		t.Fatal("verification mutated immutable configuration")
	}
}

func TestBrowserSavedFallbackRequiresZeroObservedRights(t *testing.T) {
	for _, outcome := range []string{"missing", "expired"} {
		t.Run(outcome, func(t *testing.T) {
			c, i, report, h := savedFallbackFixture(t, outcome)
			checkFallbackReportAndEvidence(t, c, i, report, h, true)
			for _, rights := range []struct {
				name        string
				reuse, save bool
			}{
				{"retained reuse", true, false}, {"retained save", false, true}, {"retained both", true, true},
			} {
				t.Run(rights.name, func(t *testing.T) {
					bad := cloneBrowserHost(t, h)
					bad.SessionAccess.Session.ReuseAllowed, bad.SessionAccess.Session.SaveAllowed = rights.reuse, rights.save
					checkFallbackReportAndEvidence(t, c, i, report, bad, false)
				})
			}
		})
	}
}

func TestBrowserSavedFallbackRejectsBrokenLifecycleAndBinding(t *testing.T) {
	for _, outcome := range []string{"missing", "expired"} {
		t.Run(outcome, func(t *testing.T) {
			c, i, report, h := savedFallbackFixture(t, outcome)
			for name, mutate := range map[string]func(*runevidence.BrowserHostWitnesses){
				"missing acquisition": func(h *runevidence.BrowserHostWitnesses) { h.Trace = h.Trace[2:] },
				"missing release":     func(h *runevidence.BrowserHostWitnesses) { h.Trace = h.Trace[:len(h.Trace)-1] },
				"missing join":        func(h *runevidence.BrowserHostWitnesses) { removeFallbackEvent(h, "join") },
				"unjoined callbacks":  func(h *runevidence.BrowserHostWitnesses) { h.Join.CallbacksJoined = false },
				"post-send fallback": func(h *runevidence.BrowserHostWitnesses) {
					for n, e := range h.Trace {
						if e.Event == "send" {
							h.Trace = append(h.Trace[:n+1], append([]runevidence.BrowserTraceEvent{h.Trace[0], h.Trace[1]}, h.Trace[n+1:]...)...)
							break
						}
					}
				},
				"changed binding": func(h *runevidence.BrowserHostWitnesses) {
					h.SessionAccess.Session.BindingSHA256 = browsercontract.SHA256([]byte("foreign"))
				},
				"changed native binding": func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.Binding.Generation++ },
				"changed expiry":         func(h *runevidence.BrowserHostWitnesses) { h.SessionAccess.Session.ExpiresAt = "2026-10-10T00:00:00Z" },
				"candidate on fallback": func(h *runevidence.BrowserHostWitnesses) {
					insertFallbackEvent(h, "join", "candidate", "staged-before-close")
				},
				"save on fallback": func(h *runevidence.BrowserHostWitnesses) {
					insertFallbackEvent(h, "session_release", "session_save", "encrypted-accepted")
				},
			} {
				t.Run(name, func(t *testing.T) {
					bad := cloneBrowserHost(t, h)
					mutate(&bad)
					checkFallbackReportAndEvidence(t, c, i, report, bad, false)
				})
			}
		})
	}
}

func removeFallbackEvent(h *runevidence.BrowserHostWitnesses, event string) {
	trace := []runevidence.BrowserTraceEvent{}
	for _, e := range h.Trace {
		if e.Event != event {
			trace = append(trace, e)
		}
	}
	h.Trace = trace
}

func insertFallbackEvent(h *runevidence.BrowserHostWitnesses, before, event, outcome string) {
	for n, e := range h.Trace {
		if e.Event == before {
			insert := h.Trace[0]
			insert.Event, insert.Outcome = event, outcome
			h.Trace = append(h.Trace[:n], append([]runevidence.BrowserTraceEvent{insert}, h.Trace[n:]...)...)
			return
		}
	}
}
