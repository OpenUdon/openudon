package browserscenario

import (
	"encoding/json"
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

func TestScenarioFailureClassificationRequiresClosedCode(t *testing.T) {
	root := t.TempDir()
	path := root + "/execution-report.json"
	write := func(summary string) {
		data, err := json.Marshal(map[string]any{
			"version": "udon.execution-report.v1", "status": "error", "started_at": "2026-08-20T00:00:00Z", "finished_at": "2026-08-20T00:00:01Z",
			"workflow_path": "workflow.json", "workflow_format": "uws-json", "workdir": root, "error_summary": summary,
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("unrelated context message")
	if got := scenarioFailureCode(path); got != "invalid_response" {
		t.Fatalf("generic context text classified as %q", got)
	}
	write("browser driver returned invalid_context")
	if got := scenarioFailureCode(path); got != "invalid_context" {
		t.Fatalf("closed failure code classified as %q", got)
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
