package synthesize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/openapidisco"
)

type QualityReport struct {
	Status    string         `json:"status"`
	Example   string         `json:"example"`
	Artifacts Result         `json:"artifacts"`
	Checks    []QualityCheck `json:"checks"`
}

type QualityCheck struct {
	Code        string `json:"code"`
	Status      string `json:"status"`
	Message     string `json:"message"`
	Detail      string `json:"detail,omitempty"`
	FailureKind string `json:"failure_kind,omitempty"`
}

func (r *QualityReport) Passed() bool {
	return r != nil && r.Status == "pass"
}

func Assess(opts Options) (*QualityReport, error) {
	return AssessContext(context.Background(), opts)
}

func AssessContext(ctx context.Context, opts Options) (*QualityReport, error) {
	return assessContext(ctx, opts, true)
}

func AssessCurrent(ctx context.Context, opts Options) (*QualityReport, error) {
	return assessContext(ctx, opts, false)
}

func assessContext(ctx context.Context, opts Options, writeReport bool) (*QualityReport, error) {
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
	report := &QualityReport{
		Status:    "pass",
		Example:   exampleDir,
		Artifacts: result,
	}

	projectBytes, projectErr := os.ReadFile(result.ProjectPath)
	projectText := string(projectBytes)
	policy := analyzeProject(projectText)
	if projectErr != nil {
		report.add("project.present", "fail", "project.md is required", projectErr.Error())
	} else {
		report.add("project.present", "pass", "project.md is readable", "")
		addProjectAuthoringChecks(report, projectText)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	candidates, err := openapidisco.LocalFiles(filepath.Join(exampleDir, "openapi"), exampleDir, projectText)
	apiSourcePaths, sourceErr := collectLocalAPISourcePaths(exampleDir)
	if err != nil && !(policy.NoOpenAPI && errors.Is(err, os.ErrNotExist)) {
		report.add("openapi.local", "fail", "OpenAPI directory could not be scanned", err.Error())
	} else if sourceErr != nil {
		report.add("openapi.local", "fail", "API source documents could not be scanned", sourceErr.Error())
	} else if policy.NoOpenAPI {
		result.OpenAPICandidates = candidates
		report.Artifacts = result
		report.add("openapi.local", "pass", "project explicitly declares OpenAPI is not required", candidateList(candidates))
	} else if len(candidates) == 0 && len(apiSourcePaths) == 0 {
		report.add("openapi.local", "fail", "no local API source documents are available", "Add a valid OpenAPI document under openapi/ or a first-class source under google-discovery/ or aws-smithy/.")
	} else {
		detail := candidateList(candidates)
		if sourceErr == nil && len(apiSourcePaths) > 0 {
			detail = strings.TrimSpace(detail + "\n" + strings.Join(apiSourcePaths, "\n"))
		}
		report.add("openapi.local", "pass", fmt.Sprintf("%d OpenAPI document(s), %d API source document(s) available", len(candidates), len(apiSourcePaths)), detail)
		result.OpenAPICandidates = candidates
		if primary, err := openapidisco.SelectPrimary(candidates); err == nil {
			result.PrimaryOpenAPI = primary.RelativePath
		}
		report.Artifacts = result
	}

	expectedPlan := assessWorkflowPlan(report, result)
	assessDiscoveryReport(report, result.DiscoveryJSONPath)
	intent, intentOK := assessIntent(report, result.IntentPath, exampleDir, candidates, result.PrimaryOpenAPI, policy)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	planOK := assessWorkflow(report, result.WorkflowPath, exampleDir, intent, policy, expectedPlan)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	assessUWS(report, result.UWSPath, opts.SchemaPath, exampleDir, expectedPlan)
	sideEffects := sideEffectProfileForOpenAPI(policy, intent, candidates, result.PrimaryOpenAPI)
	assessSideEffectProfile(report, sideEffects)
	assessSideEffectRetryPolicy(report, sideEffects, policy, expectedPlan)
	assessReview(report, result.ReviewPath, sideEffects, policy, expectedPlan)
	assessReviewHandoff(report, result.ReviewHandoffPath, sideEffects, policy, expectedPlan)
	assessSecrets(report, result)
	assessConversionDiagnostics(report, filepath.Join(exampleDir, "expected", "diagnostics.json"))

	if intentOK && planOK {
		report.add("quality.review", "pass", "workflow.hcl passed deterministic v1 quality gates", "")
	}
	report.finalize()
	if writeReport {
		if err := writeQualityFiles(result, report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

type conversionDiagnostic struct {
	Code          string `json:"code"`
	Severity      string `json:"severity"`
	Message       string `json:"message"`
	Address       string `json:"address"`
	TodoID        string `json:"todo_id"`
	StrictFailure bool   `json:"strict_failure"`
}

func collectLocalAPISourcePaths(exampleDir string) ([]string, error) {
	registry, err := newLocalAPISourceRegistry(exampleDir, nil)
	if err != nil {
		return nil, err
	}
	paths := registry.nativePaths()
	for _, path := range paths {
		if strings.Contains(path, " invalid: ") {
			return paths, fmt.Errorf("invalid API source document: %s", path)
		}
	}
	return paths, nil
}

func assessConversionDiagnostics(report *QualityReport, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			report.add("conversion.diagnostics", "pass", "no conversion diagnostics artifact is present", "")
			return
		}
		report.add("conversion.diagnostics", "fail", "conversion diagnostics could not be read", err.Error())
		return
	}
	var diagnostics []conversionDiagnostic
	if err := json.Unmarshal(data, &diagnostics); err != nil {
		report.add("conversion.diagnostics", "fail", "conversion diagnostics must be valid JSON", err.Error())
		return
	}
	var failures []string
	for _, diagnostic := range diagnostics {
		if !diagnostic.StrictFailure {
			continue
		}
		parts := []string{strings.TrimSpace(diagnostic.Code)}
		if strings.TrimSpace(diagnostic.Address) != "" {
			parts = append(parts, strings.TrimSpace(diagnostic.Address))
		}
		if strings.TrimSpace(diagnostic.TodoID) != "" {
			parts = append(parts, strings.TrimSpace(diagnostic.TodoID))
		}
		if strings.TrimSpace(diagnostic.Message) != "" {
			parts = append(parts, strings.TrimSpace(diagnostic.Message))
		}
		failures = append(failures, strings.Join(parts, ": "))
	}
	if len(failures) > 0 {
		report.add("conversion.diagnostics", "fail", "conversion diagnostics contain unresolved TODOs or unsafe assumptions", strings.Join(sortedCopy(failures), "; "))
		return
	}
	report.add("conversion.diagnostics", "pass", "conversion diagnostics contain no strict failures", "")
}
