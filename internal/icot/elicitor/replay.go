package elicitor

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/genelet/ramen/internal/projectwizard"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/OpenUdon/apitools"
)

type ReplayTurn = apitools.PromptTurn

type ReplayScript struct {
	Turns []ReplayTurn `json:"turns"`
	Input string       `json:"input"`
}

func BuildReplayScript(exampleDir string, intent *rollout.Intent) (ReplayScript, error) {
	project, err := loadReplayProjectAnswers(exampleDir)
	if err != nil {
		return ReplayScript{}, err
	}
	docs, err := DiscoverLocalAPIs(exampleDir, projectwizard.Render(project))
	if err != nil {
		return ReplayScript{}, err
	}
	docByPath := map[string]APIDocument{}
	for _, doc := range docs {
		docByPath[doc.RelativePath] = doc
	}

	var script ReplayScript
	add := func(label, answer string) {
		script.Turns = append(script.Turns, ReplayTurn{Label: label, Answer: answer})
	}

	workflowName := "workflow"
	workflowGoal := "Run the workflow."
	var workflowTimeout *float64
	if intent != nil && intent.Workflow != nil {
		workflowName = intent.Workflow.Name
		workflowGoal = intent.Workflow.Description
		workflowTimeout = intent.Workflow.Timeout
	}
	add("Workflow brief", workflowGoal)
	add("Workflow name", workflowName)
	add("Workflow goal", workflowGoal)
	add("Workflow timeout seconds (blank for none)", floatReplayAnswer(workflowTimeout))
	idempotencyKey := ""
	if intent != nil && intent.Workflow != nil && intent.Workflow.Idempotency != nil {
		idempotencyKey = intent.Workflow.Idempotency.Key
	}
	add("Workflow idempotency key (blank for none)", idempotencyKey)
	if intent != nil && intent.Workflow != nil && intent.Workflow.Idempotency != nil {
		add("Workflow idempotency onConflict (blank/reject/returnPrevious)", intent.Workflow.Idempotency.OnConflict)
		add("Workflow idempotency ttl seconds (blank for none)", floatReplayAnswer(intent.Workflow.Idempotency.TTL))
	}

	usesAPI := intent != nil && intent.RequiresOpenAPI()
	add("Use OpenAPI/API steps?", yesNoReplay(usesAPI))
	if usesAPI {
		if len(docs) == 0 {
			add("OpenAPI document path or URL", intent.OpenAPI)
		} else if len(docs) > 1 {
			useDefault := strings.TrimSpace(intent.OpenAPI) != ""
			add("Use one default OpenAPI document for all API steps?", yesNoReplay(useDefault))
			if useDefault {
				add("Choose document number", intent.OpenAPI)
			}
		} else {
			add("Choose document number", firstNonEmpty(intent.OpenAPI, docs[0].RelativePath))
		}
	}

	if intent != nil {
		add("Runtime inputs (name:type, comma-separated; blank for none)", replayInputAnswer(intent.Inputs))
		for _, step := range intent.Steps {
			if err := addStepReplay(&script, step, intent.OpenAPI, docs, docByPath); err != nil {
				return ReplayScript{}, err
			}
		}
	}
	add("Step name (blank when done)", "")
	if intent != nil && len(intent.Outputs) > 0 && intent.Outputs[0] != nil {
		add("Output name", intent.Outputs[0].Name)
		add("Output source", intent.Outputs[0].From)
	} else {
		add("Output name", "result")
		add("Output source", "result")
	}
	if intent != nil && hasRuntime(intent.Steps, "cmd") {
		add("Approve cmd runtime for this project?", yesNoReplay(project.CmdApproved))
	}
	if intent != nil && hasRuntime(intent.Steps, "ssh") {
		add("Approve ssh runtime for this project?", yesNoReplay(project.SSHApproved))
	}
	scope := project.SideEffectScope
	if scope == "" {
		scope = projectwizard.InferSideEffectScope(project.Safety)
	}
	if scope == "" {
		scope = projectwizard.SideEffectSandboxOnly
	}
	add("Side-effect scope (read-only/sandbox-only/after-approval)", scope)
	add("Credential binding names only (comma-separated; blank for none)", strings.Join(project.Credentials, ", "))
	add("Safety and approval notes", project.Safety)
	add("Fallback behavior", project.Fallback)
	add("Type save, edit <slot>, explain <assumption-id>, regenerate, or cancel", "save")

	var answers []string
	for _, turn := range script.Turns {
		answers = append(answers, turn.Answer)
	}
	script.Input = strings.Join(answers, "\n") + "\n"
	return script, nil
}

