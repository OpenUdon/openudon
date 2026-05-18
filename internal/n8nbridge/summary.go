package n8nbridge

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
)

const SummaryVersion = "openudon.n8n-pattern-summary.v1"

type Summary struct {
	Version              string                  `json:"version"`
	Fixture              string                  `json:"fixture"`
	Boundary             string                  `json:"boundary"`
	Source               SourceEvidence          `json:"source"`
	Services             []Service               `json:"services"`
	Nodes                []Node                  `json:"nodes"`
	OpenAPICandidates    []OpenAPICandidate      `json:"openapi_candidates,omitempty"`
	CredentialBindings   []CredentialBinding     `json:"credential_bindings,omitempty"`
	DataFlowHints        []DataFlowHint          `json:"data_flow_hints,omitempty"`
	UnsupportedSemantics []UnsupportedDiagnostic `json:"unsupported_semantics,omitempty"`
	GeneratedCandidates  GeneratedCandidates     `json:"generated_candidates"`
	Validation           ValidationEvidence      `json:"validation"`
	ReviewNotes          []string                `json:"review_notes,omitempty"`
}

type SourceEvidence struct {
	Kind  string   `json:"kind"`
	Paths []string `json:"paths,omitempty"`
	Notes []string `json:"notes,omitempty"`
}

type Service struct {
	Name       string   `json:"name"`
	Role       string   `json:"role,omitempty"`
	Operations []string `json:"operations,omitempty"`
}

type Node struct {
	Name                string   `json:"name"`
	Type                string   `json:"type"`
	Resource            string   `json:"resource,omitempty"`
	Operation           string   `json:"operation,omitempty"`
	OpenUdonStep        string   `json:"openudon_step,omitempty"`
	OpenAPIOperationID  string   `json:"openapi_operation_id,omitempty"`
	MappingStatus       string   `json:"mapping_status"`
	UnsupportedSemantic []string `json:"unsupported_semantics,omitempty"`
}

type OpenAPICandidate struct {
	Path        string   `json:"path"`
	OperationID string   `json:"operation_id"`
	Status      string   `json:"status"`
	Notes       []string `json:"notes,omitempty"`
}

type CredentialBinding struct {
	Name    string `json:"name"`
	Service string `json:"service"`
	Source  string `json:"source,omitempty"`
}

type DataFlowHint struct {
	From   string            `json:"from"`
	To     string            `json:"to"`
	Fields map[string]string `json:"fields,omitempty"`
	Notes  []string          `json:"notes,omitempty"`
}

type UnsupportedDiagnostic struct {
	Semantic string `json:"semantic"`
	Handling string `json:"handling"`
	Reason   string `json:"reason,omitempty"`
}

type GeneratedCandidates struct {
	ProjectPath string `json:"project_path,omitempty"`
	IntentPath  string `json:"intent_path,omitempty"`
	Promoted    bool   `json:"promoted"`
}

type ValidationEvidence struct {
	Status string   `json:"status"`
	Gates  []string `json:"gates,omitempty"`
	Notes  []string `json:"notes,omitempty"`
}

type ValidationResult struct {
	Path    string
	Summary Summary
}

