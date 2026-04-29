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
		"--force",
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
	cmd.Stdin = strings.NewReader(projectInput(false))
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
}

func TestCLIRefusesExistingProjectWithoutForce(t *testing.T) {
	example := t.TempDir()
	if err := os.WriteFile(filepath.Join(example, "project.md"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write existing project.md: %v", err)
	}
	cmd := helperCommand("--example", example)
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

func TestCLIForceOverwritesOnlyProjectMD(t *testing.T) {
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
	cmd := helperCommand("--example", example, "--force")
	cmd.Stdin = strings.NewReader(projectInput(true))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("icot --force failed: %v\n%s", err, output)
	}
	project, err := os.ReadFile(filepath.Join(example, "project.md"))
	if err != nil {
		t.Fatalf("read project.md: %v", err)
	}
	if !strings.Contains(string(project), "openapi/support.yaml") {
		t.Fatalf("project.md was not overwritten with new content:\n%s", project)
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

func helperCommand(args ...string) *exec.Cmd {
	cmdArgs := append([]string{"-test.run=TestCLIHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
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
