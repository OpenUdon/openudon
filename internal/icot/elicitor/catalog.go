package elicitor

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
	"unicode"

	"github.com/OpenUdon/apitools/catalog"
	"github.com/OpenUdon/apitools/sqlitecache"
)

const defaultSiblingAPIToolsCache = "../apitools/catalog-openapi-cache"
const defaultSiblingAPIToolsCacheDB = "../apitools/catalog-openapi-cache/cache.sqlite"

type CatalogHintOptions struct {
	CacheRoot string
}

type CatalogHint struct {
	Provider         catalog.Provider
	AuthStatus       catalog.AuthCompletenessStatus
	SpecArtifacts    []CatalogSpecArtifactHint
	OverlayArtifacts []string
	FollowUps        []string
	MatchIndex       int
}

type CatalogSpecArtifactHint struct {
	SpecRef catalog.SpecReference
	Path    string
}

type CatalogMigrationCandidate struct {
	ProviderID    string
	ProviderName  string
	SpecRefID     string
	SourceURL     string
	Kind          catalog.SpecKind
	Protocol      string
	SourcePath    string
	TargetPath    string
	RelativePath  string
	ExistingLocal bool
}

type CatalogMigrationResult struct {
	Copied   []CatalogMigrationCandidate
	Existing []CatalogMigrationCandidate
	Missing  []CatalogHint
	Notes    []string
}

func BuildCatalogHints(query string, opts CatalogHintOptions) ([]CatalogHint, error) {
	providers := matchingCatalogProviders(query)
	if len(providers) == 0 {
		return nil, nil
	}
	artifactRows, _ := catalogSpecArtifactsFromSiblingCache(defaultSiblingAPIToolsCacheDB)
	securityReport, err := catalog.BuiltInSecurityReport()
	if err != nil {
		return nil, err
	}
	authByProvider := map[string]catalog.AuthCompletenessStatus{}
	for _, row := range catalog.SecurityReportRows(securityReport) {
		authByProvider[row.ProviderID] = row.Status
	}
	cacheRoot := strings.TrimSpace(opts.CacheRoot)
	if cacheRoot == "" {
		cacheRoot = defaultSiblingAPIToolsCache
	}
	resolutionByProvider := map[string]catalog.ProviderResolution{}
	for _, resolution := range resolveCatalogProvidersWithArtifacts(providers, artifactRows) {
		resolutionByProvider[resolution.ProviderID] = resolution
	}
	var hints []CatalogHint
	for _, provider := range providers {
		hint := CatalogHint{
			Provider:   provider,
			AuthStatus: authByProvider[provider.ID],
			MatchIndex: catalogProviderMatchIndex(query, provider),
		}
		for _, ref := range provider.SpecReferences {
			path := resolutionSpecArtifactPath(cacheRoot, resolutionByProvider[provider.ID], ref)
			if path == "" {
				path = siblingCatalogSpecArtifactPath(cacheRoot, ref)
			}
			hint.SpecArtifacts = append(hint.SpecArtifacts, CatalogSpecArtifactHint{
				SpecRef: ref,
				Path:    path,
			})
		}
		hint.OverlayArtifacts = resolutionOverlayArtifactPaths(cacheRoot, resolutionByProvider[provider.ID])
		if len(hint.OverlayArtifacts) == 0 {
			hint.OverlayArtifacts = siblingCatalogOverlayArtifacts(cacheRoot, provider.ID)
		}
		hint.FollowUps = catalogHintFollowUps(provider, hint)
		hints = append(hints, hint)
	}
	return hints, nil
}

func MigrateCatalogArtifactsForSession(session Session, exampleDir string) (CatalogMigrationResult, error) {
	return MigrateCatalogArtifacts(catalogQueryForSession(session), exampleDir, CatalogHintOptions{})
}

