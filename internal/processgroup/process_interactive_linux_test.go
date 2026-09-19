//go:build linux

package processgroup

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestDescendantTrackerNeverReplacesARecordedPIDIdentity(t *testing.T) {
	known := map[int]uint64{123: 4242}
	if !recordStableDescendantIdentity(known, 123, 4242) {
		t.Fatal("matching identity was rejected")
	}
	if recordStableDescendantIdentity(known, 123, 5252) {
		t.Fatal("reused PID identity was adopted")
	}
	if known[123] != 4242 {
		t.Fatalf("recorded identity changed to %d", known[123])
	}
}

func TestRunSweepsDescendantsAfterNormalLeaderExit(t *testing.T) {
	testRunSweepsDescendant(t, false)
}

func TestRunSweepsDetachedSessionDescendantsAfterNormalLeaderExit(t *testing.T) {
	testRunSweepsDescendant(t, true)
}

func testRunSweepsDescendant(t *testing.T, detached bool) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	mode := "group"
	if detached {
		mode = "detached"
	}
	root := t.TempDir()
	pidFile, releaseFile := filepath.Join(root, "child"), filepath.Join(root, "release")
	var output bytes.Buffer
	observedTracker := make(chan *descendantTracker, 1)
	done := make(chan error, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	go func() {
		done <- runWithTracker(ctx, 10*time.Second, Invocation{
			Args: []string{executable, "-test.run=^TestNormalExitDescendantHelper$", "--", pidFile, releaseFile, mode, "normal-exit-descendant-helper"},
			Env:  os.Environ(), Stdout: &output, Stderr: io.Discard,
		}, func(pid int) *descendantTracker {
			tracker := startDescendantTracker(pid)
			observedTracker <- tracker
			return tracker
		})
	}()
	joined := false
	defer func() {
		if !joined {
			cancel()
			select {
			case <-done:
			case <-time.After(12 * time.Second):
				t.Error("fixture did not join")
			}
		}
	}()
	var tracker *descendantTracker
	select {
	case tracker = <-observedTracker:
	case <-ctx.Done():
		t.Fatal("tracker did not start")
	}
	seen := false
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		data, _ := os.ReadFile(pidFile)
		pid, parseErr := strconv.Atoi(string(data))
		if parseErr == nil {
			current, readErr := readProcIdentity(pid)
			tracker.mu.Lock()
			start, known := tracker.known[pid]
			tracker.mu.Unlock()
			if readErr == nil && known && current.startTime == start {
				seen = true
				break
			}
		}
		time.Sleep(time.Millisecond)
	}
	if !seen {
		t.Fatal("tracker did not observe fixture child before leader release")
	}
	if err := os.WriteFile(releaseFile, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		joined = true
		t.Fatal(err)
	}
	joined = true
	pid, err := strconv.Atoi(strings.TrimSpace(output.String()))
	if err != nil {
		t.Fatalf("parse descendant pid %q: %v", output.String(), err)
	}
	assertProcessGone(t, pid)
}

func TestInteractiveTerminateKillsDescendantsAfterLeaderExit(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	child, err := StartInteractive(context.Background(), []string{
		executable, "-test.run=^TestInteractiveDescendantHelper$", "--", "interactive-descendant-helper",
	}, os.Environ(), io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(child.Output()).ReadString('\n')
	if err != nil {
		t.Fatalf("read descendant pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatalf("parse descendant pid %q: %v", line, err)
	}
	if err := child.Wait(); err != nil {
		t.Fatalf("wait group leader: %v", err)
	}
	if err := child.Terminate(); err != nil {
		t.Fatalf("terminate reaped process group: %v", err)
	}
	assertProcessGone(t, pid)
}

func assertProcessGone(t *testing.T, pid int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, readErr := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
		if os.IsNotExist(readErr) || readErr == nil && strings.Contains(string(data), ") Z ") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("descendant process %d survived explicit group termination", pid)
}

func TestNormalExitDescendantHelper(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "normal-exit-descendant-helper" {
		return
	}
	command := exec.Command("/bin/sleep", "30")
	if os.Args[len(os.Args)-2] == "detached" {
		command.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	}
	if err := command.Start(); err != nil {
		os.Exit(2)
	}
	_, _ = fmt.Fprintf(os.Stdout, "%d\n", command.Process.Pid)
	pidFile, releaseFile := os.Args[len(os.Args)-4], os.Args[len(os.Args)-3]
	if os.WriteFile(pidFile, []byte(strconv.Itoa(command.Process.Pid)), 0600) != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		os.Exit(3)
	}
	// Only the test's confirmed PID/start-time observation releases this leader.
	for deadline := time.Now().Add(7 * time.Second); time.Now().Before(deadline); {
		if _, err := os.Stat(releaseFile); err == nil {
			os.Exit(0)
		}
		time.Sleep(time.Millisecond)
	}
	_ = command.Process.Kill()
	_ = command.Wait()
	os.Exit(3)
}

func TestInteractiveDescendantHelper(t *testing.T) {
	if len(os.Args) == 0 || os.Args[len(os.Args)-1] != "interactive-descendant-helper" {
		return
	}
	command := exec.Command("/bin/sleep", "30")
	if err := command.Start(); err != nil {
		os.Exit(2)
	}
	_, _ = fmt.Fprintf(os.Stdout, "%d\n", command.Process.Pid)
	os.Exit(0)
}
