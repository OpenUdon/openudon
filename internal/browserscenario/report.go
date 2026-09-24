package browserscenario

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot"
)

const (
	ReportVersion               = "openudon.browser-scenario-eval.v1"
	JourneyReportVersion        = "openudon.browser-journey-eval.v1"
	CurrentReportVersion        = "openudon.browser-scenario-eval.v2"
	CurrentJourneyReportVersion = "openudon.browser-journey-eval.v2"
	StatusPass                  = "pass"
	StatusFail                  = "fail"
	StatusNotRun                = "not_run"
	StatusSkipped               = "skipped"
	StatusQuarantined           = "quarantined"
	maxReportBytes              = 4 << 20
)

type Report struct {
	Version                  string               `json:"version"`
	Status                   string               `json:"status"`
	Suite                    string               `json:"suite"`
	GeneratedAt              string               `json:"generated_at"`
	Commit                   string               `json:"commit"`
	Command                  string               `json:"command"`
	Repositories             []RepositoryRevision `json:"repositories"`
	Dependencies             []DependencyRevision `json:"dependencies"`
	Engine                   string               `json:"engine"`
	HeadedAuthoring          bool                 `json:"headed_authoring"`
	ProviderFree             bool                 `json:"provider_free"`
	ExternalNetwork          bool                 `json:"external_network"`
	ContainsCredentialValues bool                 `json:"contains_credential_values"`
	ContainsPageContent      bool                 `json:"contains_page_content"`
	ContainsSubprocessOutput bool                 `json:"contains_subprocess_output"`
	SafeToArchive            bool                 `json:"safe_to_archive"`
	Summary                  Summary              `json:"summary"`
	Scenarios                []ScenarioResult     `json:"scenarios"`
}

type RepositoryRevision struct {
	Name   string `json:"name"`
	Commit string `json:"commit"`
	Dirty  bool   `json:"dirty"`
}

type DependencyRevision struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

type Summary struct {
	Total       int `json:"total"`
	Passed      int `json:"passed"`
	Failed      int `json:"failed"`
	Skipped     int `json:"skipped"`
	Quarantined int `json:"quarantined"`
}

type ScenarioResult struct {
	ID         string        `json:"id"`
	Status     string        `json:"status"`
	Attempts   int           `json:"attempts"`
	Phases     []PhaseResult `json:"phases"`
	Assertions []string      `json:"assertions"`
	Detail     string        `json:"detail"`
	// Private failure metadata travels separately; published v1 reports keep
	// their exact wire shape and cannot become evidence of successful authoring.
	AuthoringDiagnostic *icot.BrowserScenarioAuthorDiagnostic `json:"-"`
}

type PhaseResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func NewReport(suite string, generatedAt time.Time, repositories []RepositoryRevision, dependencies []DependencyRevision, results []ScenarioResult) *Report {
	return NewReportForStack(suite, StackHistorical, generatedAt, repositories, dependencies, results)
}

func NewReportForStack(suite, stack string, generatedAt time.Time, repositories []RepositoryRevision, dependencies []DependencyRevision, results []ScenarioResult) *Report {
	summary := summarizeScenarios(results)
	status := StatusPass
	if summary.Failed > 0 {
		status = StatusFail
	} else if summary.Passed == 0 {
		status = StatusNotRun
	}
	commit := ""
	if len(repositories) > 0 {
		commit = repositories[0].Commit
	}
	version := ReportVersion
	if suite == SuiteJourney {
		version = JourneyReportVersion
	}
	if stack == StackCurrent {
		version = CurrentReportVersion
		if suite == SuiteJourney {
			version = CurrentJourneyReportVersion
		}
	}
	return &Report{
		Version: version, Status: status, Suite: suite,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339), Commit: commit,
		Command: "openudon browser-scenario-eval", Repositories: cloneRepositories(repositories),
		Dependencies: cloneDependencies(dependencies), Engine: "chromium",
		HeadedAuthoring: suite == SuiteLoopback, ProviderFree: true,
		ExternalNetwork: suite == SuitePublic, SafeToArchive: true,
		Scenarios: cloneScenarioResults(results), Summary: summary,
	}
}

