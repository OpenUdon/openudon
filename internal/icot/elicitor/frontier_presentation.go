package elicitor

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenUdon/openudon/internal/projectwizard"
)

const (
	QuestionInputText   = "text"
	QuestionInputChoice = "choice"
)

// QuestionOption is one exact value accepted by a closed question control.
type QuestionOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// QuestionControl is engine-owned presentation metadata for one frontier
// question. It keeps workflow policy out of browser-side prompt parsing.
type QuestionControl struct {
	QuestionID string           `json:"question_id"`
	InputKind  string           `json:"input_kind"`
	Options    []QuestionOption `json:"options,omitempty"`
	Syntax     string           `json:"syntax,omitempty"`
	Deferrable bool             `json:"deferrable,omitempty"`
}

// BuildQuestionControls projects the exact current product state into
// frontend-neutral control metadata in frontier order.
func BuildQuestionControls(session Session, docs []APIDocument, frontier []QuestionPlan) []QuestionControl {
	deferrable := make(map[string]bool, len(session.Interview.Nodes))
	for _, node := range session.Interview.Nodes {
		deferrable[node.ID] = node.Deferrable
	}
	controls := make([]QuestionControl, 0, len(frontier))
	for _, question := range frontier {
		control := QuestionControl{
			QuestionID: question.ID,
			InputKind:  QuestionInputText,
			Syntax:     questionSyntax(question),
			Deferrable: deferrable[question.ID],
		}
		control.Options = questionOptions(session, docs, question)
		if len(control.Options) > 0 {
			control.InputKind = QuestionInputChoice
		}
		controls = append(controls, control)
	}
	return controls
}

func questionOptions(session Session, docs []APIDocument, question QuestionPlan) []QuestionOption {
	values := make([]string, 0)
	switch question.ID {
	case nodeActiveWorkflow:
		values = append(values, strings.TrimSpace(session.Boundary.Outcome))
		for _, candidate := range session.CandidateWorkflows {
			values = append(values, strings.TrimSpace(candidate.Title+": "+candidate.Outcome))
		}
	case nodeRemoteLookup, nodeBrowserRegistry:
		values = []string{"never", "allow"}
	case nodeBrowserSession:
		values = []string{"none", "opaque-runtime-binding-required"}
	case nodeBrowserApproval:
		if step, _, operation := selectedBrowserOperation(session, docs); step != nil && operation != nil && browserOperationMutates(operation) {
			values = []string{"approve " + step.Name}
		}
	case nodeSideEffectPosture:
		values = []string{projectwizard.SideEffectReadOnly, projectwizard.SideEffectSandboxOnly, projectwizard.SideEffectAfterApproval}
		if !stringSliceContains(values, strings.TrimSpace(question.Recommendation)) {
			values = nil
		}
	default:
		if strings.HasPrefix(question.ID, "security.alternative.") {
			if step := targetStepForPlan(&session, question); step != nil {
				if operation, ok := operationForStep(session, docs, step); ok {
					choices := securityAlternativeChoices(operation)
					options := make([]QuestionOption, 0, len(choices))
					for index, choice := range choices {
						label := strings.TrimSpace(strings.TrimPrefix(choice, strconv.Itoa(index+1)+"="))
						options = append(options, QuestionOption{Value: strconv.Itoa(index + 1), Label: fmt.Sprintf("%d — %s", index+1, label)})
					}
					return options
				}
			}
		}
		if strings.Contains(strings.Join(question.Slots, " "), "authentication_approval") {
			if step := targetStepForPlan(&session, question); step != nil {
				values = []string{"approve " + step.Name}
			}
		}
	}
	return stringOptions(values)
}

func stringOptions(values []string) []QuestionOption {
	seen := map[string]bool{}
	options := make([]QuestionOption, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		options = append(options, QuestionOption{Value: value, Label: value})
	}
	return options
}

func questionSyntax(question QuestionPlan) string {
	slots := strings.Join(question.Slots, " ")
	switch {
	case question.ID == nodeActorTrigger:
		return "Enter actor | trigger. If the trigger is omitted, it defaults to on demand."
	case question.ID == nodeSuccessEvidence || question.ID == nodeVerification:
		return "Enter one evidence item per line. Semicolon-separated items are also accepted."
	case strings.Contains(slots, ".with") || strings.Contains(slots, "credential_bindings"):
		return "Enter one name=value mapping per line. Commas and semicolons are also accepted. Use only runtime inputs, prior-step outputs, safe literals, or symbolic credential bindings."
	case strings.Contains(slots, "intent.inputs"):
		return "Enter one name:type runtime input per line. Commas and semicolons are also accepted."
	case strings.Contains(slots, "intent.outputs"):
		return "Enter one name=step.output mapping per line. Commas and semicolons are also accepted."
	case strings.Contains(slots, "authentication_flow"):
		return "Use browser-authentication/<profile>#<flow>."
	case strings.HasSuffix(slots, ".timeout"):
		return "Enter a number of seconds from 1 through 600."
	default:
		return ""
	}
}
