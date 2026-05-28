package elicitor

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	evalpkg "github.com/OpenUdon/openudon/internal/eval"
	"github.com/OpenUdon/openudon/internal/synthesize"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestReplayEvalReferencesThroughICOTChat(t *testing.T) {
	root := filepath.Join("..", "..", "..", "examples", "eval")
	fixtures, err := filepath.Glob(filepath.Join(root, "*", "reference", "intent.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	if len(fixtures) != 63 {
		t.Fatalf("fixture count = %d, want 63", len(fixtures))
	}
	for _, referencePath := range fixtures {
		exampleDir := filepath.Dir(filepath.Dir(referencePath))
		name := filepath.Base(exampleDir)
		t.Run(name, func(t *testing.T) {
			before := hashDir(t, exampleDir)
			reference, err := rollout.ParseIntentFile(referencePath)
			if err != nil {
				t.Fatalf("parse reference intent: %v", err)
			}
			script, err := BuildReplayScript(exampleDir, reference)
			if err != nil {
				t.Fatalf("build replay script: %v", err)
			}
			var stdout strings.Builder
			artifacts, err := Run(context.Background(), strings.NewReader(script.Input), &stdout, Session{}, Options{
				ExampleDir: exampleDir,
				NoLLM:      true,
				Extractor:  NewNoopExtractor(),
			})
			if err != nil {
				t.Fatalf("replay failed: %v\nstdout:\n%s\ninput:\n%s", err, stdout.String(), script.Input)
			}
			if err := AssertReplayLabelsInOrder(stdout.String(), script.Turns); err != nil {
				t.Fatalf("%v\noutput:\n%s", err, stdout.String())
			}
			if got := hashDir(t, exampleDir); got != before {
				t.Fatalf("replay wrote to fixture directory")
			}
			generatedPath := filepath.Join(t.TempDir(), "intent.hcl")
			if err := os.WriteFile(generatedPath, []byte(artifacts.IntentHCL), 0o644); err != nil {
				t.Fatalf("write generated intent: %v", err)
			}
			policy, _ := evalpkg.ReadReferencePolicy(filepath.Join(exampleDir, "reference", "policy.json"))
			issues, err := evalpkg.CompareIntentFiles(generatedPath, referencePath, policy)
			if err != nil {
				t.Fatalf("compare intent: %v\n%s", err, artifacts.IntentHCL)
			}
			for _, issue := range issues {
				if issue.Severity == "blocking" {
					t.Fatalf("blocking intent drift: %#v\nrendered:\n%s", issues, artifacts.IntentHCL)
				}
			}
			for _, check := range synthesize.LintProjectMarkdown(artifacts.ProjectMD) {
				if check.Status == "fail" {
					t.Fatalf("rendered project.md failed lint: %#v\n%s", check, artifacts.ProjectMD)
				}
			}
		})
	}
}

func hashDir(t *testing.T, root string) string {
	t.Helper()
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hash.Write([]byte(filepath.ToSlash(rel)))
		hash.Write([]byte{0})
		hash.Write(data)
		hash.Write([]byte{0})
		return nil
	})
	if err != nil {
		t.Fatalf("hash %s: %v", root, err)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
