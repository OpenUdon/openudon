// Package processgroup runs bounded subprocesses and terminates their complete
// process trees when the caller cancels or the fixed deadline expires.
package processgroup

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"
)

type Invocation struct {
	Args   []string
	Dir    string
	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func Run(ctx context.Context, timeout time.Duration, invocation Invocation) error {
	if timeout <= 0 {
		return fmt.Errorf("subprocess timeout must be positive")
	}
	return run(ctx, timeout, invocation)
}

// RunContext runs until completion or caller cancellation and terminates the
// complete process tree on cancellation. Callers that require a fixed bound
// should use Run.
func RunContext(ctx context.Context, invocation Invocation) error {
	return run(ctx, 0, invocation)
}

func run(ctx context.Context, timeout time.Duration, invocation Invocation) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(invocation.Args) == 0 {
		return fmt.Errorf("subprocess command is empty")
	}
	bounded := ctx
	cancel := func() {}
	if timeout > 0 {
		bounded, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()
	if err := bounded.Err(); err != nil {
		return err
	}
	commandPath, err := resolveCommand(invocation.Args[0], invocation.Env)
	if err != nil {
		return err
	}
	command := exec.Command(commandPath, invocation.Args[1:]...)
	command.Args[0] = invocation.Args[0]
	command.Dir = invocation.Dir
	// A nil exec.Cmd environment inherits the entire parent process. Typed
	// invocations use nil to mean an intentionally empty allowlist, so always
	// install a non-nil slice here.
	command.Env = append([]string{}, invocation.Env...)
	command.Stdin = invocation.Stdin
	command.Stdout = invocation.Stdout
	command.Stderr = invocation.Stderr
	prepare(command)
	if err := command.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		return err
	case <-bounded.Done():
		terminate(command)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
		return bounded.Err()
	}
}
