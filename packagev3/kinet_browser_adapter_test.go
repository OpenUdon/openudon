package packagev3_test

import (
	"bytes"
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/packagev3"
	"github.com/OpenUdon/uws/binding"
)

// Adapted public fixture from the frozen Kinet W20.4 source2 Count test:
// manifest 00ca8a0ce30f06f1e85318fd0dd54de5d5b9c2655ce0e2a5211c4c404cc45126,
// browser_candidate_test.go a44d2119b7cd0e858eb79ff949aac9794ddbef5bd10fbc44bd8abc0804e51dac,
// browser_candidate.go 45655159c0641b3e3b00552d10f91897f1ca6ea2345b91d820986b2fb7db9747.
// The exact JSON and Build -> Verify -> Assess sequence are retained. The old
// expected named-output refusal is changed to compatible, with shape parity.
// This proves the adapted fixture, not actual published-module worker adoption.
func TestFrozenKinetCountAdapterAgainstP11(t *testing.T) {
	source, err := os.ReadFile("testdata/browser/action-1.10.json")
	if err != nil || browsercontract.SHA256(source) != "56b48d53a4f1e6834bccab3bb4c1027b852b46fdd5b48986642c1791890bed87" {
		t.Fatal("frozen consumer source drift", err)
	}
	shapeOptions := browsertools.BrowserShapeOptions{MaxBytes: 1 << 20, MaxOperations: 100, Sources: []browsertools.BrowserShapeSourceInput{{ID: "browser", Content: source}}}
	table, err := browsertools.BuildBrowserShapeTable(context.Background(), shapeOptions)
	if err != nil {
		t.Fatal(err)
	}
	if err := browsertools.VerifyBrowserShapeTable(context.Background(), shapeOptions, table); err != nil {
		t.Fatal("independent native shape proof", err)
	}
	resolver, err := binding.NewResolver(table)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := binding.ValidateBinding(context.Background(), resolver, binding.Request{Binding: binding.Binding{Source: table.Sources[0], SelectorKind: "id", SelectorValue: "inspect_results"}, OutputReferences: []binding.OutputReference{{Location: "body", Name: "result_count"}}})
	if err != nil || direct.Outcome != binding.Compatible {
		t.Fatal("native named-output binding", direct, err)
	}
	shapeJSON, err := table.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	for _, outputs := range []bool{false, true} {
		name := "no exports"
		if outputs {
			name = "named count export"
		}
		t.Run(name, func(t *testing.T) {
			workflow := `{"uws":"1.13.0","info":{"title":"Count fixture","version":"1"},"sourceDescriptions":[{"name":"browser","type":"browser-profile","url":"sources/browser-profile/count.json"}],"operations":[{"operationId":"op","sourceDescription":"browser","sourceOperationId":"inspect_results","effect":"read"OUTPUTS}],"workflows":[{"workflowId":"main","type":"sequence","steps":[{"stepId":"leaf","operationRef":"op"}]}]}`
			value := ""
			if outputs {
				value = `,"outputs":{"count":"$response.body#/result_count"}`
			}
			workflow = strings.Replace(workflow, "OUTPUTS", value, 1)
			options := packagev3.BuildOptions{Scope: "workflows/W01-count", WorkflowYAML: []byte(workflow), DataJSON: []byte(`{}`), Sources: []packagev3.SourceInput{{ID: "browser", Kind: "browser-profile", Path: "sources/browser-profile/count.json", Bytes: source}}}
			p, err := packagev3.Build(context.Background(), options)
			if err != nil {
				t.Fatal(err)
			}
			v, err := packagev3.Verify(context.Background(), packagev3.VerifyOptions{Scope: options.Scope, ExpectedSHA256: p.SHA256, Files: p.Files})
			if err != nil {
				t.Fatal(err)
			}
			assessment, err := packagev3.Assess(context.Background(), p.Manifest, v.Snapshot())
			if err != nil || assessment.Outcome != "compatible" || !reflect.DeepEqual(assessment, p.Assessment) || !reflect.DeepEqual(assessment, v.Assessment()) {
				t.Fatal("adapted consumer assessment disagreement", assessment, err)
			}
			if !bytes.Equal(shapeJSON, p.Files[packagev3.ShapesPath]) || !bytes.Equal(p.Files[packagev3.WorkflowPath], options.WorkflowYAML) || !bytes.Equal(p.Files[options.Sources[0].Path], source) {
				t.Fatal("consumer/native shape parity or exact input bytes changed")
			}
		})
	}
}
