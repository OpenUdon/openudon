package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenUdon/openudon/internal/stepauthoring"
)

func TestPendingCLIUsesDedicatedStrictEnvelope(t *testing.T) {
	root := t.TempDir()
	fixture := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "requests", "step-bind.json")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var bind stepauthoring.BindRequest
	if err := json.Unmarshal(data, &bind); err != nil {
		t.Fatal(err)
	}
	request := stepauthoring.PendingRequest{Version: stepauthoring.PendingWireVersion, Kind: "request", Command: stepauthoring.PendingCommand, StepID: bind.StepID, Contract: bind.Contract, IntentRevision: stepauthoring.IntentRevision{State: "missing"}, Scaffold: bind.Scaffold}
	data, err = json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := runStepPendingCommand([]string{"--example", root, "--request", "-"}, bytes.NewReader(data), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pending command=%d %s %s", code, &stdout, &stderr)
	}
	var result stepauthoring.BindWireResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Version != stepauthoring.PendingWireVersion || result.Command != stepauthoring.PendingCommand || result.Result == nil {
		t.Fatalf("wrong result: %+v", result)
	}
	data = append(data[:len(data)-1], []byte(`,"operation_ref":{"guessed":"endpoint"}}`)...)
	stdout.Reset()
	stderr.Reset()
	if code := runStepPendingCommand([]string{"--example", t.TempDir(), "--request", "-"}, bytes.NewReader(data), &stdout, &stderr); code != 2 {
		t.Fatalf("unknown request fields accepted=%d %s", code, &stdout)
	}
}
