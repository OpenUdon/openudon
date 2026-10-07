package authority_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/digest"
)

func TestPublishedAuthorityWire(t *testing.T) {
	data, err := os.ReadFile("../docs/fixtures/broker-handoff-v1/authority-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	var value authority.Authority
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	wire, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if digest.SHA256(wire) != "55934733c5e33a0d73756643229ab544cf940b2de2bbaa652b2630eeccd22a53" {
		t.Fatal("published JSON field order or bytes changed")
	}
	if value.Digest() != value.PolicySHA256 {
		t.Fatal("published authority self digest changed")
	}
	now, _ := time.Parse(time.RFC3339, "2026-10-05T00:01:00Z")
	if err := value.ValidateAt(now); err != nil {
		t.Fatal(err)
	}
	value.Operations[0].Method = "POST"
	if value.ValidateAt(now) == nil {
		t.Fatal("accepted operation drift under old authority")
	}
}

func TestAuthorityRejectsWhitespaceInCanonicalHashes(t *testing.T) {
	data, err := os.ReadFile("../docs/fixtures/broker-handoff-v1/authority-valid.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*authority.Authority){
		func(a *authority.Authority) { a.GrantRevisionSHA256 = " " + a.GrantRevisionSHA256 },
		func(a *authority.Authority) { a.PackageSHA256 += "\n" },
		func(a *authority.Authority) { a.HandoffSHA256 = " " + a.HandoffSHA256 },
		func(a *authority.Authority) { a.InputsSHA256 += " " },
		func(a *authority.Authority) { a.ExecutorSHA256 += " " },
		func(a *authority.Authority) { a.Operations[0].ConstraintsSHA256 += " " },
		func(a *authority.Authority) { a.Operations[0].Bindings[0].Revision += " " },
	} {
		var a authority.Authority
		if err := json.Unmarshal(data, &a); err != nil {
			t.Fatal(err)
		}
		mutate(&a)
		a.PolicySHA256 = a.Digest()
		if a.Validate() == nil {
			t.Fatal("accepted noncanonical hash under a matching policy digest")
		}
	}
}
