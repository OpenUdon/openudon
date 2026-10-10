// Package browsercontract defines value-free versioned browser package and host evidence metadata.
// Verification proves supplied bytes; current grants, custody and effects belong to the host.
package browsercontract

import (
	"encoding/json"
	"github.com/OpenUdon/openudon/udonreport"
)

type CredentialSlotShape struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Required bool   `json:"required"`
}
type RegistrationInputSlotShape struct {
	Name      string          `json:"name"`
	Kind      string          `json:"kind"`
	Required  bool            `json:"required"`
	Schema    json.RawMessage `json:"schema"` // full native schema+extensions
	Condition json.RawMessage `json:"condition,omitempty"`
}
type BrowserOperationShape struct {
	ProfileVersion         string                       `json:"profile_version"`
	CallKind               string                       `json:"call_kind"` // action|authentication|registration
	SelectedSHA256         string                       `json:"selected_sha256"`
	Origins                []string                     `json:"origins"`
	Effects                json.RawMessage              `json:"effects"`
	ConfirmationPolicy     json.RawMessage              `json:"confirmation_policy,omitempty"`
	AuthenticationRequired bool                         `json:"authentication_required"`
	CredentialSlots        []CredentialSlotShape        `json:"credential_slots"`
	RegistrationInputSlots []RegistrationInputSlotShape `json:"registration_input_slots,omitempty"`
}
type ArtifactRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type BrowserSourceRef struct {
	ID             string      `json:"id"`
	Artifact       ArtifactRef `json:"artifact"`
	ProfileVersion string      `json:"profile_version"`
}
type BrowserSelector struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
	Key   string `json:"key"`
}
type CredentialSlotBinding struct {
	Slot string `json:"slot"`
	Name string `json:"name"`
}
type CredentialRevision struct {
	Name     string `json:"name"`
	Revision string `json:"revision"`
}
type BrowserCallV1 struct {
	StepID                   string                  `json:"step_id"`
	OperationID              string                  `json:"operation_id"`
	InvocationID             string                  `json:"invocation_id"`
	Kind                     string                  `json:"kind"`
	SourceID                 string                  `json:"source_id"`
	SourceSHA256             string                  `json:"source_sha256"`
	ProfileVersion           string                  `json:"profile_version"`
	Selector                 BrowserSelector         `json:"selector"`
	SelectedSHA256           string                  `json:"selected_sha256"`
	OuterProtocol            string                  `json:"outer_protocol"`
	InnerProtocol            string                  `json:"inner_protocol,omitempty"`
	Origins                  []string                `json:"origins"`
	Effects                  json.RawMessage         `json:"effects"`
	ConfirmationPolicySHA256 string                  `json:"confirmation_policy_sha256,omitempty"`
	CredentialSlots          []CredentialSlotBinding `json:"credential_slots"`
	AuthenticationSourceID   string                  `json:"authentication_source_id,omitempty"`
	SessionName              string                  `json:"session_name,omitempty"`
	RegistrationInputBinding string                  `json:"registration_input_binding,omitempty"`
	ReuseAllowed             bool                    `json:"reuse_allowed"`
	SaveAllowed              bool                    `json:"save_allowed"`
}
type BrowserSupplementV1 struct {
	Version         string             `json:"version"` // openudon.browser-supplement.v1
	WorkflowSHA256  string             `json:"workflow_sha256"`
	Sources         []BrowserSourceRef `json:"sources"`
	Calls           []BrowserCallV1    `json:"calls"` // complete ordered inventory
	ReviewArtifacts []ArtifactRef      `json:"review_artifacts"`
}
type BrowserWorkerV1 struct {
	Profile         string `json:"profile"`
	BinarySHA256    string `json:"binary_sha256"`
	ClosureSHA256   string `json:"closure_sha256"`
	RuntimeRevision string `json:"runtime_revision"`
}
type BrowserSessionV1 struct {
	Name          string `json:"name"`
	BindingSHA256 string `json:"binding_sha256"`
	Generation    uint64 `json:"generation"` // positive safe integer
	CreatedAt     string `json:"created_at"`
	ExpiresAt     string `json:"expires_at"`
	ReuseAllowed  bool   `json:"reuse_allowed"`
	SaveAllowed   bool   `json:"save_allowed"`
}
type BrowserConfigV1 struct {
	Version             string               `json:"version"` // openudon.browser-config.v1
	OwnerID             string               `json:"owner_id"`
	AgentID             string               `json:"agent_id"`
	RunID               string               `json:"run_id"`
	PackageSHA256       string               `json:"package_sha256"`
	HandoffSHA256       string               `json:"handoff_sha256"`
	InputsSHA256        string               `json:"inputs_sha256"`
	ApprovalSHA256      string               `json:"approval_sha256"`
	PlanSHA256          string               `json:"plan_sha256"`
	WorkflowSHA256      string               `json:"workflow_sha256"`
	SupplementSHA256    string               `json:"supplement_sha256"`
	Worker              BrowserWorkerV1      `json:"worker"`
	DriverClosureSHA256 string               `json:"driver_closure_sha256"`
	LaunchNonce         string               `json:"launch_nonce"`
	ContainmentLeaseID  string               `json:"containment_lease_id"`
	AdmittedDeadline    string               `json:"admitted_deadline"`
	Origins             []string             `json:"origins"`
	ApprovedCalls       []BrowserCallV1      `json:"approved_calls"`
	CredentialRevisions []CredentialRevision `json:"credential_revisions"`
	Session             *BrowserSessionV1    `json:"session,omitempty"`
	HostFactRefs        []string             `json:"host_fact_refs"` // independent opaque host refs
}
type BrowserAuthorityV1 struct {
	Version      string          `json:"version"` // openudon.browser-authority.v1
	ConfigSHA256 string          `json:"config_sha256"`
	Config       BrowserConfigV1 `json:"config"` // exact expected current attempt
}
type BrowserExecutorV1 struct {
	Invoked      bool   `json:"invoked"`
	Mode         string `json:"mode"`
	ReportSHA256 string `json:"report_sha256,omitempty"`
	ReportSize   int64  `json:"report_size,omitempty"`
}

// New closed wire; RunEvidence/legacy BrowserConfig do not change. No host
// paths, arguments, environment names or registration-private identities.
type BrowserRunEvidenceV1 struct {
	Version       string                    `json:"version"` // openudon.browser-run-evidence.v1
	RunID         string                    `json:"run_id"`
	CreatedAt     string                    `json:"created_at"`
	Scope         string                    `json:"scope"`
	Tier          string                    `json:"tier"`
	DryRun        bool                      `json:"dry_run"`
	ApprovalState string                    `json:"approval_state"`
	Browser       BrowserConfigV1           `json:"browser"`
	Executor      BrowserExecutorV1         `json:"executor"`
	StepExecution *udonreport.ObservationV5 `json:"step_execution,omitempty"`
}
