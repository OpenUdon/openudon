package main

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIHelpDocumentsFlags(t *testing.T) {
	cmd := helperCommand("--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: icot",
		"--example",
		"--dir",
		"--force",
		"--yes",
		"--print",
		"--from-example",
		"--answers",
		"-no-llm",
		"-provider",
		"-model",
		"-temperature",
		"project.md",
		"intent.hcl",
		"openapi/",
		"ramen build --example",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("icot help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICreatesProjectAndDirectories(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	cmd := helperCommand("--example", example, "--no-llm")
	cmd.Stdin = strings.NewReader(projectInput(false))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot failed: %v\n%s", err, output)
	}
	for _, rel := range []string{"project.md", "workflows/intent.hcl", "openapi", "workflows", "expected"} {
		if _, err := os.Stat(filepath.Join(example, rel)); err != nil {
			t.Fatalf("expected %s to exist: %v\n%s", rel, err, output)
		}
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project.md: %v", err)
	}
	text := string(project)
	for _, expected := range []string{
		"# Guided Project",
		"Render a local summary report",
		"OpenAPI: none required",
		"`cmd` is not allowed",
		"`ssh` is not allowed",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("project.md missing %q:\n%s", expected, text)
		}
	}
	intent, err := os.ReadFile(filepath.Join(example, "workflows", "intent.hcl"))
	if err != nil {
		t.Fatalf("read intent.hcl: %v", err)
	}
	if !strings.Contains(string(intent), `workflow`) || !strings.Contains(string(intent), `render_report`) {
		t.Fatalf("intent.hcl missing expected content:\n%s", intent)
	}
	if !strings.Contains(string(output), "current draft") {
		t.Fatalf("output missing final verification:\n%s", output)
	}
}

func TestCLIPrintWritesNoFiles(t *testing.T) {
	example := filepath.Join(t.TempDir(), "print")
	cmd := helperCommand("--example", example, "--print", "--no-llm")
	cmd.Stdin = strings.NewReader(projectInput(false))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot --print failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(example); !os.IsNotExist(err) {
		t.Fatalf("--print created files or unexpected stat error: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "OpenAPI: none required") {
		t.Fatalf("--print output missing rendered project:\n%s", output)
	}
	if !strings.Contains(string(output), "workflows/intent.hcl") || !strings.Contains(string(output), "render_report") {
		t.Fatalf("--print output missing rendered intent:\n%s", output)
	}
}

func TestCLICancelWritesNoFiles(t *testing.T) {
	example := filepath.Join(t.TempDir(), "cancel")
	cmd := helperCommand("--example", example, "--no-llm")
	cmd.Stdin = strings.NewReader(projectInputBeforeVerify(false) + "cancel\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot cancel failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(example); !os.IsNotExist(err) {
		t.Fatalf("cancel created files or unexpected stat error: %v\n%s", err, output)
	}
}

func TestCLIRefusesExistingProjectWithoutForce(t *testing.T) {
	example := t.TempDir()
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write existing project.md: %v", err)
	}
	cmd := helperCommand("--example", example, "--answers", writeAnswersFile(t, t.TempDir(), false))
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("icot unexpectedly overwrote existing project.md:\n%s", output)
	}
	if !strings.Contains(string(output), "already exists") || !strings.Contains(string(output), "--force") {
		t.Fatalf("existing project failure missing guidance:\n%s", output)
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project.md: %v", err)
	}
	if string(project) != "old\n" {
		t.Fatalf("project.md changed without --force:\n%s", project)
	}
}

