package elicitor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
)

// SourceRefreshOptions defines the complete local/browser/registry evidence
// authority for one authoring refresh.
type SourceRefreshOptions struct {
	ExampleDir           string
	Query                string
	LocalSources         []apitools.LocalSource
	SourceRoots          []string
	BrowserSources       []BrowserSourceInput
	BrowserRegistries    []string
	BrowserVerifications []string
	NetworkPolicy        string
	At                   time.Time
	RejectIncomplete     bool
}

// SourceRefreshResult is the shared assessment consumed by terminal, agent,
// and engine/UI surfaces. Presentation remains a caller concern.
type SourceRefreshResult struct {
	Session   Session
	Discovery LocalSourceDiscovery
	Registry  BrowserRegistryDiscovery
	Issues    []ReadinessIssue
}

// RefreshSessionSources performs discovery, incomplete-source assessment,
// registry trigger evaluation and selected-digest revalidation, source-plan
// synchronization, and browser-verification attachment as one policy unit.
func RefreshSessionSources(ctx context.Context, session Session, options SourceRefreshOptions) (SourceRefreshResult, error) {
	if options.At.IsZero() {
		options.At = time.Now().UTC()
	}
	policy := firstNonEmpty(strings.ToLower(strings.TrimSpace(options.NetworkPolicy)), "ask")
	discovery, err := DiscoverAuthoringSourcesWithBrowser(ctx, options.ExampleDir, options.Query, options.LocalSources, options.SourceRoots, options.BrowserSources, options.At)
	if err != nil {
		return SourceRefreshResult{}, err
	}
	issues := AssessSourceDiscovery(discovery)
	if options.RejectIncomplete && len(issues) > 0 {
		return SourceRefreshResult{}, errors.Join(localSourceDiscoveryBlocker(discovery.Report), browserSourceDiscoveryBlocker(discovery.BrowserReport))
	}
	registryReport := BrowserRegistryDiscovery{Candidates: []BrowserRegistryCandidate{}, Blockers: []BrowserRegistryBlocker{}}
	if len(options.BrowserRegistries) > 0 && shouldRefreshBrowserRegistry(session, discovery) {
		approved := policy == "allow" || strings.EqualFold(session.Interview.Metadata["browser_registry_lookup_decision"], "allow")
		registryReport, err = DiscoverBrowserRegistrySources(ctx, options.BrowserRegistries, firstNonEmpty(session.Boundary.Outcome, session.Project.Goal, options.Query), policy, approved, options.At)
		if err != nil {
			return SourceRefreshResult{}, err
		}
		discovery = MergeBrowserRegistrySources(discovery, registryReport)
	}
	if err := RequireFreshRegistrySources(session.SourcePlan, registryReport.Plans); err != nil {
		return SourceRefreshResult{}, err
	}
	session.SourcePlan = SyncSelectedSourcePlansWithBrowser(session, discovery.Plans, options.LocalSources, options.BrowserSources)
	session.SourcePlan, err = AttachBrowserVerifications(session.SourcePlan, options.BrowserVerifications, options.At)
	if err != nil {
		return SourceRefreshResult{}, err
	}
	if session.Interview.Metadata == nil {
		session.Interview.Metadata = map[string]string{}
	}
	session.Interview.Metadata["network_policy"] = policy
	if len(options.BrowserRegistries) > 0 {
		session.Interview.Metadata["browser_registry_configured"] = "true"
	}
	session.Normalize()
	return SourceRefreshResult{Session: session, Discovery: discovery, Registry: registryReport, Issues: issues}, nil
}

func shouldRefreshBrowserRegistry(session Session, discovery LocalSourceDiscovery) bool {
	if len(discovery.Docs) == 0 || session.BrowserRoute == "browser" {
		return true
	}
	for _, source := range session.SourcePlan {
		if source.Kind == browserSourceFamily && strings.TrimSpace(source.Registry) != "" {
			return true
		}
	}
	return false
}

// RequireFreshRegistrySources binds every selected registry profile to the
// exact freshly pulled coordinate, target, and content digests.
func RequireFreshRegistrySources(selected, discovered []SourceMaterialization) error {
	for _, source := range selected {
		if source.Kind != browserSourceFamily || strings.TrimSpace(source.Registry) == "" {
			continue
		}
		matched := false
		for _, candidate := range discovered {
			if candidate.Kind == source.Kind && candidate.Registry == source.Registry && candidate.RegistryCoordinate == source.RegistryCoordinate && candidate.TargetPath == source.TargetPath && strings.EqualFold(candidate.SHA256, source.SHA256) && strings.EqualFold(candidate.SourceSHA256, source.SourceSHA256) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("selected browser registry profile %s could not be freshly revalidated; use an available configured registry or provide the profile explicitly", firstNonEmpty(source.RegistryCoordinate, source.ID))
		}
	}
	return nil
}

// AssessSourceDiscovery expresses incomplete API and browser discovery in the
// engine-neutral readiness vocabulary.
func AssessSourceDiscovery(discovery LocalSourceDiscovery) []ReadinessIssue {
	var issues []ReadinessIssue
	if discovery.Report.Truncated || len(discovery.Report.Ambiguous) > 0 {
		issues = append(issues, ReadinessIssue{Code: "source_discovery_blocked", Severity: readinessBlocking, Slot: "source.selection", Message: "Local source discovery is incomplete; narrow roots or declare ambiguous documents with --api-source KIND:ID=PATH."})
	}
	inactive := 0
	for _, candidate := range discovery.BrowserReport.Candidates {
		if candidate.Status != "active" {
			inactive++
		}
	}
	if len(discovery.BrowserReport.Truncated) > 0 || len(discovery.BrowserReport.Ambiguous) > 0 || inactive > 0 {
		issues = append(issues, ReadinessIssue{Code: "browser_source_discovery_blocked", Severity: readinessBlocking, Slot: "source.browser", Message: "Browser source discovery is incomplete; narrow roots or declare a verified profile with --browser-profile ID=PATH."})
	}
	return issues
}
