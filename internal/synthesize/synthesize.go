package synthesize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/openapidisco"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/uwsschema"
	"github.com/OpenUdon/openudon/internal/workflowintent"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	runner "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
)

type Options struct {
	ExampleDir        string
	Provider          string
	Model             string
	Timeout           time.Duration
	MaxAttempts       int
	IntentTemperature *float64

	Discoverer *openapidisco.Discoverer
	LLMClient  rollout.LLMClient
	ChatClient rollout.ChatClient
	SchemaPath string
}

type Result struct {
	ExampleDir         string
	ProjectPath        string
	IntentPath         string
	WorkflowPath       string
	UWSPath            string
	PlanJSONPath       string
	PlanMDPath         string
	DiscoveryJSONPath  string
	DataPath           string
	RefinementJSONPath string
	RefinementMDPath   string
	ReviewPath         string
	ReviewHandoffPath  string
	QualityJSONPath    string
	QualityMDPath      string
	PrimaryOpenAPI     string
	OpenAPICandidates  []openapidisco.Candidate
	DiscoveryReport    openapidisco.DiscoveryReport
}

func Run(ctx context.Context, opts Options) (*Result, error) {
	return Synthesize(ctx, opts)
}

func Synthesize(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := prepareRefinement(ctx, opts)
	if err != nil {
		return nil, err
	}
	llm, chat, provider, model, err := resolveClients(opts)
	if err != nil {
		return nil, err
	}
	state.chat = chat
	temperature := intentTemperature(opts)
	state.intentTemperature = &temperature
	return runRefinement(ctx, opts, state, llm, provider, model, "generate_intent", func(ctx context.Context, attempt int, action, feedback string, intent *rollout.Intent, state *refinementState) (*rollout.Intent, string, error) {
		if attempt == 1 || action != "regenerate_workflow" || intent == nil {
			if err := ctx.Err(); err != nil {
				return nil, action, err
			}
			messages := intentPromptMessagesForMode(state.projectText, state.candidates, state.primaryPath, state.policy, feedback, supportsStructuredChat(state.chat))
			if attempt == 1 && state.promptSnapshot == "" {
				state.promptSnapshot = renderPromptSnapshot(messages)
			}
			var err error
			var mode string
			intent, mode, err = generateIntentFromMessagesWithMode(ctx, state.chat, messages, state.primaryPath, state.policy, state.intentTemperature)
			state.generationMode = mode
			if err != nil {
				return nil, action, fmt.Errorf("generate intent: %w", err)
			}
		}
		if !state.policy.NoOpenAPI {
			if err := ctx.Err(); err != nil {
				return nil, action, err
			}
			var attempts []openapidisco.DiscoveryAttempt
			var changed bool
			state.candidates, attempts, changed = discoverComplementaryOpenAPI(ctx, state.discoverer, state.result.ExampleDir, state.projectText, state.candidates, intent, state.policy)
			if len(attempts) > 0 {
				state.discoveryReport.Attempts = append(state.discoveryReport.Attempts, attempts...)
				state.result.DiscoveryReport = state.discoveryReport
				if err := writeDiscoveryReport(state.result, state.discoveryReport); err != nil {
					return nil, action, fmt.Errorf("write discovery report: %w", err)
				}
			}
			if changed {
				state.result.OpenAPICandidates = state.candidates
				if primary, err := openapidisco.SelectPrimary(state.candidates); err == nil {
					state.primaryPath = primary.RelativePath
					state.result.PrimaryOpenAPI = state.primaryPath
				}
				if err := ctx.Err(); err != nil {
					return nil, "discover_openapi", err
				}
				var err error
				var mode string
				intent, mode, err = generateIntentWithMode(ctx, state.chat, state.projectText, state.candidates, state.primaryPath, state.policy, "Complementary OpenAPI discovery added candidate documents; preserve the project goal and use the newly available operations when needed.", state.intentTemperature)
				state.generationMode = mode
				if err != nil {
					return nil, "discover_openapi", fmt.Errorf("regenerate intent after complementary OpenAPI discovery: %w", err)
				}
				action = "discover_openapi"
			}
		}
		return intent, action, nil
	})
}

