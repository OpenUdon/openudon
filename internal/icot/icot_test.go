package icot

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/genelet/udon/pkg/runner"
)

func TestMainPreviewEOFCancelsWithoutWriting(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader(testProjectInput(false)), &stdout, &stderr)
	if code == 0 {
		t.Fatalf("Main succeeded with EOF at preview\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("EOF wrote project.md or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("EOF wrote intent.hcl or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, ".icot", "session.yaml")); err != nil {
		t.Fatalf("EOF should preserve draft session: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "unexpected EOF") {
		t.Fatalf("stderr missing unexpected EOF:\n%s", stderr.String())
	}
}

func TestBackupProjectCreatesDistinctBackups(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.md")
	if err := os.WriteFile(projectPath, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old project: %v", err)
	}
	if err := backupProject(projectPath); err != nil {
		t.Fatalf("first backup failed: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte("new\n"), 0o644); err != nil {
		t.Fatalf("write new project: %v", err)
	}
	if err := backupProject(projectPath); err != nil {
		t.Fatalf("second backup failed: %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(dir, "project.md.bak.*"))
	if err != nil {
		t.Fatalf("glob backups: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("backup count = %d, want 2: %v", len(backups), backups)
	}
	contents := map[string]bool{}
	for _, path := range backups {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read backup %s: %v", path, err)
		}
		contents[string(data)] = true
	}
	if !contents["old\n"] || !contents["new\n"] {
		t.Fatalf("backup contents = %#v, want old and new", contents)
	}
}

func TestAutosaveResumesAndDeletesAfterSave(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader(testProjectInput(false)), &stdout, &stderr)
	if code == 0 {
		t.Fatalf("first Main unexpectedly succeeded\nstdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())
	}
	draftPath := filepath.Join(example, ".icot", "session.yaml")
	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("draft missing after EOF: %v", err)
	}
	stdout.Reset()
	stderr.Reset()
	code = Main([]string{"--example", example, "--no-llm"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("resume failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); err != nil {
		t.Fatalf("project.md missing after resume: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); err != nil {
		t.Fatalf("intent.hcl missing after resume: %v", err)
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after save: %v", err)
	}
}

func TestReconcileRegeneratesProjectOnlyAndPreservesPolicy(t *testing.T) {
	example := t.TempDir()
	intent := testIntent("support_ticket", "Fetch support ticket", "get_ticket")
	intent.OpenAPI = "openapi/support.yaml"
	intent.Steps[0].Type = "http"
	intent.Steps[0].Operation = "getTicket"
	intent.Steps[0].With = map[string]string{"api_key": "credentials.support_api_token"}
	intentHCL, err := runner.RenderIntentHCL(intent)
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	intentPath := filepath.Join(example, "workflows", "intent.hcl")
	if err := os.WriteFile(intentPath, []byte(intentHCL), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	oldProject := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Old Project",
		Goal:            "Old goal",
		Credentials:     []string{"support_api_token"},
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          "Use sandbox endpoints only",
		Fallback:        "Stop if support API is unavailable",
	})
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(oldProject), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"reconcile", "--example", example, "--yes"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reconcile failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	for _, expected := range []string{"Fetch support ticket", "openapi/support.yaml", "support_api_token", "Use sandbox endpoints only", "Stop if support API is unavailable"} {
		if !strings.Contains(string(project), expected) {
			t.Fatalf("reconciled project missing %q:\n%s", expected, project)
		}
	}
	backups, err := filepath.Glob(filepath.Join(example, "project.md.bak.*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected project backup, got %v err %v", backups, err)
	}
	intentBackups, err := filepath.Glob(filepath.Join(example, "workflows", "intent.hcl.bak.*"))
	if err != nil || len(intentBackups) != 0 {
		t.Fatalf("reconcile should not back up intent, got %v err %v", intentBackups, err)
	}
}

func TestLintDriftWarnsButExitsZero(t *testing.T) {
	example := t.TempDir()
	project := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Drift",
		Goal:            "Render something else",
		UsesOpenAPI:     false,
		SideEffectScope: projectwizard.SideEffectReadOnly,
		Fallback:        "Stop on errors",
	})
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(project), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	intentHCL, err := runner.RenderIntentHCL(testIntent("drift", "Fetch a ticket", "get_ticket"))
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(intentHCL), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"lint", "--example", example}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("lint drift should exit zero, got %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "icot.drift.goal: warn") || !strings.Contains(stdout.String(), "icot.drift.inputs: warn") {
		t.Fatalf("lint output missing drift warnings:\n%s", stdout.String())
	}
}

func TestAtomicWriterDoesNotModifyFinalFilesWhenValidationFails(t *testing.T) {
	dir := t.TempDir()
	projectPath := filepath.Join(dir, "project.md")
	intentPath := filepath.Join(dir, "workflows", "intent.hcl")
	if err := os.MkdirAll(filepath.Dir(intentPath), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(projectPath, []byte("old project\n"), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	if err := os.WriteFile(intentPath, []byte("workflow {}\n"), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	err := writeGeneratedFilesAtomic([]generatedFile{
		{Path: projectPath, Content: "new project\n"},
		{Path: intentPath, Content: "not valid hcl"},
	}, true)
	if err == nil {
		t.Fatalf("atomic write unexpectedly succeeded")
	}
	project, _ := os.ReadFile(projectPath)
	intent, _ := os.ReadFile(intentPath)
	if string(project) != "old project\n" || string(intent) != "workflow {}\n" {
		t.Fatalf("files changed after failed atomic write:\nproject=%q\nintent=%q", project, intent)
	}
}

func testIntent(name, goal, stepName string) *rollout.Intent {
	return &rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: name, Description: goal},
		Inputs:   []*rollout.Input{{Name: "ticket_id", Type: "string", Required: true}},
		Steps: []*rollout.Step{{
			Name: stepName,
			Type: "fnct",
			Do:   goal,
			With: map[string]string{"ticket_id": "inputs.ticket_id"},
		}},
		Outputs: []*rollout.Output{{Name: "ticket", From: stepName + ".received_body"}},
	}
}

func testProjectInput(withOpenAPI bool) string {
	answers := []string{
		"Render a local summary report from a runtime input",
		"guided_project",
		"Render a local summary report",
	}
	if withOpenAPI {
		answers = append(answers, "yes", "openapi/support.yaml")
	} else {
		answers = append(answers, "no")
	}
	answers = append(answers,
		"summary:string",
		"render_report",
		"fnct",
		"Render the summary report",
		"summary",
		"",
		"",
		"report",
		"render_report.received_body",
		"sandbox-only",
		"",
		"Sandbox proof runs only",
		"Stop if required services are unavailable",
	)
	return strings.Join(answers, "\n") + "\n"
}
