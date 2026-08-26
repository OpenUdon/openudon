// Package browsertransactioneval owns the canonical, value-free evidence
// contract for cross-package browser-profile transaction qualification.
package browsertransactioneval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/browserscenario"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const (
	ReportVersion  = "openudon.browser-transaction-qualification.v1"
	StatusPass     = "pass"
	StatusFail     = "fail"
	CaseBAPBCP     = "bap_bcp"
	CaseBRP        = "brp"
	maxReportBytes = 1 << 20
)

const (
	GateReportSchema          = "report_schema"
	GateBAPBCPProducer        = "bap_bcp_producer"
	GateBAPBCPTransaction     = "bap_bcp_transaction"
	GateBAPBCPPackage         = "bap_bcp_package"
	GateBAPBCPHandoff         = "bap_bcp_handoff"
	GateBAPBCPReplay          = "bap_bcp_replay"
	GateBRPProducer           = "brp_producer"
	GateBRPNetwork            = "brp_network"
	GateBRPTransaction        = "brp_transaction"
	GateBRPPackage            = "brp_package"
	GateBRPExecutorRejection  = "brp_executor_rejection"
	GateProtocolBounds        = "protocol_bounds"
	GateLifecycleDrift        = "lifecycle_drift"
	GateConcurrentLifecycle   = "concurrent_lifecycle"
	GateFilesystemRollback    = "filesystem_rollback"
	GateIndeterminateRecovery = "indeterminate_recovery"
	GateFrontendConflicts     = "frontend_conflicts"
	GateSensitiveArtifactScan = "sensitive_artifact_scan"
)

var gateOrder = []string{
	GateReportSchema,
	GateBAPBCPProducer,
	GateBAPBCPTransaction,
	GateBAPBCPPackage,
	GateBAPBCPHandoff,
	GateBAPBCPReplay,
	GateBRPProducer,
	GateBRPNetwork,
	GateBRPTransaction,
	GateBRPPackage,
	GateBRPExecutorRejection,
	GateProtocolBounds,
	GateLifecycleDrift,
	GateConcurrentLifecycle,
	GateFilesystemRollback,
	GateIndeterminateRecovery,
	GateFrontendConflicts,
	GateSensitiveArtifactScan,
}

var artifactKindOrder = []string{
	"producer_result",
	"transaction",
	"preparation",
	"qualification",
	"generation",
	"selection",
	"package",
	"handoff",
	"workflow",
}

var failureCodes = map[string]bool{
	"contract_invalid":       true,
	"dependency_mismatch":    true,
	"sandbox_unavailable":    true,
	"producer_failed":        true,
	"transaction_rejected":   true,
	"package_failed":         true,
	"handoff_failed":         true,
	"replay_failed":          true,
	"mutation_observed":      true,
	"executor_invoked":       true,
	"adversarial_acceptance": true,
	"concurrency_violation":  true,
	"rollback_violation":     true,
	"recovery_violation":     true,
	"frontend_conflict":      true,
	"sensitive_artifact":     true,
}

// Report contains only closed vocabulary, booleans, counters, commit IDs,
// timestamps, and SHA-256 digests. It has no field capable of carrying a path,
// subprocess output, browser content, account identity, or credential value.
type Report struct {
	Version      string               `json:"version"`
	Status       string               `json:"status"`
	GeneratedAt  string               `json:"generated_at"`
	Repositories []RepositoryRevision `json:"repositories"`
	Posture      Posture              `json:"posture"`
	Artifacts    []ArtifactDigest     `json:"artifacts"`
	Summary      Summary              `json:"summary"`
	Results      []GateResult         `json:"results"`
}

type RepositoryRevision struct {
	Name          string `json:"name"`
	Commit        string `json:"commit"`
	ModuleVersion string `json:"module_version"`
	Published     bool   `json:"published"`
}

type Posture struct {
	SandboxRequired                   bool     `json:"sandbox_required"`
	SandboxEnabled                    bool     `json:"sandbox_enabled"`
	LoopbackOnly                      bool     `json:"loopback_only"`
	PublicTargetsContacted            bool     `json:"public_targets_contacted"`
	RegistrationAuthoringMethods      []string `json:"registration_authoring_methods"`
	RegistrationAuthoringPostRequests int      `json:"registration_authoring_post_requests"`
	AccountCreated                    bool     `json:"account_created"`
	ExecutorInvokedForRegistration    bool     `json:"executor_invoked_for_registration"`
	ContainsPrivateMaterial           bool     `json:"contains_private_material"`
	ValueFree                         bool     `json:"value_free"`
}

type ArtifactDigest struct {
	Case   string `json:"case"`
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

type Summary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

type GateResult struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	FailureCode   string `json:"failure_code"`
	EvidenceCount int    `json:"evidence_count"`
}

type BuildRequest struct {
	GeneratedAt  time.Time
	Repositories []RepositoryRevision
	Posture      Posture
	Artifacts    []ArtifactDigest
	Results      []GateResult
}

