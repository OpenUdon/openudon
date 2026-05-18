package workflowintent

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/catalog"
)

// CatalogAdviceOptions controls optional catalog-derived intent advice.
// Catalog metadata is advisory only; explicit OpenAPI inputs keep precedence.
type CatalogAdviceOptions struct {
	Disabled                bool
	Catalog                 catalog.Catalog
	SecurityClassifications []catalog.SecurityClassification
	ExplicitOpenAPIInputs   []string
}

// CatalogAdviceReport is a metadata-only provider/spec/security view for an
// intent. It does not imply credentials, account choice, or execution.
type CatalogAdviceReport struct {
	Enabled               bool                    `json:"enabled"`
	ExplicitOpenAPIInputs []string                `json:"explicit_openapi_inputs,omitempty"`
	Providers             []CatalogProviderAdvice `json:"providers,omitempty"`
}

// CatalogProviderAdvice records catalog context matched from intent hints.
type CatalogProviderAdvice struct {
	ProviderID              string                         `json:"provider_id"`
	DisplayName             string                         `json:"display_name,omitempty"`
	MatchedHint             string                         `json:"matched_hint,omitempty"`
	MatchSource             string                         `json:"match_source,omitempty"`
	ExplicitOpenAPIOverride bool                           `json:"explicit_openapi_override,omitempty"`
	ExplicitOpenAPIInputs   []string                       `json:"explicit_openapi_inputs,omitempty"`
	UserOpenAPINeed         catalog.UserOpenAPINeed        `json:"user_openapi_need,omitempty"`
	AuthStatus              catalog.AuthCompletenessStatus `json:"auth_status,omitempty"`
	SpecRef                 *CatalogSpecAdvice             `json:"spec_ref,omitempty"`
	OverlayIDs              []string                       `json:"overlay_ids,omitempty"`
	SourceNotes             []CatalogAdviceSourceNote      `json:"source_notes,omitempty"`
}

// CatalogSpecAdvice is the preferred catalog spec reference for a provider.
type CatalogSpecAdvice struct {
	ID              string                  `json:"id"`
	Kind            catalog.SpecKind        `json:"kind"`
	URL             string                  `json:"url"`
	SourceAuthority catalog.SourceAuthority `json:"source_authority"`
	SourceNote      string                  `json:"source_note,omitempty"`
}

// CatalogAdviceSourceNote preserves provenance-backed source notes.
type CatalogAdviceSourceNote struct {
	Provenance catalog.SecurityProvenance `json:"provenance"`
	OverlayID  string                     `json:"overlay_id,omitempty"`
	SpecRefID  string                     `json:"spec_ref_id,omitempty"`
	SourceNote string                     `json:"source_note"`
}

type catalogAdviceHint struct {
	Value    string
	Source   string
	Explicit bool
}

// CatalogAdviceForIntent returns optional catalog context for providers hinted
// by intent fields. Unknown providers are ignored so private or local APIs keep
// working without a catalog entry.
func CatalogAdviceForIntent(intent *Intent, options CatalogAdviceOptions) (CatalogAdviceReport, error) {
	report := CatalogAdviceReport{Enabled: !options.Disabled}
	if options.Disabled {
		return report, nil
	}
	if intent == nil {
		report.ExplicitOpenAPIInputs = uniqueStrings(options.ExplicitOpenAPIInputs)
		return report, nil
	}

	cat := options.Catalog
	classifications := options.SecurityClassifications
	if len(cat.Providers) == 0 && len(cat.SecurityOverlays) == 0 {
		cat = catalog.BuiltInCatalog()
		if classifications == nil {
			classifications = catalog.BuiltInSecurityClassifications()
		}
	}
	if err := cat.Validate(); err != nil {
		return CatalogAdviceReport{}, err
	}

	hints := catalogAdviceHints(intent)
	explicitInputs := append([]string(nil), options.ExplicitOpenAPIInputs...)
	for _, hint := range hints {
		if hint.Explicit {
			explicitInputs = append(explicitInputs, hint.Value)
		}
	}
	report.ExplicitOpenAPIInputs = uniqueStrings(explicitInputs)

	matches := map[string]CatalogProviderAdvice{}
	for _, hint := range hints {
		provider, ok := findCatalogProviderForHint(cat, hint.Value)
		if !ok {
			continue
		}
		if _, exists := matches[provider.ID]; exists {
			continue
		}
		advice, err := catalogProviderAdvice(cat, classifications, provider, hint, report.ExplicitOpenAPIInputs)
		if err != nil {
			return CatalogAdviceReport{}, err
		}
		matches[provider.ID] = advice
	}
	for _, advice := range matches {
		report.Providers = append(report.Providers, advice)
	}
	sort.SliceStable(report.Providers, func(i, j int) bool {
		return report.Providers[i].ProviderID < report.Providers[j].ProviderID
	})
	return report, nil
}

