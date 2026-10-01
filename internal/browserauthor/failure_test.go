package browserauthor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
)

func TestAuthorFailureDetailsRetainClosedWorkerAndStreamClasses(t *testing.T) {
	diagnostic := func(code string) string {
		return fmt.Sprintf("printf '%%s\\n' '{\"protocol\":\"browsertools.author-session.v2\",\"type\":\"diagnostic\",\"diagnostic\":{\"code\":\"%s\"}}'\n", code)
	}
	result := "printf '%s\\n' '{\"protocol\":\"browsertools.author-session.v2\",\"type\":\"result\",\"result\":{\"artifactPath\":\"/private/result.json\",\"digest\":\"sha256:" + strings.Repeat("a", 64) + "\"}}'\n"
	for _, tc := range []struct {
		name, body, code, worker, phase, stream string
		beforeState, success                    bool
	}{
		{name: "browser failure", body: diagnostic("browser_failure") + "exit 1", code: "worker_protocol", worker: "browser_failure", phase: "receive", stream: "eof"},
		{name: "open failure before state", body: diagnostic("browser_failure") + "exit 1", code: "worker_protocol", worker: "browser_failure", phase: "receive", stream: "eof", beforeState: true},
		{name: "unknown code", body: diagnostic("token_canary_0123456789") + "exit 1", code: "worker_protocol", worker: "unknown", phase: "receive", stream: "eof"},
		{name: "early zero exit", body: "exit 0", code: "worker_protocol", worker: "none", phase: "receive", stream: "eof"},
		{name: "early nonzero exit", body: "exit 7", code: "worker_protocol", worker: "none", phase: "receive", stream: "eof"},
		{name: "decode", body: "printf '%s\\n' 'raw-canary'", code: "worker_protocol", worker: "none", phase: "receive", stream: "decode"},
		{name: "size", body: "printf '%s\\n' '" + strings.Repeat("x", maxProtocolLine+1) + "'", code: "worker_protocol", worker: "none", phase: "receive", stream: "size"},
		{name: "trailing decode", body: result + "printf '%s\\n' 'raw-canary'", code: "worker_protocol", worker: "none", phase: "drain", stream: "decode"},
		{name: "trailing message", body: result + diagnostic("browser_failure"), code: "worker_protocol", worker: "none", phase: "drain", stream: "trailing_message"},
		{name: "terminal exit", body: result + "exit 7", code: "worker_exit", worker: "none", phase: "none", stream: "none"},
		{name: "informational then success", body: diagnostic("synthetic_notice") + result + "exit 0", success: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session := startFailureWorker(t, tc.body, tc.beforeState)
			var terminal Event
			results := 0
			for event := range session.Events() {
				if event.Result != nil {
					results++
					if event.Failure != nil {
						t.Fatal("success acquired private failure")
					}
				}
				if event.ErrorCode != "" {
					terminal = event
				}
				wire, _ := json.Marshal(event)
				for _, secret := range []string{"worker_diagnostic", "stream_failure", "raw-canary", "token_canary"} {
					if strings.Contains(string(wire), secret) {
						t.Fatal("private details reached public event")
					}
				}
			}
			if tc.success {
				if results != 1 || terminal.ErrorCode != "" {
					t.Fatal("informational diagnostic broke clean success")
				}
				return
			}
			want := FailureDetails{WorkerDiagnostic: tc.worker, StreamPhase: tc.phase, StreamFailure: tc.stream}
			if results != 0 || terminal.ErrorCode != tc.code || terminal.Failure == nil || *terminal.Failure != want || !terminal.Failure.Valid() {
				t.Fatalf("wrong closed failure: code=%s detail=%+v", terminal.ErrorCode, terminal.Failure)
			}
		})
	}
}

