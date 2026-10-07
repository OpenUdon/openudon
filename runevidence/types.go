// Package runevidence owns neutral run-evidence wires and explicit-byte verification.
// Browser execution/verification remains on its separately pinned private CLI path.
package runevidence

import (
	asyncevidence "github.com/OpenUdon/evidence/async"
	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/udonreport"
)

const Version = "openudon.run-evidence.v2"
const StepVersion = "openudon.run-evidence.v3"
const BrokerVersion = "openudon.run-evidence.v4"
const LegacyVersion = "openudon.run-evidence.v1"
const AsyncEvidenceVersion = "openudon.async-evidence-bundle.v1"

type RunEvidence struct {
	StepExecution      *udonreport.ObservationV5 `json:"step_execution,omitempty"`
	Version            string                    `json:"version"`
	RunID              string                    `json:"run_id"`
	CreatedAt          string                    `json:"created_at"`
	Scope              string                    `json:"scope"`
	Tier               string                    `json:"tier"`
	DryRun             bool                      `json:"dry_run"`
	ApprovalState      string                    `json:"approval_state"`
	PackageSHA256      string                    `json:"package_sha256"`
	HandoffSHA256      string                    `json:"handoff_sha256"`
	ApprovalSHA256     string                    `json:"approval_sha256"`
	RunConfigSHA256    string                    `json:"run_config_sha256"`
	RunConfigPath      string                    `json:"run_config_path"`
	PackageRoot        string                    `json:"package_root"`
	WorkDir            string                    `json:"workdir"`
	StageKind          string                    `json:"stage_kind"`
	StagePath          string                    `json:"stage_path"`
	WorkflowPath       string                    `json:"workflow_path"`
	PackagePaths       []string                  `json:"package_paths"`
	APISourcePaths     []string                  `json:"api_source_paths,omitempty"`
	CredentialBindings []string                  `json:"credential_bindings,omitempty"`
	CredentialEnvNames []string                  `json:"credential_env_names,omitempty"`
	Browser            *BrowserConfig            `json:"browser,omitempty"`
	Gates              []RunEvidenceGate         `json:"gates"`
	Executor           RunEvidenceExecutor       `json:"executor"`
	AsyncEvidenceFiles []RunEvidenceAsyncFile    `json:"async_evidence_files,omitempty"`
	Broker             *authority.Authority      `json:"broker,omitempty"`
}

type RunEvidenceGate struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

type RunEvidenceExecutor struct {
	Invoked      bool     `json:"invoked"`
	Mode         string   `json:"mode"`
	RunnerPath   string   `json:"runner_path,omitempty"`
	Argv         []string `json:"argv,omitempty"`
	ReportPath   string   `json:"report_path,omitempty"`
	ReportSHA256 string   `json:"report_sha256,omitempty"`
	ReportSize   int64    `json:"report_size,omitempty"`
}

type RunEvidenceAsyncFile struct {
	Path    string `json:"path"`
	Digest  string `json:"digest"`
	Records int    `json:"records"`
	Purpose string `json:"purpose"`
}

type AsyncEvidenceBundle struct {
	Version string                `json:"version"`
	Records []AsyncEvidenceRecord `json:"records"`
}

type AsyncEvidenceRecord struct {
	Kind                        string                                     `json:"kind"`
	ExecutionRequest            *asyncevidence.ExecutionRequest            `json:"execution_request,omitempty"`
	ExecutionResponse           *asyncevidence.ExecutionResponse           `json:"execution_response,omitempty"`
	StatusObservation           *asyncevidence.StatusObservation           `json:"status_observation,omitempty"`
	ConfirmationReadObservation *asyncevidence.ConfirmationReadObservation `json:"confirmation_read_observation,omitempty"`
}

// BrowserConfig is the complete value-free browser replay contract. Secret
// and session values stay in the named environment variables.
type BrowserConfig struct {
	DriverPath                    string               `json:"driver_path,omitempty"`
	DriverArgs                    []string             `json:"driver_args,omitempty"`
	DriverEnvironment             []string             `json:"driver_environment,omitempty"`
	Protocol                      string               `json:"protocol"`
	RegistrationInputUI           bool                 `json:"registration_input_ui,omitempty"`
	RegistrationInputService      string               `json:"registration_input_service,omitempty"`
	RegistrationInputTokenEnv     string               `json:"registration_input_token_env,omitempty"`
	RegistrationInputExpectedEnv  string               `json:"registration_input_expected_env,omitempty"`
	CredentialEnvironment         []EnvironmentBinding `json:"credential_environment,omitempty"`
	SessionEnvironment            []EnvironmentBinding `json:"session_environment,omitempty"`
	ApprovedOperations            []string             `json:"approved_operations,omitempty"`
	ApprovedAuthentication        []string             `json:"approved_authentication,omitempty"`
	ApprovedRegistration          []string             `json:"approved_registration,omitempty"`
	AttestedRegistration          []string             `json:"attested_registration,omitempty"`
	RegistrationAttestationSHA256 string               `json:"registration_attestation_sha256,omitempty"`
}

// EnvironmentBinding maps a reviewed symbolic runtime name to its canonical
// allowlisted environment-variable name; it never contains a value.
type EnvironmentBinding struct {
	Name        string `json:"name"`
	Environment string `json:"environment"`
}
