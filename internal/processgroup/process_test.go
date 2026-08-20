package processgroup

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
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

func TestInteractiveOutputIsDrainedBeforeWaitClosesPipe(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exitedMarker := filepath.Join(t.TempDir(), "helper-exited")
	child, err := StartInteractive(context.Background(), []string{
		executable, "-test.run=^TestInteractiveOutputHelper$", "--", exitedMarker, "interactive-output-helper",
	}, nil, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	_ = child.Input().Close()
	// Wait until the helper has written its complete line and is exiting before
	// draining. Starting Cmd.Wait in StartInteractive used to close this pipe
	// and lose the buffered line.
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(exitedMarker); err == nil {
			break
		} else if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if time.Now().After(deadline) {
			t.Fatal("interactive output helper did not reach exit marker")
		}
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond)
	output, err := io.ReadAll(child.Output())
	if err != nil {
		t.Fatalf("read interactive output: %v", err)
	}
	if err := child.Wait(); err != nil {
		t.Fatalf("wait interactive child: %v", err)
	}
	want := strings.Repeat("x", 1024) + "\n"
	if string(output) != want {
		t.Fatalf("interactive output length = %d, want %d", len(output), len(want))
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

func TestInteractiveOutputHelper(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "interactive-output-helper" {
		return
	}
	_, _ = os.Stdout.WriteString(strings.Repeat("x", 1024) + "\n")
	if err := os.WriteFile(os.Args[len(os.Args)-2], []byte("exiting\n"), 0o600); err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}
