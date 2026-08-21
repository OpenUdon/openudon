package browserworkflow

import (
	"reflect"
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestAnalyzeUsesConservativeDocumentOrder(t *testing.T) {
	auth := func(name string) *rollout.Step {
		return &rollout.Step{Name: "authenticate_" + name, Type: "browser_authentication", BrowserSession: name}
	}
	action := func(name string) *rollout.Step {
		return &rollout.Step{Name: "use_" + name, Type: "browser", BrowserSession: name}
	}

	before := action("before")
	nestedAfter := action("nested_after")
	conditionalAfter := action("conditional")
	conditionalInside := action("conditional")
	conditionalInside.Name = "use_inside_conditional"
	caseInside := action("after_case")
	caseInside.Name = "use_inside_case"
	caseAfter := action("after_case")
	intent := &rollout.Intent{Steps: []*rollout.Step{
		before,
		auth("nested_after"),
		{Name: "sequence", Type: "sequence", Steps: []*rollout.Step{nestedAfter}},
		{Name: "optional_auth", Type: "sequence", When: "inputs.enable", Steps: []*rollout.Step{auth("conditional"), conditionalInside}},
		conditionalAfter,
		{Name: "choice", Type: "switch", Cases: []*rollout.StepCase{{Name: "one", Steps: []*rollout.Step{auth("after_case"), caseInside}}}},
		caseAfter,
	}}

	analysis := Analyze(intent)
	if analysis.EstablishedBefore(before) {
		t.Fatal("action before authentication was treated as established")
	}
	if !analysis.EstablishedBefore(nestedAfter) {
		t.Fatal("unconditional nested action did not inherit the preceding session")
	}
	if !analysis.EstablishedBefore(conditionalInside) {
		t.Fatal("action in the same conditional sequence did not inherit its authentication")
	}
	if analysis.EstablishedBefore(conditionalAfter) {
		t.Fatal("conditional authentication escaped its branch")
	}
	if !analysis.EstablishedBefore(caseInside) {
		t.Fatal("action in the same case did not inherit its authentication")
	}
	if analysis.EstablishedBefore(caseAfter) {
		t.Fatal("case authentication escaped the switch")
	}
	wantExternal := []string{"after_case", "before", "conditional"}
	if got := analysis.ExternalSessions(); !reflect.DeepEqual(got, wantExternal) {
		t.Fatalf("external sessions = %#v, want %#v", got, wantExternal)
	}
}

func TestAnalyzeTreatsLoopAuthenticationAsNonGuaranteed(t *testing.T) {
	auth := &rollout.Step{Name: "authenticate", Type: "browser_authentication", BrowserSession: "member"}
	inside := &rollout.Step{Name: "inside", Type: "browser", BrowserSession: "member"}
	after := &rollout.Step{Name: "after", Type: "browser", BrowserSession: "member"}
	intent := &rollout.Intent{Steps: []*rollout.Step{{Name: "repeat", Type: "loop", Steps: []*rollout.Step{auth, inside}}, after}}
	analysis := Analyze(intent)
	if !analysis.EstablishedBefore(inside) {
		t.Fatal("action inside loop did not inherit authentication from its iteration")
	}
	if analysis.EstablishedBefore(after) {
		t.Fatal("loop authentication escaped a possibly empty loop")
	}
}

func TestWalkEffectiveSourcesIncludesNestedCasesAndDefaults(t *testing.T) {
	sequenceChild := &rollout.Step{Name: "sequence-child"}
	caseChild := &rollout.Step{Name: "case-child", Source: "browser-profiles/case.json"}
	defaultChild := &rollout.Step{Name: "default-child"}
	intent := &rollout.Intent{Source: "browser-profiles/root.json", Steps: []*rollout.Step{{
		Name: "parent", Source: "browser-profiles/parent.json", Steps: []*rollout.Step{sequenceChild},
		Cases:   []*rollout.StepCase{{Name: "case", Steps: []*rollout.Step{caseChild}}},
		Default: &rollout.StepDefault{Steps: []*rollout.Step{defaultChild}},
	}}}
	got := map[string]string{}
	WalkEffectiveSources(intent, func(step *rollout.Step, source string) { got[step.Name] = source })
	want := map[string]string{
		"parent": "browser-profiles/parent.json", "sequence-child": "browser-profiles/parent.json",
		"case-child": "browser-profiles/case.json", "default-child": "browser-profiles/parent.json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("effective sources = %#v, want %#v", got, want)
	}
}

func TestRuntimeOperationIDMatchesBrowserLowering(t *testing.T) {
	for input, want := range map[string]string{
		"read-dashboard":            "read_dashboard",
		"  multiple...separators  ": "multiple_separators",
		"already_valid":             "already_valid",
		"9-start":                   "_9_start",
	} {
		if got := RuntimeOperationID(input); got != want {
			t.Fatalf("RuntimeOperationID(%q) = %q, want %q", input, got, want)
		}
	}
}
