package synthesize

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/genelet/ramen/internal/openapidisco"
	"github.com/genelet/udon/pkg/rollout"
	"github.com/tabilet/uws/uws1"
)

func assessSecrets(report *QualityReport, result Result) {
	paths := []string{result.ProjectPath, result.IntentPath, result.WorkflowPath, result.UWSPath, result.PlanJSONPath, result.PlanMDPath, result.DiscoveryJSONPath, result.RefinementJSONPath, result.RefinementMDPath, result.ReviewPath, result.SymphonyHandoffPath}
	var hits []string
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if containsSecretLikeToken(data) {
			hits = append(hits, relOrAbs(result.ExampleDir, path))
		}
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

const (
	minAssignedSecretLength = 12
)

var providerSecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`sk-ant-api[0-9A-Za-z_-]*-[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`sk-(?:proj-)?[0-9A-Za-z_-]{20,}`),
	regexp.MustCompile(`ghp_[0-9A-Za-z]{36,}`),
	regexp.MustCompile(`github_pat_[0-9A-Za-z_]{20,}`),
	regexp.MustCompile(`(?:AKIA|ASIA)[0-9A-Z]{16}`),
}

var (
	jwtCandidatePattern         = regexp.MustCompile(`[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	sensitiveAssignmentPattern  = regexp.MustCompile(`(?i)\b([A-Za-z0-9_.-]*(?:api[_-]?key|apikey|app[_-]?id|appid|token|secret|password|authorization)[A-Za-z0-9_.-]*)\s*[:=]\s*["']([^"'\r\n]+)["']`)
	workflowReferencePattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*(?:\[[0-9]+\])?(?:\.[A-Za-z_][A-Za-z0-9_-]*(?:\[[0-9]+\])?)*$`)
	tokenShapedValuePattern     = regexp.MustCompile(`^[A-Za-z0-9_+/=-]+$`)
	tokenSourceAssignmentSuffix = regexp.MustCompile(`(?i)(?:^|[_\-.])from$`)
)

func containsSecretLikeToken(data []byte) bool {
	for _, pattern := range providerSecretPatterns {
		if pattern.Match(data) {
			return true
		}
	}
	if containsValidatedJWT(data) {
		return true
	}
	for _, match := range sensitiveAssignmentPattern.FindAllSubmatch(data, -1) {
		if len(match) < 3 {
			continue
		}
		if isSensitiveSourceAssignment(string(match[1])) {
			continue
		}
		if isAssignedSecretLiteral(string(match[2])) {
			return true
		}
	}
	return false
}

func containsValidatedJWT(data []byte) bool {
	for _, candidate := range jwtCandidatePattern.FindAll(data, -1) {
		if isValidatedJWT(string(candidate)) {
			return true
		}
	}
	return false
}

func isValidatedJWT(candidate string) bool {
	parts := strings.Split(candidate, ".")
	if len(parts) != 3 {
		return false
	}
	header := map[string]any{}
	if !decodeBase64URLJSON(parts[0], &header) {
		return false
	}
	if _, ok := header["alg"]; !ok {
		if _, ok := header["typ"]; !ok {
			return false
		}
	}
	payload := map[string]any{}
	return decodeBase64URLJSON(parts[1], &payload)
}

func decodeBase64URLJSON(segment string, out any) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(segment)
		if err != nil {
			return false
		}
	}
	if err := json.Unmarshal(decoded, out); err != nil {
		return false
	}
	if object, ok := out.(*map[string]any); ok {
		return len(*object) > 0
	}
	return true
}

func isSensitiveSourceAssignment(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	return normalized == "token_from" || tokenSourceAssignmentSuffix.MatchString(normalized)
}

func isAssignedSecretLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || isWorkflowReferenceOrBindingName(value) {
		return false
	}
	for _, pattern := range providerSecretPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	if isValidatedJWT(value) {
		return true
	}
	if len(value) < minAssignedSecretLength {
		return false
	}
	return looksTokenShaped(value)
}

func isWorkflowReferenceOrBindingName(value string) bool {
	value = strings.TrimSpace(value)
	if !strings.ContainsAny(value, "._[]-") {
		return false
	}
	return workflowReferencePattern.MatchString(value)
}

func looksTokenShaped(value string) bool {
	if len(value) < 16 || !tokenShapedValuePattern.MatchString(value) {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}
