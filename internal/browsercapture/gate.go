package browsercapture

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"time"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

var referencePattern = regexp.MustCompile(`^ref-[0-9a-f]{32}$`)

// Validator checks a proposal against the current reduced controller event.
// It must not execute actions. Controllers retain their own semantic checks
// after an approved command is dispatched.
type Validator func(View, Command) error

// Gate is process-local, single-owner authority. Restart never restores it.
// Callers must serialize access; the supervising CLI has one event loop.
type Gate struct {
	sessionID     string
	mode          string
	deadline      time.Time
	validator     Validator
	current       Event
	pending       []byte
	pendingID     string
	pendingDigest string
	closed        bool
}

func NewGate(mode string, deadline time.Time, validator Validator, now time.Time) (*Gate, error) {
	if (mode != Authenticated && mode != Registration) || validator == nil || !validDeadline(deadline, now) || deadline.Sub(now) > 2*time.Hour {
		return nil, errors.New("invalid capture gate configuration")
	}
	id, err := newReference()
	if err != nil {
		return nil, err
	}
	return &Gate{sessionID: "capture-" + id[4:], mode: mode, deadline: deadline, validator: validator}, nil
}

func newReference() (string, error) {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		return "", errors.New("capture reference unavailable")
	}
	return "ref-" + hex.EncodeToString(data[:]), nil
}

// Observe invalidates all decisions issued for earlier worker state, including
// a proposal that had not yet been approved.
func (g *Gate) Observe(kind string, view View, now time.Time) (Event, error) {
	if g == nil || g.closed || !validDeadline(g.deadline, now) {
		return Event{}, errors.New("capture session unavailable")
	}
	switch kind {
	case "state", "observation", "approval_required", "human_checkpoint", "preview", "diagnostic", "result", "disclosure":
	default:
		g.closed = true
		g.pending = nil
		return Event{}, errors.New("invalid capture event kind")
	}
	if view.State == "" || len(view.State) > 64 || len(view.Phase) > 64 || len(view.Diagnostic) > 64 ||
		g.mode == Authenticated && view.Registration != nil || g.mode == Registration && view.Authentication != nil {
		g.closed = true
		g.pending = nil
		return Event{}, errors.New("invalid capture view")
	}
	event, err := g.issue(kind, view, nil)
	if err != nil || kind == "result" {
		g.closed = true
		g.pending = nil
	}
	return event, err
}

func (g *Gate) issue(kind string, view View, action *Action) (Event, error) {
	if g.current.Revision >= MaxEvents {
		return Event{}, errors.New("capture event limit exceeded")
	}
	id, err := newReference()
	if err != nil {
		return Event{}, err
	}
	event := Event{Binding: Binding{Version: Version, SessionID: g.sessionID, Mode: g.mode, Revision: g.current.Revision + 1, EventID: id}, Type: kind, Deadline: g.deadline.UTC().Format(time.RFC3339Nano), View: view, Action: action}
	data, err := boundedJSON(event)
	if err != nil {
		return Event{}, err
	}
	var immutable Event
	if err := decode(data, &immutable); err != nil {
		return Event{}, err
	}
	g.current = immutable
	g.pending = nil
	g.pendingID = ""
	g.pendingDigest = ""
	return event, nil
}

func (g *Gate) matches(binding Binding, now time.Time) bool {
	return g != nil && !g.closed && validDeadline(g.deadline, now) && validBinding(binding) && binding == g.current.Binding
}

// Propose creates an exact review card and never dispatches the proposed
// action. Subsequent approval must name this new event and its action digest.
func (g *Gate) Propose(proposal Proposal, now time.Time) (Event, error) {
	if proposal.Type != "propose" || !g.matches(proposal.Binding, now) || g.current.Type == "proposal" || len(g.pending) != 0 {
		return Event{}, errors.New("capture proposal reference is stale or unavailable")
	}
	data, err := boundedJSON(proposal.Command)
	if err != nil {
		return Event{}, err
	}
	var command Command
	if err = decode(data, &command); err != nil {
		return Event{}, err
	}
	count := 0
	if command.Authentication != nil {
		count++
	}
	if command.Registration != nil {
		count++
	}
	if command.DiscloseObservation != nil {
		count++
	}
	if count != 1 || g.mode == Authenticated && command.Registration != nil || g.mode == Registration && command.Authentication != nil {
		return Event{}, errors.New("capture command union is invalid")
	}
	// A caller cannot mutate the gate's current view through the validator.
	viewBytes, err := boundedJSON(g.current.View)
	if err != nil {
		return Event{}, err
	}
	var view View
	if err = decode(viewBytes, &view); err != nil {
		return Event{}, err
	}
	if err = g.validator(view, command); err != nil {
		return Event{}, errors.New("capture command is not offered by current state")
	}
	// Validation cannot silently change the command shown on the review card.
	if err = decode(data, &command); err != nil {
		return Event{}, err
	}
	id, err := newReference()
	if err != nil {
		return Event{}, err
	}
	digest := evidencefile.SHA256(data)
	event, err := g.issue("proposal", g.current.View, &Action{ID: id, CommandSHA256: digest, Command: command})
	if err != nil {
		return Event{}, err
	}
	g.pending = append([]byte(nil), data...)
	g.pendingID = id
	g.pendingDigest = digest
	return event, nil
}

// Decide consumes the exact offered action once, for both approval and refusal.
// Refusal returns no command. The caller must dispatch only the returned copy.
func (g *Gate) Decide(decision Decision, now time.Time) (*Command, error) {
	if decision.Type != "decide" || !g.matches(decision.Binding, now) || len(g.pending) == 0 ||
		decision.ActionID != g.pendingID || decision.CommandSHA256 != g.pendingDigest {
		return nil, errors.New("capture decision reference is stale or unavailable")
	}
	data := append([]byte(nil), g.pending...)
	g.pending = nil
	g.pendingID = ""
	g.pendingDigest = ""
	if !decision.Approved {
		return nil, nil
	}
	var command Command
	if err := json.Unmarshal(data, &command); err != nil {
		return nil, errors.New("capture issued command unavailable")
	}
	return &command, nil
}

// Cancel validates the latest event reference and destroys authority. Actual
// worker cancellation and supervised teardown belong to the owning adapter.
func (g *Gate) Cancel(message Cancellation, now time.Time) error {
	if message.Type != "cancel" || !g.matches(message.Binding, now) {
		return errors.New("capture cancellation reference is stale or unavailable")
	}
	g.closed = true
	g.pending = nil
	g.pendingID = ""
	g.pendingDigest = ""
	return nil
}
