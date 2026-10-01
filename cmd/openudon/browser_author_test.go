package main

import (
	"strings"
	"testing"
)

func TestCLIBrowserAuthorDispatchAndClosedOptions(t *testing.T) {
	cmd := helperCommand("browser-author", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "browser-author <plan|apply>") {
		t.Fatalf("missing native dispatch: %v %s", err, output)
	}
	for _, args := range [][]string{
		{"browser-author", "apply", "--example", ".", "--request", "-"},
		{"browser-author", "plan", "--example", ".", "--request", "-", "--credential", "sentinel-secret"},
		{"browser-author", "plan", "--example", ".", "--request", "-", "--example=other"},
	} {
		cmd := helperCommand(args...)
		output, err := cmd.CombinedOutput()
		if err == nil || strings.Contains(string(output), "sentinel-secret") || !strings.Contains(string(output), "browser author") {
			t.Fatalf("invalid real dispatch accepted or echoed: %v %s", err, output)
		}
	}
}
