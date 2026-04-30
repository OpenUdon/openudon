package openapidisco

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/genelet/openapisearch"
	"gopkg.in/yaml.v3"
)

const (
	importMaxBytes = 20 * 1024 * 1024
	importTimeout  = 30 * time.Second
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
	report, err := d.searchClient().Search(ctx, openapisearch.SearchOptions{
		Query:  projectText,
		Limit:  1,
		Source: openapisearch.SourceAPIsGuru,
	})
	if err != nil {
		return Candidate{}, err
	}
	if len(report.Results) == 0 {
		return Candidate{}, fmt.Errorf("no APIs.guru match found for project brief")
	}

	best := report.Results[0]
	candidate, err := d.ImportURL(ctx, openAPIDir, baseDir, best.SpecURL, best.Provider)
	if err != nil {
		return Candidate{}, err
	}
	candidate.Source = "apis.guru:" + best.Provider
	candidate.Score = best.Score
	return candidate, nil
}

func (d *Discoverer) ImportURL(ctx context.Context, openAPIDir, baseDir, rawURL, suggestedName string) (Candidate, error) {
	imported, err := d.searchClient().Import(ctx, openapisearch.ImportOptions{
		URL:  rawURL,
		Dir:  openAPIDir,
		Name: suggestedName,
	})
	if err != nil {
		return Candidate{}, err
	}
	rel, err := filepath.Rel(baseDir, imported.Path)
	if err != nil {
		return Candidate{}, err
	}
	return Candidate{
		Path:         imported.Path,
		RelativePath: filepath.ToSlash(rel),
		Title:        imported.Title,
		Description:  imported.Description,
	}, nil
}

func (d *Discoverer) searchClient() *openapisearch.Client {
	client := &openapisearch.Client{
		Timeout:  importTimeout,
		MaxBytes: importMaxBytes,
	}
	if d != nil {
		client.HTTPClient = d.HTTPClient
		client.APIsGuruListURL = d.APIsGuruListURL
	}
	return client
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

var urlPattern = regexp.MustCompile(`https?://[^\s<>"')]+`)

func extractURLs(text string) []string {
	matches := urlPattern.FindAllString(text, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, strings.TrimRight(match, ".,;:"))
	}
	return out
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
