package authoringcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type forbiddenDraftInput struct{}

func (forbiddenDraftInput) Read([]byte) (int, error) { panic("neutral draft read terminal input") }

func TestNeutralDraftRefusesUnreviewedPublicationAndReturnsIncompleteFrontier(t *testing.T) {
	for _, args := range [][]string{{"--example", "DEST"}, {"--example", "DEST", "--yes", "--network", "ask"}, {"--example", "DEST", "--agent", "extra"}, {"--example", "DEST", "--yes", "--review-repair"}, {"--example", "DEST", "--yes", "--provider", "fake"}} {
		base := t.TempDir()
		dst := filepath.Join(base, "example")
		var out, errOut bytes.Buffer
		args = append([]string(nil), args...)
		for i, v := range args {
			if v == "DEST" {
				args[i] = dst
			}
		}
		if RunExpert(append([]string{"draft"}, args...), forbiddenDraftInput{}, &out, &errOut) != 2 {
			t.Fatal("missing noninteractive authority refused", out.String(), errOut.String())
		}
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			t.Fatal("refusal wrote state", err)
		}
	}
	dst := filepath.Join(t.TempDir(), "example")
	var out, errOut bytes.Buffer
	if code := RunExpert([]string{"draft", "--example", dst, "--yes", "--no-llm", "--json"}, forbiddenDraftInput{}, &out, &errOut); code != 0 {
		t.Fatalf("frontier code=%d err=%s", code, errOut.String())
	}
	var r authorReport
	if json.Unmarshal(out.Bytes(), &r) != nil || r.Status != statusNeedsInput || r.TopIssue == nil {
		t.Fatal("incomplete seed lost frontier", out.String())
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("incomplete seed published state", err)
	}
}

func TestNeutralDraftPreservesPrintSeedAndExplicitPublication(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "example")
	seed := filepath.Join("..", "..", "examples", "eval", "runtime-only-render")
	var out, errOut bytes.Buffer
	args := []string{"draft", "--example", dst, "--from-example", seed, "--no-llm", "--no-transcript", "--prompt-mode", "fast"}
	if code := RunExpert(append(append([]string(nil), args...), "--print"), forbiddenDraftInput{}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "workflow") {
		t.Fatalf("print code=%d out=%s err=%s", code, out.String(), errOut.String())
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("print wrote state", err)
	}
	out.Reset()
	errOut.Reset()
	if code := RunExpert(append(args, "--yes"), forbiddenDraftInput{}, &out, &errOut); code != 0 {
		t.Fatalf("publication code=%d err=%s", code, errOut.String())
	}
	for _, name := range []string{"project.md", "workflows/intent.hcl"} {
		if _, err := os.Stat(filepath.Join(dst, name)); err != nil {
			t.Fatal("complete seeded publication missing", name, err)
		}
	}
}

func TestNeutralDraftHelpNamesOnlyCurrentEntry(t *testing.T) {
	var out, errOut bytes.Buffer
	if RunExpert([]string{"draft", "--help"}, forbiddenDraftInput{}, &out, &errOut) != 0 || !strings.Contains(out.String(), "openudon authoring draft") || strings.Contains(out.String(), "icot ui") {
		t.Fatal("help still selects retired transport", out.String(), errOut.String())
	}
}

func TestNeutralDraftRetainsFastSeedCorpusWithoutTerminalOrPrintWrites(t *testing.T) {
	for _, name := range []string{"browser-status-read", "timeout-idempotency-controls"} {
		dst := filepath.Join(t.TempDir(), "example")
		var out, errOut bytes.Buffer
		if code := RunExpert([]string{"draft", "--example", dst, "--from-example", filepath.Join("..", "..", "examples", "eval", name), "--prompt-mode", "fast", "--print", "--no-llm"}, forbiddenDraftInput{}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "----- workflows/intent.hcl -----") {
			t.Fatalf("%s code=%d out=%s err=%s", name, code, out.String(), errOut.String())
		}
		if _, err := os.Stat(dst); !os.IsNotExist(err) {
			t.Fatal("seed default normalization wrote state", err)
		}
	}
}

