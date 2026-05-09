package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	rollout "github.com/genelet/ramen/internal/workflowintent"
	runner "github.com/genelet/ramen/internal/workflowintent"
)

func TestIntentDocHCLExamplesParse(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "docs", "intent.md"))
	if err != nil {
		t.Fatal(err)
	}
	examples := fencedBlocks(string(data), "hcl")
	if len(examples) == 0 {
		t.Fatal("docs/intent.md has no hcl examples")
	}
	for i, example := range examples {
		intent, err := rollout.ParseIntent([]byte(example), "docs/intent.md")
		if err != nil {
			t.Fatalf("parse docs/intent.md hcl example %d: %v\n%s", i+1, err, example)
		}
		if _, err := runner.RenderIntentHCL(intent); err != nil {
			t.Fatalf("render docs/intent.md hcl example %d: %v\n%s", i+1, err, example)
		}
	}
}

func fencedBlocks(text, language string) []string {
	var blocks []string
	var current strings.Builder
	inBlock := false
	language = strings.ToLower(strings.TrimSpace(language))
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if !inBlock {
			if strings.EqualFold(strings.TrimPrefix(trimmed, "```"), language) && strings.HasPrefix(trimmed, "```") {
				inBlock = true
				current.Reset()
			}
			continue
		}
		if strings.HasPrefix(trimmed, "```") {
			blocks = append(blocks, current.String())
			inBlock = false
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	return blocks
}
