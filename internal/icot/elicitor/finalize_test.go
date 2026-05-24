package elicitor

import (
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestRenderArtifactsWeatherGmailAddsReportPlaceholderAndOrdersChain(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{
			Goal: "Get weather for Toronto, Canada, and Gmail me the report.",
		},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather_toronto_gmail", Description: "Get weather for Toronto, Canada, and Gmail me the report."},
			Inputs:   []*rollout.Input{{Name: "body", Type: "string", Required: true}},
			Steps: []*rollout.Step{
				{
					Name:      "gmail",
					Type:      "http",
					Provider:  "gmail",
					OpenAPI:   "openapi/gmail.yaml",
					Operation: "gmail_users_messages_send",
					Do:        "Send the weather report by Gmail.",
					DependsOn: []string{"openweathermap"},
					With:      map[string]string{"body": "inputs.body", "raw": "inputs.body"},
				},
				{
					Name:      "openweathermap",
					Type:      "http",
					Provider:  "openweathermap",
					OpenAPI:   "openapi/openweathermap.yaml",
					Operation: "getOpenWeatherMapOneCall3",
					Do:        "Fetch OpenWeatherMap weather.",
					DependsOn: []string{"geocode_openweathermap_location"},
				},
				{
					Name:      "geocode_openweathermap_location",
					Type:      "http",
					Provider:  "openweathermap",
					OpenAPI:   "openapi/openweathermap.yaml",
					Operation: "geocodeOpenWeatherMapLocationName",
					Do:        "Resolve Toronto to coordinates.",
				},
			},
			Outputs: []*rollout.Output{{Name: "result", From: "openweathermap.received_body"}},
		},
	}

	artifacts, err := RenderArtifacts(session)
	if err != nil {
		t.Fatalf("RenderArtifacts failed: %v", err)
	}
	if got := stepNames(artifacts.Session.Intent.Steps); strings.Join(got, ",") != "geocode_openweathermap_location,openweathermap,render_weather_report,gmail" {
		t.Fatalf("step order = %v\n%s", got, artifacts.IntentHCL)
	}
	render := stepByName(artifacts.Session.Intent.Steps, "render_weather_report")
	if render == nil || render.Type != "fnct" || render.Operation != "gmail.render_raw" || len(render.DependsOn) != 1 || render.DependsOn[0] != "openweathermap" {
		t.Fatalf("render step = %#v", render)
	}
	if got := render.With["input"]; got != "openweathermap.received_body" {
		t.Fatalf("render input = %q", got)
	}
	if got := render.With["to"]; got != "inputs.recipient_email" {
		t.Fatalf("render recipient = %q", got)
	}
	if render.With["subject"] == "" || render.With["body_template"] == "" {
		t.Fatalf("render message fields = %#v", render.With)
	}
	gmail := stepByName(artifacts.Session.Intent.Steps, "gmail")
	if gmail == nil || gmail.With["userId"] != "me" || gmail.With["raw"] != "render_weather_report.received_body" {
		t.Fatalf("gmail step = %#v", gmail)
	}
	if _, ok := gmail.With["body"]; ok {
		t.Fatalf("gmail body mapping was not removed: %#v", gmail.With)
	}
	if len(gmail.DependsOn) != 1 || gmail.DependsOn[0] != "render_weather_report" {
		t.Fatalf("gmail depends_on = %#v", gmail.DependsOn)
	}
	if !containsString(artifacts.Session.Credentials, "gmail_oauth_token") {
		t.Fatalf("missing gmail credential binding: %#v", artifacts.Session.Credentials)
	}
	if len(artifacts.Session.Intent.Inputs) != 1 || artifacts.Session.Intent.Inputs[0].Name != "recipient_email" {
		t.Fatalf("input reconciliation = %#v", artifacts.Session.Intent.Inputs)
	}
	if got := artifacts.Session.Intent.Outputs[0].From; got != "render_weather_report.received_body" {
		t.Fatalf("output source = %q", got)
	}
	for _, want := range []string{
		"# iCoT review warning (reviewable_fnct_placeholder)",
		"render_weather_report is a pure Gmail raw-message formatting helper",
		`step "render_weather_report"`,
		`operation = "gmail.render_raw"`,
		"- `render_weather_report`\n  - Purpose: Render a reviewable local weather report from the weather response before Gmail delivery.\n  - Function: `gmail.render_raw`.\n  - Inputs: body_template, input, subject, to.\n  - Outputs: received_body.\n  - Side effects: none.",
	} {
		rendered := artifacts.IntentHCL + "\n" + artifacts.ProjectMD
		if !strings.Contains(rendered, want) {
			t.Fatalf("rendered artifacts missing %q:\nintent:\n%s\nproject:\n%s", want, artifacts.IntentHCL, artifacts.ProjectMD)
		}
	}
	if strings.Index(artifacts.IntentHCL, `step "geocode_openweathermap_location"`) > strings.Index(artifacts.IntentHCL, `step "openweathermap"`) ||
		strings.Index(artifacts.IntentHCL, `step "openweathermap"`) > strings.Index(artifacts.IntentHCL, `step "render_weather_report"`) ||
		strings.Index(artifacts.IntentHCL, `step "render_weather_report"`) > strings.Index(artifacts.IntentHCL, `step "gmail"`) {
		t.Fatalf("intent HCL order is wrong:\n%s", artifacts.IntentHCL)
	}
}

