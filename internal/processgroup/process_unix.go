//go:build !windows

package processgroup

import (
	"os/exec"
	"syscall"
)

func prepare(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminate(command *exec.Cmd) {
	if command.Process == nil {
		return
	}
	_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	_ = command.Process.Kill()
}
