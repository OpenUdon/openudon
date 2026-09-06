package ui

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

type applicationTestClient struct {
	input   *io.PipeWriter
	output  *io.PipeReader
	encoder *json.Encoder
	decoder *json.Decoder
	done    chan error
	current Response
}

func applicationClient(t *testing.T, app Application) *applicationTestClient {
	t.Helper()
	in, write := io.Pipe()
	read, out := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	c := &applicationTestClient{input: write, output: read, encoder: json.NewEncoder(write), decoder: json.NewDecoder(read), done: make(chan error, 1)}
	go func() { c.done <- serveApplicationControl(ctx, app, in, out); _ = out.Close() }()
	t.Cleanup(func() {
		_ = write.Close()
		_ = read.Close()
		cancel()
		select {
		case <-c.done:
		case <-time.After(time.Second):
			t.Error("control did not stop")
		}
	})
	c.read(t)
	return c
}
func (c *applicationTestClient) read(t *testing.T) string {
	t.Helper()
	var state ApplicationControlState
	if err := c.decoder.Decode(&state); err != nil {
		t.Fatal("control output", err)
	}
	if state.Version != ApplicationControlVersion || json.Unmarshal(state.Application, &c.current) != nil {
		t.Fatal("state schema")
	}
	return state.Failure
}
func (c *applicationTestClient) send(t *testing.T, operation string, request any) string {
	t.Helper()
	frame := registrationControlFrame{Version: ApplicationControlVersion, Operation: operation}
	if request != nil {
		frame.Request, _ = json.Marshal(request)
	}
	if c.encoder.Encode(frame) != nil {
		t.Fatal("control input")
	}
	return c.read(t)
}
func TestApplicationControlSharesUIRevisionAndRejectsUnauthorizedWrites(t *testing.T) {
	registration, _ := controlApplication(t)
	app := Application{registration.server}
	c := applicationClient(t, app)
	initial := c.current.Revision
	if got := c.send(t, "author.approve", approveRequest{Revision: "stale", HumanApproved: true}); got != "stale_revision" {
		t.Fatal(got)
	}
	if got := c.send(t, "package.build", buildRequest{Revision: initial, Confirmed: true}); got != "invalid_lifecycle" {
		t.Fatal(got)
	}
	if got := c.send(t, "author.round", roundRequest{Revision: initial}); got != "malformed_request" {
		t.Fatal(got)
	}
	if c.current.Revision != initial || c.current.Lifecycle != lifecycleAuthoring {
		t.Fatal("rejected command changed lifecycle")
	}
	if got := c.send(t, "transaction.prepare", map[string]any{"unexpected": "value"}); got != "malformed_request" {
		t.Fatal(got)
	}
}
func TestApplicationControlClosedProtocol(t *testing.T) {
	for _, text := range []string{
		`{"version":"openudon.application-control.v1","operation":"run"}`,
		`{"version":"openudon.application-control.v1","operation":"snapshot","script":"x"}`,
		`{"version":"openudon.application-control.v1","operation":"snapshot","operation":"close"}`,
		`{"version":"openudon.application-control.v1","operation":"capture.respond","request":{"response":{"kind":"continue","value":"secret"}}}`,
		`{"version":"openudon.application-control.v1","operation":"author.approve","request":null}`,
		strings.Repeat("x", MaxRequestBytes+1),
	} {
		t.Run("closed", func(t *testing.T) {
			registration, _ := controlApplication(t)
			err := serveApplicationControl(context.Background(), Application{registration.server}, io.NopCloser(strings.NewReader(text+"\n")), io.Discard)
			if err == nil {
				t.Fatal("invalid control accepted")
			}
		})
	}
}
func TestApplicationControlEOFInterruptsRunningReadiness(t *testing.T) {
	registration, _ := controlApplication(t)
	s := registration.server
	entered, exited := make(chan struct{}), make(chan struct{})
	s.doctorBrowser = func(ctx context.Context, _, _ string) (browserauthor.DoctorReport, error) {
		close(entered)
		<-ctx.Done()
		close(exited)
		return browserauthor.DoctorReport{}, ctx.Err()
	}
	c := applicationClient(t, Application{s})
	raw, _ := json.Marshal(captureMutationRequest{Revision: c.current.Revision, CaptureRevision: c.current.CaptureRevision})
	if c.encoder.Encode(registrationControlFrame{Version: ApplicationControlVersion, Operation: "browser.preflight", Request: raw}) != nil {
		t.Fatal("input")
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("doctor not entered")
	}
	_ = c.input.Close()
	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("EOF did not cancel operation")
	}
}
func TestApplicationControlCancellationInterruptsBlockedOutput(t *testing.T) {
	registration, _ := controlApplication(t)
	reader, writer := io.Pipe()
	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := serveApplicationControl(ctx, Application{registration.server}, io.NopCloser(strings.NewReader("")), writer); err == nil || time.Since(start) > time.Second {
		t.Fatal("blocked output not canceled")
	}
}
