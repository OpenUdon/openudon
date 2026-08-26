package elicitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	"github.com/OpenUdon/openudon/internal/sourcecatalog"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// LocalSourceDiscovery contains bounded evidence gathered before the first
// interview question. Candidate documents remain external until proposal
// approval.
type LocalSourceDiscovery struct {
	Report        apitools.LocalSourceDiscoveryReport     `json:"report"`
	BrowserReport browsertools.LocalSourceDiscoveryReport `json:"browser_report"`
	Docs          []APIDocument                           `json:"documents,omitempty"`
	Plans         []SourceMaterialization                 `json:"materialization_plans,omitempty"`
}

// BrowserSourceInput is one caller-identified browser profile. Discovery still
// validates the content and the ID is used only as the package-local identity.
type BrowserSourceInput struct {
	ID   string `json:"id" yaml:"id"`
	Path string `json:"path" yaml:"path"`
}

func localSourceDiscoveryBlocker(report apitools.LocalSourceDiscoveryReport) error {
	if !report.Truncated && len(report.Ambiguous) == 0 {
		return nil
	}
	return fmt.Errorf(
		"local source discovery is incomplete (%d ambiguous document(s), truncated=%t); narrow source roots or declare ambiguous files with --api-source KIND:ID=PATH",
		len(report.Ambiguous), report.Truncated,
	)
}

func browserSourceDiscoveryBlocker(report browsertools.LocalSourceDiscoveryReport) error {
	var inactive []string
	for _, candidate := range report.Candidates {
		if candidate.Status != "active" {
			inactive = append(inactive, candidate.Path+" ("+string(candidate.Status)+")")
		}
	}
	if len(report.Truncated) == 0 && len(report.Ambiguous) == 0 && len(inactive) == 0 {
		return nil
	}
	return fmt.Errorf(
		"browser source discovery is incomplete (%d ambiguous document(s), %d truncation diagnostic(s), %d inactive candidate(s)); narrow --source-root, pass --browser-profile ID=PATH, or revalidate the profile/bundle: %s",
		len(report.Ambiguous), len(report.Truncated), len(inactive), strings.Join(inactive, ", "),
	)
}

var sourceIDSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

const (
	localSourceSidecarMaxBytes = 20 << 20
	sidecarProvenancePrefix    = "local-sidecar-for:"
)

// DiscoverLocalAPIs retains the established adapter API while routing local
// inspection through apitools' bounded multi-family discovery contract.
func DiscoverLocalAPIs(exampleDir, projectText string) ([]APIDocument, error) {
	discovery, err := DiscoverAuthoringSources(context.Background(), exampleDir, projectText, nil, nil)
	if err != nil {
		return nil, err
	}
	return discovery.Docs, nil
}

// DiscoverAuthoringSources inspects existing example sources, explicit source
// files, and explicit roots without copying or modifying any deliverables.
func DiscoverAuthoringSources(ctx context.Context, exampleDir, query string, explicit []apitools.LocalSource, roots []string) (LocalSourceDiscovery, error) {
	return DiscoverAuthoringSourcesWithBrowser(ctx, exampleDir, query, explicit, roots, nil, time.Now().UTC())
}

// DiscoverAuthoringSourcesWithBrowser composes Apitools API-family discovery
// with Browsertools browser-profile discovery. Each sibling owns validation of
// its own family; cross-family ambiguity from a mixed root is removed only
// after the other sibling has positively validated the exact same file.
func DiscoverAuthoringSourcesWithBrowser(ctx context.Context, exampleDir, query string, explicit []apitools.LocalSource, roots []string, browserExplicit []BrowserSourceInput, at time.Time) (LocalSourceDiscovery, error) {
	discovery, err := discoverAPIAuthoringSources(ctx, exampleDir, query, explicit, roots)
	if err != nil {
		return discovery, err
	}
	browser, err := discoverBrowserAuthoringSources(ctx, exampleDir, browserExplicit, roots, at)
	if err != nil {
		discovery.BrowserReport = browser.Report
		return discovery, err
	}
	discovery.BrowserReport = browser.Report
	filterCrossFamilyDiagnostics(&discovery.Report, &discovery.BrowserReport)
	discovery.Docs = append(discovery.Docs, browser.Docs...)
	sort.SliceStable(discovery.Docs, func(i, j int) bool {
		if apiDocumentPriority(discovery.Docs[i]) != apiDocumentPriority(discovery.Docs[j]) {
			return apiDocumentPriority(discovery.Docs[i]) < apiDocumentPriority(discovery.Docs[j])
		}
		return discovery.Docs[i].RelativePath < discovery.Docs[j].RelativePath
	})
	discovery.Plans = normalizeSourcePlan(append(discovery.Plans, browser.Plans...))
	return discovery, nil
}

