package elicitor

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/browsertools/registry"
	"github.com/OpenUdon/evidence/redact"
	"github.com/OpenUdon/openudon/internal/netpolicy"
)

// BrowserRegistryCandidate is prompt-safe metadata for one verified static
// registry capability. Bundle bytes and learned browser observations are never
// included in author or agent reports.
type BrowserRegistryCandidate struct {
	Registry    string   `json:"registry"`
	ID          string   `json:"id"`
	Release     string   `json:"release"`
	Title       string   `json:"title"`
	Origins     []string `json:"origins"`
	Actions     []string `json:"actions"`
	Score       int      `json:"score"`
	Digest      string   `json:"digest"`
	Status      string   `json:"status"`
	Provenance  string   `json:"provenance"`
	Materialize string   `json:"materialize_target"`
}

// BrowserRegistryBlocker makes denial, timeout, unsafe-host, empty-result, and
// verification failures visible without turning them into placeholder source
// documents.
type BrowserRegistryBlocker struct {
	Code       string `json:"code"`
	Registry   string `json:"registry"`
	Message    string `json:"message"`
	Deferrable bool   `json:"deferrable"`
}

// BrowserRegistryDiscovery contains only verified active profiles and visible
// blockers. It performs no writes.
type BrowserRegistryDiscovery struct {
	Candidates []BrowserRegistryCandidate `json:"candidates"`
	Blockers   []BrowserRegistryBlocker   `json:"blockers"`
	Docs       []APIDocument              `json:"-"`
	Plans      []SourceMaterialization    `json:"-"`
}

type BrowserRegistryDiscoveryOptions struct {
	Locations        []string
	Query            string
	Policy           string
	Approved         bool
	At               time.Time
	HTTPClient       *http.Client
	Resolver         netpolicy.Resolver
	AllowUnsafeHosts bool
	Timeout          time.Duration
}

// DiscoverBrowserRegistrySources searches caller-selected local or static
// HTTPS registries. HTTPS is governed by the same iCoT network decision as API
// lookup; local registries remain available under --network never.
func DiscoverBrowserRegistrySources(ctx context.Context, locations []string, query, policy string, approved bool, at time.Time) (BrowserRegistryDiscovery, error) {
	return DiscoverBrowserRegistrySourcesWithOptions(ctx, BrowserRegistryDiscoveryOptions{
		Locations: locations, Query: query, Policy: policy, Approved: approved, At: at,
	})
}