func Build(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result, report, err := PackageFromIntent(ctx, opts)
	if err != nil {
		return result, err
	}
	if report != nil && !report.Passed() {
		return result, fmt.Errorf("quality gate failed; see %s", result.QualityJSONPath)
	}
	return result, nil
}

// PackageFromIntent builds the normal OpenUdon review package artifacts from an
// already-authored intent.hcl without resolving an LLM provider. It writes the
// same workflow, UWS, plan, discovery, review, handoff, refinement, and quality
// artifacts used by synthesize/build, but returns quality failures in the
// report instead of treating them as infrastructure errors.
func PackageFromIntent(ctx context.Context, opts Options) (*Result, *QualityReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	state, err := prepareRefinement(ctx, opts)
	if err != nil {
		return nil, nil, err
	}
	intent, err := workflowintent.ParseFile(ctx, state.result.IntentPath)
	if err != nil {
		return &state.result, nil, fmt.Errorf("parse intent.hcl: %w", err)
	}
	applyProjectTimeoutAndIdempotency(intent, state.policy)
	primary := strings.TrimSpace(intent.OpenAPI)
	if primary == "" && !state.policy.NoOpenAPI {
		selected, err := openapidisco.SelectPrimary(state.candidates)
		if err != nil {
			return &state.result, nil, err
		}
		primary = selected.RelativePath
		intent.OpenAPI = primary
	}
	if err := validateIntentOpenAPIRefs(intent, state.result.ExampleDir, state.candidates, primary, state.policy.NoOpenAPI); err != nil {
		return &state.result, nil, err
	}
	if err := validateIntentRuntimePolicy(intent, state.policy); err != nil {
		return &state.result, nil, err
	}
	normalizeIntentSecurityCredentialBindings(intent, state.candidates, primary)
	if err := ctx.Err(); err != nil {
		return &state.result, nil, err
	}
	state.primaryPath = primary
	state.result.PrimaryOpenAPI = primary
	state.generationMode = "fixed_intent"
	workflowPlan := buildWorkflowPlan(state.result, intent, state.candidates, state.policy)
	intentHCL, err := workflowintent.RenderHCL(ctx, intent)
	if err != nil {
		return &state.result, nil, fmt.Errorf("render intent HCL: %w", err)
	}
	if err := ensureArtifactDirs(state.result); err != nil {
		return &state.result, nil, err
	}
	if err := os.WriteFile(state.result.IntentPath, []byte(intentHCL), 0o644); err != nil {
		return &state.result, nil, err
	}
	if err := writeWorkflowPlan(state.result, workflowPlan); err != nil {
		return &state.result, nil, err
	}
	if err := writeRuntimeDataFile(state.result, intent, state.policy); err != nil {
		return &state.result, nil, err
	}
	if err := generateWorkflow(ctx, state.result, intent, nil, opts.Provider, opts.Model, opts.Timeout); err != nil {
		return &state.result, nil, err
	}
	if err := promoteWorkflow(state.result, opts.SchemaPath); err != nil {
		return &state.result, nil, err
	}
	if err := writeReview(state.result, opts.Provider, opts.Model); err != nil {
		return &state.result, nil, err
	}
	report, err := AssessContext(ctx, opts)
	if err != nil {
		return &state.result, nil, err
	}
	refinement := newRefinementReport(state.result, 1)
	stopReason := "quality passed"
	if !report.Passed() {
		stopReason = "quality gate failed; review generated artifacts"
	}
	refinement.addAttempt(1, "package_from_intent", report, nil, stopReason)
	refinement.setLastAttemptMode(state.generationMode)
	if err := writeRefinementReport(state.result, refinement); err != nil {
		return &state.result, report, fmt.Errorf("write refinement report: %w", err)
	}
	return &state.result, report, nil
}

