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
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

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
	var output bytes.Buffer
	err = Run(context.Background(), 5*time.Second, Invocation{
		Args: []string{
			executable, "-test.run=^TestNormalExitDescendantHelper$", "--", mode, "normal-exit-descendant-helper",
		},
		Env: os.Environ(), Stdout: &output, Stderr: io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
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
	// Give the Linux identity monitor a chance to record the child after it has
	// detached. The runtime contract relies on continuous /proc observation,
	// not on the child retaining the original process group.
	time.Sleep(25 * time.Millisecond)
	os.Exit(0)
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
