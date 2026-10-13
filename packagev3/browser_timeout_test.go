package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/packagev3"
)

func authenticationTimeoutOptions(t *testing.T, version, timeout string) packagev3.BuildOptions {
	t.Helper()
	for _, vector := range browserVectors(t) {
		if vector.Source == "authentication-"+version+".json" {
			o := browserOptions(t, vector)
			if timeout != "" {
				o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), "    effect: write\n", "    effect: write\n    timeout: "+timeout+"\n", 1))
			}
			return o
		}
	}
	t.Fatal("missing frozen authentication source", version)
	return packagev3.BuildOptions{}
}

func TestBrowserAuthenticationTimeoutPublicProofAndPlan(t *testing.T) {
	for _, version := range []string{"1.0", "1.1"} {
		for _, timeout := range []string{"0.5", "30", "600", "0.5000", "3e1"} {
			t.Run(version+"/"+timeout, func(t *testing.T) {
				o := authenticationTimeoutOptions(t, version, timeout)
				o.DataJSON = []byte(`{"large":9007199254740993,"decimal":0.5000,"exponent":3e1,"zero":-0}`)
				p, err := packagev3.Build(context.Background(), o)
				if err != nil {
					t.Fatal("timed authentication build", err)
				}
				assessment, err := packagev3.Assess(context.Background(), p.Manifest, p.Files)
				if err != nil || !reflect.DeepEqual(assessment, p.Assessment) || assessment.Outcome == "incompatible" {
					t.Fatal("timed authentication assessment", assessment, err)
				}
				v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
				if err != nil {
					t.Fatal("timed authentication independent proof", err)
				}
				doc, raw, err := packagev3.DecodeWorkflow(context.Background(), p.Files[packagev3.WorkflowPath])
				if err != nil || doc.Operations[0].Timeout == nil || raw["operations"].([]any)[0].(map[string]any)["timeout"] != json.Number(timeout) {
					t.Fatal("authored timeout projection changed", err)
				}
				supplement, err := v.BrowserSupplement()
				if err != nil || len(supplement.Calls) != 1 || supplement.Calls[0].Kind != "authentication" || supplement.Calls[0].ProfileVersion != "uws.browser-authentication."+version || supplement.WorkflowSHA256 != browsercontract.SHA256(o.WorkflowYAML) {
					t.Fatal("timed authentication source/workflow provenance", supplement, err)
				}
				execution := executionOptions()
				if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil {
					t.Fatal("timed authentication skipped runtime admission")
				}
				admissions := 0
				// This offline adapter checks the public handoff only. It is not
				// evidence of native support, effects or consumer adoption.
				execution.RuntimeAdmission = func(ctx context.Context, r packagev3.RuntimeAdmissionRequest) error {
					admissions++
					if !bytes.Equal(r.WorkflowYAML, o.WorkflowYAML) || !bytes.Equal(r.DataJSON, o.DataJSON) || !bytes.Equal(r.Sources[o.Sources[0].Path], o.Sources[0].Bytes) || r.RuntimeRevision != execution.Worker.RuntimeRevision {
						t.Fatal("runtime admission did not receive the authored bytes")
					}
					return ctx.Err()
				}
				plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution)
				if err != nil || admissions != 1 || len(plan.Operations) != 1 || plan.Operations[0].Kind != "browser" || plan.Operations[0].Browser == nil {
					t.Fatal("timed authentication plan", plan, err)
				}
				if packagev3.CheckExecutionPlan(context.Background(), v, execution, plan) != nil || !bytes.Equal(v.Snapshot()[packagev3.WorkflowPath], o.WorkflowYAML) || !bytes.Equal(p.Files[packagev3.DataPath], o.DataJSON) {
					t.Fatal("timed proof/plan changed the exact bytes")
				}
				execution.RuntimeAdmission = func(context.Context, packagev3.RuntimeAdmissionRequest) error {
					return errors.New("independent native refusal")
				}
				if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil {
					t.Fatal("timed authentication suppressed native admission refusal")
				}
			})
		}
	}
}

