package elicitor

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browserverify"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// AttachBrowserVerifications strict-decodes explicit value-free Browsertools
// reports, matches each report to exactly one selected profile digest, and
// retains only a revalidated normalized summary plus the operator-local path.
func AttachBrowserVerifications(sources []SourceMaterialization, reportPaths []string, at time.Time) ([]SourceMaterialization, error) {
	if at.IsZero() {
		return nil, fmt.Errorf("browser verification time is required")
	}
	result := normalizeSourcePlan(sources)
	profiles := map[string][]int{}
	profileValues := map[int]*profile.Profile{}
	for index := range result {
		if result[index].Kind != browserSourceFamily {
			if len(result[index].BrowserVerifications) != 0 {
				return nil, fmt.Errorf("browser verification evidence is attached to non-capability source %q", result[index].ID)
			}
			continue
		}
		data, err := SourceMaterializationContent(result[index], at)
		if err != nil {
			return nil, fmt.Errorf("browser verification profile %s: %w", result[index].ID, err)
		}
		value, err := parseBrowserProfile(result[index].TargetPath, data)
		if err != nil {
			return nil, fmt.Errorf("browser verification profile %s: %w", result[index].ID, err)
		}
		digest, err := browserverify.ProfileDigest(value)
		if err != nil {
			return nil, err
		}
		profiles[digest] = append(profiles[digest], index)
		profileValues[index] = value
	}

	type requestedReport struct {
		path     string
		expected int
		summary  *browserverify.Summary
	}
	requests := make([]requestedReport, 0, len(reportPaths))
	// Retained attachments must precede repeated CLI paths. Stable sorting then
	// keeps the reviewed summary authoritative when both name the same file,
	// so a resumed invocation cannot silently treat replacement as a new input.
	for index := range result {
		for attachmentIndex := range result[index].BrowserVerifications {
			attachment := &result[index].BrowserVerifications[attachmentIndex]
			requests = append(requests, requestedReport{path: attachment.SourcePath, expected: index, summary: &attachment.Summary})
		}
		result[index].BrowserVerifications = nil
	}
	for _, path := range reportPaths {
		requests = append(requests, requestedReport{path: path, expected: -1})
	}
	if len(requests) > browserverify.MaxReports {
		return nil, fmt.Errorf("browser verification reports exceed limit %d", browserverify.MaxReports)
	}
	sort.SliceStable(requests, func(i, j int) bool { return filepath.Clean(requests[i].path) < filepath.Clean(requests[j].path) })
	seenPath := map[string]requestedReport{}
	logical := map[int]map[string]browserverify.Summary{}
	for _, request := range requests {
		absPath, err := filepath.Abs(strings.TrimSpace(request.path))
		if err != nil || strings.TrimSpace(request.path) == "" {
			return nil, fmt.Errorf("browser verification report path %q is invalid", request.path)
		}
		absPath = filepath.Clean(absPath)
		if prior, duplicate := seenPath[absPath]; duplicate {
			if prior.expected >= 0 && request.expected >= 0 && prior.expected != request.expected {
				return nil, fmt.Errorf("browser verification report %s is attached to multiple profiles", absPath)
			}
			if prior.summary != nil && request.summary != nil && !reflect.DeepEqual(*prior.summary, *request.summary) {
				return nil, fmt.Errorf("browser verification report %s has conflicting retained summaries", absPath)
			}
			continue
		}
		request.path = absPath
		seenPath[absPath] = request
		_, profileDigest, _, err := browserverify.ReadVersionAndProfileDigest(absPath)
		if err != nil {
			return nil, err
		}
		matches := profiles[profileDigest]
		if len(matches) == 0 {
			return nil, fmt.Errorf("browser verification report %s does not match a selected browser profile", absPath)
		}
		if len(matches) != 1 {
			return nil, fmt.Errorf("browser verification report %s matches multiple selected browser profiles", absPath)
		}
		index := matches[0]
		if request.expected >= 0 && request.expected != index {
			return nil, fmt.Errorf("browser verification report %s changed its selected profile binding", absPath)
		}
		summary, err := browserverify.Inspect(absPath, profileValues[index], at)
		if err != nil {
			return nil, err
		}
		if request.summary != nil && !reflect.DeepEqual(*request.summary, summary) {
			return nil, fmt.Errorf("browser verification report %s changed after review", absPath)
		}
		if logical[index] == nil {
			logical[index] = map[string]browserverify.Summary{}
		}
		key := browserverify.LogicalKey(summary)
		if prior, duplicate := logical[index][key]; duplicate {
			if !browserverify.EquivalentFacts(prior, summary) {
				return nil, fmt.Errorf("browser verification reports conflict for profile %s and action set %s", result[index].ID, strings.Join(summary.Actions, ","))
			}
			continue
		}
		logical[index][key] = summary
		result[index].BrowserVerifications = append(result[index].BrowserVerifications, browserverify.Attachment{SourcePath: absPath, Summary: summary})
		if len(result[index].BrowserVerifications) > browserverify.MaxReportsPerProfile {
			return nil, fmt.Errorf("browser profile %s has more than %d verification reports", result[index].ID, browserverify.MaxReportsPerProfile)
		}
	}
	return normalizeSourcePlan(result), nil
}

// RevalidateBrowserVerifications reopens every attached external report at the
// approval boundary and rejects removal, replacement, or fact changes.
func RevalidateBrowserVerifications(sources []SourceMaterialization, at time.Time) ([]SourceMaterialization, error) {
	return AttachBrowserVerifications(sources, nil, at)
}

// ValidateBrowserVerificationCoverage keeps verification optional, but when an
// operator attaches it, every report must pass and its successful action union
// must cover each selected workflow action for that profile.
func ValidateBrowserVerificationCoverage(session Session) error {
	byTarget := map[string]SourceMaterialization{}
	covered := map[string]map[string]bool{}
	for _, source := range normalizeSourcePlan(session.SourcePlan) {
		if source.Kind != browserSourceFamily {
			continue
		}
		target := normalizeBrowserVerificationSourceRef(source.TargetPath)
		byTarget[target] = source
		for _, attachment := range source.BrowserVerifications {
			if !attachment.Summary.OK {
				return fmt.Errorf("browser verification for source %s records a failed check", source.ID)
			}
			if covered[target] == nil {
				covered[target] = map[string]bool{}
			}
			for _, action := range attachment.Summary.Actions {
				covered[target][action] = true
			}
		}
	}
	var coverageErrors []string
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil || !strings.EqualFold(strings.TrimSpace(step.Type), "browser") {
			return
		}
		ref := normalizeBrowserVerificationSourceRef(firstNonEmpty(step.Source, step.OpenAPI, session.Intent.Source, session.Intent.OpenAPI))
		source, ok := byTarget[ref]
		if !ok || len(source.BrowserVerifications) == 0 {
			return
		}
		action := strings.TrimSpace(step.Operation)
		if !covered[ref][action] {
			coverageErrors = append(coverageErrors, fmt.Sprintf("browser source %s verification does not cover selected action %q", source.ID, action))
		}
	})
	if len(coverageErrors) > 0 {
		sort.Strings(coverageErrors)
		return fmt.Errorf("%s", strings.Join(coverageErrors, "; "))
	}
	return nil
}

func normalizeBrowserVerificationSourceRef(value string) string {
	value = filepath.ToSlash(strings.TrimSpace(value))
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	if value == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(value))
}