func MigrateCatalogArtifacts(query, exampleDir string, opts CatalogHintOptions) (CatalogMigrationResult, error) {
	hints, err := BuildCatalogHints(query, opts)
	if err != nil {
		return CatalogMigrationResult{}, err
	}
	cacheRoot := strings.TrimSpace(opts.CacheRoot)
	if cacheRoot == "" {
		cacheRoot = defaultSiblingAPIToolsCache
	}
	artifactRows, _ := catalogSpecArtifactsFromSiblingCache(filepath.Join(cacheRoot, "cache.sqlite"))
	export, exportErr := catalog.ExportWorkflowArtifacts(context.Background(), catalog.ExportWorkflowArtifactsOptions{
		ProviderKeys:            catalogProviderIDs(hints),
		WorkflowDir:             exampleDir,
		ArtifactDir:             "api-artifacts",
		CacheDir:                cacheRoot,
		Artifacts:               artifactRows,
		IncludeSecurityOverlays: true,
		WriteManifest:           true,
	})
	if exportErr == nil {
		result, err := migrationResultFromExport(export, hints, exampleDir)
		if err != nil {
			return result, err
		}
		if len(result.Copied)+len(result.Existing) > 0 {
			return result, nil
		}
	}
	candidates := CatalogMigrationCandidates(hints, exampleDir)
	var result CatalogMigrationResult
	availableByProvider := map[string]bool{}
	var sourceCandidates []CatalogMigrationCandidate
	for _, candidate := range candidates {
		availableByProvider[candidate.ProviderID] = true
		if catalogMigrationCandidateIsAPISource(candidate) {
			sourceCandidates = append(sourceCandidates, candidate)
		}
		if candidate.ExistingLocal {
			result.Existing = append(result.Existing, candidate)
			continue
		}
		if err := copyCatalogArtifact(candidate.SourcePath, candidate.TargetPath); err != nil {
			return result, err
		}
		result.Copied = append(result.Copied, candidate)
	}
	securityCandidates, notes, err := materializeBuiltInSecurityOverlaySidecars(sourceCandidates, exampleDir)
	if err != nil {
		return result, err
	}
	result.Notes = append(result.Notes, notes...)
	appendSecurityCandidates(&result, securityCandidates)
	for _, hint := range hints {
		if !availableByProvider[hint.Provider.ID] {
			result.Missing = append(result.Missing, hint)
		}
	}
	return result, nil
}

func CatalogMigrationCandidates(hints []CatalogHint, exampleDir string) []CatalogMigrationCandidate {
	var out []CatalogMigrationCandidate
	for _, hint := range hints {
		for _, artifact := range hint.SpecArtifacts {
			sourcePath := strings.TrimSpace(artifact.Path)
			if sourcePath == "" || !migratableSpecKind(artifact.SpecRef.Kind) {
				continue
			}
			rel := catalogArtifactTargetRelativePath(artifact.SpecRef, sourcePath)
			if rel == "" {
				continue
			}
			target := filepath.Join(exampleDir, filepath.FromSlash(rel))
			_, statErr := os.Stat(target)
			out = append(out, CatalogMigrationCandidate{
				ProviderID:    hint.Provider.ID,
				ProviderName:  firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID),
				SpecRefID:     artifact.SpecRef.ID,
				SourceURL:     artifact.SpecRef.URL,
				Kind:          artifact.SpecRef.Kind,
				Protocol:      string(artifact.SpecRef.ProtocolClassification().Protocol),
				SourcePath:    sourcePath,
				TargetPath:    target,
				RelativePath:  rel,
				ExistingLocal: statErr == nil,
			})
		}
		for _, sourcePath := range hint.OverlayArtifacts {
			sourcePath = strings.TrimSpace(sourcePath)
			if sourcePath == "" {
				continue
			}
			rel := filepath.ToSlash(filepath.Join("openapi", filepath.Base(sourcePath)))
			target := filepath.Join(exampleDir, filepath.FromSlash(rel))
			_, statErr := os.Stat(target)
			out = append(out, CatalogMigrationCandidate{
				ProviderID:    hint.Provider.ID,
				ProviderName:  firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID),
				Kind:          catalog.SpecKind("advisory-overlay"),
				Protocol:      string(catalog.SpecProtocolOpenAPI),
				SourcePath:    sourcePath,
				TargetPath:    target,
				RelativePath:  rel,
				ExistingLocal: statErr == nil,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ProviderID != out[j].ProviderID {
			return out[i].ProviderID < out[j].ProviderID
		}
		return out[i].RelativePath < out[j].RelativePath
	})
	return out
}

