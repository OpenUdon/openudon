// Package approval exposes the unchanged approval-v1/v2 wire and value-only
// validation. Validation observes supplied identity/time; it never executes.
package approval

import (
	"fmt"
	"github.com/OpenUdon/openudon/authority"
	"strings"
	"time"
)

const Version = "openudon.approval.v1"
const BrokerVersion = "openudon.approval.v2"
const TierSandbox = "sandbox"
const TierProduction = "production"
const StateApprovedForSandbox = "approved_for_sandbox"
const StateApprovedForProduction = "approved_for_production"

type Approval struct {
	Version       string               `json:"version"`
	Scope         string               `json:"scope"`
	State         string               `json:"state"`
	Reviewer      string               `json:"reviewer"`
	ApprovedAt    string               `json:"approved_at"`
	ExpiresAt     string               `json:"expires_at,omitempty"`
	PackageSHA256 string               `json:"package_sha256"`
	Notes         string               `json:"notes,omitempty"`
	Broker        *authority.Authority `json:"broker,omitempty"`
}

func Validate(approval Approval, scope, digest, tier string, now time.Time) error {
	if (approval.Version != Version || approval.Broker != nil) && (approval.Version != BrokerVersion || approval.Broker == nil) {
		return fmt.Errorf("approval version must be %s", Version)
	}
	if approval.Broker != nil {
		if approval.Broker.ValidateAt(now) != nil || approval.Broker.PackageSHA256 != digest || approval.Broker.ApprovedAt != approval.ApprovedAt || approval.Broker.ExpiresAt != approval.ExpiresAt {
			return fmt.Errorf("broker approval authority mismatch or expired")
		}
	}
	if approval.Scope != scope {
		return fmt.Errorf("approval scope %q does not match %q", approval.Scope, scope)
	}
	if strings.TrimSpace(approval.Reviewer) == "" {
		return fmt.Errorf("approval reviewer is required")
	}
	if _, err := time.Parse(time.RFC3339, approval.ApprovedAt); err != nil {
		return fmt.Errorf("approval approved_at must be RFC3339: %w", err)
	}
	if strings.TrimSpace(approval.ExpiresAt) != "" {
		expires, err := time.Parse(time.RFC3339, approval.ExpiresAt)
		if err != nil {
			return fmt.Errorf("approval expires_at must be RFC3339: %w", err)
		}
		if !now.Before(expires) {
			return fmt.Errorf("approval expired at %s", expires.Format(time.RFC3339))
		}
	}
	if approval.PackageSHA256 != digest {
		return fmt.Errorf("approval package_sha256 does not match current handoff package")
	}
	if err := ValidateTierState(tier, approval.State); err != nil {
		return err
	}
	return nil
}

func ValidateTierState(tier, state string) error {
	switch tier {
	case TierSandbox:
		if state == StateApprovedForSandbox || state == StateApprovedForProduction {
			return nil
		}
	case TierProduction:
		if state == StateApprovedForProduction {
			return nil
		}
	default:
		return fmt.Errorf("--tier must be %s or %s", TierSandbox, TierProduction)
	}
	return fmt.Errorf("approval state %q is not valid for %s tier", state, tier)
}
