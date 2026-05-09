package readiness

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/config"
)

type fakeRunner map[string]CommandResult

func (f fakeRunner) Run(_ context.Context, _ string, name string, args ...string) CommandResult {
	key := strings.Join(append([]string{name}, args...), " ")
	if result, ok := f[key]; ok {
		return result
	}
	return CommandResult{}
}

func TestBuildStaticReadinessReport(t *testing.T) {
	root := writeReadinessFixture(t)
	report, err := Build(context.Background(), Options{
		RepoRoot: root,
		Now:      fixedReadinessNow,
		Runner: fakeRunner{
			"git diff --check":       {},
			"git status --porcelain": {Output: "M README.md"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Version != ReportVersion || report.GeneratedAt != "2026-04-29T12:00:00Z" {
		t.Fatalf("unexpected report metadata: %#v", report)
	}
	if report.Status != "warn" {
		t.Fatalf("status = %q, want warn", report.Status)
	}
	if !hasCheck(report.Siblings, "uws", "pass") {
		t.Fatalf("sibling checks missing uws pass: %#v", report.Siblings)
	}
	if !hasCheck(report.IgnoredArtifacts, ".openudon-run/", "pass") || !hasCheck(report.IgnoredArtifacts, "eval/readiness/", "pass") {
		t.Fatalf("ignored artifact checks incomplete: %#v", report.IgnoredArtifacts)
	}
	if !hasCheck(report.DeterministicGates, "go.test", "skip") {
		t.Fatalf("expected skipped gate checks: %#v", report.DeterministicGates)
	}
	if !hasCheck(report.Git, "git.status", "warn") {
		t.Fatalf("expected dirty git warning: %#v", report.Git)
	}
	if report.AutomationPolicy.HostedCIEnabled || report.AutomationPolicy.RealProviderAutomation || report.AutomationPolicy.RequiredMode != "local_manual" {
		t.Fatalf("unexpected automation policy: %#v", report.AutomationPolicy)
	}
}

func TestBuildReadinessRunsDeterministicGates(t *testing.T) {
	root := writeReadinessFixture(t)
	report, err := Build(context.Background(), Options{
		RepoRoot: root,
		RunGates: true,
		Now:      fixedReadinessNow,
		Runner: fakeRunner{
			"git diff --check":       {},
			"git status --porcelain": {},
			"go test ./...":          {},
			"go vet ./...":           {},
			"make check":             {},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "pass" {
		t.Fatalf("status = %q, want pass: %#v", report.Status, report)
	}
	for _, name := range []string{"go.test", "go.vet", "make.check"} {
		if !hasCheck(report.DeterministicGates, name, "pass") {
			t.Fatalf("missing gate %s pass: %#v", name, report.DeterministicGates)
		}
	}
}

func TestBuildReadinessFailsMissingSiblingAndIgnore(t *testing.T) {
	root := writeReadinessFixture(t)
	if err := os.RemoveAll(filepath.Join(filepath.Dir(root), "uws")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("eval/runs/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Build(context.Background(), Options{
		RepoRoot: root,
		Runner: fakeRunner{
			"git diff --check":       {},
			"git status --porcelain": {},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "fail" {
		t.Fatalf("status = %q, want fail", report.Status)
	}
	if !hasCheck(report.Siblings, "uws", "fail") || !hasCheck(report.IgnoredArtifacts, ".openudon-run/", "fail") {
		t.Fatalf("expected missing sibling and ignore failures: %#v %#v", report.Siblings, report.IgnoredArtifacts)
	}
}

func TestWriteDoesNotExposeProviderSecretValues(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "secret-value")
	t.Setenv("COPILOT_API_BASE_URL", "http://localhost:4141")
	root := writeReadinessFixture(t)
	report, err := Build(context.Background(), Options{
		RepoRoot: root,
		Runner: fakeRunner{
			"git diff --check":       {},
			"git status --porcelain": {},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := Write(&b, report); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "secret-value") {
		t.Fatalf("readiness report exposed secret value:\n%s", b.String())
	}
	if !strings.Contains(b.String(), `"name": "GEMINI_API_KEY"`) || !strings.Contains(b.String(), `"name": "COPILOT_API_BASE_URL"`) || !strings.Contains(b.String(), `"present": true`) {
		t.Fatalf("readiness report did not record provider env presence:\n%s", b.String())
	}
	var decoded Report
	if err := json.Unmarshal([]byte(b.String()), &decoded); err != nil {
		t.Fatal(err)
	}
}

func writeReadinessFixture(t *testing.T) string {
	t.Helper()
	parent := t.TempDir()
	root := filepath.Join(parent, "openudon")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range config.RequiredSiblings() {
		if err := os.Mkdir(filepath.Join(parent, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	gitignore := strings.Join([]string{
		".openudon-run/",
		"approvals/",
		"eval/artifacts/",
		"eval/readiness/",
		"eval/runs/",
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func fixedReadinessNow() time.Time {
	return time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
}

func hasCheck(checks []Check, name, status string) bool {
	for _, check := range checks {
		if check.Name == name && check.Status == status {
			return true
		}
	}
	return false
}
