package eval

import (
	"path/filepath"
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestXRD008RuntimeProfileFixtureCoverage(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "eval")

	render := parseReferenceIntent(t, root, "runtime-only-render")
	if step := findStep(render, "render_report"); step == nil || step.Type != "fnct" {
		t.Fatalf("runtime-only-render step = %#v, want fnct render_report", step)
	}

	allowedCmd := parseReferenceIntent(t, root, "cmd-allowed-deploy")
	if step := findStep(allowedCmd, "check_deploy_status"); step == nil || step.Type != "cmd" {
		t.Fatalf("cmd-allowed-deploy step = %#v, want cmd check_deploy_status", step)
	}
	assertFixtureFileContains(t, root, "cmd-allowed-deploy", "project.md", "`cmd` is explicitly allowed", "`ssh` is not allowed")

	deniedCmd := parseReferenceIntent(t, root, "cmd-disallowed-deploy")
	if step := findStep(deniedCmd, "check_deploy_status"); step == nil || step.Type != "cmd" {
		t.Fatalf("cmd-disallowed-deploy negative step = %#v, want cmd check_deploy_status", step)
	}
	assertFixtureFileContains(t, root, "cmd-disallowed-deploy", "project.md", "`cmd` and `ssh` are not allowed")

	profile := parseReferenceIntent(t, root, "profile-boundary-manifest")
	step := findStep(profile, "render_export_manifest")
	if step == nil {
		t.Fatal("profile-boundary-manifest missing render_export_manifest step")
	}
	if step.Type != "fnct" {
		t.Fatalf("profile-boundary-manifest runtime = %q, want fnct", step.Type)
	}
	for _, disallowed := range []string{"cmd", "ssh", "sql", "smtp", "llm"} {
		if referenceIntentUsesRuntime(profile, disallowed) {
			t.Fatalf("profile-boundary-manifest reference intent invented disallowed runtime %q", disallowed)
		}
	}
	assertFixtureFileContains(t, root, "profile-boundary-manifest", "project.md", "direct SQL/profile execution are not allowed", "Do not emit `sql`, `smtp`, `llm`, `x-udon-*`")
	assertFixtureFileContains(t, root, "profile-boundary-manifest", filepath.Join("reference", "policy.json"), "trusted fnct manifest", "SQL, SSH, or x-udon profile runtime semantics")
}

func referenceIntentUsesRuntime(intent *rollout.Intent, runtime string) bool {
	if intent == nil {
		return false
	}
	for _, step := range flattenSteps(intent) {
		if step != nil && strings.EqualFold(strings.TrimSpace(step.Type), runtime) {
			return true
		}
	}
	return false
}
