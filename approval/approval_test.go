package approval_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/approval"
	"github.com/OpenUdon/openudon/digest"
)

func TestPublishedApprovalWireAndAuthority(t *testing.T) {
	data, err := os.ReadFile("../docs/fixtures/broker-execution-v1/approval-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	var value approval.Approval
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if digest.SHA256(wire) != "27600c38a0e5c922c73c99a541ac433e0c0027b9f851ec39eaba20e78a6af800" {
		t.Fatal("published approval JSON field order or bytes changed")
	}
	now, _ := time.Parse(time.RFC3339, "2026-04-29T12:30:00Z")
	if err := approval.Validate(value, value.Scope, value.PackageSHA256, approval.TierSandbox, now); err != nil {
		t.Fatal(err)
	}
	if approval.Validate(value, value.Scope, value.PackageSHA256, approval.TierProduction, now) == nil {
		t.Fatal("sandbox state authorized production")
	}
	if approval.Validate(value, value.Scope, value.PackageSHA256, approval.TierSandbox, now.Add(time.Hour)) == nil {
		t.Fatal("expired authority accepted")
	}
	value.Version = approval.Version
	if approval.Validate(value, value.Scope, value.PackageSHA256, approval.TierSandbox, now) == nil {
		t.Fatal("broker authority downgraded to v1")
	}
	value.Broker = nil
	value.ExpiresAt = ""
	if err := approval.Validate(value, value.Scope, value.PackageSHA256, approval.TierSandbox, now); err != nil {
		t.Fatal(err)
	}
	wire, err = json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	const legacy = `{"version":"openudon.approval.v1","scope":"examples/support-email","state":"approved_for_sandbox","reviewer":"Ada","approved_at":"2026-04-29T12:00:00Z","package_sha256":"964910aa91c2c4b554eaa22d855b56a86e3f4c75bf6ed231f86ba4b0bed3e299"}`
	if string(wire) != legacy {
		t.Fatal("legacy approval bytes changed")
	}
}
