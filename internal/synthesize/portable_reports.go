package synthesize

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/openapidisco"
)

// portableQualityReport returns the package-facing DTO. Assessment keeps
// absolute paths in memory for diagnostics, but persisted evidence contains
// only a stable example label and package-relative artifact paths.
func portableQualityReport(result Result, report *QualityReport) *QualityReport {
	if report == nil {
		return nil
	}
	portable := *report
	portable.Example = filepath.Base(filepath.Clean(result.ExampleDir))
	portable.Artifacts = portableResult(result)
	portable.Checks = append([]QualityCheck(nil), report.Checks...)
	for i := range portable.Checks {
		portable.Checks[i].Message = sanitizeReportText(result.ExampleDir, portable.Checks[i].Message)
		portable.Checks[i].Detail = sanitizeReportText(result.ExampleDir, portable.Checks[i].Detail)
	}
	return &portable
}

func portableRefinementReport(result Result, report *RefinementReport) *RefinementReport {
	if report == nil {
		return nil
	}
	portable := *report
	portable.Example = filepath.Base(filepath.Clean(result.ExampleDir))
	portable.PromptSnapshot = sanitizeReportText(result.ExampleDir, portable.PromptSnapshot)
	portable.Attempts = append([]RefinementAttempt(nil), report.Attempts...)
	for i := range portable.Attempts {
		portable.Attempts[i].FailingChecks = append([]string(nil), portable.Attempts[i].FailingChecks...)
		portable.Attempts[i].Detail = sanitizeReportText(result.ExampleDir, portable.Attempts[i].Detail)
		portable.Attempts[i].StopReason = sanitizeReportText(result.ExampleDir, portable.Attempts[i].StopReason)
	}
	portable.StopReason = sanitizeReportText(result.ExampleDir, portable.StopReason)
	return &portable
}

func portableDiscoveryReport(result Result, report openapidisco.DiscoveryReport) openapidisco.DiscoveryReport {
	portable := report
	portable.Attempts = append([]openapidisco.DiscoveryAttempt(nil), report.Attempts...)
	for i := range portable.Attempts {
		portable.Attempts[i].Source = portableDiscoverySource(result.ExampleDir, portable.Attempts[i].Source)
		portable.Attempts[i].Detail = sanitizeReportText(result.ExampleDir, portable.Attempts[i].Detail)
	}
	portable.Diagnostics = append(report.Diagnostics[:0:0], report.Diagnostics...)
	for i := range portable.Diagnostics {
		diagnostic := &portable.Diagnostics[i]
		if filepath.IsAbs(strings.TrimSpace(diagnostic.Path)) {
			diagnostic.Path = packageRelativeReportPath(result.ExampleDir, diagnostic.Path)
		} else {
			diagnostic.Path = sanitizeReportText(result.ExampleDir, diagnostic.Path)
		}
		diagnostic.Message = sanitizeReportText(result.ExampleDir, diagnostic.Message)
		diagnostic.Remediation = sanitizeReportText(result.ExampleDir, diagnostic.Remediation)
	}
	return portable
}

func portableDiscoverySource(root, source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	parsed, err := url.Parse(source)
	if err == nil && parsed.IsAbs() && parsed.Host != "" {
		parsed.User = nil
		parsed.RawQuery = ""
		parsed.ForceQuery = false
		parsed.Fragment = ""
		return parsed.String()
	}
	if filepath.IsAbs(source) {
		return packageRelativeReportPath(root, source)
	}
	return sanitizeReportText(root, source)
}

func portableResult(result Result) Result {
	root := result.ExampleDir
	result.ExampleDir = "."
	result.ProjectPath = packageRelativeReportPath(root, result.ProjectPath)
	result.IntentPath = packageRelativeReportPath(root, result.IntentPath)
	result.WorkflowPath = packageRelativeReportPath(root, result.WorkflowPath)
	result.UWSPath = packageRelativeReportPath(root, result.UWSPath)
	result.PlanJSONPath = packageRelativeReportPath(root, result.PlanJSONPath)
	result.PlanMDPath = packageRelativeReportPath(root, result.PlanMDPath)
	result.DiscoveryJSONPath = packageRelativeReportPath(root, result.DiscoveryJSONPath)
	result.DataPath = packageRelativeReportPath(root, result.DataPath)
	result.RefinementJSONPath = packageRelativeReportPath(root, result.RefinementJSONPath)
	result.RefinementMDPath = packageRelativeReportPath(root, result.RefinementMDPath)
	result.ReviewPath = packageRelativeReportPath(root, result.ReviewPath)
	result.ReviewHandoffPath = packageRelativeReportPath(root, result.ReviewHandoffPath)
	result.QualityJSONPath = packageRelativeReportPath(root, result.QualityJSONPath)
	result.QualityMDPath = packageRelativeReportPath(root, result.QualityMDPath)
	candidates := result.OpenAPICandidates
	result.OpenAPICandidates = make([]openapidisco.Candidate, len(candidates))
	for i, candidate := range candidates {
		rel := packageRelativeReportPath(root, firstNonEmpty(candidate.RelativePath, candidate.Path))
		result.OpenAPICandidates[i] = openapidisco.Candidate{
			Path:         rel,
			RelativePath: rel,
			Title:        sanitizeReportText(root, candidate.Title),
			Description:  sanitizeReportText(root, candidate.Description),
			Source:       portableCandidateSource(root, candidate.Source),
			Score:        candidate.Score,
		}
	}
	return result
}

func portableCandidateSource(root, source string) string {
	source = sanitizeReportText(root, strings.TrimSpace(source))
	prefix, rawURL, ok := strings.Cut(source, ":")
	if !ok || !strings.EqualFold(prefix, "url") {
		return source
	}
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "url"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	return "url:" + parsed.String()
}

func packageRelativeReportPath(root, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !filepath.IsAbs(path) {
		clean := filepath.Clean(path)
		if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(filepath.Base(clean))
		}
		return filepath.ToSlash(clean)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.ToSlash(filepath.Base(path))
	}
	return filepath.ToSlash(filepath.Clean(rel))
}

func sanitizeReportText(root, value string) string {
	if root == "" || value == "" {
		return value
	}
	clean := filepath.Clean(root)
	value = strings.ReplaceAll(value, clean, ".")
	if slash := filepath.ToSlash(clean); slash != clean {
		value = strings.ReplaceAll(value, slash, ".")
	}
	return value
}
