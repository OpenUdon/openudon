package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/OpenUdon/browsertools/authordiagnostic"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

func TestPrivateCaptureDiagnosticPreservesDetailsOutsideApplication(t *testing.T) {
	details := browserauthor.FailureDetails{WorkerDiagnostic: "browser_failure", StreamPhase: "receive", StreamFailure: "eof"}
	none := browserauthor.NoFailureDetails()
	var messages []string
	for _, event := range []browserauthor.Event{
		{State: "failed", ErrorCode: "worker_protocol", Failure: &details},
		{State: "failed", ErrorCode: "attestation", Failure: &none},
		{State: "failed", ErrorCode: "absolute_timeout", Failure: &none},
		{State: "canceled", ErrorCode: "operator_idle_timeout", Failure: &none},
		{State: "failed", ErrorCode: "worker_teardown", Failure: &details},
		{State: "failed", ErrorCode: "TOKEN_CANARY", Failure: &none},
	} {
		fake := &fakeEngine{}
		session := &diagnosticFakeSession{fakeCaptureSession: newFakeCaptureSession()}
		directory := t.TempDir()
		_ = os.Chmod(directory, 0700)
		path := filepath.Join(directory, "diagnostic.json")
		handler, err := NewHandler(HandlerConfig{
			CaptureDiagnostic: &CaptureDiagnosticConfig{Path: path, Attempt: "fresh_attempt", Binding: strings.Repeat("a", 64)},
			Engine:            fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority,
			PrivateRoot: "/tmp/private",
			DoctorBrowser: func(context.Context, string, string) (browserauthor.DoctorReport, error) {
				return browserauthor.DoctorReport{Version: browserauthor.DoctorVersion, Engine: browserauthor.EngineChromium, DriverReady: true, BrowserReady: true}, nil
			},
			StartCapture: func(context.Context, browserauthor.Config) (CaptureSession, error) { return session, nil },
		})
		if err != nil {
			t.Fatal(err)
		}
		initial := currentResponse(t, handler)
		preflight := doRequest(handler, http.MethodPost, "/api/v4/browser/preflight", fmt.Sprintf(`{"revision":%q,"capture_revision":%q}`, initial.Revision, initial.CaptureRevision), "application/json", true)
		if preflight.Code != http.StatusOK {
			t.Fatal("preflight")
		}
		configured := decodeResponse(t, preflight)
		started := doRequest(handler, http.MethodPost, "/api/v4/capture/start", fmt.Sprintf(`{"revision":%q,"capture_revision":%q,"profile_id":"account","url":"http://127.0.0.1:12345/login","dashboard_url":"http://127.0.0.1:12345/campaigns","goal":"Prove campaign presence","origins":["http://127.0.0.1:12345"]}`, configured.Revision, configured.CaptureRevision), "application/json", true)
		if started.Code != http.StatusAccepted {
			t.Fatal("capture start")
		}
		session.events <- event
		_ = waitForCaptureState(t, handler, "canceling")
		pending, _ := os.ReadFile(path)
		if len(pending) != 0 {
			t.Fatal("diagnostic emitted before event closure")
		}
		close(session.events)
		final := waitForCaptureState(t, handler, event.State)
		wire, err := json.Marshal(final.Capture)
		if err != nil {
			t.Fatal(err)
		}
		for _, missing := range []string{"worker_protocol", "attestation", "browser_failure", "worker_diagnostic", "stream_failure", "error_code"} {
			if strings.Contains(string(wire), missing) {
				t.Fatal("diagnostic became visible unexpectedly")
			}
		}
		messages = append(messages, final.Capture.Message)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var diagnostic captureDiagnosticV2
		if evidencefile.DecodeStrict(data, &diagnostic) != nil {
			t.Fatal("missing private diagnostic")
		}
		if diagnostic.Code != captureDiagnosticCode(event.ErrorCode) || diagnostic.Failure != *event.Failure || diagnostic.Backend.Stage != "navigation" || diagnostic.Backend.Reason != "transport" || !diagnostic.EventsClosed || diagnostic.State != event.State {
			t.Fatalf("diagnostic mismatch: %+v", diagnostic)
		}
		if diagnostic.Version != "openudon.capture-diagnostic.v2" || diagnostic.Rejection != authordiagnostic.NoRejection() {
			t.Fatal("private rejection metadata mismatch")
		}
		if strings.Contains(string(data), "TOKEN_CANARY") || strings.Contains(string(wire), "navigation") {
			t.Fatal("private value leaked")
		}
		if _, err := newCaptureDiagnostic(&CaptureDiagnosticConfig{Path: path, Attempt: "fresh_attempt", Binding: strings.Repeat("a", 64)}); err == nil {
			t.Fatal("diagnostic overwrite")
		}
		if !handler.(*Server).captureDiagnostic.used {
			t.Fatal("capture not consumed")
		}
	}
	if messages[0] == "" || messages[0] != messages[1] {
		t.Fatal("different failures did not reproduce the same public message")
	}
}

