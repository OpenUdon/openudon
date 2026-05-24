package synthesize

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/openapidisco"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/uws1"
)

func assessSecrets(report *QualityReport, result Result) {
	set, err := packageInputArtifactSet(result)
	if err != nil {
		report.add("artifacts.no_secrets", "fail", "package artifacts could not be scanned for secrets", err.Error())
		return
	}
	var hits []string
	for _, diagnostic := range authoring.ScanCredentialValues(set.Artifacts) {
		hits = append(hits, diagnostic.Path)
	}
	if len(hits) > 0 {
		report.add("artifacts.no_secrets", "fail", "artifacts contain secret-like tokens", strings.Join(hits, ", "))
		return
	}
	report.add("artifacts.no_secrets", "pass", "no obvious secret-like tokens found in artifacts", "")
}

func packageInputArtifactSet(result Result) (authoring.ArtifactSet, error) {
	paths, err := packageartifacts.RequiredPackagePaths(result.ExampleDir)
	if err != nil {
		return authoring.ArtifactSet{}, err
	}
	artifacts := make([]authoring.Artifact, 0, len(paths))
	for _, rel := range paths {
		path := filepath.Join(result.ExampleDir, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil {
			if os.IsNotExist(err) {
				if missingPackageInputAllowedDuringAssessment(rel) {
					continue
				}
				return authoring.ArtifactSet{}, fmt.Errorf("required handoff input is missing: %s", rel)
			}
			return authoring.ArtifactSet{}, fmt.Errorf("%s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return authoring.ArtifactSet{}, fmt.Errorf("required handoff input must not be a symlink: %s", rel)
		}
		if info.IsDir() {
			return authoring.ArtifactSet{}, fmt.Errorf("required handoff input must be a regular file, not a directory: %s", rel)
		}
		if !info.Mode().IsRegular() {
			return authoring.ArtifactSet{}, fmt.Errorf("required handoff input must be a regular file: %s", rel)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return authoring.ArtifactSet{}, fmt.Errorf("%s: %w", rel, err)
		}
		artifacts = append(artifacts, authoring.Artifact{
			Path:      rel,
			MediaType: reviewArtifactMediaType(path),
			Content:   content,
		})
	}
	return authoring.ArtifactSet{Artifacts: artifacts}, nil
}

func missingPackageInputAllowedDuringAssessment(rel string) bool {
	switch strings.TrimSpace(filepath.ToSlash(rel)) {
	case "expected/quality.json", "expected/refinement.json":
		return true
	default:
		return false
	}
}

func writeQualityFiles(result Result, report *QualityReport) error {
	if err := os.MkdirAll(filepath.Dir(result.QualityJSONPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(result.QualityJSONPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.WriteFile(result.QualityMDPath, []byte(qualityMarkdown(report)), 0o644)
}

func qualityMarkdown(report *QualityReport) string {
	var b strings.Builder
	b.WriteString("# OpenUdon Quality Report\n\n")
	fmt.Fprintf(&b, "Status: `%s`\n\n", report.Status)
	for _, check := range report.Checks {
		fmt.Fprintf(&b, "- `%s` %s - %s\n", check.Code, check.Status, check.Message)
		if check.Detail != "" {
			fmt.Fprintf(&b, "  Detail: %s\n", check.Detail)
		}
	}
	return b.String()
}

func missingIntentSteps(intent *rollout.Intent, workflows []*uws1.Workflow) []string {
	stepIDs := map[string]bool{}
	for _, workflow := range workflows {
		if workflow != nil {
			collectUWSStepIDs(workflow.Steps, stepIDs)
		}
	}
	var missing []string
	walkIntentSteps(intent.Steps, func(step *rollout.Step) {
		if step == nil {
			return
		}
		name := strings.TrimSpace(step.Name)
		if name != "" && !stepIDs[name] {
			missing = append(missing, name)
		}
	})
	sort.Strings(missing)
	return missing
}

func collectUWSStepIDs(steps []*uws1.Step, out map[string]bool) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if strings.TrimSpace(step.StepID) != "" {
			out[strings.TrimSpace(step.StepID)] = true
		}
		collectUWSStepIDs(step.Steps, out)
		for _, branch := range step.Cases {
			if branch != nil {
				collectUWSStepIDs(branch.Steps, out)
			}
		}
		collectUWSStepIDs(step.Default, out)
	}
}

func candidateList(candidates []openapidisco.Candidate) string {
	var items []string
	for _, candidate := range candidates {
		items = append(items, candidate.RelativePath)
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func walkIntentSteps(steps []*rollout.Step, fn func(*rollout.Step)) {
	for _, step := range steps {
		if step == nil {
			continue
		}
		fn(step)
		walkIntentSteps(step.Steps, fn)
		for _, branch := range step.Cases {
			if branch != nil {
				walkIntentSteps(branch.Steps, fn)
			}
		}
		if step.Default != nil {
			walkIntentSteps(step.Default.Steps, fn)
		}
	}
}

func (r *QualityReport) add(code, status, message, detail string) {
	r.Checks = append(r.Checks, QualityCheck{
		Code:        code,
		Status:      status,
		Message:     message,
		Detail:      detail,
		FailureKind: classifyQualityFailureKind(code, status, detail),
	})
}

const (
	QualityFailureArtifact       = "artifact"
	QualityFailureInfrastructure = "infrastructure"
)

func classifyQualityFailureKind(code, status, detail string) string {
	if status != "fail" {
		return ""
	}
	if qualityFailureLooksInfrastructure(code, detail) {
		return QualityFailureInfrastructure
	}
	return QualityFailureArtifact
}

func qualityFailureLooksInfrastructure(code, detail string) bool {
	lower := strings.ToLower(strings.TrimSpace(detail))
	if lower == "" {
		return false
	}
	for _, marker := range []string{
		"permission denied",
		"operation not permitted",
		"read-only file system",
		"input/output error",
		"i/o error",
		"too many open files",
		"no space left on device",
		"resource temporarily unavailable",
		"stale file handle",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	switch strings.TrimSpace(code) {
	case "openapi.local", "review_handoff.contract":
		return strings.Contains(lower, "could not be checked") || strings.Contains(lower, "could not be scanned")
	default:
		return false
	}
}

func (r *QualityReport) finalize() {
	for _, check := range r.Checks {
		if check.Status == "fail" {
			r.Status = "fail"
			return
		}
	}
	r.Status = "pass"
}
