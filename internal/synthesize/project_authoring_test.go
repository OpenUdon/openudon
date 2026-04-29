package synthesize

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeProjectExtractsPromptRequirements(t *testing.T) {
	policy := analyzeProject(`# Demo

## Inputs

- ticket_id: required string from event.

## Outputs

- result: from send_email.received_body

## Data Flow

- Pass get_ticket.received_body.summary to send_email.body.

## Function Contracts

- send_email
  - Inputs: to, subject, body.
  - Outputs: status.
  - Side effects: sends email through approved runtime.
`)
	if len(policy.Inputs) != 1 || policy.Inputs[0].Name != "ticket_id" || policy.Inputs[0].Type != "string" || !policy.Inputs[0].Required {
		t.Fatalf("unexpected inputs: %#v", policy.Inputs)
	}
	if len(policy.Outputs) != 1 || policy.Outputs[0].From != "send_email.received_body" {
		t.Fatalf("unexpected outputs: %#v", policy.Outputs)
	}
	if len(policy.BindingHints) != 1 || policy.BindingHints[0].From != "get_ticket.received_body.summary" || policy.BindingHints[0].To != "send_email.body" || policy.BindingHints[0].Field != "body" {
		t.Fatalf("unexpected binding hints: %#v", policy.BindingHints)
	}
	if len(policy.FunctionContracts) != 1 || policy.FunctionContracts[0].Name != "send_email" || len(policy.FunctionContracts[0].Inputs) != 3 {
		t.Fatalf("unexpected function contracts: %#v", policy.FunctionContracts)
	}
}

func TestAnalyzeProjectExtractsLiteralRequestHints(t *testing.T) {
	policy := analyzeProject("# Demo\n\n" +
		"## Data Flow\n\n" +
		"- Fetch page 1 with literal `page = 1` and literal `limit = 50`.\n" +
		"- Pass literal page `2` and limit `50` to the list operation.\n" +
		"- Resolve Toronto to coordinates before fetching weather.\n")
	want := map[string]bool{
		"page=1:page_1":           true,
		"limit=50:page_1":         true,
		"page=2:list":             true,
		"limit=50:list":           true,
		"q=Toronto,CA:coordinate": true,
	}
	for _, hint := range policy.BindingHints {
		delete(want, hint.Field+"="+hint.From+":"+hint.StepSelector)
	}
	if len(want) != 0 {
		t.Fatalf("missing literal hints: %#v from %#v", want, policy.BindingHints)
	}
}

func TestLintProjectMarkdownPassesForEvalCorpus(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "examples", "eval", "*", "project.md"))
	if err != nil {
		t.Fatalf("glob eval projects: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("expected eval project fixtures")
	}
	for _, path := range paths {
		t.Run(filepath.Base(filepath.Dir(path)), func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			for _, check := range LintProjectMarkdown(string(data)) {
				if check.Status == "fail" {
					t.Fatalf("lint failed %s: %#v", path, check)
				}
			}
		})
	}
}

func TestLintProjectMarkdownReportsMissingGoal(t *testing.T) {
	checks := LintProjectMarkdown("# Missing Goal\n\n" +
		"## External Systems and OpenAPI\n\n" +
		"OpenAPI: none required\n\n" +
		"## Runtime Policy\n\n" +
		"- Allowed runtimes: `openapi`, `http`, `fnct`.\n\n" +
		"## Safety and Approval Boundary\n\n" +
		"- Generate and validate artifacts only.\n\n" +
		"## Fallback Behavior\n\n" +
		"- Stop if required runtime capabilities are missing.\n")
	if !hasQualityCheck(checks, "project.authoring.goal", "warn") {
		t.Fatalf("expected missing goal warning, got %#v", checks)
	}
}

func TestLintProjectMarkdownFailsOnSecretLikeContent(t *testing.T) {
	checks := LintProjectMarkdown("# Secret\n\n" +
		"## Goal\n\n" +
		"Call an API.\n\n" +
		"## External Systems and OpenAPI\n\n" +
		"- API: use `openapi/api.yaml`.\n\n" +
		"## Runtime Policy\n\n" +
		"- Allowed runtimes: `openapi`, `http`, `fnct`.\n\n" +
		"## Credentials and Secrets\n\n" +
		"- api_key = \"AKIA1234567890ABCDEF\"\n\n" +
		"## Safety and Approval Boundary\n\n" +
		"- Generate and validate artifacts only.\n\n" +
		"## Fallback Behavior\n\n" +
		"- Stop if credentials are missing.\n")
	if !hasQualityCheck(checks, "project.no_secrets", "fail") {
		t.Fatalf("expected secret lint failure, got %#v", checks)
	}
}

func hasQualityCheck(checks []QualityCheck, code, status string) bool {
	for _, check := range checks {
		if check.Code == code && check.Status == status {
			return true
		}
	}
	return false
}