type refinementState struct {
	result            Result
	projectText       string
	policy            projectPolicy
	discoverer        *openapidisco.Discoverer
	candidates        []openapidisco.Candidate
	primaryPath       string
	discoveryReport   openapidisco.DiscoveryReport
	chat              rollout.ChatClient
	promptSnapshot    string
	generationMode    string
	intentTemperature *float64
}

type refinementIntentSupplier func(context.Context, int, string, string, *rollout.Intent, *refinementState) (*rollout.Intent, string, error)

func prepareRefinement(ctx context.Context, opts Options) (*refinementState, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	exampleDir, err := resolveExampleDir(opts.ExampleDir)
	if err != nil {
		return nil, err
	}
	result := resultPaths(exampleDir)
	projectBytes, err := os.ReadFile(result.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("read project brief: %w", err)
	}
	projectText := string(projectBytes)
	policy := analyzeProject(projectText)
	if err := validateStructuredProjectPolicy(policy); err != nil {
		return nil, fmt.Errorf("project policy: %w", err)
	}
	discoverer := opts.Discoverer
	if discoverer == nil {
		discoverer = &openapidisco.Discoverer{}
	}
	state := &refinementState{
		result:      result,
		projectText: projectText,
		policy:      policy,
		discoverer:  discoverer,
	}
	if !policy.NoOpenAPI {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		candidates, discoveryReport, err := discoverer.DiscoverWithReport(ctx, exampleDir, projectText)
		state.candidates = candidates
		state.discoveryReport = discoveryReport
		state.result.DiscoveryReport = discoveryReport
		if err := writeDiscoveryReport(state.result, discoveryReport); err != nil {
			return nil, fmt.Errorf("write discovery report: %w", err)
		}
		if err != nil {
			return nil, fmt.Errorf("discover OpenAPI documents: %w", err)
		}
		primary, err := openapidisco.SelectPrimary(candidates)
		if err != nil {
			return nil, err
		}
		state.primaryPath = primary.RelativePath
	}
	state.result.PrimaryOpenAPI = state.primaryPath
	state.result.OpenAPICandidates = state.candidates
	state.result.DiscoveryReport = state.discoveryReport
	return state, nil
}

