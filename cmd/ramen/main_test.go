package main

import (
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIVersionSmoke(t *testing.T) {
	cmd := helperCommand("version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version failed: %v\n%s", err, output)
	}
	if strings.TrimSpace(string(output)) != version {
		t.Fatalf("version output = %q, want %q", output, version)
	}
}

func TestCLIUnknownCommandSmoke(t *testing.T) {
	cmd := helperCommand("nope")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("unknown command succeeded:\n%s", output)
	}
	if !strings.Contains(string(output), `unknown command "nope"`) {
		t.Fatalf("unknown command output missing error:\n%s", output)
	}
}

func TestCLIArtifactHelpIncludesExamplesAndArtifacts(t *testing.T) {
	cmd := helperCommand("synthesize", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("synthesize help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: ramen synthesize",
		"Examples:",
		"gemini-2.5-flash",
		"Artifacts:",
		"workflows/intent.hcl",
		"expected/quality.json",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLIEvalHelpIncludesReleaseGate(t *testing.T) {
	cmd := helperCommand("eval", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("eval help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: ramen eval",
		"--release-gate",
		"--min-briefs",
		"--compare",
		"--no-compare",
		"--archive-dir",
		"gemini-2.5-flash",
		"writes JSON/Markdown reports",
		"Normal evals print comparison regressions",
		"With --release-gate, absolute release criteria and comparison regressions fail",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("eval help missing %q:\n%s", expected, text)
		}
	}
}

func TestNextActionForQualityCheck(t *testing.T) {
	got := nextActionForQualityCheck("artifacts.no_secrets")
	if !strings.Contains(got, "credential binding names") {
		t.Fatalf("unexpected next action: %q", got)
	}
	got = nextActionForQualityCheck("review.credential_bindings")
	if !strings.Contains(got, "Credentials and Secrets") {
		t.Fatalf("unexpected credential review action: %q", got)
	}
	cases := map[string]string{
		"side_effects.environment":     "production handoff approval",
		"credentials.security_schemes": "OpenAPI security schemes",
		"review.approval_states":       "approved_for_sandbox",
		"review.sandbox_handoff":       "sandbox or proof runs",
		"review.trusted_runner":        "trusted-runner handoff command",
		"review.production_boundary":   "does not directly execute production workflows",
	}
	for code, expected := range cases {
		got := nextActionForQualityCheck(code)
		if !strings.Contains(got, expected) {
			t.Fatalf("next action for %s = %q, want substring %q", code, got, expected)
		}
	}
}

func helperCommand(args ...string) *exec.Cmd {
	cmdArgs := append([]string{"-test.run=TestCLIHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "RAMEN_CLI_HELPER=1")
	return cmd
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("RAMEN_CLI_HELPER") != "1" {
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