func migrationResultFromExport(export catalog.ExportReport, hints []CatalogHint, exampleDir string) (CatalogMigrationResult, error) {
	var result CatalogMigrationResult
	availableByProvider := map[string]bool{}
	for _, provider := range export.Providers {
		var providerSources []CatalogMigrationCandidate
		for _, artifact := range provider.Artifacts {
			rel := materializedArtifactTargetRelativePath(artifact)
			if rel == "" {
				continue
			}
			target := filepath.Join(exampleDir, filepath.FromSlash(rel))
			candidate := CatalogMigrationCandidate{
				ProviderID:   provider.ProviderID,
				ProviderName: firstNonEmpty(provider.DisplayName, provider.ProviderID),
				SpecRefID:    artifact.SpecRefID,
				SourceURL:    firstNonEmpty(artifact.SourceURL, providerResolutionSpecRefURL(provider.Resolution, artifact.SpecRefID)),
				Kind:         catalog.SpecKind(artifact.Kind),
				Protocol:     string(artifact.Protocol),
				SourcePath:   artifact.TargetPath,
				TargetPath:   target,
				RelativePath: rel,
			}
			providerSources = append(providerSources, candidate)
			availableByProvider[provider.ProviderID] = true
			if _, err := os.Stat(target); err == nil {
				candidate.ExistingLocal = true
				result.Existing = append(result.Existing, candidate)
				continue
			}
			if err := copyCatalogArtifact(artifact.TargetPath, target); err != nil {
				return result, err
			}
			result.Copied = append(result.Copied, candidate)
		}
		securityCandidates, notes, err := materializeExportedSecurityOverlaySidecars(provider, providerSources, exampleDir)
		if err != nil {
			return result, err
		}
		result.Notes = append(result.Notes, notes...)
		appendSecurityCandidates(&result, securityCandidates)
	}
	for _, hint := range hints {
		if !availableByProvider[hint.Provider.ID] {
			result.Missing = append(result.Missing, hint)
		}
	}
	return result, nil
}

func materializedArtifactTargetRelativePath(artifact catalog.MaterializedArtifact) string {
	if dir := uwsSourceTypeArtifactDir(artifact.UWSSourceType); dir != "" {
		return filepath.ToSlash(filepath.Join(dir, filepath.Base(artifact.TargetPath)))
	}
	switch artifact.Protocol {
	case catalog.SpecProtocolOpenAPI, catalog.SpecProtocolSwagger:
		return filepath.ToSlash(filepath.Join("openapi", filepath.Base(artifact.TargetPath)))
	case catalog.SpecProtocolGoogleDiscovery:
		return filepath.ToSlash(filepath.Join("google-discovery", filepath.Base(artifact.TargetPath)))
	case catalog.SpecProtocolSmithy:
		return filepath.ToSlash(filepath.Join("aws-smithy", filepath.Base(artifact.TargetPath)))
	default:
		switch strings.TrimSpace(artifact.Kind) {
		case "openapi", "openapi-index", "advisory-overlay":
			return filepath.ToSlash(filepath.Join("openapi", filepath.Base(artifact.TargetPath)))
		case "google-discovery":
			return filepath.ToSlash(filepath.Join("google-discovery", filepath.Base(artifact.TargetPath)))
		case "smithy-json":
			return filepath.ToSlash(filepath.Join("aws-smithy", filepath.Base(artifact.TargetPath)))
		default:
			return ""
		}
	}
}

func providerResolutionSpecRefURL(resolution catalog.ProviderResolution, specRefID string) string {
	specRefID = strings.TrimSpace(specRefID)
	if specRefID == "" {
		return ""
	}
	for _, ref := range resolution.SpecReferences {
		if ref.ID == specRefID {
			return strings.TrimSpace(ref.URL)
		}
	}
	return ""
}

type catalogSecurityOverlayCandidate struct {
	ProviderID    string
	ProviderName  string
	OverlayID     string
	SpecRefID     string
	SourceRefs    []string
	SourcePath    string
	Overlay       *catalog.SecurityOverlay
	TargetSource  CatalogMigrationCandidate
	TargetPath    string
	RelativePath  string
	ExistingLocal bool
}

