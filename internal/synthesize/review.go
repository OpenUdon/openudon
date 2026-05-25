package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func writeReview(result Result, provider, model string) error {
	if err := ensureArtifactDirs(result); err != nil {
		return err
	}
	projectText := ""
	if data, err := os.ReadFile(result.ProjectPath); err == nil {
		projectText = string(data)
	}
	policy := analyzeProject(projectText)
	var intent *rollout.Intent
	if parsed, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		intent = parsed
	}
	profile := sideEffectProfileForOpenAPI(policy, intent, result.OpenAPICandidates, result.PrimaryOpenAPI)
	if err := writeReviewHandoff(result, policy, profile); err != nil {
		return err
	}
	return os.WriteFile(result.ReviewPath, []byte(reviewMarkdown(result, provider, model)), 0o644)
}

func reviewMarkdown(result Result, provider, model string) string {
	var b strings.Builder
	projectText := ""
	if data, err := os.ReadFile(result.ProjectPath); err == nil {
		projectText = string(data)
	}
	policy := analyzeProject(projectText)
	var intent *rollout.Intent
	if parsed, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		intent = parsed
	}
	profile := sideEffectProfileForOpenAPI(policy, intent, result.OpenAPICandidates, result.PrimaryOpenAPI)
	declaredCredentials := credentialBindingNames(policy)
	expectedPlan := readWorkflowPlan(result.PlanJSONPath)
	expectedCredentials := credentialNamesFromPlan(expectedPlan)
	commonReview := reviewLeafAdapter(reviewArtifactSet(result), declaredCredentials, expectedCredentials)
	commonPackage := commonReview.MinimumReviewPackage()
	b.WriteString("# OpenUdon Review Evidence\n\n")
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
	b.WriteString("\n## Minimum Review Package\n\n")
	fmt.Fprintf(&b, "- Project brief: `%s`\n", relOrAbs(result.ExampleDir, result.ProjectPath))
	fmt.Fprintf(&b, "- Intent HCL: `%s`\n", relOrAbs(result.ExampleDir, result.IntentPath))
	fmt.Fprintf(&b, "- Workflow HCL: `%s`\n", relOrAbs(result.ExampleDir, result.WorkflowPath))
	fmt.Fprintf(&b, "- UWS artifact: `%s`\n", relOrAbs(result.ExampleDir, result.UWSPath))
	fmt.Fprintf(&b, "- Expected plan: `%s`\n", relOrAbs(result.ExampleDir, result.PlanJSONPath))
	fmt.Fprintf(&b, "- Quality report: `%s`\n", relOrAbs(result.ExampleDir, result.QualityJSONPath))
	fmt.Fprintf(&b, "- Refinement report: `%s`\n", relOrAbs(result.ExampleDir, result.RefinementJSONPath))
	fmt.Fprintf(&b, "- Review evidence: `%s`\n", relOrAbs(result.ExampleDir, result.ReviewPath))
	fmt.Fprintf(&b, "- Review handoff manifest: `%s`\n", relOrAbs(result.ExampleDir, result.ReviewHandoffPath))
	fmt.Fprintf(&b, "- OpenUdon review package: `%d` artifact(s), `%d` symbolic binding(s), execution deferred to OpenUdon trusted-runtime policy.\n", len(commonPackage.Artifacts), len(commonPackage.BindingNames))
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
	if intent != nil {
		advice, err := rollout.CatalogAdviceForIntent(intent, rollout.CatalogAdviceOptions{
			ExplicitOpenAPIInputs: reviewExplicitOpenAPIInputs(result, intent),
		})
		if err == nil {
			if markdown := rollout.RenderCatalogAdviceMarkdown(advice); markdown != "" {
				b.WriteString("\n")
				b.WriteString(markdown)
			}
		}
	}
	if intent != nil {
		b.WriteString("\n## Inferred Steps And Data Flow\n\n")
		writeIntentDataFlowReview(&b, intent)
	}
	b.WriteString("\n## Side-Effect Summary\n\n")
	if profile.SideEffectful {
		b.WriteString("- Side-effectful workflow: yes\n")
		for _, reason := range profile.Reasons {
			fmt.Fprintf(&b, "- Evidence: %s\n", reason)
		}
	} else {
		b.WriteString("- Side-effectful workflow: no side-effectful behavior inferred from project policy or intent steps.\n")
	}
	if profile.HasApprovalPolicy {
		b.WriteString("- Approval/trusted-runtime policy: present in project.md.\n")
	} else {
		b.WriteString("- Approval/trusted-runtime policy: not detected in project.md.\n")
	}
	if profile.HasSandboxPolicy {
		b.WriteString("- Sandbox/test proof-run policy: present in project.md.\n")
	} else {
		b.WriteString("- Sandbox/test proof-run policy: not detected in project.md.\n")
	}
	b.WriteString("- Credential binding audit: runtime binding names only; literal secrets are prohibited in prompts, examples, and artifacts.\n")
	b.WriteString("- Direct production execution: not performed by OpenUdon synthesis.\n")
	writeSideEffectRiskReview(&b, profile)
	b.WriteString("\n## Approval State Requirements\n\n")
	b.WriteString("- OpenUdon emitted state: `generated`; no approval is implied by artifact generation.\n")
	b.WriteString("- `validated`: required validators and quality gates have passed or known warnings are attached.\n")
	b.WriteString("- `review_required`: human review is required before side-effectful execution.\n")
	b.WriteString("- `approved_for_sandbox`: sandbox or test-endpoint execution only.\n")
	b.WriteString("- `approved_for_production`: production execution through a trusted runner and approved credentials.\n")
	b.WriteString("- `rejected`: artifact rejected or regeneration requested.\n")
	if profile.SideEffectful {
		b.WriteString("- Required next state: `review_required` before any side-effectful execution.\n")
		b.WriteString("- Sandbox proof run requires review state `approved_for_sandbox`.\n")
		b.WriteString("- Production execution requires review state `approved_for_production` and trusted credentials.\n")
	} else {
		b.WriteString("- `approved_for_sandbox` and `approved_for_production` are not required unless future changes add side effects.\n")
	}
	b.WriteString("- Approval artifact: create `openudon.approval.v1` JSON with `openudon approval-template` only after reviewing the current digest-covered package.\n")
	b.WriteString("\n## Approval Artifact Checklist\n\n")
	b.WriteString("- Approval JSON version: `openudon.approval.v1`.\n")
	b.WriteString("- Required fields: `scope`, `state`, `reviewer`, `approved_at`, `package_sha256`.\n")
	b.WriteString("- Optional fields: `expires_at`, `notes`.\n")
	b.WriteString("- Sandbox tier accepts `approved_for_sandbox` or `approved_for_production`; production tier requires `approved_for_production`.\n")
	b.WriteString("- `package_sha256` must match the current handoff package digest at `openudon run` time.\n")
	b.WriteString("- Regenerate approval JSON after any digest-covered package file changes.\n")
	b.WriteString("\n## Credential Binding Audit\n\n")
	if len(declaredCredentials) == 0 && len(expectedCredentials) == 0 {
		b.WriteString("- No credential bindings declared or required.\n")
	} else {
		if len(declaredCredentials) > 0 {
			fmt.Fprintf(&b, "- Declared credential bindings: `%s`\n", strings.Join(declaredCredentials, "`, `"))
		}
		if len(expectedCredentials) > 0 {
			fmt.Fprintf(&b, "- Expected plan credential bindings: `%s`\n", strings.Join(expectedCredentials, "`, `"))
		}
	}
	audit := commonReview.BindingAudit()
	if len(audit.DeclaredSymbolicBindings) > 0 {
		fmt.Fprintf(&b, "- Symbolic binding audit: `%s`\n", strings.Join(audit.DeclaredSymbolicBindings, "`, `"))
	}
	b.WriteString("- Credential values must stay outside prompts, examples, generated artifacts, and logs.\n")
	writeCredentialScopeMatrix(&b, expectedPlan, declaredCredentials, expectedCredentials)
	b.WriteString("\n## Unresolved Risks\n\n")
	if profile.SideEffectful && !profile.HasApprovalPolicy {
		b.WriteString("- Side-effectful workflow lacks explicit approval or trusted-runtime policy.\n")
	} else if profile.SideEffectful && !profile.HasSandboxPolicy {
		b.WriteString("- Side-effectful workflow lacks explicit sandbox/test proof-run policy.\n")
	} else {
		b.WriteString("- No unresolved execution-boundary risks detected by deterministic review.\n")
	}
	b.WriteString("\n## Validation\n\n")
	b.WriteString("- Generated intent.hcl from project.md.\n")
	b.WriteString("- Generated workflow.hcl as a public UWS document from OpenUdon intent.\n")
	b.WriteString("- Exported workflow.uws.yaml and validated it against the UWS schema and local execution-profile checks.\n")
	b.WriteString("- Side-effectful execution was skipped.\n\n")
	b.WriteString("## Trusted Execution Handoff\n\n")
	b.WriteString("- Direct production execution: not performed by OpenUdon synthesis.\n")
	b.WriteString("- Human approval and trusted-runner invocation are required before operational side effects.\n")
	if profile.SideEffectful {
		b.WriteString("- Trusted proof run command is for sandbox/test execution only after `approved_for_sandbox`.\n")
		b.WriteString("- Production execution requires `approved_for_production`; do not use this command as production approval.\n")
	} else {
		b.WriteString("- Sandbox/test proof run is optional unless future changes add side effects.\n")
	}
	b.WriteString("- Credential binding audit must verify named runtime bindings and no literal secret values.\n")
	b.WriteString("- Dry-run handoff validates approval state, package digest, stored/current quality, tier compatibility, credential-value policy, and direct-production policy before executor invocation.\n")
	b.WriteString("- The generated run config is `openudon.executor-run.v1`; it carries package paths, `package_sha256`, tier, workdir, and credential binding names, not credential values.\n\n")
	b.WriteString("Trusted dry run, before any executor invocation:\n\n")
	fmt.Fprintf(&b, "```bash\nopenudon run --example %s --tier sandbox --approval approvals/%s.json --dry-run\n```\n\n", relOrAbs(filepath.Dir(result.ExampleDir), result.ExampleDir), filepath.Base(result.ExampleDir))
	b.WriteString("Trusted proof run, only when explicitly approved:\n\n")
	fmt.Fprintf(&b, "```bash\nopenudon run --example %s --tier sandbox --approval approvals/%s.json\n```\n", relOrAbs(filepath.Dir(result.ExampleDir), result.ExampleDir), filepath.Base(result.ExampleDir))
	return b.String()
}

