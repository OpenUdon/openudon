package openapidisco

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultAPIsGuruListURL = "https://api.apis.guru/v2/list.json"
	importMaxBytes         = 20 * 1024 * 1024
	importTimeout          = 30 * time.Second
)

type Candidate struct {
	Path         string
	RelativePath string
	Title        string
	Description  string
	Source       string
	Score        int
}

type DiscoveryReport struct {
	Attempts []DiscoveryAttempt `json:"attempts,omitempty"`
}

type DiscoveryAttempt struct {
	Kind   string `json:"kind"`
	Source string `json:"source,omitempty"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type Discoverer struct {
	HTTPClient      *http.Client
	APIsGuruListURL string
}

func (d *Discoverer) Discover(ctx context.Context, exampleDir, projectText string) ([]Candidate, error) {
	candidates, _, err := d.DiscoverWithReport(ctx, exampleDir, projectText)
	return candidates, err
}

func (d *Discoverer) DiscoverWithReport(ctx context.Context, exampleDir, projectText string) ([]Candidate, DiscoveryReport, error) {
	openAPIDir := filepath.Join(exampleDir, "openapi")
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		return nil, DiscoveryReport{}, err
	}

	var candidates []Candidate
	var report DiscoveryReport
	local, err := LocalFiles(openAPIDir, exampleDir, projectText)
	if err != nil {
		return nil, report, err
	}
	report.Attempts = append(report.Attempts, DiscoveryAttempt{
		Kind:   "local",
		Source: filepath.ToSlash(openAPIDir),
		Status: "pass",
		Detail: fmt.Sprintf("%d local OpenAPI document(s)", len(local)),
	})
	candidates = append(candidates, local...)

	imported, attempts := d.ImportProjectURLsWithReport(ctx, openAPIDir, exampleDir, projectText)
	report.Attempts = append(report.Attempts, attempts...)
	candidates = append(candidates, imported...)

	if len(candidates) == 0 {
		fromGuru, err := d.ImportBestAPIsGuruMatch(ctx, openAPIDir, exampleDir, projectText)
		if err != nil {
			report.Attempts = append(report.Attempts, DiscoveryAttempt{Kind: "apis.guru", Status: "fail", Detail: err.Error()})
			return nil, report, err
		}
		if fromGuru.Path != "" {
			report.Attempts = append(report.Attempts, DiscoveryAttempt{Kind: "apis.guru", Source: fromGuru.Source, Status: "pass", Detail: fromGuru.RelativePath})
			candidates = append(candidates, fromGuru)
		}
	}

	sortCandidates(candidates)
	return candidates, report, nil
}

func (d *Discoverer) ImportProjectURLs(ctx context.Context, openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	out, _ := d.ImportProjectURLsWithReport(ctx, openAPIDir, baseDir, projectText)
	return out, nil
}

func (d *Discoverer) ImportProjectURLsWithReport(ctx context.Context, openAPIDir, baseDir, projectText string) ([]Candidate, []DiscoveryAttempt) {
	var out []Candidate
	var attempts []DiscoveryAttempt
	seen := map[string]bool{}
	for _, rawURL := range extractURLs(projectText) {
		if seen[rawURL] {
			continue
		}
		seen[rawURL] = true
		candidate, err := d.ImportURL(ctx, openAPIDir, baseDir, rawURL, "")
		if err != nil {
			attempts = append(attempts, DiscoveryAttempt{Kind: "url", Source: rawURL, Status: "fail", Detail: err.Error()})
			continue
		}
		candidate.Source = "url:" + rawURL
		candidate.Score = scoreText(projectText, candidate.Title+" "+candidate.Description+" "+candidate.RelativePath)
		attempts = append(attempts, DiscoveryAttempt{Kind: "url", Source: rawURL, Status: "pass", Detail: candidate.RelativePath})
		out = append(out, candidate)
	}
	return out, attempts
}

func LocalFiles(openAPIDir, baseDir, projectText string) ([]Candidate, error) {
	var candidates []Candidate
	err := filepath.WalkDir(openAPIDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !hasOpenAPIExt(path) {
			return nil
		}
		ok, title, description, err := openAPIFileMetadata(path)
		if err != nil || !ok {
			return nil
		}
		rel, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		candidates = append(candidates, Candidate{
			Path:         path,
			RelativePath: rel,
			Title:        title,
			Description:  description,
			Source:       "local",
			Score:        scoreText(projectText, title+" "+description+" "+rel),
		})
		return nil
	})
	return candidates, err
}

func (d *Discoverer) ImportBestAPIsGuruMatch(ctx context.Context, openAPIDir, baseDir, projectText string) (Candidate, error) {
	listURL := ""
	if d != nil {
		listURL = strings.TrimSpace(d.APIsGuruListURL)
	}
	if listURL == "" {
		listURL = defaultAPIsGuruListURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listURL, nil)
	if err != nil {
		return Candidate{}, err
	}
	resp, err := d.redirectSafeClient().Do(req)
	if err != nil {
		return Candidate{}, err
	}
	defer resp.Body.Close()
	if resp.Request != nil && resp.Request.URL != nil {
		if err := rejectPrivateHost(ctx, resp.Request.URL.Hostname()); err != nil {
			return Candidate{}, err
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Candidate{}, fmt.Errorf("APIs.guru list returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, importMaxBytes+1))
	if err != nil {
		return Candidate{}, err
	}
	if len(body) > importMaxBytes {
		return Candidate{}, fmt.Errorf("APIs.guru list is larger than %d bytes", importMaxBytes)
	}

	var raw map[string]apisGuruAPI
	if err := json.Unmarshal(body, &raw); err != nil {
		return Candidate{}, fmt.Errorf("parse APIs.guru list: %w", err)
	}

	var best apisGuruVersion
	var bestName string
	bestScore := 0
	for name, api := range raw {
		version := api.preferred()
		if version.URL == "" {
			continue
		}
		score := scoreText(projectText, name+" "+version.Info.Title+" "+version.Info.Description)
		if score > bestScore {
			bestScore = score
			bestName = name
			best = version
		}
	}
	if bestScore == 0 {
		return Candidate{}, fmt.Errorf("no APIs.guru match found for project brief")
	}

	candidate, err := d.ImportURL(ctx, openAPIDir, baseDir, best.URL, bestName)
	if err != nil {
		return Candidate{}, err
	}
	candidate.Source = "apis.guru:" + bestName
	candidate.Score = bestScore
	return candidate, nil
}

func (d *Discoverer) ImportURL(ctx context.Context, openAPIDir, baseDir, rawURL, suggestedName string) (Candidate, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return Candidate{}, fmt.Errorf("valid OpenAPI URL is required")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Candidate{}, fmt.Errorf("OpenAPI URL scheme must be http or https")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := rejectPrivateHost(ctx, parsed.Hostname()); err != nil {
		return Candidate{}, err
	}
	if err := os.MkdirAll(openAPIDir, 0o755); err != nil {
		return Candidate{}, err
	}

	ctx, cancel := context.WithTimeout(ctx, importTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return Candidate{}, err
	}
	resp, err := d.redirectSafeClient().Do(req)
	if err != nil {
		return Candidate{}, err
	}
	defer resp.Body.Close()
	if resp.Request != nil && resp.Request.URL != nil {
		if err := rejectPrivateHost(ctx, resp.Request.URL.Hostname()); err != nil {
			return Candidate{}, err
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Candidate{}, fmt.Errorf("download OpenAPI document: %s", resp.Status)
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, importMaxBytes+1))
	if err != nil {
		return Candidate{}, err
	}
	if len(content) > importMaxBytes {
		return Candidate{}, fmt.Errorf("OpenAPI document is larger than %d bytes", importMaxBytes)
	}
	if !looksLikeOpenAPIContent(content) {
		return Candidate{}, fmt.Errorf("downloaded document does not look like OpenAPI or Swagger")
	}

	name := uniqueFileName(openAPIDir, fileNameForImport(suggestedName, parsed, content))
	path := filepath.Join(openAPIDir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return Candidate{}, err
	}
	title, description := openAPIInfo(content)
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return Candidate{}, err
	}
	return Candidate{
		Path:         path,
		RelativePath: filepath.ToSlash(rel),
		Title:        title,
		Description:  description,
	}, nil
}

func SelectPrimary(candidates []Candidate) (Candidate, error) {
	if len(candidates) == 0 {
		return Candidate{}, fmt.Errorf("no OpenAPI documents discovered")
	}
	sortCandidates(candidates)
	return candidates[0], nil
}

func sortCandidates(candidates []Candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].RelativePath < candidates[j].RelativePath
	})
}

type apisGuruAPI struct {
	Versions  map[string]apisGuruVersion `json:"versions"`
	Preferred string                     `json:"preferred"`
}

type apisGuruVersion struct {
	URL  string `json:"swaggerUrl"`
	Info struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	} `json:"info"`
}

func (a apisGuruAPI) preferred() apisGuruVersion {
	if a.Preferred != "" {
		if version, ok := a.Versions[a.Preferred]; ok {
			return version
		}
	}
	keys := make([]string, 0, len(a.Versions))
	for key := range a.Versions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return apisGuruVersion{}
	}
	return a.Versions[keys[len(keys)-1]]
}

func (d *Discoverer) client() *http.Client {
	if d != nil && d.HTTPClient != nil {
		return d.HTTPClient
	}
	return http.DefaultClient
}

func (d *Discoverer) redirectSafeClient() *http.Client {
	base := d.client()
	clone := *base
	baseCheck := base.CheckRedirect
	clone.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many OpenAPI URL redirects")
		}
		if req == nil || req.URL == nil {
			return fmt.Errorf("redirect target is missing")
		}
		if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
			return fmt.Errorf("OpenAPI redirect URL scheme must be http or https")
		}
		if err := rejectPrivateHost(req.Context(), req.URL.Hostname()); err != nil {
			return err
		}
		if baseCheck != nil {
			return baseCheck(req, via)
		}
		return nil
	}
	return &clone
}

var urlPattern = regexp.MustCompile(`https?://[^\s<>"')]+`)

func extractURLs(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.TrimRight(match, ".,;:"))
	}
	return out
}

