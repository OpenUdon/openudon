package runevidence_test

import (
	"encoding/json"
	"testing"

	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/runevidence"
	"github.com/OpenUdon/openudon/udonreport"
)

func TestBrowserOptionalActualAccessMatchesAcquiringLeaf(t *testing.T) {
	var fixture browserReportFixture
	for _, f := range browserReportFixtures(t) {
		if f.Name == "registration-then-separate-authentication" {
			fixture = f
		}
	}
	c, inventory, host := fixtureInputs(fixture)
	// The original fresh-context metadata acquires under unnamed registration.
	// Supplying an actual authentication-bound access for that leaf is invalid,
	// even with zero durable permissions. This is an explicit adverse witness,
	// independent of the normal fixture adapter's nil native-access projection.
	actual := fixtureNativeSessionBinding(fixture.Host.Binding)
	expected := fixtureNativeSessionBinding(fixture.Expected)
	host.SessionAccess = &runevidence.BrowserSessionAccessWitness{LeaseID: fixture.Host.Join.SessionAccessLeaseID, Outcome: "fresh", Session: *fixture.Host.Binding.Session, Binding: actual}
	host.ExpectedSessionBinding = &expected
	host.DurableBindingCount = 1
	host.Join = fixture.Host.Join
	host.Trace = append([]runevidence.BrowserTraceEvent{}, fixture.Trace...)
	if c.Validate() != nil || udonreport.ObserveV5(inventory, fixture.Report).State != "validated" {
		t.Fatal("adverse witness setup changed the valid config/report")
	}
	if _, err := runevidence.ObserveBrowserReport(c, inventory, fixture.Report, host); err == nil {
		t.Fatal("optional actual authentication access acquired under another leaf")
	}
	c, inventory, report, positive := validBrowserReportInputs(t)
	if _, err := runevidence.ObserveBrowserReport(c, inventory, report, positive); err != nil {
		t.Fatal("exact named fresh access refused", err)
	}
}

func TestBrowserRefusedContinuationCannotUpgradeInflightUncertainty(t *testing.T) {
	var fixture browserReportFixture
	for _, f := range browserReportFixtures(t) {
		if f.Name == "claimed-push-continuation" {
			fixture = f
		}
	}
	c, inventory, host := fixtureInputs(fixture)
	var initial runevidence.BrowserTraceEvent
	trace := []runevidence.BrowserTraceEvent{}
	for _, e := range host.Trace {
		if e.Event == "send" && e.Outcome == "authentication" {
			initial = e
		}
		if e.Event == "claim_dispatch" && e.Outcome == "mfa_push" {
			e.Outcome = "refused"
		}
		if e.Event == "send" && e.Outcome == "mfa_push" {
			continue
		}
		if e.Event == "response" {
			e = initial
			e.Event, e.Outcome = "response", "success"
		}
		trace = append(trace, e)
	}
	host.Trace = trace
	if _, err := runevidence.ObserveBrowserReport(c, inventory, fixture.Report, host); err == nil {
		t.Fatal("refused continuation upgraded the initial in-flight send to success")
	}
	withoutResponse := []runevidence.BrowserTraceEvent{}
	for _, e := range trace {
		if e.Event != "response" {
			withoutResponse = append(withoutResponse, e)
		}
	}
	host.Trace = withoutResponse
	var report udonreport.ReportV5
	if json.Unmarshal(fixture.Report, &report) != nil {
		t.Fatal("report setup")
	}
	report.Status, report.FinishedAt = "incomplete", ""
	report.Steps[0].Outcome, report.Steps[0].FinishedAt = "unknown", ""
	raw, _ := json.Marshal(report)
	result, err := runevidence.ObserveBrowserReport(c, inventory, raw, host)
	if err != nil || result.Observation.Steps[0].Outcome != "unknown" || result.SuccessorEligible {
		t.Fatal("refused continuation lost conservative uncertainty", err)
	}
}

