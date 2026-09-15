package browserscenario

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot"
)

const authoringFailureDiagnosticVersion = "openudon.browser-scenario-authoring-diagnostic.v1"
const authoringFailureDiagnosticLimit = 64 << 10

type scenarioAuthoringFailure struct {
	Scenario  string                               `json:"scenario"`
	Authoring icot.BrowserScenarioAuthorDiagnostic `json:"authoring"`
}

type authoringFailureDiagnostic struct {
	Version      string                     `json:"version"`
	ReportSHA256 string                     `json:"report_sha256"`
	Failures     []scenarioAuthoringFailure `json:"failures"`
}

func appendAuthoringFailure(result ScenarioResult, cause error) ScenarioResult {
	diagnostic, ok := icot.BrowserScenarioFailureDiagnostic(cause)
	if !ok {
		diagnostic = icot.BrowserScenarioAuthorDiagnostic{Phase: "unknown", Code: "operation_failed"}
	}
	result.AuthoringDiagnostic = &diagnostic
	return appendFailure(result, "authoring_v2", "authoring_failed")
}

// AuthoringFailureDiagnostic preserves closed metadata before fixture cleanup
// and binds it to the compact JSON encoding of the unchanged scenario report.
// No diagnostic is produced for successful or non-authoring outcomes.
func AuthoringFailureDiagnostic(report *Report) ([]byte, error) {
	if report == nil {
		return nil, nil
	}
	record := authoringFailureDiagnostic{Version: authoringFailureDiagnosticVersion}
	for _, scenario := range report.Scenarios {
		if scenario.AuthoringDiagnostic != nil {
			record.Failures = append(record.Failures, scenarioAuthoringFailure{scenario.ID, *scenario.AuthoringDiagnostic})
		}
	}
	if len(record.Failures) == 0 {
		return nil, nil
	}
	sort.Slice(record.Failures, func(i, j int) bool { return record.Failures[i].Scenario < record.Failures[j].Scenario })
	data, err := json.Marshal(report)
	if err != nil {
		return nil, errors.New("scenario_diagnostic_invalid")
	}
	sum := sha256.Sum256(data)
	record.ReportSHA256 = hex.EncodeToString(sum[:])
	encoded, err := json.Marshal(record)
	if err != nil || ValidateAuthoringFailureDiagnostic(encoded, report) != nil {
		return nil, errors.New("scenario_diagnostic_invalid")
	}
	return encoded, nil
}

func ValidateAuthoringFailureDiagnostic(data []byte, report *Report) error {
	invalid := errors.New("scenario_diagnostic_invalid")
	if len(data) > authoringFailureDiagnosticLimit || report == nil || report.Status != StatusFail || ValidateLocalQualificationReport(report) != nil {
		return invalid
	}
	var record authoringFailureDiagnostic
	if evidencefile.DecodeStrict(data, &record) != nil || record.Version != authoringFailureDiagnosticVersion || len(record.Failures) == 0 || len(record.Failures) > len(report.Scenarios) {
		return invalid
	}
	encoded, err := json.Marshal(report)
	sum := sha256.Sum256(encoded)
	if err != nil || record.ReportSHA256 != hex.EncodeToString(sum[:]) {
		return invalid
	}
	eligible := make(map[string]bool)
	for _, scenario := range report.Scenarios {
		if scenario.Status != StatusFail || scenario.Detail != "authoring_failed" {
			continue
		}
		for _, phase := range scenario.Phases {
			if phase.ID == "authoring_v2" && phase.Status == StatusFail && phase.Detail == "authoring_failed" {
				eligible[scenario.ID] = true
			}
		}
	}
	previous := ""
	for _, failure := range record.Failures {
		if failure.Scenario <= previous || !eligible[failure.Scenario] || !failure.Authoring.Valid() {
			return invalid
		}
		previous = failure.Scenario
	}
	return nil
}

func writeAuthoringFailureDiagnostic(reportPath string, report *Report) error {
	data, err := AuthoringFailureDiagnostic(report)
	if err != nil || len(data) == 0 {
		return err
	}
	file, err := os.OpenFile(reportPath+".authoring-diagnostic.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("scenario_diagnostic_output")
	}
	_, writeErr := file.Write(append(data, '\n'))
	if errors.Join(writeErr, file.Sync(), file.Close()) != nil {
		return errors.New("scenario_diagnostic_output")
	}
	return nil
}
