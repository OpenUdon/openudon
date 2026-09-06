package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

func controlApplication(t *testing.T) (RegistrationApplication, *fakeRegistrationAuthoringSession) {
	t.Helper()
	fake := &fakeEngine{}
	session := newFakeRegistrationAuthoringSession()
	handler, err := NewHandler(HandlerConfig{Context: context.Background(), Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, PrivateRoot: "/tmp/private", BrowserTransactions: newFakeBrowserTransactions(), StartRegistration: func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
		return session, nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	app := RegistrationApplication{server: handler.(*Server)}
	t.Cleanup(func() {
		if err := app.close(); err != nil {
			t.Error(err)
		}
	})
	return app, session
}
func TestRegistrationControlRejectsBrokenUnionAndClosesInput(t *testing.T) {
	for _, input := range []string{`{"version":"wrong","operation":"snapshot"}`, `{"version":"openudon.registration-control.v1","operation":"snapshot","operation":"close"}`, `{"version":"openudon.registration-control.v1","operation":"evaluate","request":{}}`, `{"version":"openudon.registration-control.v1","operation":"snapshot","request":{}}`, strings.Repeat("x", MaxRequestBytes+1)} {
		app, _ := controlApplication(t)
		var out bytes.Buffer
		if err := serveRegistrationControl(context.Background(), app, io.NopCloser(strings.NewReader(input+"\n")), &out); err == nil {
			t.Fatal("invalid protocol accepted")
		}
		if strings.Contains(out.String(), "evaluate") || strings.Contains(out.String(), "wrong") {
			t.Fatal("input reflected")
		}
	}
}
func TestRegistrationControlAndUIShareRevisionAndAttempt(t *testing.T) {
	app, session := controlApplication(t)
	current := app.observe()
	request := registrationAuthoringStartRequest{Revision: current.Revision, RegistrationRevision: "stale", ProfileID: "synthetic", URL: "https://app.example.test/register", Origins: []string{"https://app.example.test"}}
	if result := app.Start(context.Background(), request); result.Failure == nil || result.Failure.Code != "stale_revision" {
		t.Fatal("stale mutation accepted")
	}
	request.RegistrationRevision = current.RegistrationRevision
	data, _ := json.Marshal(request)
	frame, _ := json.Marshal(registrationControlFrame{Version: RegistrationControlVersion, Operation: "start", Request: data})
	var out bytes.Buffer
	if err := serveRegistrationControl(context.Background(), app, io.NopCloser(bytes.NewReader(append(frame, '\n'))), &out); err != nil {
		t.Fatal(err)
	}
	if !app.server.registrationAttemptConsumed {
		t.Fatal("attempt not consumed")
	}
	if result := app.Start(context.Background(), request); result.Failure == nil || result.Failure.Code != "registration_authorization_consumed" {
		t.Fatal("automatic retry accepted")
	}
	session.Cancel()
	if err := app.close(); err != nil {
		t.Fatal(err)
	}
}
func TestRegistrationControlCancellationInterruptsBlockedInput(t *testing.T) {
	app, _ := controlApplication(t)
	reader, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := serveRegistrationControl(ctx, app, reader, io.Discard); err == nil || time.Since(start) > time.Second {
		t.Fatal("cancellation not bounded")
	}
	if _, err := writer.Write([]byte("x")); err == nil {
		t.Fatal("input remained open")
	}
}

func TestRegistrationControlCancellationInterruptsBlockedOutput(t *testing.T) {
	app, _ := controlApplication(t)
	reader, writer := io.Pipe()
	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	started := time.Now()
	if err := serveRegistrationControl(ctx, app, io.NopCloser(strings.NewReader("")), writer); err == nil || time.Since(started) > time.Second {
		t.Fatal("blocked output did not cancel")
	}
}
