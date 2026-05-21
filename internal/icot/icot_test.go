package icot

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	runner "github.com/OpenUdon/openudon/internal/workflowintent"
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
	code = Main([]string{"--example", example, "--no-llm"}, strings.NewReader("save\n"), &stdout, &stderr)
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

func TestCompleteDraftResumeCancelDeletesDraftWithoutWriting(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader("cancel\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cancel failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("cancel wrote project.md or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("cancel wrote intent.hcl or unexpected stat error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after cancel: %v", err)
	}
}

func TestCompleteDraftResumeSaveWritesAndDeletesDraft(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader("save\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("save failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	for _, rel := range []string{"project.md", "workflows/intent.hcl"} {
		if _, err := os.Stat(filepath.Join(example, rel)); err != nil {
			t.Fatalf("%s missing after save: %v\nstdout:\n%s\nstderr:\n%s", rel, err, stdout.String(), stderr.String())
		}
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatalf("draft not deleted after save: %v", err)
	}
}

func TestCompleteDraftPrintWritesNoFilesAndPreservesDraft(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	draftPath := writeCompleteDraft(t, example)
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--print", "--no-llm"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("print failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, "project.md")); !os.IsNotExist(err) {
		t.Fatalf("--print wrote project.md or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(example, "workflows", "intent.hcl")); !os.IsNotExist(err) {
		t.Fatalf("--print wrote intent.hcl or unexpected stat error: %v", err)
	}
	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("--print should preserve draft: %v", err)
	}
	if !strings.Contains(stdout.String(), "----- project.md -----") || !strings.Contains(stdout.String(), "----- workflows/intent.hcl -----") {
		t.Fatalf("print output missing artifacts:\n%s", stdout.String())
	}
}

func TestNoTranscriptSkipsLocalTranscript(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm", "--no-transcript"}, strings.NewReader(testProjectInput(false)+"save\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Main failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(example, ".icot", "transcript.json")); !os.IsNotExist(err) {
		t.Fatalf("--no-transcript wrote transcript or unexpected stat error: %v", err)
	}
}