func TestBrowserStoppingAnotherLeafCannotUpgradeCurrentUncertainty(t *testing.T) {
	for _, reason := range []string{"refused future leaf", "checkpoint of completed leaf"} {
		t.Run(reason, func(t *testing.T) {
			c, inventory, originalReport, host := validBrowserReportInputs(t)
			var report udonreport.ReportV5
			if json.Unmarshal(originalReport, &report) != nil {
				t.Fatal("report setup")
			}
			trace := []runevidence.BrowserTraceEvent{}
			current := 1
			if reason == "refused future leaf" {
				current = 0
				refused := host.Trace[0]
				refused.Event, refused.Outcome = "claim_dispatch", "refused"
				refused.StepID, refused.OperationID, refused.InvocationID = c.ApprovedCalls[1].StepID, c.ApprovedCalls[1].OperationID, c.ApprovedCalls[1].InvocationID
				for _, e := range host.Trace {
					if e.StepID == c.ApprovedCalls[1].StepID && e.Event != "join" && e.Event != "session_release" {
						continue
					}
					trace = append(trace, e)
					if e.Event == "send" && e.StepID == c.ApprovedCalls[0].StepID {
						trace = append(trace, refused)
					}
				}
				report.Status, report.FinishedAt = "incomplete", ""
				report.Steps[1].Outcome, report.Steps[1].StartedAt, report.Steps[1].FinishedAt = "not_started", "", ""
			} else {
				failure := host.Trace[0]
				failure.Event, failure.Outcome = "failure", "checkpoint_failed"
				for _, e := range host.Trace {
					if e.Event == "response" && e.StepID == c.ApprovedCalls[1].StepID {
						trace = append(trace, failure)
					}
					trace = append(trace, e)
				}
				report.Status, report.ErrorCode = "error", "checkpoint_failed"
			}
			host.Trace = trace
			raw, _ := json.Marshal(report)
			if udonreport.ObserveV5(inventory, raw).State != "validated" {
				t.Fatal("counterexample report setup")
			}
			if _, err := runevidence.ObserveBrowserReport(c, inventory, raw, host); err == nil {
				t.Fatal("stop concerning another leaf accepted the current late success")
			}
			withoutResponse := []runevidence.BrowserTraceEvent{}
			for _, e := range trace {
				if e.Event != "response" || e.StepID != c.ApprovedCalls[current].StepID {
					withoutResponse = append(withoutResponse, e)
				}
			}
			host.Trace = withoutResponse
			report.Steps[current].Outcome, report.Steps[current].FinishedAt = "unknown", ""
			raw, _ = json.Marshal(report)
			result, err := runevidence.ObserveBrowserReport(c, inventory, raw, host)
			if err != nil || result.Observation.Steps[current].Outcome != "unknown" || result.SuccessorEligible {
				t.Fatal("current uncertainty was lost", err)
			}
			if current == 1 && result.Observation.Steps[0].Outcome != "succeeded" {
				t.Fatal("checkpoint erased the earlier completed authentication")
			}
		})
	}
}

