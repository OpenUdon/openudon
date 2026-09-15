package icot

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

type heldScenarioSession struct {
	events                chan browserauthor.Event
	joining, canceled     chan struct{}
	reads                 int
	cancelOnce, closeOnce sync.Once
}

func (s *heldScenarioSession) Events() <-chan browserauthor.Event {
	s.reads++
	if s.reads == 2 {
		close(s.joining)
	}
	return s.events
}

func (s *heldScenarioSession) Respond(context.Context, browserauthor.Response) error {
	return errors.New("unexpected test response")
}
func (s *heldScenarioSession) Cancel() { s.cancelOnce.Do(func() { close(s.canceled) }) }
func (s *heldScenarioSession) close()  { s.closeOnce.Do(func() { close(s.events) }) }

func TestScenarioControllerJoinsWorkerBeforeReturning(t *testing.T) {
	for _, kind := range []string{"failure", "expected_rejection", "success", "late_failure", "rejection_late_failure"} {
		t.Run(kind, func(t *testing.T) {
			session := &heldScenarioSession{events: make(chan browserauthor.Event, 1), joining: make(chan struct{}), canceled: make(chan struct{})}
			t.Cleanup(session.close)
			request := BrowserScenarioAuthorRequest{}
			switch kind {
			case "failure":
				session.events <- browserauthor.Event{State: "failed", ErrorCode: "worker_protocol"}
			case "expected_rejection", "rejection_late_failure":
				request.Fault = "outputs_17"
				request.Outputs = make([]BrowserScenarioOutput, 17)
				session.events <- browserauthor.Event{State: "completion_review", Checkpoint: &authorsession.Checkpoint{Kind: "completion"}, Observation: &authorsession.Observation{}}
			default:
				session.events <- browserauthor.Event{Result: &authorsession.Result{ArtifactPath: "/private/result.json"}, Attestation: &browserauthor.Attestation{}}
			}
			type outcome struct {
				result   liveProtocolResult
				rejected bool
				class    string
				err      error
			}
			returned := make(chan outcome, 1)
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			go func() {
				r, rejected, class, err := runBrowserScenarioSession(ctx, request, session)
				returned <- outcome{r, rejected, class, err}
			}()
			select {
			case <-returned:
				t.Fatal("controller returned before worker event stream joined")
			case <-session.joining:
			case <-ctx.Done():
				t.Fatal("controller did not begin joining worker")
			}
			select {
			case <-returned:
				t.Fatal("controller returned while worker remained active")
			default:
			}
			if kind == "failure" || kind == "expected_rejection" || kind == "rejection_late_failure" {
				select {
				case <-session.canceled:
				default:
					t.Fatal("non-success did not cancel worker before joining")
				}
			}
			if kind == "late_failure" || kind == "rejection_late_failure" {
				session.events <- browserauthor.Event{State: "failed", ErrorCode: "worker_teardown"}
			}
			session.close()
			select {
			case got := <-returned:
				switch kind {
				case "failure":
					if got.err == nil {
						t.Fatal("initial failure was lost")
					}
				case "expected_rejection":
					if got.err != nil || !got.rejected || got.class != "output_bound" {
						t.Fatal("clean joined expected rejection changed")
					}
				case "success":
					if got.err != nil || got.result.ArtifactPath == "" || got.rejected {
						t.Fatal("clean joined result was lost")
					}
				case "late_failure", "rejection_late_failure":
					if got.err == nil || !strings.Contains(got.err.Error(), "worker_teardown") || got.result.ArtifactPath != "" || got.rejected {
						t.Fatal("late teardown failure did not invalidate result")
					}
				}
			case <-ctx.Done():
				t.Fatal("controller did not return after worker closed")
			}
		})
	}
}
