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
