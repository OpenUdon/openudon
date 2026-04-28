package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/genelet/ramen/internal/synthesize"
)

type EvalResult struct {
	Name               string           `json:"name"`
	PromptVersion      string           `json:"prompt_version,omitempty"`
	Provider           string           `json:"provider,omitempty"`
	Model              string           `json:"model,omitempty"`
	Mode               string           `json:"mode,omitempty"`
	UsedLegacyExtract  bool             `json:"used_legacy_extract,omitempty"`
	Passed             bool             `json:"passed"`
	AttemptCount       int              `json:"attempt_count,omitempty"`
	AttemptsToPass     int              `json:"attempts_to_pass"`
	RepeatedRepairLoop bool             `json:"repeated_repair_loop,omitempty"`
	FailureClass       string           `json:"failure_class,omitempty"`
	FailingChecks      []string         `json:"failing_checks,omitempty"`
	DurationMs         int64            `json:"duration_ms"`
	PromptTokensApprox int              `json:"prompt_tokens_approx"`
	Error              string           `json:"error,omitempty"`
	ReferenceIssues    []CompareIssue   `json:"reference_issues,omitempty"`
	ReferenceSummary   ReferenceSummary `json:"reference_summary,omitempty"`
	GeneratedDir       string           `json:"generated_dir,omitempty"`
}

type ReferenceSummary struct {
	Advisory int `json:"advisory,omitempty"`
	Warning  int `json:"warning,omitempty"`
	Blocking int `json:"blocking,omitempty"`
}

type ReleaseCriteria struct {
	MinPassRate          float64
	MaxLegacyFallbacks   int
	MaxAttemptsPerBrief  int
	MaxBlockingReference int
	RequireNoSecretScan  bool
}

func DefaultReleaseCriteria() ReleaseCriteria {
	return ReleaseCriteria{
		MinPassRate:          1,
		MaxLegacyFallbacks:   0,
		MaxAttemptsPerBrief:  2,
		MaxBlockingReference: 0,
		RequireNoSecretScan:  true,
	}
}

func RunOne(ctx context.Context, exampleDir string, opts synthesize.Options) EvalResult {
	start := time.Now()
	name := filepath.Base(filepath.Clean(exampleDir))
	result := EvalResult{
		Name:               name,
		Provider:           strings.TrimSpace(opts.Provider),
		Model:              strings.TrimSpace(opts.Model),
		PromptTokensApprox: approximatePromptTokens(exampleDir),
	}
	workDir, err := copyExampleToTemp(exampleDir)
	if err != nil {
		result.DurationMs = time.Since(start).Milliseconds()
		result.Error = err.Error()
		return result
	}
	result.GeneratedDir = workDir
	opts.ExampleDir = workDir

	synthResult, err := synthesize.Synthesize(ctx, opts)
	if synthResult != nil {
		result.GeneratedDir = synthResult.ExampleDir
	}
	refinement := readRefinement(filepath.Join(workDir, "expected", "refinement.json"))
	if refinement != nil {
		result.PromptVersion = refinement.PromptVersion
		result.AttemptCount = len(refinement.Attempts)
		result.RepeatedRepairLoop = result.AttemptCount > 1
		for _, attempt := range refinement.Attempts {
			if attempt.Status == "pass" && result.AttemptsToPass == 0 {
				result.AttemptsToPass = attempt.Number
			}
			if len(attempt.FailingChecks) > 0 {
				result.FailingChecks = attempt.FailingChecks
			}
			if attempt.FailureClass != "" {
				result.FailureClass = attempt.FailureClass
			}
			if attempt.Mode != "" {
				result.Mode = attempt.Mode
				result.UsedLegacyExtract = attempt.Mode == "legacy"
			}
		}
	}
	if err != nil {
		result.Error = err.Error()
	}
	report, assessErr := synthesize.Assess(synthesize.Options{ExampleDir: workDir, SchemaPath: opts.SchemaPath})
	if assessErr == nil && report != nil {
		result.Passed = report.Passed()
		if len(result.FailingChecks) == 0 {
			result.FailingChecks = failingCodes(report)
		}
	} else if err == nil && assessErr != nil {
		result.Error = assessErr.Error()
	}
	referenceIntent := filepath.Join(exampleDir, "reference", "intent.hcl")
	if _, statErr := os.Stat(referenceIntent); statErr == nil {
		issues, compareErr := CompareIntentFiles(filepath.Join(workDir, "workflows", "intent.hcl"), referenceIntent)
		if compareErr != nil {
			issue := CompareIssue{Code: "reference.compare", Detail: compareErr.Error()}
			issue.Severity = referenceIssueSeverity(issue)
			result.ReferenceIssues = []CompareIssue{issue}
		} else {
			result.ReferenceIssues = issues
		}
		result.ReferenceSummary = summarizeReferenceIssues(result.ReferenceIssues)
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result
}

func RunAll(ctx context.Context, evalRoot string, opts synthesize.Options, concurrency int) []EvalResult {
	examples := discoverExamples(evalRoot)
	if concurrency <= 0 {
		concurrency = 2
	}
	if concurrency > len(examples) && len(examples) > 0 {
		concurrency = len(examples)
	}
	type job struct {
		index int
		path  string
	}
	jobs := make(chan job)
	results := make([]EvalResult, len(examples))
	var wg sync.WaitGroup
	for worker := 0; worker < concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for current := range jobs {
				results[current.index] = RunOne(ctx, current.path, opts)
			}
		}()
	}
	for index, path := range examples {
		jobs <- job{index: index, path: path}
	}
	close(jobs)
	wg.Wait()
	return results
}

