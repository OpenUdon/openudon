//go:build linux

package processgroup

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

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
