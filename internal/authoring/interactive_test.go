package authoring

import (
	"context"
	"strings"
	"testing"
)

func TestPromptSessionShowsAutoAcceptedDefaultsInNormalMode(t *testing.T) {
	var out strings.Builder
	session := NewPromptSession(strings.NewReader("manual\n"), &out)
	session.SetDefaultMode(PromptDefaultsShow)

	value, err := session.AskDefault("Choose operation", "getWeather")
	if err != nil {
		t.Fatalf("AskDefault failed: %v", err)
	}
	if value != "getWeather" {
		t.Fatalf("AskDefault = %q, want default", value)
	}
	yes, err := session.AskYesNo("Use API?", true)
	if err != nil {
		t.Fatalf("AskYesNo failed: %v", err)
	}
	if !yes {
		t.Fatal("AskYesNo = false, want default true")
	}
	optional, err := session.AskOptionalDefault("Optional timeout", "")
	if err != nil {
		t.Fatalf("AskOptionalDefault failed: %v", err)
	}
	if optional != "" {
		t.Fatalf("AskOptionalDefault = %q, want blank default", optional)
	}
	required, err := session.Ask("Workflow goal")
	if err != nil {
		t.Fatalf("Ask failed: %v", err)
	}
	if required != "manual" {
		t.Fatalf("Ask = %q, want manual input", required)
	}
	output := out.String()
	for _, expected := range []string{"Choose operation [getWeather]: getWeather", "Use API? [Y/n]: yes", "Optional timeout:", "Workflow goal: "} {
		if !strings.Contains(output, expected) {
			t.Fatalf("prompt output missing %q:\n%s", expected, output)
		}
	}
	if turns := session.Turns(); len(turns) != 4 || turns[0].Answer != "getWeather" || turns[1].Answer != "yes" || turns[2].Answer != "" || turns[3].Answer != "manual" {
		t.Fatalf("turns = %#v", turns)
	}
}

func TestPromptSessionHidesAutoAcceptedDefaultsInFastMode(t *testing.T) {
	var out strings.Builder
	session := NewPromptSession(strings.NewReader("manual\n"), &out)
	session.SetDefaultMode(PromptDefaultsSilent)

	if value, err := session.AskDefault("Choose operation", "getWeather"); err != nil || value != "getWeather" {
		t.Fatalf("AskDefault = %q, %v; want default", value, err)
	}
	if yes, err := session.AskYesNo("Use API?", true); err != nil || !yes {
		t.Fatalf("AskYesNo = %v, %v; want default true", yes, err)
	}
	if optional, err := session.AskOptionalDefault("Optional timeout", ""); err != nil || optional != "" {
		t.Fatalf("AskOptionalDefault = %q, %v; want blank default", optional, err)
	}
	if required, err := session.Ask("Workflow goal"); err != nil || required != "manual" {
		t.Fatalf("Ask = %q, %v; want manual input", required, err)
	}

	output := out.String()
	for _, unexpected := range []string{"Choose operation", "Use API?", "Optional timeout"} {
		if strings.Contains(output, unexpected) {
			t.Fatalf("auto-accepted prompt %q was printed:\n%s", unexpected, output)
		}
	}
	if !strings.Contains(output, "Workflow goal: ") {
		t.Fatalf("required prompt was not printed:\n%s", output)
	}
}

func TestProgressiveLoopStopsOnBlankRequiredAnswerWithoutDefault(t *testing.T) {
	var out strings.Builder
	applyCalled := false
	_, err := RunProgressiveICOT[int, string, string](context.Background(), strings.NewReader("\n"), &out, ProgressiveLoopHooks[int, string, string]{
		Opening:     "existing goal",
		NoLLM:       true,
		MaxAttempts: 3,
		CheckReadiness: func(int, []string) []ReadinessIssue {
			return []ReadinessIssue{{Severity: "blocking", Code: "missing_operation", Message: "Choose operation.", Slot: "steps.api.operation"}}
		},
		Ready: func(int, []ReadinessIssue) bool {
			return false
		},
		PlanQuestion: func(int, []string, []ReadinessIssue) InteractiveQuestion {
			return InteractiveQuestion{Prompt: "Which operationId should api use?", Slots: []string{"steps.api.operation"}}
		},
		ApplyAnswer: func(*int, InteractiveQuestion, string, []string) error {
			applyCalled = true
			return nil
		},
	})
	if err == nil || !strings.Contains(err.Error(), "requires operator input") {
		t.Fatalf("error = %v, want operator input error", err)
	}
	if applyCalled {
		t.Fatal("blank required answer should not be applied")
	}
	if got := strings.Count(out.String(), "Which operationId should api use?"); got != 1 {
		t.Fatalf("prompt count = %d, output:\n%s", got, out.String())
	}
}
