package browserauthor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
)

func TestNormalizeConfigFixesFiniteAuthority(t *testing.T) {
	privateRoot := t.TempDir()
	if err := os.Chmod(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	config, err := normalizeConfig(Config{
		PrivateRoot: privateRoot,
		InitialURL:  "https://MEMBERS.example.test/login", DashboardURL: "https://members.example.test/dashboard",
		Goal: "Review account status", Origins: []string{"https://members.example.test"}, ProfileID: "member",
		GoalPredicate: authorresult.GoalPredicate{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.OperatorIdle != DefaultOperatorIdle || config.Absolute != DefaultAbsolute || config.InitialURL != "https://members.example.test/login" {
		t.Fatalf("normalized config = %#v", config)
	}
	for _, test := range []struct {
		name   string
		mutate func(*Config)
	}{
		{name: "initial query", mutate: func(value *Config) { value.InitialURL = "https://members.example.test/login?next=dashboard" }},
		{name: "dashboard query", mutate: func(value *Config) { value.DashboardURL = "https://members.example.test/dashboard?tab=home" }},
		{name: "encoded path traversal", mutate: func(value *Config) { value.InitialURL = "https://members.example.test/%2e%2e/private" }},
		{name: "goal path traversal", mutate: func(value *Config) { value.GoalPredicate.Path = "/../private" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := config
			test.mutate(&invalid)
			if _, err := normalizeConfig(invalid); err == nil {
				t.Fatal("unsupported capture URL was accepted")
			}
		})
	}
	config.Origins = []string{"https://other.example.test"}
	if _, err := normalizeConfig(config); err == nil {
		t.Fatal("origin mismatch was accepted")
	}
	config.Origins = []string{"https://members.example.test"}
	config.ProfileID = "../member"
	if _, err := normalizeConfig(config); err == nil {
		t.Fatal("unsafe profile ID was accepted")
	}
}

func TestTypedResponsesCannotInventBrowserAuthority(t *testing.T) {
	observation := authorsession.Observation{
		Origin: "https://members.example.test", Path: "/login", Context: "main", Contexts: map[string]authorresult.Context{},
		Candidates: []authorsession.Candidate{{ID: "candidate-0123456789abcdef", Role: "button", Label: "Sign in", Matches: 1}},
	}
	config := Config{DashboardURL: "https://members.example.test/dashboard", Origins: []string{"https://members.example.test"}}
	if _, err := observationResponse(Response{Kind: "click", CandidateID: "candidate-ffffffffffffffff", POSTBudget: 1}, observation, config); err == nil {
		t.Fatal("invented candidate was accepted")
	}
	if _, err := observationResponse(Response{Kind: "navigate_get", URL: "https://other.example.test/"}, observation, config); err == nil {
		t.Fatal("unapproved origin was accepted")
	}
	message, err := observationResponse(Response{Kind: "click", CandidateID: observation.Candidates[0].ID, POSTBudget: 1}, observation, config)
	if err != nil || message.Type != "execute" || message.Action != "click" {
		t.Fatalf("typed click = %#v, %v", message, err)
	}
	checkpoint := authorsession.Checkpoint{Kind: "mfa", CandidateID: "candidate-0123456789abcdef", ChallengeKinds: []string{"totp", "security_key"}}
	if _, err := checkpointResponse(Response{Kind: "continue", CandidateID: checkpoint.CandidateID, ChallengeKind: "sms_otp"}, checkpoint); err == nil {
		t.Fatal("unreported MFA kind was accepted")
	}
}

func TestStabilizeExecutableCreatesPrivateCopy(t *testing.T) {
	privateRoot := t.TempDir()
	if err := os.Chmod(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "icot")
	if err := os.WriteFile(source, []byte("worker"), 0o700); err != nil {
		t.Fatal(err)
	}
	target, cleanup, err := stabilizeExecutable(source, privateRoot)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	info, err := os.Lstat(target)
	if err != nil || info.Mode().Perm() != 0o500 || !info.Mode().IsRegular() {
		t.Fatalf("stable executable = %v, %v", info, err)
	}
}

func TestMinimalEnvironmentExcludesCredentialAndModelValues(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sentinel-model-secret")
	t.Setenv("MEMBER_PASSWORD", "sentinel-browser-secret")
	t.Setenv("PATH", "/usr/bin")
	environment := strings.Join(minimalEnvironment(), "\n")
	if strings.Contains(environment, "sentinel-model-secret") || strings.Contains(environment, "sentinel-browser-secret") || !strings.Contains(environment, "PATH=/usr/bin") {
		t.Fatalf("minimal environment = %q", environment)
	}
}

func TestOperatorIdleCancellationIsBounded(t *testing.T) {
	session := &Session{cancel: func() {}, responses: make(chan Response), events: make(chan Event, 2), done: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, ok := session.awaitResponse(ctx, time.Millisecond, Event{State: "human_input"}); ok {
		t.Fatal("idle checkpoint unexpectedly received a response")
	}
	first := <-session.events
	second := <-session.events
	if first.State != "human_input" || second.State != "canceled" || second.ErrorCode != "operator_idle_timeout" {
		t.Fatalf("events = %#v, %#v", first, second)
	}
}
