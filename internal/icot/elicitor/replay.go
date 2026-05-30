package elicitor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

type ReplayTurn = authoring.PromptTurn

type ReplayScript struct {
	Turns []ReplayTurn `json:"turns"`
	Input string       `json:"input"`
}

func BuildProgressiveReplayScript(exampleDir string, intent *rollout.Intent) (ReplayScript, error) {
	project, err := loadReplayProjectAnswers(exampleDir)
	if err != nil {
		return ReplayScript{}, err
	}
	workflowGoal := strings.TrimSpace(project.Goal)
	if intent != nil && intent.Workflow != nil && strings.TrimSpace(intent.Workflow.Description) != "" {
		workflowGoal = strings.TrimSpace(intent.Workflow.Description)
	}
	if workflowGoal == "" {
		workflowGoal = "Run the workflow."
	}
	turns := []ReplayTurn{{Label: "Workflow goal", Answer: workflowGoal}}
	answers := []string{workflowGoal}
	for i := 0; i < 128; i++ {
		answers = append(answers, "")
	}
	return ReplayScript{Turns: turns, Input: strings.Join(answers, "\n") + "\n"}, nil
}

func BuildReplayScript(exampleDir string, intent *rollout.Intent) (ReplayScript, error) {
	return BuildProgressiveReplayScript(exampleDir, intent)
}

func AssertReplayLabelsInOrder(output string, turns []ReplayTurn) error {
	return authoring.AssertPromptLabelsInOrder(output, turns)
}

func loadReplayProjectAnswers(exampleDir string) (projectwizard.Answers, error) {
	data, err := os.ReadFile(filepath.Join(exampleDir, "project.md"))
	if err != nil {
		return projectwizard.Answers{}, err
	}
	return projectwizard.LoadAnswersFromMarkdown(string(data))
}
