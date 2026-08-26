package browserauthor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorresult"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/icot/engine"
)

const registrationControllerFixture = `profile: uws.browser-registration.1.0
info:
  title: Synthetic dedicated test registration
  applicationOrigins: [https://app.example.test]
  registrationOrigins: [https://app.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-25T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-25T00:00:00Z"}
credentialSlots:
  identifier: {kind: identifier}
  password: {kind: password}
flows:
  create_dedicated_test_user:
    sequence:
      - navigate: https://app.example.test/register
      - type_credential: {locator: {role: textbox}, slot: identifier}
      - type_credential: {locator: {role: textbox, name: Password}, slot: password}
      - submit: {locator: {role: button, name: Register}}
      - human_checkpoint: {kind: email_verification}
      - wait_for: {locator: {role: status}}
    effects: [creates_account, sends_verification, requires_human_verification]
    confirmationPolicy: {required: true}
    success: {origin: https://app.example.test, path: /registration-complete, locator: {role: status}}
`

func TestExternalRegistrationWorkerAdoptsOnlyAfterCleanExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test worker uses a POSIX script")
	}
	root := registrationControllerRoot(t)
	profileBytes, resultBytes, resultDigest := registrationControllerResult(t)
	fixture := filepath.Join(t.TempDir(), "producer-result.json")
	if err := os.WriteFile(fixture, resultBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	resultName := "registration-authoring-" + strings.TrimPrefix(resultDigest, "sha256:")[:16] + ".json"
	worker := writeRegistrationWorker(t, fmt.Sprintf(`#!/bin/sh
if [ "${OPENAI_API_KEY+x}" = x ]; then exit 91; fi
if [ "${CHROME_DEVEL_SANDBOX-}" != "/validated/chrome_sandbox" ]; then exit 92; fi
printf '%%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"hello","capabilities":["get_head_only","no_submit","reduced_observation","registration_review"]}'
IFS= read -r start
printf '%%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"state","phase":"observing","bounds":{"navigationTimeoutMs":20000,"totalTimeoutMs":300000,"maxRequests":256,"maxResponseBytes":33554432,"maxObservations":64,"maxCandidates":128}}'
IFS= read -r observe
printf '%%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"observation","observation":{"generation":1,"origin":"https://app.example.test","path":"/register","candidates":[{"id":"candidate-0123456789abcdef","role":"button","label":"Register","matches":1}],"diagnostics":[]}}'
IFS= read -r review
printf '%%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"state","phase":"reviewed"}'
IFS= read -r finish
printf '%%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"state","phase":"closed"}'
cp '%s' "$4/%s"
chmod 600 "$4/%s"
`, fixture, resultName, resultName))
	t.Setenv("OPENAI_API_KEY", "sentinel-model-secret")
	t.Setenv("CHROME_DEVEL_SANDBOX", "/validated/chrome_sandbox")
	originalClock := registrationAssessmentClock
	defer func() { registrationAssessmentClock = originalClock }()
	registrationAssessmentClock = func() time.Time { return time.Date(2026, 8, 25, 1, 0, 1, 0, time.UTC) }
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := StartExternalRegistration(ctx, registrationControllerConfig(root), worker)
	if err != nil {
		t.Fatal(err)
	}
	wantEvent(t, session, "ready")
	sendRegistrationCommand(t, ctx, session, RegistrationCommand{
		Type: "start", ProfileID: "synthetic_registration", URL: "https://app.example.test/register",
		Origins: []string{"https://app.example.test"},
	})
	wantEvent(t, session, "observing")
	sendRegistrationCommand(t, ctx, session, RegistrationCommand{Type: "observe"})
	observation := wantEvent(t, session, "observation")
	if observation.Observation == nil || len(observation.Observation.Candidates) != 1 {
		t.Fatalf("observation = %#v", observation)
	}
	sendRegistrationCommand(t, ctx, session, RegistrationCommand{
		Type: "review", Confirmed: true, Profile: profileBytes,
		CandidateIDs: []string{observation.Observation.Candidates[0].ID},
		Flow:         "create_dedicated_test_user", CleanupDisposition: "delete_separately",
		CredentialBindings: registrationControllerBindings(),
	})
	wantEvent(t, session, "reviewed")
	sendRegistrationCommand(t, ctx, session, RegistrationCommand{Type: "finish", Confirmed: true})
	wantEvent(t, session, "closed")
	adopted := wantEvent(t, session, "candidate")
	if adopted.Candidate == nil || adopted.Candidate.Transaction().Provenance.ResultSHA256 != resultDigest || adopted.Candidate.Transaction().Session != "" {
		t.Fatalf("adopted event = %#v", adopted)
	}
	virtualInput, err := engine.RegistrationVirtualBrowserTransaction(adopted.Candidate, true)
	if err != nil {
		t.Fatal(err)
	}
	if virtualInput.Transaction.State != browsertransaction.StateReviewed || virtualInput.Transaction.Session != "" || virtualInput.Sources[0].CleanupDisposition != "delete_separately" {
		t.Fatalf("reviewed registration adapter = %#v", virtualInput.Transaction)
	}
	discovery, err := elicitor.DiscoverVirtualBrowserSources([]elicitor.VirtualBrowserTransactionInput{virtualInput}, time.Date(2026, 8, 25, 1, 0, 1, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Candidates) != 1 || discovery.Candidates[0].ProvidesSession != "" || discovery.Candidates[0].RequiresSession != "" || len(discovery.Docs) != 1 || len(discovery.Docs[0].Operations) != 1 {
		t.Fatalf("actual producer registration discovery = %#v", discovery.Candidates)
	}
	if _, ok := <-session.Events(); ok {
		t.Fatal("registration event stream did not close")
	}
}

func TestExternalRegistrationWorkerExitAndProtocolFailuresAreClosed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test worker uses a POSIX script")
	}
	tests := []struct {
		name   string
		script string
		want   string
	}{
		{
			name: "worker exit",
			script: `#!/bin/sh
printf '%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"hello","capabilities":["get_head_only","no_submit","reduced_observation","registration_review"]}'
IFS= read -r start
exit 7
`,
			want: "worker_exit",
		},
		{
			name: "cross variant field",
			script: `#!/bin/sh
printf '%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"hello","capabilities":["get_head_only","no_submit","reduced_observation","registration_review"],"phase":"observing"}'
`,
			want: "protocol_negotiation",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := registrationControllerRoot(t)
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			session, err := StartExternalRegistration(ctx, registrationControllerConfig(root), writeRegistrationWorker(t, test.script))
			if err != nil {
				t.Fatal(err)
			}
			first := <-session.Events()
			if first.State == "ready" {
				sendRegistrationCommand(t, ctx, session, RegistrationCommand{
					Type: "start", ProfileID: "synthetic_registration", URL: "https://app.example.test/register",
					Origins: []string{"https://app.example.test"},
				})
				first = <-session.Events()
			}
			if first.State != "failed" || first.ErrorCode != test.want || strings.Contains(fmt.Sprintf("%#v", first), root) {
				t.Fatalf("failure = %#v", first)
			}
		})
	}
}

