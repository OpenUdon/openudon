package processgroup

import (
	"bytes"
	"context"
	"errors"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

const environmentSentinel = "OPENUDON_PROCESSGROUP_UNRELATED_SENTINEL"

func TestRunCapturesOutput(t *testing.T) {
	var output bytes.Buffer
	args := []string{"sh", "-c", "printf ready"}
	if runtime.GOOS == "windows" {
		args = []string{"cmd", "/c", "echo|set /p=ready"}
	}
	err := Run(context.Background(), time.Second, Invocation{Args: args, Stdout: &output, Env: os.Environ()})
	if err != nil || output.String() != "ready" {
		t.Fatalf("Run() = %q, %v", output.String(), err)
	}
}

func TestRunEnforcesDeadline(t *testing.T) {
	args := []string{"sh", "-c", "sleep 30"}
	if runtime.GOOS == "windows" {
		args = []string{"cmd", "/c", "ping -n 30 127.0.0.1 >NUL"}
	}
	started := time.Now()
	err := Run(context.Background(), 50*time.Millisecond, Invocation{Args: args, Env: os.Environ()})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v", err)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("deadline termination took %s", elapsed)
	}
}

func TestRunDoesNotInheritEnvironmentForEmptyAllowlist(t *testing.T) {
	t.Setenv(environmentSentinel, "must-not-pass")
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	// Starting a second race-instrumented test binary can take longer than one
	// second on loaded builders. Keep the helper bounded without making the
	// environment-isolation assertion depend on startup speed.
	err = Run(context.Background(), 5*time.Second, Invocation{
		Args: []string{executable, "-test.run=^TestProcessgroupEnvironmentHelper$", "--", "processgroup-env-helper"},
		Env:  nil, Stdout: &output,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output.String()) != "absent" {
		t.Fatalf("empty allowlist inherited parent environment: %q", output.String())
	}
}

func TestRunResolvesCommandFromInvocationPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix executable fixture")
	}
	directory := t.TempDir()
	tool := directory + string(os.PathSeparator) + "invocation-path-tool"
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nprintf invocation-path\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	err := Run(context.Background(), time.Second, Invocation{
		Args: []string{"invocation-path-tool"}, Env: []string{"PATH=" + directory}, Stdout: &output,
	})
	if err != nil || strings.TrimSpace(output.String()) != "invocation-path" {
		t.Fatalf("invocation PATH command = %q, %v", output.String(), err)
	}
}

func TestProcessgroupEnvironmentHelper(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "processgroup-env-helper" {
		return
	}
	if os.Getenv(environmentSentinel) == "" {
		_, _ = os.Stdout.WriteString("absent\n")
	} else {
		_, _ = os.Stdout.WriteString("leaked\n")
	}
	os.Exit(0)
}
