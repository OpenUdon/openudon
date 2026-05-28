package main

import (
	"bytes"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/localcheck"
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
		"Usage: openudon synthesize",
		"Examples:",
		"gpt-5.4-mini",
		"Artifacts:",
		"workflows/intent.hcl",
		"expected/review-handoff.json",
		"expected/quality.json",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLIBuildHelpStatesDeterministicIntentBuild(t *testing.T) {
	cmd := helperCommand("build", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: openudon build --example examples/<name> [--provider label --model label]",
		"Deterministically regenerate workflow",
		"no LLM is required",
		"openudon build --example examples/support-email",
		"optional review-evidence label for build",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("build help missing %q:\n%s", expected, text)
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
		"Usage: openudon eval",
		"--release-gate",
		"--min-briefs",
		"--compare",
		"--no-compare",
		"--archive-dir",
		"gpt-5.4-mini",
		"writes JSON/Markdown reports",
		"Normal evals print comparison regressions",
		"With --release-gate, absolute release criteria and comparison regressions fail",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("eval help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLIRunHelpIncludesApprovalGates(t *testing.T) {
	cmd := helperCommand("run", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: openudon run",
		"--tier sandbox|production",
		"--approval",
		"--dry-run",
		"approved_for_sandbox",
		"approved_for_production",
		"trusted executor",
		"run evidence",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("run help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLIConvertTFHelpIncludesContract(t *testing.T) {
	cmd := helperCommand("convert", "tf", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("convert tf help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: openudon convert tf",
		"--config-dir",
		"--api-source",
		"--openapi",
		"--action",
		"--target",
		"--strict",
		"does not execute Terraform",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("convert tf help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICatalogHelpIncludesProviderCommands(t *testing.T) {
	cmd := helperCommand("catalog", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("catalog help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: openudon catalog <command>",
		"list",
		"inspect",
		"advisory",
		"specs",
		"security-report",
		"import-openapi",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("catalog help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICatalogListJSONSmoke(t *testing.T) {
	cmd := helperCommand("catalog", "list", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("catalog list failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		`"id": "gmail"`,
		`"machine_availability": "known"`,
		`"id": "github"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("catalog list missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICatalogInspectDiscoveryProvider(t *testing.T) {
	cmd := helperCommand("catalog", "inspect", "gmail", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("catalog inspect failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		`"id": "gmail"`,
		`"kind": "google-discovery"`,
		`"security_status": "overlay-required"`,
		`"overlay_id": "gmail-discovery-auth-overlay"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("catalog inspect missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICatalogAdvisoryExampleUsesIntentProviders(t *testing.T) {
	example := t.TempDir()
	mustWriteCLIFile(t, filepath.Join(example, "workflows", "intent.hcl"), []byte(`
workflow {
  name = "github_issue_triage"
}

step "create_issue" {
  type     = "openapi"
  provider = "github"
  openapi  = "openapi/github.yaml"
  operation = "issues/create"
}
`))
	cmd := helperCommand("catalog", "advisory", "--example", example)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("catalog advisory example failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"## Catalog Advisory",
		"Provider: `GitHub` (`github`)",
		"Explicit OpenAPI input overrides built-in catalog spec",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("catalog advisory missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICatalogImportOpenAPIRejectsDiscoveryOnlyProvider(t *testing.T) {
	cmd := helperCommand("catalog", "import-openapi", "--provider", "gmail", "--example", t.TempDir())
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("catalog import-openapi unexpectedly succeeded:\n%s", output)
	}
	if !strings.Contains(string(output), "has no directly importable OpenAPI spec") {
		t.Fatalf("catalog import-openapi error missing boundary:\n%s", output)
	}
}

func TestCLIN8nBridgeHelpIncludesBoundary(t *testing.T) {
	cmd := helperCommand("n8n-bridge", "validate", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("n8n-bridge help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: openudon n8n-bridge validate",
		"--root",
		"--file",
		"openudon.n8n-pattern-summary.v1",
		"does not import, execute, or emulate n8n workflows",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("n8n-bridge help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLIN8nBridgeValidateSmoke(t *testing.T) {
	root := t.TempDir()
	mustWriteCLIFile(t, filepath.Join(root, "n8n-slack-message-post", "reference", "n8n-bridge.json"), []byte(`{
  "version": "openudon.n8n-pattern-summary.v1",
  "fixture": "n8n-slack-message-post",
  "boundary": "authoring_assistance_only",
  "source": {"kind": "n8n_workflow_fixture"},
  "services": [{"name": "Slack"}],
  "nodes": [{"name": "Slack", "type": "n8n-nodes-base.slack", "mapping_status": "advisory"}],
  "generated_candidates": {"promoted": false},
  "validation": {"status": "advisory"}
}`))
	cmd := helperCommand("n8n-bridge", "validate", "--root", root)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("n8n-bridge validate failed: %v\n%s", err, output)
	}
	text := string(output)
	if !strings.Contains(text, "openudon: n8n bridge validated 1 summary file") || !strings.Contains(text, "n8n-slack-message-post") {
		t.Fatalf("unexpected n8n-bridge output:\n%s", text)
	}
}

func TestCLIConvertTFWritesDraftArtifacts(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "tf")
	openAPIPath := filepath.Join(root, "openapi.yaml")
	outDir := filepath.Join(root, "out")
	mustWriteCLIFile(t, filepath.Join(configDir, "main.tf"), []byte(`
resource "aws_instance" "web" {
  name = "web"
}
`))
	mustWriteCLIFile(t, openAPIPath, []byte(`openapi: 3.0.0
info:
  title: AWS Test
  version: v1
paths:
  /instances:
    post:
      operationId: createAwsInstance
      responses:
        "200":
          description: ok
`))
	cmd := helperCommand("convert", "tf", "--config-dir", configDir, "--openapi", "aws="+openAPIPath, "--action", "create", "--out", outDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("convert tf failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "openudon: convert tf wrote") {
		t.Fatalf("convert output missing summary:\n%s", output)
	}
	for _, rel := range []string{"project.md", "workflows/intent.hcl", "expected/diagnostics.json", "expected/diagnostics.md", "expected/review.md"} {
		if _, err := os.Stat(filepath.Join(outDir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
}

func TestCLIApprovalTemplateHelpIncludesSchema(t *testing.T) {
	cmd := helperCommand("approval-template", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("approval-template help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: openudon approval-template",
		"approved_for_sandbox|approved_for_production",
		"openudon.approval.v1",
		"package_sha256",
		"expires_at",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("approval-template help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLIReadinessHelpIncludesXRD007Report(t *testing.T) {
	cmd := helperCommand("readiness", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("readiness help failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"Usage: openudon readiness",
		"--run-gates",
		"--out",
		"openudon.local-readiness.v1",
		"without printing secret values",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("readiness help missing %q:\n%s", expected, text)
		}
	}
}

func TestCLICheckDocMemorySmoke(t *testing.T) {
	root := t.TempDir()
	for _, rel := range localcheck.RequiredMemoryFiles {
		mustWriteCLIFile(t, filepath.Join(root, filepath.FromSlash(rel)), []byte(rel+"\n"))
	}
	cmd := helperCommand("check-doc-memory")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check-doc-memory failed: %v\n%s", err, output)
	}
	text := string(output)
	for _, expected := range []string{
		"openudon: doc memory check passed",
		"openudon: checked memory-bank/product.md",
		"openudon: checked evolution/result-v1.md",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("check-doc-memory output missing %q:\n%s", expected, text)
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
		"side_effects.environment":         "production handoff approval",
		"credentials.security_schemes":     "symbolic credential binding names",
		"openapi.discovery":                "operation IDs",
		"intent.data_flow.required_params": "required OpenAPI path",
		"intent.data_flow.response_paths":  "avoid guessing SaaS response paths",
		"intent.data_flow.explicit":        "request field sources",
		"intent.openapi_operations":        "listed in local OpenAPI documents",
		"review.approval_states":           "approved_for_sandbox",
		"review.sandbox_handoff":           "sandbox or proof runs",
		"review.trusted_runner":            "trusted-runner handoff command",
		"review.trusted_runner_dry_run":    "trusted-runner dry-run command",
		"review.production_boundary":       "does not directly execute production workflows",
		"review.approval_artifact":         "package_sha256",
		"review.credential_scope":          "credential scope matrix",
		"review.side_effect_risk":          "approved sandbox/production handoff states",
		"review_handoff.contract":          "review handoff",
	}
	for code, expected := range cases {
		got := nextActionForQualityCheck(code)
		if !strings.Contains(got, expected) {
			t.Fatalf("next action for %s = %q, want substring %q", code, got, expected)
		}
	}
}

func TestValidateUWSPathValidatesDirectoryArtifacts(t *testing.T) {
	dir := t.TempDir()
	schema := filepath.Join(dir, "schema.json")
	mustWriteCLIFile(t, schema, []byte(`{"type":"object","required":["uws"]}`))
	mustWriteCLIFile(t, filepath.Join(dir, "nested", "workflow.uws.yaml"), []byte("uws: 1.0.0\n"))
	mustWriteCLIFile(t, filepath.Join(dir, "ignored.yaml"), []byte("uws: 1.0.0\n"))

	var out bytes.Buffer
	err := validateUWSPathWithSchema(dir, &out, func(string) string {
		return schema
	}, false)
	if err != nil {
		t.Fatalf("validateUWSPath returned error: %v", err)
	}
	text := out.String()
	if !strings.Contains(text, "found 1 UWS artifact") || !strings.Contains(text, "workflow.uws.yaml is valid UWS") {
		t.Fatalf("unexpected validate output:\n%s", text)
	}
}

func TestValidateUWSPathAcceptsUWS13AsyncAPI(t *testing.T) {
	dir := t.TempDir()
	doc := filepath.Join(dir, "workflow.uws.yaml")
	mustWriteCLIFile(t, doc, []byte(`uws: 1.3.0
info:
  title: async source
  version: 1.0.0
sourceDescriptions:
  - name: billing_events
    url: asyncapi/billing-events.yaml
    type: asyncapi
operations:
  - operationId: publish_invoice
    sourceDescription: billing_events
    sourceOperationId: publishInvoice
workflows:
  - workflowId: main
    type: sequence
    steps:
      - stepId: publish_invoice
        operationRef: publish_invoice
`))

	var out bytes.Buffer
	if err := validateUWSPath(doc, &out, false); err != nil {
		t.Fatalf("validateUWSPath returned error: %v", err)
	}
	if !strings.Contains(out.String(), "is valid UWS") {
		t.Fatalf("unexpected validate output:\n%s", out.String())
	}
}

func TestValidateUWSPathReportsDirectoryWithNoArtifacts(t *testing.T) {
	var out bytes.Buffer
	dir := t.TempDir()
	err := validateUWSPathWithSchema(dir, &out, func(string) string { return "" }, false)
	if err == nil || !strings.Contains(err.Error(), "no UWS artifacts found") {
		t.Fatalf("expected no-artifacts error, got %v", err)
	}
	if out.String() != "" {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestValidateUWSPathAllowsEmptyDirectoryWhenRequested(t *testing.T) {
	var out bytes.Buffer
	dir := t.TempDir()
	if err := validateUWSPathWithSchema(dir, &out, func(string) string { return "" }, true); err != nil {
		t.Fatalf("validateUWSPath returned error: %v", err)
	}
	if !strings.Contains(out.String(), "no UWS artifacts found") {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func mustWriteCLIFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func helperCommand(args ...string) *exec.Cmd {
	cmdArgs := append([]string{"-test.run=TestCLIHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(), "OPENUDON_CLI_HELPER=1")
	return cmd
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("OPENUDON_CLI_HELPER") != "1" {
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