func expectedPlanCredentialNames(path string) []string {
	return credentialNamesFromPlan(readWorkflowPlan(path))
}

func reviewExplicitOpenAPIInputs(result Result, intent *rollout.Intent) []string {
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	add(result.PrimaryOpenAPI)
	if intent != nil {
		add(intent.OpenAPI)
		walkIntentSteps(intent.Steps, func(step *rollout.Step) {
			add(step.OpenAPI)
		})
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func readWorkflowPlan(path string) *WorkflowPlan {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var plan WorkflowPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil
	}
	return &plan
}

func credentialNamesFromPlan(plan *WorkflowPlan) []string {
	if plan == nil {
		return nil
	}
	seen := map[string]bool{}
	for _, step := range plan.Steps {
		for _, param := range step.RequestParams {
			for _, credential := range credentialBindingsForPlanParam(param) {
				seen[credential] = true
			}
		}
	}
	var out []string
	for credential := range seen {
		out = append(out, credential)
	}
	sort.Strings(out)
	return out
}

func writeSideEffectRiskReview(b *strings.Builder, profile sideEffectProfile) {
	b.WriteString("\n## Side-Effect Risk Review\n\n")
	if !profile.SideEffectful {
		b.WriteString("- No side-effectful operations were inferred for this package.\n")
		return
	}
	if len(profile.Effects) == 0 {
		b.WriteString("- Side-effectful behavior was inferred, but no step-level effect details were available.\n")
		b.WriteString("- Required approval path: `review_required` -> `approved_for_sandbox` for proof runs; `approved_for_production` for production.\n")
		return
	}
	for _, effect := range profile.Effects {
		step := firstNonEmpty(effect.Step, "<unknown>")
		kind := firstNonEmpty(effect.Kind, "effect")
		source := firstNonEmpty(effect.Source, "project/intent evidence")
		fmt.Fprintf(b, "- `%s` %s", step, kind)
		if effect.Operation != "" {
			fmt.Fprintf(b, " operation `%s`", effect.Operation)
		}
		if effect.Method != "" || effect.Path != "" {
			fmt.Fprintf(b, " `%s %s`", strings.TrimSpace(effect.Method), strings.TrimSpace(effect.Path))
		}
		fmt.Fprintf(b, " from `%s`: %s\n", source, firstNonEmpty(effect.Risk, "requires review before trusted-runner execution"))
	}
	b.WriteString("- Required approval path: `review_required` -> `approved_for_sandbox` for sandbox/test proof runs; `approved_for_production` for production.\n")
}

func writeCredentialScopeMatrix(b *strings.Builder, plan *WorkflowPlan, declaredCredentials, expectedCredentials []string) {
	b.WriteString("\n## Credential Scope Matrix\n\n")
	if len(declaredCredentials) == 0 && len(expectedCredentials) == 0 {
		b.WriteString("- No credential bindings are declared or expected from the plan.\n")
		b.WriteString("- Credential values: not allowed in generated artifacts.\n")
		return
	}
	if plan == nil || len(plan.Steps) == 0 {
		b.WriteString("- Plan step credential scope was unavailable; review declared and expected binding inventories above.\n")
		b.WriteString("- Credential values: not allowed in generated artifacts.\n")
		return
	}
	var wrote bool
	for _, step := range plan.Steps {
		bindings := credentialBindingsForPlanStep(step)
		if len(bindings) == 0 {
			continue
		}
		openapi := firstNonEmpty(step.OpenAPI, "local/runtime")
		operation := firstNonEmpty(step.Operation, step.Type)
		stepName := firstNonEmpty(step.Name, "<unnamed>")
		fmt.Fprintf(b, "- `%s`: scope `%s %s`; bindings `%s`\n", stepName, openapi, operation, strings.Join(bindings, "`, `"))
		wrote = true
	}
	if !wrote {
		b.WriteString("- No step-level credential references were present; review declared binding inventory above.\n")
	}
	b.WriteString("- Credential values: not allowed in generated artifacts; the trusted runner resolves only named bindings at execution time.\n")
}

func credentialBindingsForPlanStep(step PlanStep) []string {
	seen := map[string]bool{}
	for _, param := range step.RequestParams {
		for _, credential := range credentialBindingsForPlanParam(param) {
			seen[credential] = true
		}
	}
	out := make([]string, 0, len(seen))
	for credential := range seen {
		out = append(out, credential)
	}
	sort.Strings(out)
	return out
}

func credentialBindingsForPlanParam(param PlanParam) []string {
	if !param.Credential || param.SourceKind != "credential" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, credential := range []string{param.ExpectedCredential, param.ExpectedSource} {
		credential = planCredentialBindingName(credential)
		if credential == "" || seen[credential] {
			continue
		}
		seen[credential] = true
		out = append(out, credential)
	}
	sort.Strings(out)
	return out
}

func planCredentialBindingName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "credentials.")
	return strings.TrimSpace(value)
}