func runRefinement(ctx context.Context, opts Options, state *refinementState, llm rollout.LLMClient, provider, model, initialAction string, supplyIntent refinementIntentSupplier) (*Result, error) {
	attempts := maxAttempts(opts.MaxAttempts)
	refinement := newRefinementReport(state.result, attempts)
	var previousSignature string
	action := initialAction
	var feedback string
	var intent *rollout.Intent
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return &state.result, err
		}
		var err error
		intent, action, err = supplyIntent(ctx, attempt, action, feedback, intent, state)
		if refinement.PromptSnapshot == "" && state.promptSnapshot != "" {
			refinement.PromptSnapshot = state.promptSnapshot
		}
		if err != nil {
			stopReason := refinementStopReasonForError(err)
			refinement.addAttempt(attempt, action, nil, err, stopReason)
			refinement.setLastAttemptMode(state.generationMode)
			if writeErr := writeRefinementReport(state.result, refinement); writeErr != nil {
				return nil, fmt.Errorf("write refinement report: %w", writeErr)
			}
			return &state.result, err
		}
		if err := validateIntentOpenAPIRefs(intent, state.result.ExampleDir, state.candidates, state.primaryPath, state.policy.NoOpenAPI); err != nil {
			refinement.addAttempt(attempt, action, nil, err, "intent references unavailable OpenAPI metadata")
			refinement.setLastAttemptMode(state.generationMode)
			if writeErr := writeRefinementReport(state.result, refinement); writeErr != nil {
				return nil, fmt.Errorf("write refinement report: %w", writeErr)
			}
			return &state.result, err
		}
		if err := validateIntentRuntimePolicy(intent, state.policy); err != nil {
			refinement.addAttempt(attempt, action, nil, err, "intent uses a runtime outside project policy")
			refinement.setLastAttemptMode(state.generationMode)
			if writeErr := writeRefinementReport(state.result, refinement); writeErr != nil {
				return nil, fmt.Errorf("write refinement report: %w", writeErr)
			}
			return &state.result, err
		}
		normalizeIntentSecurityCredentialBindings(intent, state.candidates, state.primaryPath)
		workflowPlan := buildWorkflowPlan(state.result, intent, state.candidates, state.policy)
		intentHCL, err := workflowintent.RenderHCL(ctx, intent)
		if err != nil {
			refinement.addAttempt(attempt, action, nil, fmt.Errorf("render intent HCL: %w", err), "intent rendering failed")
			refinement.setLastAttemptMode(state.generationMode)
			if writeErr := writeRefinementReport(state.result, refinement); writeErr != nil {
				return nil, fmt.Errorf("write refinement report: %w", writeErr)
			}
			return &state.result, fmt.Errorf("render intent HCL: %w", err)
		}
		if err := ensureArtifactDirs(state.result); err != nil {
			return nil, err
		}
		if err := os.WriteFile(state.result.IntentPath, []byte(intentHCL), 0o644); err != nil {
			return nil, err
		}
		if err := writeWorkflowPlan(state.result, workflowPlan); err != nil {
			return nil, err
		}
		if err := writeRuntimeDataFile(state.result, intent, state.policy); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return &state.result, err
		}
		if err := generateWorkflow(ctx, state.result, intent, llm, provider, model, opts.Timeout); err != nil {
			stopReason := ""
			if attempt == attempts {
				stopReason = "maximum refinement attempts reached"
			}
			refinement.addAttempt(attempt, action, nil, err, stopReason)
			refinement.setLastAttemptMode(state.generationMode)
			if writeErr := writeRefinementReport(state.result, refinement); writeErr != nil {
				return nil, fmt.Errorf("write refinement report: %w", writeErr)
			}
			if stopReason != "" {
				return &state.result, err
			}
			action = "regenerate_intent"
			feedback = err.Error()
			continue
		}
		if err := ctx.Err(); err != nil {
			return &state.result, err
		}
		if err := promoteWorkflow(state.result, opts.SchemaPath); err != nil {
			stopReason := ""
			if attempt == attempts {
				stopReason = "maximum refinement attempts reached"
			}
			refinement.addAttempt(attempt, action, nil, err, stopReason)
			refinement.setLastAttemptMode(state.generationMode)
			if writeErr := writeRefinementReport(state.result, refinement); writeErr != nil {
				return nil, fmt.Errorf("write refinement report: %w", writeErr)
			}
			if stopReason != "" {
				return &state.result, err
			}
			action = "regenerate_intent"
			feedback = err.Error()
			continue
		}
		if err := writeReview(state.result, provider, model); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return &state.result, err
		}
		report, err := AssessContext(ctx, opts)
		if err != nil {
			return nil, err
		}
		if report.Passed() {
			refinement.addAttempt(attempt, action, report, nil, "quality passed")
			refinement.setLastAttemptMode(state.generationMode)
			if err := writeRefinementReport(state.result, refinement); err != nil {
				return nil, fmt.Errorf("write refinement report: %w", err)
			}
			return &state.result, nil
		}
		nextAction, terminal := classifyRefinementAction(report)
		signature := qualityFailureSignature(report)
		stopReason := ""
		if initialAction == "regenerate_workflow" && (terminal || nextAction != "regenerate_workflow") {
			stopReason = "quality failure requires project.md or intent.hcl repair"
		} else if terminal {
			stopReason = "terminal quality failure"
		} else if attempt == attempts {
			stopReason = "maximum refinement attempts reached"
		} else if attempt > 1 && signature == previousSignature {
			stopReason = "repeated quality failure"
		}
		refinement.addAttempt(attempt, action, report, nil, stopReason)
		refinement.setLastAttemptMode(state.generationMode)
		if err := writeRefinementReport(state.result, refinement); err != nil {
			return nil, fmt.Errorf("write refinement report: %w", err)
		}
		if stopReason != "" {
			return &state.result, fmt.Errorf("quality gate failed; see %s", state.result.QualityJSONPath)
		}
		previousSignature = signature
		action = nextAction
		feedback = failingCheckDetails(report)
	}
	return &state.result, fmt.Errorf("quality gate failed; see %s", state.result.QualityJSONPath)
}

