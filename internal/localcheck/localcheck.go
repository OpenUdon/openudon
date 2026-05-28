package localcheck

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	blockedAPIToolsImports   = regexp.MustCompile(`github\.com/OpenUdon/apitools/(llm|icot|context7)`)
	blockedAPIToolsSymbols   = regexp.MustCompile(`apitools\.(Artifact(Set)?|Assumption|Binding(Contract|Field|Ref)|BuildBindingContract|BuildReviewPackage|ChatClient|CompleteJSONWithFallback|ComputeReviewHandoffDigest|ContainsLikelyCredentialValue|Documentation(Context|Snippet)|Draft|Flow|Interactive|JSONCompletion|Leaf(Adapter|Options)|NewLeafAdapter|Question(Plan)?|Review(Handoff|State|Package|OwnerSplit|ExecutionPolicy|CredentialBindings|TrustedRunner)|Slot|SymbolicBinding|Transcript|ValidateReviewHandoff)`)
	blockedExecutorImports   = regexp.MustCompile(`github\.com/(OpenUdon/udon|genelet/(udon|cmd|dns|fileio|fnct|ldaps|llm|s3|scp|sftp|smtp|sql|ssh))(/[^"]*)?`)
	blockedTerraformImports  = regexp.MustCompile(`github\.com/(opentofu/opentofu|hashicorp/terraform|OpenUdon/tfconfig)(/[^"]*)?`)
	staleDocReferencePattern = regexp.MustCompile(`ICOT\.md|SYMPHONY_WRAPPER\.md|WORKFLOW\.md|openudon\.md|TODO\.md|migrate\.md`)
)

var RequiredMemoryFiles = []string{
	"memory-bank/product.md",
	"memory-bank/architecture.md",
	"memory-bank/tech-stack.md",
	"memory-bank/milestone.md",
	"evolution/prompt-v1.md",
	"evolution/result-v1.md",
}

type DocMemoryResult struct {
	CheckedFiles []string
	Warnings     []string
}

func CheckAPIToolsBoundary(root string) error {
	hits, err := scanFiles(root, func(rel string, data []byte) (string, bool) {
		if filepath.Ext(rel) != ".go" || strings.HasSuffix(rel, "_test.go") {
			return "", false
		}
		text := string(data)
		switch {
		case blockedAPIToolsImports.MatchString(text):
			return formatHit(rel, data, blockedAPIToolsImports), true
		case blockedAPIToolsSymbols.MatchString(text):
			return formatHit(rel, data, blockedAPIToolsSymbols), true
		case blockedExecutorImports.MatchString(text):
			return formatHit(rel, data, blockedExecutorImports), true
		case blockedTerraformImports.MatchString(text):
			return formatHit(rel, data, blockedTerraformImports), true
		default:
			return "", false
		}
	})
	if err != nil {
		return err
	}
	if len(hits) > 0 {
		sort.Strings(hits)
		return fmt.Errorf("repository boundary violation found:\n%s", strings.Join(hits, "\n"))
	}
	return nil
}

func CheckDocMemory(root string) (DocMemoryResult, error) {
	var result DocMemoryResult
	for _, file := range RequiredMemoryFiles {
		path := filepath.Join(root, filepath.FromSlash(file))
		info, err := os.Stat(path)
		if err != nil {
			return result, fmt.Errorf("missing: %s", file)
		}
		if info.IsDir() {
			return result, fmt.Errorf("missing file: %s is a directory", file)
		}
		result.CheckedFiles = append(result.CheckedFiles, file)
	}

	hits, err := scanFiles(root, func(rel string, data []byte) (string, bool) {
		if shouldSkipDocMemoryFile(rel) {
			return "", false
		}
		if staleDocReferencePattern.Match(data) {
			return formatHit(rel, data, staleDocReferencePattern), true
		}
		return "", false
	})
	if err != nil {
		return result, err
	}
	if len(hits) > 0 {
		sort.Strings(hits)
		return result, fmt.Errorf("stale removed-doc references found:\n%s", strings.Join(hits, "\n"))
	}

	if changedMilestoneWithoutEvolution(root) {
		result.Warnings = append(result.Warnings, "memory-bank/milestone.md changed without evolution/ changes; confirm no new evolution version is needed")
	}
	return result, nil
}

func scanFiles(root string, match func(rel string, data []byte) (string, bool)) ([]string, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rels, ok, err := gitCandidateFiles(rootAbs)
	if err != nil {
		return nil, err
	}
	if ok {
		return scanListedFiles(rootAbs, rels, match)
	}
	return scanWalkedFiles(rootAbs, match)
}

func gitCandidateFiles(root string) ([]string, bool, error) {
	rels, err := gitLines(root, "ls-files", "--cached", "--others", "--exclude-standard", "--")
	if err != nil {
		return nil, false, nil
	}
	sort.Strings(rels)
	return rels, true, nil
}

func scanListedFiles(rootAbs string, rels []string, match func(rel string, data []byte) (string, bool)) ([]string, error) {
	var hits []string
	for _, rel := range rels {
		path := filepath.Join(rootAbs, filepath.FromSlash(rel))
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if hit, ok := match(filepath.ToSlash(rel), data); ok {
			hits = append(hits, hit)
		}
	}
	return hits, nil
}

func scanWalkedFiles(rootAbs string, match func(rel string, data []byte) (string, bool)) ([]string, error) {
	var hits []string
	err := filepath.WalkDir(rootAbs, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := entry.Name()
		if entry.IsDir() {
			if name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if hit, ok := match(rel, data); ok {
			hits = append(hits, hit)
		}
		return nil
	})
	return hits, err
}

func shouldSkipDocMemoryFile(rel string) bool {
	parts := strings.Split(rel, "/")
	for _, part := range parts[:len(parts)-1] {
		switch part {
		case "readiness", "runs", "artifacts":
			return true
		}
	}
	return false
}

func formatHit(rel string, data []byte, pattern *regexp.Regexp) string {
	lines := bytes.Split(data, []byte{'\n'})
	for i, line := range lines {
		if pattern.Match(line) {
			return fmt.Sprintf("%s:%d:%s", rel, i+1, string(line))
		}
	}
	return rel
}

func changedMilestoneWithoutEvolution(root string) bool {
	changed, err := gitLines(root, "diff", "--name-only", "HEAD", "--")
	if err != nil {
		return false
	}
	untracked, _ := gitLines(root, "ls-files", "--others", "--exclude-standard", "evolution")
	hasMilestone := false
	hasEvolution := false
	for _, path := range append(changed, untracked...) {
		switch {
		case path == "memory-bank/milestone.md":
			hasMilestone = true
		case strings.HasPrefix(path, "evolution/"):
			hasEvolution = true
		}
	}
	return hasMilestone && !hasEvolution
}

func gitLines(root string, args ...string) ([]string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}