func ValidateFile(path string) (ValidationResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ValidationResult{}, err
	}
	var summary Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		return ValidationResult{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := Validate(summary); err != nil {
		return ValidationResult{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := validateCandidatePaths(path, summary); err != nil {
		return ValidationResult{}, fmt.Errorf("%s: %w", path, err)
	}
	return ValidationResult{Path: filepath.ToSlash(path), Summary: summary}, nil
}

func Validate(summary Summary) error {
	var problems []string
	if summary.Version != SummaryVersion {
		problems = append(problems, fmt.Sprintf("version must be %q", SummaryVersion))
	}
	if strings.TrimSpace(summary.Fixture) == "" {
		problems = append(problems, "fixture is required")
	}
	if summary.Boundary != "authoring_assistance_only" {
		problems = append(problems, `boundary must be "authoring_assistance_only"`)
	}
	if strings.TrimSpace(summary.Source.Kind) == "" {
		problems = append(problems, "source.kind is required")
	}
	if len(summary.Services) == 0 {
		problems = append(problems, "at least one service is required")
	}
	for i, service := range summary.Services {
		if strings.TrimSpace(service.Name) == "" {
			problems = append(problems, fmt.Sprintf("services[%d].name is required", i))
		}
	}
	if len(summary.Nodes) == 0 {
		problems = append(problems, "at least one node is required")
	}
	for i, node := range summary.Nodes {
		if strings.TrimSpace(node.Name) == "" || strings.TrimSpace(node.Type) == "" {
			problems = append(problems, fmt.Sprintf("nodes[%d] requires name and type", i))
		}
		if strings.TrimSpace(node.MappingStatus) == "" {
			problems = append(problems, fmt.Sprintf("nodes[%d].mapping_status is required", i))
		}
	}
	for i, candidate := range summary.OpenAPICandidates {
		if strings.TrimSpace(candidate.Path) == "" || strings.TrimSpace(candidate.OperationID) == "" || strings.TrimSpace(candidate.Status) == "" {
			problems = append(problems, fmt.Sprintf("openapi_candidates[%d] requires path, operation_id, and status", i))
		}
	}
	for i, binding := range summary.CredentialBindings {
		if strings.TrimSpace(binding.Name) == "" || strings.TrimSpace(binding.Service) == "" {
			problems = append(problems, fmt.Sprintf("credential_bindings[%d] requires name and service", i))
			continue
		}
		if authoring.ContainsLikelyCredentialValue([]byte(binding.Name)) {
			problems = append(problems, fmt.Sprintf("credential_bindings[%d].name looks like a credential value", i))
		}
	}
	for i, diagnostic := range summary.UnsupportedSemantics {
		if strings.TrimSpace(diagnostic.Semantic) == "" || strings.TrimSpace(diagnostic.Handling) == "" {
			problems = append(problems, fmt.Sprintf("unsupported_semantics[%d] requires semantic and handling", i))
			continue
		}
		switch diagnostic.Handling {
		case "diagnostic", "todo", "manual_contract", "unsupported":
		default:
			problems = append(problems, fmt.Sprintf("unsupported_semantics[%d].handling must be diagnostic, todo, manual_contract, or unsupported", i))
		}
	}
	if summary.GeneratedCandidates.Promoted && strings.TrimSpace(summary.GeneratedCandidates.IntentPath) == "" {
		problems = append(problems, "promoted generated candidates must name intent_path")
	}
	switch summary.Validation.Status {
	case "advisory", "validated", "blocked":
	default:
		problems = append(problems, "validation.status must be advisory, validated, or blocked")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func validateCandidatePaths(summaryPath string, summary Summary) error {
	fixtureRoot := filepath.Dir(filepath.Dir(summaryPath))
	var problems []string
	for label, rel := range map[string]string{
		"generated_candidates.project_path": summary.GeneratedCandidates.ProjectPath,
		"generated_candidates.intent_path":  summary.GeneratedCandidates.IntentPath,
	} {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		cleanRel, err := cleanFixtureRelativePath(fixtureRoot, rel)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s must stay inside the fixture: %s", label, rel))
			continue
		}
		info, err := os.Lstat(filepath.Join(fixtureRoot, filepath.FromSlash(cleanRel)))
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s does not exist: %s", label, rel))
			continue
		}
		if !info.Mode().IsRegular() {
			problems = append(problems, fmt.Sprintf("%s must be a regular file: %s", label, rel))
		}
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func cleanFixtureRelativePath(fixtureRoot, rel string) (string, error) {
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("absolute path")
	}
	cleanRel := filepath.Clean(filepath.FromSlash(rel))
	if cleanRel == "." {
		return "", fmt.Errorf("empty path")
	}
	rootAbs, err := filepath.Abs(fixtureRoot)
	if err != nil {
		return "", err
	}
	candidateAbs, err := filepath.Abs(filepath.Join(fixtureRoot, cleanRel))
	if err != nil {
		return "", err
	}
	inside, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil {
		return "", err
	}
	if inside == "." || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("outside fixture")
	}
	return filepath.ToSlash(cleanRel), nil
}

func ValidateRoot(root string) ([]ValidationResult, error) {
	files, err := FindSummaries(root)
	if err != nil {
		return nil, err
	}
	results := make([]ValidationResult, 0, len(files))
	var problems []string
	for _, file := range files {
		result, err := ValidateFile(file)
		if err != nil {
			problems = append(problems, err.Error())
			continue
		}
		results = append(results, result)
	}
	if len(problems) > 0 {
		return results, errors.New(strings.Join(problems, "\n"))
	}
	return results, nil
}

func FindSummaries(root string) ([]string, error) {
	var files []string
	if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if filepath.Base(path) == "n8n-bridge.json" && filepath.Base(filepath.Dir(path)) == "reference" {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}
