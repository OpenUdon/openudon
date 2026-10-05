package trustedrunner

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/brokerhandoff"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/udonreport"
	"github.com/OpenUdon/openudon/internal/udonrunner"
)

const brokerAPI = `openapi: 3.0.3
info: {title: Broker fixture, version: 1.0.0}
servers: [{url: 'https://service.test'}]
security: [{token: []}]
components:
  securitySchemes:
    token: {type: http, scheme: bearer}
paths:
  /value:
    get: {operationId: get, responses: {'200': {description: OK}}}
    post: {operationId: post, responses: {'200': {description: OK}}}
`

func fixtureBroker(t *testing.T) (Options, Approval, string, net.Listener) {
	t.Helper()
	return fixtureBrokerAPI(t, brokerAPI)
}

func fixtureBrokerAPI(t *testing.T, api string) (Options, Approval, string, net.Listener) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Unix broker profile")
	}
	root, example := writeFixture(t, fixtureOptions{extraRequiredInputs: []string{"openapi/service.yaml"}, credentialBindings: []string{"token"}})
	mustWriteFile(t, filepath.Join(example, "workflows/workflow.uws.yaml"), []byte(v5Workflow))
	mustWriteFile(t, filepath.Join(example, "openapi/service.yaml"), []byte(api))
	approvalPath := writeApprovalTemplate(t, root, example, StateApprovedForSandbox, fixedNow())
	a, _, err := readApprovalDocument(approvalPath)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(root, "executor")
	mustWriteFile(t, binary, []byte("#!/bin/sh\nexit 0\n"))
	if err := os.Chmod(binary, 0700); err != nil {
		t.Fatal(err)
	}
	legacy, err := Run(context.Background(), Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath, DryRun: true, ExecutorReportVersion: "v5", Now: fixedNow(), Assess: passAssess})
	if err != nil {
		t.Fatal(err)
	}
	c, err := udonrunner.LoadConfig(legacy.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	wf, err := os.ReadFile(filepath.Join(example, c.WorkflowPath))
	if err != nil {
		t.Fatal(err)
	}
	runID := "0123456789abcdef0123456789abcdef"
	inv, err := udonreport.InventoryFromWorkflowV5(wf, "uws-yaml", runID)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := udonrunner.InspectBrokerPlan(legacy.StagePath, filepath.Join(legacy.StagePath, c.WorkflowPath), "uws-yaml", c.APISourcePaths, inv)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	a.Version = BrokerApprovalVersion
	a.ExpiresAt = fixedNow()().Add(time.Hour).Format(time.RFC3339)
	a.Broker = &brokerhandoff.Authority{Version: brokerhandoff.Version, RunID: runID, OwnerID: "owner", AgentID: "agent", GrantID: "one-off", GrantRevisionSHA256: strings.Repeat("a", 64), OccurrenceID: "occurrence", PackageSHA256: a.PackageSHA256, HandoffSHA256: c.HandoffSHA256, InputsSHA256: plan.InputsSHA256, ExecutorSHA256: evidencefile.SHA256(data), ApprovedAt: a.ApprovedAt, ExpiresAt: a.ExpiresAt, Operations: plan.Operations}
	for i := range a.Broker.Operations {
		for j := range a.Broker.Operations[i].Bindings {
			a.Broker.Operations[i].Bindings[j].Revision = strings.Repeat("b", 64)
		}
	}
	a.Broker.PolicySHA256 = a.Broker.Digest()
	writeApprovalFile(t, approvalPath, a)
	privateDir, err := os.MkdirTemp("", "m97-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(privateDir) })
	socket := filepath.Join(privateDir, "broker.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	if err := os.Chmod(socket, 0600); err != nil {
		t.Fatal(err)
	}
	private := brokerhandoff.PrivateConfig{Version: brokerhandoff.TransportVersion, SocketPath: socket, Capability: strings.Repeat("c", 64), RunID: runID, PolicyDigest: a.Broker.PolicySHA256}
	for _, op := range a.Broker.Operations {
		private.Invocations = append(private.Invocations, brokerhandoff.PrivateInvocation{StepID: op.StepID, OperationID: op.OperationID, InvocationID: op.InvocationID, Bindings: op.Bindings})
	}
	data, _ = json.Marshal(private)
	privatePath := filepath.Join(privateDir, "private.json")
	if err := os.WriteFile(privatePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	return Options{RepoRoot: root, ExampleDir: example, Tier: TierSandbox, ApprovalPath: approvalPath, WorkDir: filepath.Join(root, "broker-runs"), BrokerConfigPath: privatePath, Env: []string{"OPENUDON_EXECUTOR=" + binary, "UDON_CREDENTIAL_TOKEN=SECRET_CANARY", "HTTP_PROXY=http://proxy.invalid"}, Now: fixedNow(), Assess: passAssess}, a, privatePath, listener
}

func TestBrokerRunPrivateHandoffAndUncertainReplay(t *testing.T) {
	opts, approval, privatePath, _ := fixtureBroker(t)
	calls := 0
	transportSnapshot := ""
	opts.Invoke = func(_ context.Context, call udonrunner.Invocation) error {
		calls++
		snapshot := argValue(t, call.Argv, "--http-broker-config")
		transportSnapshot = snapshot
		if snapshot == privatePath {
			t.Fatal("mutable source transport used directly")
		}
		if _, err := brokerhandoff.ReadPrivate(snapshot, *approval.Broker); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.Join(call.Env, "\n"), "SECRET_CANARY") || strings.Contains(strings.Join(call.Env, "\n"), "HTTP_PROXY") {
			t.Fatal("host environment crossed worker boundary")
		}
		if argValue(t, call.Argv, "--execution-run-id") != "0123456789abcdef0123456789abcdef" {
			t.Fatal("run identity changed")
		}
		return errors.New("lost executor reply")
	}
	result, err := Run(context.Background(), opts)
	if err == nil || result == nil || result.RunEvidencePath == "" || calls != 1 {
		t.Fatalf("missing durable uncertainty: %v", err)
	}
	e := readRunEvidenceFile(t, result.RunEvidencePath)
	if e.Version != BrokerRunEvidenceVersion || e.Broker == nil || e.StepExecution.State != "missing" {
		t.Fatal("missing broker uncertainty")
	}
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(result.RunEvidencePath)
	if strings.Contains(string(data), privatePath) || strings.Contains(string(data), strings.Repeat("c", 64)) || strings.Contains(string(data), "SECRET_CANARY") {
		t.Fatal("private transport or values leaked")
	}
	for _, artifact := range []string{result.RunEvidencePath, result.AsyncEvidencePath} {
		data, err := os.ReadFile(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), privatePath) || strings.Contains(string(data), transportSnapshot) || strings.Contains(string(data), strings.Repeat("c", 64)) || strings.Contains(string(data), "SECRET_CANARY") {
			t.Fatal("private transport reference leaked into portable evidence")
		}
	}
	before := append([]byte(nil), data...)
	if _, err := Run(context.Background(), opts); err == nil || calls != 1 {
		t.Fatal("uncertain run replayed")
	}
	data, _ = os.ReadFile(result.RunEvidencePath)
	if string(before) != string(data) {
		t.Fatal("replay replaced original evidence")
	}
	archive, err := ArchiveRunEvidence(ArchiveOptions{RunEvidencePath: result.RunEvidencePath, ArchiveDir: filepath.Join(opts.RepoRoot, "archive")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyRunEvidenceFile(archive.RunEvidencePath); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerRefusesBeforeExecutorInvocation(t *testing.T) {
	for _, name := range []string{"legacy-version", "missing-private", "public-private", "public-socket", "wrong-run", "stale-package", "stale-handoff", "stale-input", "wrong-method", "wrong-origin", "wrong-constraints", "wrong-executor", "missing-binding", "public-sandbox", "expired"} {
		t.Run(name, func(t *testing.T) {
			opts, a, private, _ := fixtureBroker(t)
			switch name {
			case "legacy-version":
				a.Version = ApprovalVersion
			case "missing-private":
				opts.BrokerConfigPath = ""
			case "public-private":
				if err := os.Chmod(private, 0644); err != nil {
					t.Fatal(err)
				}
			case "public-socket":
				data, _ := os.ReadFile(private)
				var c brokerhandoff.PrivateConfig
				if err := json.Unmarshal(data, &c); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(c.SocketPath, 0644); err != nil {
					t.Fatal(err)
				}
			case "wrong-run":
				a.Broker.RunID = "ffffffffffffffff"
			case "stale-package":
				a.Broker.PackageSHA256 = strings.Repeat("f", 64)
			case "stale-handoff":
				a.Broker.HandoffSHA256 = strings.Repeat("f", 64)
			case "stale-input":
				a.Broker.InputsSHA256 = strings.Repeat("f", 64)
			case "wrong-method":
				a.Broker.Operations[0].Method = "DELETE"
			case "wrong-origin":
				a.Broker.Operations[0].Origin = "https://other.test"
			case "wrong-constraints":
				a.Broker.Operations[0].ConstraintsSHA256 = strings.Repeat("f", 64)
			case "wrong-executor":
				a.Broker.ExecutorSHA256 = strings.Repeat("f", 64)
			case "missing-binding":
				for i := range a.Broker.Operations {
					a.Broker.Operations[i].Bindings = nil
				}
			case "public-sandbox":
				a.Broker.Operations[0].Origin = "https://public.example.net.attacker.com"
			case "expired":
				opts.Now = func() time.Time { return fixedNow()().Add(2 * time.Hour) }
			}
			a.Broker.PolicySHA256 = a.Broker.Digest()
			writeApprovalFile(t, opts.ApprovalPath, a)
			// Match transport to changed authority so these refusals prove the
			// reviewed-package checks, rather than merely a stale private policy.
			data, _ := os.ReadFile(private)
			var c brokerhandoff.PrivateConfig
			if err := json.Unmarshal(data, &c); err != nil {
				t.Fatal(err)
			}
			if name != "wrong-run" {
				c.RunID = a.Broker.RunID
			}
			c.PolicyDigest = a.Broker.PolicySHA256
			for i, op := range a.Broker.Operations {
				c.Invocations[i] = brokerhandoff.PrivateInvocation{StepID: op.StepID, OperationID: op.OperationID, InvocationID: op.InvocationID, Bindings: op.Bindings}
			}
			data, _ = json.Marshal(c)
			if err := os.WriteFile(private, data, 0600); err != nil {
				t.Fatal(err)
			}
			called := false
			opts.Invoke = func(context.Context, udonrunner.Invocation) error { called = true; return nil }
			if _, err := Run(context.Background(), opts); err == nil || called {
				t.Fatalf("invalid authority invoked executor: %v", err)
			}
		})
	}
}

func TestBrokerInspectionApprovalAndExternalBoundary(t *testing.T) {
	opts, approval, privatePath, _ := fixtureBroker(t)
	inspection, err := InspectBrokerPackage(context.Background(), TemplateOptions{RepoRoot: opts.RepoRoot, ExampleDir: opts.ExampleDir, Assess: passAssess})
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Version != "openudon.broker-package.v1" || inspection.Plan.InputsSHA256 != approval.Broker.InputsSHA256 || len(inspection.Plan.Requests) != 2 {
		t.Fatal("missing exact review metadata")
	}
	for i, request := range inspection.Plan.Requests {
		if evidencefile.SHA256(request.Constraints) != inspection.Plan.Operations[i].ConstraintsSHA256 || request.PathTemplate != "/value" {
			t.Fatal("review constraint drift")
		}
	}
	template, err := ApprovalTemplate(context.Background(), TemplateOptions{RepoRoot: opts.RepoRoot, ExampleDir: opts.ExampleDir, State: approval.State, Reviewer: approval.Reviewer, Broker: approval.Broker, Now: fixedNow(), Assess: passAssess})
	if err != nil || template.Version != BrokerApprovalVersion {
		t.Fatalf("broker template refused: %v", err)
	}
	opts.DryRun = true
	result, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyRunEvidenceFile(result.RunEvidencePath); err != nil {
		t.Fatal(err)
	}
	configBytes, err := os.ReadFile(result.RunConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	external := ExternalOptions{ConfigPath: result.RunConfigPath, ConfigSHA256: evidencefile.SHA256(configBytes), ApprovalPath: opts.ApprovalPath, BrokerConfigPath: privatePath, Env: opts.Env, Now: fixedNow(), Assess: passAssess, Invoke: func(_ context.Context, call udonrunner.Invocation) error {
		calls++
		if _, err := brokerhandoff.ReadPrivate(argValue(t, call.Argv, "--http-broker-config"), *approval.Broker); err != nil {
			t.Fatal(err)
		}
		return errors.New("synthetic interrupted external executor")
	}}
	if _, err := RunExternal(context.Background(), external); err == nil || calls != 1 {
		t.Fatalf("external boundary not invoked once: %v", err)
	}
	if _, err := RunExternal(context.Background(), external); err == nil || calls != 1 {
		t.Fatal("external boundary replayed uncertain run")
	}
}

// Only a concrete production approval admits a reviewed public origin. The
// injected executor never contacts that origin; these are local boundary tests.
func TestBrokerProductionApprovalPreservesSandboxBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, tier, state string
		invoked           bool
	}{
		{"approved-production", TierProduction, StateApprovedForProduction, true},
		{"sandbox-public-refusal", TierSandbox, StateApprovedForProduction, false},
		{"sandbox-approval-refusal", TierProduction, StateApprovedForSandbox, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts, old, privatePath, _ := fixtureBroker(t)
			mustWriteFile(t, filepath.Join(opts.ExampleDir, "openapi/service.yaml"), []byte(strings.ReplaceAll(brokerAPI, "https://service.test", "https://api.customer.net")))
			writeApprovalTemplate(t, opts.RepoRoot, opts.ExampleDir, tc.state, fixedNow())
			approval, _, err := readApprovalDocument(opts.ApprovalPath)
			if err != nil {
				t.Fatal(err)
			}
			inspection, err := InspectBrokerPackage(context.Background(), TemplateOptions{RepoRoot: opts.RepoRoot, ExampleDir: opts.ExampleDir, Assess: passAssess})
			if err != nil {
				t.Fatal(err)
			}
			authority := *old.Broker
			authority.PackageSHA256, authority.HandoffSHA256 = inspection.PackageSHA256, inspection.HandoffSHA256
			authority.Operations = inspection.Plan.Operations
			for i := range authority.Operations {
				authority.Operations[i].Bindings = append([]brokerhandoff.Binding(nil), old.Broker.Operations[i].Bindings...)
			}
			authority.PolicySHA256 = authority.Digest()
			approval.Version, approval.Broker, approval.ExpiresAt = BrokerApprovalVersion, &authority, authority.ExpiresAt
			writeApprovalFile(t, opts.ApprovalPath, approval)
			data, err := os.ReadFile(privatePath)
			if err != nil {
				t.Fatal(err)
			}
			var private brokerhandoff.PrivateConfig
			if err := json.Unmarshal(data, &private); err != nil {
				t.Fatal(err)
			}
			private.PolicyDigest = authority.PolicySHA256
			data, err = json.Marshal(private)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(privatePath, data, 0600); err != nil {
				t.Fatal(err)
			}
			opts.Tier = tc.tier
			calls := 0
			opts.Invoke = func(context.Context, udonrunner.Invocation) error { calls++; return errors.New("synthetic lost reply") }
			result, err := Run(context.Background(), opts)
			if err == nil {
				t.Fatal("expected refusal or synthetic uncertainty")
			}
			if (calls == 1) != tc.invoked || calls > 1 {
				t.Fatalf("production boundary invoked %d times", calls)
			}
			if tc.invoked {
				if result == nil || result.RunEvidencePath == "" {
					t.Fatal("production uncertainty not persisted")
				}
				e := readRunEvidenceFile(t, result.RunEvidencePath)
				if e.Tier != TierProduction || e.Broker == nil || e.Version != BrokerRunEvidenceVersion {
					t.Fatal("production broker evidence mismatch")
				}
			}
		})
	}
}

func TestBrokerAPIKeyMetadataReachesPrivateExecutorBinding(t *testing.T) {
	for _, tc := range []struct{ in, parameter string }{{"header", "X-Api-Key"}, {"query", "api_key"}} {
		t.Run(tc.in, func(t *testing.T) {
			api := strings.ReplaceAll(brokerAPI, "{type: http, scheme: bearer}", "{type: apiKey, in: "+tc.in+", name: "+tc.parameter+"}")
			opts, authority, _, _ := fixtureBrokerAPI(t, api)
			calls := 0
			opts.Invoke = func(_ context.Context, call udonrunner.Invocation) error {
				calls++
				private, err := brokerhandoff.ReadPrivate(argValue(t, call.Argv, "--http-broker-config"), *authority.Broker)
				if err != nil {
					t.Fatal(err)
				}
				for _, inv := range private.Invocations {
					if len(inv.Bindings) != 1 || inv.Bindings[0].Kind != "api_key" || inv.Bindings[0].In != tc.in || inv.Bindings[0].Parameter != tc.parameter {
						t.Fatal("APItools-backed API-key metadata drift")
					}
				}
				if strings.Contains(strings.Join(call.Env, "\n"), "SECRET_CANARY") {
					t.Fatal("credential value crossed API-key worker boundary")
				}
				return errors.New("synthetic lost reply")
			}
			r, err := Run(context.Background(), opts)
			if err == nil || r == nil || r.RunEvidencePath == "" || calls != 1 {
				t.Fatalf("API-key handoff failed: %v", err)
			}
		})
	}
}
