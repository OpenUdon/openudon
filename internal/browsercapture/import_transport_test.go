package browsercapture

import (
	"bufio"
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browserauthor"
)

func TestProfileImportIsSeparatelyApprovedAfterWorkerJoin(t *testing.T) {
	for _, mode := range []string{Authenticated, Registration} {
		for _, outcome := range []string{"approve", "refuse", "stale", "expiry", "commit_failure"} {
			t.Run(mode+"/"+outcome, func(t *testing.T) {
				input, client := io.Pipe()
				reader, output := io.Pipe()
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				defer client.Close()
				defer reader.Close()
				updates := make(chan int)
				joined := make(chan struct{})
				release := make(chan struct{})
				go func() { updates <- 1; <-release; close(joined); close(updates) }()
				var commits atomic.Int32
				done := make(chan error, 1)
				ttl := 2 * time.Second
				if outcome == "expiry" {
					ttl = 100 * time.Millisecond
				}
				go func() {
					_, err := drive(ctx, mode, ttl, input, output,
						func(context.Context) (<-chan int, func(), error) { return updates, func() {}, nil },
						func(int) (string, View, bool) { return "result", View{State: "captured"}, true },
						func(View, Command) error { return errors.New("no native command after completion") },
						func(context.Context, Command) error { t.Error("joined native worker was dispatched"); return nil }, nil, nil,
						func(context.Context, int) (*profileImport, error) {
							select {
							case <-joined:
							default:
								t.Error("review before worker joined")
							}
							return &profileImport{result: Result{ProfileID: "reviewed-profile", TransactionSHA256: strings.Repeat("a", 64), Effect: "write"}, commit: func(context.Context) error {
								commits.Add(1)
								if outcome == "commit_failure" {
									return errors.New("PRIVATE_IMPORT_FAILURE")
								}
								return nil
							}}, nil
						})
					done <- err
				}()
				// The terminal producer update is not an import approval or a durable write.
				close(release)
				h := &authHarness{in: client, out: bufio.NewReader(reader)}
				review := h.read(t)
				if review.Type != "state" || review.View.State != "import_review" || commits.Load() != 0 {
					t.Fatal("import review authority lost")
				}
				if outcome == "expiry" {
					select {
					case err := <-done:
						if !errors.Is(err, ErrCanceled) {
							t.Fatal(err)
						}
					case <-time.After(time.Second):
						t.Fatal("expiry did not close import")
					}
					if commits.Load() != 0 {
						t.Fatal("expired import wrote")
					}
					return
				}
				command := Command{Authentication: &browserauthor.Response{Kind: "confirm", Confirmed: true}}
				if mode == Registration {
					command = Command{Registration: &RegistrationCommand{Type: "finish", Confirmed: true}}
				}
				card := h.review(t, review, command)
				if card.Action == nil || commits.Load() != 0 {
					t.Fatal("proposal imported before approval")
				}
				if outcome == "stale" {
					h.send(t, Decision{Binding: review.Binding, Type: "decide", ActionID: card.Action.ID, CommandSHA256: card.Action.CommandSHA256, Approved: true})
				} else {
					h.decide(t, card, outcome != "refuse")
				}
				if outcome == "refuse" {
					fresh := h.read(t)
					if fresh.View.State != "import_review" || commits.Load() != 0 {
						t.Fatal("refusal changed package")
					}
					h.send(t, Cancellation{Binding: fresh.Binding, Type: "cancel"})
				}
				if outcome == "approve" {
					imported := h.read(t)
					if imported.Type != "result" || imported.View.State != "imported" || imported.View.Result.Effect != "write" {
						t.Fatal("final import metadata invalid")
					}
				}
				select {
				case err := <-done:
					if outcome == "approve" {
						if err != nil || commits.Load() != 1 {
							t.Fatal(err, commits.Load())
						}
					} else if err == nil || strings.Contains(err.Error(), "PRIVATE_IMPORT_FAILURE") {
						t.Fatal("failure leaked or disappeared", err)
					}
				case <-time.After(time.Second):
					t.Fatal("import did not join")
				}
				if outcome == "refuse" || outcome == "stale" {
					if commits.Load() != 0 {
						t.Fatal("unapproved import wrote")
					}
				}
			})
		}
	}
}
