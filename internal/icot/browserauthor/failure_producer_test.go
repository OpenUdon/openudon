package browserauthor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
)

type diagnosticFailureBrowser struct{ session *diagnosticFailureSession }

func (b diagnosticFailureBrowser) Open(context.Context, authorsession.BrowserRequest) (authorsession.Session, error) {
	return b.session, nil
}

type diagnosticFailureSession struct{ closed bool }

func (*diagnosticFailureSession) Observe(context.Context, string) (authorsession.RawObservation, error) {
	return authorsession.RawObservation{}, errors.New("unexpected_navigation private-backend-canary")
}
func (*diagnosticFailureSession) Focus(context.Context, authorsession.BrowserAction) error {
	return errors.New("unexpected call")
}
func (*diagnosticFailureSession) Execute(context.Context, authorsession.BrowserAction) (authorsession.Execution, error) {
	return authorsession.Execution{}, errors.New("unexpected call")
}
func (*diagnosticFailureSession) AddOrigin(string) error { return errors.New("unexpected call") }
func (s *diagnosticFailureSession) Close() error         { s.closed = true; return nil }

func TestAuthorActualProducerFailureRetained(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	goal := authorresult.GoalPredicate{Origin: "http://127.0.0.1:12345", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard"}
	bounds := authorresult.Bounds{NavigationTimeoutMS: 20000, TotalTimeoutMS: 600000, MaxRequests: 512, MaxResponseBytes: 33554432, MaxObservations: 64, MaxCandidates: 128, MaxOutputs: 16}
	start := authorsession.ClientMessage{Protocol: authorsession.Protocol, Type: "start", Title: "member", URL: "http://127.0.0.1:12345/login", DashboardURL: "http://127.0.0.1:12345/dashboard", Goal: "review dashboard", Origins: []string{goal.Origin}, GoalPredicate: &goal, Bounds: &bounds}
	var input, output bytes.Buffer
	for _, m := range []authorsession.ClientMessage{start, {Protocol: authorsession.Protocol, Type: "observe", Context: "main"}} {
		if err := json.NewEncoder(&input).Encode(m); err != nil {
			t.Fatal(err)
		}
	}
	backend := &diagnosticFailureSession{}
	err := authorsession.Serve(t.Context(), &input, &output, diagnosticFailureBrowser{backend}, authorsession.ServeOptions{PrivateRoot: root, Clock: func() time.Time { return time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC) }})
	if err == nil || !backend.closed {
		t.Fatal("synthetic backend failure was not closed")
	}
	if !bytes.Contains(output.Bytes(), []byte(`"code":"browser_failure"`)) || bytes.Contains(output.Bytes(), []byte("private-backend-canary")) {
		t.Fatal("producer failed to retain closed diagnostic safely")
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatal("unexpected producer message count")
	}
	for _, line := range lines {
		if _, err := decodeServerMessage([]byte(line)); err != nil {
			t.Fatal("actual producer message failed decoder")
		}
	}
	session := startFailureWorker(t, "printf '%s\\n' '"+lines[2]+"'\nexit 1\n", false)
	var terminal Event
	for event := range session.Events() {
		if event.ErrorCode != "" {
			terminal = event
		}
	}
	if terminal.ErrorCode != "worker_protocol" || terminal.Failure == nil || terminal.Failure.WorkerDiagnostic != "browser_failure" || terminal.Failure.StreamFailure != "eof" {
		t.Fatal("actual producer failure lost")
	}
}