func deniedActionSaveFixture(t *testing.T, candidateBeforeDenial bool) (browsercontract.BrowserConfigV1, udonreport.InventoryV5, []byte, runevidence.BrowserHostWitnesses) {
	t.Helper()
	c, inventory, raw, host := validBrowserReportInputs(t)
	s := *c.Session
	s.SaveAllowed = true
	c.Session = &s
	c.ApprovedCalls[0].SaveAllowed = true
	host.Config = c
	host.SessionAccess.Session = s
	host.SavePermission = &runevidence.BrowserSavePermissionWitness{PermissionID: "fixture-save-denial", SessionName: s.Name, BindingSHA256: s.BindingSHA256, Generation: s.Generation, AuthenticateOperationID: c.ApprovedCalls[0].OperationID, OneUse: true, SaveAllowed: true}
	authEvent := host.Trace[0]
	candidate := authEvent
	candidate.Event, candidate.Outcome = "candidate", "staged-before-close"
	actionEvent := authEvent
	actionEvent.StepID, actionEvent.OperationID, actionEvent.InvocationID = c.ApprovedCalls[1].StepID, c.ApprovedCalls[1].OperationID, c.ApprovedCalls[1].InvocationID
	question := actionEvent
	question.Event, question.Outcome = "question", "runtime_confirmation"
	question.Question = &runevidence.BrowserQuestionWitness{ID: "fixture-denied-action", Revision: 1, IssuedSHA256: browsercontract.SHA256([]byte("fixture denied action question"))}
	denial := question
	denial.Event, denial.Outcome = "interact", "deny"
	check, save := authEvent, authEvent
	check.Event, check.Outcome = "generation_recheck", "current-3"
	save.Event, save.Outcome = "session_save", "encrypted-accepted"
	trace := []runevidence.BrowserTraceEvent{}
	for _, e := range host.Trace {
		if e.StepID == c.ApprovedCalls[1].StepID && e.Event != "join" && e.Event != "session_release" {
			continue
		}
		trace = append(trace, e)
		if e.Event == "response" {
			if candidateBeforeDenial {
				trace = append(trace, candidate)
			}
			trace = append(trace, question, denial)
			if !candidateBeforeDenial {
				trace = append(trace, candidate)
			}
		}
		if e.Event == "join" {
			trace = append(trace, check, save)
		}
	}
	host.Trace = trace
	var report udonreport.ReportV5
	if json.Unmarshal(raw, &report) != nil {
		t.Fatal("report setup")
	}
	report.Status, report.FinishedAt = "incomplete", ""
	report.Steps[1].Outcome, report.Steps[1].StartedAt, report.Steps[1].FinishedAt = "not_started", "", ""
	raw, _ = json.Marshal(report)
	if c.Validate() != nil || udonreport.ObserveV5(inventory, raw).State != "validated" {
		t.Fatal("denial fixture has invalid config/report")
	}
	return c, inventory, raw, host
}

func TestBrowserExplicitDenialWithholdsCandidateAndDurableSave(t *testing.T) {
	for _, candidateBefore := range []bool{true, false} {
		c, inventory, report, host := deniedActionSaveFixture(t, candidateBefore)
		if _, err := runevidence.ObserveBrowserReport(c, inventory, report, host); err == nil {
			t.Fatalf("explicit denial accepted candidate/save; candidate before denial=%v", candidateBefore)
		}
		withoutSave := []runevidence.BrowserTraceEvent{}
		denied := false
		for _, e := range host.Trace {
			denied = denied || e.Event == "interact" && e.Outcome == "deny"
			if e.Event != "session_save" && !(denied && e.Event == "candidate") {
				withoutSave = append(withoutSave, e)
			}
		}
		host.Trace = withoutSave
		result, err := runevidence.ObserveBrowserReport(c, inventory, report, host)
		if err != nil || result.Observation.Steps[0].Outcome != "succeeded" || result.Observation.Steps[1].Outcome != "not_started" {
			t.Fatal("denial lost prior authentication or invented action dispatch", err)
		}
	}
	// An unrelated checkpoint failure preserves a completed authentication's
	// existing one-use save permission. Explicit denial is the withholding rule.
	c, inventory, report, host := deniedActionSaveFixture(t, true)
	trace := []runevidence.BrowserTraceEvent{}
	for _, e := range host.Trace {
		if e.Event == "question" {
			continue
		}
		if e.Event == "interact" {
			e.Event, e.Outcome, e.Question = "failure", "checkpoint_failed", nil
			e.StepID, e.OperationID, e.InvocationID = c.ApprovedCalls[0].StepID, c.ApprovedCalls[0].OperationID, c.ApprovedCalls[0].InvocationID
		}
		trace = append(trace, e)
	}
	host.Trace = trace
	if _, err := runevidence.ObserveBrowserReport(c, inventory, report, host); err != nil {
		t.Fatal("unrelated failure made existing save permission unusable", err)
	}
}