func TestRenderArtifactsWeatherGmailDoesNotTreatGeocoderGoalTextAsGmailStep(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "get weather in toronto, canada, and send the report using Google Gmail"},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather_toronto_gmail", Description: "get weather in toronto, canada, and send the report using Google Gmail"},
			Steps: []*rollout.Step{
				{
					Name:      "geocode_openweathermap_location",
					Type:      "http",
					Provider:  "openweathermap",
					OpenAPI:   "openapi/openweathermap.yaml",
					Operation: "geocodeOpenWeatherMapLocationName",
					Do:        "Resolve toronto, canada, and send the report using Google Gmail to OpenWeatherMap coordinates.",
					DependsOn: []string{"render_weather_report"},
					With:      map[string]string{"q": "toronto, canada", "raw": "render_weather_report.received_body", "userId": "me"},
				},
				{
					Name:      "fetch_weather_toronto",
					Type:      "http",
					Provider:  "openweathermap",
					OpenAPI:   "openapi/openweathermap.yaml",
					Operation: "getOpenWeatherMapOneCall3",
					DependsOn: []string{"geocode_openweathermap_location"},
				},
				{
					Name:      "email_weather_report",
					Type:      "http",
					Provider:  "gmail",
					OpenAPI:   "google-discovery/gmail-discovery-v1.json",
					Operation: "gmail_users_messages_send",
					Do:        "Send the weather summary to the user via Gmail as a report.",
					DependsOn: []string{"fetch_weather_toronto"},
				},
			},
			Outputs: []*rollout.Output{{Name: "weather_report", From: "fetch_weather_toronto.received_body"}},
		},
	}

	artifacts, err := RenderArtifacts(session)
	if err != nil {
		t.Fatalf("RenderArtifacts failed: %v", err)
	}
	geocode := stepByName(artifacts.Session.Intent.Steps, "geocode_openweathermap_location")
	if geocode == nil {
		t.Fatalf("geocode step missing: %#v", artifacts.Session.Intent.Steps)
	}
	for _, field := range []string{"raw", "userId"} {
		if _, ok := geocode.With[field]; ok {
			t.Fatalf("geocode step was mutated with Gmail field %q: %#v", field, geocode)
		}
	}
	if len(geocode.DependsOn) != 0 {
		t.Fatalf("geocode should not depend on Gmail report renderer: %#v", geocode.DependsOn)
	}
	gmail := stepByName(artifacts.Session.Intent.Steps, "email_weather_report")
	if gmail == nil || gmail.With["raw"] != "render_weather_report.received_body" || gmail.With["userId"] != "me" {
		t.Fatalf("gmail step was not prepared for report delivery: %#v", gmail)
	}
	if !containsString(artifacts.Session.Credentials, "googleOAuth2") {
		t.Fatalf("missing Google Discovery Gmail credential binding: %#v", artifacts.Session.Credentials)
	}
}