func TestCLIForceYesCreatesBackupsAndOverwritesGeneratedFiles(t *testing.T) {
	example := t.TempDir()
	for _, rel := range []string{"openapi", "workflows", "expected"} {
		if err := os.MkdirAll(filepath.Join(example, rel), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
	}
	preserved := map[string]string{
		"openapi/support.yaml":  "openapi: 3.0.0\n",
		"workflows/intent.hcl":  "workflow \"old\" {}\n",
		"expected/quality.json": "{\"status\":\"old\"}\n",
	}
	for rel, content := range preserved {
		if err := os.WriteFile(filepath.Join(example, rel), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write existing project.md: %v", err)
	}
	cmd := helperCommand("--example", example, "--force", "--yes", "--answers", writeSessionFile(t, t.TempDir()))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot --force --yes failed: %v\n%s", err, output)
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project.md: %v", err)
	}
	if !strings.Contains(string(project), "openapi/support.yaml") {
		t.Fatalf("project.md was not overwritten with new content:\n%s", project)
	}
	projectBackups, err := filepath.Glob(filepath.Join(example, "project.md.bak.*"))
	if err != nil || len(projectBackups) != 1 {
		t.Fatalf("expected one project backup, got %v, err %v", projectBackups, err)
	}
	backup, err := os.ReadFile(projectBackups[0])
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backup) != "old\n" {
		t.Fatalf("backup content = %q", backup)
	}
	intentBackups, err := filepath.Glob(filepath.Join(example, "workflows", "intent.hcl.bak.*"))
	if err != nil || len(intentBackups) != 1 {
		t.Fatalf("expected one intent backup, got %v, err %v", intentBackups, err)
	}
	intent, err := os.ReadFile(filepath.Join(example, "workflows", "intent.hcl"))
	if err != nil {
		t.Fatalf("read overwritten intent: %v", err)
	}
	if !strings.Contains(string(intent), "openapi/support.yaml") {
		t.Fatalf("intent.hcl was not overwritten with new content:\n%s", intent)
	}
	for rel, want := range preserved {
		if rel == "workflows/intent.hcl" {
			continue
		}
		got, err := os.ReadFile(filepath.Join(example, rel))
		if err != nil {
			t.Fatalf("read preserved %s: %v", rel, err)
		}
		if string(got) != want {
			t.Fatalf("%s changed by icot --force:\ngot  %q\nwant %q", rel, got, want)
		}
	}
}

func TestCLIFromExampleSeedsDefaults(t *testing.T) {
	example := filepath.Join(t.TempDir(), "seeded")
	cmd := helperCommand("--example", example, "--from-example", "examples/eval/runtime-only-render")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot --from-example failed: %v\n%s", err, output)
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read seeded project.md: %v", err)
	}
	text := string(project)
	if !strings.Contains(text, "OpenAPI: none required") || !strings.Contains(text, "summary report") {
		t.Fatalf("seeded project missing expected runtime-only content:\n%s", text)
	}
}

func TestCLIAnswersYAMLNoPrompts(t *testing.T) {
	example := filepath.Join(t.TempDir(), "answers")
	cmd := helperCommand("--example", example, "--answers", writeAnswersFile(t, t.TempDir(), false))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot --answers failed: %v\n%s", err, output)
	}
	if strings.Contains(string(output), "Project name:") || strings.Contains(string(output), "preview") {
		t.Fatalf("--answers unexpectedly prompted:\n%s", output)
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project.md: %v", err)
	}
	if !strings.Contains(string(project), "OpenAPI: none required") {
		t.Fatalf("answers project missing runtime-only OpenAPI policy:\n%s", project)
	}
}

func TestCLIAnswersJSONPrintWritesNoFiles(t *testing.T) {
	dir := t.TempDir()
	answersPath := filepath.Join(dir, "answers.json")
	writeSessionJSON(t, answersPath)
	example := filepath.Join(dir, "out")
	cmd := helperCommand("--example", example, "--answers", answersPath, "--print")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot --answers --print failed: %v\n%s", err, output)
	}
	if _, err := os.Stat(example); !os.IsNotExist(err) {
		t.Fatalf("--answers --print created files or unexpected stat error: %v\n%s", err, output)
	}
	text := string(output)
	if !strings.Contains(text, "# JSON Project") || !strings.Contains(text, "`support_api_token`") || !strings.Contains(text, "openapi/support.yaml") {
		t.Fatalf("answers print missing rendered content:\n%s", text)
	}
}