type diagnosticFakeSession struct{ *fakeCaptureSession }

func (*diagnosticFakeSession) BackendDiagnostic() (string, authordiagnostic.Class) {
	return "available", authordiagnostic.Class{Stage: "navigation", Reason: "transport"}
}

type rejectionFakeSession struct {
	*fakeCaptureSession
	rejection authordiagnostic.Rejection
}

func (*rejectionFakeSession) BackendDiagnostic() (string, authordiagnostic.Class) {
	return "available", authordiagnostic.Class{Stage: "policy", Reason: "origin_escape"}
}
func (s *rejectionFakeSession) BackendRejectionDiagnostic() authordiagnostic.Rejection {
	return s.rejection
}

func TestCaptureRejectionV2ProjectionAndMalformedDetail(t *testing.T) {
	for _, detail := range []authordiagnostic.Rejection{
		{Boundary: "request", Resource: "document", OriginRelation: "port_mismatch"},
		{Boundary: "observation", Resource: "document", OriginRelation: "host_mismatch"},
		{Boundary: "request", Resource: "TOKEN_CANARY", OriginRelation: "host_mismatch"},
	} {
		dir := t.TempDir()
		_ = os.Chmod(dir, 0700)
		path := filepath.Join(dir, "diagnostic")
		sink, err := newCaptureDiagnostic(&CaptureDiagnosticConfig{Path: path, Attempt: "fresh", Binding: strings.Repeat("a", 64)})
		if err != nil {
			t.Fatal(err)
		}
		s := &Server{captureDiagnostic: sink}
		s.finishCaptureDiagnostic(&rejectionFakeSession{newFakeCaptureSession(), detail}, "failed", "worker_protocol", nil)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var r captureDiagnosticV2
		if evidencefile.DecodeStrict(data, &r) != nil || r.Version != "openudon.capture-diagnostic.v2" {
			t.Fatal("invalid diagnostic")
		}
		valid := detail.ValidFor(authordiagnostic.Class{Stage: "policy", Reason: "origin_escape"})
		if valid && (r.Rejection != detail || r.BackendStatus != "available") {
			t.Fatal("lost rejection detail")
		}
		if !valid && (r.Rejection != authordiagnostic.NoRejection() || r.BackendStatus != "invalid") {
			t.Fatal("accepted malformed detail")
		}
		if strings.Contains(string(data), "CANARY") {
			t.Fatal("leaked private value")
		}
	}
}

func TestPrivateCaptureDiagnosticWriteFailureBlocksResult(t *testing.T) {
	dir := t.TempDir()
	_ = os.Chmod(dir, 0700)
	sink, err := newCaptureDiagnostic(&CaptureDiagnosticConfig{Path: filepath.Join(dir, "diagnostic.json"), Attempt: "write_failure", Binding: strings.Repeat("a", 64)})
	if err != nil {
		t.Fatal(err)
	}
	_ = sink.file.Close() // deterministic disk/output failure without a browser
	s := &Server{captureDiagnostic: sink, capture: &CaptureState{State: "stage_review", ResultReady: true}}
	s.finishCaptureDiagnostic(nil, "stage_review", "", nil)
	if !s.captureContainmentFailed || s.capture.State != "failed" || s.capture.ResultReady || sink.file != nil {
		t.Fatal("diagnostic failure left a promotable capture")
	}
}
