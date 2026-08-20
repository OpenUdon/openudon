//go:build windows

package processgroup

import (
	"context"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func prepare(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

func terminate(command *exec.Cmd) {
	if command.Process == nil {
		return
	}
	// taskkill /T is the Windows process-tree primitive. Process.Kill remains a
	// fail-closed fallback if taskkill is unavailable.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = exec.CommandContext(ctx, "taskkill", "/T", "/F", "/PID", strconv.Itoa(command.Process.Pid)).Run()
	_ = command.Process.Kill()
}