func TestNeutralDraftDoesNotClaimUnwrittenTerminalTranscript(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "example")
	var out, errOut bytes.Buffer
	if code := RunExpert([]string{"draft", "--example", dst, "--from-example", filepath.Join("..", "..", "examples", "eval", "runtime-only-render"), "--prompt-mode", "fast", "--yes", "--no-llm"}, forbiddenDraftInput{}, &out, &errOut); code != 0 {
		t.Fatalf("code=%d err=%s", code, errOut.String())
	}
	if strings.Contains(out.String(), "transcript.json") {
		t.Fatal("claimed terminal transcript", out.String())
	}
	if _, err := os.Stat(filepath.Join(dst, ".icot", "transcript.json")); !os.IsNotExist(err) {
		t.Fatal("fabricated terminal transcript", err)
	}
}

func TestNeutralReconcileNeverReadsTerminalAndRequiresPublicationApproval(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "example")
	writeCompleteDraftWithPolicy(t, dst, []string{"sample_token"}, "Sample safety", "Sample fallback")
	var out, errOut bytes.Buffer
	if code := RunExpert([]string{"draft", "--example", dst, "--yes", "--no-llm"}, forbiddenDraftInput{}, &out, &errOut); code != 0 {
		t.Fatalf("seed code=%d err=%s", code, errOut.String())
	}
	before, err := os.ReadFile(filepath.Join(dst, "project.md"))
	if err != nil {
		t.Fatal(err)
	}
	out.Reset()
	errOut.Reset()
	if code := RunExpert([]string{"reconcile", "--example", dst}, forbiddenDraftInput{}, &out, &errOut); code != 2 {
		t.Fatalf("unapproved code=%d err=%s", code, errOut.String())
	}
	after, err := os.ReadFile(filepath.Join(dst, "project.md"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("unapproved reconcile changed project", err)
	}
	if code := RunExpert([]string{"reconcile", "--example", dst, "--yes"}, forbiddenDraftInput{}, &out, &errOut); code != 0 {
		t.Fatalf("approved code=%d err=%s", code, errOut.String())
	}
}

func TestNeutralDraftPrintConflictsRefuseBeforeAnyWrites(t *testing.T) {
	for _, extra := range [][]string{{"--report", "report.json"}, {"--report", "report.json", "--yes"}} {
		t.Run(strings.Join(extra, "_"), func(t *testing.T) {
			root := t.TempDir()
			target := filepath.Join(root, "package")
			report := filepath.Join(root, "report.json")
			args := []string{"--print", "--example", target, "--from-example", filepath.Join(root, "missing-source")}
			for _, arg := range extra {
				if arg == "report.json" {
					arg = report
				}
				args = append(args, arg)
			}
			var out, errOut bytes.Buffer
			if code := RunDraft(args, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "--print cannot be combined") {
				t.Fatalf("print conflict = %d / %s", code, errOut.String())
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("print conflict wrote files: %v %v", entries, err)
			}
		})
	}
}

func TestNeutralDraftAgentPrintRetainsReadOnlyFrontier(t *testing.T) {
	for _, mode := range []struct{ complete, yes bool }{{false, false}, {true, false}, {false, true}, {true, true}} {
		root := t.TempDir()
		target := filepath.Join(root, "package")
		args := []string{"--example", target, "--agent", "--print", "--json"}
		if mode.yes {
			args = append(args, "--yes")
		}
		if mode.complete {
			args = append(args, "--answers", writeCompleteRuntimeSession(t, root))
		}
		var out, errOut bytes.Buffer
		if code := RunDraft(args, &out, &errOut); code != 0 {
			t.Fatalf("agent print = %d / %s", code, errOut.String())
		}
		var report authorReport
		if err := json.Unmarshal(out.Bytes(), &report); err != nil || report.Status != statusNeedsInput {
			t.Fatalf("frontier lost: %v / %s", err, out.String())
		}
		if _, err := os.Stat(target); !os.IsNotExist(err) {
			t.Fatalf("agent print wrote package: %v", err)
		}
	}
}
