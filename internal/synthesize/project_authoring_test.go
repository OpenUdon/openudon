package synthesize

import "testing"

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
