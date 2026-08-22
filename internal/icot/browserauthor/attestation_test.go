package browserauthor

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
)

func TestVerifyAttestationRejectsFabricatedEnvelopeFacts(t *testing.T) {
	a, envelope := validAttestationEnvelope(t)
	if err := VerifyAttestation(a, envelope); err != nil {
		t.Fatalf("valid envelope: %v", err)
	}
	childOwned := cloneEnvelope(t, envelope)
	childOwned.Trace[0].POSTObserved = 1
	childOwned.OutputSelections[0].RoleMatches = 2
	if err := VerifyAttestation(a, childOwned); err != nil {
		t.Fatalf("bounded child-owned facts were rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*authorresult.Envelope)
	}{
		{name: "insert trace", mutate: func(value *authorresult.Envelope) { value.Trace = append(value.Trace, value.Trace[0]) }},
		{name: "remove trace", mutate: func(value *authorresult.Envelope) { value.Trace = value.Trace[:1] }},
		{name: "reorder trace", mutate: func(value *authorresult.Envelope) { value.Trace[0], value.Trace[1] = value.Trace[1], value.Trace[0] }},
		{name: "authentication locator", mutate: func(value *authorresult.Envelope) {
			value.AuthenticationProfile = []byte(strings.Replace(string(value.AuthenticationProfile), `"name":"Dashboard"`, `"name":"Admin"`, 1))
		}},
		{name: "output candidate", mutate: func(value *authorresult.Envelope) {
			value.OutputSelections[0].CandidateID = "candidate-ffffffffffffffff"
		}},
		{name: "output key", mutate: func(value *authorresult.Envelope) { value.OutputSelections[0].Key = "invented" }},
		{name: "origin ledger", mutate: func(value *authorresult.Envelope) {
			value.Origins = append(value.Origins, "https://other.example.test")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			altered := cloneEnvelope(t, envelope)
			test.mutate(altered)
			if err := VerifyAttestation(a, altered); err == nil {
				t.Fatal("fabricated envelope was accepted")
			}
		})
	}
}

func TestOriginApprovalBindsExactPendingAction(t *testing.T) {
	a, _ := validAttestationEnvelope(t)
	a.pendingAction = &authorsession.ClientMessage{Type: "execute", Action: "navigate_get", URL: "https://other.example.test/next", Context: "main"}
	approval := authorsession.Approval{ID: "approval-0001", Kind: "origin", Origin: "https://other.example.test", Action: "navigate_get"}
	if !safeApproval(approval, a.originLedger()) {
		t.Fatal("canonical new-origin approval was rejected")
	}
	if err := a.recordApproval(approval); err != nil {
		t.Fatal(err)
	}
	if !contains(a.originLedger(), approval.Origin) {
		t.Fatal("approved origin was not added to the parent ledger")
	}
	for _, stale := range []authorsession.Approval{
		{ID: "approval-0002", Kind: "origin", Origin: "https://stale.example.test", Action: "navigate_get"},
		{ID: "approval-0003", Kind: "origin", Origin: "https://other.example.test", Action: "click"},
		{ID: "approval-0001", Kind: "origin", Origin: "https://other.example.test", Action: "navigate_get"},
	} {
		if err := a.recordApproval(stale); err == nil {
			t.Fatalf("stale or mismatched approval was accepted: %#v", stale)
		}
	}
}

func TestAttestationHasNoSerializableState(t *testing.T) {
	a, _ := validAttestationEnvelope(t)
	data, err := json.Marshal(a)
	if err != nil || string(data) != "{}" {
		t.Fatalf("attestation serialization = %q, %v", data, err)
	}
	event, err := json.Marshal(Event{State: "completion_review", Attestation: a})
	if err != nil || strings.Contains(string(event), "attestation") || strings.Contains(string(event), "Dashboard") {
		t.Fatalf("event exposed attestation: %s, %v", event, err)
	}
}

func validAttestationEnvelope(t *testing.T) (*Attestation, *authorresult.Envelope) {
	t.Helper()
	predicate := authorresult.GoalPredicate{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard"}
	bounds := authorresult.Bounds{NavigationTimeoutMS: 20_000, TotalTimeoutMS: 600_000, MaxRequests: 512, MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128, MaxOutputs: 16}
	a, err := newAttestation(Config{DashboardURL: "https://members.example.test/dashboard", Goal: "review dashboard", Origins: []string{"https://members.example.test"}, GoalPredicate: predicate}, bounds)
	if err != nil {
		t.Fatal(err)
	}
	login := authorsession.Observation{Origin: predicate.Origin, Path: "/login", Context: "main", Contexts: map[string]authorresult.Context{}, Candidates: []authorsession.Candidate{{ID: "candidate-1111111111111111", Role: "button", Label: "Sign in", Matches: 1}}}
	if err := a.recordObservation("authentication", login); err != nil {
		t.Fatal(err)
	}
	if err := a.recordClient(authorsession.ClientMessage{Type: "execute", Action: "click", CandidateID: login.Candidates[0].ID, POSTBudget: 2}, "authentication", &login); err != nil {
		t.Fatal(err)
	}
	dashboard := authorsession.Observation{Origin: predicate.Origin, Path: predicate.Path, Context: "main", Contexts: map[string]authorresult.Context{}, Candidates: []authorsession.Candidate{{ID: "candidate-2222222222222222", Role: "heading", Label: "Dashboard", Matches: 1}}}
	if err := a.recordObservation("authentication", dashboard); err != nil {
		t.Fatal(err)
	}
	if err := a.recordClient(authorsession.ClientMessage{Type: "execute", Action: "navigate_get", URL: "https://members.example.test/dashboard", Context: "main"}, "exploration", &dashboard); err != nil {
		t.Fatal(err)
	}
	if err := a.recordObservation("exploration", dashboard); err != nil {
		t.Fatal(err)
	}
	requests := []authorsession.OutputRequest{{CandidateID: dashboard.Candidates[0].ID, Key: "dashboard_title", Type: "string", LocatorMode: "exact_name"}}
	if err := a.recordClient(authorsession.ClientMessage{Type: "human_complete", Confirmed: true, Outputs: &requests}, "exploration", &dashboard); err != nil {
		t.Fatal(err)
	}
	authentication := json.RawMessage(`{"flows":{"authenticated_goal":{"success":{"origin":"https://members.example.test","path":"/dashboard","context":"main","locator":{"role":"heading","name":"Dashboard"}}}}}`)
	return a, &authorresult.Envelope{
		Goal: "review dashboard", Origins: []string{"https://members.example.test"}, Contexts: map[string]authorresult.Context{}, Bounds: bounds,
		Trace: []authorresult.TraceStep{
			{Kind: "click", Phase: "authentication", CandidateID: login.Candidates[0].ID, Context: "main", Role: "button", Label: "Sign in", POSTBudget: 2},
			{Kind: "navigate", Phase: "exploration", Context: "main", URL: "https://members.example.test/dashboard"},
		},
		GoalPredicate: predicate, GoalProof: authorresult.GoalProof{Origin: predicate.Origin, Path: predicate.Path, Context: "main", Role: "heading", Label: "Dashboard", Matches: 1},
		OutputSelections:      []authorresult.OutputSelection{{CandidateID: dashboard.Candidates[0].ID, Key: "dashboard_title", Type: "string", LocatorMode: "exact_name", Observation: 3, Context: "main", Role: "heading", Name: "Dashboard", Matches: 1, RoleMatches: 1}},
		AuthenticationProfile: authentication, HumanConfirmed: true, Diagnostics: []string{},
	}
}

func cloneEnvelope(t *testing.T, value *authorresult.Envelope) *authorresult.Envelope {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result authorresult.Envelope
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return &result
}
