package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/genelet/udon/pkg/rollout"
)

func TestXRD004OpenAPIFixtureCoverage(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "eval")

	cursor := parseReferenceIntent(t, root, "cursor-pagination-report")
	cursorStep := findStep(cursor, "list_events_second_page")
	if cursorStep == nil {
		t.Fatal("cursor fixture missing second page step")
	}
	if cursorStep.Operation != "listAuditEvents" {
		t.Fatalf("second page operation = %q, want listAuditEvents", cursorStep.Operation)
	}
	if got := bindSource(cursorStep, "list_events_first_page", "cursor"); got != "received_body.page.nextCursor" {
		t.Fatalf("cursor binding = %q, want received_body.page.nextCursor", got)
	}
	if got := cursorStep.With["Authorization"]; got != "audit_events_bearer_token" {
		t.Fatalf("cursor fixture bearer binding = %q", got)
	}

	chain := parseReferenceIntent(t, root, "order-fulfillment-chain")
	create := findStep(chain, "create_fulfillment_order")
	if create == nil {
		t.Fatal("fulfillment fixture missing create_fulfillment_order step")
	}
	if create.OpenAPI != "openapi/orders.yaml" || create.Operation != "createFulfillmentOrder" {
		t.Fatalf("create step target = %s %s", create.OpenAPI, create.Operation)
	}
	if got := bindSource(create, "get_customer", "shippingAddressId"); got != "received_body.defaultShippingAddress.id" {
		t.Fatalf("shipping address binding = %q", got)
	}
	if got := bindSource(create, "check_inventory", "warehouseId"); got != "received_body.preferredWarehouseId" {
		t.Fatalf("warehouse binding = %q", got)
	}
	if len(distinctStepOpenAPIs(chain)) < 3 {
		t.Fatalf("fulfillment fixture should use at least three step-local OpenAPI documents")
	}

	assertFixtureFileContains(t, root, "order-fulfillment-chain", filepath.Join("openapi", "orders.yaml"), "requestBody:", "post:", "securitySchemes:")
	assertFixtureFileContains(t, root, "cursor-pagination-report", filepath.Join("openapi", "audit-events.yaml"), "securitySchemes:", "nextCursor:")
}

func TestN8nSlackReducibilityFixtureCoverage(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "eval")

	fixtures := []struct {
		name          string
		openapi       string
		step          string
		operationID   string
		nodeType      string
		nodeResource  string
		nodeOperation string
	}{
		{"n8n-airtable-record-get", "openapi/airtable.json", "get_record", "getAirtableRecord", "n8n-nodes-base.airtable", "record", "get"},
		{"n8n-gmail-message-send", "openapi/gmail.json", "send_message", "sendMessage", "n8n-nodes-base.gmail", "message", "send"},
		{"n8n-google-drive-file-upload", "openapi/google_drive.json", "upload_file", "uploadFile", "n8n-nodes-base.googleDrive", "file", "upload"},
		{"n8n-hubspot-deal-list", "openapi/hubspot.json", "list_deals", "listDeals", "n8n-nodes-base.hubspot", "deal", "getAll"},
		{"n8n-jira-issue-get", "openapi/jira.json", "get_issue", "getIssue", "n8n-nodes-base.jira", "issue", "get"},
		{"n8n-openweathermap-current-weather", "openapi/openweathermap.json", "get_current_weather", "getOpenWeatherMapCurrentWeather", "n8n-nodes-base.openWeatherMap", "weather", "currentWeather"},
		{"n8n-pagerduty-user-get", "openapi/pagerduty.json", "get_user", "getUser", "n8n-nodes-base.pagerDuty", "user", "get"},
		{"n8n-slack-message-post", "openapi/slack.json", "post_message", "postMessage", "n8n-nodes-base.slack", "message", "post"},
		{"n8n-trello-list-get-all", "openapi/trello.json", "list_board_lists", "listTrelloBoardLists", "n8n-nodes-base.trello", "list", "getAll"},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			intent := parseReferenceIntent(t, root, fixture.name)
			if strings.TrimSpace(intent.OpenAPI) != fixture.openapi {
				t.Fatalf("%s openapi = %q, want %s", fixture.name, intent.OpenAPI, fixture.openapi)
			}
			step := findStep(intent, fixture.step)
			if step == nil {
				t.Fatalf("%s missing %s step", fixture.name, fixture.step)
			}
			if step.Type != "http" || step.Operation != fixture.operationID {
				t.Fatalf("%s = type %q operation %q, want http %s", fixture.step, step.Type, step.Operation, fixture.operationID)
			}

			meta := readN8nFixtureMetadata(t, filepath.Join(root, fixture.name, "reference", "n8n.json"))
			if meta.SelectedOpenAPI != fixture.openapi {
				t.Fatalf("%s metadata selected_openapi = %q, want %s", fixture.name, meta.SelectedOpenAPI, fixture.openapi)
			}
			if meta.OperationID != fixture.operationID {
				t.Fatalf("%s metadata operation_id = %q, want %s", fixture.name, meta.OperationID, fixture.operationID)
			}
			if meta.Node.Type != fixture.nodeType || meta.Node.Resource != fixture.nodeResource || meta.Node.Operation != fixture.nodeOperation {
				t.Fatalf("%s node metadata = %#v", fixture.name, meta.Node)
			}
			assertFixtureFileContains(t, root, fixture.name, fixture.openapi, `"`+fixture.operationID+`"`)
		})
	}
	assertFixtureFileContains(t, root, "n8n-slack-message-post", filepath.Join("openapi", "slack.json"), `"operationId": "postMessage"`, `"channel"`, `"text"`)
}

