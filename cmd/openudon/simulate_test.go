package main

import (
	"encoding/json"
	"os/exec"
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