func reviewArtifactSet(result Result) authoring.ArtifactSet {
	artifactPaths := []string{
		result.ProjectPath,
		result.IntentPath,
		result.WorkflowPath,
		result.UWSPath,
		result.PlanJSONPath,
		result.PlanMDPath,
		result.DiscoveryJSONPath,
		result.QualityJSONPath,
		result.QualityMDPath,
		result.RefinementJSONPath,
		result.RefinementMDPath,
		result.ReviewPath,
		result.ReviewHandoffPath,
	}
	artifacts := make([]authoring.Artifact, 0, len(artifactPaths))
	for _, path := range artifactPaths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		content, _ := os.ReadFile(path)
		artifacts = append(artifacts, authoring.Artifact{
			Path:      relOrAbs(result.ExampleDir, path),
			MediaType: reviewArtifactMediaType(path),
			Content:   content,
		})
	}
	return authoring.ArtifactSet{Artifacts: artifacts}
}

func reviewLeafAdapter(set authoring.ArtifactSet, declaredCredentials, expectedCredentials []string) authoring.LeafAdapter {
	seen := map[string]bool{}
	var bindings []authoring.SymbolicBinding
	for _, name := range append(append([]string(nil), declaredCredentials...), expectedCredentials...) {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		bindings = append(bindings, authoring.SymbolicBinding{
			Name:        name,
			Kind:        "credential",
			Source:      "openudon.review",
			Description: "OpenUdon runtime credential binding name; value is supplied outside generated artifacts.",
		})
	}
	set.SymbolicBindings = bindings
	return authoring.NewLeafAdapter(set, authoring.LeafOptions{
		Name:   "OpenUdon Review Evidence",
		Source: "openudon.synthesize",
	})
}

