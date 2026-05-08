package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/uws/uws1"
	"github.com/genelet/ramen/internal/authoring"
	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/udon/pkg/rollout"
)

func assessSecrets(report *QualityReport, result Result) {
	var hits []string
	for _, diagnostic := range authoring.ScanCredentialValues(reviewArtifactSet(result).Artifacts) {
		hits = append(hits, diagnostic.Path)
	}
	if len(hits) > 0 {
		report.add("artifacts.no_secrets", "fail", "artifacts contain secret-like tokens", strings.Join(hits, ", "))
		return
	}
	report.add("artifacts.no_secrets", "pass", "no obvious secret-like tokens found in artifacts", "")
}

func writeQualityFiles(result Result, report *QualityReport) error {
	if err := os.MkdirAll(filepath.Dir(result.QualityJSONPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.QualityJSONPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(result.QualityMDPath, []byte(qualityMarkdown(report)), 0o644)
}

func qualityMarkdown(report *QualityReport) string {
	var b strings.Builder
	b.WriteString("# Ramen Quality Report\n\n")
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	for _, check := range report.Checks {
		fmt.Fprintf(&b, "- `%s` %s - %s\n", check.Code, check.Status, check.Message)
		if check.Detail != "" {
			fmt.Fprintf(&b, "  Detail: %s\n", check.Detail)
		}
	}
	return b.String()
}

func missingIntentSteps(intent *rollout.Intent, workflows []*uws1.Workflow) []string {
	stepIDs := map[string]bool{}
	for _, workflow := range workflows {
		if workflow != nil {
			collectUWSStepIDs(workflow.Steps, stepIDs)
		}
	}
	var missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name != "" && !stepIDs[name] {
			missing = append(missing, name)
		}
	})
	sort.Strings(missing)
	return missing
}

func collectUWSStepIDs(steps []*uws1.Step, out map[string]bool) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if strings.TrimSpace(step.StepID) != "" {
			out[strings.TrimSpace(step.StepID)] = true
		}
		collectUWSStepIDs(step.Steps, out)
		for _, branch := range step.Cases {
			if branch != nil {
				collectUWSStepIDs(branch.Steps, out)
			}
		}
		collectUWSStepIDs(step.Default, out)
	}
}

func candidateList(candidates []openapidisco.Candidate) string {
	var items []string
	for _, candidate := range candidates {
		items = append(items, candidate.RelativePath)
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func walkIntentSteps(steps []*rollout.Step, fn func(*rollout.Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		fn(step)
		walkIntentSteps(step.Steps, fn)
		for _, branch := range step.Cases {
			if branch != nil {
				walkIntentSteps(branch.Steps, fn)
			}
		}
		if step.Default != nil {
			walkIntentSteps(step.Default.Steps, fn)
		}
	}
}

func (r *QualityReport) add(code, status, message, detail string) {
	r.Checks = append(r.Checks, QualityCheck{
		Code:    code,
		Status:  status,
		Message: message,
		Detail:  detail,
	})
}

func (r *QualityReport) finalize() {
	for _, check := range r.Checks {
		if check.Status == "fail" {
			r.Status = "fail"
			return
		}
	}
	r.Status = "pass"
}