// DiscoverBrowserRegistrySourcesWithOptions exposes testable transport
// injection while retaining the same safe defaults used by the CLI.
func DiscoverBrowserRegistrySourcesWithOptions(ctx context.Context, options BrowserRegistryDiscoveryOptions) (BrowserRegistryDiscovery, error) {
	report := BrowserRegistryDiscovery{Candidates: []BrowserRegistryCandidate{}, Blockers: []BrowserRegistryBlocker{}}
	if options.At.IsZero() {
		return report, fmt.Errorf("browser registry assessment time is required")
	}
	policy := strings.ToLower(strings.TrimSpace(options.Policy))
	if policy == "" {
		policy = "ask"
	}
	if policy != "never" && policy != "ask" && policy != "allow" {
		return report, fmt.Errorf("network policy must be never, ask, or allow")
	}
	seenLocation := map[string]bool{}
	seenCoordinate := map[string]bool{}
	seenTarget := map[string]bool{}
	var safeClient *http.Client
	for _, rawLocation := range options.Locations {
		location := strings.TrimSpace(rawLocation)
		if location == "" || seenLocation[location] {
			continue
		}
		seenLocation[location] = true
		safeLocation := browserRegistryOperatorText(location)
		remote := browserRegistryIsRemote(location)
		if remote && (policy == "never" || policy == "ask" && !options.Approved) {
			code, message := "browser_registry.network_denied", "Remote browser registry lookup is disabled by --network never."
			if policy == "ask" {
				code, message = "browser_registry.approval_required", "Approve the bounded static browser registry lookup, provide a local profile, or defer source selection."
			}
			report.Blockers = append(report.Blockers, BrowserRegistryBlocker{Code: code, Registry: safeLocation, Message: message, Deferrable: true})
			continue
		}
		if remote && safeClient == nil {
			var err error
			safeClient, err = netpolicy.SafeHTTPClient(options.HTTPClient, netpolicy.Options{AllowUnsafe: options.AllowUnsafeHosts, Resolver: options.Resolver})
			if err != nil {
				return report, err
			}
		}
		search, pulls, err := browserRegistryLookup(ctx, browserRegistryLookupOptions{
			Location: location, Query: options.Query, At: options.At, Remote: remote,
			HTTPClient: safeClient, AllowUnsafeHosts: options.AllowUnsafeHosts,
			Timeout: firstPositiveDuration(options.Timeout, registry.DefaultTimeout),
		})
		if err != nil {
			report.Blockers = append(report.Blockers, browserRegistryErrorBlocker(safeLocation, err))
			continue
		}
		if len(search.Results) == 0 {
			report.Blockers = append(report.Blockers, BrowserRegistryBlocker{Code: "browser_registry.empty", Registry: safeLocation, Message: "The bounded static browser registry search returned no active capability profiles.", Deferrable: true})
			continue
		}
		for _, attempt := range pulls {
			if len(report.Candidates) >= registry.DefaultMaxResults {
				break
			}
			match := attempt.Match
			coordinate := registry.Coordinate{ID: match.Entry.ID, Release: match.Entry.Release}
			coordinateKey := coordinate.ID + "@" + coordinate.Release
			registryCoordinateKey := location + "\x00" + coordinateKey
			if seenCoordinate[registryCoordinateKey] {
				continue
			}
			if attempt.Err != nil {
				report.Blockers = append(report.Blockers, browserRegistryErrorBlocker(safeLocation+"#"+browserRegistryOperatorText(coordinateKey), attempt.Err))
				continue
			}
			pulled := attempt.Pulled
			plan, doc, candidate, convertErr := browserRegistryMaterialization(location, match.Score, pulled, options.At)
			if convertErr != nil {
				report.Blockers = append(report.Blockers, BrowserRegistryBlocker{Code: "browser_registry.invalid_bundle", Registry: safeLocation, Message: browserRegistryOperatorText(convertErr.Error()), Deferrable: true})
				continue
			}
			if assignUniqueBrowserRegistryTarget(&plan, &candidate, seenTarget, location) {
				doc = browserProfileDocument(plan, &pulled.Bundle.Payload.Profile)
			}
			seenCoordinate[registryCoordinateKey] = true
			seenTarget[plan.TargetPath] = true
			report.Candidates = append(report.Candidates, candidate)
			report.Plans = append(report.Plans, plan)
			report.Docs = append(report.Docs, doc)
		}
	}
	sort.SliceStable(report.Candidates, func(i, j int) bool {
		if report.Candidates[i].Score != report.Candidates[j].Score {
			return report.Candidates[i].Score > report.Candidates[j].Score
		}
		if report.Candidates[i].ID != report.Candidates[j].ID {
			return report.Candidates[i].ID < report.Candidates[j].ID
		}
		return report.Candidates[i].Release < report.Candidates[j].Release
	})
	sort.SliceStable(report.Blockers, func(i, j int) bool {
		if report.Blockers[i].Registry != report.Blockers[j].Registry {
			return report.Blockers[i].Registry < report.Blockers[j].Registry
		}
		return report.Blockers[i].Code < report.Blockers[j].Code
	})
	report.Plans = normalizeSourcePlan(report.Plans)
	sort.SliceStable(report.Docs, func(i, j int) bool { return report.Docs[i].RelativePath < report.Docs[j].RelativePath })
	return report, nil
}

func assignUniqueBrowserRegistryTarget(plan *SourceMaterialization, candidate *BrowserRegistryCandidate, seen map[string]bool, location string) bool {
	if plan == nil || candidate == nil || !seen[plan.TargetPath] {
		return false
	}
	baseID := plan.ID
	targetDigest := sha256.Sum256([]byte(location + "\x00" + plan.SourceSHA256))
	baseSuffix := hex.EncodeToString(targetDigest[:6])
	for variant := 1; ; variant++ {
		suffix := baseSuffix
		if variant > 1 {
			suffix += fmt.Sprintf("-%d", variant)
		}
		plan.ID = baseID + "-" + suffix
		plan.TargetPath = filepath.ToSlash(filepath.Join("browser-profiles", plan.ID+".json"))
		if !seen[plan.TargetPath] {
			candidate.Materialize = plan.TargetPath
			return true
		}
	}
}

func firstPositiveDuration(value, fallback time.Duration) time.Duration {
	if value > 0 {
		return value
	}
	return fallback
}

// MergeBrowserRegistrySources adds verified registry profiles to an existing
// local discovery result while preserving API-first document ordering.
func MergeBrowserRegistrySources(discovery LocalSourceDiscovery, registryDiscovery BrowserRegistryDiscovery) LocalSourceDiscovery {
	discovery.Docs = append(discovery.Docs, registryDiscovery.Docs...)
	discovery.Plans = normalizeSourcePlan(append(discovery.Plans, registryDiscovery.Plans...))
	sort.SliceStable(discovery.Docs, func(i, j int) bool {
		if apiDocumentPriority(discovery.Docs[i]) != apiDocumentPriority(discovery.Docs[j]) {
			return apiDocumentPriority(discovery.Docs[i]) < apiDocumentPriority(discovery.Docs[j])
		}
		return discovery.Docs[i].RelativePath < discovery.Docs[j].RelativePath
	})
	return discovery
}