func TestITOpsTemplateInspiredFixtureCoverage(t *testing.T) {
	root := filepath.Join("..", "..", "examples", "eval")

	backup := parseReferenceIntent(t, root, "itops-workflow-backup-github")
	if findStep(backup, "get_workflow").Operation != "getWorkflow" ||
		findStep(backup, "get_existing_backup").Operation != "getContent" ||
		findStep(backup, "upsert_backup_file").Operation != "putContent" {
		t.Fatalf("workflow backup operations did not match expected n8n/GitHub chain")
	}
	if step := findStep(backup, "render_backup_file"); step == nil || step.Type != "fnct" {
		t.Fatalf("workflow backup render step = %#v, want fnct", step)
	}
	assertFixtureFileContains(t, root, "itops-workflow-backup-github", filepath.Join("reference", "source.json"), "1534-back-up-your-n8n-workflows-to-github")

	intake := parseReferenceIntent(t, root, "itops-slack-jira-issue-intake")
	if findStep(intake, "get_slack_message").Operation != "getSlackMessage" ||
		findStep(intake, "create_jira_issue").Operation != "createIssue" ||
		findStep(intake, "post_slack_confirmation").Operation != "postMessage" {
		t.Fatalf("Slack Jira intake operations did not match expected Slack/Jira chain")
	}
	if step := findStep(intake, "parse_issue_report"); step == nil || step.Type != "fnct" {
		t.Fatalf("Slack Jira intake parse step = %#v, want fnct", step)
	}
	assertFixtureFileContains(t, root, "itops-slack-jira-issue-intake", filepath.Join("reference", "source.json"), "8813-automated-slack-to-jira-issue-creation-with-attachments")

	incident := parseReferenceIntent(t, root, "itops-incident-response-archive")
	if findStep(incident, "create_jira_incident").Operation != "createIssue" ||
		findStep(incident, "post_slack_alert").Operation != "postMessage" ||
		findStep(incident, "upload_timeline_report").Operation != "uploadFile" {
		t.Fatalf("incident response operations did not match expected Jira/Slack/Drive chain")
	}
	for _, name := range []string{"format_slack_alert", "render_timeline_report"} {
		if step := findStep(incident, name); step == nil || step.Type != "fnct" {
			t.Fatalf("incident response %s step = %#v, want fnct", name, step)
		}
	}
	assertFixtureFileContains(t, root, "itops-incident-response-archive", filepath.Join("reference", "source.json"), "9826-automate-incident-response-with-jira-slack-google-sheets-and-drive")
}

type n8nFixtureMetadata struct {
	SelectedOpenAPI string `json:"selected_openapi"`
	OperationID     string `json:"operation_id"`
	Node            struct {
		Type      string `json:"type"`
		Resource  string `json:"resource"`
		Operation string `json:"operation"`
	} `json:"node"`
}

func readN8nFixtureMetadata(t *testing.T, path string) n8nFixtureMetadata {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read n8n fixture metadata: %v", err)
	}
	var meta n8nFixtureMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parse n8n fixture metadata: %v", err)
	}
	return meta
}

func parseReferenceIntent(t *testing.T, root, name string) *rollout.Intent {
	t.Helper()
	intent, err := rollout.ParseIntentFile(filepath.Join(root, name, "reference", "intent.hcl"))
	if err != nil {
		t.Fatalf("parse %s reference intent: %v", name, err)
	}
	return intent
}

func findStep(intent *rollout.Intent, name string) *rollout.Step {
	for _, step := range flattenSteps(intent) {
		if step != nil && step.Name == name {
			return step
		}
	}
	return nil
}

func flattenSteps(intent *rollout.Intent) []*rollout.Step {
	if intent == nil {
		return nil
	}
	var out []*rollout.Step
	var walk func([]*rollout.Step)
	walk = func(steps []*rollout.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			out = append(out, step)
			walk(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					walk(branch.Steps)
				}
			}
			if step.Default != nil {
				walk(step.Default.Steps)
			}
		}
	}
	walk(intent.Steps)
	return out
}

func bindSource(step *rollout.Step, from, target string) string {
	if step == nil {
		return ""
	}
	for _, bind := range step.Binds {
		if bind != nil && bind.From == from {
			return bind.Fields[target]
		}
	}
	return ""
}

func distinctStepOpenAPIs(intent *rollout.Intent) map[string]bool {
	out := map[string]bool{}
	for _, step := range flattenSteps(intent) {
		if step != nil && strings.TrimSpace(step.OpenAPI) != "" {
			out[strings.TrimSpace(step.OpenAPI)] = true
		}
	}
	return out
}

func assertFixtureFileContains(t *testing.T, root, fixture, rel string, needles ...string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, fixture, rel))
	if err != nil {
		t.Fatalf("read %s/%s: %v", fixture, rel, err)
	}
	text := string(data)
	for _, needle := range needles {
		if !strings.Contains(text, needle) {
			t.Fatalf("%s/%s missing %q", fixture, rel, needle)
		}
	}
}
