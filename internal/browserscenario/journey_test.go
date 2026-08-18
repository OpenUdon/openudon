package browserscenario

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/guide"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

func TestJourneyGuidedBundlesAuthorAndImportStrictly(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, time.UTC)
	manifests, err := LoadManifests(now)
	if err != nil {
		t.Fatal(err)
	}
	journeys, err := SelectManifests(manifests, SuiteJourney, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, manifest := range journeys {
		t.Run(manifest.ID, func(t *testing.T) {
			blueprint, err := journeyScenarioBlueprint(manifest, "http://127.0.0.1:54321")
			if err != nil {
				t.Fatal(err)
			}
			data, err := buildJourneyBundle(blueprint.actions, "http://127.0.0.1:54321", now)
			if err != nil {
				t.Fatal(err)
			}
			var bundle guide.Bundle
			if err := json.Unmarshal(data, &bundle); err != nil {
				t.Fatal(err)
			}
			if bundle.Profile.Schema != "uws.browser.1.5" || len(bundle.Profile.Actions) != len(blueprint.actions) {
				t.Fatalf("guided bundle profile = %q with %d actions", bundle.Profile.Schema, len(bundle.Profile.Actions))
			}
			root := t.TempDir()
			exampleDir := filepath.Join(root, "example")
			if err := os.Mkdir(exampleDir, 0o700); err != nil {
				t.Fatal(err)
			}
			bundlePath := filepath.Join(root, "bundle.json")
			authenticationPath := filepath.Join(root, "authentication.json")
			if err := os.WriteFile(bundlePath, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := writeJourneyAuthentication(authenticationPath, "http://127.0.0.1:54321", now); err != nil {
				t.Fatal(err)
			}
			capabilityPath, importedAuthenticationPath, err := stageJourneySources(context.Background(), exampleDir, bundlePath, authenticationPath, now)
			if err != nil {
				t.Fatal(err)
			}
			if !regularFile(capabilityPath) || !regularFile(importedAuthenticationPath) {
				t.Fatal("strict import did not materialize both canonical profiles")
			}
			workflow, err := synthesize.WriteBrowserScenarioWorkflow(synthesize.BrowserScenarioWorkflowRequest{
				ExampleDir: exampleDir, AuthenticationPath: importedAuthenticationPath, CapabilityPath: capabilityPath,
				AuthenticationFlow: journeyAuthenticationFlow, Session: journeySession,
				CredentialSlotBindings: map[string]string{}, Inputs: blueprint.inputs, Actions: blueprint.workflow,
			})
			if err != nil {
				t.Fatal(err)
			}
			if workflow.UWSVersion != manifest.Expected.UWSVersion {
				t.Fatalf("workflow UWS = %s, want %s", workflow.UWSVersion, manifest.Expected.UWSVersion)
			}
		})
	}
}
