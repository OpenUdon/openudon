package browsercapture

import (
	"bufio"
	"context"
	"errors"
	"io"
	"time"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

var (
	ErrCanceled = errors.New("browser capture canceled")
	errInput    = errors.New("browser capture input invalid or disconnected")
	errWorker   = errors.New("browser capture worker failed")
)

// drive owns both closeable transport ends and serializes all gate/worker
// decisions. Close must unblock Read/Write; CLI pipes and local sockets do so.
// The controller's event channel must close only after its worker is joined.
// This loop is shared by both capture adapters, not a second browser engine.
func drive[E any](ctx context.Context, mode string, absolute time.Duration, in io.ReadCloser, out io.WriteCloser,
	start func(context.Context) (<-chan E, func(), error), reduce func(E) (string, View, bool),
	validate Validator, respond func(context.Context, Command) error,
	prepare func(context.Context, E) error, retained func() (E, bool),
	complete func(context.Context, E) (*profileImport, error)) (result E, err error) {
	var zero E
	if ctx == nil || in == nil || out == nil {
		return zero, errors.New("browser capture transport required")
	}
	bounded, cancel := context.WithTimeout(ctx, absolute)
	defer cancel()
	closeTransport := func() { _ = in.Close(); _ = out.Close() }
	defer closeTransport()
	stopClose := context.AfterFunc(bounded, closeTransport)
	defer stopClose()
	deadline, _ := bounded.Deadline()
	var admission *profileImport
	gate, err := NewGate(mode, deadline, func(view View, command Command) error {
		if admission != nil {
			return admission.validate(mode, view, command)
		}
		return validate(view, command)
	}, time.Now())
	if err != nil {
		return zero, errors.New("browser capture authority invalid")
	}
	updates, stopWorker, err := start(bounded)
	if err != nil {
		return zero, errors.New("browser capture worker unavailable")
	}
	workerUpdates := updates
	defer func() {
		stopWorker()
		// Drain until the controller has joined its owned process tree.
		// Late teardown failure must not be mistaken for successful cancel.
		for update := range workerUpdates {
			_, view, _ := reduce(update)
			if view.Diagnostic == "worker_teardown" {
				err = errWorker
			}
		}
		if retained != nil {
			if terminal, ok := retained(); ok {
				_, view, _ := reduce(terminal)
				if view.Diagnostic == "worker_teardown" {
					err = errWorker
				}
			}
		}
	}()
	frames := make(chan []byte, 1)
	readDone := make(chan struct{})
	readContext, stopRead := context.WithCancel(bounded)
	go func() {
		defer close(readDone)
		defer close(frames)
		scanner := bufio.NewScanner(in)
		scanner.Buffer(make([]byte, 4096), MaxMessageBytes+2)
		for scanner.Scan() {
			data := append([]byte(nil), scanner.Bytes()...)
			if len(data) == 0 || len(data) > MaxMessageBytes {
				return
			}
			select {
			case frames <- data:
			case <-readContext.Done():
				return
			}
		}
	}()
	defer func() { stopRead(); _ = in.Close(); <-readDone }()
	emit := func(event Event) error {
		data, encodeErr := boundedJSON(event)
		if encodeErr != nil {
			return encodeErr
		}
		n, writeErr := out.Write(append(data, '\n'))
		if writeErr != nil || n != len(data)+1 {
			return errors.New("browser capture output disconnected")
		}
		return nil
	}
	var current View
	var currentKind string
	ready, finished := false, false
	var last E
	for {
		select {
		case <-bounded.Done():
			return zero, ErrCanceled
		case update, ok := <-updates:
			if !ok {
				if retained != nil {
					if terminal, recorded := retained(); recorded {
						_, current, finished = reduce(terminal)
						last = terminal
					}
				}
				if !finished {
					return zero, errWorker
				}
				if complete != nil && current.Diagnostic == "" && current.State == "captured" {
					admission, err = complete(bounded, last)
					if err != nil || admission == nil {
						return zero, errors.New("browser capture profile review failed")
					}
					// The controller has joined. Keep the same transport, gate,
					// deadline and reader for separate exact import approval.
					updates = nil
					currentKind = "state"
					current = View{State: "import_review", Result: &admission.result}
					ready, finished = true, false
					event, observeErr := gate.Observe(currentKind, current, time.Now())
					if observeErr != nil {
						return zero, ErrCanceled
					}
					if emitErr := emit(event); emitErr != nil {
						return zero, emitErr
					}
					continue
				}
				final, observeErr := gate.Observe("result", current, time.Now())
				if observeErr != nil {
					return zero, ErrCanceled
				}
				if emitErr := emit(final); emitErr != nil {
					return zero, emitErr
				}
				return last, nil
			}
			if prepare != nil {
				if prepareErr := prepare(bounded, update); prepareErr != nil {
					return zero, errWorker
				}
			}
			kind, view, terminal := reduce(update)
			if finished && !terminal {
				return zero, errWorker
			}
			currentKind, current = kind, view
			ready = !terminal
			if terminal {
				finished, last = true, update
				// Invalidate pending decisions immediately. Publish terminal
				// output only after the controller's joined channel closure.
				gate.pending = nil
				continue
			}
			event, observeErr := gate.Observe(kind, view, time.Now())
			if observeErr != nil {
				return zero, errWorker
			}
			if emitErr := emit(event); emitErr != nil {
				return zero, emitErr
			}
		case data, ok := <-frames:
			if !ok {
				return zero, errInput
			}
			var tag struct {
				Type string `json:"type"`
			}
			// Decode the union through its exact schema below. This initial
			// tag extraction does not interpret or act on other fields.
			if err := decodeTag(data, &tag); err != nil {
				return zero, errInput
			}
			if finished || !ready && tag.Type != "cancel" {
				return zero, errInput
			}
			switch tag.Type {
			case "propose":
				proposal, decodeErr := DecodeProposal(data)
				if decodeErr != nil {
					return zero, errInput
				}
				event, proposeErr := gate.Propose(proposal, time.Now())
				if proposeErr != nil {
					return zero, errInput
				}
				if emitErr := emit(event); emitErr != nil {
					return zero, emitErr
				}
			case "decide":
				decision, decodeErr := DecodeDecision(data)
				if decodeErr != nil {
					return zero, errInput
				}
				command, decideErr := gate.Decide(decision, time.Now())
				if decideErr != nil {
					return zero, errInput
				}
				if command == nil || command.DiscloseObservation != nil {
					kind := currentKind
					if command != nil && *command.DiscloseObservation {
						kind = "disclosure"
					}
					// Only this fresh event's observation is consented. There
					// is no model call or session-wide disclosure permission.
					event, observeErr := gate.Observe(kind, current, time.Now())
					if observeErr != nil {
						return zero, ErrCanceled
					}
					if emitErr := emit(event); emitErr != nil {
						return zero, emitErr
					}
					continue
				}
				ready = false
				if admission != nil {
					if commitErr := admission.commit(bounded); commitErr != nil {
						return zero, errors.New("browser capture profile import failed; inspect package before retry")
					}
					final, observeErr := gate.Observe("result", View{State: "imported", Result: &admission.result}, time.Now())
					if observeErr != nil {
						return zero, ErrCanceled
					}
					if emitErr := emit(final); emitErr != nil {
						return zero, emitErr
					}
					return last, nil
				}
				if respondErr := respond(bounded, *command); respondErr != nil {
					return zero, errWorker
				}
			case "cancel":
				message, decodeErr := DecodeCancellation(data)
				if decodeErr != nil || gate.Cancel(message, time.Now()) != nil {
					return zero, errInput
				}
				return zero, ErrCanceled
			default:
				return zero, errInput
			}
		}
	}
}

func decodeTag(data []byte, tag *struct {
	Type string `json:"type"`
}) error {
	// DecodeStrict into a map checks duplicate fields/depth/multiple documents;
	// each closed typed decoder subsequently rejects unknown members.
	var object map[string]any
	if len(data) > MaxMessageBytes || evidencefile.DecodeStrict(data, &object) != nil {
		return errInput
	}
	tag.Type, _ = object["type"].(string)
	return nil
}