func TestProviderFromEnvDefaultsToCopilotAPI(t *testing.T) {
	t.Setenv("OPENUDON_LLM_PROVIDER", "")
	t.Setenv("COPILOT_API_BASE_URL", "")
	t.Setenv("COPILOT_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "gemini-key")
	t.Setenv("OPENAI_API_KEY", "openai-key")
	t.Setenv("ANTHROPIC_API_KEY", "anthropic-key")

	if got := providerFromEnv(); got != "copilot-api" {
		t.Fatalf("providerFromEnv() = %q, want copilot-api", got)
	}
}

func TestProviderFromEnvHonorsOpenUdonProviderOverride(t *testing.T) {
	t.Setenv("OPENUDON_LLM_PROVIDER", "gemini")

	if got := providerFromEnv(); got != "gemini" {
		t.Fatalf("providerFromEnv() = %q, want gemini", got)
	}
}

func TestCompleteDraftEditsReplaceSeededPolicyFields(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	writeCompleteDraftWithPolicy(t, example, []string{"old_token"}, "Old safety note", "Old fallback note")
	input := strings.Join([]string{
		"edit credentials",
		"new_token",
		"edit safety",
		"New safety note",
		"edit fallback",
		"New fallback note",
		"save",
	}, "\n") + "\n"
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("policy edit failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	text := string(project)
	for _, unexpected := range []string{"old_token", "Old safety note", "Old fallback note"} {
		if strings.Contains(text, unexpected) {
			t.Fatalf("project retained stale policy value %q:\n%s", unexpected, text)
		}
	}
	for _, expected := range []string{"new_token", "New safety note", "New fallback note"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("project missing edited policy value %q:\n%s", expected, text)
		}
	}
}

func TestCompleteDraftCredentialsNoneClearsSeededBindings(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	writeCompleteDraftWithPolicy(t, example, []string{"old_token"}, "Sandbox proof runs only", "Stop on errors")
	var stdout, stderr bytes.Buffer
	code := Main([]string{"--example", example, "--no-llm"}, strings.NewReader("edit credentials\nnone\nsave\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("credential clear failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project: %v", err)
	}
	text := string(project)
	if strings.Contains(text, "old_token") || !strings.Contains(text, "Credential bindings: none declared") {
		t.Fatalf("credentials were not cleared:\n%s", text)
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

func TestReconcileProjectIncludesNestedIntentDetails(t *testing.T) {
	example := t.TempDir()
	intent := nestedIntent()
	intentHCL, err := runner.RenderIntentHCL(intent)
	if err != nil {
		t.Fatalf("render intent: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(example, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	if err := os.WriteFile(filepath.Join(example, "workflows", "intent.hcl"), []byte(intentHCL), 0o644); err != nil {
		t.Fatalf("write intent: %v", err)
	}
	project := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Nested",
		Credentials:     []string{"support_api_token"},
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          "Sandbox proof runs only",
		Fallback:        "Stop on errors",
	})
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte(project), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	var stdout, stderr bytes.Buffer
	code := Main([]string{"reconcile", "--example", example, "--print"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("reconcile print failed with code %d\nstdout:\n%s\nstderr:\n%s", code, stdout.String(), stderr.String())
	}
	text := stdout.String()
	for _, expected := range []string{
		"openapi/nested.yaml",
		"`nested_lookup.ticketId` comes from `get_ticket.received_body.id`",
		"`prepare_default`: Prepare the default result",
		"support_api_token",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("reconciled project missing nested detail %q:\n%s", expected, text)
		}
	}
}

func TestLintDriftWarnsForNestedIntentFieldsAndRuntimeApprovals(t *testing.T) {
	example := t.TempDir()
	project := projectwizard.Render(projectwizard.Answers{
		ProjectName:     "Nested Drift",
		Goal:            "Handle nested workflow",
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
	intentHCL, err := runner.RenderIntentHCL(nestedIntent())
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
	text := stdout.String()
	for _, expected := range []string{
		"icot.drift.openapi: warn",
		"icot.drift.data_flow: warn",
		"icot.drift.functions: warn",
		"icot.drift.credentials: warn",
		"icot.drift.runtime.cmd: warn",
		"icot.drift.runtime.ssh: warn",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("lint output missing nested drift warning %q:\n%s", expected, text)
		}
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

func nestedIntent() *rollout.Intent {
	return &rollout.Intent{
		OpenAPI:  "openapi/nested.yaml",
		Workflow: &rollout.WorkflowMeta{Name: "nested_workflow", Description: "Handle nested workflow"},
		Inputs:   []*rollout.Input{{Name: "ticketId", Type: "string", Required: true}},
		Steps: []*rollout.Step{
			{
				Name:      "get_ticket",
				Type:      "http",
				Do:        "Fetch the ticket",
				Operation: "getTicket",
				With:      map[string]string{"ticketId": "inputs.ticketId", "api_key": "credentials.support_api_token"},
			},
			{
				Name:      "route_ticket",
				Type:      "switch",
				DependsOn: []string{"get_ticket"},
				Cases: []*rollout.StepCase{{
					Name: "urgent",
					When: "get_ticket.received_body.priority == \"urgent\"",
					Steps: []*rollout.Step{
						{
							Name:      "nested_lookup",
							Type:      "http",
							Do:        "Look up nested context",
							OpenAPI:   "openapi/nested.yaml",
							Operation: "getNested",
							Binds: []*rollout.StepBind{{
								From:   "get_ticket",
								Fields: map[string]string{"ticketId": "received_body.id"},
							}},
						},
						{Name: "run_local_command", Type: "cmd", Do: "Run approved local command"},
					},
				}},
				Default: &rollout.StepDefault{Steps: []*rollout.Step{
					{
						Name: "prepare_default",
						Type: "fnct",
						Do:   "Prepare the default result",
						Binds: []*rollout.StepBind{{
							From:   "get_ticket",
							Fields: map[string]string{"ticket": "received_body"},
						}},
					},
					{Name: "check_remote_host", Type: "ssh", Do: "Check remote host"},
				}},
			},
		},
		Outputs: []*rollout.Output{{Name: "result", From: "prepare_default.received_body"}},
	}
}

func writeCompleteDraft(t *testing.T, example string) string {
	t.Helper()
	return writeCompleteDraftWithPolicy(t, example, nil, "Sandbox proof runs only", "Stop if inputs are missing")
}

func writeCompleteDraftWithPolicy(t *testing.T, example string, credentials []string, safety, fallback string) string {
	t.Helper()
	project := projectwizard.Answers{
		ProjectName:     "Draft Project",
		Goal:            "Render a draft report",
		Credentials:     credentials,
		SideEffectScope: projectwizard.SideEffectSandboxOnly,
		Safety:          safety,
		Fallback:        fallback,
	}
	session := elicitor.SessionFromIntent(testIntent("draft_project", "Render a draft report", "render_report"), project)
	draftPath := elicitor.DraftPath(example)
	if err := elicitor.SaveDraft(draftPath, session); err != nil {
		t.Fatalf("save draft: %v", err)
	}
	return draftPath
}

func testProjectInput(withOpenAPI bool) string {
	answers := []string{
		"Render a local summary report from a runtime input",
		"guided_project",
		"Render a local summary report",
		"",
		"",
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
		"",
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