func rejectPrivateHost(ctx context.Context, host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("URL host is required")
	}
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("refusing localhost OpenAPI URL")
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if isUnsafeIP(ip) {
			return fmt.Errorf("refusing private OpenAPI URL host %q", host)
		}
		return nil
	}
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if isUnsafeIP(addr.IP) {
			return fmt.Errorf("refusing private OpenAPI URL host %q", host)
		}
	}
	return nil
}

func isUnsafeIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}

func hasOpenAPIExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func openAPIFileMetadata(path string) (bool, string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, "", "", err
	}
	if !looksLikeOpenAPIContent(data) {
		return false, "", "", nil
	}
	title, description := openAPIInfo(data)
	return true, title, description, nil
}

func looksLikeOpenAPIContent(content []byte) bool {
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}
	if trimmed[0] == '{' {
		var root map[string]any
		if err := json.Unmarshal(trimmed, &root); err != nil {
			return false
		}
		_, hasOpenAPI := root["openapi"]
		_, hasSwagger := root["swagger"]
		return hasOpenAPI || hasSwagger
	}
	var root map[string]any
	if err := yaml.Unmarshal(trimmed, &root); err != nil {
		return false
	}
	_, hasOpenAPI := root["openapi"]
	_, hasSwagger := root["swagger"]
	return hasOpenAPI || hasSwagger
}

