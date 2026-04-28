package synthesize

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/genelet/udon/pkg/rollout"
)

func writeReview(result Result, provider, model string) error {
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	return os.WriteFile(result.ReviewPath, []byte(reviewMarkdown(result, provider, model)), 0o644)
}

func reviewMarkdown(result Result, provider, model string) string {
	var b strings.Builder
	b.WriteString("# Ramen Review Evidence\n\n")
	fmt.Fprintf(&b, "- Project brief: `%s`\n", relOrAbs(result.ExampleDir, result.ProjectPath))
	fmt.Fprintf(&b, "- Intent HCL: `%s`\n", relOrAbs(result.ExampleDir, result.IntentPath))
	fmt.Fprintf(&b, "- Workflow HCL: `%s`\n", relOrAbs(result.ExampleDir, result.WorkflowPath))
	fmt.Fprintf(&b, "- UWS artifact: `%s`\n", relOrAbs(result.ExampleDir, result.UWSPath))
	fmt.Fprintf(&b, "- Expected plan: `%s`\n", relOrAbs(result.ExampleDir, result.PlanJSONPath))
	fmt.Fprintf(&b, "- Discovery report: `%s`\n", relOrAbs(result.ExampleDir, result.DiscoveryJSONPath))
	fmt.Fprintf(&b, "- Refinement report: `%s`\n", relOrAbs(result.ExampleDir, result.RefinementJSONPath))
	fmt.Fprintf(&b, "- Primary OpenAPI: `%s`\n", result.PrimaryOpenAPI)
	if provider != "" || model != "" {
		fmt.Fprintf(&b, "- LLM: `%s` `%s`\n", provider, model)
	}
	b.WriteString("\n## OpenAPI Candidates\n\n")
	for _, candidate := range result.OpenAPICandidates {
		fmt.Fprintf(&b, "- `%s`", candidate.RelativePath)
		if candidate.Title != "" {
			fmt.Fprintf(&b, " - %s", candidate.Title)
		}
		if candidate.Source != "" {
			fmt.Fprintf(&b, " (%s)", candidate.Source)
		}
		b.WriteString("\n")
	}
	if len(result.DiscoveryReport.Attempts) > 0 {
		b.WriteString("\n## OpenAPI Discovery Attempts\n\n")
		for _, attempt := range result.DiscoveryReport.Attempts {
			fmt.Fprintf(&b, "- `%s` %s", attempt.Kind, attempt.Status)
			if attempt.Source != "" {
				fmt.Fprintf(&b, " `%s`", attempt.Source)
			}
			if attempt.Detail != "" {
				fmt.Fprintf(&b, " - %s", attempt.Detail)
			}
			b.WriteString("\n")
		}
	}
	if intent, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		b.WriteString("\n## Inferred Steps And Data Flow\n\n")
		writeIntentDataFlowReview(&b, intent)
	}
	b.WriteString("\n## Validation\n\n")
	b.WriteString("- Generated intent.hcl from project.md.\n")
	b.WriteString("- Generated workflow.hcl through udon rollout generation.\n")
	b.WriteString("- Compiled workflow.hcl through udon runtime plan generation.\n")
	b.WriteString("- Exported workflow.uws.yaml and validated it against the UWS schema.\n")
	b.WriteString("- Side-effectful execution was skipped.\n\n")
	b.WriteString("Trusted proof run, only when explicitly approved:\n\n")
	fmt.Fprintf(&b, "```bash\n./scripts/run-udon.sh %s %s\n```\n", relOrAbs(filepath.Dir(result.ExampleDir), result.WorkflowPath), result.ExampleDir)
	return b.String()
}

func writeIntentDataFlowReview(b *strings.Builder, intent *rollout.Intent) {
	if intent == nil || len(intent.Steps) == 0 {
		b.WriteString("- No intent steps were available for review.\n")
		return
	}
	var wrote bool
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		typ := strings.TrimSpace(step.Type)
		if typ == "" {
			typ = "unspecified"
		}
		fmt.Fprintf(b, "- `%s` (%s)", name, typ)
		if step.Operation != "" {
			fmt.Fprintf(b, " operation `%s`", step.Operation)
		}
		if step.Do != "" {
			fmt.Fprintf(b, ": %s", strings.Join(strings.Fields(step.Do), " "))
		}
		b.WriteString("\n")
		wrote = true
		for _, bind := range step.Binds {
			if bind == nil {
				continue
			}
			fmt.Fprintf(b, "  - bind from `%s`", bind.From)
			if len(bind.Fields) == 0 {
				b.WriteString("\n")
				continue
			}
			keys := make([]string, 0, len(bind.Fields))
			for key := range bind.Fields {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				fmt.Fprintf(b, ": `%s <- %s`", key, bind.Fields[key])
			}
			b.WriteString("\n")
		}
	})
	if !wrote {
		b.WriteString("- No leaf intent steps were available for review.\n")
	}
}