func reviewArtifactMediaType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		return "text/markdown"
	case ".hcl":
		return "text/hcl"
	case ".json":
		return "application/json"
	case ".yaml", ".yml":
		return "application/yaml"
	default:
		return "application/octet-stream"
	}
}

func writeIntentDataFlowReview(b *strings.Builder, intent *rollout.Intent) {
	if intent == nil || len(intent.Steps) == 0 {
		b.WriteString("- No intent steps were available for review.\n")
		return
	}
	var wrote bool
	writeIntentStepReview(b, intent.Steps, reviewStepContext{}, &wrote)
	if !wrote {
		b.WriteString("- No leaf intent steps were available for review.\n")
	}
}

type reviewStepContext struct {
	Parent     string
	Branch     string
	BranchWhen string
}

func writeIntentStepReview(b *strings.Builder, steps []*rollout.Step, ctx reviewStepContext, wrote *bool) {
	for _, step := range steps {
		if step == nil {
			continue
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
		*wrote = true
		if ctx.Parent != "" {
			fmt.Fprintf(b, "  - parent: `%s`\n", ctx.Parent)
		}
		if ctx.Branch != "" {
			fmt.Fprintf(b, "  - branch: `%s`", ctx.Branch)
			if ctx.BranchWhen != "" {
				fmt.Fprintf(b, " when `%s`", ctx.BranchWhen)
			}
			b.WriteString("\n")
		}
		if step.When != "" {
			fmt.Fprintf(b, "  - when: `%s`\n", step.When)
		}
		if step.ForEach != "" {
			fmt.Fprintf(b, "  - for_each: `%s`\n", step.ForEach)
		}
		if step.Items != "" {
			fmt.Fprintf(b, "  - items: `%s`\n", step.Items)
		}
		if step.Mode != "" {
			fmt.Fprintf(b, "  - mode: `%s`\n", step.Mode)
		}
		if step.BatchSize != "" {
			fmt.Fprintf(b, "  - batch_size: `%s`\n", step.BatchSize)
		}
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
		writeIntentActionReview(b, step)
		writeIntentStepReview(b, step.Steps, reviewStepContext{Parent: name}, wrote)
		for _, branch := range step.Cases {
			if branch == nil {
				continue
			}
			writeIntentStepReview(b, branch.Steps, reviewStepContext{
				Parent:     name,
				Branch:     strings.TrimSpace(branch.Name),
				BranchWhen: strings.TrimSpace(branch.When),
			}, wrote)
		}
		if step.Default != nil {
			writeIntentStepReview(b, step.Default.Steps, reviewStepContext{
				Parent: name,
				Branch: "default",
			}, wrote)
		}
	}
}

func writeIntentActionReview(b *strings.Builder, step *rollout.Step) {
	if b == nil || step == nil {
		return
	}
	for _, criterion := range step.SuccessCriteria {
		if criterion == nil {
			continue
		}
		fmt.Fprintf(b, "  - successCriteria: `%s`", criterion.Condition)
		if criterion.Type != "" {
			fmt.Fprintf(b, " type `%s`", criterion.Type)
		}
		b.WriteString("\n")
	}
	for _, action := range step.OnFailure {
		if action == nil {
			continue
		}
		fmt.Fprintf(b, "  - onFailure: `%s` type `%s`", action.Name, action.Type)
		if action.StepID != "" {
			fmt.Fprintf(b, " stepId `%s`", action.StepID)
		}
		if action.WorkflowID != "" {
			fmt.Fprintf(b, " workflowId `%s`", action.WorkflowID)
		}
		if action.RetryLimit > 0 {
			fmt.Fprintf(b, " retryLimit `%d`", action.RetryLimit)
		}
		if action.RetryAfter > 0 {
			fmt.Fprintf(b, " retryAfter `%g`", action.RetryAfter)
		}
		b.WriteString("\n")
	}
	for _, action := range step.OnSuccess {
		if action == nil {
			continue
		}
		fmt.Fprintf(b, "  - onSuccess: `%s` type `%s`", action.Name, action.Type)
		if action.StepID != "" {
			fmt.Fprintf(b, " stepId `%s`", action.StepID)
		}
		if action.WorkflowID != "" {
			fmt.Fprintf(b, " workflowId `%s`", action.WorkflowID)
		}
		b.WriteString("\n")
	}
}
