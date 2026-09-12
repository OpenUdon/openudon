package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/icot/engine"
)

const RegistrationControlVersion = "openudon.registration-control.v1"

// The supervisor owns this private stdin/stdout channel. It carries reviewed
// observation metadata, never worker prose. It is not a durable report stream.
type registrationControlFrame struct {
	Version   string          `json:"version"`
	Operation string          `json:"operation"`
	Request   json.RawMessage `json:"request,omitempty"`
}

type registrationControlState struct {
	Version              string                      `json:"version"`
	Revision             string                      `json:"revision"`
	RegistrationRevision string                      `json:"registration_revision"`
	State                *RegistrationAuthoringState `json:"state,omitempty"`
	Failure              string                      `json:"failure,omitempty"`
}

func (app RegistrationApplication) observe() registrationControlState {
	s := app.server
	s.mu.Lock()
	defer s.mu.Unlock()
	// responseLocked deep-copies the mutable disclosures while holding the lock.
	snapshot := s.responseLocked()
	return registrationControlState{Version: RegistrationControlVersion, Revision: s.revision, RegistrationRevision: s.registrationRevision, State: snapshot.RegistrationAuthoring}
}

func (app RegistrationApplication) close() error {
	s := app.server
	s.mu.Lock()
	if s.registrationSession != nil {
		s.registrationSession.Cancel()
	}
	if s.captureSession != nil {
		s.captureSession.Cancel()
	}
	if s.captureCancel != nil {
		s.captureCancel()
	}
	s.mu.Unlock()
	deadline := time.NewTimer(15 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		s.mu.Lock()
		done, failed := s.registrationSession == nil && s.captureSession == nil && s.captureCancel == nil, s.browserContainmentFailedLocked()
		s.mu.Unlock()
		if done {
			if failed {
				return errors.New("worker_teardown")
			}
			return nil
		}
		select {
		case <-deadline.C:
			return errors.New("worker_teardown")
		case <-tick.C:
		}
	}
}

// RunRegistrationControl uses the UI application state without an HTTP listener,
// browser controller or agent session. EOF, malformed input, signals and the
// nonrenewable deadline cancel the worker and join its retained terminal state.
func RunRegistrationControl(ctx context.Context, config RunConfig, input io.ReadCloser) error {
	return runPrivateControl(ctx, config, input, false)
}

// RunApplicationControl opens the shared application over private owned pipes.
func RunApplicationControl(ctx context.Context, config RunConfig, input io.ReadCloser) error {
	return runPrivateControl(ctx, config, input, true)
}

func runPrivateControl(ctx context.Context, config RunConfig, input io.ReadCloser, complete bool) (resultErr error) {
	if config.RegistrationAuthority != nil {
		deadline, err := time.Parse(time.RFC3339, config.RegistrationAuthority.ExpiresAt)
		if err != nil {
			return err
		}
		var stop context.CancelFunc
		ctx, stop = context.WithDeadline(ctx, deadline)
		defer stop()
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()
	author, snapshot, err := engine.Open(ctx, config.EngineConfig)
	if err != nil {
		return errors.New("application_start")
	}
	token, err := GenerateToken()
	if err != nil {
		return errors.New("application_start")
	}
	code, err := GenerateAccessCode()
	if err != nil {
		return errors.New("application_start")
	}
	handler, err := NewHandler(HandlerConfig{
		Context: ctx, Engine: author, Snapshot: snapshot, ExampleDir: config.EngineConfig.ExampleDir,
		Token: token, AccessCode: code, Authority: "127.0.0.1:1", ErrOut: io.Discard,
		PrivateRoot: config.EngineConfig.PrivateRoot, DriverDir: config.EngineConfig.DriverDir,
		BrowserTransactions: config.BrowserTransactions, PrepareCapture: config.PrepareCapture,
		RegistrationAuthority: config.RegistrationAuthority,
	})
	if err != nil {
		return errors.New("application_start")
	}
	app := RegistrationApplication{server: handler.(*Server)}
	defer func() {
		cancel()
		if resultErr != nil && resultErr.Error() == "operation_teardown" {
			return
		}
		if err := app.close(); err != nil {
			resultErr = err
		}
	}()
	if complete {
		return serveApplicationControl(ctx, Application{server: app.server}, input, config.Out)
	}
	return serveRegistrationControl(ctx, app, input, config.Out)
}

func serveRegistrationControl(ctx context.Context, app RegistrationApplication, input io.ReadCloser, output io.Writer) error {
	if input == nil || output == nil {
		return errors.New("input")
	}
	defer input.Close()
	// Owned pipe output must also unblock on deadline or signal cancellation.
	outputDone := make(chan struct{})
	defer close(outputDone)
	if closer, ok := output.(io.WriteCloser); ok {
		go func() {
			select {
			case <-ctx.Done():
				_ = closer.Close()
			case <-outputDone:
			}
		}()
	}
	type line struct {
		data []byte
		err  error
	}
	lines := make(chan line)
	done := make(chan struct{})
	defer close(done)
	go func() {
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 4096), MaxRequestBytes)
		for scanner.Scan() {
			select {
			case lines <- line{data: append([]byte(nil), scanner.Bytes()...)}:
			case <-done:
				return
			}
		}
		err := scanner.Err()
		if err == nil {
			err = io.EOF
		}
		select {
		case lines <- line{err: err}:
		case <-done:
		}
	}()
	encoder := json.NewEncoder(output)
	if err := encoder.Encode(app.observe()); err != nil {
		return errors.New("output")
	}
	for {
		var item line
		select {
		case <-ctx.Done():
			return errors.New("deadline_or_cancellation")
		case item = <-lines:
		}
		if item.err != nil {
			if item.err == io.EOF {
				return nil
			}
			return errors.New("input")
		}
		var frame registrationControlFrame
		if evidencefile.DecodeStrict(item.data, &frame) != nil || frame.Version != RegistrationControlVersion {
			return errors.New("protocol")
		}
		var result registrationResult
		switch frame.Operation {
		case "snapshot":
			if len(frame.Request) != 0 {
				return errors.New("protocol")
			}
		case "start":
			var request registrationAuthoringStartRequest
			if evidencefile.DecodeStrict(frame.Request, &request) != nil {
				return errors.New("protocol")
			}
			result = app.Start(ctx, request)
		case "command":
			var request registrationAuthoringCommandRequest
			if evidencefile.DecodeStrict(frame.Request, &request) != nil {
				return errors.New("protocol")
			}
			result = app.Command(ctx, request)
		case "cancel":
			var request registrationAuthoringCancelRequest
			if evidencefile.DecodeStrict(frame.Request, &request) != nil {
				return errors.New("protocol")
			}
			result = app.Cancel(ctx, request)
		case "close":
			if len(frame.Request) != 0 {
				return errors.New("protocol")
			}
			return nil
		default:
			return errors.New("protocol")
		}
		state := app.observe()
		if result.Failure != nil {
			state.Failure = result.Failure.Code
		}
		if err := encoder.Encode(state); err != nil {
			return errors.New("output")
		}
	}
}
