package browserscenario

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ReportVersion     = "openudon.browser-scenario-eval.v1"
	StatusPass        = "pass"
	StatusFail        = "fail"
	StatusSkipped     = "skipped"
	StatusQuarantined = "quarantined"
	maxReportBytes    = 4 << 20
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
}

type PhaseResult struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func NewReport(suite string, generatedAt time.Time, repositories []RepositoryRevision, dependencies []DependencyRevision, results []ScenarioResult) *Report {
	status := StatusPass
	for _, result := range results {
		if result.Status == StatusFail {
			status = StatusFail
		}
	}
	commit := ""
	if len(repositories) > 0 {
		commit = repositories[0].Commit
	}
	return &Report{
		Version: ReportVersion, Status: status, Suite: suite,
		GeneratedAt: generatedAt.UTC().Format(time.RFC3339), Commit: commit,
		Command: "openudon browser-scenario-eval", Repositories: cloneRepositories(repositories),
		Dependencies: cloneDependencies(dependencies), Engine: "chromium",
		HeadedAuthoring: suite == SuiteLoopback, ProviderFree: true,
		ExternalNetwork: suite == SuitePublic, SafeToArchive: true,
		Scenarios: cloneScenarioResults(results), Summary: summarizeScenarios(results),
	}
}

func ValidateReport(report *Report) error {
	if report == nil || report.Version != ReportVersion || (report.Status != StatusPass && report.Status != StatusFail) ||
		(report.Suite != SuiteLoopback && report.Suite != SuitePublic) || report.Command != "openudon browser-scenario-eval" || report.Engine != "chromium" {
		return fmt.Errorf("browser scenario report identity is invalid")
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
	manifests, err := LoadManifests(parsedAt)
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
			if !allowedPhaseIDs[report.Suite][phase.ID] || phaseSeen[phase.ID] || !allowedPhaseStatuses[phase.Status] || !allowedDetails[phase.Detail] {
				return fmt.Errorf("browser scenario phase %q is invalid", phase.ID)
			}
			phaseSeen[phase.ID] = true
		}
		assertionSeen := map[string]bool{}
		for _, assertion := range result.Assertions {
			if !allowedAssertions[assertion] || assertionSeen[assertion] {
				return fmt.Errorf("browser scenario assertion %q is invalid", assertion)
			}
			assertionSeen[assertion] = true
		}
		if result.Status == StatusPass && len(result.Assertions) == 0 {
			return fmt.Errorf("passing browser scenario %q has no assertion evidence", result.ID)
		}
	}
	if len(report.Scenarios) == 0 || report.Summary != summarizeScenarios(report.Scenarios) || (report.Status == StatusPass) != (report.Summary.Failed == 0) {
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
	if err := writeAtomic(filename, data, 0o644); err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:]) + "  " + filepath.Base(filename) + "\n"
	return writeAtomic(filename+".sha256", []byte(digest), 0o644)
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
	return &report, nil
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
	want := []string{"openudon", "browsertools", "udon", "browserdriver"}
	if len(revisions) != len(want) {
		return fmt.Errorf("browser scenario repository inventory is incomplete")
	}
	for index, revision := range revisions {
		if revision.Name != want[index] || !commitPattern.MatchString(revision.Commit) {
			return fmt.Errorf("browser scenario repository revision is invalid")
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
	info, err := os.Lstat(filename)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > limit {
		return nil, fmt.Errorf("browser scenario evidence must be a bounded regular file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, fmt.Errorf("read browser scenario evidence")
	}
	after, err := file.Stat()
	if err != nil || !os.SameFile(info, after) || int64(len(data)) != after.Size() {
		return nil, fmt.Errorf("browser scenario evidence changed while reading")
	}
	return data, nil
}

func writeAtomic(filename string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, ".browser-scenario-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(mode); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, filename)
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
	}
	return result
}

var commitPattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
var allowedScenarioStatuses = map[string]bool{StatusPass: true, StatusFail: true, StatusSkipped: true, StatusQuarantined: true}
var allowedPhaseStatuses = map[string]bool{StatusPass: true, StatusFail: true, StatusSkipped: true}
var allowedPhaseIDs = map[string]map[string]bool{
	SuiteLoopback: {"fixture_ready": true, "authoring_v2": true, "profiles_staged": true, "profile_versions": true, "udon_v3": true, "browserdriver_replay": true, "outputs_validated": true, "teardown": true},
	SuitePublic:   {"browsertools_probe": true, "udon_browserdriver_probe": true, "teardown": true},
}
var allowedAssertions = map[string]bool{
	"author_session_v2": true, "reviewed_mfa_kind": true, "reviewed_outputs": true,
	"profiles_reconstructed": true, "oldest_sufficient_versions": true, "udon_v3_lowering": true,
	"browserdriver_replay": true, "typed_outputs_exact": true, "expected_rejection": true,
	"browsertools_presence": true, "runtime_presence": true, "private_material_absent": true,
}
var allowedDetails = map[string]bool{
	"": true, "ok": true, "quarantined": true, "dependency_unavailable": true,
	"fixture_failed": true, "authoring_failed": true, "staging_failed": true,
	"profile_mismatch": true, "replay_failed": true, "output_mismatch": true,
	"timeout": true, "target_unreachable": true, "shape_drift": true,
	"origin_policy_drift": true, "contract_drift": true, "teardown_failed": true,
	"output_bound": true, "stale_candidate": true, "ambiguous_output": true,
	"invalid_context": true, "invalid_response": true, "secret_output": true,
	"origin_rejected": true,
}

func canonicalAssertions(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
