package browserscenario

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoopbackRuntimeRequiresCredentialsChallengeAndSession(t *testing.T) {
	manifest := loopbackManifestForTest(t, "mfa-sms-otp")
	fixture, err := NewLoopbackFixture(manifest)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	fixture.SetRuntime(true)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	response, err := client.Get(fixture.DashboardURL())
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthenticated dashboard status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response, err = client.PostForm(fixture.InitialURL(), url.Values{"identifier": {"wrong"}, "password": {"wrong"}})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong credentials status = %d", response.StatusCode)
	}
	_ = response.Body.Close()

	response, err = client.PostForm(fixture.InitialURL(), url.Values{"identifier": {"member@example.test"}, "password": {"scenario-password-value"}})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !bodyContains(t, response, "Verification required") {
		t.Fatalf("valid login did not reach challenge: %d", response.StatusCode)
	}
	response, err = client.PostForm(fixture.server.URL+fixture.path("challenge"), url.Values{"challenge": {"000000"}})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong challenge status = %d", response.StatusCode)
	}
	_ = response.Body.Close()
	response, err = client.PostForm(fixture.server.URL+fixture.path("challenge"), url.Values{"challenge": {"123456"}})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || !bodyContains(t, response, "Member dashboard") {
		t.Fatalf("valid challenge did not reach dashboard: %d", response.StatusCode)
	}
	if !fixture.AuthenticatedReplayObserved() {
		t.Fatal("successful authenticated replay was not recorded")
	}
}

func TestLoopbackOriginEscapeUsesReachableAlternateOrigin(t *testing.T) {
	manifest := loopbackManifestForTest(t, "origin-escape-rejected")
	fixture, err := NewLoopbackFixture(manifest)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	if fixture.escapeOrigin == "" || fixture.escapeOrigin == fixture.server.URL || !strings.Contains(fixture.escapeOrigin, "localhost:") {
		t.Fatalf("alternate origin = %q, server origin = %q", fixture.escapeOrigin, fixture.server.URL)
	}
	response, err := http.Get(fixture.escapeOrigin + fixture.path("login"))
	if err != nil {
		t.Fatalf("alternate loopback origin is not reachable: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("alternate loopback origin status = %d", response.StatusCode)
	}
}

func TestLoopbackChallengeValuesAreKindSpecific(t *testing.T) {
	at := time.Unix(1_800_000_000, 0).UTC()
	valid := map[string]string{
		"totp": loopbackTOTP("JBSWY3DPEHPK3PXP", at), "sms_otp": "123456", "email_otp": "123456", "voice_otp": "123456",
		"push": "push", "push_number_match": "push_number_match", "passkey": "passkey", "security_key": "security_key",
	}
	for kind, value := range valid {
		if !validLoopbackChallenge(kind, value, at) {
			t.Fatalf("valid %s challenge was rejected", kind)
		}
		if validLoopbackChallenge(kind, "definitely-wrong", at) {
			t.Fatalf("wrong %s challenge was accepted", kind)
		}
	}
}

func TestLoopbackApprovalChallengeRequiresSessionBoundServerApproval(t *testing.T) {
	for _, test := range []struct{ id, kind string }{
		{"mfa-push", "push"},
		{"mfa-push-number-match", "push_number_match"},
		{"mfa-passkey", "passkey"},
		{"mfa-security-key", "security_key"},
	} {
		t.Run(test.kind, func(t *testing.T) {
			manifest := loopbackManifestForTest(t, test.id)
			fixture, err := NewLoopbackFixture(manifest)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.Close()
			fixture.SetRuntime(true)
			jar, _ := cookiejar.New(nil)
			client := &http.Client{Jar: jar}
			response, err := client.PostForm(fixture.InitialURL(), url.Values{"identifier": {"member@example.test"}, "password": {"scenario-password-value"}})
			if err != nil {
				t.Fatal(err)
			}
			_ = response.Body.Close()

			noRedirect := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
			response, err = noRedirect.PostForm(fixture.server.URL+fixture.path("challenge"), url.Values{"challenge": {test.kind}})
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusForbidden {
				t.Fatalf("direct approval bypass status = %d", response.StatusCode)
			}
			_ = response.Body.Close()
			if err := fixture.ApprovePendingChallenge(test.kind); err != nil {
				t.Fatal(err)
			}
			response, err = noRedirect.PostForm(fixture.server.URL+fixture.path("challenge"), url.Values{"challenge": {test.kind}})
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusSeeOther {
				t.Fatalf("server-approved challenge status = %d", response.StatusCode)
			}
			_ = response.Body.Close()
		})
	}
}

func TestLoopbackApprovalRequiresExactlyOneObservedSession(t *testing.T) {
	manifest := loopbackManifestForTest(t, "mfa-passkey")
	fixture, err := NewLoopbackFixture(manifest)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	fixture.SetRuntime(true)
	if err := fixture.ApprovePendingChallenge("passkey"); err == nil {
		t.Fatal("approval succeeded without an observed session")
	}
	for range 2 {
		jar, _ := cookiejar.New(nil)
		client := &http.Client{Jar: jar}
		response, postErr := client.PostForm(fixture.InitialURL(), url.Values{"identifier": {"member@example.test"}, "password": {"scenario-password-value"}})
		if postErr != nil {
			t.Fatal(postErr)
		}
		_ = response.Body.Close()
	}
	if err := fixture.ApprovePendingChallenge("passkey"); err == nil {
		t.Fatal("approval succeeded with multiple pending sessions")
	}
}

func TestLoopbackPasswordOnlyStrayDashboardPostIsRejected(t *testing.T) {
	manifest := loopbackManifestForTest(t, "password-main")
	fixture, err := NewLoopbackFixture(manifest)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	fixture.SetRuntime(true)
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.PostForm(fixture.InitialURL(), url.Values{"identifier": {"member@example.test"}, "password": {"scenario-password-value"}})
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	response, err = client.PostForm(fixture.DashboardURL(), url.Values{"challenge": {"push"}})
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("password-only stray POST status = %d", response.StatusCode)
	}
	_ = response.Body.Close()
}

