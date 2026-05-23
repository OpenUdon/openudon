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
	MaxBlocking         *int              `json:"max_blocking,omitempty"`
	MaxWarning          *int              `json:"max_warning,omitempty"`
	MaxAdvisory         *int              `json:"max_advisory,omitempty"`
	MaxUnresolvedReview *int              `json:"max_unresolved_review,omitempty"`
}

func (p ReferencePolicy) IsZero() bool {
	return strings.TrimSpace(p.Mode) == "" &&
		len(p.Notes) == 0 &&
		len(p.SeverityOverrides) == 0 &&
		len(p.IssueNotes) == 0 &&
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