// RenderCatalogAdviceMarkdown renders advisory catalog context for review
// evidence. Empty reports intentionally render nothing.
func RenderCatalogAdviceMarkdown(report CatalogAdviceReport) string {
	if !report.Enabled || len(report.Providers) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Catalog Advisory\n\n")
	b.WriteString("- Catalog metadata is advisory. Explicit OpenAPI inputs and human review remain authoritative for generated intent.\n")
	for _, provider := range report.Providers {
		name := firstNonEmpty(provider.DisplayName, provider.ProviderID)
		fmt.Fprintf(&b, "- Provider: `%s` (`%s`)\n", name, provider.ProviderID)
		if provider.MatchedHint != "" {
			fmt.Fprintf(&b, "  - Matched hint: `%s` from `%s`\n", provider.MatchedHint, provider.MatchSource)
		}
		if provider.ExplicitOpenAPIOverride {
			fmt.Fprintf(&b, "  - Explicit OpenAPI input overrides built-in catalog spec: `%s`\n", strings.Join(provider.ExplicitOpenAPIInputs, "`, `"))
		}
		if provider.SpecRef != nil {
			fmt.Fprintf(&b, "  - Catalog spec: `%s` `%s` `%s`\n", provider.SpecRef.ID, provider.SpecRef.Kind, provider.SpecRef.URL)
		}
		if provider.UserOpenAPINeed != "" {
			fmt.Fprintf(&b, "  - User OpenAPI need: `%s`\n", provider.UserOpenAPINeed)
		}
		if provider.AuthStatus != "" {
			fmt.Fprintf(&b, "  - Auth/security status: `%s`\n", provider.AuthStatus)
		}
		if len(provider.OverlayIDs) > 0 {
			fmt.Fprintf(&b, "  - Security overlays: `%s`\n", strings.Join(provider.OverlayIDs, "`, `"))
		}
		for _, note := range provider.SourceNotes {
			fmt.Fprintf(&b, "  - Source note: %s\n", note.SourceNote)
		}
	}
	return b.String()
}

func catalogProviderAdvice(cat catalog.Catalog, classifications []catalog.SecurityClassification, provider catalog.Provider, hint catalogAdviceHint, explicitInputs []string) (CatalogProviderAdvice, error) {
	view, err := catalog.BuildSecurityInspectionView(catalog.SecurityInspectionOptions{
		Catalog:                 cat,
		SecurityClassifications: classifications,
		ProviderKey:             provider.ID,
	})
	if err != nil {
		return CatalogProviderAdvice{}, err
	}
	advice := CatalogProviderAdvice{
		ProviderID:              provider.ID,
		DisplayName:             provider.DisplayName,
		MatchedHint:             strings.TrimSpace(hint.Value),
		MatchSource:             hint.Source,
		ExplicitOpenAPIOverride: len(explicitInputs) > 0,
		ExplicitOpenAPIInputs:   append([]string(nil), explicitInputs...),
		UserOpenAPINeed:         provider.UserOpenAPINeed,
		AuthStatus:              view.Status,
		OverlayIDs:              securityInspectionOverlayIDs(view),
		SourceNotes:             securityInspectionSourceNotes(view),
	}
	if spec, ok := preferredCatalogSpec(provider); ok {
		advice.SpecRef = &CatalogSpecAdvice{
			ID:              spec.ID,
			Kind:            spec.Kind,
			URL:             spec.URL,
			SourceAuthority: spec.SourceAuthority,
			SourceNote:      spec.SourceNote,
		}
	}
	return advice, nil
}

func catalogAdviceHints(intent *Intent) []catalogAdviceHint {
	if intent == nil {
		return nil
	}
	var hints []catalogAdviceHint
	if strings.TrimSpace(intent.OpenAPI) != "" {
		hints = append(hints, catalogAdviceHint{Value: intent.OpenAPI, Source: "intent.openapi", Explicit: true})
	}
	walkSteps(intent.Steps, func(step *Step) {
		if strings.TrimSpace(step.Provider) != "" {
			hints = append(hints, catalogAdviceHint{Value: step.Provider, Source: stepHintSource(step, "provider"), Explicit: false})
		}
		if strings.TrimSpace(step.OpenAPI) != "" {
			hints = append(hints, catalogAdviceHint{Value: step.OpenAPI, Source: stepHintSource(step, "openapi"), Explicit: true})
		}
	})
	return hints
}

