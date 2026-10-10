package approval

import (
	"encoding/json"
	"reflect"
	"time"

	"github.com/OpenUdon/openudon/browsercontract"
)

const BrowserVersion = "openudon.browser-approval.v1"

// BrowserApprovalV1 is an independently finalized human review receipt. It
// has no enclosing Authority/Config digest or host-fact self reference. The
// trusted host alone confirms it and owns fresh grants/claims/revocation.
type BrowserApprovalV1 struct {
	Version             string                               `json:"version"`
	State               string                               `json:"state"`
	OwnerID             string                               `json:"owner_id"`
	AgentID             string                               `json:"agent_id"`
	RunID               string                               `json:"run_id"`
	PackageSHA256       string                               `json:"package_sha256"`
	HandoffSHA256       string                               `json:"handoff_sha256"`
	InputsSHA256        string                               `json:"inputs_sha256"`
	PlanSHA256          string                               `json:"plan_sha256"`
	WorkflowSHA256      string                               `json:"workflow_sha256"`
	SupplementSHA256    string                               `json:"supplement_sha256"`
	Worker              browsercontract.BrowserWorkerV1      `json:"worker"`
	DriverClosureSHA256 string                               `json:"driver_closure_sha256"`
	AdmittedDeadline    string                               `json:"admitted_deadline"`
	Origins             []string                             `json:"origins"`
	ApprovedCalls       []browsercontract.BrowserCallV1      `json:"approved_calls"`
	CredentialRevisions []browsercontract.CredentialRevision `json:"credential_revisions"`
	Session             *browsercontract.BrowserSessionV1    `json:"session,omitempty"`
	ApprovedAt          string                               `json:"approved_at"`
}

// ReviewBrowserConfig constructs review metadata, never human confirmation.
// The host finalizes its exact bytes before constructing the containing Config.
func ReviewBrowserConfig(c browsercontract.BrowserConfigV1, approvedAt string) (BrowserApprovalV1, error) {
	a := BrowserApprovalV1{Version: BrowserVersion, State: "approved_for_sandbox", OwnerID: c.OwnerID, AgentID: c.AgentID, RunID: c.RunID, PackageSHA256: c.PackageSHA256, HandoffSHA256: c.HandoffSHA256, InputsSHA256: c.InputsSHA256, PlanSHA256: c.PlanSHA256, WorkflowSHA256: c.WorkflowSHA256, SupplementSHA256: c.SupplementSHA256, Worker: c.Worker, DriverClosureSHA256: c.DriverClosureSHA256, AdmittedDeadline: c.AdmittedDeadline, Origins: c.Origins, ApprovedCalls: c.ApprovedCalls, CredentialRevisions: c.CredentialRevisions, Session: c.Session, ApprovedAt: approvedAt}
	raw, err := json.Marshal(a)
	if err != nil {
		return BrowserApprovalV1{}, browsercontract.ErrContract
	}
	return DecodeBrowserApproval(raw)
}
func (a BrowserApprovalV1) config() browsercontract.BrowserConfigV1 {
	return browsercontract.BrowserConfigV1{Version: browsercontract.ConfigVersion, OwnerID: a.OwnerID, AgentID: a.AgentID, RunID: a.RunID, PackageSHA256: a.PackageSHA256, HandoffSHA256: a.HandoffSHA256, InputsSHA256: a.InputsSHA256, ApprovalSHA256: browsercontract.SHA256([]byte("review-metadata-only")), PlanSHA256: a.PlanSHA256, WorkflowSHA256: a.WorkflowSHA256, SupplementSHA256: a.SupplementSHA256, Worker: a.Worker, DriverClosureSHA256: a.DriverClosureSHA256, LaunchNonce: "review-only", ContainmentLeaseID: "review-only", AdmittedDeadline: a.AdmittedDeadline, Origins: a.Origins, ApprovedCalls: a.ApprovedCalls, CredentialRevisions: a.CredentialRevisions, Session: a.Session, HostFactRefs: []string{"review-only"}}
}
func (a BrowserApprovalV1) Validate() error {
	approved, e1 := time.Parse(time.RFC3339Nano, a.ApprovedAt)
	deadline, e2 := time.Parse(time.RFC3339Nano, a.AdmittedDeadline)
	if a.Version != BrowserVersion || a.State != "approved_for_sandbox" || e1 != nil || e2 != nil || !deadline.After(approved) || a.config().Validate() != nil {
		return browsercontract.ErrContract
	}
	return nil
}
func (a BrowserApprovalV1) CanonicalBytes() ([]byte, error) {
	if a.Validate() != nil {
		return nil, browsercontract.ErrContract
	}
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, browsercontract.ErrContract
	}
	return browsercontract.CanonicalJSON(raw)
}
func DecodeBrowserApproval(data []byte) (BrowserApprovalV1, error) {
	var a BrowserApprovalV1
	if browsercontract.Decode(data, &a) != nil || a.Validate() != nil {
		return BrowserApprovalV1{}, browsercontract.ErrContract
	}
	return a, nil
}

// CheckBrowserApproval compares an independently finalized receipt to the exact
// expected current Config. Neither receipt nor Config attests human authority.
func CheckBrowserApproval(data []byte, c browsercontract.BrowserConfigV1) error {
	a, err := DecodeBrowserApproval(data)
	if err != nil || c.Validate() != nil || browsercontract.SHA256(data) != c.ApprovalSHA256 {
		return browsercontract.ErrContract
	}
	expected, err := ReviewBrowserConfig(c, a.ApprovedAt)
	if err != nil || !reflect.DeepEqual(a, expected) {
		return browsercontract.ErrContract
	}
	return nil
}