func discoverAPIAuthoringSources(ctx context.Context, exampleDir, query string, explicit []apitools.LocalSource, roots []string) (LocalSourceDiscovery, error) {
	allRoots := append([]string(nil), roots...)
	for _, dir := range sourcecatalog.API() {
		path := filepath.Join(exampleDir, dir)
		if _, err := os.Lstat(path); err == nil {
			allRoots = append(allRoots, path)
		} else if !os.IsNotExist(err) {
			return LocalSourceDiscovery{}, err
		}
	}
	if len(allRoots) == 0 && len(explicit) == 0 {
		return LocalSourceDiscovery{Report: apitools.LocalSourceDiscoveryReport{Version: apitools.LocalSourceDiscoveryVersion}}, nil
	}
	report, err := apitools.DiscoverLocalSources(ctx, apitools.LocalSourceDiscoveryOptions{
		Roots: allRoots, Sources: explicit, Query: query,
	})
	if err != nil {
		return LocalSourceDiscovery{Report: report}, err
	}
	discovery := LocalSourceDiscovery{Report: report}
	for _, candidate := range report.Candidates {
		plan, err := sourceMaterializationForCandidate(exampleDir, candidate)
		if err != nil {
			return discovery, err
		}
		inventory, err := apitools.BuildAPISourceOperationInventory(ctx, apitools.APISourceInventoryOptions{
			Documents: []apitools.APISourceDocument{{
				Kind: candidate.Kind, Name: firstNonEmpty(candidate.ID, candidate.Title), Path: candidate.Path, RelativePath: plan.TargetPath,
			}},
			Query: query,
		})
		if err != nil {
			return discovery, err
		}
		for _, diagnostic := range inventory.Diagnostics {
			if strings.EqualFold(diagnostic.Severity, "error") {
				if strings.HasPrefix(strings.ToLower(strings.TrimSpace(diagnostic.Code)), "prompt.") {
					// Prompt-budget compaction is an execution-critical technical
					// blocker, but it does not invalidate the source document. Keep
					// the bounded operation and its embedded readiness issue so the
					// interview can expose and explicitly defer the missing detail.
					continue
				}
				return discovery, fmt.Errorf("inspect %s: %s", candidate.Path, diagnostic.Message)
			}
		}
		doc := APIDocument{
			ID: firstNonEmpty(candidate.ID, plan.ID), Path: candidate.Path, RelativePath: plan.TargetPath,
			Title: candidate.Title, Operations: inventory.Operations,
		}
		if candidate.Kind == apitools.APISourceKindGoogleDiscovery {
			for i := range doc.Operations {
				doc.Operations[i].OperationID = strings.ReplaceAll(doc.Operations[i].OperationID, ".", "_")
				doc.Operations[i].ID = strings.ReplaceAll(doc.Operations[i].ID, ".", "_")
			}
		}
		if len(inventory.Documents) > 0 {
			doc.Title = firstNonEmpty(doc.Title, inventory.Documents[0].Title, inventory.Documents[0].Name)
			doc.Description = inventory.Documents[0].Description
		}
		discovery.Docs = append(discovery.Docs, doc)
		discovery.Plans = append(discovery.Plans, plan)
		sidecars, err := sourceSidecarPlans(plan)
		if err != nil {
			return discovery, err
		}
		discovery.Plans = append(discovery.Plans, sidecars...)
	}
	sort.SliceStable(discovery.Docs, func(i, j int) bool {
		if apiDocumentPriority(discovery.Docs[i]) != apiDocumentPriority(discovery.Docs[j]) {
			return apiDocumentPriority(discovery.Docs[i]) < apiDocumentPriority(discovery.Docs[j])
		}
		return discovery.Docs[i].RelativePath < discovery.Docs[j].RelativePath
	})
	discovery.Plans = normalizeSourcePlan(discovery.Plans)
	return discovery, nil
}