func TestBrowserAuthenticationTimeoutBindsFreshPlanAndRefusesByteTamper(t *testing.T) {
	for _, version := range []string{"1.0", "1.1"} {
		t.Run(version, func(t *testing.T) {
			original := authenticationTimeoutOptions(t, version, "30")
			successor := authenticationTimeoutOptions(t, version, "31")
			v := verifiedPackage(t, original)
			changed := verifiedPackage(t, successor)
			execution := executionOptions()
			execution.RuntimeAdmission = func(ctx context.Context, _ packagev3.RuntimeAdmissionRequest) error { return ctx.Err() }
			plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution)
			if err != nil {
				t.Fatal(err)
			}
			fresh, err := packagev3.DeriveExecutionPlan(context.Background(), changed, execution)
			if err != nil {
				t.Fatal(err)
			}
			if v.SHA256() == changed.SHA256() || plan.InputsSHA256 == fresh.InputsSHA256 || plan.HandoffSHA256 == fresh.HandoffSHA256 || plan.Operations[0].ConstraintsSHA256 == fresh.Operations[0].ConstraintsSHA256 || plan.PlanSHA256 == fresh.PlanSHA256 {
				t.Fatal("timeout-only successor did not change its bound identities")
			}
			if plan.Operations[0].Browser.SelectedSHA256 != fresh.Operations[0].Browser.SelectedSHA256 || plan.Operations[0].Source != fresh.Operations[0].Source {
				t.Fatal("operation timeout changed independent source provenance")
			}
			if packagev3.CheckExecutionPlan(context.Background(), changed, execution, plan) == nil {
				t.Fatal("old plan transferred to timed successor")
			}
			tampered := v.Snapshot()
			tampered[packagev3.WorkflowPath] = successor.WorkflowYAML
			if _, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: original.Scope, ExpectedSHA256: v.SHA256(), Files: tampered}); err == nil {
				t.Fatal("timeout-only tamper retained old proof")
			}
		})
	}
}

func TestBrowserAuthenticationTimeoutInvalidBytesRefuse(t *testing.T) {
	for _, version := range []string{"1.0", "1.1"} {
		for _, timeout := range []string{"0", "-0", "-0.0", "-1", "600.0001", "601", "1e309", ".nan", ".inf", "-.inf", "NaN", "Infinity", "'30'", "true", "{}", "[]", "3e", "0x1", "1_0"} {
			t.Run(version+"/"+timeout, func(t *testing.T) {
				if _, err := packagev3.Build(context.Background(), authenticationTimeoutOptions(t, version, timeout)); err == nil {
					t.Fatal("invalid authentication timing supplied public proof")
				}
			})
		}
	}
}

func TestBrowserAuthenticationTimeoutKeepsProfileAndControlRestrictions(t *testing.T) {
	for _, c := range []struct {
		name, from, to string
	}{
		{"wrong profile", "uws.browser-authentication-call.1.0", "uws.browser-authentication-call.1.2"},
		{"case profile", "uws.browser-authentication-call.1.0", "UWS.browser-authentication-call.1.0"},
		{"mismatched source version", "uws.browser-authentication-call.1.0", "uws.browser-authentication-call.1.1"},
		{"extra operation extension", "    timeout: 30\n", "    timeout: 30\n    x-extra: {}\n"},
		{"extra authentication extension field", "      session: member\n", "      session: member\n      timeout: 30\n"},
		{"missing authentication extension", "x-uws-browser-authentication:", "x-missing-authentication:"},
		{"missing flow", "flow: member_login_push", "flow: missing"},
		{"missing source", "profile: sources/browser-profile/authentication-1.0.json", "profile: sources/browser-profile/missing.json"},
		{"missing session", "session: member", "session: ''"},
		{"missing credential binding", `credentialBindings: {"password":"fixture_password","username":"fixture_username"}`, "credentialBindings: {}"},
		{"operation when", "    timeout: 30\n", "    timeout: 30\n    when: '$inputs.ready'\n"},
		{"operation forEach", "    timeout: 30\n", "    timeout: 30\n    forEach: '$inputs.items'\n"},
		{"operation wait", "    timeout: 30\n", "    timeout: 30\n    wait: '$inputs.ready'\n"},
		{"operation parallel group", "    timeout: 30\n", "    timeout: 30\n    parallelGroup: grouped\n"},
		{"operation dependsOn", "    timeout: 30\n", "    timeout: 30\n    dependsOn: [op]\n"},
		{"operation failure", "    timeout: 30\n", "    timeout: 30\n    onFailure: [{name: retry, type: retry, retryLimit: 1}]\n"},
		{"operation success", "    timeout: 30\n", "    timeout: 30\n    onSuccess: [{name: next, type: goto, stepId: leaf}]\n"},
		{"step timing", "stepId: leaf, operationRef: op}", "stepId: leaf, operationRef: op, timeout: 30}"},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := authenticationTimeoutOptions(t, "1.0", "30")
			if !strings.Contains(string(o.WorkflowYAML), c.from) {
				t.Fatal("test mutation did not match the frozen input")
			}
			o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), c.from, c.to, 1))
			if _, err := packagev3.Build(context.Background(), o); err == nil {
				t.Fatal("timing bypassed authentication provenance or a flat control")
			}
		})
	}
	for _, vector := range browserVectors(t) {
		if vector.CallKind == "authentication" {
			continue
		}
		t.Run("timed "+vector.Name, func(t *testing.T) {
			o := browserOptions(t, vector)
			o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), "  - operationId: op\n", "  - operationId: op\n    timeout: 30\n", 1))
			if _, err := packagev3.Build(context.Background(), o); err == nil {
				t.Fatal("action or registration acquired authentication timing")
			}
		})
	}
}

