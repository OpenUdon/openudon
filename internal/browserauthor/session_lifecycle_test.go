package browserauthor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
)

func TestExternalAuthorResultRequiresCleanWorkerExit(t *testing.T) {
	for _, test := range []struct {
		name, trailer, code string
		wantResult          bool
	}{
		{name: "clean", trailer: "exit 0", wantResult: true},
		{name: "nonzero", trailer: "exit 7", code: "worker_exit"},
		{name: "malformed trailing output", trailer: "printf '%s\\n' 'not-json'\nexit 0", code: "worker_protocol"},
		{name: "unexpected trailing message", trailer: `printf '%s\n' '{"protocol":"browsertools.author-session.v2","type":"diagnostic","diagnostic":{"code":"late_failure"}}'` + "\nexit 0", code: "worker_protocol"},
	} {
		t.Run(test.name, func(t *testing.T) {
			session := startLifecycleWorker(t, test.trailer)
			results, code := 0, ""
			for event := range session.Events() {
				if event.Result != nil {
					results++
				}
				if event.ErrorCode != "" {
					code = event.ErrorCode
				}
			}
			if (results == 1) != test.wantResult || results > 1 || code != test.code {
				t.Fatalf("result count %d and code %q; want result=%t, code=%q", results, code, test.wantResult, test.code)
			}
		})
	}
}

func TestAuthorScannerCancellationJoinsBlockedDelivery(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	messages := make(chan authorsession.ServerMessage)
	failures := make(chan error, 1)
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		scanMessages(ctx, reader, messages, failures)
	}()
	written := make(chan error, 1)
	go func() {
		_, err := io.WriteString(writer, "{\"protocol\":\"browsertools.author-session.v2\",\"type\":\"hello\",\"capabilities\":[]}\n")
		written <- err
	}()
	select {
	case err := <-written:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("scanner did not read the helper message")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("scanner retained a blocked delivery after cancellation")
	}
	if err := <-failures; !errors.Is(err, context.Canceled) {
		t.Fatal("scanner did not retain cancellation")
	}
}

func TestAuthorDrainDoesNotMaskMalformedTrailerAsEOF(t *testing.T) {
	for range 100 {
		messages := make(chan authorsession.ServerMessage)
		failures := make(chan error, 1)
		malformed := errors.New("malformed trailer")
		failures <- malformed
		close(messages)
		if err := drainAuthorOutput(t.Context(), messages, failures); !errors.Is(err, malformed) {
			t.Fatal("closed message stream masked malformed trailing bytes")
		}
	}
}

func startLifecycleWorker(t *testing.T, trailer string) *Session {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test worker uses a POSIX script")
	}
	privateRoot := t.TempDir()
	if err := os.Chmod(privateRoot, 0700); err != nil {
		t.Fatal(err)
	}
	worker := filepath.Join(t.TempDir(), "worker")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' '{"protocol":"browsertools.author-session.v2","type":"hello","capabilities":["chromium","human_credentials","reviewed_mfa_kind","reviewed_outputs","reduced_observation","popup","frame","typed_goal"]}'
IFS= read -r start
printf '%%s\n' '{"protocol":"browsertools.author-session.v2","type":"state","phase":"authentication","context":"main","bounds":{"navigationTimeoutMs":20000,"totalTimeoutMs":600000,"maxRequests":512,"maxResponseBytes":33554432,"maxObservations":64,"maxCandidates":128,"maxOutputs":16}}'
IFS= read -r observe
printf '%%s\n' '{"protocol":"browsertools.author-session.v2","type":"result","result":{"artifactPath":"/private/result.json","digest":"sha256:%s"}}'
%s
`, strings.Repeat("a", 64), trailer)
	if err := os.WriteFile(worker, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	session, err := StartExternal(ctx, Config{
		PrivateRoot: privateRoot, InitialURL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard",
		Goal: "review dashboard", Origins: []string{"https://members.example.test"}, ProfileID: "member",
		GoalPredicate: authorresult.GoalPredicate{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard"},
	}, worker)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(session.Cancel)
	return session
}
