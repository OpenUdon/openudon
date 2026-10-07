package packagev3_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/packagev3"
)

func executionOptions() packagev3.ExecutionOptions {
	return packagev3.ExecutionOptions{Worker: packagev3.WorkerIdentity{BinarySHA256: strings.Repeat("d", 64), ClosureSHA256: strings.Repeat("e", 64), RuntimeRevision: strings.Repeat("f", 40)}}
}
func verifiedPackage(t *testing.T, options packagev3.BuildOptions) packagev3.VerifiedPackage {
	t.Helper()
	p, err := packagev3.Build(context.Background(), options)
	if err != nil {
		t.Fatal(err)
	}
	v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files, RuntimeVerifier: options.RuntimeVerifier})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func brokerOptions() packagev3.BrokerOptions {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	return packagev3.BrokerOptions{Execution: executionOptions(), Now: now, Seed: authority.Authority{RunID: strings.Repeat("a", 32), OwnerID: "owner", AgentID: "agent", GrantID: "fresh-grant", GrantRevisionSHA256: strings.Repeat("b", 64), OccurrenceID: "fresh-once", ApprovedAt: now.Add(-time.Minute).Format(time.RFC3339), ExpiresAt: now.Add(time.Minute).Format(time.RFC3339)}}
}
func TestDeriveConcreteAuthorityBindsPackageInputsWorkerAndOperation(t *testing.T) {
	v := verifiedPackage(t, buildOptions())
	o := brokerOptions()
	a, err := packagev3.DeriveBrokerAuthority(context.Background(), v, o)
	if err != nil {
		t.Fatal(err)
	}
	if a.PackageSHA256 != v.SHA256() || a.ExecutorSHA256 != o.Execution.Worker.BinarySHA256 || len(a.Operations) != 1 || a.Operations[0].Method != "GET" || a.Operations[0].Origin != "https://fixture.example.test" {
		t.Fatal("wrong concrete authority")
	}
	if err := packagev3.CheckBrokerAuthority(context.Background(), v, o.Execution, a, nil, o.Now); err != nil {
		t.Fatal(err)
	}
	plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, o.Execution)
	if err != nil {
		t.Fatal(err)
	}
	if err := packagev3.CheckExecutionPlan(context.Background(), v, o.Execution, plan); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*packagev3.ExecutionOptions){func(o *packagev3.ExecutionOptions) { o.Worker.BinarySHA256 = strings.Repeat("c", 64) }, func(o *packagev3.ExecutionOptions) { o.Worker.ClosureSHA256 = strings.Repeat("c", 64) }, func(o *packagev3.ExecutionOptions) { o.Worker.RuntimeRevision = strings.Repeat("c", 40) }} {
		e := o.Execution
		mutate(&e)
		if packagev3.CheckBrokerAuthority(context.Background(), v, e, a, nil, o.Now) == nil || packagev3.CheckExecutionPlan(context.Background(), v, e, plan) == nil {
			t.Fatal("changed worker retained authority")
		}
	}
	for _, mutate := range []func(*authority.Authority){func(a *authority.Authority) { a.InputsSHA256 = strings.Repeat("c", 64) }, func(a *authority.Authority) { a.HandoffSHA256 = strings.Repeat("c", 64) }, func(a *authority.Authority) { a.Operations[0].Method = "DELETE" }, func(a *authority.Authority) { a.Operations[0].Origin = "https://other.example.test" }} {
		a, err := packagev3.DeriveBrokerAuthority(context.Background(), v, o)
		if err != nil {
			t.Fatal(err)
		}
		mutate(&a)
		a.PolicySHA256 = a.Digest()
		if packagev3.CheckBrokerAuthority(context.Background(), v, o.Execution, a, nil, o.Now) == nil {
			t.Fatal("rehashed forgery retained authority")
		}
	}
	changed := buildOptions()
	changed.DataJSON = []byte(`{"precise":9007199254740994}`)
	v2 := verifiedPackage(t, changed)
	if packagev3.CheckBrokerAuthority(context.Background(), v2, o.Execution, a, nil, o.Now) == nil {
		t.Fatal("changed exact data retained grant")
	}
}
func TestConcreteAuthorityRejectsPendingAmbiguousControlAndUnknownSchemas(t *testing.T) {
	for name, yaml := range map[string]string{
		"mismatch":           strings.Replace(yamlFixture, "n: 9007199254740993", "n: 9007199254740992", 1),
		"unknown expression": strings.Replace(yamlFixture, "n: 9007199254740993", "n: '$inputs.x'", 1),
		"loop":               strings.Replace(yamlFixture, "type: sequence", "type: loop\n    items: '$variables.items'", 1),
		"operation override": strings.Replace(yamlFixture, "    effect: read", "    effect: read\n    x-udon-config: {host: 'https://other.test'}", 1),
		"step values":        strings.Replace(yamlFixture, "operationRef: fetch}", "operationRef: fetch, inputs: {n: 1}}", 1),
	} {
		t.Run(name, func(t *testing.T) {
			options := buildOptions()
			options.WorkflowYAML = []byte(yaml)
			v := verifiedPackage(t, options)
			if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, brokerOptions()); !errors.Is(err, packagev3.ErrPackage) {
				t.Fatal("unsupported authority", err)
			}
		})
	}
	if _, err := packagev3.DeriveExecutionPlan(context.Background(), packagev3.VerifiedPackage{}, executionOptions()); !errors.Is(err, packagev3.ErrPackage) {
		t.Fatal("zero verification value used")
	}
}
func securityOptions(kind string) packagev3.BuildOptions {
	options := buildOptions()
	spec := strings.Replace(apiFixture, "security: []", "security: [{key: []}]", 1)
	if kind == "apiKey" {
		spec += "components:\n  securitySchemes:\n    key: {type: apiKey, in: header, name: X-API-Key}\n"
	} else {
		spec += "components:\n  securitySchemes:\n    key: {type: http, scheme: " + kind + "}\n"
	}
	options.Sources[0].Bytes = []byte(spec)
	return options
}
func TestAuthorityIndependentlyChecksSecurityAndCurrentSymbolicRevisions(t *testing.T) {
	for _, kind := range []string{"apiKey", "bearer"} {
		t.Run(kind, func(t *testing.T) {
			v := verifiedPackage(t, securityOptions(kind))
			if len(v.CredentialNames()) != 1 || v.CredentialNames()[0] != "key" {
				t.Fatal("review credential inventory")
			}
			options := brokerOptions()
			b := authority.Binding{Name: "key", Revision: strings.Repeat("a", 64), Kind: "api_key", In: "header", Parameter: "X-Api-Key"}
			if kind == "bearer" {
				b.Kind, b.Parameter = "bearer", "Authorization"
			}
			options.Credentials = map[string]authority.Binding{"key": b}
			a, err := packagev3.DeriveBrokerAuthority(context.Background(), v, options)
			if err != nil {
				t.Fatal(err)
			}
			if err := packagev3.CheckBrokerAuthority(context.Background(), v, options.Execution, a, options.Credentials, options.Now); err != nil {
				t.Fatal(err)
			}
			b.Revision = strings.Repeat("b", 64)
			if packagev3.CheckBrokerAuthority(context.Background(), v, options.Execution, a, map[string]authority.Binding{"key": b}, options.Now) == nil {
				t.Fatal("stale credential revision")
			}
			b.Revision = strings.Repeat("a", 64)
			b.Parameter = "X-Wrong"
			options.Credentials["key"] = b
			if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, options); err == nil {
				t.Fatal("forged credential placement")
			}
		})
	}
	// Basic and bearer have the same coarse shape Type=http. APItools' exact
	// native security summary must prevent a caller substituting bearer for basic.
	v := verifiedPackage(t, securityOptions("basic"))
	options := brokerOptions()
	options.Credentials = map[string]authority.Binding{"key": {Name: "key", Revision: strings.Repeat("a", 64), Kind: "bearer", In: "header", Parameter: "Authorization"}}
	if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, options); err == nil {
		t.Fatal("basic became bearer")
	}
	options = brokerOptions()
	options.Seed.PackageSHA256 = strings.Repeat("a", 64)
	if _, err := packagev3.DeriveBrokerAuthority(context.Background(), v, options); err == nil {
		t.Fatal("historical seed silently reused")
	}
}
