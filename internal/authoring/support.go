package authoring

import (
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
)

// OpenAPI-derived summaries remain owned by the narrowed apitools boundary.
type OperationInventory = apitools.OperationInventory
type AuthRequirementSummary = apitools.AuthRequirementSummary

// DocumentationContext is Ramen-owned advisory context attached to local
// authoring artifacts. OpenAPI documents remain the authoritative API contract.
type DocumentationContext struct {
	Snippets    []DocumentationSnippet `json:"snippets,omitempty"`
	Diagnostics []Diagnostic           `json:"diagnostics,omitempty"`
}

type DocumentationSnippet struct {
	Title     string `json:"title,omitempty"`
	Content   string `json:"content,omitempty"`
	SourceURL string `json:"source_url,omitempty"`
	Source    string `json:"source,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Language  string `json:"language,omitempty"`
	LibraryID string `json:"library_id,omitempty"`
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sortedUniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func mergeAuthRequirements(in []AuthRequirementSummary) []AuthRequirementSummary {
	seen := map[string]AuthRequirementSummary{}
	for _, requirement := range in {
		key := strings.Join([]string{
			strings.TrimSpace(requirement.Kind),
			strings.TrimSpace(requirement.Scheme),
			strings.TrimSpace(requirement.Type),
			strings.TrimSpace(requirement.In),
			strings.TrimSpace(requirement.ParameterName),
		}, "\x00")
		if strings.Trim(key, "\x00") == "" {
			continue
		}
		if existing, ok := seen[key]; ok {
			existing.Flows = sortedUniqueStrings(append(existing.Flows, requirement.Flows...))
			existing.Scopes = sortedUniqueStrings(append(existing.Scopes, requirement.Scopes...))
			existing.CredentialFields = sortedUniqueStrings(append(existing.CredentialFields, requirement.CredentialFields...))
			existing.OptionalCredentialFields = sortedUniqueStrings(append(existing.OptionalCredentialFields, requirement.OptionalCredentialFields...))
			existing.ConfigFields = sortedUniqueStrings(append(existing.ConfigFields, requirement.ConfigFields...))
			existing.OptionalConfigFields = sortedUniqueStrings(append(existing.OptionalConfigFields, requirement.OptionalConfigFields...))
			if existing.Dialect == "" {
				existing.Dialect = requirement.Dialect
			}
			if existing.AuthorizationURL == "" {
				existing.AuthorizationURL = requirement.AuthorizationURL
			}
			if existing.TokenURL == "" {
				existing.TokenURL = requirement.TokenURL
			}
			if existing.RefreshURL == "" {
				existing.RefreshURL = requirement.RefreshURL
			}
			if existing.Description == "" {
				existing.Description = requirement.Description
			}
			seen[key] = existing
			continue
		}
		requirement.Flows = sortedUniqueStrings(requirement.Flows)
		requirement.Scopes = sortedUniqueStrings(requirement.Scopes)
		requirement.CredentialFields = sortedUniqueStrings(requirement.CredentialFields)
		requirement.OptionalCredentialFields = sortedUniqueStrings(requirement.OptionalCredentialFields)
		requirement.ConfigFields = sortedUniqueStrings(requirement.ConfigFields)
		requirement.OptionalConfigFields = sortedUniqueStrings(requirement.OptionalConfigFields)
		seen[key] = requirement
	}
	out := make([]AuthRequirementSummary, 0, len(seen))
	for _, requirement := range seen {
		out = append(out, requirement)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.Join([]string{out[i].Kind, out[i].Scheme, out[i].Type, out[i].In, out[i].ParameterName}, "\x00") <
			strings.Join([]string{out[j].Kind, out[j].Scheme, out[j].Type, out[j].In, out[j].ParameterName}, "\x00")
	})
	return out
}
