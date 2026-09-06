package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const ApplicationControlVersion = "openudon.application-control.v1"

// ApplicationControlState is private operator-facing state. It has the same
// reviewed resources as the UI, never producer envelopes or credential values.
// Application is encoded while holding the state lock to prevent alias races.
type ApplicationControlState struct {
	Version     string          `json:"version"`
	Application json.RawMessage `json:"application"`
	Failure     string          `json:"failure,omitempty"`
	QuestionID  string          `json:"question_id,omitempty"`
}

func (app Application) observe(ctx context.Context) (ApplicationControlState, error) {
	reply := app.Snapshot(ctx)
	if reply.Failure != nil {
		return ApplicationControlState{}, errors.New("state")
	}
	data, ok := reply.Data.(json.RawMessage)
	if !ok || len(data) > 4*MaxRequestBytes {
		return ApplicationControlState{}, errors.New("state")
	}
	return ApplicationControlState{Version: ApplicationControlVersion, Application: data}, nil
}

func applicationRequest[T any](raw json.RawMessage, apply func(T) applicationResult) (applicationResult, error) {
	var request T
	if evidencefile.DecodeStrict(raw, &request) != nil || string(raw) == "null" {
		return applicationResult{}, errors.New("protocol")
	}
	return apply(request), nil
}

func registrationApplicationResult(result registrationResult) applicationResult {
	return applicationResult{Status: result.Status, Data: result.Snapshot, ETag: result.ETag, Revision: result.Revision, Failure: result.Failure}
}

// dispatch is a closed application union; it cannot select HTTP routes, invoke
// an executor, supply source bytes, or issue arbitrary browser commands.
func (app Application) dispatch(ctx context.Context, frame registrationControlFrame) (applicationResult, error) {
	raw := frame.Request
	switch frame.Operation {
	case "snapshot":
		if len(raw) != 0 {
			return applicationResult{}, errors.New("protocol")
		}
		return applicationResult{}, nil
	case "registration.start":
		return applicationRequest(raw, func(r registrationAuthoringStartRequest) applicationResult {
			return registrationApplicationResult((RegistrationApplication{app.server}).Start(ctx, r))
		})
	case "registration.command":
		return applicationRequest(raw, func(r registrationAuthoringCommandRequest) applicationResult {
			return registrationApplicationResult((RegistrationApplication{app.server}).Command(ctx, r))
		})
	case "registration.cancel":
		return applicationRequest(raw, func(r registrationAuthoringCancelRequest) applicationResult {
			return registrationApplicationResult((RegistrationApplication{app.server}).Cancel(ctx, r))
		})
	case "browser.preflight":
		return applicationRequest(raw, func(r captureMutationRequest) applicationResult { return app.BrowserPreflight(ctx, r) })
	case "capture.start":
		return applicationRequest(raw, func(r captureStartRequest) applicationResult { return app.CaptureStart(ctx, r) })
	case "capture.respond":
		return applicationRequest(raw, func(r captureRespondRequest) applicationResult { return app.CaptureRespond(ctx, r) })
	case "capture.cancel":
		return applicationRequest(raw, func(r captureMutationRequest) applicationResult { return app.CaptureCancel(ctx, r) })
	case "capture.stage":
		return applicationRequest(raw, func(r captureMutationRequest) applicationResult { return app.CaptureStage(ctx, r) })
	case "author.journey":
		return applicationRequest(raw, func(r journeyRequest) applicationResult { return app.Journey(ctx, r) })
	case "author.round":
		return applicationRequest(raw, func(r roundRequest) applicationResult { return app.Round(ctx, r) })
	case "author.reopen":
		return applicationRequest(raw, func(r reopenRequest) applicationResult { return app.Reopen(ctx, r) })
	case "author.approve":
		return applicationRequest(raw, func(r approveRequest) applicationResult { return app.Approve(ctx, r) })
	case "author.resume":
		return applicationRequest(raw, func(r revisionRequest) applicationResult { return app.Resume(ctx, r) })
	case "package.build":
		return applicationRequest(raw, func(r buildRequest) applicationResult { return app.PackageBuild(ctx, r) })
	}
	routes := map[string]string{
		"transaction.start": "start", "transaction.review": "review", "transaction.prepare": "prepare",
		"transaction.promote": "promote", "transaction.cancel": "cancel", "transaction.inspect_recovery": "recovery/inspect",
		"transaction.recover": "recovery/reconcile", "transaction.inspect_selected": "selected/inspect",
	}
	if route, ok := routes[frame.Operation]; ok {
		if len(raw) == 0 || string(raw) == "null" {
			return applicationResult{}, errors.New("protocol")
		}
		return app.Transaction(ctx, "/api/v4/browser-transactions/"+route, raw), nil
	}
	return applicationResult{}, errors.New("protocol")
}

