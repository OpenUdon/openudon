package packagev3_test

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

func TestUnsupportedSecuritySymbolsRemainIndeterminateReviewMetadata(t *testing.T) {
	for _, name := range []string{"api/key", "review key", "κλειδί"} {
		t.Run(name, func(t *testing.T) {
			o := securityOptions("apiKey")
			o.Sources[0].Bytes = bytes.ReplaceAll(o.Sources[0].Bytes, []byte("key:"), []byte(name+":"))
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal(err)
			}
			if p.Assessment.Outcome != "indeterminate" {
				t.Fatalf("%s: %+v", p.Assessment.Outcome, p.Assessment.Findings)
			}
			if len(p.Handoff.Credentials) != 0 {
				t.Fatal("invented an addressable credential slot")
			}
			if !bytes.Equal(p.Files[o.Sources[0].Path], o.Sources[0].Bytes) {
				t.Fatal("source symbol rewritten")
			}
			table, err := binding.ParseTable(p.Files[packagev3.ShapesPath])
			if err != nil {
				t.Fatal(err)
			}
			security := table.Operations[0].Security
			if !security.Known || len(security.Alternatives) != 1 || len(security.Alternatives[0].Requirements) != 1 || security.Alternatives[0].Requirements[0].Scheme != name {
				t.Fatal("security renamed, lost or made anonymous")
			}
			a, err := packagev3.Assess(context.Background(), p.Manifest, p.Files)
			if err != nil || !reflect.DeepEqual(a, p.Assessment) {
				t.Fatal("review assessment not reproducible", err)
			}
			v := verifiedPackage(t, o)
			if v.Assessment().Outcome != "indeterminate" || len(v.CredentialNames()) != 0 {
				t.Fatal("verification changed review semantics")
			}
			plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, executionOptions())
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(plan.Operations[0].Security, security) {
				t.Fatal("plan changed original security")
			}
			for _, credentials := range []map[string]authority.Binding{nil, {name: {Name: name, Revision: strings.Repeat("a", 64), Kind: "api_key", In: "header", Parameter: "X-Api-Key"}}, {"review_key": {Name: "review_key", Revision: strings.Repeat("a", 64), Kind: "api_key", In: "header", Parameter: "X-Api-Key"}}} {
				broker := brokerOptions()
				broker.Credentials = credentials
				if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, broker); err == nil {
					t.Fatal("unbound or renamed unsupported scheme acquired authority")
				}
			}
		})
	}
}

func TestUnsupportedSecurityAlternativeNeverBecomesAnonymousReview(t *testing.T) {
	o := securityOptions("apiKey")
	o.Sources[0].Bytes = bytes.ReplaceAll(o.Sources[0].Bytes, []byte("key:"), []byte("api/key:"))
	o.Sources[0].Bytes = bytes.Replace(o.Sources[0].Bytes, []byte("security: [{api/key: []}]"), []byte("security: [{api/key: []}, {}]"), 1)
	v := verifiedPackage(t, o)
	if v.Assessment().Outcome != "indeterminate" {
		t.Fatal("unsupported alternative erased")
	}
	if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, brokerOptions()); err == nil {
		t.Fatal("OR alternatives supplied anonymous broker authority")
	}
}