func ValidateReport(report *Report) error {
	if report == nil || (report.Status != StatusPass && report.Status != StatusFail && report.Status != StatusNotRun) ||
		(report.Suite != SuiteLoopback && report.Suite != SuiteJourney && report.Suite != SuitePublic) || report.Command != "openudon browser-scenario-eval" || report.Engine != "chromium" {
		return fmt.Errorf("browser scenario report identity is invalid")
	}
	stack := StackHistorical
	if report.Version == CurrentReportVersion || report.Version == CurrentJourneyReportVersion {
		stack = StackCurrent
	}
	if stack == StackCurrent && report.Suite == SuitePublic {
		return fmt.Errorf("public browser scenarios cannot use the current stack")
	}
	wantVersion := ReportVersion
	if report.Suite == SuiteJourney {
		wantVersion = JourneyReportVersion
	}
	if stack == StackCurrent {
		wantVersion = CurrentReportVersion
		if report.Suite == SuiteJourney {
			wantVersion = CurrentJourneyReportVersion
		}
	}
	if report.Version != wantVersion {
		return fmt.Errorf("browser scenario report version is invalid for its suite")
	}
	parsedAt, err := time.Parse(time.RFC3339, report.GeneratedAt)
	if err != nil || parsedAt.UTC().Format(time.RFC3339) != report.GeneratedAt || !commitPattern.MatchString(report.Commit) {
		return fmt.Errorf("browser scenario report provenance is invalid")
	}
	if report.HeadedAuthoring != (report.Suite == SuiteLoopback) || report.ExternalNetwork != (report.Suite == SuitePublic) || !report.ProviderFree ||
		report.ContainsCredentialValues || report.ContainsPageContent || report.ContainsSubprocessOutput || !report.SafeToArchive {
		return fmt.Errorf("browser scenario report safety claims are invalid")
	}
	if err := validateRepositoryRevisions(report.Repositories, report.Commit); err != nil {
		return err
	}
	if err := validateDependencyRevisions(report.Dependencies); err != nil {
		return err
	}
	if stack == StackCurrent {
		lock, err := LoadCurrentCompatibilityLock()
		if err != nil {
			return err
		}
		if err := validateCurrentReportRevisions(report, lock); err != nil {
			return err
		}
	}
	var manifests []Manifest
	if stack == StackCurrent {
		manifests, err = LoadCurrentManifests(parsedAt)
	} else {
		manifests, err = LoadManifests(parsedAt)
	}
	if err != nil {
		return err
	}
	validIDs := map[string]bool{}
	for _, manifest := range manifests {
		if manifest.Suite == report.Suite {
			validIDs[manifest.ID] = true
		}
	}
	seen := map[string]bool{}
	for _, result := range report.Scenarios {
		if !validIDs[result.ID] || seen[result.ID] || !allowedScenarioStatuses[result.Status] || result.Attempts < 0 || result.Attempts > 1 || !allowedDetails[result.Detail] {
			return fmt.Errorf("browser scenario result %q is invalid", result.ID)
		}
		seen[result.ID] = true
		if result.Status == StatusQuarantined {
			if report.Suite != SuitePublic || result.Attempts != 0 || result.Detail != "quarantined" || len(result.Phases) != 0 {
				return fmt.Errorf("browser scenario quarantine result is invalid")
			}
		} else if result.Attempts != 1 || len(result.Phases) == 0 {
			return fmt.Errorf("browser scenario result %q has no execution evidence", result.ID)
		}
		phaseSeen := map[string]bool{}
		for _, phase := range result.Phases {
			allowedPhase := allowedPhaseIDs[report.Suite][phase.ID] || (stack == StackCurrent && report.Suite == SuiteJourney && currentPhaseIDs[phase.ID])
			if !allowedPhase || phaseSeen[phase.ID] || !allowedPhaseStatuses[phase.Status] || !allowedDetails[phase.Detail] {
				return fmt.Errorf("browser scenario phase %q is invalid", phase.ID)
			}
			phaseSeen[phase.ID] = true
		}
		assertionSeen := map[string]bool{}
		for _, assertion := range result.Assertions {
			if (!allowedAssertions[assertion] && (stack != StackCurrent || !currentAssertions[assertion])) || assertionSeen[assertion] {
				return fmt.Errorf("browser scenario assertion %q is invalid", assertion)
			}
			assertionSeen[assertion] = true
		}
		if result.Status == StatusPass && len(result.Assertions) == 0 {
			return fmt.Errorf("passing browser scenario %q has no assertion evidence", result.ID)
		}
		if stack == StackCurrent && result.Status == StatusPass {
			if required := currentCaseAssertion[result.ID]; required != "" && (!assertionSeen[required] || !assertionSeen["udon_v10_replay"] || !phaseSeen["udon_v10"]) {
				return fmt.Errorf("current browser scenario %q has no v10/template evidence", result.ID)
			}
		}
	}
	expectedStatus := StatusPass
	if report.Summary.Failed > 0 {
		expectedStatus = StatusFail
	} else if report.Summary.Passed == 0 {
		expectedStatus = StatusNotRun
	}
	if len(report.Scenarios) == 0 || report.Summary != summarizeScenarios(report.Scenarios) || report.Status != expectedStatus {
		return fmt.Errorf("browser scenario report summary is invalid")
	}
	return nil
}

