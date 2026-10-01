package browsercapture

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browserauthor"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/uws/browserregistration"
)

type registrationInstruction struct {
	event    *browserauthor.RegistrationEvent
	retained *browserauthor.RegistrationEvent
	finish   bool
}
type fakeRegistration struct {
	updates      chan browserauthor.RegistrationEvent
	commands     chan registrationInstruction
	responses    chan browserauthor.RegistrationCommand
	stop, joined chan struct{}
	once         sync.Once
	mu           sync.Mutex
	state        string
	generation   int
	checks       int
	terminal     browserauthor.RegistrationEvent
	terminalSet  bool
	lateTeardown bool
}

func (s *fakeRegistration) Events() <-chan browserauthor.RegistrationEvent { return s.updates }
func (s *fakeRegistration) Cancel()                                        { s.once.Do(func() { close(s.stop) }) }
func (s *fakeRegistration) Send(ctx context.Context, command browserauthor.RegistrationCommand) error {
	select {
	case s.responses <- command:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *fakeRegistration) ValidateDecision(state string, generation int, command browserauthor.RegistrationCommand) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.checks++
	if state != s.state || generation != s.generation {
		return errors.New("mock current state mismatch")
	}
	return nil
}
func (s *fakeRegistration) TerminalEvent() (browserauthor.RegistrationEvent, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.terminal, s.terminalSet
}
func (s *fakeRegistration) run() {
	defer close(s.updates)
	defer close(s.joined)
	for {
		select {
		case <-s.stop:
			if s.lateTeardown {
				s.mu.Lock()
				s.terminal = browserauthor.RegistrationEvent{State: "failed", ErrorCode: "worker_teardown", Diagnostic: "network_policy"}
				s.terminalSet = true
				s.mu.Unlock()
			}
			return
		case command := <-s.commands:
			if command.retained != nil {
				s.mu.Lock()
				s.terminal, s.terminalSet = *command.retained, true
				s.mu.Unlock()
			}
			if command.finish {
				return
			}
			s.mu.Lock()
			s.state = command.event.State
			s.generation = 0
			if command.event.Observation != nil {
				s.generation = command.event.Observation.Generation
			}
			s.mu.Unlock()
			select {
			case s.updates <- *command.event:
			case <-s.stop:
				return
			}
		}
	}
}

type registrationRunResult struct {
	completion RegistrationCompletion
	err        error
}
type registrationHarness struct {
	*authHarness       // Only its pipe/read/review helpers; native sessions stay distinct.
	registration       *fakeRegistration
	registrationResult chan registrationRunResult
}

func newRegistrationHarness(t *testing.T, ttl time.Duration, late bool) *registrationHarness {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	cfg := browserauthor.RegistrationConfig{PrivateRoot: root, Protocol: registrationauthorsession.ProtocolV4, TransactionID: "test-registration", Absolute: ttl}
	start := browserauthor.RegistrationCommand{Type: "start", ProfileID: "register", URL: "https://app.example.test/register", Origins: []string{"https://app.example.test"}}
	ctx, cancel := context.WithCancel(context.Background())
	in, client := io.Pipe()
	output, out := io.Pipe()
	worker := &fakeRegistration{updates: make(chan browserauthor.RegistrationEvent), commands: make(chan registrationInstruction, 1), responses: make(chan browserauthor.RegistrationCommand, 4), stop: make(chan struct{}), joined: make(chan struct{}), lateTeardown: late}
	h := &registrationHarness{authHarness: &authHarness{in: client, out: bufio.NewReader(output), cancel: cancel}, registration: worker, registrationResult: make(chan registrationRunResult, 1)}
	go func() {
		completion, err := runRegistration(ctx, cfg, start, in, out, func(_ context.Context, normalized browserauthor.RegistrationConfig) (registrationSession, error) {
			if normalized.Protocol != registrationauthorsession.ProtocolV4 {
				return nil, errors.New("mock protocol changed")
			}
			go worker.run()
			return worker, nil
		})
		h.registrationResult <- registrationRunResult{completion, err}
	}()
	t.Cleanup(func() { cancel(); _ = client.Close(); _ = output.Close() })
	return h
}
func (h *registrationHarness) update(event browserauthor.RegistrationEvent) {
	h.registration.commands <- registrationInstruction{event: &event}
}
func (h *registrationHarness) done(t *testing.T) registrationRunResult {
	t.Helper()
	select {
	case result := <-h.registrationResult:
		select {
		case <-h.registration.joined:
		default:
			t.Fatal("registration worker not joined")
		}
		return result
	case <-time.After(4 * time.Second):
		t.Fatal("registration teardown did not complete")
		return registrationRunResult{}
	}
}
func (h *registrationHarness) noDispatch(t *testing.T) {
	t.Helper()
	select {
	case command := <-h.registration.responses:
		t.Fatalf("unapproved registration dispatch: %#v", command)
	default:
	}
}
func (h *registrationHarness) ready(t *testing.T) {
	t.Helper()
	h.update(browserauthor.RegistrationEvent{State: "ready"})
	_ = h.read(t)
	select {
	case command := <-h.registration.responses:
		if command.Type != "start" || command.URL != "https://app.example.test/register" || command.ProfileID != "register" {
			t.Fatal("fixed approved start changed")
		}
	case <-time.After(time.Second):
		t.Fatal("approved start not dispatched")
	}
}
func registrationObservation() browserauthor.RegistrationEvent {
	return browserauthor.RegistrationEvent{State: "observation", Phase: "observing", Observation: &registrationauthorsession.Observation{Generation: 1, Origin: "https://app.example.test", Path: "/register", Candidates: []registrationauthorsession.Candidate{{ID: "candidate-0123456789abcdef", Role: "button", Label: "Register", Matches: 1}}, Diagnostics: []string{}}}
}

