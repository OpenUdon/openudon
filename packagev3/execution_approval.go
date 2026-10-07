package packagev3

import (
	"context"
	"time"

	"github.com/OpenUdon/openudon/approval"
	"github.com/OpenUdon/openudon/authority"
)

type ExecutionApprovalOptions struct {
	Execution ExecutionOptions
	// ExpectedPlanSHA256 comes from the trusted host's exact confirmed review,
	// including selected worker/closure. Package text and producer flags are not
	// custody for this identity. The host separately checks owner/grant freshness.
	ExpectedPlanSHA256 string
	Approval           approval.Approval
	Credentials        map[string]authority.Binding
	Tier               string
	Now                time.Time
}

// CheckExecutionApproval binds unchanged approval wires to the separately
// confirmed exact operation/input/worker plan. HTTP always requires broker v2;
// pure functions retain v1 after independent native admission. It performs no
// effects and cannot replace current host grant/owner/revocation checks.
func CheckExecutionApproval(ctx context.Context, verified VerifiedPackage, options ExecutionApprovalOptions) error {
	if options.Now.IsZero() || !digestValid(options.ExpectedPlanSHA256) {
		return ErrPackage
	}
	plan, err := DeriveExecutionPlan(ctx, verified, options.Execution)
	if err != nil {
		return err
	}
	if plan.PlanSHA256 != options.ExpectedPlanSHA256 {
		return ErrPackage
	}
	a := options.Approval
	if approval.Validate(a, plan.Scope, plan.PackageSHA256, options.Tier, options.Now) != nil {
		return ErrPackage
	}
	approved, e1 := time.Parse(time.RFC3339, a.ApprovedAt)
	expires, e2 := time.Parse(time.RFC3339, a.ExpiresAt)
	if e1 != nil || e2 != nil || approved.After(options.Now) || !expires.After(approved) || !options.Now.Before(expires) {
		return ErrPackage
	}
	hasHTTP, hasFunctions := false, false
	for _, op := range plan.Operations {
		hasHTTP = hasHTTP || op.Kind == "http"
		hasFunctions = hasFunctions || op.Kind == "fnct"
	}
	if hasHTTP {
		if hasFunctions || a.Version != approval.BrokerVersion || a.Broker == nil {
			return ErrPackage
		}
		return CheckBrokerAuthority(ctx, verified, options.Execution, *a.Broker, options.Credentials, options.Now)
	}
	if !hasFunctions || a.Version != approval.Version || a.Broker != nil || len(options.Credentials) != 0 {
		return ErrPackage
	}
	return ctx.Err()
}