// NewReport derives status and summary instead of trusting caller counters.
func NewReport(request BuildRequest) (*Report, error) {
	results := append([]GateResult(nil), request.Results...)
	report := &Report{
		Version: ReportVersion, Status: StatusPass,
		GeneratedAt:  request.GeneratedAt.UTC().Format(time.RFC3339),
		Repositories: append([]RepositoryRevision(nil), request.Repositories...),
		Posture:      clonePosture(request.Posture),
		Artifacts:    append([]ArtifactDigest(nil), request.Artifacts...),
		Results:      results,
	}
	report.Summary = summarize(results)
	if report.Summary.Failed != 0 {
		report.Status = StatusFail
	}
	if err := Validate(report); err != nil {
		return nil, err
	}
	return report, nil
}

// Validate independently enforces the complete closed evidence contract.
func Validate(report *Report) error {
	if report == nil {
		return errors.New("browser transaction qualification report is required")
	}
	if report.Version != ReportVersion || report.Status != StatusPass && report.Status != StatusFail {
		return errors.New("browser transaction qualification identity is invalid")
	}
	generatedAt, err := time.Parse(time.RFC3339, report.GeneratedAt)
	if err != nil || generatedAt.IsZero() || report.GeneratedAt != generatedAt.UTC().Format(time.RFC3339) {
		return errors.New("browser transaction qualification timestamp is invalid")
	}
	if err := validateRepositories(report.Repositories); err != nil {
		return err
	}
	if err := validatePosture(report.Posture); err != nil {
		return err
	}
	if err := validateArtifacts(report.Artifacts); err != nil {
		return err
	}
	if len(report.Results) != len(gateOrder) {
		return fmt.Errorf("browser transaction qualification result count = %d, want %d", len(report.Results), len(gateOrder))
	}
	for index, result := range report.Results {
		if result.ID != gateOrder[index] || result.EvidenceCount <= 0 || result.EvidenceCount > 1<<20 {
			return fmt.Errorf("browser transaction qualification result %d is invalid", index)
		}
		switch result.Status {
		case StatusPass:
			if result.FailureCode != "" {
				return fmt.Errorf("passing browser transaction gate %q has a failure code", result.ID)
			}
		case StatusFail:
			if !failureCodes[result.FailureCode] {
				return fmt.Errorf("failing browser transaction gate %q has an invalid failure code", result.ID)
			}
		default:
			return fmt.Errorf("browser transaction gate %q has an invalid status", result.ID)
		}
	}
	wantSummary := summarize(report.Results)
	if report.Summary != wantSummary || (report.Status == StatusPass) != (wantSummary.Failed == 0) {
		return errors.New("browser transaction qualification summary is invalid")
	}
	return nil
}