func sourceSidecarPlans(source SourceMaterialization) ([]SourceMaterialization, error) {
	var plans []SourceMaterialization
	for _, sidecarPath := range packageartifacts.AdvisorySecuritySidecarPathCandidates(source.SourcePath) {
		content, _, err := evidencefile.ReadRegular(sidecarPath, localSourceSidecarMaxBytes)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read security sidecar %s: %w", sidecarPath, err)
		}
		digest := sha256.Sum256(content)
		target, err := sidecarTargetPath(source.SourcePath, source.TargetPath, sidecarPath)
		if err != nil {
			return nil, err
		}
		plans = append(plans, SourceMaterialization{
			Kind: "security-overlay", ID: source.ID + "-security-overlay", SourcePath: sidecarPath,
			TargetPath: target, SHA256: hex.EncodeToString(digest[:]), Title: source.Title + " security overlay",
			Provenance: sidecarProvenancePrefix + source.TargetPath + ";source=local:" + filepath.ToSlash(sidecarPath),
		})
	}
	return plans, nil
}

func sidecarTargetPath(sourcePath, targetPath, sidecarPath string) (string, error) {
	if suffix, ok := strings.CutPrefix(sidecarPath, sourcePath); ok {
		return filepath.ToSlash(targetPath + suffix), nil
	}
	sourceBase := strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath))
	if suffix, ok := strings.CutPrefix(sidecarPath, sourceBase); ok {
		targetBase := strings.TrimSuffix(targetPath, filepath.Ext(targetPath))
		return filepath.ToSlash(targetBase + suffix), nil
	}
	return "", fmt.Errorf("security sidecar %s is not adjacent to source %s", sidecarPath, sourcePath)
}

func sourceMaterializationForCandidate(exampleDir string, candidate apitools.LocalSourceCandidate) (SourceMaterialization, error) {
	absExample, err := filepath.Abs(exampleDir)
	if err != nil {
		return SourceMaterialization{}, err
	}
	absSource, err := filepath.Abs(candidate.Path)
	if err != nil {
		return SourceMaterialization{}, err
	}
	target := ""
	if rel, err := filepath.Rel(absExample, absSource); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
		target = filepath.ToSlash(rel)
	}
	id := strings.TrimSpace(candidate.ID)
	if id == "" {
		id = strings.TrimSuffix(filepath.Base(candidate.Path), filepath.Ext(candidate.Path))
	}
	id = strings.Trim(sourceIDSanitizer.ReplaceAllString(id, "-"), ".-")
	if id == "" {
		return SourceMaterialization{}, fmt.Errorf("cannot derive a stable source ID for %s", candidate.Path)
	}
	if target == "" {
		ext := strings.ToLower(filepath.Ext(candidate.Path))
		if ext == "" {
			ext = defaultSourceExtension(candidate.Kind)
		}
		target = filepath.ToSlash(filepath.Join(sourceFamilyDirectory(candidate.Kind), id+ext))
	}
	return SourceMaterialization{
		Kind: candidate.Kind, ID: id, SourcePath: absSource, TargetPath: target,
		SHA256: candidate.SHA256, Title: candidate.Title, OperationCount: candidate.OperationCount, Provenance: candidate.Provenance,
	}, nil
}

// SourceMaterializationForCandidate returns the canonical package target for
// one already validated Apitools candidate. Frontends use this during an
// explicit upload review before any bytes are staged into the workspace.
func SourceMaterializationForCandidate(exampleDir string, candidate apitools.LocalSourceCandidate) (SourceMaterialization, error) {
	return sourceMaterializationForCandidate(exampleDir, candidate)
}

func sourceFamilyDirectory(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "openapi", "swagger":
		return "openapi"
	case "google-discovery":
		return "google-discovery"
	case "aws-smithy":
		return "aws-smithy"
	case "grpc", "protobuf", "grpc-protobuf":
		return "grpc-protobuf"
	default:
		return strings.ToLower(strings.TrimSpace(kind))
	}
}