func refinementStopReasonForError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "cancelled"
	}
	if strings.Contains(err.Error(), "write discovery report") {
		return "discovery evidence could not be written"
	}
	if strings.Contains(err.Error(), "generate intent") {
		return "intent generation failed"
	}
	return ""
}

func Promote(ctx context.Context, opts Options) (*Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	exampleDir, err := resolveExampleDir(opts.ExampleDir)
	if err != nil {
		return nil, err
	}
	result := resultPaths(exampleDir)
	projectBytes, err := os.ReadFile(result.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("read project brief: %w", err)
	}
	candidates, err := openapidisco.LocalFiles(filepath.Join(exampleDir, "openapi"), exampleDir, string(projectBytes))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("scan local OpenAPI documents: %w", err)
	}
	result.OpenAPICandidates = candidates
	result.DiscoveryReport = openapidisco.DiscoveryReport{Attempts: []openapidisco.DiscoveryAttempt{{
		Kind:   "local",
		Source: filepath.ToSlash(filepath.Join(exampleDir, "openapi")),
		Status: "pass",
		Detail: fmt.Sprintf("%d local OpenAPI document(s)", len(candidates)),
	}}}
	if err := writeDiscoveryReport(result, result.DiscoveryReport); err != nil {
		return nil, fmt.Errorf("write discovery report: %w", err)
	}
	if len(candidates) > 0 {
		if primary, err := openapidisco.SelectPrimary(candidates); err == nil {
			result.PrimaryOpenAPI = primary.RelativePath
		}
	}
	if intent, err := rollout.ParseIntentFile(result.IntentPath); err == nil {
		policy := analyzeProject(string(projectBytes))
		if err := validateStructuredProjectPolicy(policy); err != nil {
			return nil, fmt.Errorf("project policy: %w", err)
		}
		applyProjectTimeoutAndIdempotency(intent, policy)
		normalizeIntentSecurityCredentialBindings(intent, candidates, result.PrimaryOpenAPI)
		if err := writeWorkflowPlan(result, buildWorkflowPlan(result, intent, candidates, policy)); err != nil {
			return nil, err
		}
		if err := writeRuntimeDataFile(result, intent, policy); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := promoteWorkflow(result, opts.SchemaPath); err != nil {
		return nil, err
	}
	if err := writeReview(result, opts.Provider, opts.Model); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	report, err := AssessContext(ctx, opts)
	if err != nil {
		return nil, err
	}
	if !report.Passed() {
		return &result, fmt.Errorf("quality gate failed; see %s", result.QualityJSONPath)
	}
	return &result, nil
}

func generateWorkflow(ctx context.Context, result Result, intent *rollout.Intent, llm rollout.LLMClient, provider, model string, timeout time.Duration) error {
	_, _, _, _ = llm, provider, model, timeout
	if err := ctx.Err(); err != nil {
		return err
	}
	doc, err := generateWorkflowDocument(result, intent)
	if err != nil {
		return fmt.Errorf("generate UWS workflow: %w", err)
	}
	return writeWorkflowHCL(result, doc, intent)
}