func TestExternalRegistrationCancellationCannotAdoptCandidate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test worker uses a POSIX script")
	}
	root := registrationControllerRoot(t)
	worker := writeRegistrationWorker(t, `#!/bin/sh
printf '%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"hello","capabilities":["get_head_only","no_submit","reduced_observation","registration_review"]}'
IFS= read -r start
while :; do sleep 1; done
`)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, err := StartExternalRegistration(ctx, registrationControllerConfig(root), worker)
	if err != nil {
		t.Fatal(err)
	}
	wantEvent(t, session, "ready")
	session.Cancel()
	wantEvent(t, session, "canceled")
	for event := range session.Events() {
		if event.Candidate != nil {
			t.Fatalf("canceled registration produced a candidate: %#v", event)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "registration-authoring-") {
			t.Fatalf("canceled registration retained a result: %s", entry.Name())
		}
	}
}

func TestRegistrationControllerRequiresExactHumanReview(t *testing.T) {
	bounds := expectedRegistrationBounds(nil)
	state := registrationRunState{
		started: true, profileID: "synthetic_registration", origins: []string{"https://app.example.test"},
		generation: 1, observations: 1, minimumRequests: 1, bounds: bounds,
		observation: &registrationauthorsession.Observation{
			Generation: 1, Origin: "https://app.example.test", Path: "/register",
			Candidates:  []registrationauthorsession.Candidate{{ID: "candidate-0123456789abcdef", Role: "button", Label: "Register", Matches: 1}},
			Diagnostics: []string{},
		},
	}
	profileBytes, _, _ := registrationControllerResult(t)
	for _, command := range []RegistrationCommand{
		{Type: "review", Confirmed: false, Profile: profileBytes, CandidateIDs: []string{"candidate-0123456789abcdef"}, Flow: "create_dedicated_test_user", CleanupDisposition: "delete_separately", CredentialBindings: registrationControllerBindings()},
		{Type: "review", Confirmed: true, Profile: append(profileBytes, '\n'), CandidateIDs: []string{"candidate-0123456789abcdef"}, Flow: "create_dedicated_test_user", CleanupDisposition: "delete_separately", CredentialBindings: registrationControllerBindings()},
		{Type: "observe", URL: "https://app.example.test/register"},
		{Type: "finish", Confirmed: true, Flow: "invented"},
		{Type: "review", Confirmed: true, Profile: profileBytes, CandidateIDs: []string{"candidate-0123456789abcdef"}, Flow: "create_dedicated_test_user", CleanupDisposition: "delete_separately", CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "identifier", Binding: "registration_identifier"}}},
	} {
		if _, _, err := prepareRegistrationCommand(command, state); err == nil {
			t.Fatalf("unsafe command was accepted: %#v", command)
		}
	}
	message, _, err := prepareRegistrationCommand(RegistrationCommand{
		Type: "review", Confirmed: true, Profile: profileBytes,
		CandidateIDs: []string{"candidate-0123456789abcdef"}, Flow: "create_dedicated_test_user",
		CleanupDisposition: "delete_separately", CredentialBindings: registrationControllerBindings(),
	}, state)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(message)
	if err != nil || strings.Contains(string(encoded), "registration_identifier") || strings.Contains(string(encoded), "credentialBindings") {
		t.Fatalf("OpenUdon-only symbolic bindings crossed the worker wire: %s, %v", encoded, err)
	}
}

