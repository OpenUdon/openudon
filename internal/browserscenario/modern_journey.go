package browserscenario

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/synthesize"
)

// The modern corpus uses reviewed, schema-checked local profiles. The older
// eight journeys continue to use Browsertools guided-authoring bundles.
func stageModernJourney(exampleDir, origin, kind string, at time.Time) (string, string, journeyBlueprint, error) {
	authPath := filepath.Join(exampleDir, "browser-authentication", "journey.json")
	profileDir := filepath.Join(exampleDir, "browser-profiles")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o700); err != nil {
		return "", "", journeyBlueprint{}, err
	}
	if err := os.MkdirAll(profileDir, 0o700); err != nil {
		return "", "", journeyBlueprint{}, err
	}
	if err := writeJourneyAuthentication(authPath, origin, at); err != nil {
		return "", "", journeyBlueprint{}, err
	}

	modernPath := filepath.Join(profileDir, "modern.json")
	legacyPath := filepath.Join(profileDir, "legacy.json")
	var modernVersion, modernName, target, marker string
	var values map[string]any
	inputs := []synthesize.BrowserScenarioInput{{Name: "id", Type: "integer", Required: true}, {Name: "tag", Type: "string", Required: true}}
	var wideDefault int64
	var actions []synthesize.BrowserScenarioAction
	switch kind {
	case "template_browser18":
		modernVersion, modernName = profile.SchemaV18, "read_wide"
		target, marker = "/template18/{{id}}?view={{tag}}", "Template 18 OK"
		wideDefault = int64(9223372036854775807)
		values = map[string]any{"tag": "a/b"}
		inputs = []synthesize.BrowserScenarioInput{{Name: "tag", Type: "string", Required: true}}
		actions = []synthesize.BrowserScenarioAction{{Name: "read_wide", Operation: modernName, With: map[string]string{"tag": "tag"}}}
	case "template_browser19":
		modernVersion, modernName = profile.SchemaV19, "preview_text"
		target, marker = "/template19/{{{{literal}}}}/{{id}}?tag={{tag}}", "Preview verified"
		values = map[string]any{"id": int64(9007199254740991), "tag": "a/b"}
		actions = []synthesize.BrowserScenarioAction{{Name: "preview_text", Operation: modernName, With: map[string]string{"id": "id", "tag": "tag"}}}
	case "mixed_legacy_modern":
		modernVersion, modernName = profile.SchemaV19, "read_mixed"
		target, marker = "/mixed/{{id}}?tag={{tag}}", "Mixed OK"
		values = map[string]any{"id": int64(9007199254740991), "tag": "a/b"}
		if err := writeModernProfile(legacyPath, profile.SchemaV15, origin, at, "open_workspace", "/workspace", "Run marker", false, false, 0); err != nil {
			return "", "", journeyBlueprint{}, err
		}
		actions = []synthesize.BrowserScenarioAction{
			{Name: "open_workspace", Operation: "open_workspace"},
			{Name: "read_mixed", Operation: modernName, Source: modernPath, With: map[string]string{"id": "id", "tag": "tag"}},
		}
	case "campaign_count_browser110_zero", "campaign_count_browser110_one", "campaign_count_browser110_multiple":
		modernVersion, modernName = profile.SchemaV110, "count_campaign_rows"
		target = "/topics"
		values = map[string]any{}
		inputs = nil
		actions = []synthesize.BrowserScenarioAction{{Name: "count_campaign_rows", Operation: modernName}}
	default:
		return "", "", journeyBlueprint{}, fmt.Errorf("unknown modern journey kind")
	}
	if err := writeModernProfile(modernPath, modernVersion, origin, at, modernName, target, marker, modernVersion != profile.SchemaV110, kind == "template_browser19", wideDefault); err != nil {
		return "", "", journeyBlueprint{}, err
	}
	capability := modernPath
	if kind == "mixed_legacy_modern" {
		capability = legacyPath
	}
	blueprint := journeyBlueprint{
		workflow:        actions,
		inputs:          inputs,
		values:          values,
		expectedOutputs: modernJourneyExpectedOutputs(kind, marker),
	}
	return capability, authPath, blueprint, nil
}

