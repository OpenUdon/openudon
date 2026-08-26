package icot

import (
	"errors"
	"testing"
)

func TestScenarioExpectedFailuresRequireExactCause(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{name: "output bound", actual: scenarioOutputFailure("outputs_17", errors.New("output selection bound exceeded")), expected: "output_bound"},
		{name: "ambiguous output", actual: scenarioOutputFailure("ambiguous_unique_role", errors.New("required reduced scenario candidate is missing or ambiguous")), expected: "ambiguous_output"},
		{name: "stale candidate", actual: scenarioImportFailure("stale_candidate", errors.New("authenticated-authoring output selection was not requested by the operator")), expected: "stale_candidate"},
		{name: "fabricated trace", actual: scenarioImportFailure("fabricated_trace", errors.New("authenticated-authoring trace length mismatch")), expected: "fabricated_trace"},
		{name: "wrong output cause", actual: scenarioOutputFailure("outputs_17", errors.New("unrelated reconstruction failure"))},
		{name: "wrong import cause", actual: scenarioImportFailure("stale_candidate", errors.New("unrelated reconstruction failure"))},
		{name: "wrong phase", actual: scenarioImportFailure("outputs_17", errors.New("output selection bound exceeded"))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.actual != test.expected {
				t.Fatalf("failure class = %q, want %q", test.actual, test.expected)
			}
		})
	}
}

func TestScenarioFrameResponseUsesReviewedContext(t *testing.T) {
	observation := liveObservation{Context: "main", Contexts: map[string]liveContext{
		"frame_7": {Kind: "frame", Parent: "main", Origin: "https://app.example.test", Path: "/embedded"},
	}}
	response, err := scenarioFrameResponse("frame_7", observation)
	if err != nil || response.Kind != "observe" || response.Context != "frame_7" {
		t.Fatalf("frame response = %#v, %v", response, err)
	}
	if _, err := scenarioFrameResponse("frame_1", observation); err == nil {
		t.Fatal("unreviewed frame context was accepted")
	}
}