func browserRegistryMaterialization(location string, score int, pulled registry.PullResult, at time.Time) (SourceMaterialization, APIDocument, BrowserRegistryCandidate, error) {
	if pulled.Bundle == nil {
		return SourceMaterialization{}, APIDocument{}, BrowserRegistryCandidate{}, fmt.Errorf("browser registry returned an empty capability bundle")
	}
	value := &pulled.Bundle.Payload.Profile
	if err := validateBrowserAuthoringProfile(value); err != nil {
		return SourceMaterialization{}, APIDocument{}, BrowserRegistryCandidate{}, err
	}
	expiresAt, err := browserProfileExpiry(value)
	if err != nil {
		return SourceMaterialization{}, APIDocument{}, BrowserRegistryCandidate{}, err
	}
	if !expiresAt.After(at) {
		return SourceMaterialization{}, APIDocument{}, BrowserRegistryCandidate{}, fmt.Errorf("browser profile %s@%s expired at %s", pulled.Entry.ID, pulled.Entry.Release, expiresAt.Format(time.RFC3339))
	}
	materialized, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return SourceMaterialization{}, APIDocument{}, BrowserRegistryCandidate{}, err
	}
	materialized = append(materialized, '\n')
	digest := sha256.Sum256(materialized)
	id := strings.Trim(sourceIDSanitizer.ReplaceAllString(pulled.Entry.ID, "-"), ".-")
	if id == "" {
		return SourceMaterialization{}, APIDocument{}, BrowserRegistryCandidate{}, fmt.Errorf("browser registry capability has no safe ID")
	}
	target := filepath.ToSlash(filepath.Join("browser-profiles", id+".json"))
	provenance := browserRegistryOperatorText(firstNonEmpty(pulled.Entry.Provenance.Source, pulled.Source, "browser-registry:"+location))
	title := browserRegistryOperatorText(value.Info.Title)
	safeLocation := browserRegistryOperatorText(location)
	plan := SourceMaterialization{
		Kind: browserSourceFamily, SourceKind: string(browsertools.LocalSourceBundle), ID: id, Release: pulled.Entry.Release,
		SourcePath: pulled.Source, TargetPath: target, SHA256: hex.EncodeToString(digest[:]),
		SourceSHA256: strings.TrimPrefix(strings.ToLower(pulled.Entry.Bundle.Digest.String()), "sha256:"),
		Title:        title, OperationCount: len(value.Actions), Actions: value.SortedActionNames(), Origins: append([]string(nil), value.Info.Origin...),
		Lifecycle: "active", ExpiresAt: expiresAt.Format(time.RFC3339), LoginStateRequired: value.Info.LoginStateRequired,
		Provenance: provenance, Registry: safeLocation, RegistryCoordinate: pulled.Entry.ID + "@" + pulled.Entry.Release,
		MaterializedContent: append([]byte(nil), materialized...),
	}
	candidate := BrowserRegistryCandidate{
		Registry: safeLocation, ID: pulled.Entry.ID, Release: pulled.Entry.Release, Title: title,
		Origins: append([]string(nil), value.Info.Origin...), Actions: value.SortedActionNames(), Score: score,
		Digest: pulled.Entry.Bundle.Digest.String(), Status: "active", Provenance: provenance, Materialize: target,
	}
	return plan, browserProfileDocument(plan, value), candidate, nil
}

func browserRegistryErrorBlocker(location string, err error) BrowserRegistryBlocker {
	code := "browser_registry.lookup_failed"
	rawMessage := browserRegistryOperatorText(err.Error())
	var networkErr net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded), errors.As(err, &networkErr) && networkErr.Timeout():
		code = "browser_registry.timeout"
	case strings.Contains(strings.ToLower(rawMessage), "unsafe") || strings.Contains(strings.ToLower(rawMessage), "private") || strings.Contains(strings.ToLower(rawMessage), "loopback"):
		code = "browser_registry.unsafe_host"
	case strings.Contains(strings.ToLower(rawMessage), "not found"):
		code = "browser_registry.empty"
	}
	message := map[string]string{
		"browser_registry.lookup_failed": "The bounded browser registry lookup failed.",
		"browser_registry.timeout":       "The bounded browser registry lookup timed out.",
		"browser_registry.unsafe_host":   "The browser registry endpoint did not satisfy the remote-host safety policy.",
		"browser_registry.empty":         "The browser registry capability was not found.",
	}[code]
	return BrowserRegistryBlocker{Code: code, Registry: browserRegistryOperatorText(location), Message: message, Deferrable: true}
}

func browserRegistryOperatorText(value string) string {
	value = redact.String(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	const maxRunes = 512
	runes := []rune(value)
	if len(runes) > maxRunes {
		value = string(runes[:maxRunes])
	}
	return value
}

func browserRegistryIsRemote(location string) bool {
	parsed, err := url.Parse(strings.TrimSpace(location))
	return err == nil && (strings.EqualFold(parsed.Scheme, "https") || strings.EqualFold(parsed.Scheme, "http"))
}
