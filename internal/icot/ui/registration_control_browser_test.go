//go:build browser_system_qualification

package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/processgroup"
)

// Exercise the shipped CLI, private pipes and real observation worker. Each
// ending receives fresh synthetic state and must join its entire process tree.
func TestBrowserSystemSupervisedControl(t *testing.T) {
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal("source")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	binary := filepath.Join(t.TempDir(), "icot")
	if err := processgroup.Run(ctx, time.Minute, processgroup.Invocation{Args: []string{"go", "build", "-o", binary, "./cmd/icot"}, Dir: root, Env: os.Environ(), Stdout: io.Discard, Stderr: io.Discard}); err != nil {
		t.Fatal("application_build")
	}
	var mutations atomic.Int64
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			mutations.Add(1)
			w.WriteHeader(405)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body><h1>Create account</h1><form method="post"><label>Email<input name="email" autocomplete="email"></label><label>Password<input type="password" name="password"></label><button type="submit">Register</button></form></body></html>`)
	}))
	defer fixture.Close()
	for _, ending := range []string{"cancel", "eof", "broken_protocol"} {
		t.Run(ending, func(t *testing.T) {
			state := t.TempDir()
			for _, name := range []string{"private", "scratch", "store"} {
				if os.Mkdir(filepath.Join(state, name), 0700) != nil {
					t.Fatal("state")
				}
			}
			parent := filepath.Join(root, "eval", "runs")
			if os.MkdirAll(parent, 0700) != nil {
				t.Fatal("workspace")
			}
			example, err := os.MkdirTemp(parent, ".browser-system-control-")
			if err != nil {
				t.Fatal("workspace")
			}
			defer os.RemoveAll(example)
			var stderr bytes.Buffer
			child, err := processgroup.StartInteractive(ctx, []string{binary, "control", "--no-open", "--network", "never", "--example", example, "--private-root", filepath.Join(state, "private"), "--package-scope", "qualification/control", "--package-scratch", filepath.Join(state, "scratch"), "--package-store", filepath.Join(state, "store")}, os.Environ(), &stderr)
			if err != nil {
				t.Fatal("application_start")
			}
			defer child.Terminate()
			encoder, decoder := json.NewEncoder(child.Input()), json.NewDecoder(child.Output())
			read := func() registrationControlState {
				var current registrationControlState
				if err := decoder.Decode(&current); err != nil || current.Version != RegistrationControlVersion {
					t.Fatal("application_protocol")
				}
				return current
			}
			send := func(operation string, request any) registrationControlState {
				frame := registrationControlFrame{Version: RegistrationControlVersion, Operation: operation}
				if request != nil {
					frame.Request, _ = json.Marshal(request)
				}
				if encoder.Encode(frame) != nil {
					t.Fatal("application_input")
				}
				return read()
			}
			await := func(expected string) registrationControlState {
				deadline := time.Now().Add(time.Minute)
				for time.Now().Before(deadline) {
					current := send("snapshot", nil)
					if current.State != nil && current.State.State == expected {
						return current
					}
					if current.Failure != "" || current.State != nil && current.State.State == "failed" {
						t.Fatal("worker_state")
					}
					time.Sleep(20 * time.Millisecond)
				}
				t.Fatal("worker_readiness")
				return registrationControlState{}
			}
			current := read()
			start := registrationAuthoringStartRequest{Revision: current.Revision, RegistrationRevision: "stale", ProfileID: "synthetic_control", URL: fixture.URL + "/register", Origins: []string{fixture.URL}}
			if stale := send("start", start); stale.Failure != "stale_revision" {
				t.Fatal("stale_start")
			}
			start.RegistrationRevision = current.RegistrationRevision
			if send("start", start).Failure != "" {
				t.Fatal("start")
			}
			current = await("observing")
			if send("command", registrationAuthoringCommandRequest{Revision: current.Revision, RegistrationRevision: current.RegistrationRevision, Type: "observe"}).Failure != "" {
				t.Fatal("observe")
			}
			current = await("observation")
			if current.State.Observation == nil || !current.State.AttemptConsumed {
				t.Fatal("observation")
			}
			if ending == "cancel" {
				if send("cancel", registrationAuthoringCancelRequest{RegistrationRevision: "stale"}).Failure != "stale_registration_revision" {
					t.Fatal("stale_cancel")
				}
				if send("cancel", registrationAuthoringCancelRequest{RegistrationRevision: current.RegistrationRevision}).Failure != "" {
					t.Fatal("cancel")
				}
				await("canceled")
				if send("start", start).Failure != "registration_authorization_consumed" {
					t.Fatal("retry")
				}
			} else if ending == "broken_protocol" {
				_, _ = io.WriteString(child.Input(), `{"version":"openudon.registration-control.v1","operation":"snapshot","operation":"start"}`+"\n")
			}
			_ = child.Input().Close()
			if _, err := io.Copy(io.Discard, child.Output()); err != nil {
				t.Fatal("protocol_drain")
			}
			err = child.Wait()
			if errors.Is(err, processgroup.ErrTerminationTimeout) || (ending == "broken_protocol") != (err != nil) {
				t.Fatal("application_teardown")
			}
			if ending == "broken_protocol" && stderr.String() != "icot control: protocol\n" {
				t.Fatal("protocol_failure_class")
			}
			if mutations.Load() != 0 {
				t.Fatal("observation_mutated_fixture")
			}
		})
	}
}