func TestCLILintExampleAndFile(t *testing.T) {
	cmd := helperCommand("lint", "--example", "examples/eval/runtime-only-render")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot lint --example failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "project.authoring.goal") {
		t.Fatalf("lint output missing authoring checks:\n%s", output)
	}
	path := filepath.Join(t.TempDir(), "project.md")
	if err := os.WriteFile(path, []byte("# Bad\n\n## Goal\n\nx\n\napi_key = \"AKIA1234567890ABCDEF\"\n"), 0o644); err != nil {
		t.Fatalf("write project: %v", err)
	}
	cmd = helperCommand("lint", "--file", path)
	output, err = cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("icot lint succeeded on secret-like content:\n%s", output)
	}
	if !strings.Contains(string(output), "project.no_secrets") {
		t.Fatalf("lint output missing secret check:\n%s", output)
	}
}

func helperCommand(args ...string) *exec.Cmd {
	cmdArgs := append([]string{"-test.run=TestCLIHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Dir = filepath.Join("..", "..")
	cmd.Env = append(os.Environ(),
		"ICOT_CLI_HELPER=1",
		"GEMINI_API_KEY=",
		"OPENAI_API_KEY=",
		"ANTHROPIC_API_KEY=",
	)
	return cmd
}

func projectInput(withOpenAPI bool) string {
	return projectInputBeforeVerify(withOpenAPI) + "save\n"
}

func projectInputBeforeVerify(withOpenAPI bool) string {
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

func writeAnswersFile(t *testing.T, dir string, withOpenAPI bool) string {
	t.Helper()
	path := filepath.Join(dir, "answers.yaml")
	openapi := "false"
	openapiText := ""
	if withOpenAPI {
		openapi = "true"
		openapiText = "openapi: 'Support API: use `openapi/support.yaml`'\n"
	}
	data := "project_name: Guided Project\n" +
		"goal: Fetch a ticket and prepare a draft\n" +
		"inputs: '`ticket_id`: required string'\n" +
		"outputs: Stored draft reply\n" +
		"data_flow: Pass ticket body to draft writer\n" +
		"function_contracts: '`write_draft`: inputs ticket body; outputs draft id'\n" +
		"uses_openapi: " + openapi + "\n" +
		openapiText +
		"credentials:\n" +
		"  - support_api_token\n" +
		"safety: Sandbox proof runs only\n" +
		"fallback: Stop if required services are unavailable\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write answers file: %v", err)
	}
	return path
}

func writeSessionFile(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "session.yaml")
	data := "project:\n" +
		"  project_name: Guided Project\n" +
		"  goal: Call an API\n" +
		"credentials:\n" +
		"  - support_api_token\n" +
		"safety: Sandbox proof runs only\n" +
		"fallback: Stop if required services are unavailable\n" +
		"side_effect_scope: sandbox-only\n" +
		"intent:\n" +
		"  openapi: openapi/support.yaml\n" +
		"  workflow:\n" +
		"    name: guided_project\n" +
		"    description: Call an API\n" +
		"  steps:\n" +
		"    - name: call_support\n" +
		"      type: http\n" +
		"      do: Call the support API\n" +
		"      operation: getTicket\n" +
		"      with:\n" +
		"        ticketId: inputs.ticketId\n" +
		"  inputs:\n" +
		"    - name: ticketId\n" +
		"      type: string\n" +
		"      required: true\n" +
		"  outputs:\n" +
		"    - name: ticket\n" +
		"      from: call_support.received_body\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write session file: %v", err)
	}
	return path
}

func writeSessionJSON(t *testing.T, path string) {
	t.Helper()
	data := `{
  "project": {"project_name": "JSON Project", "goal": "Call an API"},
  "credentials": ["support_api_token"],
  "side_effect_scope": "sandbox-only",
  "intent": {
    "openapi": "openapi/support.yaml",
    "workflow": {"name": "json_project", "description": "Call an API"},
    "inputs": [{"name": "ticketId", "type": "string", "required": true}],
    "steps": [{
      "name": "call_support",
      "type": "http",
      "do": "Call the support API",
      "operation": "getTicket",
      "with": {"ticketId": "inputs.ticketId"}
    }],
    "outputs": [{"name": "ticket", "from": "call_support.received_body"}]
  }
}`
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write session JSON: %v", err)
	}
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("ICOT_CLI_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			break
		}
	}
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	main()
	os.Exit(0)
}