func TestBrowserAuthenticationTimeoutKeepsNonBrowserAndWorkflowPlanRestrictions(t *testing.T) {
	for _, kind := range []string{"http", "function", "workflow"} {
		t.Run(kind, func(t *testing.T) {
			o := buildOptions()
			execution := executionOptions()
			switch kind {
			case "http":
				o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), "    effect: read\n", "    effect: read\n    timeout: 30\n", 1))
			case "function":
				o = runtimeOptions(t)
				execution.Worker.RuntimeRevision = runtimeRevision
				o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), "   effect: read\n", "   effect: read\n   timeout: 30\n", 1))
			case "workflow":
				o = authenticationTimeoutOptions(t, "1.0", "30")
				o.WorkflowYAML = []byte(strings.Replace(string(o.WorkflowYAML), "    type: sequence\n", "    type: sequence\n    timeout: 30\n", 1))
			}
			v := verifiedPackage(t, o)
			called := false
			execution.RuntimeAdmission = func(context.Context, packagev3.RuntimeAdmissionRequest) error { called = true; return nil }
			if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil || called {
				t.Fatal("unrelated timing reached runtime admission")
			}
		})
	}
}

func TestBrowserAuthenticationTimeoutUntimedPublicCompatibility(t *testing.T) {
	for _, version := range []string{"1.0", "1.1"} {
		t.Run(version, func(t *testing.T) {
			o := authenticationTimeoutOptions(t, version, "")
			p, err := packagev3.Build(context.Background(), o)
			if err != nil {
				t.Fatal("old untimed authentication build", err)
			}
			v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
			if err != nil {
				t.Fatal("old untimed authentication proof", err)
			}
			history, err := packagev3.InspectHistory(context.Background(), packagev3.HistoryRequest{FormatVersion: packagev3.PackageVersion, Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
			if err != nil || !history.ReadOnly || history.SourceProof {
				t.Fatal("old untimed read-only history", err)
			}
			execution := executionOptions()
			execution.RuntimeAdmission = func(_ context.Context, r packagev3.RuntimeAdmissionRequest) error {
				if !bytes.Equal(r.WorkflowYAML, o.WorkflowYAML) {
					t.Fatal("untimed metadata acquired a default timeout")
				}
				return nil
			}
			if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err != nil {
				t.Fatal("old untimed metadata behavior changed", err)
			}
			execution.RuntimeAdmission = func(context.Context, packagev3.RuntimeAdmissionRequest) error {
				return errors.New("missing native timeout")
			}
			if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil {
				t.Fatal("untimed metadata fabricated native support")
			}
			if !bytes.Equal(v.Snapshot()[packagev3.WorkflowPath], o.WorkflowYAML) {
				t.Fatal("old untimed workflow bytes changed")
			}
		})
	}
}
