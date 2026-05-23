package elicitor

import (
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestSessionValidateReportsMissingSlots(t *testing.T) {
	var session Session
	err := session.Validate()
	if err == nil || !strings.Contains(err.Error(), "workflow name") {
		t.Fatalf("Validate error = %v", err)
	}
}

func TestSessionValidateRejectsDuplicateStepNames(t *testing.T) {
	session := supportTicketDraft(true)
	session.Intent.Steps = append(session.Intent.Steps, &rollout.Step{
		Name:      session.Intent.Steps[0].Name,
		Type:      "http",
		Do:        "Fetch the ticket again.",
		Operation: "getTicket",
		With:      map[string]string{"ticketId": "inputs.ticketId"},
	})

	err := session.Validate()
	if err == nil || !strings.Contains(err.Error(), "duplicate step name get_ticket") {
		t.Fatalf("Validate error = %v", err)
	}
}