func deterministicNoOpenAPICommandWorkflow(intent *rollout.Intent, primaryOpenAPI string) (string, bool) {
	if intent == nil || strings.TrimSpace(primaryOpenAPI) != "" || strings.TrimSpace(intent.OpenAPI) != "" || len(intent.Steps) != 1 {
		return "", false
	}
	step := intent.Steps[0]
	if step == nil || !strings.EqualFold(strings.TrimSpace(step.Type), "cmd") {
		return "", false
	}
	name := strings.TrimSpace(step.Name)
	if name == "" {
		name = "run_command"
	}
	command := strings.TrimSpace(step.With["command"])
	if command == "" {
		command = strings.TrimSpace(step.Do)
	}
	if command == "" {
		return "", false
	}
	var b strings.Builder
	fmt.Fprintf(&b, "cmd %s {\n", strconv.Quote(name))
	if do := strings.TrimSpace(step.Do); do != "" {
		fmt.Fprintf(&b, "  description = %s\n", strconv.Quote(do))
	}
	fmt.Fprintf(&b, "  command = %s\n", strconv.Quote(command))
	b.WriteString("}\n")
	return b.String(), true
}

func resolveExampleDir(example string) (string, error) {
	example = strings.TrimSpace(example)
	if example == "" {
		return "", fmt.Errorf("--example is required")
	}
	return filepath.Abs(example)
}

func resultPaths(exampleDir string) Result {
	workflowsDir := filepath.Join(exampleDir, "workflows")
	expectedDir := filepath.Join(exampleDir, "expected")
	return Result{
		ExampleDir:         exampleDir,
		ProjectPath:        filepath.Join(exampleDir, "project.md"),
		IntentPath:         filepath.Join(workflowsDir, "intent.hcl"),
		WorkflowPath:       filepath.Join(workflowsDir, "workflow.hcl"),
		UWSPath:            filepath.Join(workflowsDir, "workflow.uws.yaml"),
		PlanJSONPath:       filepath.Join(expectedDir, "plan.json"),
		PlanMDPath:         filepath.Join(expectedDir, "plan.md"),
		DiscoveryJSONPath:  filepath.Join(expectedDir, "discovery.json"),
		DataPath:           filepath.Join(expectedDir, "data.hcl"),
		RefinementJSONPath: filepath.Join(expectedDir, "refinement.json"),
		RefinementMDPath:   filepath.Join(expectedDir, "refinement.md"),
		ReviewPath:         filepath.Join(expectedDir, "review.md"),
		ReviewHandoffPath:  filepath.Join(expectedDir, filepath.Base(packageartifacts.ReviewHandoffPath)),
		QualityJSONPath:    filepath.Join(expectedDir, "quality.json"),
		QualityMDPath:      filepath.Join(expectedDir, "quality.md"),
		OpenAPICandidates:  nil,
	}
}