func discoverExamples(evalRoot string) []string {
	entries, err := os.ReadDir(evalRoot)
	if err != nil {
		return nil
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(evalRoot, entry.Name())
		if _, err := os.Stat(filepath.Join(path, "project.md")); err == nil {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func readRefinement(path string) *synthesize.RefinementReport {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var report synthesize.RefinementReport
	if err := json.Unmarshal(data, &report); err != nil {
		return nil
	}
	return &report
}

func failingCodes(report *synthesize.QualityReport) []string {
	if report == nil {
		return nil
	}
	var out []string
	for _, check := range report.Checks {
		if check.Status == "fail" {
			out = append(out, check.Code)
		}
	}
	return out
}

func copyExampleToTemp(exampleDir string) (string, error) {
	base := filepath.Base(filepath.Clean(exampleDir))
	root, err := os.MkdirTemp("", "ramen-eval-"+base+"-")
	if err != nil {
		return "", err
	}
	dst := filepath.Join(root, base)
	if err := copyTree(exampleDir, dst); err != nil {
		return "", err
	}
	return dst, nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			name := entry.Name()
			if name == "workflows" || name == "expected" {
				return filepath.SkipDir
			}
			return os.MkdirAll(target, 0o755)
		}
		if entry.Type()&os.ModeType != 0 {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			_ = in.Close()
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			_ = in.Close()
			_ = out.Close()
			return err
		}
		if err := in.Close(); err != nil {
			_ = out.Close()
			return err
		}
		return out.Close()
	})
}

func approximatePromptTokens(exampleDir string) int {
	var bytes int64
	for _, rel := range []string{"project.md", "openapi"} {
		path := filepath.Join(exampleDir, rel)
		_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
			if err != nil || entry == nil || entry.IsDir() {
				return nil
			}
			info, err := entry.Info()
			if err == nil {
				bytes += info.Size()
			}
			return nil
		})
	}
	if bytes == 0 {
		return 0
	}
	return int(bytes / 4)
}

func FindPreviousRun(outPath string) (string, error) {
	dir := filepath.Dir(outPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	current, _ := filepath.Abs(outPath)
	var candidates []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		abs, _ := filepath.Abs(path)
		if abs == current {
			continue
		}
		candidates = append(candidates, path)
	}
	sort.Strings(candidates)
	if len(candidates) == 0 {
		return "", nil
	}
	return candidates[len(candidates)-1], nil
}

func RegressionError(current []EvalResult, previous []EvalResult) error {
	if len(previous) == 0 {
		return nil
	}
	currentRate := passRate(current)
	previousRate := passRate(previous)
	if currentRate < previousRate {
		return fmt.Errorf("eval pass rate regressed from %.1f%% to %.1f%%", previousRate*100, currentRate*100)
	}
	if currentLegacy, previousLegacy := legacyExtractCount(current), legacyExtractCount(previous); currentLegacy > previousLegacy {
		return fmt.Errorf("legacy extractJSON fallback count regressed from %d to %d", previousLegacy, currentLegacy)
	}
	currentByName := map[string]EvalResult{}
	for _, result := range current {
		currentByName[result.Name] = result
	}
	var regressions []string
	var referenceRegressions []string
	for _, prior := range previous {
		now, ok := currentByName[prior.Name]
		if !ok {
			continue
		}
		if prior.Passed && !now.Passed {
			regressions = append(regressions, prior.Name)
		}
		if blockingReferenceCount(now) > blockingReferenceCount(prior) {
			referenceRegressions = append(referenceRegressions, prior.Name)
		}
	}
	if len(regressions) > 0 {
		return fmt.Errorf("previously passing eval brief(s) failed: %s", strings.Join(regressions, ", "))
	}
	if len(referenceRegressions) > 0 {
		return fmt.Errorf("blocking reference issue count regressed for brief(s): %s", strings.Join(referenceRegressions, ", "))
	}
	return nil
}

