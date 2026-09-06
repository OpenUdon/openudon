package elicitor

import (
	"encoding/json"
	"strings"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// A browser action is selected before the next round chooses how to establish
// its session. That intermediate state must survive cloning and draft resume;
// it still cannot become an approved artifact.
func TestBrowserSessionDecisionCanRemainPendingInDraft(t *testing.T) {
	session := supportTicketDraft(true)
	session.BrowserRoute = "browser"
	session.BrowserSession = ""
	session.Normalize()
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := DecodeSession(data, ".json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RenderArtifacts(resumed); err == nil || !strings.Contains(err.Error(), "explicit runtime session posture") {
		t.Fatalf("unresolved session became renderable: %v", err)
	}
	resumed.BrowserSession = "invented-posture"
	data, _ = json.Marshal(resumed)
	if _, err := DecodeSession(data, ".json"); err == nil {
		t.Fatal("draft accepted an invalid session posture")
	}
}

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