// CanonicalBytes produces the sole accepted JSON encoding for the report.
func CanonicalBytes(report *Report) ([]byte, error) {
	if err := Validate(report); err != nil {
		return nil, err
	}
	data, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func Write(path string, report *Report) error {
	data, err := CanonicalBytes(report)
	if err != nil {
		return err
	}
	if strings.TrimSpace(path) == "" {
		return errors.New("browser transaction qualification report path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := atomicfile.Write(path, data, 0o644); err != nil {
		return err
	}
	return evidencefile.WriteDigestSidecar(path, data, 0o644)
}

func VerifyFile(path string, requirePass bool) (*Report, error) {
	data, _, err := evidencefile.ReadRegular(path, maxReportBytes)
	if err != nil {
		return nil, err
	}
	if err := verifyDigestSidecar(path, data); err != nil {
		return nil, err
	}
	if err := validateRequiredWireFields(data); err != nil {
		return nil, err
	}
	var report Report
	if err := evidencefile.DecodeStrict(data, &report); err != nil {
		return nil, fmt.Errorf("decode browser transaction qualification report: %w", err)
	}
	canonical, err := CanonicalBytes(&report)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(data, canonical) {
		return nil, errors.New("browser transaction qualification report is not canonical JSON")
	}
	if requirePass && report.Status != StatusPass {
		return &report, errors.New("browser transaction qualification report did not pass")
	}
	return &report, nil
}

func validateRepositories(repositories []RepositoryRevision) error {
	if len(repositories) != 3 {
		return errors.New("browser transaction qualification requires three repositories")
	}
	lock, err := browserscenario.LoadCompatibilityLock()
	if err != nil {
		return err
	}
	locked := make(map[string]browserscenario.LockedRevision, len(lock.Components))
	for _, component := range lock.Components {
		locked[component.Name] = component
	}
	want := []string{"openudon", "browsertools", "uws"}
	for index, repository := range repositories {
		if repository.Name != want[index] || !evidencefile.ValidGitObject(repository.Commit) {
			return errors.New("browser transaction qualification repository identity is invalid")
		}
		if repository.Name == "openudon" {
			if repository.Published || repository.ModuleVersion != "" {
				return errors.New("browser transaction qualification must classify OpenUdon as local and unpublished")
			}
			continue
		}
		component := locked[repository.Name]
		if repository.Commit != component.Commit || repository.ModuleVersion != component.Version || !repository.Published {
			return fmt.Errorf("browser transaction qualification %s revision does not match the published lock", repository.Name)
		}
	}
	return nil
}

func validatePosture(posture Posture) error {
	if !posture.SandboxRequired || !posture.SandboxEnabled || !posture.LoopbackOnly || posture.PublicTargetsContacted ||
		posture.RegistrationAuthoringPostRequests != 0 || posture.AccountCreated || posture.ExecutorInvokedForRegistration ||
		posture.ContainsPrivateMaterial || !posture.ValueFree || !equalStrings(posture.RegistrationAuthoringMethods, []string{"GET", "HEAD"}) {
		return errors.New("browser transaction qualification posture is invalid")
	}
	return nil
}

func validateArtifacts(artifacts []ArtifactDigest) error {
	wantCount := 2 * len(artifactKindOrder)
	if len(artifacts) != wantCount {
		return fmt.Errorf("browser transaction qualification artifact count = %d, want %d", len(artifacts), wantCount)
	}
	index := 0
	for _, caseID := range []string{CaseBAPBCP, CaseBRP} {
		for _, kind := range artifactKindOrder {
			artifact := artifacts[index]
			if artifact.Case != caseID || artifact.Kind != kind || !validTaggedSHA256(artifact.SHA256) {
				return fmt.Errorf("browser transaction qualification artifact %d is invalid", index)
			}
			index++
		}
	}
	return nil
}

func summarize(results []GateResult) Summary {
	summary := Summary{Total: len(results)}
	for _, result := range results {
		if result.Status == StatusPass {
			summary.Passed++
		} else if result.Status == StatusFail {
			summary.Failed++
		}
	}
	return summary
}

func validateRequiredWireFields(data []byte) error {
	root, err := requiredObject(data, "report", "version", "status", "generated_at", "repositories", "posture", "artifacts", "summary", "results")
	if err != nil {
		return err
	}
	var repositories []json.RawMessage
	if err := json.Unmarshal(root["repositories"], &repositories); err != nil {
		return err
	}
	for index, raw := range repositories {
		if _, err := requiredObject(raw, fmt.Sprintf("repositories[%d]", index), "name", "commit", "module_version", "published"); err != nil {
			return err
		}
	}
	if _, err := requiredObject(root["posture"], "posture", "sandbox_required", "sandbox_enabled", "loopback_only", "public_targets_contacted", "registration_authoring_methods", "registration_authoring_post_requests", "account_created", "executor_invoked_for_registration", "contains_private_material", "value_free"); err != nil {
		return err
	}
	var artifacts []json.RawMessage
	if err := json.Unmarshal(root["artifacts"], &artifacts); err != nil {
		return err
	}
	for index, raw := range artifacts {
		if _, err := requiredObject(raw, fmt.Sprintf("artifacts[%d]", index), "case", "kind", "sha256"); err != nil {
			return err
		}
	}
	if _, err := requiredObject(root["summary"], "summary", "total", "passed", "failed"); err != nil {
		return err
	}
	var results []json.RawMessage
	if err := json.Unmarshal(root["results"], &results); err != nil {
		return err
	}
	for index, raw := range results {
		if _, err := requiredObject(raw, fmt.Sprintf("results[%d]", index), "id", "status", "failure_code", "evidence_count"); err != nil {
			return err
		}
	}
	return nil
}

func requiredObject(data []byte, label string, keys ...string) (map[string]json.RawMessage, error) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		return nil, fmt.Errorf("browser transaction qualification %s is invalid", label)
	}
	for _, key := range keys {
		value, ok := object[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, fmt.Errorf("browser transaction qualification %s requires non-null field %q", label, key)
		}
	}
	return object, nil
}

func verifyDigestSidecar(path string, data []byte) error {
	sidecar, _, err := evidencefile.ReadRegular(path+".sha256", 256)
	if err != nil {
		return fmt.Errorf("browser transaction qualification digest sidecar: %w", err)
	}
	sum := sha256.Sum256(data)
	want := "sha256:" + hex.EncodeToString(sum[:]) + "  " + filepath.Base(path) + "\n"
	if string(sidecar) != want {
		return errors.New("browser transaction qualification digest mismatch")
	}
	return nil
}

func validTaggedSHA256(value string) bool {
	return strings.HasPrefix(value, "sha256:") && evidencefile.ValidSHA256(strings.TrimPrefix(value, "sha256:"))
}

func clonePosture(posture Posture) Posture {
	posture.RegistrationAuthoringMethods = append([]string(nil), posture.RegistrationAuthoringMethods...)
	return posture
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
