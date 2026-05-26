package icot

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/icot/elicitor"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func runRepair(args []string, out, errOut io.Writer) int {
	fs := flag.NewFlagSet("icot repair", flag.ContinueOnError)
	fs.SetOutput(out)
	example := fs.String("example", "", "Example directory containing project.md and workflows/intent.hcl")
	dryRun := fs.Bool("dry-run", false, "Report proposed repairs without writing files")
	jsonOutput := fs.Bool("json", false, "Write a structured JSON report to stdout")
	reportPath := fs.String("report", "", "Write a structured JSON report to this path")
	maxAttempts := fs.Int("max-attempts", 2, "Maximum bounded repair attempts")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: icot repair --example examples/<name> [--dry-run] [--json]\n\n")
		fmt.Fprintf(fs.Output(), "Applies bounded iCoT repairs for request mappings, output sources, and depends_on only.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	exampleDir := strings.TrimSpace(*example)
	if exampleDir == "" {
		fmt.Fprintln(errOut, "--example is required")
		return 2
	}
	report := runRepairExample(exampleDir, *dryRun, *maxAttempts)
	if strings.TrimSpace(*reportPath) != "" {
		if err := writeJSONFile(*reportPath, report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	}
	if *jsonOutput {
		if err := json.NewEncoder(out).Encode(report); err != nil {
			fmt.Fprintln(errOut, err)
			return 1
		}
	} else {
		printRepairReport(out, report)
	}
	if report.Status == statusFail {
		return 1
	}
	return 0
}

func runRepairExample(exampleDir string, dryRun bool, maxAttempts int) repairReport {
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	report := repairReport{
		Version:     repairReportVersion,
		Status:      statusPass,
		Example:     exampleDir,
		DryRun:      dryRun,
		MaxAttempts: maxAttempts,
	}
	session, err := loadCurrentExampleSession(exampleDir)
	if err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureIntentParse
		return report
	}
	for attempt := 0; attempt < maxAttempts; attempt++ {
		report.Attempts++
		changed, rejected := applyDeterministicRepairs(exampleDir, &session, &report)
		report.Rejected = append(report.Rejected, rejected...)
		if !changed {
			break
		}
	}
	artifacts, err := elicitor.RenderArtifacts(session)
	if err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureIntentParse
		return report
	}
	if dryRun {
		report.Status = statusDryRun
		report.GeneratedProject = filepath.Join(exampleDir, "project.md")
		report.GeneratedIntent = filepath.Join(exampleDir, "workflows", "intent.hcl")
		return report
	}
	projectPath := filepath.Join(exampleDir, "project.md")
	intentPath := filepath.Join(exampleDir, "workflows", "intent.hcl")
	if err := writeGeneratedFilesAtomic([]generatedFile{
		{Path: projectPath, Content: artifacts.ProjectMD},
		{Path: intentPath, Content: artifacts.IntentHCL},
	}, true); err != nil {
		report.Status = statusFail
		report.Error = err.Error()
		report.FailureFamily = failureIntentParse
		return report
	}
	report.GeneratedProject = projectPath
	report.GeneratedIntent = intentPath
	_, quality, buildErr := synthesize.PackageFromIntent(context.Background(), synthesize.Options{ExampleDir: exampleDir})
	report.QualityReport = filepath.Join(exampleDir, "expected", "quality.json")
	if buildErr != nil {
		report.Status = statusFail
		report.Error = buildErr.Error()
		report.FailureCodes = []string{"build:error"}
		report.FailureFamily = failureFamilyForDetail(buildErr.Error())
		return report
	}
	if quality != nil && !quality.Passed() {
		report.Status = statusFail
		report.FailureCodes = scorecardFailedQualityCodes(quality)
		report.FailureFamily = failureFamilyForQualityCode(firstFailedQualityCode(quality))
		return report
	}
	return report
}

func loadCurrentExampleSession(exampleDir string) (elicitor.Session, error) {
	projectPath := filepath.Join(exampleDir, "project.md")
	intentPath := filepath.Join(exampleDir, "workflows", "intent.hcl")
	var project projectwizard.Answers
	if data, err := os.ReadFile(projectPath); err == nil {
		loaded, err := projectwizard.LoadAnswersFromMarkdown(string(data))
		if err != nil {
			return elicitor.Session{}, err
		}
		project = loaded
	} else {
		return elicitor.Session{}, err
	}
	intent, err := rollout.ParseIntentFile(intentPath)
	if err != nil {
		return elicitor.Session{}, err
	}
	return elicitor.SessionFromIntent(intent, project), nil
}