func TestRegistrationControllerRejectsInvalidParentBounds(t *testing.T) {
	invalid := expectedRegistrationBounds(nil)
	invalid.MaxRequests = 0
	if _, _, err := prepareRegistrationCommand(RegistrationCommand{
		Type: "start", ProfileID: "synthetic_registration", URL: "https://app.example.test/register",
		Origins: []string{"https://app.example.test"}, Bounds: &invalid,
	}, registrationRunState{phase: "awaiting_start"}); err == nil {
		t.Fatal("invalid parent bounds were accepted")
	}
}

func TestRegistrationServerDecoderRejectsAmbiguousJSON(t *testing.T) {
	for _, data := range [][]byte{
		[]byte(`{"protocol":"browsertools.registration-author-session.v1","type":"state","type":"state","phase":"observing"}`),
		[]byte(`{"protocol":"browsertools.registration-author-session.v1","type":"state","phase":"observing","privatePath":"/private/result"}`),
		[]byte(`{"protocol":"browsertools.registration-author-session.v1","type":"state","phase":"observing"} {}`),
		{0xff},
	} {
		if _, err := decodeRegistrationServerMessage(data); err == nil {
			t.Fatalf("ambiguous protocol JSON was accepted: %q", data)
		}
	}
}

func registrationControllerConfig(root string) RegistrationConfig {
	return RegistrationConfig{
		PrivateRoot: root, TransactionID: "registration-transaction",
		OperatorIdle: time.Second, Absolute: 5 * time.Second,
	}
}

func registrationControllerBindings() []browsertransaction.CredentialBinding {
	return []browsertransaction.CredentialBinding{
		{Slot: "identifier", Binding: "registration_identifier"},
		{Slot: "password", Binding: "registration_password"},
	}
}

func registrationControllerResult(t *testing.T) ([]byte, []byte, string) {
	t.Helper()
	profile, err := registrationprofile.Parse([]byte(registrationControllerFixture))
	if err != nil {
		t.Fatal(err)
	}
	profileBytes, err := registrationprofile.MarshalJSON(profile)
	if err != nil {
		t.Fatal(err)
	}
	observedAt := time.Date(2026, 8, 25, 0, 30, 0, 0, time.UTC)
	result, err := registrationauthorresult.Build(registrationauthorresult.BuildRequest{
		CreatedAt: observedAt.Add(30 * time.Minute),
		Completion: &registrationauthorsession.Completion{
			Protocol: registrationauthorsession.Protocol, ProfileID: "synthetic_registration",
			Profile: *profile, ProfileBytes: profileBytes,
			ReviewedCandidates: []registrationauthorsession.ReviewedCandidate{{
				ID: "candidate-0123456789abcdef", Generation: 1, Role: "button", Label: "Register", Matches: 1,
			}},
			Flow: "create_dedicated_test_user", CleanupDisposition: "delete_separately",
			Origins: []string{"https://app.example.test"}, ObservedAt: observedAt,
			Bounds: registrationauthorsession.Bounds{
				NavigationTimeoutMS: 20_000, TotalTimeoutMS: 300_000, MaxRequests: 256,
				MaxResponseBytes: 32 << 20, MaxObservations: 64, MaxCandidates: 128,
			},
			Observations: 1, Network: registrationauthorsession.NetworkSummary{Requests: 1, GETRequests: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	resultBytes, err := registrationauthorresult.MarshalDeterministic(result)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := registrationauthorresult.Digest(result)
	if err != nil {
		t.Fatal(err)
	}
	return profileBytes, resultBytes, digest
}

func writeRegistrationWorker(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "browsertools-worker")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func registrationControllerRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Chmod(root, 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func sendRegistrationCommand(t *testing.T, ctx context.Context, session *RegistrationSession, command RegistrationCommand) {
	t.Helper()
	if err := session.Send(ctx, command); err != nil {
		t.Fatal(err)
	}
}

func wantEvent(t *testing.T, session *RegistrationSession, state string) RegistrationEvent {
	t.Helper()
	event, ok := <-session.Events()
	if !ok || event.State != state {
		t.Fatalf("event = %#v, open=%v, want state %q", event, ok, state)
	}
	return event
}
