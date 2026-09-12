package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

func TestConsumerRegistrationAuthorityIsSharedByHTTPAndControl(t *testing.T) {
	for _, transport := range []string{"http", "control"} {
		for _, change := range []string{"valid", "profile", "version", "origin", "url", "expired"} {
			t.Run(transport+"/"+change, func(t *testing.T) {
				now := time.Now().UTC().Truncate(time.Second)
				authority := &RegistrationAuthority{Version: "openudon.registration-authority.v1", ProfileID: "synthetic", ProfileVersion: "1.1", InitialURL: "https://app.example.test/register?action=startnew", Origins: []string{"https://app.example.test"}, NavigationURLs: []string{"https://app.example.test/register?action=startnew"}, ExpiresAt: now.Add(10 * time.Minute).Format(time.RFC3339)}
				fake := &fakeEngine{}
				session := newFakeRegistrationAuthoringSession()
				starts := 0
				handler, err := NewHandler(HandlerConfig{Context: context.Background(), Engine: fake, Snapshot: fake.snapshot, ExampleDir: "/tmp/example", Token: testToken, AccessCode: testAccessCode, Authority: testAuthority, PrivateRoot: "/tmp/private", BrowserTransactions: newFakeBrowserTransactions(), Now: func() time.Time { return now }, RegistrationAuthority: authority, StartRegistration: func(context.Context, browserauthor.RegistrationConfig) (RegistrationAuthoringSession, error) {
					starts++
					return session, nil
				}})
				if err != nil {
					t.Fatal(err)
				}
				app := RegistrationApplication{server: handler.(*Server)}
				defer app.close()
				state := app.observe()
				request := registrationAuthoringStartRequest{Revision: state.Revision, RegistrationRevision: state.RegistrationRevision, ProfileID: authority.ProfileID, ProfileVersion: authority.ProfileVersion, URL: authority.InitialURL, Origins: append([]string(nil), authority.Origins...)}
				// Caller mutation must not widen the retained authority or its snapshot.
				authority.NavigationURLs[0] = "https://app.example.test/escape"
				if app.server.registrationAuthority.allowsNavigation(authority.NavigationURLs[0], now) {
					t.Fatal("authority aliases caller memory")
				}
				switch change {
				case "profile":
					request.ProfileID = "different"
				case "version":
					request.ProfileVersion = "1.0"
				case "origin":
					request.Origins = []string{"https://other.example.test"}
				case "url":
					request.URL = "https://app.example.test/escape"
				case "expired":
					now = now.Add(11 * time.Minute)
				}
				data, _ := json.Marshal(request)
				if transport == "http" {
					response := doRequest(handler, http.MethodPost, "/api/v4/registration-authoring/start", string(data), "application/json", true)
					want := http.StatusForbidden
					if change == "valid" {
						want = http.StatusAccepted
					}
					if response.Code != want {
						t.Fatalf("status %d: %s", response.Code, response.Body.String())
					}
				} else {
					frame, _ := json.Marshal(registrationControlFrame{Version: RegistrationControlVersion, Operation: "start", Request: data})
					var out bytes.Buffer
					if err := serveRegistrationControl(context.Background(), app, io.NopCloser(bytes.NewReader(append(frame, '\n'))), &out); err != nil {
						t.Fatal(err)
					}
					if change != "valid" && !bytes.Contains(out.Bytes(), []byte("registration_authority")) {
						t.Fatal("control did not report scope denial")
					}
				}
				if (starts == 1) != (change == "valid") || app.server.registrationAttemptConsumed != (change == "valid") {
					t.Fatal("scope denial launched or consumed an attempt")
				}
				capture := (Application{server: app.server}).CaptureStart(context.Background(), captureStartRequest{})
				if capture.Failure == nil {
					t.Fatal("registration authority allowed another capture")
				}
			})
		}
	}
}

func TestRegistrationAuthorityFileIsClosedAndBounded(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	a := RegistrationAuthority{Version: "openudon.registration-authority.v1", ProfileID: "synthetic", ProfileVersion: "1.1", InitialURL: "https://app.example.test/register", Origins: []string{"https://app.example.test"}, NavigationURLs: []string{"https://app.example.test/register"}, ExpiresAt: now.Add(time.Minute).Format(time.RFC3339)}
	data, _ := json.Marshal(a)
	path := filepath.Join(t.TempDir(), "authority.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegistrationAuthority(path, now); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegistrationAuthority(path, now); err == nil {
		t.Fatal("public authority accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegistrationAuthority(path, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expired authority accepted")
	}
	link := path + "-link"
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadRegistrationAuthority(link, now); err == nil {
		t.Fatal("linked authority accepted")
	}
}
