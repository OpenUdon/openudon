package main

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestBrowserSystemInputCLIChild(t *testing.T) {
	if os.Getenv("OPENUDON_INPUT_CLI_FIXTURE") == "" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			runBrowserSystemInput(os.Args[i+1:])
			return
		}
	}
	t.Fatal("child arguments")
}

func TestBrowserSystemInputCLIRefusesUnsupportedFormsBeforeInventory(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{nil, {"--repo-root", "/abs"}, {"--repo-root", "/abs", "--udon-repo", "/abs", "extra"}, {"--suite", "loopback"}, {"--repo-root", "private-input-canary", "--udon-repo", "/abs"}} {
		cmd := exec.Command(self, append([]string{"-test.run=^TestBrowserSystemInputCLIChild$", "--"}, args...)...)
		cmd.Env = []string{"OPENUDON_INPUT_CLI_FIXTURE=1"}
		output, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatal("unsupported input CLI accepted", args)
		}
		if strings.Contains(string(output), "private-input-canary") {
			t.Fatal("input path escaped in error")
		}
	}
}
