package udonrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExplicitHumanInputReachesOwnedExecutorWithoutEnteringArtifacts(t *testing.T) {
	for _, interactive := range []bool{false, true} {
		config := validRunnerConfig(t)
		executable := filepath.Join(t.TempDir(), "fixture-executor")
		script := []byte("#!/usr/bin/bash\nIFS= read -r response || exit 41\n[[ $response == fixture-human-response ]] || exit 42\nprintf 'accepted\\n'\n")
		if os.WriteFile(executable, script, 0700) != nil {
			t.Fatal("fixture")
		}
		var input io.Reader
		if interactive {
			input = strings.NewReader("fixture-human-response\n")
		}
		var output bytes.Buffer
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, err := Run(ctx, config, Options{RepoRoot: t.TempDir(), Env: []string{"OPENUDON_EXECUTOR=" + executable, "PATH=/usr/bin:/bin"}, Stdin: input, Stdout: &output, Stderr: io.Discard})
		cancel()
		encoded, _ := json.Marshal(result)
		if bytes.Contains(encoded, []byte("fixture-human-response")) {
			t.Fatal("human response entered invocation artifacts")
		}
		if interactive {
			if err != nil || output.String() != "accepted\n" {
				t.Fatalf("interactive executor = %v", err)
			}
		} else if err == nil {
			t.Fatal("noninteractive invocation supplied a response")
		}
	}
}

func TestExecutorPersistsRelativeArtifactsInsideItsExactStage(t *testing.T) {
	config := validRunnerConfig(t)
	repository := t.TempDir()
	executable := filepath.Join(t.TempDir(), "fixture-executor")
	// A real process verifies cwd against the actual --workdir argument, then
	// writes the relative output used by the inherited local file backend.
	script := []byte("#!/usr/bin/bash\n[[ $1 == --workdir && $PWD == \"$2\" ]] || exit 41\nmkdir output\nprintf 'fixture-result\\n' > output/udon.hcl\n")
	if err := os.WriteFile(executable, script, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := Run(ctx, config, Options{RepoRoot: repository, Env: []string{"OPENUDON_EXECUTOR=" + executable, "PATH=/usr/bin:/bin"}, Stdout: io.Discard, Stderr: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(result.StagePath, "output", "udon.hcl"))
	if err != nil || string(data) != "fixture-result\n" {
		t.Fatalf("stage output: %v", err)
	}
	entries, err := os.ReadDir(repository)
	if err != nil || len(entries) != 0 {
		t.Fatal("executor wrote into package repository")
	}
}
