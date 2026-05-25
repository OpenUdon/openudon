package eval

import (
	"encoding/json"
	"os"
	"strings"
)

type ReferencePolicy struct {
	Mode                string            `json:"mode,omitempty"`
	Notes               []string          `json:"notes,omitempty"`
	SeverityOverrides   map[string]string `json:"severity_overrides,omitempty"`
	IssueNotes          map[string]string `json:"issue_notes,omitempty"`
	SeedBuild           *SeedBuildPolicy  `json:"seed_build,omitempty"`
	MaxBlocking         *int              `json:"max_blocking,omitempty"`
	MaxWarning          *int              `json:"max_warning,omitempty"`
	MaxAdvisory         *int              `json:"max_advisory,omitempty"`
	MaxUnresolvedReview *int              `json:"max_unresolved_review,omitempty"`
}

type SeedBuildPolicy struct {
	Expected            string   `json:"expected,omitempty"`
	Class               string   `json:"class,omitempty"`
	Reason              string   `json:"reason,omitempty"`
	AllowedFailureCodes []string `json:"allowed_failure_codes,omitempty"`
}

func (p ReferencePolicy) IsZero() bool {
	return strings.TrimSpace(p.Mode) == "" &&
		len(p.Notes) == 0 &&
		len(p.SeverityOverrides) == 0 &&
		len(p.IssueNotes) == 0 &&
		p.SeedBuild == nil &&
		p.MaxBlocking == nil &&
		p.MaxWarning == nil &&
		p.MaxAdvisory == nil &&
		p.MaxUnresolvedReview == nil
}

func ReadReferencePolicy(path string) (ReferencePolicy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ReferencePolicy{}, err
	}
	var policy ReferencePolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return ReferencePolicy{}, err
	}
	policy.Mode = normalizeReferencePolicyMode(policy.Mode)
	policy.SeverityOverrides = normalizeSeverityMap(policy.SeverityOverrides)
	policy.SeedBuild = normalizeSeedBuildPolicy(policy.SeedBuild)
	return policy, nil
}

func normalizeReferencePolicyMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "strict":
		return "strict"
	case "advisory":
		return "advisory"
	default:
		return ""
	}
}

func normalizeSeverityMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = normalizeSeverityValue(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeSeedBuildPolicy(policy *SeedBuildPolicy) *SeedBuildPolicy {
	if policy == nil {
		return nil
	}
	out := *policy
	out.Expected = normalizeSeedBuildExpected(out.Expected)
	out.Class = normalizeSeedBuildClass(out.Class)
	out.AllowedFailureCodes = normalizeStringList(out.AllowedFailureCodes)
	out.Reason = strings.TrimSpace(out.Reason)
	if out.Expected == "" && out.Class == "" && out.Reason == "" && len(out.AllowedFailureCodes) == 0 {
		return nil
	}
	return &out
}

func normalizeSeedBuildExpected(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pass":
		return "pass"
	case "build_fail", "build-fail", "fail":
		return "build_fail"
	case "icot_fail", "icot-fail":
		return "icot_fail"
	default:
		return ""
	}
}

func normalizeSeedBuildClass(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict-positive", "strict_positive", "strict":
		return "strict-positive"
	case "expected-negative", "expected_negative", "negative":
		return "expected-negative"
	case "advisory":
		return "advisory"
	default:
		return ""
	}
}

func normalizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