func TestRenderArtifactsWeatherGmailDoesNotInsertReportPlaceholderForMultipleProducers(t *testing.T) {
	session := Session{
		Project: projectwizard.Answers{Goal: "Get weather for Toronto, compare weather for Ottawa, and Gmail me the report."},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "weather_compare", Description: "Get weather for Toronto, compare weather for Ottawa, and Gmail me the report."},
			Steps: []*rollout.Step{
				{Name: "get_toronto_weather", Type: "http", Provider: "openweathermap", OpenAPI: "openapi/openweathermap.yaml", Operation: "getOpenWeatherMapOneCall3"},
				{Name: "get_ottawa_weather", Type: "http", Provider: "openweathermap", OpenAPI: "openapi/openweathermap.yaml", Operation: "getOpenWeatherMapOneCall3"},
				{Name: "gmail", Type: "http", Provider: "gmail", OpenAPI: "openapi/gmail.yaml", Operation: "gmail_users_messages_send", With: map[string]string{"userId": "me", "raw": "inputs.body"}},
			},
			Inputs:  []*rollout.Input{{Name: "body", Type: "string", Required: true}},
			Outputs: []*rollout.Output{{Name: "sent_message", From: "gmail.received_body"}},
		},
	}

	artifacts, err := RenderArtifacts(session)
	if err != nil {
		t.Fatalf("RenderArtifacts failed: %v", err)
	}
	if stepByName(artifacts.Session.Intent.Steps, "render_weather_report") != nil {
		t.Fatalf("unexpected report placeholder:\n%s", artifacts.IntentHCL)
	}
	if strings.Contains(artifacts.IntentHCL, "reviewable_fnct_placeholder") {
		t.Fatalf("unexpected placeholder warning:\n%s", artifacts.IntentHCL)
	}
}

func TestRenderArtifactsWeatherGmailUsesWorkflowDescriptionWhenProjectGoalEmpty(t *testing.T) {
	session := Session{Intent: rollout.Intent{
		Workflow: &rollout.WorkflowMeta{Name: "weather_toronto_gmail", Description: "Get weather for Toronto, Canada, and Gmail me the report."},
		Steps: []*rollout.Step{
			{Name: "geocode_city", Type: "http", OpenAPI: "openapi/weather.yaml", Operation: "geocodeCity", With: map[string]string{"q": "Toronto,CA"}},
			{Name: "get_weather", Type: "http", OpenAPI: "openapi/weather.yaml", Operation: "getWeatherByLatLon", With: map[string]string{"lat": "geocode_city.received_body.lat", "lon": "geocode_city.received_body.lon"}},
			{Name: "send_gmail", Type: "http", OpenAPI: "openapi/gmail.yaml", Operation: "gmail_users_messages_send", With: map[string]string{"userId": "me", "raw": "get_weather.received_body"}},
		},
		Outputs: []*rollout.Output{{Name: "sent_message", From: "send_gmail.received_body"}},
	}}

	artifacts, err := RenderArtifacts(session)
	if err != nil {
		t.Fatalf("RenderArtifacts failed: %v", err)
	}
	if stepByName(artifacts.Session.Intent.Steps, "render_weather_report") == nil {
		t.Fatalf("missing report placeholder:\n%s", artifacts.IntentHCL)
	}
}

func stepNames(steps []*rollout.Step) []string {
	names := make([]string, 0, len(steps))
	for _, step := range steps {
		if step != nil {
			names = append(names, step.Name)
		}
	}
	return names
}
