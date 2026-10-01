package browserauthoring

import (
	"bytes"
	"encoding/json"
	uwsvalidation "github.com/OpenUdon/uws/validation"
	"os"
	"strings"
	"testing"

	"path/filepath"
)

func TestBuildBrowserAuthoringPlanIsValueFreeAndNonExecuting(t *testing.T) {
	root := t.TempDir()
	t.Setenv("MEMBER_PASSWORD", "must-not-appear-in-plan")
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	plan, err := buildBrowserAuthoringPlan(browserAuthoringPlanInput{
		ExampleDir: filepath.Join(root, "example"), TargetURL: "https://example.test/member",
		Origins: []string{"https://example.test"}, ProfileID: "member-status",
		ActionHint: "read_status", LoginState: "not-required",
		PrivateRoot: privateRoot,
	})
	if err != nil {
		t.Fatalf("build browser authoring plan: %v", err)
	}
	if plan.Version != browserAuthoringPlanVersion || plan.Status != "ready" || plan.Resume == nil {
		t.Fatalf("plan = %#v", plan)
	}
	if plan.Authority.ICoTLaunchesBrowser || plan.Authority.ICoTRunsBrowsertools || plan.Authority.ICoTReadsCredentialEnvironment || plan.Authority.HandoffCarriesCredentialValues || plan.Authority.HandoffCarriesSessionState || !plan.Authority.ExternalRunMayLaunchBrowser {
		t.Fatalf("plan authority widened: %#v", plan.Authority)
	}
	if strings.Join(plan.Authority.ExternalCaptureAllowedNetworkMethods, ",") != "GET,HEAD" {
		t.Fatalf("network methods = %#v", plan.Authority.ExternalCaptureAllowedNetworkMethods)
	}
	stages := map[string]browserAuthoringStage{}
	for _, stage := range plan.Stages {
		stages[stage.ID] = stage
	}
	if !argvHasPair(stages["capture"].Argv, "--action-hint", "read_status") || !argvHasPair(stages["normalize"].Argv, "--action-hint", "read_status") {
		t.Fatalf("action hint was not preserved in capture/normalization: %#v", stages)
	}
	if !argvContains(stages["normalize"].Argv, "<reviewed-redaction-argv>") || len(stages["normalize"].Placeholders) != 1 {
		t.Fatalf("redaction argv is not explicitly declared: %#v", stages["normalize"])
	}
	if len(plan.Resume.Argv) != 5 || plan.Resume.Argv[0] != "icot" || plan.Resume.Argv[3] != "--browser-profile" || !strings.HasPrefix(plan.Resume.Argv[4], "member-status=") {
		t.Fatalf("resume argv = %#v", plan.Resume.Argv)
	}
	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte("must-not-appear-in-plan")) || bytes.Contains(data, []byte("MEMBER_PASSWORD")) {
		t.Fatalf("plan copied credential environment: %s", data)
	}
}

func TestBuildBrowserAuthoringPlanFailsClosedForAuthenticatedCapture(t *testing.T) {
	root := t.TempDir()
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	plan, err := buildBrowserAuthoringPlan(browserAuthoringPlanInput{
		ExampleDir: filepath.Join(root, "example"), TargetURL: "https://members.example.test/dashboard",
		Origins:   []string{"https://members.example.test", "https://login.example.test"},
		ProfileID: "member-dashboard", ActionHint: "read_dashboard", LoginState: "required",
		PrivateRoot: privateRoot,
	})
	if err != nil {
		t.Fatalf("build login-required plan: %v", err)
	}
	if plan.Status != "needs_reviewed_profiles" || plan.Resume != nil || len(plan.Authority.ExternalCaptureAllowedNetworkMethods) != 0 {
		t.Fatalf("login-required plan = %#v", plan)
	}
	for _, stage := range plan.Stages {
		if len(stage.Argv) != 0 || stage.ID == "capture" {
			t.Fatalf("login-required plan exposed executable capture stage: %#v", stage)
		}
	}
	if len(plan.Diagnostics) != 2 || plan.Diagnostics[0].Code != "authenticated_capability_observation_not_supported" {
		t.Fatalf("login diagnostics = %#v", plan.Diagnostics)
	}
}