func writeDiscoveryReport(result Result, report openapidisco.DiscoveryReport) error {
	if len(report.Attempts) == 0 {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(result.DiscoveryJSONPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(result.DiscoveryJSONPath, append(data, '\n'), 0o644)
}

func ensureArtifactDirs(result Result) error {
	if err := os.MkdirAll(filepath.Dir(result.IntentPath), 0o755); err != nil {
		return err
	}
	return os.MkdirAll(filepath.Dir(result.ReviewPath), 0o755)
}

func defaultSchemaPath(exampleDir string) string {
	return defaultSchemaPathForVersion(exampleDir, "1.0.0")
}

func defaultSchemaPathForVersion(exampleDir, version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "1.0.0"
	}
	return uwsschema.PathForVersion(exampleDir, version)
}

func defaultSchemaPathForDocument(exampleDir, documentPath string) string {
	doc, err := loadUWSDocumentFile(documentPath)
	if err != nil || doc == nil || strings.TrimSpace(doc.UWS) == "" {
		return defaultSchemaPath(exampleDir)
	}
	return defaultSchemaPathForVersion(exampleDir, doc.UWS)
}

func resolveClients(opts Options) (rollout.LLMClient, rollout.ChatClient, string, string, error) {
	if opts.LLMClient != nil && opts.ChatClient != nil {
		return opts.LLMClient, opts.ChatClient, strings.TrimSpace(opts.Provider), strings.TrimSpace(opts.Model), nil
	}
	temperature := intentTemperature(opts)
	llm, provider, model, err := runner.NewLLMClientFromEnvWithOptions(opts.Provider, opts.Model, runner.LLMOptions{
		Temperature: &temperature,
	})
	if err != nil {
		return nil, nil, "", "", err
	}
	chat, ok := llm.(rollout.ChatClient)
	if !ok {
		return nil, nil, "", "", fmt.Errorf("selected provider %s does not support chat", provider)
	}
	return llm, chat, provider, model, nil
}

func intentTemperature(opts Options) float64 {
	if opts.IntentTemperature == nil {
		return 0.2
	}
	return *opts.IntentTemperature
}

func validateIntentOpenAPIRefs(intent *rollout.Intent, exampleDir string, candidates []openapidisco.Candidate, primary string, noOpenAPI bool) error {
	if intent == nil {
		return nil
	}
	if noOpenAPI {
		var refs []string
		if strings.TrimSpace(intent.Source) != "" || strings.TrimSpace(intent.OpenAPI) != "" {
			refs = append(refs, "top-level api source")
		}
		walkIntentSteps(intent.Steps, func(step *rollout.Step) {
			if step == nil {
				return
			}
			if strings.TrimSpace(step.Source) != "" || strings.TrimSpace(step.OpenAPI) != "" {
				refs = append(refs, fmt.Sprintf("%s.source", strings.TrimSpace(step.Name)))
			}
			if strings.TrimSpace(step.Operation) != "" {
				refs = append(refs, fmt.Sprintf("%s.operation", strings.TrimSpace(step.Name)))
			}
		})
		if len(refs) > 0 {
			sort.Strings(refs)
			return fmt.Errorf("project declares OpenAPI: none required but intent references OpenAPI metadata: %s", strings.Join(refs, ", "))
		}
		return nil
	}
	allowed := map[string]bool{}
	if strings.TrimSpace(primary) != "" {
		allowed[normalizeAPISourceRef(primary)] = true
	}
	for _, candidate := range candidates {
		allowed[normalizeAPISourceRef(candidate.RelativePath)] = true
	}
	sourceRegistry, sourceRegistryErr := newLocalAPISourceRegistry(exampleDir, candidates)
	if sourceRegistryErr != nil && !errors.Is(sourceRegistryErr, os.ErrNotExist) {
		return fmt.Errorf("local API source registry could not be scanned: %w", sourceRegistryErr)
	}
	if strings.TrimSpace(intent.Source) != "" {
		intent.Source = normalizeAPISourceRef(intent.Source)
		if strings.TrimSpace(intent.OpenAPI) == "" {
			intent.OpenAPI = intent.Source
		}
	}
	if strings.TrimSpace(intent.Source) == "" && strings.TrimSpace(intent.OpenAPI) == "" && primary != "" {
		intent.OpenAPI = normalizeAPISourceRef(primary)
	}
	if ref := normalizeAPISourceRef(firstNonEmpty(intent.Source, intent.OpenAPI)); ref != "" {
		if entry, ok := sourceRegistry.get(ref); ok && entry.Err != nil {
			return fmt.Errorf("generated intent referenced invalid API source document %q: %w", ref, entry.Err)
		}
		if sourceDescriptionTypeForPath(ref) == uws1.SourceDescriptionTypeOpenAPI {
			if !allowed[ref] {
				return fmt.Errorf("generated intent referenced unavailable OpenAPI document %q", ref)
			}
			if strings.TrimSpace(intent.Source) != "" {
				intent.Source = ref
			} else {
				intent.OpenAPI = ref
			}
		} else {
			entry, ok := sourceRegistry.get(ref)
			if !ok {
				return fmt.Errorf("generated intent referenced unavailable API source document %q", ref)
			}
			if entry.Err != nil {
				return fmt.Errorf("generated intent referenced invalid API source document %q: %w", ref, entry.Err)
			}
			intent.OpenAPI = entry.RelativePath
			if strings.TrimSpace(intent.Source) != "" {
				intent.Source = entry.RelativePath
			}
		}
	}
	var bad []string
	var invalid []string
	var walk func([]*rollout.Step)
	walk = func(steps []*rollout.Step) {
		for _, step := range steps {
			if step == nil {
				continue
			}
			if strings.TrimSpace(step.OpenAPI) == "" && strings.TrimSpace(step.Source) != "" {
				step.OpenAPI = normalizeAPISourceRef(step.Source)
				step.Source = step.OpenAPI
			}
			if ref := normalizeAPISourceRef(firstNonEmpty(step.Source, step.OpenAPI)); ref != "" {
				if entry, ok := sourceRegistry.get(ref); ok && entry.Err != nil {
					invalid = append(invalid, fmt.Sprintf("%s: %v", ref, entry.Err))
					continue
				}
				if sourceDescriptionTypeForPath(ref) == uws1.SourceDescriptionTypeOpenAPI {
					if !allowed[ref] {
						bad = append(bad, ref)
					} else if strings.TrimSpace(step.Source) != "" {
						step.Source = ref
					} else {
						step.OpenAPI = ref
					}
				} else {
					entry, ok := sourceRegistry.get(ref)
					if !ok {
						bad = append(bad, ref)
					} else if entry.Err != nil {
						invalid = append(invalid, fmt.Sprintf("%s: %v", ref, entry.Err))
					} else {
						step.OpenAPI = entry.RelativePath
						if strings.TrimSpace(step.Source) != "" {
							step.Source = entry.RelativePath
						}
					}
				}
			}
			walk(step.Steps)
			for _, branch := range step.Cases {
				if branch != nil {
					walk(branch.Steps)
				}
			}
			if step.Default != nil {
				walk(step.Default.Steps)
			}
		}
	}
	walk(intent.Steps)
	if len(bad) > 0 {
		sort.Strings(bad)
		return fmt.Errorf("generated intent referenced unavailable step API source documents: %s", strings.Join(bad, ", "))
	}
	if len(invalid) > 0 {
		sort.Strings(invalid)
		return fmt.Errorf("generated intent referenced invalid step API source documents: %s", strings.Join(invalid, "; "))
	}
	return nil
}

func validateIntentRuntimePolicy(intent *rollout.Intent, policy projectPolicy) error {
	if intent == nil {
		return nil
	}
	var blocked []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		typ := strings.ToLower(strings.TrimSpace(step.Type))
		if typ == "" || allowedIntentRuntimeType(typ) {
			if (typ == "cmd" || typ == "ssh") && !policy.AllowedRuntime[typ] {
				name := strings.TrimSpace(step.Name)
				if name == "" {
					name = "<unnamed>"
				}
				blocked = append(blocked, fmt.Sprintf("%s uses %s", name, typ))
			}
			return
		}
		name := strings.TrimSpace(step.Name)
		if name == "" {
			name = "<unnamed>"
		}
		blocked = append(blocked, fmt.Sprintf("%s uses unsupported runtime %s", name, typ))
	})
	if len(blocked) > 0 {
		sort.Strings(blocked)
		return fmt.Errorf("intent uses runtime not allowed by project policy: %s", strings.Join(blocked, "; "))
	}
	return nil
}

func allowedIntentRuntimeType(typ string) bool {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "http", "openapi", "fnct", "cmd", "ssh",
		"sequence", "parallel", "switch", "merge", "loop", "await":
		return true
	default:
		return false
	}
}

func relOrAbs(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func trimForPrompt(value string, max int) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= max {
		return value
	}
	return value[:max] + "..."
}
