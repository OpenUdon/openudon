package eval

import (
	"fmt"
	"sort"
	"strings"

	"github.com/genelet/udon/pkg/rollout"
)

type CompareIssue struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func CompareIntentFiles(generatedPath, referencePath string) ([]CompareIssue, error) {
	generated, err := rollout.ParseIntentFile(generatedPath)
	if err != nil {
		return nil, fmt.Errorf("parse generated intent: %w", err)
	}
	reference, err := rollout.ParseIntentFile(referencePath)
	if err != nil {
		return nil, fmt.Errorf("parse reference intent: %w", err)
	}
	return CompareIntents(generated, reference), nil
}

func CompareIntents(generated, reference *rollout.Intent) []CompareIssue {
	var issues []CompareIssue
	issues = append(issues, compareInputs(generated, reference)...)
	issues = append(issues, compareOutputs(generated, reference)...)
	issues = append(issues, compareSteps(generated, reference)...)
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Code != issues[j].Code {
			return issues[i].Code < issues[j].Code
		}
		return issues[i].Detail < issues[j].Detail
	})
	return issues
}

func compareInputs(generated, reference *rollout.Intent) []CompareIssue {
	got := map[string]string{}
	want := map[string]string{}
	if generated != nil {
		for _, input := range generated.Inputs {
			if input != nil && strings.TrimSpace(input.Name) != "" {
				got[strings.TrimSpace(input.Name)] = strings.TrimSpace(input.Type)
			}
		}
	}
	if reference != nil {
		for _, input := range reference.Inputs {
			if input != nil && strings.TrimSpace(input.Name) != "" {
				want[strings.TrimSpace(input.Name)] = strings.TrimSpace(input.Type)
			}
		}
	}
	return compareStringMaps("intent.inputs", got, want)
}

func compareOutputs(generated, reference *rollout.Intent) []CompareIssue {
	got := map[string]string{}
	want := map[string]string{}
	if generated != nil {
		for _, output := range generated.Outputs {
			if output != nil && strings.TrimSpace(output.Name) != "" {
				got[strings.TrimSpace(output.Name)] = strings.TrimSpace(output.From)
			}
		}
	}
	if reference != nil {
		for _, output := range reference.Outputs {
			if output != nil && strings.TrimSpace(output.Name) != "" {
				want[strings.TrimSpace(output.Name)] = strings.TrimSpace(output.From)
			}
		}
	}
	return compareStringMaps("intent.outputs", got, want)
}

func compareSteps(generated, reference *rollout.Intent) []CompareIssue {
	got := stepIndex(intentSteps(generated))
	want := stepIndex(intentSteps(reference))
	var issues []CompareIssue
	for name, wantStep := range want {
		gotStep := got[name]
		if gotStep == nil {
			issues = append(issues, CompareIssue{Code: "intent.steps", Detail: fmt.Sprintf("missing step %q", name)})
			continue
		}
		if strings.TrimSpace(gotStep.Type) != strings.TrimSpace(wantStep.Type) {
			issues = append(issues, CompareIssue{Code: "intent.step_type", Detail: fmt.Sprintf("%s expected %q got %q", name, wantStep.Type, gotStep.Type)})
		}
		if strings.TrimSpace(gotStep.Operation) != strings.TrimSpace(wantStep.Operation) {
			issues = append(issues, CompareIssue{Code: "intent.step_operation", Detail: fmt.Sprintf("%s expected %q got %q", name, wantStep.Operation, gotStep.Operation)})
		}
		issues = append(issues, compareStringMaps("intent.step_with", gotStep.With, wantStep.With)...)
		gotBinds := bindMap(gotStep)
		wantBinds := bindMap(wantStep)
		issues = append(issues, compareStringMaps("intent.step_bind", gotBinds, wantBinds)...)
	}
	for name := range got {
		if want[name] == nil {
			issues = append(issues, CompareIssue{Code: "intent.steps", Detail: fmt.Sprintf("extra step %q", name)})
		}
	}
	return issues
}

func intentSteps(intent *rollout.Intent) []*rollout.Step {
	if intent == nil {
		return nil
	}
	var out []*rollout.Step
	walkSteps(intent.Steps, func(step *rollout.Step) {
		out = append(out, step)
	})
	return out
}

func walkSteps(steps []*rollout.Step, fn func(*rollout.Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		fn(step)
		walkSteps(step.Steps, fn)
		for _, branch := range step.Cases {
			if branch != nil {
				walkSteps(branch.Steps, fn)
			}
		}
		if step.Default != nil {
			walkSteps(step.Default.Steps, fn)
		}
	}
}

func stepIndex(steps []*rollout.Step) map[string]*rollout.Step {
	out := map[string]*rollout.Step{}
	for _, step := range steps {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			out[strings.TrimSpace(step.Name)] = step
		}
	}
	return out
}

func bindMap(step *rollout.Step) map[string]string {
	out := map[string]string{}
	if step == nil {
		return out
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		from := strings.TrimSpace(bind.From)
		for target, source := range bind.Fields {
			key := from + "." + strings.TrimSpace(target)
			out[key] = strings.TrimSpace(source)
		}
	}
	return out
}

func compareStringMaps(code string, got, want map[string]string) []CompareIssue {
	var issues []CompareIssue
	for key, wantValue := range want {
		gotValue, ok := got[key]
		if !ok {
			issues = append(issues, CompareIssue{Code: code, Detail: fmt.Sprintf("missing %q", key)})
			continue
		}
		if strings.TrimSpace(gotValue) != strings.TrimSpace(wantValue) {
			issues = append(issues, CompareIssue{Code: code, Detail: fmt.Sprintf("%s expected %q got %q", key, wantValue, gotValue)})
		}
	}
	for key := range got {
		if _, ok := want[key]; !ok {
			issues = append(issues, CompareIssue{Code: code, Detail: fmt.Sprintf("extra %q", key)})
		}
	}
	return issues
}