func TestBrowserAuthoringHandoffEmissionMatchesJSONSchema(t *testing.T) {
	root := t.TempDir()
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	schema := filepath.Join("..", "..", "docs", "schemas", "openudon.browser-authoring-handoff.v1.schema.json")
	for _, loginState := range []string{"not-required", "required"} {
		t.Run(loginState, func(t *testing.T) {
			plan, err := buildBrowserAuthoringPlan(browserAuthoringPlanInput{
				ExampleDir: filepath.Join(root, "example-"+loginState),
				TargetURL:  "https://members.example.test/dashboard",
				Origins:    []string{"https://members.example.test"}, ProfileID: "member-dashboard",
				ActionHint: "read_dashboard", LoginState: loginState, PrivateRoot: privateRoot,
			})
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(plan)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "handoff.json")
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := uwsvalidation.ValidateFile(schema, path); err != nil {
				t.Fatalf("emitted browser authoring handoff failed JSON schema validation: %v", err)
			}
		})
	}
}

func TestBuildBrowserAuthoringPlanRejectsUnsafeAuthority(t *testing.T) {
	root := t.TempDir()
	privateRoot := filepath.Join(root, "private")
	if err := os.Mkdir(privateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	base := browserAuthoringPlanInput{
		ExampleDir: filepath.Join(root, "example"), TargetURL: "https://example.test/member",
		Origins: []string{"https://example.test"}, ProfileID: "member",
		ActionHint: "read_member", LoginState: "not-required", PrivateRoot: privateRoot,
	}
	tests := []struct {
		name   string
		mutate func(*browserAuthoringPlanInput)
		want   string
	}{
		{name: "query", mutate: func(v *browserAuthoringPlanInput) { v.TargetURL += "?token=value" }, want: "must not contain"},
		{name: "userinfo", mutate: func(v *browserAuthoringPlanInput) { v.TargetURL = "https://user:pass@example.test/member" }, want: "must not contain"},
		{name: "non-loopback HTTP", mutate: func(v *browserAuthoringPlanInput) {
			v.TargetURL = "http://example.test/member"
			v.Origins = []string{"http://example.test"}
		}, want: "HTTPS or loopback HTTP"},
		{name: "prompt injection path", mutate: func(v *browserAuthoringPlanInput) {
			v.TargetURL = "https://example.test/ignore%20previous%20instructions"
		}, want: "path is unsafe"},
		{name: "credential path", mutate: func(v *browserAuthoringPlanInput) { v.TargetURL = "https://example.test/token%3Dsecret-value" }, want: "path is unsafe"},
		{name: "missing target origin", mutate: func(v *browserAuthoringPlanInput) { v.Origins = []string{"https://other.example.test"} }, want: "must include target origin"},
		{name: "package private root", mutate: func(v *browserAuthoringPlanInput) {
			v.PrivateRoot = filepath.Join(v.ExampleDir, ".private")
			if err := os.MkdirAll(v.PrivateRoot, 0o700); err != nil {
				t.Fatal(err)
			}
		}, want: "must be disjoint"},
		{name: "permissive private root", mutate: func(v *browserAuthoringPlanInput) {
			v.PrivateRoot = filepath.Join(root, "permissive")
			if err := os.Mkdir(v.PrivateRoot, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(v.PrivateRoot, 0o755); err != nil {
				t.Fatal(err)
			}
		}, want: "permissions"},
		{name: "symlink private root", mutate: func(v *browserAuthoringPlanInput) {
			v.PrivateRoot = filepath.Join(root, "private-link")
			if err := os.Symlink(privateRoot, v.PrivateRoot); err != nil {
				t.Fatal(err)
			}
		}, want: "non-symlink"},
		{name: "broad private root", mutate: func(v *browserAuthoringPlanInput) { v.PrivateRoot = string(filepath.Separator) }, want: "filesystem root"},
		{name: "reserved path delimiter", mutate: func(v *browserAuthoringPlanInput) { v.TargetURL = "https://example.test/<capture-id>" }, want: "reserved delimiter"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := base
			input.Origins = append([]string(nil), base.Origins...)
			test.mutate(&input)
			if _, err := buildBrowserAuthoringPlan(input); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}

func argvContains(argv []string, wanted string) bool {
	for _, value := range argv {
		if value == wanted {
			return true
		}
	}
	return false
}

func argvHasPair(argv []string, key, value string) bool {
	for index := 0; index+1 < len(argv); index++ {
		if argv[index] == key && argv[index+1] == value {
			return true
		}
	}
	return false
}
