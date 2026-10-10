package packagev3

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/OpenUdon/openudon/approval"
	"github.com/OpenUdon/openudon/browsercontract"
)

type BrowserAuthorityOptions struct {
	ExpectedConfig      browsercontract.BrowserConfigV1 // independent current host attempt
	FinalizedApproval   []byte                          // independent exact human review receipt; no Config hash
	Now                 time.Time                       // independent host time; no implicit clock or deadline renewal
	ConfirmedPlanSHA256 string                          // host's separately confirmed exact rendered plan
}

// DeriveBrowserAuthority independently binds verified bytes, admitted worker,
// complete browser inventory and exact finalized receipt. This value is review
// metadata, never a host grant, durable claim, browser launch or credential lease.
func DeriveBrowserAuthority(ctx context.Context, v VerifiedPackage, execution ExecutionOptions, o BrowserAuthorityOptions) (browsercontract.BrowserAuthorityV1, error) {
	plan, err := DeriveExecutionPlan(ctx, v, execution)
	if err != nil {
		return browsercontract.BrowserAuthorityV1{}, err
	}
	c := o.ExpectedConfig
	deadline, e1 := time.Parse(time.RFC3339Nano, c.AdmittedDeadline)
	receipt, e2 := approval.DecodeBrowserApproval(o.FinalizedApproval)
	approved, e3 := time.Parse(time.RFC3339Nano, receipt.ApprovedAt)
	if o.Now.IsZero() || e1 != nil || e2 != nil || e3 != nil || o.Now.Before(approved) || !o.Now.Before(deadline) {
		return browsercontract.BrowserAuthorityV1{}, ErrPackage
	}
	if c.Session != nil {
		expires, err := time.Parse(time.RFC3339Nano, c.Session.ExpiresAt)
		if err != nil || !o.Now.Before(expires) {
			return browsercontract.BrowserAuthorityV1{}, ErrPackage
		}
	}
	if v.manifest.Browser == nil || c.Validate() != nil || o.ConfirmedPlanSHA256 != plan.PlanSHA256 || c.PlanSHA256 != plan.PlanSHA256 || c.PackageSHA256 != v.sha256 || c.HandoffSHA256 != plan.HandoffSHA256 || c.InputsSHA256 != plan.InputsSHA256 || c.WorkflowSHA256 != v.manifest.Workflow.SHA256 || c.SupplementSHA256 != v.manifest.Browser.SHA256 || c.Worker.BinarySHA256 != execution.Worker.BinarySHA256 || c.Worker.ClosureSHA256 != execution.Worker.ClosureSHA256 || c.Worker.RuntimeRevision != execution.Worker.RuntimeRevision || approval.CheckBrowserApproval(o.FinalizedApproval, c) != nil {
		return browsercontract.BrowserAuthorityV1{}, ErrPackage
	}
	calls := []BrowserCallV1{}
	for _, leaf := range plan.Operations {
		if leaf.Browser != nil {
			calls = append(calls, *leaf.Browser)
		}
	}
	if !reflect.DeepEqual(calls, c.ApprovedCalls) {
		return browsercontract.BrowserAuthorityV1{}, ErrPackage
	}
	// Decode an independent copy, so changing expected host inputs cannot mutate
	// the derived authority after this pure check.
	canonical, err := browsercontract.CanonicalDigest(c)
	if err != nil {
		return browsercontract.BrowserAuthorityV1{}, ErrPackage
	}
	a := browsercontract.BrowserAuthorityV1{Version: browsercontract.AuthorityVersion, ConfigSHA256: canonical, Config: c}
	raw, err := canonicalAuthorityBytes(a)
	if err != nil {
		return browsercontract.BrowserAuthorityV1{}, ErrPackage
	}
	copy, err := browsercontract.DecodeAuthority(raw)
	if err != nil {
		return browsercontract.BrowserAuthorityV1{}, ErrPackage
	}
	return copy, nil
}
func CheckBrowserAuthority(ctx context.Context, v VerifiedPackage, execution ExecutionOptions, o BrowserAuthorityOptions, submitted browsercontract.BrowserAuthorityV1) error {
	expected, err := DeriveBrowserAuthority(ctx, v, execution, o)
	if err != nil {
		return err
	}
	if submitted.Validate() != nil || !reflect.DeepEqual(expected, submitted) {
		return ErrPackage
	}
	return nil
}

func canonicalAuthorityBytes(a browsercontract.BrowserAuthorityV1) ([]byte, error) {
	raw, err := json.Marshal(a)
	if err != nil {
		return nil, ErrPackage
	}
	return browsercontract.CanonicalJSON(raw)
}