func TestExactPromptWriterMatchesSplitPrompt(t *testing.T) {
	var destination bytes.Buffer
	matched := make(chan struct{})
	writer := &exactPromptWriter{destination: &destination, prompt: []byte("Approve? [y/N]: "), matched: matched}
	_, _ = writer.Write([]byte("Approve? "))
	_, _ = writer.Write([]byte("[y/N]: "))
	select {
	case <-matched:
	default:
		t.Fatal("split exact prompt was not observed")
	}
}

func TestRunBoundedAfterPromptDoesNotWaitForClosedChildStdin(t *testing.T) {
	if len(os.Args) > 0 && os.Args[len(os.Args)-1] == "prompt-exit-helper" {
		_, _ = fmt.Fprint(os.Stdout, "Browser authentication push challenge. Approve? [y/N]: ")
		os.Exit(0)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	approved := false
	started := time.Now()
	result := runBoundedAfterPrompt(context.Background(), 5*time.Second, t.TempDir(), []string{
		executable, "-test.run=^TestRunBoundedAfterPromptDoesNotWaitForClosedChildStdin$", "--", "prompt-exit-helper",
	}, nil, "Browser authentication push challenge. Approve? [y/N]: ", func() error {
		approved = true
		return nil
	})
	if result.err != nil {
		t.Fatal(result.err)
	}
	if !approved || time.Since(started) > 2*time.Second {
		t.Fatalf("prompt approval did not finish promptly: approved=%t duration=%s", approved, time.Since(started))
	}
}

func TestScenarioFailureClassificationRequiresClosedCode(t *testing.T) {
	root := t.TempDir()
	path := root + "/execution-report.json"
	write := func(code, summary string) {
		data, err := json.Marshal(map[string]any{
			"version": "udon.execution-report.v2", "status": "error", "started_at": "2026-08-20T00:00:00Z", "finished_at": "2026-08-20T00:00:01Z",
			"workflow_path": "workflow.json", "workflow_format": "uws-json", "workdir": root, "error_code": code, "error_summary": summary,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("unclassified", "unrelated context message")
	if got := scenarioFailureCode(path); got != "unclassified" {
		t.Fatalf("generic context text classified as %q", got)
	}
	write("invalid_context", "redacted browser failure")
	if got := scenarioFailureCode(path); got != "invalid_context" {
		t.Fatalf("closed failure code classified as %q", got)
	}
	write("invented", "invalid_context text must not be classified")
	if got := scenarioFailureCode(path); got != "unclassified" {
		t.Fatalf("unknown failure code classified as %q", got)
	}
}

func TestJourneyFixtureUsesBoundedNoStoreResponses(t *testing.T) {
	manifest := journeyManifestForTest(t)
	fixture, err := NewJourneyFixture(manifest)
	if err != nil {
		t.Fatal(err)
	}
	defer fixture.Close()
	response, err := http.Get(fixture.Origin() + "/ready")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Content-Security-Policy") == "" || fixture.server.Config.ReadHeaderTimeout != scenarioHTTPTimeout || fixture.server.Config.WriteTimeout != scenarioHTTPTimeout {
		t.Fatalf("journey fixture safety headers/timeouts are incomplete: %#v", response.Header)
	}
}

func loopbackManifestForTest(t *testing.T, id string) Manifest {
	t.Helper()
	values, err := LoadManifests(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if value.Suite == SuiteLoopback && value.ID == id {
			return value
		}
	}
	t.Fatalf("loopback manifest %q is unavailable", id)
	return Manifest{}
}

func journeyManifestForTest(t *testing.T) Manifest {
	t.Helper()
	values, err := LoadManifests(time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range values {
		if value.Suite == SuiteJourney {
			return value
		}
	}
	t.Fatal("journey manifest is unavailable")
	return Manifest{}
}

func bodyContains(t *testing.T, response *http.Response, wanted string) bool {
	t.Helper()
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Contains(string(data), wanted)
}
