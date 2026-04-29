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
		"project.md",
		"openapi/",
		"ramen synthesize --example",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("icot help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICreatesProjectAndDirectories(t *testing.T) {
	example := filepath.Join(t.TempDir(), "guided")
	cmd := helperCommand("--example", example)
	cmd.Stdin = strings.NewReader(projectInput(false) + "y\n")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot failed: %v\n%s", err, output)
	}
	for _, rel := range []string{"project.md", "openapi", "workflows", "expected"} {
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
		"OpenAPI: none required",
		"`cmd` is not allowed",
		"`ssh` is not allowed",
		"`support_api_token`",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("project.md missing %q:\n%s", expected, text)
		}
	}
	if !strings.Contains(string(output), "project.md preview") {
		t.Fatalf("output missing preview:\n%s", output)
	}
}

func TestCLIPrintWritesNoFiles(t *testing.T) {
	example := filepath.Join(t.TempDir(), "print")
	cmd := helperCommand("--example", example, "--print")
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
}

func TestCLIPreviewCancelWritesNoFiles(t *testing.T) {
	example := filepath.Join(t.TempDir(), "cancel")
	cmd := helperCommand("--example", example)
	cmd.Stdin = strings.NewReader(projectInput(false) + "cancel\n")
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

func TestCLIForceYesCreatesBackupAndOverwritesOnlyProjectMD(t *testing.T) {
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
	cmd := helperCommand("--example", example, "--force", "--yes", "--answers", writeAnswersFile(t, t.TempDir(), true))
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
	backups, err := filepath.Glob(filepath.Join(example, "project.md.bak.*"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one backup, got %v, err %v", backups, err)
	}
	backup, err := os.ReadFile(backups[0])
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}
	if string(backup) != "old\n" {
		t.Fatalf("backup content = %q", backup)
	}
	for rel, want := range preserved {
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
	cmd.Stdin = strings.NewReader(strings.Repeat("\n", 12) + "y\n")
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
	data := `{"project_name":"JSON Project","goal":"Call an API","uses_openapi":true,"openapi":"Support API: use ` + "`openapi/support.yaml`" + `","credentials":["support_api_token"]}`
	if err := os.WriteFile(answersPath, []byte(data), 0o644); err != nil {
		t.Fatalf("write answers: %v", err)
	}
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
	cmd.Env = append(os.Environ(), "ICOT_CLI_HELPER=1")
	return cmd
}

func projectInput(withOpenAPI bool) string {
	answers := []string{
		"Guided Project",
		"Fetch a ticket and prepare a draft",
		"`ticket_id`: required string",
		"Stored draft reply",
		"Pass ticket body to draft writer",
		"`write_draft`: inputs ticket body; outputs draft id",
	}
	if withOpenAPI {
		answers = append(answers, "yes", "Support API: use `openapi/support.yaml`")
	} else {
		answers = append(answers, "")
	}
	answers = append(answers,
		"",
		"",
		"support_api_token",
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
