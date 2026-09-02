// Package processgroup runs bounded subprocesses and terminates their complete
// process trees when the caller cancels or the fixed deadline expires.
package processgroup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// ErrTerminationTimeout means the complete process tree did not confirm
// termination within the containment deadline.
var ErrTerminationTimeout = errors.New("process tree did not terminate")

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
	tracker *descendantTracker
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
	child := &InteractiveChild{
		command: command,
		tracker: startDescendantTracker(command.Process.Pid),
		input:   input,
		output:  output,
		done:    make(chan struct{}),
	}
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
		waitErr := child.command.Wait()
		child.err = errors.Join(waitErr, sweepProcessTree(child.command, child.tracker))
		close(child.done)
	})
	<-child.done
	return child.err
}
func (child *InteractiveChild) Terminate() error {
	if child == nil || child.command == nil {
		return nil
	}
	select {
	case <-child.done:
		return child.err
	default:
	}
	if runtime.GOOS == "linux" {
		child.tracker.killRootGroupIfLive()
		go func() { _ = child.Wait() }()
		if err := child.tracker.terminateAndVerify(5 * time.Second); err != nil {
			return err
		}
	} else {
		go func() { _ = child.Wait() }()
		terminate(child.command)
	}
	select {
	case <-child.done:
		return child.err
	case <-time.After(5 * time.Second):
		return ErrTerminationTimeout
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
	tracker := startDescendantTracker(command.Process.Pid)
	done := make(chan error, 1)
	go func() {
		waitErr := command.Wait()
		done <- errors.Join(waitErr, sweepProcessTree(command, tracker))
	}()
	select {
	case err := <-done:
		return err
	case <-bounded.Done():
		select {
		case err := <-done:
			return err
		default:
		}
		if runtime.GOOS == "linux" {
			tracker.killRootGroupIfLive()
			if err := tracker.terminateAndVerify(5 * time.Second); err != nil {
				return errors.Join(bounded.Err(), ErrTerminationTimeout)
			}
		} else {
			terminate(command)
		}
		select {
		case waitErr := <-done:
			return canceledRunError(bounded.Err(), waitErr)
		case <-time.After(5 * time.Second):
			return errors.Join(bounded.Err(), ErrTerminationTimeout)
		}
	}
}

func canceledRunError(contextErr, waitErr error) error {
	if errors.Is(waitErr, ErrTerminationTimeout) {
		return errors.Join(contextErr, ErrTerminationTimeout)
	}
	return contextErr
}

func sweepProcessTree(command *exec.Cmd, tracker *descendantTracker) error {
	// A successful group leader may leave children running. Sweep after every
	// exit path, not only cancellation. Linux uses only immutable PID/start-time
	// identities after Wait; signaling a reaped numeric PGID could target an
	// unrelated process after PID reuse. Other platforms retain their native
	// post-wait tree cleanup before the platform tracker verifies completion.
	if runtime.GOOS != "linux" {
		terminate(command)
	}
	if tracker == nil {
		return nil
	}
	return tracker.terminateAndVerify(5 * time.Second)
}