func applyDeterministicRepairs(exampleDir string, session *elicitor.Session, report *repairReport) (bool, []string) {
	if session == nil {
		return false, nil
	}
	projectText := projectwizard.Render(session.Project)
	docs, _ := elicitor.DiscoverLocalAPIs(exampleDir, projectText)
	var changed bool
	var rejected []string
	if depChanges := repairDependsOnFromReferences(session); len(depChanges) > 0 {
		changed = true
		report.Applied = append(report.Applied, depChanges...)
	}
	issues := elicitor.CheckReadiness(*session, docs)
	for _, issue := range issues {
		switch issue.Code {
		case "missing_required_request_values":
			ok, reject := repairRequestMappingFromSuggestion(session, docs, issue)
			if ok {
				changed = true
				report.Applied = append(report.Applied, repairChange{Slot: issue.Slot, After: issue.SuggestedAnswer, Reason: issue.Message})
			} else if reject != "" {
				rejected = append(rejected, reject)
			}
		case "missing_outputs":
			ok, reject := repairOutputFromSuggestion(session, issue)
			if ok {
				changed = true
				report.Applied = append(report.Applied, repairChange{Slot: issue.Slot, After: issue.SuggestedAnswer, Reason: issue.Message})
			} else if reject != "" {
				rejected = append(rejected, reject)
			}
		case "conflicting_decision_evidence", "low_confidence_decision", "missing_operation", "missing_api_doc", "missing_credential_bindings", "missing_side_effect_policy":
			rejected = append(rejected, issue.Slot+" ("+issue.Code+" outside repair scope)")
		}
	}
	return changed, dedupeRepairStrings(rejected)
}

func repairRequestMappingFromSuggestion(session *elicitor.Session, docs []elicitor.APIDocument, issue elicitor.ReadinessIssue) (bool, string) {
	stepName := strings.TrimSuffix(strings.TrimPrefix(issue.Slot, "steps."), ".with")
	if stepName == issue.Slot || strings.TrimSpace(stepName) == "" {
		return false, issue.Slot + " (missing step name)"
	}
	step := repairStepByName(session.Intent.Steps, stepName)
	if step == nil {
		return false, issue.Slot + " (unknown step)"
	}
	assignments := parseRepairAssignments(issue.SuggestedAnswer)
	if len(assignments) == 0 {
		return false, issue.Slot + " (no suggested assignments)"
	}
	request := elicitor.BuildRequestMappingRequest("", *session, docs, []elicitor.ReadinessIssue{issue}, elicitor.QuestionPlan{Slots: []string{issue.Slot}})
	var allowed []string
	for _, requestStep := range request.Steps {
		if requestStep.Name == stepName {
			allowed = append(allowed, requestStep.MissingFields...)
			break
		}
	}
	if len(allowed) == 0 {
		return false, issue.Slot + " (no declared request fields)"
	}
	normalized := map[string]string{}
	for field, source := range assignments {
		if !safeRepairSource(source) {
			return false, issue.Slot + " (unsafe source)"
		}
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		normalizedField, ok := elicitor.NormalizeRequestMappingField(field, allowed)
		if !ok {
			return false, issue.Slot + " (unknown request field " + field + ")"
		}
		normalized[normalizedField] = source
	}
	if step.With == nil {
		step.With = map[string]string{}
	}
	changed := false
	for field, source := range normalized {
		if strings.TrimSpace(step.With[field]) == "" {
			step.With[field] = source
			changed = true
		}
	}
	return changed, ""
}

func repairDependsOnFromReferences(session *elicitor.Session) []repairChange {
	if session == nil {
		return nil
	}
	known := map[string]bool{}
	for _, step := range repairFlattenSteps(session.Intent.Steps) {
		if step != nil && strings.TrimSpace(step.Name) != "" {
			known[step.Name] = true
		}
	}
	var changes []repairChange
	for _, step := range repairFlattenSteps(session.Intent.Steps) {
		if step == nil || strings.TrimSpace(step.Name) == "" {
			continue
		}
		before := strings.Join(step.DependsOn, ",")
		deps := append([]string(nil), step.DependsOn...)
		for _, source := range step.With {
			if dep := repairStepReference(source, known); dep != "" && dep != step.Name {
				deps = append(deps, dep)
			}
		}
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			if dep := strings.TrimSpace(bind.From); dep != "" && known[dep] && dep != step.Name {
				deps = append(deps, dep)
			}
			for _, source := range bind.Fields {
				if dep := repairStepReference(source, known); dep != "" && dep != step.Name {
					deps = append(deps, dep)
				}
			}
		}
		deps = repairKnownDependencies(deps, known, step.Name)
		after := strings.Join(deps, ",")
		if before == after {
			continue
		}
		step.DependsOn = deps
		changes = append(changes, repairChange{
			Slot:   "steps." + step.Name + ".depends_on",
			Before: before,
			After:  after,
			Reason: "Added dependency wiring from existing step output references.",
		})
	}
	return changes
}

