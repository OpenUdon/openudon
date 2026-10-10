package packagev3_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/approval"
	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/packagev3"
)

func browserAuthorityFixture(t *testing.T) (packagev3.VerifiedPackage, packagev3.ExecutionOptions, packagev3.BrowserAuthorityOptions) {
	t.Helper()
	p, err := packagev3.Build(context.Background(), browserOptions(t, browserVectors(t)[4]))
	if err != nil {
		t.Fatal(err)
	}
	v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
	if err != nil {
		t.Fatal(err)
	}
	execution := packagev3.ExecutionOptions{Worker: packagev3.WorkerIdentity{BinarySHA256: strings.Repeat("a", 64), ClosureSHA256: strings.Repeat("b", 64), RuntimeRevision: strings.Repeat("c", 40)}, RuntimeAdmission: func(context.Context, packagev3.RuntimeAdmissionRequest) error { return nil }}
	plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution)
	if err != nil {
		t.Fatal(err)
	}
	supplement, err := v.BrowserSupplement()
	if err != nil {
		t.Fatal(err)
	}
	c := browsercontract.BrowserConfigV1{Version: browsercontract.ConfigVersion, OwnerID: "owner", AgentID: "agent", RunID: "run", PackageSHA256: p.SHA256, HandoffSHA256: plan.HandoffSHA256, InputsSHA256: plan.InputsSHA256, ApprovalSHA256: strings.Repeat("d", 64), PlanSHA256: plan.PlanSHA256, WorkflowSHA256: p.Manifest.Workflow.SHA256, SupplementSHA256: p.Manifest.Browser.SHA256, Worker: browsercontract.BrowserWorkerV1{Profile: "fixture.exec.browser.v1", BinarySHA256: execution.Worker.BinarySHA256, ClosureSHA256: execution.Worker.ClosureSHA256, RuntimeRevision: execution.Worker.RuntimeRevision}, DriverClosureSHA256: strings.Repeat("e", 64), LaunchNonce: "launch", ContainmentLeaseID: "contained", AdmittedDeadline: "2026-10-10T00:05:00Z", Origins: append([]string{}, supplement.Calls[0].Origins...), ApprovedCalls: supplement.Calls, CredentialRevisions: []browsercontract.CredentialRevision{}, HostFactRefs: []string{"host:current-attempt"}}
	receipt, err := approval.ReviewBrowserConfig(c, "2026-10-10T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	finalized, err := receipt.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	c.ApprovalSHA256 = browsercontract.SHA256(finalized)
	return v, execution, packagev3.BrowserAuthorityOptions{ExpectedConfig: c, FinalizedApproval: finalized, ConfirmedPlanSHA256: plan.PlanSHA256, Now: time.Date(2026, 10, 10, 0, 0, 1, 0, time.UTC)}
}
func TestBrowserAuthorityBindsVerifiedPlanAndAcyclicFinalizedReceipt(t *testing.T) {
	v, execution, o := browserAuthorityFixture(t)
	a, err := packagev3.DeriveBrowserAuthority(context.Background(), v, execution, o)
	if err != nil {
		t.Fatal(err)
	}
	if a.Validate() != nil || a.ConfigSHA256 == "" || packagev3.CheckBrowserAuthority(context.Background(), v, execution, o, a) != nil {
		t.Fatal("authority binding")
	}
	raw, _ := json.Marshal(a)
	decoded, err := browsercontract.DecodeAuthority(raw)
	if err != nil || !reflect.DeepEqual(a, decoded) {
		t.Fatal("closed authority wire", err)
	}
	a.Config.ApprovedCalls[0].Origins[0] = "https://changed.test"
	if o.ExpectedConfig.ApprovedCalls[0].Origins[0] == "https://changed.test" {
		t.Fatal("caller-owned authority")
	}
	if packagev3.CheckBrowserAuthority(context.Background(), v, execution, o, a) == nil {
		t.Fatal("mutated authority accepted")
	}
	// The finalized receipt deliberately contains no enclosing Config/Authority digest.
	var fields map[string]json.RawMessage
	json.Unmarshal(o.FinalizedApproval, &fields)
	if fields["config_sha256"] != nil || fields["approval_sha256"] != nil || fields["host_fact_refs"] != nil {
		t.Fatal("cyclic receipt")
	}
	changed := append([]byte(nil), o.FinalizedApproval...)
	changed = append(changed, ' ')
	if approval.CheckBrowserApproval(changed, o.ExpectedConfig) == nil {
		t.Fatal("changed receipt bytes accepted")
	}
}
func TestBrowserAuthorityRefusesEveryReboundIdentityAndLegacyApproval(t *testing.T) {
	cases := map[string]func(*packagev3.BrowserAuthorityOptions){
		"unconfirmed plan": func(o *packagev3.BrowserAuthorityOptions) { o.ConfirmedPlanSHA256 = strings.Repeat("f", 64) },
		"changed plan":     func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.PlanSHA256 = strings.Repeat("f", 64) },
		"changed package":  func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.PackageSHA256 = strings.Repeat("f", 64) },
		"changed input":    func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.InputsSHA256 = strings.Repeat("f", 64) },
		"changed origin":   func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.Origins[0] = "https://changed.test" },
		"changed selected action": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.ApprovedCalls[0].SelectedSHA256 = strings.Repeat("f", 64)
		},
		"changed policy": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.ApprovedCalls[0].ConfirmationPolicySHA256 = strings.Repeat("f", 64)
		},
		"changed effects": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.ApprovedCalls[0].Effects = json.RawMessage(`["creates_account"]`)
		},
		"changed worker": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.Worker.BinarySHA256 = strings.Repeat("f", 64)
		},
		"changed driver": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.DriverClosureSHA256 = strings.Repeat("f", 64)
		},
		"changed deadline": func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.AdmittedDeadline = "2026-10-10T00:06:00Z" },
		"session widening": func(o *packagev3.BrowserAuthorityOptions) { o.ExpectedConfig.ApprovedCalls[0].ReuseAllowed = true },
		"unexpected credential": func(o *packagev3.BrowserAuthorityOptions) {
			o.ExpectedConfig.CredentialRevisions = []browsercontract.CredentialRevision{{Name: "unapproved", Revision: strings.Repeat("f", 64)}}
		},
		"expired":        func(o *packagev3.BrowserAuthorityOptions) { o.Now = o.Now.Add(10 * time.Minute) },
		"implicit clock": func(o *packagev3.BrowserAuthorityOptions) { o.Now = time.Time{} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			v, execution, o := browserAuthorityFixture(t)
			mutate(&o)
			if _, err := packagev3.DeriveBrowserAuthority(context.Background(), v, execution, o); err == nil {
				t.Fatal("changed authority accepted")
			}
		})
	}
	v, execution, o := browserAuthorityFixture(t)
	legacy := approval.Approval{Version: approval.Version, Scope: "workflows/W01-browser", State: approval.StateApprovedForSandbox, Reviewer: "fixture", ApprovedAt: "2026-10-10T00:00:00Z", ExpiresAt: "2026-10-10T00:05:00Z", PackageSHA256: v.SHA256()}
	if packagev3.CheckExecutionApproval(context.Background(), v, packagev3.ExecutionApprovalOptions{Execution: execution, ExpectedPlanSHA256: o.ConfirmedPlanSHA256, Approval: legacy, Tier: approval.TierSandbox, Now: o.Now}) == nil {
		t.Fatal("old approval transferred to browser")
	}
}
func TestFrozenBrowserConfigGoldenAndClosedPrivacyBoundary(t *testing.T) {
	raw, err := os.ReadFile("testdata/browser/m51-canonical-golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		ConfigInput  json.RawMessage `json:"config_input"`
		ConfigSHA256 string          `json:"config_sha256"`
	}
	json.Unmarshal(raw, &fixture)
	c, err := browsercontract.DecodeConfig(fixture.ConfigInput)
	if err != nil {
		t.Fatal(err)
	}
	sum, err := c.Digest()
	if err != nil || sum != fixture.ConfigSHA256 {
		t.Fatal("golden config identity", err)
	}
	for _, name := range []string{"driver_path", "driver_args", "environment_names", "credential_values", "display", "staging_paths", "engine_sockets", "registration_snapshot", "registration_input_identity", "private_value_hashes"} {
		var fields map[string]json.RawMessage
		json.Unmarshal(fixture.ConfigInput, &fields)
		fields[name] = json.RawMessage(`"private"`)
		bad, _ := json.Marshal(fields)
		if _, err := browsercontract.DecodeConfig(bad); err == nil {
			t.Fatal("private field accepted", name)
		}
	}
	for name, mutate := range map[string]func(*browsercontract.BrowserConfigV1){
		"zero generation":         func(c *browsercontract.BrowserConfigV1) { c.Session.Generation = 0 },
		"unsafe generation":       func(c *browsercontract.BrowserConfigV1) { c.Session.Generation = 9007199254740992 },
		"save widening":           func(c *browsercontract.BrowserConfigV1) { c.ApprovedCalls[0].SaveAllowed = false },
		"foreign durable session": func(c *browsercontract.BrowserConfigV1) { c.Session.Name = "other" },
		"extended maximum age":    func(c *browsercontract.BrowserConfigV1) { c.Session.ExpiresAt = "2026-12-16T00:00:00Z" },
		"missing origins":         func(c *browsercontract.BrowserConfigV1) { c.Origins = nil },
		"missing witness refs":    func(c *browsercontract.BrowserConfigV1) { c.HostFactRefs = nil },
		"incompatible protocol":   func(c *browsercontract.BrowserConfigV1) { c.ApprovedCalls[0].OuterProtocol = "udon.browser-driver.v6" },
	} {
		t.Run(name, func(t *testing.T) {
			c, err := browsercontract.DecodeConfig(fixture.ConfigInput)
			if err != nil {
				t.Fatal(err)
			}
			mutate(&c)
			if c.Validate() == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}

func TestBrowserCurrentAuthorityRefusesFutureSessionCreation(t *testing.T) {
	v, execution, o := browserAuthorityFixture(t)
	// A fresh access-binding metadata object grants no reuse/save permission.
	o.ExpectedConfig.Session = &browsercontract.BrowserSessionV1{Name: "unreused-fresh-binding", BindingSHA256: strings.Repeat("e", 64), Generation: 1, CreatedAt: "2026-10-10T00:01:00Z", ExpiresAt: "2026-10-17T00:01:00Z"}
	record, err := approval.ReviewBrowserConfig(o.ExpectedConfig, "2026-10-10T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	o.FinalizedApproval, err = record.CanonicalBytes()
	if err != nil {
		t.Fatal(err)
	}
	o.ExpectedConfig.ApprovalSHA256 = browsercontract.SHA256(o.FinalizedApproval)
	if _, err := packagev3.DeriveBrowserAuthority(context.Background(), v, execution, o); err == nil {
		t.Fatal("future session accepted as current")
	}
	o.Now = time.Date(2026, 10, 10, 0, 1, 0, 0, time.UTC)
	if _, err := packagev3.DeriveBrowserAuthority(context.Background(), v, execution, o); err != nil {
		t.Fatal("creation boundary refused", err)
	}
}
