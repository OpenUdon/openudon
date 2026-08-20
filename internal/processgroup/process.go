// Package processgroup runs bounded subprocesses and terminates their complete
// process trees when the caller cancels or the fixed deadline expires.
package processgroup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
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

// InteractiveChild is a process-group-owned child with pipe access. Parent
// cancellation and Terminate both kill the complete process tree.
type InteractiveChild struct {
	command *exec.Cmd
	input   io.WriteCloser
	output  io.ReadCloser
	done    chan struct{}
	wait    sync.Once
	err     error
}

func StartInteractive(ctx context.Context, args, environment []string, stderr io.Writer) (*InteractiveChild, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("subprocess command is empty")
	}
	commandPath, err := resolveCommand(args[0], environment)
	if err != nil {
		return nil, err
	}
	command := exec.Command(commandPath, args[1:]...)
	command.Args[0] = args[0]
	command.Env = append([]string{}, environment...)
	command.Stderr = stderr
	prepare(command)
	input, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, err := command.StdoutPipe()
	if err != nil {
		_ = input.Close()
		return nil, err
	}
	if err := command.Start(); err != nil {
		_ = input.Close()
		_ = output.Close()
		return nil, err
	}
	child := &InteractiveChild{command: command, input: input, output: output, done: make(chan struct{})}
	go func() {
		select {
		case <-ctx.Done():
			// Prefer a completed Wait when cancellation races normal shutdown.
			// Otherwise terminate the complete process group and reap the leader.
			select {
			case <-child.done:
				return
			default:
			}
			_ = child.Terminate()
		case <-child.done:
		}
	}()
	return child, nil
}

func (child *InteractiveChild) Input() io.WriteCloser { return child.input }
func (child *InteractiveChild) Output() io.ReadCloser { return child.output }
func (child *InteractiveChild) Wait() error {
	if child == nil {
		return os.ErrInvalid
	}
	// StdoutPipe requires its reader to drain protocol output before Wait closes
	// the pipe. The caller owns that ordering; concurrent cleanup calls share
	// this single reap operation.
	child.wait.Do(func() {
		child.err = child.command.Wait()
		close(child.done)
	})
	<-child.done
	return child.err
}
func (child *InteractiveChild) Terminate() error {
	if child == nil || child.command == nil {
		return nil
	}
	// A reaped group leader does not prove that its descendants exited. Always
	// target the process tree when explicit termination is requested.
	terminate(child.command)
	go func() { _ = child.Wait() }()
	select {
	case <-child.done:
		return child.err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("process tree did not terminate")
	}
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