func modernJourneyExpectedOutputs(kind, marker string) map[string]any {
	switch kind {
	case "campaign_count_browser110_zero":
		return map[string]any{"campaign_count": 0}
	case "campaign_count_browser110_one":
		return map[string]any{"campaign_count": 1}
	case "campaign_count_browser110_multiple":
		return map[string]any{"campaign_count": 3}
	default:
		return map[string]any{"marker": marker}
	}
}

func writeModernProfile(path, version, origin string, at time.Time, actionName, target, marker string, parameters, textSink bool, wideDefault int64) error {
	sequence := []any{map[string]any{"navigate": target}}
	if textSink {
		sequence = append(sequence,
			map[string]any{"type_text": map[string]any{"locator": map[string]any{"role": "textbox", "name": "Preview text"}, "value": "{{{{draft}}}} {{tag}}"}},
			map[string]any{"click": map[string]any{"locator": map[string]any{"role": "button", "name": "Preview"}, "wait_for": map[string]any{"navigation": "domcontentloaded"}}},
		)
	}
	outputs := map[string]any{"marker": map[string]any{"type": "string", "source": "a11y", "locator": map[string]any{"role": "status", "name": marker}}}
	if version == profile.SchemaV110 {
		outputs = map[string]any{"campaign_count": map[string]any{
			"type": "integer", "source": "css", "selector": ".campaign-row", "within": "#campaign-rows",
			"fallbackReason": "no_a11y_region", "matchCount": true, "visibility": "rendered",
			"validation": map[string]any{"type": "integer", "minimum": 0, "maximum": 100},
		}}
	} else {
		sequence = append(sequence, map[string]any{"wait_for": map[string]any{"role": "status", "name": marker}})
	}
	action := map[string]any{
		"description": "Read one reviewed synthetic browser target.",
		"sequence":    sequence,
		"outputs":     outputs,
		"sideEffects": []string{"read_only"}, "confirmationPolicy": map[string]any{"required": false},
	}
	if parameters {
		id := map[string]any{"type": "integer"}
		required := []string{"id", "tag"}
		if wideDefault != 0 {
			id["default"] = wideDefault
			required = []string{"tag"}
		}
		action["parameters"] = map[string]any{"type": "object", "properties": map[string]any{"id": id, "tag": map[string]any{"type": "string"}}, "required": required}
	}
	value := map[string]any{
		"profile":         version,
		"info":            map[string]any{"title": "OpenUdon current journey", "origin": origin, "loginStateRequired": true},
		"observationKind": "accessibility_snapshot",
		"evidence":        map[string]any{"learnedAt": at.Format(time.RFC3339), "source": "reviewed_local_journey_fixture"},
		"confidence":      "high", "expiresAfter": "P14D",
		"verification": map[string]any{"lastVerifiedAt": at.Format(time.RFC3339), "successfulRuns": 1},
		"actions":      map[string]any{actionName: action},
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if _, err := profile.ParseJSON(data); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func (executor *realExecutor) executeModernJourney(ctx context.Context, manifest Manifest, environment Environment) ScenarioResult {
	result := ScenarioResult{ID: manifest.ID, Attempts: 1}
	fixture, err := NewJourneyFixture(manifest)
	if err != nil {
		return failedScenario(manifest, "fixture_ready", "fixture_failed")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "fixture_ready", Status: StatusPass, Detail: "ok"})
	caseRoot, err := os.MkdirTemp(executor.root, "modern-journey-")
	if err != nil {
		fixture.Close()
		return appendFailure(result, "profile_reviewed", "staging_failed")
	}
	fail := func(phase, detail string) ScenarioResult {
		return finishJourneyResult(appendFailure(result, phase, detail), fixture, caseRoot)
	}
	exampleDir := filepath.Join(caseRoot, "example")
	if err := os.Mkdir(exampleDir, 0o700); err != nil {
		return fail("profile_reviewed", "staging_failed")
	}
	capability, authentication, blueprint, err := stageModernJourney(exampleDir, fixture.Origin(), manifest.Journey.Kind, environment.Now)
	if err != nil {
		return fail("profile_reviewed", "profile_mismatch")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "profile_reviewed", Status: StatusPass, Detail: "ok"})
	workflow, err := synthesize.WriteBrowserScenarioWorkflow(synthesize.BrowserScenarioWorkflowRequest{
		ExampleDir: exampleDir, AuthenticationPath: authentication, CapabilityPath: capability,
		AuthenticationFlow: journeyAuthenticationFlow, Session: journeySession,
		CredentialSlotBindings: map[string]string{}, Inputs: blueprint.inputs, Actions: blueprint.workflow,
	})
	if err != nil || workflow.UWSVersion != manifest.Expected.UWSVersion {
		return fail("uws_synthesized", "profile_mismatch")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "uws_synthesized", Status: StatusPass, Detail: "ok"})
	last := blueprint.workflow[len(blueprint.workflow)-1].Name
	protocol := "v10"
	udonPhase := "udon_v10"
	if manifest.Expected.BrowserProfile == profile.SchemaV110 {
		protocol, udonPhase = "v11", "udon_v11"
	}
	replay := executor.runJourneyUdonWithProtocol(ctx, exampleDir, filepath.Join(exampleDir, "journey-data.hcl"), workflow.Path, blueprint.values, nil, last, protocol)
	if replay.failureCode != "" {
		if isBrowser110CountKind(manifest.Journey) {
			reportPath := filepath.Join(exampleDir, "execution-report.json")
			result.failureCategory = closedExecutionFailureCategory(reportPath)
			result.failureSummary = executionFailureSummary(reportPath)
		}
		return fail("browserdriver_replay", replay.failureCode)
	}
	if !scenarioOutputsEqual(replay.outputs, blueprint.expectedOutputs) {
		if isBrowser110CountKind(manifest.Journey) {
			result.failureCategory = "expected_output_mismatch"
			result.failureSummary = fmt.Sprintf("campaign_count=%v", replay.outputs["campaign_count"])
		}
		return fail("browserdriver_replay", "output_mismatch")
	}
	if manifest.Expected.BrowserProfile == profile.SchemaV110 {
		persisted, readErr := os.ReadFile(filepath.Join(exampleDir, "output", "udon.hcl"))
		if readErr != nil || !strings.Contains(string(persisted), "campaign_count = ") || strings.Contains(string(persisted), "synthetic private page text") {
			result.failureCategory = "persisted_output_contract"
			result.failureSummary = fmt.Sprintf("read_error=%t count_attribute=%t private_text_present=%t", readErr != nil, strings.Contains(string(persisted), "campaign_count = "), strings.Contains(string(persisted), "synthetic private page text"))
			return fail("browserdriver_replay", "output_mismatch")
		}
	}
	result.Phases = append(result.Phases, PhaseResult{ID: udonPhase, Status: StatusPass, Detail: "ok"}, PhaseResult{ID: "browserdriver_replay", Status: StatusPass, Detail: "ok"})
	if !validJourneyPostconditions(manifest, fixture) {
		return fail("postconditions", "contract_drift")
	}
	result.Phases = append(result.Phases, PhaseResult{ID: "postconditions", Status: StatusPass, Detail: "ok"})
	result.Status, result.Detail = StatusPass, "ok"
	udonAssertion := "udon_v10_replay"
	if manifest.Expected.BrowserProfile == profile.SchemaV110 {
		udonAssertion = "udon_v11_replay"
	}
	result.Assertions = []string{udonAssertion, "browserdriver_replay", "structured_outputs_exact", "private_material_absent"}
	switch manifest.Journey.Kind {
	case "template_browser18":
		result.Assertions = append(result.Assertions, "browser18_template")
	case "template_browser19":
		result.Assertions = append(result.Assertions, "browser19_template")
	case "mixed_legacy_modern":
		result.Assertions = append(result.Assertions, "mixed_profile_session", "browser19_template")
	case "campaign_count_browser110_zero", "campaign_count_browser110_one", "campaign_count_browser110_multiple":
		result.Assertions = append(result.Assertions, "browser110_count")
	}
	result.Assertions = canonicalAssertions(result.Assertions)
	return finishJourneyResult(result, fixture, caseRoot)
}
