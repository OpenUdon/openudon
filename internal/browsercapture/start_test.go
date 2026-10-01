package browsercapture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browserauthor"
)

func validAuthenticationStart() StartRequest {
	return StartRequest{Version: StartVersion, Mode: Authenticated, AbsoluteSeconds: 60, OperatorIdleSeconds: 30, Authentication: &AuthenticationStart{
		ProfileID: "member", URL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard", GoalURL: "https://members.example.test/status?view=summary", Goal: "Review status", Origins: []string{"https://members.example.test"}, GoalRole: "heading", GoalContext: "main", GoalLabel: "Status", AfterAuthentication: "continue_current_page",
	}}
}

func TestCaptureStartUsesExactOfflineClosedSchema(t *testing.T) {
	request := validAuthenticationStart()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeStart(data); err != nil {
		t.Fatal(err)
	}
	reg := StartRequest{Version: StartVersion, Mode: Registration, Registration: &RegistrationStart{ProfileID: "register", URL: "https://members.example.test/register", Origins: []string{"https://members.example.test"}, TransactionID: "register-test"}}
	regData, _ := json.Marshal(reg)
	if _, err := DecodeStart(regData); err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"null":               []byte("null"),
		"unknown provider":   []byte(strings.Replace(string(data), `"mode":"authenticated"`, `"mode":"authenticated","provider":"PRIVATE_CANARY"`, 1)),
		"credential":         []byte(strings.Replace(string(data), `"goal":"Review status"`, `"goal":"Review status","password":"PRIVATE_CANARY"`, 1)),
		"duplicates":         []byte(strings.Replace(string(data), `"absolute_seconds":60`, `"absolute_seconds":60,"absolute_seconds":7201`, 1)),
		"expiry ceiling":     []byte(strings.Replace(string(data), `"absolute_seconds":60`, `"absolute_seconds":7201`, 1)),
		"idle zero":          []byte(strings.Replace(string(data), `"operator_idle_seconds":30`, `"operator_idle_seconds":0`, 1)),
		"mixed mode":         []byte(strings.TrimSuffix(string(data), "}") + `,"registration":` + string(regData) + `}`),
		"extra document":     append(append([]byte(nil), data...), []byte(" {}")...),
		"missing fixed goal": []byte(strings.Replace(string(data), `"goal_url":"https://members.example.test/status?view=summary",`, "", 1)),
		"oversize":           []byte(strings.Repeat(" ", MaxMessageBytes+1)),
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeStart(payload); err == nil || strings.Contains(err.Error(), "PRIVATE_CANARY") {
				t.Fatal("invalid start accepted or exposed", err)
			}
		})
	}
}

func TestCaptureStartPreservesDistinctGoalDashboardAndNativePolicy(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	example := filepath.Join(t.TempDir(), "package")
	request := validAuthenticationStart()
	cfg, live, err := request.AuthenticationConfig(example, root, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DashboardURL != "https://members.example.test/dashboard" || cfg.GoalPredicate.Path != "/status" || cfg.GoalPredicate.Label != "Status" || live.GoalURL != request.Authentication.GoalURL || live.AfterAuthentication != "continue_current_page" || !live.NoLLM || cfg.Absolute != time.Minute || cfg.OperatorIdle != 30*time.Second {
		t.Fatalf("fixed authority changed: %#v", cfg)
	}
	request.Authentication.BlockedScriptOrigin = "https://members.example.test"
	if _, _, err := request.AuthenticationConfig(example, root, ""); err == nil {
		t.Fatal("blocked origin was admitted")
	}
	request.Authentication.BlockedScriptOrigin = ""
	request.Authentication.GoalURL += "&token=PRIVATE_CANARY"
	if _, _, err := request.AuthenticationConfig(example, root, ""); err == nil || strings.Contains(err.Error(), "PRIVATE_CANARY") {
		t.Fatal("private goal URL accepted or exposed")
	}
}

func TestCaptureRegistrationStartKeepsFiniteNativeDefaults(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	request := StartRequest{Version: StartVersion, Mode: Registration, Registration: &RegistrationStart{ProfileID: "register", URL: "https://members.example.test/register", Origins: []string{"https://members.example.test"}, TransactionID: "test-registration"}}
	cfg, start, err := request.RegistrationConfig(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Absolute != browserauthor.DefaultAbsolute || cfg.OperatorIdle != browserauthor.DefaultOperatorIdle || start.URL != request.Registration.URL || start.Confirmed {
		t.Fatal("registration start authority changed")
	}
}

func TestReviewedRegistrationStartSelectsExistingNativeProtocols(t *testing.T) {
	root := t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	for _, protocol := range []string{registrationauthorsession.ProtocolV1, registrationauthorsession.ProtocolV2, registrationauthorsession.ProtocolV3, registrationauthorsession.ProtocolV4} {
		request := StartRequest{Version: StartVersion, Mode: Registration, Registration: &RegistrationStart{Protocol: protocol, ProfileID: "register", URL: "https://members.example.test/register", Origins: []string{"https://members.example.test"}, TransactionID: "register-test"}}
		data, _ := json.Marshal(request)
		decoded, err := DecodeStart(data)
		if err != nil {
			t.Fatal(err)
		}
		config, _, err := decoded.RegistrationConfig(root, "")
		if err != nil || config.Protocol != protocol {
			t.Fatal("reviewed native protocol changed", err)
		}
		request.Registration.Protocol = "browsertools.registration-author-session.v999"
		data, _ = json.Marshal(request)
		if _, err := DecodeStart(data); err == nil {
			t.Fatal("unknown native protocol accepted")
		}
	}
}
