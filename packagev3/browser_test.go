package packagev3_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

type browserVector struct {
	Name, Source   string
	SHA256         string
	ProfileVersion string `json:"profile_version"`
	CallKind       string `json:"call_kind"`
	Selector       string
}

func browserVectors(t *testing.T) []browserVector {
	t.Helper()
	data, err := os.ReadFile("testdata/browser/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		KinetManifestSHA256 string            `json:"kinet_manifest_sha256"`
		FrozenFiles         map[string]string `json:"frozen_files"`
		Vectors             []browserVector
	}
	if json.Unmarshal(data, &m) != nil || m.KinetManifestSHA256 != "0c445a5c90d2c09be561e713c364747f4ab9a3698ea8ab46e7b7b774ccf16bad" || len(m.Vectors) != 11 {
		t.Fatal("invalid corpus provenance")
	}
	for file, want := range m.FrozenFiles {
		raw, err := os.ReadFile(filepath.Join("testdata/browser", file))
		if err != nil || browsercontract.SHA256(raw) != want {
			t.Fatal("frozen fixture drift", file)
		}
	}
	return m.Vectors
}
func browserOptions(t *testing.T, v browserVector) packagev3.BuildOptions {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata/browser", v.Source))
	if err != nil || browsercontract.SHA256(raw) != v.SHA256 {
		t.Fatal("source drift", v.Name)
	}
	path := "sources/browser-profile/" + v.Source
	prefix := "uws: 1.13.0\ninfo: {title: Browser fixture, version: '1'}\n"
	operation := ""
	switch v.CallKind {
	case "action":
		operation = fmt.Sprintf("sourceDescriptions: [{name: browser, type: browser-profile, url: %s}]\noperations:\n  - operationId: op\n    sourceDescription: browser\n    sourceOperationId: %s\n    effect: read\n", path, v.Selector)
	case "authentication":
		var root struct {
			CredentialSlots map[string]any `json:"credentialSlots"`
		}
		json.Unmarshal(raw, &root)
		bindings := map[string]string{}
		for slot := range root.CredentialSlots {
			bindings[slot] = "fixture_" + slot
		}
		bindingBytes, _ := json.Marshal(bindings)
		operation = fmt.Sprintf("operations:\n  - operationId: op\n    effect: write\n    x-uws-operation-profile: %s\n    x-uws-browser-authentication:\n      profile: %s\n      flow: %s\n      session: member\n      credentialBindings: %s\n", strings.Replace(v.ProfileVersion, "browser-authentication.", "browser-authentication-call.", 1), path, v.Selector, bindingBytes)
	case "registration":
		var root struct {
			CredentialSlots map[string]any `json:"credentialSlots"`
		}
		json.Unmarshal(raw, &root)
		bindings := map[string]string{}
		for slot := range root.CredentialSlots {
			bindings[slot] = "fixture_" + slot
		}
		bindingBytes, _ := json.Marshal(bindings)
		operation = fmt.Sprintf("operations:\n  - operationId: op\n    effect: write\n    x-uws-operation-profile: %s\n    x-uws-browser-registration:\n      profile: %s\n      flow: %s\n      credentialBindings: %s\n      approval: registration_fixture\n      duplicatePrevention: operator_attestation\n      onDuplicate: fail\n      ambiguousOutcome: stop_without_retry\n      cleanupDisposition: delete_separately\n", strings.Replace(v.ProfileVersion, "browser-registration.", "browser-registration-call.", 1), path, v.Selector, bindingBytes)
		if v.ProfileVersion != "uws.browser-registration.1.0" {
			operation += "      inputBinding: fixture_input\n"
		}
	}
	yaml := prefix + operation + "workflows:\n  - workflowId: main\n    type: sequence\n    steps: [{stepId: leaf, operationRef: op}]\n"
	return packagev3.BuildOptions{Scope: "workflows/W01-browser", WorkflowYAML: []byte(yaml), DataJSON: []byte(`{}`), Sources: []packagev3.SourceInput{{ID: "browser", Kind: "browser-profile", Path: path, Bytes: raw}}}
}
func TestBrowserSupplementAllNativeFamiliesAndImmutableIdentity(t *testing.T) {
	for _, v := range browserVectors(t) {
		t.Run(v.Name, func(t *testing.T) {
			options := browserOptions(t, v)
			p, err := packagev3.Build(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			if p.Manifest.Browser == nil || p.Manifest.Browser.Path != packagev3.BrowserPath {
				t.Fatal("missing supplement")
			}
			verified, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
			if err != nil {
				t.Fatal(err)
			}
			supplement, err := verified.BrowserSupplement()
			if err != nil || len(supplement.Calls) != 1 {
				t.Fatal(err)
			}
			call := supplement.Calls[0]
			if call.Kind != v.CallKind || call.ProfileVersion != v.ProfileVersion || call.SourceSHA256 != v.SHA256 || call.Selector.Key != v.Selector || call.ReuseAllowed || call.SaveAllowed {
				t.Fatal("incorrect call", call)
			}
			inputs, err := p.Manifest.InputArtifacts()
			if err != nil || len(inputs) != 5 {
				t.Fatal("input inventory", err)
			}
			table, err := binding.ParseTable(p.Files[packagev3.ShapesPath])
			if err != nil {
				t.Fatal(err)
			}
			opts := browsertools.BrowserShapeOptions{Sources: []browsertools.BrowserShapeSourceInput{{ID: "browser", Content: options.Sources[0].Bytes}}}
			if browsertools.VerifyBrowserShapeTable(context.Background(), opts, table) != nil {
				t.Fatal("not independently reproduced")
			}
			var source map[string]json.RawMessage
			json.Unmarshal(options.Sources[0].Bytes, &source)
			name := "actions"
			if v.CallKind != "action" {
				name = "flows"
			}
			var selected map[string]json.RawMessage
			json.Unmarshal(source[name], &selected)
			canonical, err := browsercontract.CanonicalJSON(selected[v.Selector])
			if err != nil || browsercontract.SHA256(canonical) != call.SelectedSHA256 {
				t.Fatal("selected whole subtree identity")
			}
			supplement.Calls[0].Origins[0] = "https://mutated.test"
			again, _ := verified.BrowserSupplement()
			if again.Calls[0].Origins[0] == "https://mutated.test" {
				t.Fatal("mutable supplement")
			}
			for _, file := range []string{packagev3.BrowserPath, packagev3.ShapesPath, options.Sources[0].Path} {
				tampered := verified.Snapshot()
				tampered[file] = append(tampered[file], ' ')
				if _, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: tampered}); err == nil {
					t.Fatal("tamper accepted", file)
				}
			}
			worker := packagev3.WorkerIdentity{BinarySHA256: strings.Repeat("a", 64), ClosureSHA256: strings.Repeat("b", 64), RuntimeRevision: strings.Repeat("c", 40)}
			// Inert schemas do not prove native runtime support; missing admission refuses.
			if _, err := packagev3.DeriveExecutionPlan(context.Background(), verified, packagev3.ExecutionOptions{Worker: worker}); err == nil {
				t.Fatal("native admission skipped")
			}
		})
	}
}
func TestBrowserCanonicalFrozenGoldenAndUnicodeRefusal(t *testing.T) {
	raw, err := os.ReadFile("testdata/browser/m51-canonical-golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var g struct {
		InputJSON           string `json:"input_json"`
		CanonicalUTF8       string `json:"canonical_utf8"`
		SHA256              string
		ConfigInput         json.RawMessage `json:"config_input"`
		ConfigCanonicalUTF8 string          `json:"config_canonical_utf8"`
		ConfigSHA256        string          `json:"config_sha256"`
	}
	json.Unmarshal(raw, &g)
	canonical, err := browsercontract.CanonicalJSON([]byte(g.InputJSON))
	if err != nil || string(canonical) != g.CanonicalUTF8 || browsercontract.SHA256(canonical) != g.SHA256 {
		t.Fatal("golden mismatch", err)
	}
	config, err := browsercontract.CanonicalJSON(g.ConfigInput)
	if err != nil || string(config) != g.ConfigCanonicalUTF8 || browsercontract.SHA256(config) != g.ConfigSHA256 {
		t.Fatal("config golden mismatch", err)
	}
	for _, bad := range []string{`{"x":1,"x":2}`, `{"x":"\ud800"}`, `{"x":"\udc00"}`, `{"x":"\ud800A"}`, "{\"x\":\"\xff\"}"} {
		if _, err := browsercontract.CanonicalJSON([]byte(bad)); err == nil {
			t.Fatal("ambiguous canonical accepted")
		}
	}
	if good, err := browsercontract.CanonicalJSON([]byte(`{"x":"\ud83d\ude00"}`)); err != nil || !bytes.Contains(good, []byte("😀")) {
		t.Fatal("valid pair refused")
	}
}