func stepHintSource(step *Step, field string) string {
	if step == nil || strings.TrimSpace(step.Name) == "" {
		return "step." + field
	}
	return "step." + strings.TrimSpace(step.Name) + "." + field
}

func findCatalogProviderForHint(cat catalog.Catalog, hint string) (catalog.Provider, bool) {
	for _, candidate := range catalogProviderHintCandidates(hint) {
		if provider, ok := cat.FindProvider(candidate); ok {
			return provider, true
		}
	}
	return catalog.Provider{}, false
}

var catalogHintSplitter = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func catalogProviderHintCandidates(hint string) []string {
	trimmed := strings.TrimSpace(hint)
	if trimmed == "" {
		return nil
	}
	values := []string{trimmed}
	if parsed, err := url.Parse(trimmed); err == nil && parsed.Path != "" {
		values = append(values, path.Base(parsed.Path))
	}
	values = append(values, filepath.Base(trimmed))

	var out []string
	for _, value := range values {
		cleaned := strings.TrimSpace(value)
		if cleaned == "" || cleaned == "." || cleaned == "/" {
			continue
		}
		out = append(out, cleaned)
		stem := trimKnownSpecExtensions(cleaned)
		out = append(out, stem)
		for _, token := range catalogHintSplitter.Split(stem, -1) {
			token = strings.TrimSpace(token)
			if len(token) >= 3 {
				out = append(out, token)
			}
		}
	}
	return uniqueStrings(out)
}

func trimKnownSpecExtensions(value string) string {
	out := value
	for {
		ext := strings.ToLower(filepath.Ext(out))
		switch ext {
		case ".json", ".yaml", ".yml":
			out = strings.TrimSuffix(out, filepath.Ext(out))
		default:
			return out
		}
	}
}

func preferredCatalogSpec(provider catalog.Provider) (catalog.SpecReference, bool) {
	for _, kind := range []catalog.SpecKind{
		catalog.SpecKindOpenAPI,
		catalog.SpecKindOpenAPIIndex,
		catalog.SpecKindGoogleDiscovery,
		catalog.SpecKindDropboxStone,
		catalog.SpecKindHumanDocs,
	} {
		refs := provider.SpecReferencesByKind(kind)
		if len(refs) > 0 {
			return refs[0], true
		}
	}
	return catalog.SpecReference{}, false
}

func securityInspectionOverlayIDs(view catalog.SecurityInspectionView) []string {
	seen := map[string]bool{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			seen[value] = true
		}
	}
	for _, scheme := range view.SecuritySchemes {
		add(scheme.OverlayID)
	}
	for _, requirement := range view.RootSecurity {
		add(requirement.OverlayID)
	}
	for _, operation := range view.OperationSecurity {
		add(operation.OverlayID)
		for _, requirement := range operation.Security {
			add(requirement.OverlayID)
		}
	}
	for _, conflict := range view.Conflicts {
		add(conflict.OverlayID)
	}
	for _, note := range view.SourceNotes {
		add(note.OverlayID)
	}
	return sortedStringSet(seen)
}

func securityInspectionSourceNotes(view catalog.SecurityInspectionView) []CatalogAdviceSourceNote {
	seen := map[string]bool{}
	var out []CatalogAdviceSourceNote
	for _, note := range view.SourceNotes {
		if strings.TrimSpace(note.SourceNote) == "" {
			continue
		}
		key := string(note.Provenance) + "\x00" + note.OverlayID + "\x00" + note.SpecRefID + "\x00" + note.SourceNote
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, CatalogAdviceSourceNote{
			Provenance: note.Provenance,
			OverlayID:  note.OverlayID,
			SpecRefID:  note.SpecRefID,
			SourceNote: note.SourceNote,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].OverlayID != out[j].OverlayID {
			return out[i].OverlayID < out[j].OverlayID
		}
		if out[i].SpecRefID != out[j].SpecRefID {
			return out[i].SpecRefID < out[j].SpecRefID
		}
		return out[i].SourceNote < out[j].SourceNote
	})
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}

func uniqueStrings(values []string) []string {
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

func sortedStringSet(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
