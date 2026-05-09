package synthesize

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/genelet/ramen/internal/openapidisco"
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
	if err != nil && !(policy.NoOpenAPI && errors.Is(err, os.ErrNotExist)) {
		report.add("openapi.local", "fail", "OpenAPI directory could not be scanned", err.Error())
	} else if policy.NoOpenAPI {
		result.OpenAPICandidates = candidates
		report.Artifacts = result
		report.add("openapi.local", "pass", "project explicitly declares OpenAPI is not required", candidateList(candidates))
	} else if len(candidates) == 0 {
		report.add("openapi.local", "fail", "no local OpenAPI documents are available", "Add a valid .json, .yaml, or .yml OpenAPI document under openapi/, or rerun synthesize with an explicit URL in project.md.")
	} else {
		report.add("openapi.local", "pass", fmt.Sprintf("%d OpenAPI document(s) available", len(candidates)), candidateList(candidates))
		result.OpenAPICandidates = candidates
		if primary, err := openapidisco.SelectPrimary(candidates); err == nil {
			result.PrimaryOpenAPI = primary.RelativePath
		}
		report.Artifacts = result
	}

	expectedPlan := assessWorkflowPlan(report, result)
	assessDiscoveryReport(report, result.DiscoveryJSONPath)
	intent, intentOK := assessIntent(report, result.IntentPath, candidates, result.PrimaryOpenAPI, policy)
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
	assessSymphonyHandoff(report, result.SymphonyHandoffPath, sideEffects, policy, expectedPlan)
	assessSecrets(report, result)

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