func WriteReport(filename string, report *Report) error {
	if err := ValidateReport(report); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if len(data) > maxReportBytes {
		return fmt.Errorf("browser scenario report exceeds size bound")
	}
	if err := atomicfile.Write(filename, data, 0o644); err != nil {
		return err
	}
	return evidencefile.WriteDigestSidecar(filename, data, 0o644)
}

func VerifyReportFile(filename string, requirePassing bool) (*Report, error) {
	data, err := readBoundedRegular(filename, maxReportBytes)
	if err != nil {
		return nil, err
	}
	digestData, err := readBoundedRegular(filename+".sha256", 512)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	want := "sha256:" + hex.EncodeToString(sum[:]) + "  " + filepath.Base(filename) + "\n"
	if string(digestData) != want {
		return nil, fmt.Errorf("browser scenario report digest mismatch")
	}
	var report Report
	if err := decodeStrict(data, &report); err != nil {
		return nil, fmt.Errorf("browser scenario report: %w", err)
	}
	if err := requireReportWireFields(data); err != nil {
		return nil, err
	}
	if err := ValidateReport(&report); err != nil {
		return nil, err
	}
	if requirePassing && report.Status != StatusPass {
		return &report, fmt.Errorf("browser scenario report status is %s", report.Status)
	}
	if requirePassing && (report.Version == CurrentReportVersion || report.Version == CurrentJourneyReportVersion) {
		generatedAt, err := time.Parse(time.RFC3339, report.GeneratedAt)
		if err != nil {
			return &report, err
		}
		manifests, err := LoadCurrentManifests(generatedAt)
		if err != nil {
			return &report, err
		}
		expected := map[string]bool{}
		for _, manifest := range manifests {
			if manifest.Suite == report.Suite {
				expected[manifest.ID] = true
			}
		}
		if len(report.Scenarios) != len(expected) {
			return &report, fmt.Errorf("current browser scenario report is not a complete suite")
		}
		for _, result := range report.Scenarios {
			if !expected[result.ID] || result.Status != StatusPass {
				return &report, fmt.Errorf("current browser scenario report is not a complete passing suite")
			}
		}
	}
	return &report, nil
}

func validateCurrentReportRevisions(report *Report, lock CompatibilityLock) error {
	states := map[string]RepositoryState{}
	for _, revision := range report.Repositories {
		if revision.Name != "openudon" {
			states[revision.Name] = RepositoryState{Commit: revision.Commit, Dirty: revision.Dirty}
		}
	}
	if err := ValidateRepositoryStates(lock, states); err != nil {
		return err
	}
	for _, component := range lock.Components {
		for _, dep := range report.Dependencies {
			if dep.Module == component.Module && dep.Version != component.Version {
				return fmt.Errorf("current browser scenario dependency differs from lock")
			}
		}
	}
	return nil
}

func requireReportWireFields(data []byte) error {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	fields := []string{"version", "status", "suite", "generated_at", "commit", "command", "repositories", "dependencies", "engine", "headed_authoring", "provider_free", "external_network", "contains_credential_values", "contains_page_content", "contains_subprocess_output", "safe_to_archive", "summary", "scenarios"}
	for _, field := range fields {
		if value, ok := root[field]; !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("browser scenario report requires field %q", field)
		}
	}
	return nil
}

func validateRepositoryRevisions(revisions []RepositoryRevision, rootCommit string) error {
	want := []string{"openudon", "browsertools", "uws", "udon", "browserdriver"}
	if len(revisions) != len(want) {
		return fmt.Errorf("browser scenario repository inventory is incomplete")
	}
	for index, revision := range revisions {
		if revision.Name != want[index] || !commitPattern.MatchString(revision.Commit) {
			return fmt.Errorf("browser scenario repository revision is invalid")
		}
		if revision.Dirty {
			return fmt.Errorf("browser scenario repository %s is dirty", revision.Name)
		}
	}
	if revisions[0].Commit != rootCommit {
		return fmt.Errorf("browser scenario root commit mismatch")
	}
	return nil
}

func validateDependencyRevisions(revisions []DependencyRevision) error {
	want := []string{"github.com/OpenUdon/browsertools", "github.com/OpenUdon/uws"}
	if len(revisions) != len(want) {
		return fmt.Errorf("browser scenario dependency inventory is incomplete")
	}
	for index, revision := range revisions {
		if revision.Module != want[index] || strings.TrimSpace(revision.Version) == "" {
			return fmt.Errorf("browser scenario dependency revision is invalid")
		}
	}
	return nil
}

func summarizeScenarios(results []ScenarioResult) Summary {
	summary := Summary{Total: len(results)}
	for _, result := range results {
		switch result.Status {
		case StatusPass:
			summary.Passed++
		case StatusFail:
			summary.Failed++
		case StatusSkipped:
			summary.Skipped++
		case StatusQuarantined:
			summary.Quarantined++
		}
	}
	return summary
}