func materializeExportedSecurityOverlaySidecars(provider catalog.MaterializationReport, sources []CatalogMigrationCandidate, exampleDir string) ([]catalogSecurityOverlayCandidate, []string, error) {
	var overlays []catalogSecurityOverlayCandidate
	for _, materialized := range provider.SecurityOverlays {
		overlay := catalogSecurityOverlayCandidate{
			ProviderID:   provider.ProviderID,
			ProviderName: firstNonEmpty(provider.DisplayName, provider.ProviderID),
			OverlayID:    materialized.OverlayID,
			SpecRefID:    materialized.SpecRefID,
			SourcePath:   materialized.TargetPath,
		}
		if data, err := os.ReadFile(materialized.TargetPath); err == nil {
			var parsed catalog.SecurityOverlay
			if err := json.Unmarshal(data, &parsed); err == nil {
				overlay.SourceRefs = append([]string(nil), parsed.SourceRefs...)
			}
		}
		overlays = append(overlays, overlay)
	}
	return materializeSecurityOverlaySidecars(overlays, sources, exampleDir)
}

func materializeBuiltInSecurityOverlaySidecars(sources []CatalogMigrationCandidate, exampleDir string) ([]catalogSecurityOverlayCandidate, []string, error) {
	var overlays []catalogSecurityOverlayCandidate
	seenProvider := map[string]bool{}
	for _, source := range sources {
		if source.ProviderID == "" || seenProvider[source.ProviderID] {
			continue
		}
		seenProvider[source.ProviderID] = true
		for _, overlay := range catalog.SecurityOverlaysForProvider(source.ProviderID) {
			overlayCopy := overlay
			overlays = append(overlays, catalogSecurityOverlayCandidate{
				ProviderID:   source.ProviderID,
				ProviderName: source.ProviderName,
				OverlayID:    overlay.ID,
				SpecRefID:    overlay.SpecRefID,
				SourceRefs:   append([]string(nil), overlay.SourceRefs...),
				Overlay:      &overlayCopy,
			})
		}
	}
	return materializeSecurityOverlaySidecars(overlays, sources, exampleDir)
}

func materializeSecurityOverlaySidecars(overlays []catalogSecurityOverlayCandidate, sources []CatalogMigrationCandidate, exampleDir string) ([]catalogSecurityOverlayCandidate, []string, error) {
	var out []catalogSecurityOverlayCandidate
	var notes []string
	for _, overlay := range overlays {
		source, ok, note := matchSecurityOverlaySource(overlay, sources)
		if !ok {
			if strings.TrimSpace(note) != "" {
				notes = append(notes, note)
			}
			continue
		}
		overlay.TargetSource = source
		overlay.RelativePath = catalogSecurityOverlayTargetRelativePath(source.RelativePath)
		if overlay.RelativePath == "" {
			notes = append(notes, fmt.Sprintf("skipped security overlay %s for %s: source target has no sidecar path", firstNonEmpty(overlay.OverlayID, "<unknown>"), firstNonEmpty(source.RelativePath, source.SpecRefID)))
			continue
		}
		overlay.TargetPath = filepath.Join(exampleDir, filepath.FromSlash(overlay.RelativePath))
		if _, err := os.Stat(overlay.TargetPath); err == nil {
			overlay.ExistingLocal = true
			out = append(out, overlay)
			continue
		}
		if err := writeSecurityOverlaySidecar(overlay); err != nil {
			return out, notes, err
		}
		out = append(out, overlay)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ProviderID != out[j].ProviderID {
			return out[i].ProviderID < out[j].ProviderID
		}
		return out[i].RelativePath < out[j].RelativePath
	})
	sort.Strings(notes)
	return out, notes, nil
}

