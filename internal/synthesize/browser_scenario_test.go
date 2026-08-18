package synthesize

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/uws/uws1"
)

func TestWireBrowserScenarioOutputsUsesUWSStepContext(t *testing.T) {
	document := &uws1.Document{
		Operations: []*uws1.Operation{{OperationID: "authenticate"}, {OperationID: "read"}},
		Workflows: []*uws1.Workflow{{
			WorkflowID: "main",
			Steps: []*uws1.Step{
				{StepID: "authenticate", OperationRef: "authenticate"},
				{StepID: "read", OperationRef: "read"},
			},
		}},
	}

	if err := wireBrowserScenarioOutputs(document); err != nil {
		t.Fatal(err)
	}
	if got := document.Operations[1].Outputs["received_body"]; got != "$response.body" {
		t.Fatalf("operation output = %q", got)
	}
	if got := document.Workflows[0].Steps[1].Outputs["received_body"]; got != "$response.body" {
		t.Fatalf("step output = %q", got)
	}
	if got := document.Workflows[0].Outputs["scenario_result"]; got != "$steps.read.outputs.received_body" {
		t.Fatalf("workflow output = %q", got)
	}
}

func TestWriteBrowserScenarioWorkflowSupportsOrderedParameterizedActions(t *testing.T) {
	example := t.TempDir()
	authenticationPath := filepath.Join(example, "browser-authentication", "member.yaml")
	capabilityPath := filepath.Join(example, "browser-profiles", "member.json")
	mustWriteSynthesizeTestFile(t, authenticationPath, synthesizeBrowserAuthenticationFixture())
	mustWriteSynthesizeTestFile(t, capabilityPath, synthesizeBrowserProfileFixture(false, true, "item"))
	request := BrowserScenarioWorkflowRequest{
		ExampleDir: example, AuthenticationPath: authenticationPath, CapabilityPath: capabilityPath,
		AuthenticationFlow: "member_login_push", Session: "member_portal",
		CredentialSlotBindings: map[string]string{"username": "member_username", "password": "member_password"},
		Inputs:                 []BrowserScenarioInput{{Name: "item", Type: "string", Required: true}},
		Actions: []BrowserScenarioAction{
			{Name: "read_first", Operation: "read_status", With: map[string]string{"item": "item"}},
			{Name: "read_second", Operation: "read_status", With: map[string]string{"item": "item"}},
		},
	}
	result, err := WriteBrowserScenarioWorkflow(request)
	if err != nil {
		t.Fatal(err)
	}
	if result.UWSVersion != "1.7.0" {
		t.Fatalf("UWS version = %q", result.UWSVersion)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	var document uws1.Document
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Workflows) != 1 || len(document.Workflows[0].Steps) != 3 || document.Workflows[0].Outputs["scenario_result"] != "$steps.read_second.outputs.received_body" {
		t.Fatalf("ordered browser workflow = %#v", document.Workflows)
	}
	body, _ := document.Operations[1].Request["body"].(map[string]any)
	item, _ := body["item"].(map[string]any)
	if item["$expr"] != "variables.inputs.item" {
		t.Fatalf("browser parameter binding = %#v", document.Operations[1].Request)
	}

	request.Actions[0].With = map[string]string{"item": "item", "unexpected": "item"}
	if _, err := WriteBrowserScenarioWorkflow(request); err == nil || !strings.Contains(err.Error(), "not declared") {
		t.Fatalf("undeclared browser parameter = %v", err)
	}
}

func TestWireBrowserScenarioOutputsRejectsMissingReadStep(t *testing.T) {
	document := &uws1.Document{
		Operations: []*uws1.Operation{{OperationID: "read"}},
		Workflows:  []*uws1.Workflow{{WorkflowID: "main"}},
	}
	if err := wireBrowserScenarioOutputs(document); err == nil {
		t.Fatal("missing read step was accepted")
	}
}
