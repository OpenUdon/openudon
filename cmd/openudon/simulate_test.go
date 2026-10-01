package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/simulation"
)

func TestSimulateTopLevelDispatchRefusalAndHelp(t *testing.T) {
	for _, args := range [][]string{{"simulate"}, {"simulate", "--private-canary"}, {"simulate", "--example", "PRIVATE-PATH-CANARY"}, {"simulate", "--example", "missing", "--input", "PRIVATE-INPUT-CANARY"}} {
		cmd := helperCommand(args...)
		output, err := cmd.CombinedOutput()
		if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
			t.Fatalf("expected bounded refusal, got %v %s", err, output)
		}
		var report simulation.Report
		if json.Unmarshal(output, &report) != nil || report.Version != simulation.Version || report.Status != "blocked" || strings.Contains(strings.ToLower(string(output)), "canary") {
			t.Fatalf("invalid/unsafe top-level simulation refusal: %s", output)
		}
	}
	cmd := helperCommand("simulate", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "example") || !strings.Contains(string(output), "fixtures") {
		t.Fatalf("simulation help unavailable: %v %s", err, output)
	}
}

func TestSimulateCLIUsesPublishedPendingPackageWithoutChangingIt(t *testing.T) {
	repo := t.TempDir()
	relative := filepath.Join("docs", "examples", "simulation", "v1", "example")
	source := filepath.Join("..", "..", relative)
	destination := filepath.Join(repo, relative)
	before := map[string][]byte{}
	if err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		before[name] = data
		mustWriteCLIFile(t, filepath.Join(destination, name), data)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"simulate", "--example", relative}, {"approval-template", "--example", relative, "--state", "approved_for_sandbox", "--reviewer", "Fixture"}, {"run", "--example", relative, "--tier", "sandbox", "--approval", "missing.json", "--dry-run"}} {
		cmd := helperCommand(args...)
		cmd.Dir = repo
		output, err := cmd.CombinedOutput()
		if args[0] == "simulate" {
			if err != nil {
				t.Fatalf("pending CLI simulation failed: %v %s", err, output)
			}
			var report simulation.Report
			if json.Unmarshal(output, &report) != nil || report.Status != "completed" || !report.Hypothetical || !report.PackageUnchanged || report.Steps[0].Binding != "pending" {
				t.Fatalf("invalid pending simulation CLI: %s", output)
			}
		} else if err == nil {
			t.Fatalf("pending package accepted by %s: %s", args[0], output)
		}
	}
	for name, data := range before {
		after, err := os.ReadFile(filepath.Join(destination, name))
		if err != nil || !bytes.Equal(data, after) {
			t.Fatal("CLI preview/refusal changed package:", name)
		}
	}
	var afterCount int
	if err := filepath.WalkDir(destination, func(_ string, entry os.DirEntry, err error) error {
		if !entry.IsDir() {
			afterCount++
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if afterCount != len(before) {
		t.Fatal("CLI refusal created package artifacts")
	}
}