func TestRegistrationAdapterKeepsExactVerificationRefusalWithoutTraffic(t *testing.T) {
	h := newRegistrationHarness(t, 3*time.Second, false)
	h.ready(t)
	h.update(registrationObservation())
	current := h.read(t)
	verification := &browserregistration.HumanVerification{Provider: "hcaptcha", Activation: "approved_submit", WidgetBinding: "single_in_submit_form", SubmissionURL: "https://app.example.test/register", Dependencies: browserregistration.VerificationDependencies{Policy: "hcaptcha.v1", MaxRequests: 256, MaxResponseBytes: 32 << 20, TimeoutMS: 120000}}
	command := RegistrationCommand{Type: "approve_verification", Confirmed: true, Verification: verification, CandidateID: current.View.Registration.Observation.Candidates[0].ID}
	card := h.review(t, current, Command{Registration: &command})
	h.noDispatch(t)
	h.decide(t, card, false)
	fresh := h.read(t)
	h.noDispatch(t)
	if fresh.Type == "disclosure" || fresh.Revision <= card.Revision {
		t.Fatal("verification refusal gained authority")
	}
	card = h.review(t, fresh, Command{Registration: &command})
	h.decide(t, card, true)
	select {
	case got := <-h.registration.responses:
		if !reflect.DeepEqual(got, registrationCommand(command)) {
			t.Fatal("exact reviewed verification changed")
		}
	case <-time.After(time.Second):
		t.Fatal("reviewed authority not dispatched")
	}
	h.cancel()
	if result := h.done(t); result.err == nil {
		t.Fatal("canceled authoring adopted profile")
	}
}

func TestRegistrationAdapterSendsOnlyCurrentObservationAndLatestPreview(t *testing.T) {
	h := newRegistrationHarness(t, 3*time.Second, false)
	h.ready(t)
	observation := registrationObservation()
	observation.History = make([]registrationauthorsession.Observation, 3000)
	checked := true
	observation.Previews = []registrationauthorsession.PreviewRecord{
		{Request: registrationauthorsession.PreviewRequest{CandidateID: "candidate-1123456789abcdef", Generation: 1, Action: "check", Purpose: "public_form_preview", Checked: &checked}, NextGeneration: 2},
		{Request: registrationauthorsession.PreviewRequest{CandidateID: "candidate-2123456789abcdef", Generation: 2, Action: "check", Purpose: "public_form_preview", Checked: &checked}, NextGeneration: 3},
	}
	h.update(observation)
	current := h.read(t)
	data, _ := json.Marshal(current)
	if current.Type != "preview" || current.View.Registration.Preview.NextGeneration != 3 || len(data) > 3000 || strings.Contains(string(data), "history") || strings.Contains(string(data), "candidate-1123456789abcdef") {
		t.Fatal("full controller history or previous preview was retransmitted")
	}
	consent := true
	card := h.review(t, current, Command{DiscloseObservation: &consent})
	h.decide(t, card, true)
	disclosure := h.read(t)
	if disclosure.Type != "disclosure" {
		t.Fatal("explicit current-observation consent missing")
	}
	h.noDispatch(t)
	h.send(t, Cancellation{Binding: disclosure.Binding, Type: "cancel"})
	if result := h.done(t); !errors.Is(result.err, ErrCanceled) {
		t.Fatal(result.err)
	}
}

