//go:build linux

package processgroup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestDescendantTrackerChildDiscoveryContinuesDuringBlockedHostScan(t *testing.T) {
	root := t.TempDir()
	start, finish, pidFile := filepath.Join(root, "start"), filepath.Join(root, "finish"), filepath.Join(root, "child")
	script := `while ! test -e "$1"; do /bin/sleep 0.005; done
/usr/bin/setsid /bin/sleep 30 >/dev/null 2>&1 & child=$!
printf '%s' "$child" > "$3"
while ! test -e "$2"; do /bin/sleep 0.005; done`
	command := exec.Command("/usr/bin/bash", "-c", script, "fixture", start, finish, pidFile)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	rootDone := make(chan struct{})
	go func() { _ = command.Wait(); close(rootDone) }()
	entered, release := make(chan struct{}), make(chan struct{})
	var scanOnce, releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	tracker := startDescendantTrackerWithScan(command.Process.Pid, func() map[int]procIdentity {
		scanOnce.Do(func() { close(entered); <-release })
		return readProcIdentities()
	})
	t.Cleanup(func() {
		unblock()
		_ = tracker.terminateAndVerify(5 * time.Second)
		_ = command.Process.Kill()
		select {
		case <-rootDone:
		case <-time.After(2 * time.Second):
			t.Error("fixture leader did not stop")
		}
	})
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("host scan did not start")
	}
	if err := os.WriteFile(start, nil, 0600); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	var child procIdentity
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(pidFile)
		pid, err := strconv.Atoi(string(data))
		if err == nil {
			child, err = readProcIdentity(pid)
			if err == nil && child.parentPID == command.Process.Pid {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	if child.pid == 0 || child.parentPID != command.Process.Pid {
		t.Fatal("owned fixture identity missing")
	}
	t.Cleanup(func() {
		if current, err := readProcIdentity(child.pid); err == nil && current.startTime == child.startTime && current.state != 'Z' {
			_ = syscall.Kill(child.pid, syscall.SIGKILL)
		}
	})
	observed := false
	for time.Now().Before(deadline) {
		tracker.mu.Lock()
		observed = tracker.known[child.pid] == child.startTime
		tracker.mu.Unlock()
		if observed {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !observed {
		t.Fatal("blocked host scan prevented owned child discovery")
	}
	if err := os.WriteFile(finish, nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-rootDone:
	case <-time.After(2 * time.Second):
		t.Fatal("fixture leader did not exit")
	}
	unblock()
	if err := tracker.terminateAndVerify(5 * time.Second); err != nil {
		t.Fatal(err)
	}
	current, err := readProcIdentity(child.pid)
	if err == nil && current.startTime == child.startTime && current.state != 'Z' {
		t.Fatal("detached child survived teardown")
	}
}