func startFailureWorker(t *testing.T, body string, beforeState bool) *Session {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	worker := filepath.Join(t.TempDir(), "worker")
	script := `#!/bin/sh
printf '%s\n' '{"protocol":"browsertools.author-session.v2","type":"hello","capabilities":["chromium","human_credentials","reviewed_mfa_kind","reviewed_outputs","reduced_observation","popup","frame","typed_goal"]}'
IFS= read -r start || exit 2
`
	if !beforeState {
		script += `printf '%s\n' '{"protocol":"browsertools.author-session.v2","type":"state","phase":"authentication","context":"main","bounds":{"navigationTimeoutMs":20000,"totalTimeoutMs":600000,"maxRequests":512,"maxResponseBytes":33554432,"maxObservations":64,"maxCandidates":128,"maxOutputs":16}}'
IFS= read -r observe || exit 2
`
	}
	script += body + "\n"
	return startFailureScript(t, root, worker, script)
}

func startFailureScript(t *testing.T, root, worker, script string) *Session {
	t.Helper()
	if err := os.WriteFile(worker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	session, err := StartExternal(ctx, Config{PrivateRoot: root, InitialURL: "http://127.0.0.1:12345/login", DashboardURL: "http://127.0.0.1:12345/dashboard", Goal: "review dashboard", Origins: []string{"http://127.0.0.1:12345"}, ProfileID: "member", GoalPredicate: authorresult.GoalPredicate{Origin: "http://127.0.0.1:12345", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard"}}, worker)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Cancel)
	return session
}

func TestAuthorNegotiationRetainsStreamClass(t *testing.T) {
	for _, tc := range []struct{ body, stream string }{
		{"exit 0", "eof"},
		{"printf '%s\\n' 'private-negotiation-canary'", "decode"},
		{"printf '%s\\n' '" + strings.Repeat("x", maxProtocolLine+1) + "'", "size"},
	} {
		t.Run(tc.stream, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Chmod(root, 0700); err != nil {
				t.Fatal(err)
			}
			session := startFailureScript(t, root, filepath.Join(t.TempDir(), "worker"), "#!/bin/sh\n"+tc.body+"\n")
			var terminal Event
			for event := range session.Events() {
				terminal = event
			}
			if terminal.ErrorCode != "protocol_negotiation" || terminal.Failure == nil || terminal.Failure.StreamPhase != "receive" || terminal.Failure.StreamFailure != tc.stream {
				t.Fatal("negotiation stream class lost")
			}
		})
	}
}

func TestAuthorReceivePreservesFinalScannerStatus(t *testing.T) {
	for _, err := range []error{io.EOF, errAuthorDecode, errAuthorRead, errAuthorSize} {
		for range 100 {
			messages := make(chan authorsession.ServerMessage)
			done := make(chan error, 1)
			done <- err
			close(messages)
			_, got := receive(t.Context(), messages, done)
			if !errors.Is(got, err) {
				t.Fatal("closed stream masked final scanner status")
			}
		}
	}
}

type failingAuthorReader struct{}

func (failingAuthorReader) Read([]byte) (int, error) { return 0, errors.New("private-read-canary") }

func TestAuthorScannerReducesReadError(t *testing.T) {
	messages := make(chan authorsession.ServerMessage)
	done := make(chan error, 1)
	scanMessages(t.Context(), failingAuthorReader{}, messages, done)
	_, err := receive(t.Context(), messages, done)
	if !errors.Is(err, errAuthorRead) || strings.Contains(err.Error(), "canary") {
		t.Fatal("read error was not reduced")
	}
}

func TestAuthorFailureDetailsVocabulary(t *testing.T) {
	for _, d := range []FailureDetails{{"raw_canary", "none", "none"}, {"browser_failure", "none", "eof"}, {"browser_failure", "drain", "eof"}, {"none", "receive", "trailing_message"}, {"none", "receive", "canary"}, {"none", "canary", "read"}, {}} {
		if d.Valid() {
			t.Fatal("invalid private diagnostic admitted")
		}
	}
}