func openAPIInfo(content []byte) (string, string) {
	var root struct {
		Info struct {
			Title       string `json:"title" yaml:"title"`
			Description string `json:"description" yaml:"description"`
		} `json:"info" yaml:"info"`
	}
	trimmed := bytes.TrimSpace(content)
	if len(trimmed) == 0 {
		return "", ""
	}
	if trimmed[0] == '{' {
		_ = json.Unmarshal(trimmed, &root)
	} else {
		_ = yaml.Unmarshal(trimmed, &root)
	}
	return strings.TrimSpace(root.Info.Title), strings.TrimSpace(root.Info.Description)
}

func fileNameForImport(suggested string, parsed *url.URL, content []byte) string {
	base := sanitizeFileBase(suggested)
	if base == "" {
		base = sanitizeFileBase(filepath.Base(parsed.Path))
	}
	if base == "" || base == "." || base == "/" {
		sum := sha256.Sum256(content)
		base = "openapi-" + hex.EncodeToString(sum[:])[:12]
	}
	ext := strings.ToLower(filepath.Ext(base))
	if ext != ".json" && ext != ".yaml" && ext != ".yml" {
		base += inferredExt(content)
	}
	return base
}

func sanitizeFileBase(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\\", "/")
	value = filepath.Base(value)
	value = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-_.")
	return value
}

func inferredExt(content []byte) string {
	if bytes.HasPrefix(bytes.TrimSpace(content), []byte("{")) {
		return ".json"
	}
	return ".yaml"
}

func uniqueFileName(dir, name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	candidate := name
	for i := 2; ; i++ {
		if _, err := os.Stat(filepath.Join(dir, candidate)); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d%s", base, i, ext)
	}
}

func scoreText(query, haystack string) int {
	terms := tokens(query)
	if len(terms) == 0 {
		return 0
	}
	hay := strings.ToLower(haystack)
	score := 0
	for _, term := range terms {
		if strings.Contains(hay, term) {
			score++
		}
	}
	return score
}

func tokens(text string) []string {
	seen := map[string]bool{}
	var out []string
	for _, field := range regexp.MustCompile(`[a-zA-Z0-9]+`).FindAllString(strings.ToLower(text), -1) {
		if len(field) < 3 || stopWords[field] || seen[field] {
			continue
		}
		seen[field] = true
		out = append(out, field)
	}
	return out
}

var stopWords = map[string]bool{
	"and": true, "are": true, "but": true, "can": true, "for": true, "from": true,
	"has": true, "into": true, "the": true, "then": true, "this": true, "use": true,
	"when": true, "with": true, "workflow": true, "openapi": true, "api": true,
}
