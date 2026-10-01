package browserauthor

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRegistrationWorkerTerminalDiagnostics(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test worker uses a POSIX script")
	}
	for _, tc := range []struct{ name, diagnostic, state, code, retained string }{
		{"browser failure", `{"code":"browser_failure"}`, "failed", "worker_failed", "browser_failure"},
		{"observation failure", `{"code":"invalid_observation"}`, "failed", "worker_failed", "invalid_observation"},
		{"teardown failure", `{"code":"teardown_failure"}`, "failed", "worker_teardown", "teardown_failure"},
		{"canceled", `{"code":"canceled"}`, "canceled", "", "canceled"},
		{"warning is not failure", `{"code":"synthetic_fixture"}`, "failed", "malformed_diagnostic", ""},
		{"unknown private code", `{"code":"private-credential-canary"}`, "failed", "malformed_diagnostic", ""},
		{"missing diagnostic", `null`, "failed", "malformed_diagnostic", ""},
		{"extra private field", `{"code":"browser_failure","detail":"private-credential-canary"}`, "failed", "worker_protocol", ""},
		{"duplicate code", `{"code":"browser_failure","code":"canceled"}`, "failed", "worker_protocol", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			worker := writeRegistrationWorker(t, fmt.Sprintf(`#!/bin/sh
printf '%%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"hello","capabilities":["get_head_only","no_submit","reduced_observation","registration_review"]}'
IFS= read -r start
printf '%%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"diagnostic","diagnostic":%s}'
printf '%%s\n' 'private-credential-canary' >&2
exit 1
`, tc.diagnostic))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			session, err := StartExternalRegistration(ctx, registrationControllerConfig(registrationControllerRoot(t)), worker)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Cancel()
			wantEvent(t, session, "ready")
			sendRegistrationCommand(t, ctx, session, RegistrationCommand{Type: "start", ProfileID: "terminal_fixture", URL: "https://app.example.test/register", Origins: []string{"https://app.example.test"}})
			for event := range session.Events() {
				if event.Candidate != nil || strings.Contains(fmt.Sprint(event), "private-credential-canary") {
					t.Fatal("failure exported a private detail or candidate")
				}
			}
			terminal, ok := session.TerminalEvent()
			if !ok || terminal.State != tc.state || terminal.ErrorCode != tc.code || terminal.Diagnostic != tc.retained {
				t.Fatalf("terminal = %#v", terminal)
			}
		})
	}
}

func TestRegistrationTeardownRetainsFirstWorkerDiagnostic(t *testing.T) {
	s := &RegistrationSession{events: make(chan RegistrationEvent, 1)}
	s.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_failed", Diagnostic: "browser_failure"})
	s.publishTerminal(RegistrationEvent{State: "failed", ErrorCode: "worker_teardown"})
	event, _ := s.TerminalEvent()
	if event.ErrorCode != "worker_teardown" || event.Diagnostic != "browser_failure" {
		t.Fatalf("teardown obscured original closed cause: %#v", event)
	}
}
