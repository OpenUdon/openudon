package eval

import (
	"fmt"
	"sort"
	"strings"
)

type RunComparison struct {
	PreviousPath            string          `json:"previous_path,omitempty"`
	PreviousPassRate        float64         `json:"previous_pass_rate"`
	CurrentPassRate         float64         `json:"current_pass_rate"`
	PassRateDelta           float64         `json:"pass_rate_delta"`
	LegacyFallbackDelta     int             `json:"legacy_fallback_delta"`
	BlockingReferenceDelta  int             `json:"blocking_reference_delta"`
	PromptTokensApproxDelta int             `json:"prompt_tokens_approx_delta"`
	DurationMsDelta         int64           `json:"duration_ms_delta"`
	NewlyFailingBriefs      []string        `json:"newly_failing_briefs,omitempty"`
	FixedBriefs             []string        `json:"fixed_briefs,omitempty"`
	AttemptRegressions      []BriefIntDelta `json:"attempt_regressions,omitempty"`
	AttemptImprovements     []BriefIntDelta `json:"attempt_improvements,omitempty"`
	NewFailingChecks        []string        `json:"new_failing_checks,omitempty"`
	ResolvedFailingChecks   []string        `json:"resolved_failing_checks,omitempty"`
	HasRegression           bool            `json:"has_regression,omitempty"`
}

type BriefIntDelta struct {
	Name     string `json:"name"`
	Previous int    `json:"previous"`
	Current  int    `json:"current"`
}

func CompareRuns(current []EvalResult, previous []EvalResult, previousPath string) RunComparison {
	comparison := RunComparison{
		PreviousPath:            strings.TrimSpace(previousPath),
		PreviousPassRate:        passRate(previous),
		CurrentPassRate:         passRate(current),
		LegacyFallbackDelta:     legacyExtractCount(current) - legacyExtractCount(previous),
		BlockingReferenceDelta:  blockingReferenceTotal(current) - blockingReferenceTotal(previous),
		PromptTokensApproxDelta: promptTokensApproxTotal(current) - promptTokensApproxTotal(previous),
		DurationMsDelta:         durationMsTotal(current) - durationMsTotal(previous),
	}
	comparison.PassRateDelta = comparison.CurrentPassRate - comparison.PreviousPassRate
	if len(previous) == 0 {
		return comparison
	}

	currentByName := resultsByName(current)
	previousByName := resultsByName(previous)
	for name, prior := range previousByName {
		now, ok := currentByName[name]
		if !ok {
			continue
		}
		if prior.Passed && !now.Passed {
			comparison.NewlyFailingBriefs = append(comparison.NewlyFailingBriefs, name)
		}
		if !prior.Passed && now.Passed {
			comparison.FixedBriefs = append(comparison.FixedBriefs, name)
		}
		previousAttempts := evalAttempts(prior)
		currentAttempts := evalAttempts(now)
		if previousAttempts > 0 && currentAttempts > previousAttempts {
			comparison.AttemptRegressions = append(comparison.AttemptRegressions, BriefIntDelta{Name: name, Previous: previousAttempts, Current: currentAttempts})
		}
		if currentAttempts > 0 && previousAttempts > currentAttempts {
			comparison.AttemptImprovements = append(comparison.AttemptImprovements, BriefIntDelta{Name: name, Previous: previousAttempts, Current: currentAttempts})
		}
	}

	comparison.NewFailingChecks = stringSetDelta(failingCheckSet(current), failingCheckSet(previous))
	comparison.ResolvedFailingChecks = stringSetDelta(failingCheckSet(previous), failingCheckSet(current))
	sort.Strings(comparison.NewlyFailingBriefs)
	sort.Strings(comparison.FixedBriefs)
	sortBriefIntDeltas(comparison.AttemptRegressions)
	sortBriefIntDeltas(comparison.AttemptImprovements)
	comparison.HasRegression = comparison.PassRateDelta < 0 ||
		comparison.LegacyFallbackDelta > 0 ||
		comparison.BlockingReferenceDelta > 0 ||
		len(comparison.NewlyFailingBriefs) > 0 ||
		len(comparison.AttemptRegressions) > 0 ||
		len(comparison.NewFailingChecks) > 0
	return comparison
}

func ComparisonRegressionError(comparison *RunComparison) error {
	if comparison == nil || !comparison.HasRegression {
		return nil
	}
	if comparison.PassRateDelta < 0 {
		return fmt.Errorf("eval pass rate regressed from %.1f%% to %.1f%%", comparison.PreviousPassRate*100, comparison.CurrentPassRate*100)
	}
	if comparison.LegacyFallbackDelta > 0 {
		return fmt.Errorf("legacy extractJSON fallback count regressed by %d", comparison.LegacyFallbackDelta)
	}
	if len(comparison.NewlyFailingBriefs) > 0 {
		return fmt.Errorf("previously passing eval brief(s) failed: %s", strings.Join(comparison.NewlyFailingBriefs, ", "))
	}
	if comparison.BlockingReferenceDelta > 0 {
		return fmt.Errorf("blocking reference issue count regressed by %d", comparison.BlockingReferenceDelta)
	}
	if len(comparison.AttemptRegressions) > 0 {
		return fmt.Errorf("eval attempt count regressed for brief(s): %s", briefIntDeltaNames(comparison.AttemptRegressions))
	}
	if len(comparison.NewFailingChecks) > 0 {
		return fmt.Errorf("new eval failing check(s): %s", strings.Join(comparison.NewFailingChecks, ", "))
	}
	return nil
}

func resultsByName(results []EvalResult) map[string]EvalResult {
	out := map[string]EvalResult{}
	for _, result := range results {
		name := strings.TrimSpace(result.Name)
		if name != "" {
			out[name] = result
		}
	}
	return out
}

func evalAttempts(result EvalResult) int {
	if result.AttemptCount > 0 {
		return result.AttemptCount
	}
	return result.AttemptsToPass
}

func blockingReferenceTotal(results []EvalResult) int {
	var total int
	for _, result := range results {
		total += blockingReferenceCount(result)
	}
	return total
}

func promptTokensApproxTotal(results []EvalResult) int {
	var total int
	for _, result := range results {
		total += result.PromptTokensApprox
	}
	return total
}

func durationMsTotal(results []EvalResult) int64 {
	var total int64
	for _, result := range results {
		total += result.DurationMs
	}
	return total
}

func failingCheckSet(results []EvalResult) map[string]bool {
	out := map[string]bool{}
	for _, result := range results {
		for _, check := range result.FailingChecks {
			check = strings.TrimSpace(check)
			if check != "" {
				out[check] = true
			}
		}
	}
	return out
}

func stringSetDelta(values map[string]bool, baseline map[string]bool) []string {
	var out []string
	for value := range values {
		if !baseline[value] {
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func sortBriefIntDeltas(deltas []BriefIntDelta) {
	sort.Slice(deltas, func(i, j int) bool {
		return deltas[i].Name < deltas[j].Name
	})
}

func briefIntDeltaNames(deltas []BriefIntDelta) string {
	parts := make([]string, 0, len(deltas))
	for _, delta := range deltas {
		parts = append(parts, fmt.Sprintf("%s=%d>%d", delta.Name, delta.Current, delta.Previous))
	}
	return strings.Join(parts, ", ")
}