// serveApplicationControl owns one operation at a time. Close, cancel, EOF and
// signals can interrupt an operation while it is waiting on worker readiness or
// package I/O. Pipelined mutations fail instead of being replayed against a new
// revision. The caller joins all worker trees after the operation has stopped.
func serveApplicationControl(ctx context.Context, app Application, input io.ReadCloser, output io.Writer) error {
	if input == nil || output == nil {
		return errors.New("input")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer input.Close()
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
	readerDone := make(chan struct{})
	defer close(readerDone)
	go func() {
		scanner := bufio.NewScanner(input)
		scanner.Buffer(make([]byte, 4096), MaxRequestBytes)
		for scanner.Scan() {
			select {
			case lines <- line{data: append([]byte(nil), scanner.Bytes()...)}:
			case <-readerDone:
				return
			}
		}
		err := scanner.Err()
		if err == nil {
			err = io.EOF
		}
		select {
		case lines <- line{err: err}:
		case <-readerDone:
		}
	}()
	encoder := json.NewEncoder(output)
	send := func(reply applicationResult) error {
		state, err := app.observe(ctx)
		if err != nil {
			return err
		}
		if reply.Failure != nil {
			state.Failure = reply.Failure.Code
			state.QuestionID = reply.Failure.QuestionID
		}
		if err := encoder.Encode(state); err != nil {
			return errors.New("output")
		}
		return nil
	}
	if err := send(applicationResult{}); err != nil {
		return err
	}
	decode := func(item line) (registrationControlFrame, error) {
		var frame registrationControlFrame
		if item.err != nil {
			return frame, item.err
		}
		if evidencefile.DecodeStrict(item.data, &frame) != nil || frame.Version != ApplicationControlVersion {
			return frame, errors.New("protocol")
		}
		return frame, nil
	}
	terminal := func(frame registrationControlFrame) bool {
		return (frame.Operation == "close" || frame.Operation == "cancel") && len(frame.Request) == 0
	}
	for {
		var item line
		select {
		case <-ctx.Done():
			return errors.New("deadline_or_cancellation")
		case item = <-lines:
		}
		frame, err := decode(item)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.New("protocol")
		}
		if terminal(frame) {
			return nil
		}
		type completed struct {
			reply applicationResult
			err   error
		}
		done := make(chan completed, 1)
		operationCtx, stop := context.WithCancel(ctx)
		go func() { reply, err := app.dispatch(operationCtx, frame); done <- completed{reply, err} }()
		var result completed
		var interrupted error
		select {
		case result = <-done:
			stop()
			if result.err != nil {
				return result.err
			}
			if err := send(result.reply); err != nil {
				return err
			}
			continue
		case <-ctx.Done():
			interrupted = errors.New("deadline_or_cancellation")
		case next := <-lines:
			nextFrame, err := decode(next)
			if err != io.EOF && (err != nil || !terminal(nextFrame)) {
				interrupted = errors.New("protocol")
			}
		}
		stop()
		select {
		case result = <-done:
			if interrupted != nil {
				return interrupted
			}
			return result.err
		case <-time.After(15 * time.Second):
			return errors.New("operation_teardown")
		}
	}
}
