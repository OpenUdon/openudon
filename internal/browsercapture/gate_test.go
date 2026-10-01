package browsercapture

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/openudon/internal/browserauthor"
)

func testGate(t *testing.T) (*Gate, Event, time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	g, err := NewGate(Authenticated, now.Add(time.Minute), func(view View, command Command) error {
		if command.Authentication == nil || command.Authentication.Kind != "click" || view.Authentication == nil || view.Authentication.Observation == nil {
			return errors.New("not offered")
		}
		for _, candidate := range view.Authentication.Observation.Candidates {
			if candidate.ID == command.Authentication.CandidateID {
				return nil
			}
		}
		return errors.New("candidate not issued")
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	e, err := g.Observe("observation", View{State: "observation", Authentication: &AuthenticatedView{Observation: &authorsession.Observation{Origin: "http://localhost", Path: "/login", Context: "main", Candidates: []authorsession.Candidate{{ID: "candidate-1111111111111111", Role: "button", Matches: 1}}}}}, now)
	if err != nil {
		t.Fatal(err)
	}
	return g, e, now
}

func proposalFor(event Event) Proposal {
	return Proposal{Binding: event.Binding, Type: "propose", Command: Command{Authentication: &browserauthor.Response{Kind: "click", CandidateID: "candidate-1111111111111111", POSTBudget: 1}}}
}

func decisionFor(event Event, approved bool) Decision {
	return Decision{Binding: event.Binding, Type: "decide", ActionID: event.Action.ID, CommandSHA256: event.Action.CommandSHA256, Approved: approved}
}

func TestIssuedActionIsExactAndSingleUse(t *testing.T) {
	g, event, now := testGate(t)
	proposal := proposalFor(event)
	card, err := g.Propose(proposal, now)
	if err != nil {
		t.Fatal(err)
	}
	proposal.Command.Authentication.CandidateID = "candidate-2222222222222222"
	card.Action.Command.Authentication.POSTBudget = 32
	got, err := g.Decide(decisionFor(card, true), now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Authentication.CandidateID != "candidate-1111111111111111" || got.Authentication.POSTBudget != 1 {
		t.Fatalf("issued payload changed: %+v", got)
	}
	if _, err = g.Decide(decisionFor(card, true), now); err == nil {
		t.Fatal("decision replay accepted")
	}
	if _, err = g.Propose(proposalFor(card), now); err == nil {
		t.Fatal("stale worker state reused after dispatch")
	}
}

func TestRefusalCannotDispatchOrReuse(t *testing.T) {
	g, event, now := testGate(t)
	card, err := g.Propose(proposalFor(event), now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := g.Decide(decisionFor(card, false), now)
	if err != nil || got != nil {
		t.Fatalf("refusal returned command: %v %v", got, err)
	}
	if _, err = g.Decide(decisionFor(card, true), now); err == nil {
		t.Fatal("refused command revived")
	}
}

func TestBindingAndDigestForgeriesFailWithoutConsuming(t *testing.T) {
	for _, field := range []string{"session", "mode", "revision", "event", "action", "digest"} {
		t.Run(field, func(t *testing.T) {
			g, event, now := testGate(t)
			card, err := g.Propose(proposalFor(event), now)
			if err != nil {
				t.Fatal(err)
			}
			d := decisionFor(card, true)
			switch field {
			case "session":
				d.SessionID = "capture-" + strings.Repeat("0", 32)
			case "mode":
				d.Mode = Registration
			case "revision":
				d.Revision++
			case "event":
				d.EventID = "ref-" + strings.Repeat("0", 32)
			case "action":
				d.ActionID = "ref-" + strings.Repeat("0", 32)
			case "digest":
				d.CommandSHA256 = strings.Repeat("0", 64)
			}
			if _, err = g.Decide(d, now); err == nil {
				t.Fatal("forged decision accepted")
			}
			if _, err = g.Decide(decisionFor(card, true), now); err != nil {
				t.Fatal("forgery consumed legitimate action", err)
			}
		})
	}
}

func TestChangedWorkerStateInvalidatesPendingDecision(t *testing.T) {
	g, event, now := testGate(t)
	card, err := g.Propose(proposalFor(event), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.Observe("state", View{State: "observing"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err = g.Decide(decisionFor(card, true), now); err == nil {
		t.Fatal("old action survived worker state change")
	}
}

func TestExpiredAndCanceledAuthorityDoesNotResume(t *testing.T) {
	g, event, now := testGate(t)
	card, err := g.Propose(proposalFor(event), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.Decide(decisionFor(card, true), now.Add(time.Minute)); err == nil {
		t.Fatal("expired decision accepted")
	}
	if err = g.Cancel(Cancellation{Binding: card.Binding, Type: "cancel"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err = g.Observe("state", View{State: "observing"}, now); err == nil {
		t.Fatal("canceled session resumed")
	}
	if _, err = NewGate(Authenticated, now.Add(3*time.Hour), func(View, Command) error { return nil }, now); err == nil {
		t.Fatal("unbounded deadline accepted")
	}
}

func TestUnknownCandidateAndWrongModeCannotBeProposed(t *testing.T) {
	g, event, now := testGate(t)
	p := proposalFor(event)
	p.Command.Authentication.CandidateID = "candidate-2222222222222222"
	if _, err := g.Propose(p, now); err == nil {
		t.Fatal("unissued candidate accepted")
	}
	p = proposalFor(event)
	p.Command.Registration = &RegistrationCommand{Type: "observe"}
	if _, err := g.Propose(p, now); err == nil {
		t.Fatal("mixed mode union accepted")
	}
}

func TestStrictBoundedWireRejectsCredentialFields(t *testing.T) {
	_, event, _ := testGate(t)
	data, err := json.Marshal(proposalFor(event))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = DecodeProposal(data); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{
		strings.Replace(string(data), `"kind":"click"`, `"kind":"click","password":"canary"`, 1),
		strings.Replace(string(data), `"type":"propose"`, `"type":"propose","type":"propose"`, 1),
		string(data) + string(data), strings.Repeat(" ", MaxMessageBytes+1), "null",
	} {
		if _, err = DecodeProposal([]byte(bad)); err == nil {
			t.Fatal("unsafe wire accepted")
		}
	}
}

func TestTerminalAndOversizedViewsFailClosed(t *testing.T) {
	g, event, now := testGate(t)
	card, err := g.Propose(proposalFor(event), now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.Observe("observation", View{State: "observation", Authentication: &AuthenticatedView{Observation: &authorsession.Observation{Path: strings.Repeat("x", MaxMessageBytes+1)}}}, now); err == nil {
		t.Fatal("unbounded view accepted")
	}
	if _, err := g.Decide(decisionFor(card, true), now); err == nil {
		t.Fatal("failed worker-state publication retained action authority")
	}
	g, _, now = testGate(t)
	if _, err := g.Observe("result", View{State: "completed"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Observe("state", View{State: "observing"}, now); err == nil {
		t.Fatal("terminal session resumed")
	}
}
