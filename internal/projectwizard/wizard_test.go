package projectwizard

import (
	"strings"
	"testing"
)

func TestRenderAPIBackedProjectIncludesOpenAPIAndHeadings(t *testing.T) {
	doc := Render(Answers{
		ProjectName:       "Support Ticket Draft",
		Goal:              "Fetch ticket details and write a draft reply",
		Inputs:            "`ticket_id`: required string",
		Outputs:           "Stored draft reply",
		DataFlow:          "Pass `get_ticket.received_body` to `write_draft.ticket`",
		FunctionContracts: "`write_draft`: inputs ticket body; outputs draft id; side effects writes draft only",
		UsesOpenAPI:       true,
		OpenAPI:           "Support API: use `openapi/support.yaml`",
		Credentials:       []string{"support_api_token"},
		Safety:            "Use sandbox endpoints for proof runs",
		Fallback:          "Stop if the Support API OpenAPI document is missing",
	})
	for _, expected := range []string{
		"# Support Ticket Draft",
		"## Goal",
		"## Inputs",
		"## Outputs",
		"## Data Flow",
		"## Function Contracts",
		"## External Systems and OpenAPI",
		"openapi/support.yaml",
		"## Runtime Policy",
		"## Credentials and Secrets",
		"## Safety and Approval Boundary",
		"## Fallback Behavior",
	} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("rendered project missing %q:\n%s", expected, doc)
		}
	}
}

func TestRenderRuntimeOnlyProjectDeclaresNoOpenAPI(t *testing.T) {
	doc := Render(Answers{
		ProjectName: "Runtime Only",
		Goal:        "Render a local report",
		UsesOpenAPI: false,
	})
	if !strings.Contains(doc, "OpenAPI: none required") {
		t.Fatalf("runtime-only project missing exact OpenAPI policy:\n%s", doc)
	}
}

func TestRenderEmptyOptionalAnswersAreExplicit(t *testing.T) {
	doc := Render(Answers{ProjectName: "Sparse", Goal: "Do the work"})
	for _, expected := range []string{
		"- none declared.",
		"- Credential bindings: none declared.",
		"- Stop if required OpenAPI documents, runtime capabilities, or credential bindings are missing.",
	} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("rendered project missing explicit empty/default text %q:\n%s", expected, doc)
		}
	}
}

func TestCredentialAnswersAreBindingNamesOnly(t *testing.T) {
	doc := Render(Answers{
		ProjectName: "Credentials",
		Goal:        "Use a secured API",
		UsesOpenAPI: true,
		OpenAPI:     "Secured API: use `openapi/secured.yaml`",
		Credentials: credentialBindings("support_api_token=supersecret, Billing API: billing_api_key"),
	})
	for _, expected := range []string{
		"`support_api_token`",
		"`billing_api_key`",
	} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("rendered project missing credential binding %q:\n%s", expected, doc)
		}
	}
	if strings.Contains(doc, "supersecret") {
		t.Fatalf("rendered project leaked credential value:\n%s", doc)
	}
}

func TestCmdAndSSHRemainDisallowedByDefault(t *testing.T) {
	doc := Render(Answers{ProjectName: "Defaults", Goal: "Do the work"})
	for _, expected := range []string{
		"`cmd` is not allowed",
		"`ssh` is not allowed",
	} {
		if !strings.Contains(doc, expected) {
			t.Fatalf("rendered project missing runtime default %q:\n%s", expected, doc)
		}
	}
}

func TestRunPromptsScriptedAnswers(t *testing.T) {
	input := strings.Join([]string{
		"Support",
		"Fetch a ticket",
		"`ticket_id`: required string",
		"Stored draft",
		"Pass ticket body to draft writer",
		"`write_draft`: ticket in, draft out",
		"yes",
		"`openapi/support.yaml`",
		"",
		"",
		"support_api_token",
		"Sandbox proof runs only",
		"Stop if ticket lookup is unavailable",
	}, "\n") + "\n"
	var prompts strings.Builder
	doc, err := Run(strings.NewReader(input), &prompts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !strings.Contains(prompts.String(), "Project name: ") {
		t.Fatalf("prompts missing project name prompt:\n%s", prompts.String())
	}
	if !strings.Contains(doc, "`openapi/support.yaml`") {
		t.Fatalf("rendered doc missing scripted OpenAPI answer:\n%s", doc)
	}
	if !strings.Contains(doc, "`cmd` is not allowed") || !strings.Contains(doc, "`ssh` is not allowed") {
		t.Fatalf("rendered doc did not keep default runtime denies:\n%s", doc)
	}
}