func TestRegistrationAdapterConsultsRetainedOutcomeAfterJoinedStream(t *testing.T) {
	for _, outcome := range []string{"candidate", "late failure"} {
		t.Run(outcome, func(t *testing.T) {
			h := newRegistrationHarness(t, 3*time.Second, false)
			h.ready(t)
			h.update(browserauthor.RegistrationEvent{State: "closed", Phase: "closed"})
			terminal := browserauthor.RegistrationEvent{State: "candidate", Candidate: &browsercandidate.Registration{}}
			if outcome == "late failure" {
				terminal = browserauthor.RegistrationEvent{State: "failed", ErrorCode: "worker_teardown", Diagnostic: "network_policy"}
			}
			// Simulate terminal publication dropped by the bounded native stream.
			h.registration.commands <- registrationInstruction{retained: &terminal, finish: true}
			final := h.read(t)
			result := h.done(t)
			if outcome == "candidate" {
				if final.Type != "result" || final.View.State != "captured" || final.View.Result != nil || result.err != nil || result.completion.Candidate == nil {
					t.Fatal("joined private candidate missing", result.err)
				}
				data, _ := json.Marshal(result.completion)
				if string(data) != "{}" {
					t.Fatal("private candidate serialized")
				}
			} else if result.err == nil || final.View.Diagnostic != "worker_teardown" {
				t.Fatal("late failure hidden by earlier close", result.err)
			}
		})
	}
}

func TestRegistrationAdapterLateContainmentFailureOverridesCancel(t *testing.T) {
	h := newRegistrationHarness(t, 3*time.Second, true)
	h.ready(t)
	h.update(registrationObservation())
	current := h.read(t)
	h.send(t, Cancellation{Binding: current.Binding, Type: "cancel"})
	if result := h.done(t); !errors.Is(result.err, errWorker) {
		t.Fatal("dropped late containment failure hidden", result.err)
	}
	h.noDispatch(t)
}

func TestRegistrationAdapterRejectsUnknownValuesAndReplayedCommands(t *testing.T) {
	for _, operation := range []string{"private value", "replay"} {
		t.Run(operation, func(t *testing.T) {
			h := newRegistrationHarness(t, 3*time.Second, false)
			h.ready(t)
			h.update(registrationObservation())
			current := h.read(t)
			proposal := Proposal{Binding: current.Binding, Type: "propose", Command: Command{Registration: &RegistrationCommand{Type: "observe"}}}
			if operation == "private value" {
				data, _ := json.Marshal(proposal)
				data = []byte(strings.Replace(string(data), `"type":"observe"`, `"type":"observe","verification_code":"PRIVATE_CODE_CANARY"`, 1))
				_, _ = h.in.Write(append(data, '\n'))
			} else {
				card := h.review(t, current, proposal.Command)
				h.decide(t, card, true)
				<-h.registration.responses
				h.decide(t, card, true)
			}
			result := h.done(t)
			if result.err == nil || strings.Contains(result.err.Error(), "PRIVATE_CODE_CANARY") {
				t.Fatal("invalid decision accepted or private input exposed")
			}
			h.noDispatch(t)
		})
	}
}

func TestRegistrationAdapterIntentionalCloseIsCancellation(t *testing.T) {
	h := newRegistrationHarness(t, 3*time.Second, false)
	h.ready(t)
	h.update(browserauthor.RegistrationEvent{State: "closed", Phase: "closed"})
	h.registration.commands <- registrationInstruction{finish: true}
	final := h.read(t)
	if final.View.State != "closed" {
		t.Fatal("intentional close was relabeled")
	}
	if result := h.done(t); !errors.Is(result.err, ErrCanceled) || result.completion.Candidate != nil {
		t.Fatal("close granted completion or reported worker failure", result.err)
	}
}
