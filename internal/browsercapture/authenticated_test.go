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

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/openudon/internal/browserauthor"
)

type authInstruction struct {
	event  *browserauthor.Event
	finish bool
}
type fakeAuthentication struct {
	updates      chan browserauthor.Event
	commands     chan authInstruction
	responses    chan browserauthor.Response
	stop         chan struct{}
	joined       chan struct{}
	once         sync.Once
	lateTeardown bool
}

func (s *fakeAuthentication) Events() <-chan browserauthor.Event { return s.updates }
func (s *fakeAuthentication) Respond(ctx context.Context, response browserauthor.Response) error {
	select {
	case s.responses <- response:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func (s *fakeAuthentication) Cancel() { s.once.Do(func() { close(s.stop) }) }
func (s *fakeAuthentication) run() {
	defer close(s.updates)
	defer close(s.joined)
	for {
		select {
		case <-s.stop:
			if s.lateTeardown {
				s.updates <- browserauthor.Event{State: "failed", ErrorCode: "worker_teardown"}
			}
			return
		case command := <-s.commands:
			if command.finish {
				return
			}
			select {
			case s.updates <- *command.event:
			case <-s.stop:
				return
			}
		}
	}
}

type authRunResult struct {
	completion AuthenticatedCompletion
	err        error
}
type authHarness struct {
	in     *io.PipeWriter
	out    *bufio.Reader
	worker *fakeAuthentication
	result chan authRunResult
	cancel context.CancelFunc
}

func newAuthHarness(t *testing.T, absolute time.Duration, lateTeardown bool, options ...authenticationOptions) *authHarness {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	config := browserauthor.Config{PrivateRoot: root, InitialURL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard", Goal: "Review account status", Origins: []string{"https://members.example.test"}, ProfileID: "member", Absolute: absolute}
	ctx, cancel := context.WithCancel(context.Background())
	in, client := io.Pipe()
	output, out := io.Pipe()
	worker := &fakeAuthentication{updates: make(chan browserauthor.Event), commands: make(chan authInstruction, 1), responses: make(chan browserauthor.Response, 4), stop: make(chan struct{}), joined: make(chan struct{}), lateTeardown: lateTeardown}
	h := &authHarness{in: client, out: bufio.NewReader(output), worker: worker, result: make(chan authRunResult, 1), cancel: cancel}
	go func() {
		completion, err := runAuthenticated(ctx, config, in, out, func(_ context.Context, normalized browserauthor.Config) (authenticationSession, error) {
			if normalized.DashboardURL != config.DashboardURL || normalized.ProfileID != "member" {
				return nil, errors.New("normalized authority changed")
			}
			go worker.run()
			return worker, nil
		}, options...)
		h.result <- authRunResult{completion, err}
	}()
	t.Cleanup(func() { cancel(); _ = client.Close(); _ = output.Close() })
	return h
}

func TestContinueCurrentPageRejectsDashboardShortcut(t *testing.T) {
	h := newAuthHarness(t, 3*time.Second, false, authenticationOptions{continuation: "continue_current_page"})
	h.update(captureObservation())
	current := h.read(t)
	h.send(t, Proposal{Binding: current.Binding, Type: "propose", Command: Command{Authentication: &browserauthor.Response{Kind: "authenticated"}}})
	if result := h.done(t); result.err == nil {
		t.Fatal("dashboard shortcut ignored reviewed continuation")
	}
	h.noDispatch(t)
}
func (h *authHarness) update(event browserauthor.Event) {
	h.worker.commands <- authInstruction{event: &event}
}
func (h *authHarness) read(t *testing.T) Event {
	t.Helper()
	data, err := h.out.ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	event, err := DecodeEvent(data)
	if err != nil {
		t.Fatal(err)
	}
	return event
}
func (h *authHarness) send(t *testing.T, message any) {
	t.Helper()
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.in.Write(append(data, '\n')); err != nil {
		t.Fatal(err)
	}
}
func (h *authHarness) review(t *testing.T, current Event, command Command) Event {
	t.Helper()
	h.send(t, Proposal{Binding: current.Binding, Type: "propose", Command: command})
	return h.read(t)
}
func (h *authHarness) decide(t *testing.T, card Event, approved bool) {
	t.Helper()
	h.send(t, Decision{Binding: card.Binding, Type: "decide", ActionID: card.Action.ID, CommandSHA256: card.Action.CommandSHA256, Approved: approved})
}
func (h *authHarness) done(t *testing.T) authRunResult {
	t.Helper()
	select {
	case result := <-h.result:
		select {
		case <-h.worker.joined:
		default:
			t.Fatal("adapter returned before worker joined")
		}
		return result
	case <-time.After(4 * time.Second):
		t.Fatal("capture teardown did not complete")
		return authRunResult{}
	}
}
func (h *authHarness) noDispatch(t *testing.T) {
	t.Helper()
	select {
	case response := <-h.worker.responses:
		t.Fatalf("unapproved dispatch: %#v", response)
	default:
	}
}
func captureObservation() browserauthor.Event {
	return browserauthor.Event{State: "exploration", Phase: "authentication", Observation: &authorsession.Observation{Origin: "https://members.example.test", Path: "/login", Context: "main", Contexts: map[string]authorresult.Context{}, Candidates: []authorsession.Candidate{{ID: "candidate-0123456789abcdef", Role: "button", Label: "Sign in", Matches: 1}}, Diagnostics: []string{}}}
}

func TestAuthenticatedAdapterRequiresExactReviewBeforeDispatch(t *testing.T) {
	h := newAuthHarness(t, 3*time.Second, false)
	h.update(captureObservation())
	current := h.read(t)
	command := Command{Authentication: &browserauthor.Response{Kind: "click", CandidateID: current.View.Authentication.Observation.Candidates[0].ID, POSTBudget: 1}}
	card := h.review(t, current, command)
	h.noDispatch(t)
	if card.Type != "proposal" || card.Action.Command.Authentication.POSTBudget != 1 {
		t.Fatal("exact command not shown")
	}
	h.decide(t, card, true)
	select {
	case response := <-h.worker.responses:
		if !reflect.DeepEqual(response, *command.Authentication) {
			t.Fatal("approved command changed")
		}
	case <-time.After(time.Second):
		t.Fatal("approved command not dispatched")
	}
	// Replaying the consumed card fails closed and never dispatches twice.
	h.decide(t, card, true)
	if result := h.done(t); result.err == nil {
		t.Fatal("replayed decision accepted")
	}
	h.noDispatch(t)
}

func TestAuthenticatedAdapterRefusalAndDisclosureAreObservationBound(t *testing.T) {
	h := newAuthHarness(t, 3*time.Second, false)
	h.update(captureObservation())
	current := h.read(t)
	consent := true
	card := h.review(t, current, Command{DiscloseObservation: &consent})
	h.decide(t, card, false)
	fresh := h.read(t)
	if fresh.Type == "disclosure" || fresh.Revision <= card.Revision {
		t.Fatal("refusal disclosed observation or failed to renew state")
	}
	h.noDispatch(t)
	card = h.review(t, fresh, Command{DiscloseObservation: &consent})
	h.decide(t, card, true)
	disclosed := h.read(t)
	if disclosed.Type != "disclosure" || disclosed.View.Authentication.Observation.Path != "/login" {
		t.Fatal("exact consented observation unavailable")
	}
	h.noDispatch(t)
	changed := captureObservation()
	changed.Observation.Path = "/dashboard"
	h.update(changed)
	next := h.read(t)
	if next.Type == "disclosure" {
		t.Fatal("model consent applied to later observation")
	}
	h.send(t, Cancellation{Binding: next.Binding, Type: "cancel"})
	if result := h.done(t); !errors.Is(result.err, ErrCanceled) {
		t.Fatal(result.err)
	}
}

func TestAuthenticatedAdapterTOTPAndCredentialCheckpointAreValueFree(t *testing.T) {
	for _, kind := range []string{"credential", "mfa"} {
		t.Run(kind, func(t *testing.T) {
			h := newAuthHarness(t, 3*time.Second, false)
			checkpoint := &authorsession.Checkpoint{Kind: kind, CandidateID: "candidate-0123456789abcdef", InputKind: "password"}
			response := browserauthor.Response{Kind: "continue", CandidateID: checkpoint.CandidateID}
			if kind == "mfa" {
				checkpoint.InputKind = "otp"
				checkpoint.ChallengeKinds = []string{"totp", "sms_otp"}
				response.ChallengeKind = "totp"
			}
			h.update(browserauthor.Event{State: "human_input", Phase: "authentication", Checkpoint: checkpoint})
			current := h.read(t)
			card := h.review(t, current, Command{Authentication: &response})
			h.noDispatch(t)
			h.decide(t, card, true)
			select {
			case got := <-h.worker.responses:
				if !reflect.DeepEqual(got, response) {
					t.Fatal("checkpoint choice changed")
				}
			case <-time.After(time.Second):
				t.Fatal("checkpoint acknowledgement missing")
			}
			h.cancel()
			if result := h.done(t); !errors.Is(result.err, ErrCanceled) {
				t.Fatal(result.err)
			}
		})
	}
}

func TestAuthenticatedAdapterRejectsUnofferedMFAAndPrivateFields(t *testing.T) {
	for _, payload := range []string{"unoffered", "credential", "duplicates", "oversize"} {
		t.Run(payload, func(t *testing.T) {
			h := newAuthHarness(t, 3*time.Second, false)
			h.update(browserauthor.Event{State: "human_input", Checkpoint: &authorsession.Checkpoint{Kind: "mfa", CandidateID: "candidate-0123456789abcdef", InputKind: "otp", ChallengeKinds: []string{"totp"}}})
			current := h.read(t)
			proposal := Proposal{Binding: current.Binding, Type: "propose", Command: Command{Authentication: &browserauthor.Response{Kind: "continue", CandidateID: "candidate-0123456789abcdef", ChallengeKind: "sms_otp"}}}
			data, _ := json.Marshal(proposal)
			switch payload {
			case "credential":
				data = []byte(strings.Replace(string(data), `"kind":"continue"`, `"kind":"continue","password":"PRIVATE_VALUE_CANARY"`, 1))
			case "duplicates":
				data = []byte(strings.Replace(string(data), `"type":"propose"`, `"type":"propose","type":"cancel"`, 1))
			case "oversize":
				data = []byte(strings.Repeat("x", MaxMessageBytes+1))
			}
			_, _ = h.in.Write(append(data, '\n'))
			result := h.done(t)
			if result.err == nil || strings.Contains(result.err.Error(), "PRIVATE_VALUE_CANARY") {
				t.Fatal("invalid capture input accepted or exposed")
			}
			h.noDispatch(t)
		})
	}
}

func TestAuthenticatedAdapterKeepsWorkerResultPrivateAndWaitsForJoin(t *testing.T) {
	h := newAuthHarness(t, 3*time.Second, false)
	h.update(browserauthor.Event{State: "launching"})
	_ = h.read(t)
	private := browserauthor.Event{State: "completion_review", Phase: "completed", Result: &authorsession.Result{ArtifactPath: "/private/WORKER_PATH_CANARY", Digest: strings.Repeat("a", 64)}, Attestation: &browserauthor.Attestation{}}
	h.update(private)
	select {
	case <-h.result:
		t.Fatal("completed before worker closure")
	default:
	}
	h.worker.commands <- authInstruction{finish: true}
	final := h.read(t)
	data, _ := json.Marshal(final)
	if final.Type != "result" || final.View.State != "captured" || strings.Contains(string(data), "WORKER_PATH_CANARY") || strings.Contains(string(data), strings.Repeat("a", 64)) {
		t.Fatal("private worker result exposed")
	}
	result := h.done(t)
	if result.err != nil || result.completion.Event.Result.ArtifactPath != private.Result.ArtifactPath {
		t.Fatal("private completion missing", result.err)
	}
	encoded, _ := json.Marshal(result.completion)
	if string(encoded) != "{}" {
		t.Fatal("completion has a public JSON representation")
	}
}

func TestAuthenticatedAdapterDisconnectExpiryAndLateTeardown(t *testing.T) {
	for _, operation := range []string{"EOF", "cancel", "expiry", "late teardown"} {
		t.Run(operation, func(t *testing.T) {
			ttl := 3 * time.Second
			if operation == "expiry" {
				ttl = 30 * time.Millisecond
			}
			h := newAuthHarness(t, ttl, operation == "late teardown")
			h.update(browserauthor.Event{State: "launching"})
			current := h.read(t)
			switch operation {
			case "EOF":
				_ = h.in.Close()
			case "cancel", "late teardown":
				h.send(t, Cancellation{Binding: current.Binding, Type: "cancel"})
			}
			result := h.done(t)
			if result.err == nil {
				t.Fatal("failed transport granted completion")
			}
			if operation == "late teardown" && !errors.Is(result.err, errWorker) {
				t.Fatal("teardown gap hidden", result.err)
			}
			h.noDispatch(t)
		})
	}
}

func TestAuthenticatedAdapterClosesBlockedOutputOnCancellation(t *testing.T) {
	h := newAuthHarness(t, 3*time.Second, false)
	h.update(browserauthor.Event{State: "launching"}) // Nobody reads the output pipe.
	h.cancel()
	if result := h.done(t); result.err == nil {
		t.Fatal("blocked output survived cancellation")
	}
}

func TestAuthenticatedAdapterPreservesSeparateWorkerOriginApproval(t *testing.T) {
	h := newAuthHarness(t, 3*time.Second, false)
	h.update(captureObservation())
	current := h.read(t)
	card := h.review(t, current, Command{Authentication: &browserauthor.Response{Kind: "navigate_get", URL: "https://other.example.test/"}})
	h.decide(t, card, true)
	<-h.worker.responses
	approval := &authorsession.Approval{ID: "approval-0001", Kind: "origin", Origin: "https://other.example.test", Action: "navigate_get"}
	h.update(browserauthor.Event{State: "action_approval", Phase: "authentication", Approval: approval})
	current = h.read(t)
	if current.Type != "approval_required" || current.View.Authentication.Approval.Origin != approval.Origin {
		t.Fatal("worker's separate exact origin approval was skipped")
	}
	card = h.review(t, current, Command{Authentication: &browserauthor.Response{Kind: "deny", ApprovalID: approval.ID}})
	h.noDispatch(t)
	h.decide(t, card, true) // Approve dispatch of the worker denial, not navigation.
	select {
	case response := <-h.worker.responses:
		if response.Kind != "deny" || response.ApprovalID != approval.ID {
			t.Fatal("worker denial changed")
		}
	case <-time.After(time.Second):
		t.Fatal("worker denial not dispatched")
	}
	h.cancel()
	if result := h.done(t); result.err == nil {
		t.Fatal("canceled capture produced profiles")
	}
}

func TestAuthenticatedAdapterChangedWorkerStateInvalidatesReviewCard(t *testing.T) {
	h := newAuthHarness(t, 3*time.Second, false)
	h.update(captureObservation())
	current := h.read(t)
	consent := true
	card := h.review(t, current, Command{DiscloseObservation: &consent})
	changed := captureObservation()
	changed.Observation.Path = "/other"
	h.update(changed)
	_ = h.read(t)
	h.decide(t, card, true)
	if result := h.done(t); result.err == nil {
		t.Fatal("stale observation consent accepted")
	}
	h.noDispatch(t)
}