func TestBrowserShapesCannotSelfAttestAfterRehash(t *testing.T) {
	v := browserVectors(t)[4]
	p, err := packagev3.Build(context.Background(), browserOptions(t, v))
	if err != nil {
		t.Fatal(err)
	}
	table, err := binding.ParseTable(p.Files[packagev3.ShapesPath])
	if err != nil {
		t.Fatal(err)
	}
	table.Operations[0].Browser.SelectedSHA256 = strings.Repeat("f", 64)
	claimed, err := table.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	p.Files[packagev3.ShapesPath] = claimed
	p.Manifest.Shapes.SHA256 = browsercontract.SHA256(claimed)
	if _, err := packagev3.Assess(context.Background(), p.Manifest, p.Files); err == nil {
		t.Fatal("self-hashed shape accepted without source reproduction")
	}
}

func TestBrowserPlanUsesExactCallAndRequiresIndependentRuntime(t *testing.T) {
	p, err := packagev3.Build(context.Background(), browserOptions(t, browserVectors(t)[4]))
	if err != nil {
		t.Fatal(err)
	}
	if p.Assessment.Outcome != "compatible" {
		t.Fatalf("assessment: %+v", p.Assessment)
	}
	v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
	if err != nil {
		t.Fatal(err)
	}
	worker := packagev3.WorkerIdentity{BinarySHA256: strings.Repeat("a", 64), ClosureSHA256: strings.Repeat("b", 64), RuntimeRevision: strings.Repeat("c", 40)}
	admission := false
	opts := packagev3.ExecutionOptions{Worker: worker, RuntimeAdmission: func(ctx context.Context, r packagev3.RuntimeAdmissionRequest) error {
		admission = true
		if r.RuntimeRevision != worker.RuntimeRevision || !bytes.Equal(r.WorkflowYAML, p.Files[packagev3.WorkflowPath]) || len(r.Sources) != 1 {
			t.Fatal("runtime snapshot mismatch")
		}
		r.WorkflowYAML[0] = '!'
		return nil
	}}
	plan, err := packagev3.DeriveExecutionPlan(context.Background(), v, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !admission || len(plan.Operations) != 1 || plan.Operations[0].Kind != "browser" || plan.Operations[0].Browser == nil || plan.Operations[0].Method != "" || plan.Operations[0].Origin != "" {
		t.Fatal("browser plan", plan)
	}
	if v.Snapshot()[packagev3.WorkflowPath][0] != 'u' {
		t.Fatal("runtime adapter mutated retained bytes")
	}
	if err := packagev3.CheckExecutionPlan(context.Background(), v, opts, plan); err != nil {
		t.Fatal(err)
	}
	plan.Operations[0].Browser.Origins[0] = "https://changed.test"
	plan.PlanSHA256 = plan.Digest()
	if packagev3.CheckExecutionPlan(context.Background(), v, opts, plan) == nil {
		t.Fatal("rehashed changed browser plan accepted")
	}
}

func TestSelectedBrowserProfileRefusesMixedExecutionBeforeNativeAdmission(t *testing.T) {
	for _, kind := range []string{"http", "fnct"} {
		t.Run(kind, func(t *testing.T) {
			options := browserOptions(t, browserVectors(t)[4])
			yaml := string(options.WorkflowYAML)
			worker := packagev3.WorkerIdentity{BinarySHA256: strings.Repeat("a", 64), ClosureSHA256: strings.Repeat("b", 64), RuntimeRevision: strings.Repeat("c", 40)}
			if kind == "http" {
				options.Sources = append(options.Sources, packagev3.SourceInput{ID: "api", Kind: "openapi", Path: "sources/openapi/api.yaml", Bytes: []byte(apiFixture)})
				yaml = strings.Replace(yaml, "sourceDescriptions: [{name: browser, type: browser-profile, url: sources/browser-profile/action-1.9.json}]", "sourceDescriptions: [{name: browser, type: browser-profile, url: sources/browser-profile/action-1.9.json}, {name: api, type: openapi, url: sources/openapi/api.yaml}]", 1)
				yaml = strings.Replace(yaml, "workflows:", "  - operationId: mixed\n    sourceDescription: api\n    sourceOperationId: fetch\n    effect: read\n    request: {query: {n: 9007199254740993}}\nworkflows:", 1)
			} else {
				native := runtimeOptions(t)
				options.Sources = append(options.Sources, native.Sources...)
				options.RuntimeVerifier = native.RuntimeVerifier
				worker.RuntimeRevision = runtimeRevision
				yaml = strings.Replace(yaml, "workflows:", "  - operationId: mixed\n    effect: read\n    x-uws-operation-profile: uws.runtime.1.0\n    x-uws-runtime: {type: fnct, function: identity, arguments: [literal]}\nworkflows:", 1)
			}
			yaml = strings.Replace(yaml, "steps: [{stepId: leaf, operationRef: op}]", "steps: [{stepId: leaf, operationRef: op}, {stepId: mixed, operationRef: mixed}]", 1)
			options.WorkflowYAML = []byte(yaml)
			p, err := packagev3.Build(context.Background(), options)
			if err != nil {
				t.Fatal("mixed review package", err)
			}
			v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: p.Manifest.Scope, ExpectedSHA256: p.SHA256, Files: p.Files, RuntimeVerifier: options.RuntimeVerifier})
			if err != nil {
				t.Fatal("mixed read-only proof", err)
			}
			called := false
			execution := packagev3.ExecutionOptions{Worker: worker, RuntimeAdmission: func(context.Context, packagev3.RuntimeAdmissionRequest) error { called = true; return nil }}
			if _, err := packagev3.DeriveExecutionPlan(context.Background(), v, execution); err == nil || called {
				t.Fatal("unsupported mixed profile reached native admission")
			}
		})
	}
}