func AssertReplayLabelsInOrder(output string, turns []ReplayTurn) error {
	return apitools.AssertPromptLabelsInOrder(output, turns)
}

func addStepReplay(script *ReplayScript, step *rollout.Step, defaultOpenAPI string, docs []APIDocument, docByPath map[string]APIDocument) error {
	if step == nil {
		return nil
	}
	add := func(label, answer string) {
		script.Turns = append(script.Turns, ReplayTurn{Label: label, Answer: answer})
	}
	add("Step name (blank when done)", step.Name)
	add("Step type (http/openapi/fnct/cmd/ssh)", step.Type)
	add("Step action", step.Do)
	add("Step timeout seconds (blank for none)", floatReplayAnswer(step.Timeout))
	fields := replayStepFieldNames(step)
	if step.Type == "http" || step.Type == "openapi" {
		docPath := firstNonEmpty(step.OpenAPI, defaultOpenAPI)
		if docPath == "" && len(docs) == 1 {
			docPath = docs[0].RelativePath
		}
		var required []string
		if len(docs) > 0 {
			add("Choose document number", docPath)
			add("Choose operation number", step.Operation)
			doc := docByPath[docPath]
			op := replayOperationByID(doc, step.Operation)
			required = requiredFields(op)
		} else {
			add("Operation ID", step.Operation)
			add("Required request fields (comma-separated; blank for none)", strings.Join(fields, ", "))
			required = fields
		}
		extra := replayStringSetDifference(fields, required)
		add("Additional step fields (comma-separated; blank for none)", strings.Join(extra, ", "))
		for _, field := range dedupeStrings(append(required, extra...)) {
			source, err := replayStepFieldSource(step, field)
			if err != nil {
				return err
			}
			add("Value for `"+field+"` (runtime input, literal, credential binding, or prior step output)", source)
		}
		return nil
	}
	add("Step input fields (comma-separated; blank for none)", strings.Join(fields, ", "))
	for _, field := range fields {
		source, err := replayStepFieldSource(step, field)
		if err != nil {
			return err
		}
		add("Value for `"+field+"` (runtime input, literal, credential binding, or prior step output)", source)
	}
	return nil
}

func loadReplayProjectAnswers(exampleDir string) (projectwizard.Answers, error) {
	data, err := os.ReadFile(filepath.Join(exampleDir, "project.md"))
	if err != nil {
		return projectwizard.Answers{}, err
	}
	return projectwizard.LoadAnswersFromMarkdown(string(data))
}

func replayOperationByID(doc APIDocument, operationID string) *rollout.OperationInfo {
	for _, op := range doc.Operations {
		if op != nil && op.OperationID == operationID {
			return op
		}
	}
	return nil
}

func replayInputAnswer(inputs []*rollout.Input) string {
	var parts []string
	for _, input := range inputs {
		if input == nil || input.Name == "" {
			continue
		}
		parts = append(parts, input.Name+":"+firstNonEmpty(input.Type, "string"))
	}
	return strings.Join(parts, ", ")
}

func replayStepFieldNames(step *rollout.Step) []string {
	seen := map[string]bool{}
	for field := range step.With {
		seen[field] = true
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		for field := range bind.Fields {
			seen[field] = true
		}
	}
	var out []string
	for field := range seen {
		out = append(out, field)
	}
	sort.Strings(out)
	return out
}

func replayStepFieldSource(step *rollout.Step, field string) (string, error) {
	if value := strings.TrimSpace(step.With[field]); value != "" {
		return value, nil
	}
	for _, bind := range step.Binds {
		if bind == nil {
			continue
		}
		if value := strings.TrimSpace(bind.Fields[field]); value != "" {
			return bind.From + "." + value, nil
		}
	}
	return "", fmt.Errorf("step %s has no source for field %s", step.Name, field)
}

func replayStringSetDifference(values, baseline []string) []string {
	blocked := map[string]bool{}
	for _, value := range baseline {
		blocked[value] = true
	}
	var out []string
	for _, value := range values {
		if !blocked[value] {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func yesNoReplay(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func floatReplayAnswer(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}