func repairOutputFromSuggestion(session *elicitor.Session, issue elicitor.ReadinessIssue) (bool, string) {
	assignments := parseRepairAssignments(issue.SuggestedAnswer)
	name := "result"
	from := ""
	for key, value := range assignments {
		if strings.EqualFold(key, "output") || strings.EqualFold(key, "name") {
			name = value
			continue
		}
		if strings.EqualFold(key, "from") {
			from = value
			continue
		}
		name = key
		from = value
	}
	if strings.TrimSpace(from) == "" {
		return false, issue.Slot + " (no output source)"
	}
	if !safeRepairSource(from) {
		return false, issue.Slot + " (unsafe output source)"
	}
	for _, output := range session.Intent.Outputs {
		if output != nil && output.Name == name {
			if output.From == from {
				return false, ""
			}
			output.From = from
			return true, ""
		}
	}
	session.Intent.Outputs = append(session.Intent.Outputs, &rollout.Output{Name: name, From: from})
	return true, ""
}

func parseRepairAssignments(value string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.FieldsFunc(value, func(r rune) bool { return r == '\n' || r == ',' || r == ';' }) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "=") {
			pieces := strings.SplitN(part, "=", 2)
			key := strings.TrimSpace(pieces[0])
			val := strings.Trim(strings.TrimSpace(pieces[1]), "`\"")
			if key != "" && val != "" {
				out[key] = val
			}
		}
	}
	return out
}

func repairStepByName(steps []*rollout.Step, name string) *rollout.Step {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if step.Name == name {
			return step
		}
		if found := repairStepByName(step.Steps, name); found != nil {
			return found
		}
	}
	return nil
}

func repairFlattenSteps(steps []*rollout.Step) []*rollout.Step {
	var out []*rollout.Step
	for _, step := range steps {
		if step == nil {
			continue
		}
		out = append(out, step)
		out = append(out, repairFlattenSteps(step.Steps)...)
		for _, branch := range step.Cases {
			if branch != nil {
				out = append(out, repairFlattenSteps(branch.Steps)...)
			}
		}
		if step.Default != nil {
			out = append(out, repairFlattenSteps(step.Default.Steps)...)
		}
	}
	return out
}

func repairStepReference(source string, known map[string]bool) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	for _, token := range strings.FieldsFunc(source, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == '(' || r == ')' || r == '[' || r == ']' || r == '{' || r == '}' ||
			r == ',' || r == '+' || r == '-' || r == '*' || r == '/' || r == '?' || r == ':'
	}) {
		parts := strings.Split(token, ".")
		if len(parts) == 0 {
			continue
		}
		if parts[0] == "steps" && len(parts) > 1 && known[parts[1]] {
			return parts[1]
		}
		if known[parts[0]] {
			return parts[0]
		}
	}
	return ""
}

func repairKnownDependencies(values []string, known map[string]bool, self string) []string {
	var out []string
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || value == self || !known[value] || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func safeRepairSource(source string) bool {
	source = strings.TrimSpace(source)
	if source == "" || len(source) > 240 || strings.ContainsAny(source, "\r\n") {
		return false
	}
	lower := strings.ToLower(source)
	if strings.Contains(lower, "sk-") || strings.Contains(lower, "secret=") || strings.Contains(lower, "token=") {
		return false
	}
	if strings.Contains(lower, "operation") || strings.Contains(lower, "credential binding") {
		return false
	}
	return true
}

func dedupeRepairStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func printRepairReport(out io.Writer, report repairReport) {
	fmt.Fprintf(out, "icot repair: %s %s\n", report.Status, report.Example)
	for _, change := range report.Applied {
		fmt.Fprintf(out, "  applied %s -> %s\n", change.Slot, change.After)
	}
	for _, rejected := range report.Rejected {
		fmt.Fprintf(out, "  rejected %s\n", rejected)
	}
	if report.Error != "" {
		fmt.Fprintf(out, "  error: %s\n", report.Error)
	}
}
