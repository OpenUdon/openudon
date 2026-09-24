package browserscenario

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
