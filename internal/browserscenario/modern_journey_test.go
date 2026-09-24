package browserscenario

import (
	"encoding/json"
	"os"
	"path/filepath"
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
