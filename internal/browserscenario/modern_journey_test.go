package browserscenario

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

func TestCurrentScenarioCorpusAndModernSynthesis(t *testing.T) {
	at := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	historical, err := LoadManifests(at)
	if err != nil {
		t.Fatal(err)
	}
	current, err := LoadCurrentManifests(at)
	if err != nil {
		t.Fatal(err)
	}
	historicalJourneys, _ := SelectManifests(historical, SuiteJourney, nil)
	currentJourneys, _ := SelectManifests(current, SuiteJourney, nil)
	if len(historicalJourneys) != 8 || len(currentJourneys) != 11 {
		t.Fatalf("journey inventory = historical %d current %d", len(historicalJourneys), len(currentJourneys))
	}
	for _, kind := range []string{"template_browser18", "template_browser19", "mixed_legacy_modern"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			capability, authentication, blueprint, err := stageModernJourney(root, "http://127.0.0.1:12345", kind, at)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(capability); err != nil {
				t.Fatal(err)
			}
			if kind == "template_browser18" {
				data, err := os.ReadFile(capability)
				if err != nil {
					t.Fatal(err)
				}
				parsed, err := profile.ParseJSON(data)
				if err != nil {
					t.Fatal(err)
				}
				properties := parsed.Actions["read_wide"].Parameters["properties"].(map[string]any)
				value := properties["id"].(map[string]any)["default"]
				if value != json.Number("9223372036854775807") || len(blueprint.inputs) != 1 {
					t.Fatalf("wide profile default or input binding was rounded: %v", value)
				}
			}
			result, err := synthesize.WriteBrowserScenarioWorkflow(synthesize.BrowserScenarioWorkflowRequest{
				ExampleDir: root, AuthenticationPath: authentication, CapabilityPath: capability,
				AuthenticationFlow: journeyAuthenticationFlow, Session: journeySession,
				CredentialSlotBindings: map[string]string{}, Inputs: blueprint.inputs, Actions: blueprint.workflow,
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.UWSVersion != "1.11.0" || filepath.Dir(result.Path) != root {
				t.Fatalf("workflow = %#v", result)
			}
		})
	}
}

func TestCurrentV4CampaignCountProfilesAndFixtures(t *testing.T) {
	at := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	v3, err := LoadCurrentManifests(at)
	if err != nil {
		t.Fatal(err)
	}
	v4, err := LoadCurrentManifestsV4(at)
	if err != nil {
		t.Fatal(err)
	}
	oldJourneys, _ := SelectManifests(v3, SuiteJourney, nil)
	newJourneys, _ := SelectManifests(v4, SuiteJourney, nil)
	if len(oldJourneys) != 11 || len(newJourneys) != 14 {
		t.Fatalf("v3/v4 journey inventories = %d/%d", len(oldJourneys), len(newJourneys))
	}
	wantCounts := map[string]int{
		"campaign_count_browser110_zero":     0,
		"campaign_count_browser110_one":      1,
		"campaign_count_browser110_multiple": 3,
	}
	for kind, want := range wantCounts {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			capability, authentication, blueprint, err := stageModernJourney(root, "http://127.0.0.1:12345", kind, at)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(capability)
			if err != nil {
				t.Fatal(err)
			}
			profileValue, err := profile.ParseJSON(data)
			if err != nil {
				t.Fatal(err)
			}
			if profileValue.Schema != profile.SchemaV110 || filepath.Base(authentication) != "journey.json" {
				t.Fatalf("Browser 1.10 profile or authentication = %s / %s", profileValue.Schema, authentication)
			}
			action := profileValue.Actions["count_campaign_rows"]
			if len(action.Outputs) != 1 {
				t.Fatalf("count outputs = %#v", action.Outputs)
			}
			count := action.Outputs["campaign_count"]
			if count.Type != profile.OutputInteger || count.Source != profile.OutputCSS || !count.MatchCount || count.Selector != ".campaign-row" || count.Within != "#campaign-rows" || count.Visibility != profile.OutputVisibilityRendered || count.Validation["minimum"] != json.Number("0") || count.Validation["maximum"] != json.Number("100") {
				t.Fatalf("count declaration = %#v", count)
			}
			if !scenarioOutputsEqual(blueprint.expectedOutputs, map[string]any{"campaign_count": want}) {
				t.Fatalf("expected outputs = %#v", blueprint.expectedOutputs)
			}
			workflow, err := synthesize.WriteBrowserScenarioWorkflow(synthesize.BrowserScenarioWorkflowRequest{
				ExampleDir: root, AuthenticationPath: authentication, CapabilityPath: capability,
				AuthenticationFlow: journeyAuthenticationFlow, Session: journeySession,
				CredentialSlotBindings: map[string]string{}, Actions: blueprint.workflow,
			})
			if err != nil || workflow.UWSVersion != "1.11.0" {
				t.Fatalf("workflow = %#v, err = %v", workflow, err)
			}

			manifest := Manifest{Suite: SuiteJourney, Journey: &Journey{Kind: kind}}
			fixture, err := NewJourneyFixture(manifest)
			if err != nil {
				t.Fatal(err)
			}
			defer fixture.Close()
			response, err := http.Get(fixture.Origin() + "/topics")
			if err != nil {
				t.Fatal(err)
			}
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil || response.StatusCode != http.StatusOK || strings.Count(string(body), "Synthetic campaign row") != want || !strings.Contains(string(body), `class="campaign-row" hidden`) || !strings.Contains(string(body), "outside synthetic row") {
				t.Fatalf("fixture page for %s had the wrong rows: status=%d err=%v", kind, response.StatusCode, readErr)
			}
		})
	}
}