func matchSecurityOverlaySource(overlay catalogSecurityOverlayCandidate, sources []CatalogMigrationCandidate) (CatalogMigrationCandidate, bool, string) {
	providerSources := migratedSourceCandidatesForProvider(sources, overlay.ProviderID)
	overlayID := firstNonEmpty(overlay.OverlayID, "<unknown>")
	if specRefID := strings.TrimSpace(overlay.SpecRefID); specRefID != "" {
		var matches []CatalogMigrationCandidate
		for _, source := range providerSources {
			if strings.TrimSpace(source.SpecRefID) == specRefID {
				matches = append(matches, source)
			}
		}
		switch len(matches) {
		case 1:
			return matches[0], true, ""
		case 0:
		default:
			return CatalogMigrationCandidate{}, false, fmt.Sprintf("skipped security overlay %s: spec ref %s matched multiple migrated sources", overlayID, specRefID)
		}
	}
	sourceRefSet := map[string]bool{}
	for _, ref := range overlay.SourceRefs {
		if ref = strings.TrimSpace(ref); ref != "" {
			sourceRefSet[ref] = true
		}
	}
	if len(sourceRefSet) > 0 {
		var matches []CatalogMigrationCandidate
		for _, source := range providerSources {
			if sourceURL := strings.TrimSpace(source.SourceURL); sourceURL != "" && sourceRefSet[sourceURL] {
				matches = append(matches, source)
			}
		}
		switch len(matches) {
		case 1:
			return matches[0], true, ""
		case 0:
		default:
			return CatalogMigrationCandidate{}, false, fmt.Sprintf("skipped security overlay %s: source refs matched multiple migrated sources", overlayID)
		}
	}
	if len(providerSources) > 1 {
		return CatalogMigrationCandidate{}, false, fmt.Sprintf("skipped security overlay %s: no exact source match among multiple migrated %s sources", overlayID, overlay.ProviderID)
	}
	return CatalogMigrationCandidate{}, false, fmt.Sprintf("skipped security overlay %s: no exact source match", overlayID)
}

func migratedSourceCandidatesForProvider(sources []CatalogMigrationCandidate, providerID string) []CatalogMigrationCandidate {
	var out []CatalogMigrationCandidate
	for _, source := range sources {
		if source.ProviderID != providerID || !catalogMigrationCandidateIsAPISource(source) {
			continue
		}
		if strings.TrimSpace(source.RelativePath) == "" {
			continue
		}
		out = append(out, source)
	}
	return out
}

func catalogMigrationCandidateIsAPISource(candidate CatalogMigrationCandidate) bool {
	switch strings.TrimSpace(strings.Split(filepath.ToSlash(candidate.RelativePath), "/")[0]) {
	case "openapi", "google-discovery", "aws-smithy":
		return candidate.Kind != catalog.SpecKind("security-overlay")
	default:
		return false
	}
}

func catalogSecurityOverlayTargetRelativePath(sourceRel string) string {
	sourceRel = filepath.ToSlash(strings.TrimSpace(sourceRel))
	if sourceRel == "" {
		return ""
	}
	ext := pathExtSlash(sourceRel)
	base := strings.TrimSuffix(sourceRel, ext)
	if base == "" || base == sourceRel && ext == "" {
		base = sourceRel
	}
	return base + ".security-overlay.json"
}

func pathExtSlash(path string) string {
	return filepath.Ext(filepath.FromSlash(path))
}

