package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/approval"
	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/packagev3"
)

func savedBrowserAuthorityFixture(t *testing.T) (packagev3.VerifiedPackage, packagev3.ExecutionOptions, packagev3.BrowserAuthorityOptions) {
	t.Helper()
	options := browserOptions(t, browserVectors(t)[6])
	options.Browser = &packagev3.BrowserBuildOptions{Permissions: map[string]packagev3.BrowserPermission{"leaf": {ReuseAllowed: true, SaveAllowed: true}}}
	session := &browsercontract.BrowserSessionV1{Name: "member", BindingSHA256: strings.Repeat("e", 64), Generation: 7, CreatedAt: "2026-10-09T00:00:00Z", ExpiresAt: "2026-10-10T00:00:01Z", ReuseAllowed: true, SaveAllowed: true}
	return browserAuthorityBuildFixture(t, options, session)
}

func refinalizeBrowserApproval(t *testing.T, o *packagev3.BrowserAuthorityOptions, at string) {
	t.Helper()
	receipt, err := approval.ReviewBrowserConfig(o.ExpectedConfig, at)
	if err != nil {
		t.Fatal(err)
	}
	o.FinalizedApproval, err = receipt.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	o.ExpectedConfig.ApprovalSHA256 = browsercontract.SHA256(o.FinalizedApproval)
}

func TestBrowserAuthoritySavedExpiryDoesNotExpireCurrentExecution(t *testing.T) {
	for _, tc := range []struct{ name, now string }{
		{"before", "2026-10-10T00:00:00.999999999Z"},
		{"at", "2026-10-10T00:00:01Z"},
		{"after", "2026-10-10T00:00:02Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, execution, o := savedBrowserAuthorityFixture(t)
			o.Now, _ = time.Parse(time.RFC3339Nano, tc.now)
			// This receipt is freshly finalized for this execution, including at
			// and after the original saved-state expiry. No binding is renewed.
			refinalizeBrowserApproval(t, &o, tc.now)
			original, _ := json.Marshal(o.ExpectedConfig)
			receipt := append([]byte(nil), o.FinalizedApproval...)
			a, err := packagev3.DeriveBrowserAuthority(context.Background(), v, execution, o)
			if err != nil {
				t.Fatal("saved expiry blocked current execution metadata", err)
			}
			want, _ := o.ExpectedConfig.Digest()
			if a.ConfigSHA256 != want || !reflect.DeepEqual(a.Config, o.ExpectedConfig) || packagev3.CheckBrowserAuthority(context.Background(), v, execution, o, a) != nil {
				t.Fatal("authority changed exact approved configuration")
			}
			after, _ := json.Marshal(o.ExpectedConfig)
			if !bytes.Equal(original, after) || !bytes.Equal(receipt, o.FinalizedApproval) || !a.Config.Session.ReuseAllowed || !a.Config.Session.SaveAllowed || a.Config.Session.Generation != 7 {
				t.Fatal("saved metadata or immutable permissions mutated")
			}
		})
	}
}

func TestBrowserAuthorityExpiredSavedMetadataRetainsAllRefusals(t *testing.T) {
	for name, mutate := range map[string]func(*packagev3.BrowserAuthorityOptions){
		"at run deadline": func(o *packagev3.BrowserAuthorityOptions) {
			o.Now, _ = time.Parse(time.RFC3339Nano, o.ExpectedConfig.AdmittedDeadline)
		},
		"after run deadline": func(o *packagev3.BrowserAuthorityOptions) { o.Now = o.Now.Add(10 * time.Minute) },
		"future approval":    func(o *packagev3.BrowserAuthorityOptions) { refinalizeBrowserApproval(t, o, "2026-10-10T00:00:03Z") },
		"future creation": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Session.CreatedAt, o.ExpectedConfig.Session.ExpiresAt = "2026-10-10T00:00:03Z", "2026-10-11T00:00:03Z"
			refinalizeBrowserApproval(t, o, "2026-10-10T00:00:00Z")
		},
		"malformed creation": func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.Session.CreatedAt = "invalid" },
		"malformed expiry":   func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.Session.ExpiresAt = "invalid" },
		"empty lifetime": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Session.ExpiresAt = o.ExpectedConfig.Session.CreatedAt
		},
		"overlong lifetime": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Session.ExpiresAt = "2026-12-10T00:00:00Z"
		},
		"changed generation": func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.Session.Generation++ },
		"changed binding": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Session.BindingSHA256 = strings.Repeat("f", 64)
		},
		"changed expiry": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Session.ExpiresAt = "2026-10-10T00:00:00Z"
		},
		"cleared approved permissions": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Session.ReuseAllowed, o.ExpectedConfig.Session.SaveAllowed = false, false
		},
		"changed calls": func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.ApprovedCalls[0].SaveAllowed = false },
		"changed worker": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Worker.BinarySHA256 = strings.Repeat("f", 64)
		},
		"changed plan":          func(o *packagev3.BrowserAuthorityOptions) { o.ConfirmedPlanSHA256 = strings.Repeat("f", 64) },
		"changed config hash":   func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.ApprovalSHA256 = strings.Repeat("f", 64) },
		"changed receipt bytes": func(o *packagev3.BrowserAuthorityOptions) { o.FinalizedApproval = append(o.FinalizedApproval, ' ') },
	} {
		t.Run(name, func(t *testing.T) {
			v, execution, o := savedBrowserAuthorityFixture(t)
			o.Now = o.Now.Add(time.Second)
			mutate(&o)
			if _, err := packagev3.DeriveBrowserAuthority(context.Background(), v, execution, o); err == nil {
				t.Fatal("invalid authority accepted with expired saved metadata")
			}
		})
	}
}
