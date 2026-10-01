package browserauthor

import (
	"context"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/uws/browserregistration"
)

func TestRegistrationDecisionReusesCurrentNativeVerificationAndPreviewGates(t *testing.T) {
	const origin = "https://app.example.test"
	const id = "candidate-0123456789abcdef"
	verification := &browserregistration.HumanVerification{Provider: "hcaptcha", Activation: "approved_submit", WidgetBinding: "single_in_submit_form", SubmissionURL: origin + "/register", Dependencies: browserregistration.VerificationDependencies{Policy: "hcaptcha.v1", MaxRequests: 256, MaxResponseBytes: 32 << 20, TimeoutMS: 120000}}
	checked := false
	observation := registrationauthorsession.Observation{Generation: 1, Origin: origin, Path: "/register", Candidates: []registrationauthorsession.Candidate{
		{ID: id, Role: "button", Label: "Register", Matches: 1, Verification: &registrationauthorsession.VerificationObservation{Provider: verification.Provider, Activation: verification.Activation, SubmissionURL: verification.SubmissionURL, WidgetBinding: verification.WidgetBinding, Coverage: "standard_single_widget"}},
		{ID: "candidate-1123456789abcdef", Role: "checkbox", Label: "Member", Matches: 1, Control: &registrationauthorsession.ControlMetadata{Kind: "checkbox"}},
	}}
	state := registrationRunState{started: true, protocol: registrationauthorsession.ProtocolV4, phase: "observing", origins: []string{origin}, bounds: expectedRegistrationBounds(nil), generation: 1, observations: 1, observation: &observation, history: []registrationauthorsession.Observation{observation}}
	session := &RegistrationSession{}
	session.rememberDecisionState(state, RegistrationEvent{State: "observation", Observation: &observation})
	preview := &registrationauthorsession.PreviewRequest{CandidateID: observation.Candidates[1].ID, Generation: 1, Purpose: "public_form_preview", Action: "check", Checked: &checked}
	tests := []struct {
		name    string
		command RegistrationCommand
		valid   bool
	}{
		{"verification", RegistrationCommand{Type: "approve_verification", Confirmed: true, Verification: verification, VerificationCandidateID: id}, true},
		{"unconfirmed", RegistrationCommand{Type: "approve_verification", Verification: verification, VerificationCandidateID: id}, false},
		{"stale candidate", RegistrationCommand{Type: "approve_verification", Confirmed: true, Verification: verification, VerificationCandidateID: "candidate-ffffffffffffffff"}, false},
		{"mixed verification payload", RegistrationCommand{Type: "approve_verification", Confirmed: true, Verification: verification, VerificationCandidateID: id, URL: origin}, false},
		{"public preview", RegistrationCommand{Type: "preview", Confirmed: true, Preview: preview}, true},
		{"unconfirmed preview", RegistrationCommand{Type: "preview", Preview: preview}, false},
		{"navigation", RegistrationCommand{Type: "navigate", Method: "GET", URL: origin + "/register?view=member"}, true},
		{"new origin refused", RegistrationCommand{Type: "navigate", Method: "GET", URL: "https://other.example.test/register"}, false},
		{"submit refused", RegistrationCommand{Type: "navigate", Method: "POST", URL: origin + "/register"}, false},
		{"private query refused", RegistrationCommand{Type: "navigate", Method: "GET", URL: origin + "/register?token=PRIVATE_CANARY"}, false},
		{"observe", RegistrationCommand{Type: "observe"}, true},
		{"unreviewed finish", RegistrationCommand{Type: "finish", Confirmed: true}, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := session.ValidateDecision("observation", 1, test.command); (err == nil) != test.valid {
				t.Fatalf("offered=%v, error=%v", test.valid, err)
			}
		})
	}
	if state.verification != nil {
		t.Fatal("pure validation granted verification authority")
	}
	if err := session.ValidateDecision("observation", 2, RegistrationCommand{Type: "observe"}); err == nil {
		t.Fatal("stale observation generation accepted")
	}
	session.publishTerminal(RegistrationEvent{State: "failed"})
	if err := session.ValidateDecision("observation", 1, RegistrationCommand{Type: "observe"}); err == nil {
		t.Fatal("terminal state retained decision authority")
	}
}

func TestRegistrationEventClosureWaitsForPrivateCleanup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test worker uses POSIX shell")
	}
	root := registrationControllerRoot(t)
	inbox, err := browsercandidate.OpenPrivateInbox(root)
	if err != nil {
		t.Fatal(err)
	}
	worker := writeRegistrationWorker(t, `#!/bin/sh
printf '%s\n' '{"protocol":"browsertools.registration-author-session.v1","type":"hello","capabilities":["get_head_only","no_submit","reduced_observation","registration_review"]}'
while IFS= read -r ignored; do :; done
`)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	session, err := startRegistrationProcess(ctx, RegistrationConfig{PrivateRoot: root, TransactionID: "cleanup", Protocol: registrationauthorsession.ProtocolV1, OperatorIdle: time.Second, Absolute: 3 * time.Second}, inbox, []string{worker}, func() { close(entered); <-release })
	if err != nil {
		_ = inbox.Close()
		t.Fatal(err)
	}
	defer session.Cancel()
	select {
	case event := <-session.Events():
		if event.State != "ready" {
			t.Fatal("worker not ready")
		}
	case <-ctx.Done():
		t.Fatal("worker readiness timed out")
	}
	closed := make(chan struct{})
	go func() {
		for range session.Events() {
		}
		close(closed)
	}()
	session.Cancel()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("worker/reader failed to join before cleanup")
	}
	select {
	case <-closed:
		t.Fatal("event channel closed before private cleanup")
	default:
	}
	unblock()
	select {
	case <-closed:
	case <-ctx.Done():
		t.Fatal("event channel failed to close after joined cleanup")
	}
}

func TestNormalizeRegistrationStartFixesOnlyReviewedFiniteAuthority(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	cfg, err := NormalizeRegistrationConfig(RegistrationConfig{PrivateRoot: root, TransactionID: "test", Protocol: registrationauthorsession.ProtocolV4})
	if err != nil || cfg.Absolute != DefaultAbsolute {
		t.Fatal("config normalization", err)
	}
	start := RegistrationCommand{Type: "start", ProfileID: "register", URL: "https://app.example.test/register?view=member", Origins: []string{"https://app.example.test"}}
	if _, err := NormalizeRegistrationStart(cfg.Protocol, start); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func(*RegistrationCommand)
	}{
		{"unapproved origin", func(v *RegistrationCommand) { v.Origins = []string{"https://other.example.test"} }},
		{"credential query", func(v *RegistrationCommand) { v.URL += "&token=PRIVATE_CANARY" }},
		{"mixed payload", func(v *RegistrationCommand) { v.VerificationCandidateID = "candidate-0123456789abcdef" }},
		{"invalid profile", func(v *RegistrationCommand) { v.ProfileID = "../register" }},
		{"duplicate origin", func(v *RegistrationCommand) { v.Origins = append(v.Origins, v.Origins[0]) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copy := cloneRegistrationCommand(start)
			test.mutate(&copy)
			if _, err := NormalizeRegistrationStart(cfg.Protocol, copy); err == nil {
				t.Fatal("invalid authority accepted")
			}
		})
	}
}