func writeSecurityOverlaySidecar(candidate catalogSecurityOverlayCandidate) error {
	if err := os.MkdirAll(filepath.Dir(candidate.TargetPath), 0o755); err != nil {
		return err
	}
	if strings.TrimSpace(candidate.SourcePath) != "" {
		return copyCatalogArtifact(candidate.SourcePath, candidate.TargetPath)
	}
	if candidate.Overlay == nil {
		return fmt.Errorf("security overlay %s has no source metadata", firstNonEmpty(candidate.OverlayID, "<unknown>"))
	}
	data, err := json.MarshalIndent(candidate.Overlay, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(candidate.TargetPath, data, 0o644)
}

func appendSecurityCandidates(result *CatalogMigrationResult, candidates []catalogSecurityOverlayCandidate) {
	for _, candidate := range candidates {
		migrated := CatalogMigrationCandidate{
			ProviderID:    candidate.ProviderID,
			ProviderName:  candidate.ProviderName,
			SpecRefID:     candidate.SpecRefID,
			Kind:          catalog.SpecKind("security-overlay"),
			Protocol:      candidate.TargetSource.Protocol,
			SourcePath:    firstNonEmpty(candidate.SourcePath, candidate.TargetSource.SourcePath),
			TargetPath:    candidate.TargetPath,
			RelativePath:  candidate.RelativePath,
			ExistingLocal: candidate.ExistingLocal,
		}
		if candidate.ExistingLocal {
			result.Existing = append(result.Existing, migrated)
			continue
		}
		result.Copied = append(result.Copied, migrated)
	}
}

func CatalogProvidersMissingMigratableDocs(hints []CatalogHint, exampleDir string) []string {
	candidates := CatalogMigrationCandidates(hints, exampleDir)
	available := map[string]bool{}
	for _, candidate := range candidates {
		available[candidate.ProviderID] = true
	}
	var missing []string
	for _, hint := range hints {
		if !available[hint.Provider.ID] {
			missing = append(missing, firstNonEmpty(hint.Provider.DisplayName, hint.Provider.ID))
		}
	}
	return missing
}

func CatalogProvidersWithMigratableDocs(hints []CatalogHint, exampleDir string) []string {
	seen := map[string]bool{}
	var out []string
	for _, candidate := range CatalogMigrationCandidates(hints, exampleDir) {
		if seen[candidate.ProviderID] {
			continue
		}
		seen[candidate.ProviderID] = true
		out = append(out, candidate.ProviderName)
	}
	return out
}

func catalogQueryForSession(session Session) string {
	return strings.Join([]string{
		session.Project.ProjectName,
		session.Project.Goal,
		session.Project.OpenAPI,
		session.Project.DataFlow,
		session.Project.FunctionContracts,
		session.IntentName(),
		session.IntentDescription(),
	}, "\n")
}

func catalogProviderIDs(hints []CatalogHint) []string {
	var out []string
	for _, hint := range hints {
		if hint.Provider.ID != "" {
			out = append(out, hint.Provider.ID)
		}
	}
	return out
}

func CatalogHintsForSession(session Session) []CatalogHint {
	hints, _ := BuildCatalogHints(catalogQueryForSession(session), CatalogHintOptions{})
	return hints
}

func CatalogProviderPlan(hints []CatalogHint) []string {
	var out []string
	for _, hint := range hints {
		if hint.Provider.DisplayName != "" {
			out = append(out, hint.Provider.DisplayName)
		}
	}
	return out
}

func PrintCatalogHints(out io.Writer, query string) {
	hints, err := BuildCatalogHints(query, CatalogHintOptions{})
	if err != nil {
		fmt.Fprintf(out, "icot: apitools catalog advisory skipped: %v\n", err)
		return
	}
	printCatalogHints(out, hints)
}

func printCatalogHints(out io.Writer, hints []CatalogHint) {
	if len(hints) == 0 {
		return
	}
	fmt.Fprintln(out, "icot: apitools catalog matches first-class provider metadata:")
	for _, hint := range hints {
		fmt.Fprintf(out, "  - %s (%s): OpenAPI=%s, machine=%s, auth=%s\n", hint.Provider.DisplayName, hint.Provider.ID, hint.Provider.OfficialOpenAPIAvailability, hint.Provider.OfficialMachineSpecAvailability, hint.AuthStatus)
		for _, artifact := range hint.SpecArtifacts {
			ref := artifact.SpecRef
			if strings.TrimSpace(artifact.Path) != "" {
				fmt.Fprintf(out, "    artifact: %s %s (%s)\n", ref.Kind, artifact.Path, ref.ID)
				continue
			}
			if ref.Kind != catalog.SpecKindHumanDocs {
				fmt.Fprintf(out, "    catalog ref: %s %s (%s)\n", ref.Kind, ref.URL, ref.ID)
			}
		}
		for _, path := range hint.OverlayArtifacts {
			fmt.Fprintf(out, "    advisory overlay artifact: %s\n", path)
		}
		for _, followUp := range hint.FollowUps {
			fmt.Fprintf(out, "    follow-up: %s\n", followUp)
		}
	}
}

func matchingCatalogProviders(query string) []catalog.Provider {
	tokens := tokenSet(query)
	if len(tokens) == 0 {
		return nil
	}
	categoryCounts := map[string]int{}
	for _, provider := range catalog.BuiltInProviders() {
		if provider.Category != "" {
			categoryCounts[normalizeToken(provider.Category)]++
		}
	}
	var out []catalog.Provider
	for _, provider := range catalog.BuiltInProviders() {
		if catalogProviderMatches(provider, tokens, categoryCounts) {
			out = append(out, provider)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		iIndex := catalogProviderMatchIndex(query, out[i])
		jIndex := catalogProviderMatchIndex(query, out[j])
		if iIndex != jIndex {
			return iIndex < jIndex
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func catalogProviderMatches(provider catalog.Provider, tokens map[string]bool, categoryCounts map[string]int) bool {
	for _, phrase := range append([]string{provider.ID, provider.DisplayName}, provider.Aliases...) {
		if phraseTokensMatch(tokens, phrase) {
			return true
		}
	}
	category := normalizeToken(provider.Category)
	return category != "" && !genericCatalogCategoryToken(category) && categoryCounts[category] == 1 && tokens[category]
}

func genericCatalogCategoryToken(category string) bool {
	switch normalizeToken(category) {
	case "support":
		return true
	default:
		return false
	}
}

func phraseTokensMatch(tokens map[string]bool, phrase string) bool {
	parts := normalizedTokens(phrase)
	if len(parts) == 0 {
		return false
	}
	for _, part := range parts {
		if !tokens[part] {
			return false
		}
	}
	return true
}

func catalogProviderMatchIndex(query string, provider catalog.Provider) int {
	lower := strings.ToLower(query)
	best := len(lower) + 1
	for _, phrase := range append([]string{provider.ID, provider.DisplayName, provider.Category}, provider.Aliases...) {
		for _, token := range normalizedTokens(phrase) {
			if token == "" {
				continue
			}
			if idx := strings.Index(normalizeSearchText(lower), token); idx >= 0 && idx < best {
				best = idx
			}
		}
	}
	return best
}

func normalizeSearchText(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func tokenSet(value string) map[string]bool {
	out := map[string]bool{}
	for _, token := range normalizedTokens(value) {
		out[token] = true
	}
	return out
}

func normalizedTokens(value string) []string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	var out []string
	for _, field := range strings.Fields(b.String()) {
		if token := normalizeToken(field); token != "" {
			out = append(out, token)
		}
	}
	return out
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	return value
}

func siblingCatalogSpecArtifactPath(cacheRoot string, ref catalog.SpecReference) string {
	for _, candidate := range siblingCatalogSpecArtifactCandidates(cacheRoot, ref) {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.ToSlash(candidate)
		}
	}
	return ""
}

func siblingCatalogSpecArtifactCandidates(cacheRoot string, ref catalog.SpecReference) []string {
	var dirs []string
	switch ref.Kind {
	case catalog.SpecKindGoogleDiscovery:
		dirs = []string{"google-discovery"}
	case catalog.SpecKindSmithyJSON:
		dirs = []string{"aws-smithy"}
	default:
		dirs = []string{"openapi"}
	}
	extensions := []string{".json", ".yaml", ".yml", ".tar.gz"}
	var out []string
	for _, dir := range dirs {
		for _, ext := range extensions {
			out = append(out, filepath.Join(cacheRoot, dir, ref.ID+ext))
		}
	}
	return out
}

func siblingCatalogOverlayArtifacts(cacheRoot, providerID string) []string {
	pattern := filepath.Join(cacheRoot, "advisory-overlays", providerID+"*.json")
	matches, _ := filepath.Glob(pattern)
	for i := range matches {
		matches[i] = filepath.ToSlash(matches[i])
	}
	sort.Strings(matches)
	return matches
}

func resolveCatalogProvidersWithArtifacts(providers []catalog.Provider, artifacts []catalog.CatalogSpecArtifact) []catalog.ProviderResolution {
	if len(providers) == 0 {
		return nil
	}
	keys := make([]string, 0, len(providers))
	for _, provider := range providers {
		keys = append(keys, provider.ID)
	}
	resolutions, err := catalog.ResolveProvidersWithOptions(catalog.ProviderResolutionOptions{
		ProviderKeys: keys,
		Artifacts:    artifacts,
	})
	if err != nil {
		return nil
	}
	return resolutions
}

func resolutionSpecArtifactPath(cacheRoot string, resolution catalog.ProviderResolution, ref catalog.SpecReference) string {
	for _, artifact := range resolution.Artifacts {
		if artifact.SpecRefID != ref.ID || strings.TrimSpace(artifact.Path) == "" {
			continue
		}
		path := artifact.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(cacheRoot, filepath.FromSlash(path))
		}
		if _, err := os.Stat(path); err == nil {
			return filepath.ToSlash(path)
		}
	}
	return ""
}

func resolutionOverlayArtifactPaths(cacheRoot string, resolution catalog.ProviderResolution) []string {
	var out []string
	for _, artifact := range resolution.Artifacts {
		if strings.TrimSpace(artifact.OverlayPath) == "" && strings.TrimSpace(artifact.Kind) != "advisory-overlay" {
			continue
		}
		path := firstNonEmpty(artifact.OverlayPath, artifact.Path)
		if !filepath.IsAbs(path) {
			path = filepath.Join(cacheRoot, filepath.FromSlash(path))
		}
		if _, err := os.Stat(path); err == nil {
			out = append(out, filepath.ToSlash(path))
		}
	}
	sort.Strings(out)
	return out
}

func catalogSpecArtifactsFromSiblingCache(path string) ([]catalog.CatalogSpecArtifact, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	cache, err := sqlitecache.Open(path)
	if err != nil {
		return nil, err
	}
	defer cache.Close()
	artifacts, err := cache.ListCatalogArtifacts(context.Background())
	if err != nil {
		return nil, err
	}
	rows := make([]catalog.CatalogSpecArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		rows = append(rows, catalog.CatalogSpecArtifact{
			ProviderID:  artifact.ProviderID,
			SpecRefID:   artifact.ArtifactID,
			ArtifactID:  artifact.ArtifactID,
			Kind:        artifact.Kind,
			Path:        artifact.Path,
			SourceURL:   artifact.SourceURL,
			OverlayPath: artifact.OverlayPath,
			BuilderPath: artifact.BuilderPath,
			SHA256:      artifact.SHA256,
			Bytes:       artifact.Bytes,
			Metadata:    artifact.Metadata,
		})
	}
	return rows, nil
}

func migratableSpecKind(kind catalog.SpecKind) bool {
	switch kind {
	case catalog.SpecKindOpenAPI, catalog.SpecKindGoogleDiscovery, catalog.SpecKindSmithyJSON:
		return true
	default:
		return false
	}
}

func catalogArtifactTargetRelativePath(ref catalog.SpecReference, sourcePath string) string {
	base := filepath.Base(sourcePath)
	if base == "." || base == string(filepath.Separator) || strings.TrimSpace(base) == "" {
		return ""
	}
	dir := ref.ProtocolClassification().SourceAlignedArtifactDir()
	if dir == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Join(dir, base))
}

func uwsSourceTypeArtifactDir(sourceType string) string {
	switch strings.TrimSpace(sourceType) {
	case "openapi":
		return "openapi"
	case "google-discovery":
		return "google-discovery"
	case "aws-smithy":
		return "aws-smithy"
	default:
		return ""
	}
}

func copyCatalogArtifact(sourcePath, targetPath string) error {
	sourcePath = filepath.Clean(sourcePath)
	targetPath = filepath.Clean(targetPath)
	if sourcePath == "" || targetPath == "" {
		return errors.New("catalog migration requires source and target paths")
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0o644)
}

func catalogHintFollowUps(provider catalog.Provider, hint CatalogHint) []string {
	var followUps []string
	hasDirectOpenAPI := false
	hasNonOpenAPIMachineArtifact := false
	hasAdvisoryOverlay := len(hint.OverlayArtifacts) > 0
	for _, artifact := range hint.SpecArtifacts {
		if strings.TrimSpace(artifact.Path) == "" {
			continue
		}
		switch artifact.SpecRef.Kind {
		case catalog.SpecKindOpenAPI:
			hasDirectOpenAPI = true
		default:
			if artifact.SpecRef.Kind != catalog.SpecKindHumanDocs {
				hasNonOpenAPIMachineArtifact = true
			}
		}
	}
	switch {
	case hasDirectOpenAPI:
		followUps = append(followUps, "Direct OpenAPI artifact is available from the sibling apitools cache; copy or import it into this example's openapi/ directory before synthesis.")
	case hasAdvisoryOverlay:
		followUps = append(followUps, "Reviewed advisory OpenAPI overlay is available from the sibling apitools cache; iCoT can materialize it for operation selection.")
	case hasNonOpenAPIMachineArtifact:
		followUps = append(followUps, "Machine-readable metadata exists, but it is not directly OpenAPI; OpenUdon synthesis needs lowering or a user-provided OpenAPI file.")
	case provider.UserOpenAPINeed == catalog.UserOpenAPINeedLikely || provider.OfficialOpenAPIAvailability == catalog.SpecAvailabilityUnavailable:
		followUps = append(followUps, "No direct OpenAPI artifact is recorded; provide or generate a local OpenAPI slice before synthesis.")
	}
	return dedupeStrings(followUps)
}