func readBoundedRegular(filename string, limit int64) ([]byte, error) {
	data, _, err := evidencefile.ReadRegular(filename, limit)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("browser scenario evidence must be a bounded regular file: %w", err)
	}
	return data, nil
}

func cloneRepositories(values []RepositoryRevision) []RepositoryRevision {
	return append([]RepositoryRevision(nil), values...)
}

func cloneDependencies(values []DependencyRevision) []DependencyRevision {
	return append([]DependencyRevision(nil), values...)
}

func cloneScenarioResults(values []ScenarioResult) []ScenarioResult {
	result := make([]ScenarioResult, len(values))
	for index, value := range values {
		result[index] = value
		result[index].Phases = append([]PhaseResult(nil), value.Phases...)
		result[index].Assertions = append([]string(nil), value.Assertions...)
		if value.AuthoringDiagnostic != nil {
			copy := *value.AuthoringDiagnostic
			if copy.Failure != nil {
				detail := *copy.Failure
				copy.Failure = &detail
			}
			result[index].AuthoringDiagnostic = &copy
		}
	}
	return result
}

var commitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
var allowedScenarioStatuses = map[string]bool{StatusPass: true, StatusFail: true, StatusSkipped: true, StatusQuarantined: true}
var allowedPhaseStatuses = map[string]bool{StatusPass: true, StatusFail: true, StatusSkipped: true}
var allowedPhaseIDs = map[string]map[string]bool{
	SuiteLoopback: {"fixture_ready": true, "authoring_v2": true, "profiles_staged": true, "profile_versions": true, "udon_v3": true, "browserdriver_replay": true, "outputs_validated": true, "teardown": true},
	SuiteJourney:  {"fixture_ready": true, "bundle_authored": true, "profile_imported": true, "uws_synthesized": true, "udon_v3": true, "browserdriver_replay": true, "postconditions": true, "teardown": true},
	SuitePublic:   {"browsertools_probe": true, "udon_browserdriver_probe": true, "teardown": true},
}
var allowedAssertions = map[string]bool{
	"author_session_v2": true, "reviewed_mfa_kind": true, "reviewed_outputs": true,
	"profiles_reconstructed": true, "oldest_sufficient_versions": true, "udon_v3_lowering": true,
	"browserdriver_replay": true, "typed_outputs_exact": true, "expected_rejection": true,
	"browsertools_presence": true, "runtime_presence": true, "private_material_absent": true,
	"guided_authoring_v1": true, "canonical_profile_only": true, "closed_macro_vocabulary": true,
	"structured_outputs_exact": true, "operation_approval_exact": true, "server_state_exact": true,
	"parameter_contract_exact": true, "session_isolated": true,
}
var currentAssertions = map[string]bool{
	"browser18_template": true, "browser19_template": true,
	"udon_v10_replay": true, "mixed_profile_session": true,
}
var currentPhaseIDs = map[string]bool{"profile_reviewed": true, "udon_v10": true}
var currentCaseAssertion = map[string]string{
	"template-browser18":  "browser18_template",
	"template-browser19":  "browser19_template",
	"mixed-legacy-modern": "mixed_profile_session",
}
var allowedDetails = map[string]bool{
	"": true, "ok": true, "quarantined": true, "dependency_unavailable": true,
	"fixture_failed": true, "authoring_failed": true, "staging_failed": true,
	"profile_mismatch": true, "replay_failed": true, "output_mismatch": true,
	"timeout": true, "target_unreachable": true, "shape_drift": true,
	"origin_policy_drift": true, "contract_drift": true, "teardown_failed": true,
	"output_bound": true, "stale_candidate": true, "ambiguous_output": true,
	"invalid_context": true, "invalid_response": true, "secret_output": true,
	"origin_rejected": true, "authentication_not_proven": true,
	"approval_required": true, "ambiguous_locator": true, "invalid_parameters": true,
	"mfa_timeout": true, "mfa_denied": true, "credentials_invalid": true, "session_expired": true,
	"driver_error": true, "unsupported_challenge": true, "captcha_required": true, "unclassified": true,
}

func canonicalAssertions(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}

// ValidateLocalQualificationReport permits only the OpenUdon engineering delta.
// Its enclosing system report, not a legacy scenario file, owns the source
// digest. Every legacy publication verifier still requires clean repositories.
func ValidateLocalQualificationReport(report *Report) error {
	if report == nil {
		return fmt.Errorf("local scenario report is missing")
	}
	copy := *report
	copy.Repositories = append([]RepositoryRevision(nil), report.Repositories...)
	for i, revision := range copy.Repositories {
		if revision.Name == "openudon" {
			copy.Repositories[i].Dirty = false
		}
	}
	return ValidateReport(&copy)
}