func defaultSourceExtension(kind string) string {
	switch sourceFamilyDirectory(kind) {
	case "graphql":
		return ".graphql"
	case "grpc-protobuf":
		return ".proto"
	case "odata":
		return ".xml"
	default:
		return ".json"
	}
}

func mergeSelectedSourcePlansWithBrowser(session Session, available []SourceMaterialization, explicit []apitools.LocalSource, browserExplicit []BrowserSourceInput) []SourceMaterialization {
	selected := append([]SourceMaterialization(nil), session.SourcePlan...)
	explicitPaths := map[string]bool{}
	for _, source := range explicit {
		if abs, err := filepath.Abs(source.Path); err == nil {
			explicitPaths[filepath.Clean(abs)] = true
		}
	}
	for _, source := range browserExplicit {
		if abs, err := filepath.Abs(source.Path); err == nil {
			explicitPaths[filepath.Clean(abs)] = true
		}
	}
	for _, plan := range available {
		if explicitPaths[filepath.Clean(plan.SourcePath)] {
			selected = append(selected, plan)
		}
	}
	return normalizeSourcePlan(selected)
}

func syncSelectedSourcePlans(session Session, available []SourceMaterialization, explicit []apitools.LocalSource) []SourceMaterialization {
	return syncSelectedSourcePlansWithBrowser(session, available, explicit, nil)
}

func syncSelectedSourcePlansWithBrowser(session Session, available []SourceMaterialization, explicit []apitools.LocalSource, browserExplicit []BrowserSourceInput) []SourceMaterialization {
	selected := mergeSelectedSourcePlansWithBrowser(session, available, explicit, browserExplicit)
	selectedVirtualTargets := map[string]bool{}
	for _, plan := range selected {
		if strings.HasPrefix(plan.SourcePath, virtualBrowserPrefix) {
			selectedVirtualTargets[filepath.ToSlash(plan.TargetPath)] = true
		}
	}
	for _, plan := range available {
		if selectedVirtualTargets[filepath.ToSlash(plan.TargetPath)] && strings.HasPrefix(plan.SourcePath, virtualBrowserPrefix) {
			// Exact virtual freshness is validated before synchronization. Add the
			// rediscovered plan so normalization reattaches its unpersisted bytes
			// even before an intent step references the selected source.
			selected = append(selected, plan)
		}
	}
	refs := map[string]bool{}
	if ref := filepath.ToSlash(strings.TrimSpace(firstNonEmpty(session.Intent.Source, session.Intent.OpenAPI))); ref != "" {
		refs[ref] = true
	}
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		if ref := filepath.ToSlash(strings.TrimSpace(firstNonEmpty(step.Source, step.OpenAPI))); ref != "" {
			refs[ref] = true
		}
	})
	for _, plan := range available {
		if refs[filepath.ToSlash(plan.TargetPath)] {
			selected = append(selected, plan)
		}
	}
	selectedTargets := map[string]bool{}
	for _, plan := range selected {
		selectedTargets[filepath.ToSlash(plan.TargetPath)] = true
	}
	for _, plan := range available {
		if target := sidecarSourceTarget(plan.Provenance); target != "" && selectedTargets[target] {
			selected = append(selected, plan)
		}
	}
	return normalizeSourcePlan(selected)
}

func sidecarSourceTarget(provenance string) string {
	value, ok := strings.CutPrefix(strings.TrimSpace(provenance), sidecarProvenancePrefix)
	if !ok {
		return ""
	}
	target, _, _ := strings.Cut(value, ";")
	return filepath.ToSlash(strings.TrimSpace(target))
}

// SyncSelectedSourcePlans exposes the reviewed source-materialization plan to
// the non-interactive adapter without duplicating selection rules.
func SyncSelectedSourcePlans(session Session, available []SourceMaterialization, explicit []apitools.LocalSource) []SourceMaterialization {
	return syncSelectedSourcePlans(session, available, explicit)
}

// SyncSelectedSourcePlansWithBrowser selects explicitly supplied API/browser
// sources and every source referenced by the active intent.
func SyncSelectedSourcePlansWithBrowser(session Session, available []SourceMaterialization, explicit []apitools.LocalSource, browserExplicit []BrowserSourceInput) []SourceMaterialization {
	return syncSelectedSourcePlansWithBrowser(session, available, explicit, browserExplicit)
}
