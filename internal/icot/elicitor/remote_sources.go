package elicitor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/apitools/catalog"
)

const (
	remoteLookupDeadline = 8 * time.Second
	remoteLookupLimit    = 3
)

type RemoteSourceCandidate struct {
	Kind       string `json:"kind"`
	ID         string `json:"id"`
	Title      string `json:"title,omitempty"`
	URL        string `json:"url"`
	Provenance string `json:"provenance"`
	Score      int    `json:"score,omitempty"`
}

type RemoteSourceBlocker struct {
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	ProviderHints []string `json:"provider_hints,omitempty"`
	SourceHints   []string `json:"source_hints,omitempty"`
	Deferrable    bool     `json:"deferrable"`
}

type RemoteSourceLookupReport struct {
	Version    string                   `json:"version"`
	Policy     string                   `json:"policy"`
	Candidates []RemoteSourceCandidate  `json:"candidates,omitempty"`
	Blocker    *RemoteSourceBlocker     `json:"blocker,omitempty"`
	Attempts   []apitools.SearchAttempt `json:"attempts,omitempty"`
}

type RemoteSourceLookupOptions struct {
	Policy           string
	Approved         bool
	APIsGuruListURL  string
	HTTPClient       *http.Client
	AllowUnsafeHosts bool
	Deadline         time.Duration
}

// DiscoverRemoteSourceHints performs metadata lookup only after explicit
// approval. It consults the curated apitools catalog and exactly one APIs.guru
// search, never a general crawler and never writes a source document.
func DiscoverRemoteSourceHints(ctx context.Context, query string, opts RemoteSourceLookupOptions) (RemoteSourceLookupReport, error) {
	policy := strings.ToLower(strings.TrimSpace(opts.Policy))
	if policy == "" {
		policy = "ask"
	}
	report := RemoteSourceLookupReport{Version: "openudon.icot-remote-source-lookup.v2", Policy: policy}
	if policy == "never" || (policy == "ask" && !opts.Approved) {
		code := "network.denied"
		message := "Remote API-source lookup was not approved."
		if policy == "ask" {
			code, message = "network.approval_required", "Approve the bounded catalog and APIs.guru lookup, provide a local source, or defer source selection."
		}
		report.Blocker = remoteBlocker(code, message, query)
		return report, nil
	}
	if policy != "allow" && policy != "ask" {
		return report, fmt.Errorf("network policy must be never, ask, or allow")
	}
	deadline := opts.Deadline
	if deadline <= 0 || deadline > remoteLookupDeadline {
		deadline = remoteLookupDeadline
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	for _, provider := range catalog.BuiltInProviders() {
		score := apitools.ScoreText(query, strings.Join(append([]string{provider.ID, provider.DisplayName}, provider.Aliases...), " "))
		if score <= 0 {
			continue
		}
		for _, ref := range provider.SpecReferences {
			if ref.Kind == catalog.SpecKindHumanDocs || strings.TrimSpace(ref.URL) == "" {
				continue
			}
			if err := safeRemoteSourceURL(ref.URL, opts.AllowUnsafeHosts); err != nil {
				report.Attempts = append(report.Attempts, apitools.SearchAttempt{Source: "apitools-catalog", URL: ref.URL, Status: "rejected", Detail: err.Error()})
				continue
			}
			report.Candidates = append(report.Candidates, RemoteSourceCandidate{Kind: string(ref.Kind), ID: ref.ID, Title: provider.DisplayName, URL: ref.URL, Provenance: "apitools catalog " + provider.ID, Score: score})
		}
	}
	sortRemoteCandidates(report.Candidates)
	if len(report.Candidates) > remoteLookupLimit-1 {
		report.Candidates = report.Candidates[:remoteLookupLimit-1]
	}

	client := &apitools.Client{HTTPClient: opts.HTTPClient, APIsGuruListURL: opts.APIsGuruListURL, Timeout: deadline, MaxBytes: apitools.DefaultMaxBytes, AllowUnsafeHosts: opts.AllowUnsafeHosts}
	guru, err := client.Search(deadlineCtx, apitools.SearchOptions{Query: query, Limit: remoteLookupLimit, Source: apitools.SourceAPIsGuru, CacheMode: apitools.CacheModeBypass})
	report.Attempts = append(report.Attempts, guru.Attempts...)
	if err != nil {
		code := "network.lookup_failed"
		if deadlineCtx.Err() != nil {
			code = "network.timeout"
		}
		report.Blocker = remoteBlocker(code, err.Error(), query)
		return report, nil
	}
	for _, result := range guru.Results {
		if len(report.Candidates) >= remoteLookupLimit {
			break
		}
		if err := safeRemoteSourceURL(result.SpecURL, opts.AllowUnsafeHosts); err != nil {
			report.Attempts = append(report.Attempts, apitools.SearchAttempt{Source: "apis-guru", URL: result.SpecURL, Status: "rejected", Detail: err.Error()})
			continue
		}
		candidate := RemoteSourceCandidate{Kind: "openapi", ID: result.ID, Title: result.Title, URL: result.SpecURL, Provenance: "apis.guru", Score: result.Score}
		if !hasRemoteCandidate(report.Candidates, candidate.URL) {
			report.Candidates = append(report.Candidates, candidate)
		}
	}
	sortRemoteCandidates(report.Candidates)
	if len(report.Candidates) == 0 {
		report.Blocker = remoteBlocker("network.empty", "The bounded catalog and APIs.guru lookup returned no validated source candidates.", query)
	}
	return report, nil
}

func remoteBlocker(code, message, query string) *RemoteSourceBlocker {
	return &RemoteSourceBlocker{Code: code, Message: message, ProviderHints: queryTokens(query, 3), SourceHints: []string{"--api-source KIND:ID=PATH", "--source-root PATH"}, Deferrable: true}
}

func safeRemoteSourceURL(raw string, allowUnsafe bool) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("invalid source URL")
	}
	if parsed.Scheme != "https" && !(allowUnsafe && parsed.Scheme == "http") {
		return fmt.Errorf("source URL must use HTTPS")
	}
	host := strings.ToLower(parsed.Hostname())
	if allowUnsafe {
		return nil
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return fmt.Errorf("unsafe local source host")
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
		return fmt.Errorf("unsafe private source host")
	}
	return nil
}

func hasRemoteCandidate(candidates []RemoteSourceCandidate, rawURL string) bool {
	for _, candidate := range candidates {
		if candidate.URL == rawURL {
			return true
		}
	}
	return false
}

func sortRemoteCandidates(candidates []RemoteSourceCandidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].Provenance != candidates[j].Provenance {
			return candidates[i].Provenance < candidates[j].Provenance
		}
		return candidates[i].ID < candidates[j].ID
	})
}

func queryTokens(query string, limit int) []string {
	seen := map[string]bool{}
	var out []string
	for _, token := range strings.FieldsFunc(strings.ToLower(query), func(r rune) bool { return r < 'a' || r > 'z' }) {
		if len(token) < 3 || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
		if len(out) >= limit {
			break
		}
	}
	return out
}