func ReleaseCriteriaError(results []EvalResult, criteria ReleaseCriteria) error {
	if criteria.MinPassRate == 0 {
		criteria.MinPassRate = 1
	}
	if criteria.MaxAttemptsPerBrief == 0 {
		criteria.MaxAttemptsPerBrief = 2
	}
	if criteria.MaxBlockingReference < 0 {
		criteria.MaxBlockingReference = 0
	}
	var failures []string
	if rate := passRate(results); rate < criteria.MinPassRate {
		failures = append(failures, fmt.Sprintf("pass rate %.1f%% below required %.1f%%", rate*100, criteria.MinPassRate*100))
	}
	if legacy := legacyExtractCount(results); legacy > criteria.MaxLegacyFallbacks {
		failures = append(failures, fmt.Sprintf("legacy fallback count %d exceeds allowed %d", legacy, criteria.MaxLegacyFallbacks))
	}
	var attemptFailures []string
	var referenceFailures []string
	var secretFailures []string
	for _, result := range results {
		attempts := result.AttemptCount
		if attempts == 0 {
			attempts = result.AttemptsToPass
		}
		if attempts > criteria.MaxAttemptsPerBrief {
			attemptFailures = append(attemptFailures, fmt.Sprintf("%s=%d", result.Name, attempts))
		}
		if blocking := blockingReferenceCount(result); blocking > criteria.MaxBlockingReference {
			referenceFailures = append(referenceFailures, fmt.Sprintf("%s=%d", result.Name, blocking))
		}
		if criteria.RequireNoSecretScan && containsFailureCode(result.FailingChecks, "artifacts.no_secrets") {
			secretFailures = append(secretFailures, result.Name)
		}
	}
	if len(attemptFailures) > 0 {
		sort.Strings(attemptFailures)
		failures = append(failures, fmt.Sprintf("attempt count exceeds %d: %s", criteria.MaxAttemptsPerBrief, strings.Join(attemptFailures, ", ")))
	}
	if len(referenceFailures) > 0 {
		sort.Strings(referenceFailures)
		failures = append(failures, fmt.Sprintf("blocking reference issues exceed %d: %s", criteria.MaxBlockingReference, strings.Join(referenceFailures, ", ")))
	}
	if len(secretFailures) > 0 {
		sort.Strings(secretFailures)
		failures = append(failures, "secret-scan failures: "+strings.Join(secretFailures, ", "))
	}
	if len(failures) > 0 {
		return fmt.Errorf("release criteria failed: %s", strings.Join(failures, "; "))
	}
	return nil
}

func containsFailureCode(values []string, code string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == code {
			return true
		}
	}
	return false
}

func summarizeReferenceIssues(issues []CompareIssue) ReferenceSummary {
	var summary ReferenceSummary
	for _, issue := range issues {
		switch normalizedReferenceSeverity(issue) {
		case "advisory":
			summary.Advisory++
		case "blocking":
			summary.Blocking++
		default:
			summary.Warning++
		}
	}
	return summary
}

func normalizedReferenceSeverity(issue CompareIssue) string {
	severity := strings.ToLower(strings.TrimSpace(issue.Severity))
	if severity == "" {
		severity = referenceIssueSeverity(issue)
	}
	switch severity {
	case "advisory", "warning", "blocking":
		return severity
	default:
		return "warning"
	}
}

func blockingReferenceCount(result EvalResult) int {
	if result.ReferenceSummary.Blocking > 0 {
		return result.ReferenceSummary.Blocking
	}
	return summarizeReferenceIssues(result.ReferenceIssues).Blocking
}

func passRate(results []EvalResult) float64 {
	if len(results) == 0 {
		return 0
	}
	var passed int
	for _, result := range results {
		if result.Passed {
			passed++
		}
	}
	return float64(passed) / float64(len(results))
}
