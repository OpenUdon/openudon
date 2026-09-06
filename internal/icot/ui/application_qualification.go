package ui

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/OpenUdon/openudon/internal/processgroup"
)

// Qualification checkpoints are synthetic authority, so accept only the exact
// numeric loopback origin emitted by the local fixture. This is an application
// navigation constraint, not network-wide containment.
func qualificationLoopbackURL(origin, target string) bool {
	o, err := url.Parse(origin)
	if err != nil || o.Scheme != "http" || o.Hostname() != "127.0.0.1" || o.User != nil {
		return false
	}
	port, err := strconv.Atoi(o.Port())
	if err != nil || port < 1 || port > 65535 || origin != "http://127.0.0.1:"+strconv.Itoa(port) {
		return false
	}
	u, err := url.Parse(target)
	return err == nil && u.Scheme == o.Scheme && u.Host == o.Host && u.User == nil && u.Fragment == "" && u.Opaque == "" && u.Path != ""
}

// applicationQualification adapts the existing fixture journey vocabulary to
// a real shipped icot process. No application handler is invoked by this adapter.
type applicationQualification struct {
	child   *processgroup.InteractiveChild
	encoder *json.Encoder
	decoder *json.Decoder
	initial *Response
}

func newApplicationQualification(ctx context.Context, options RegistrationQualificationOptions) (*applicationQualification, error) {
	child, err := processgroup.StartInteractiveIn(ctx, options.RepoRoot, []string{options.ApplicationExecutable, "control", "--protocol", ApplicationControlVersion, "--no-open", "--network", "never", "--example", options.ExampleDir, "--private-root", options.PrivateRoot, "--package-scope", options.Scope, "--package-scratch", options.ScratchParent, "--package-store", options.StoreDir}, os.Environ(), io.Discard)
	if err != nil {
		return nil, errors.New("application_start")
	}
	q := &applicationQualification{child: child, encoder: json.NewEncoder(child.Input()), decoder: json.NewDecoder(child.Output())}
	current, err := q.read()
	if err != nil {
		_ = child.Terminate()
		return nil, err
	}
	q.initial = &current
	return q, nil
}
func (q *applicationQualification) read() (Response, error) {
	var state ApplicationControlState
	if err := q.decoder.Decode(&state); err != nil || state.Version != ApplicationControlVersion {
		return Response{}, errors.New("application_response")
	}
	if state.Failure != "" {
		return Response{}, errors.New("application_response:" + state.Failure + ":" + state.QuestionID)
	}
	var response Response
	if json.Unmarshal(state.Application, &response) != nil {
		return Response{}, errors.New("application_state")
	}
	return response, nil
}
func (q *applicationQualification) Close() error {
	_ = q.child.Input().Close()
	_, readErr := io.Copy(io.Discard, q.child.Output())
	err := q.child.Wait()
	if readErr != nil || err != nil {
		return errors.New("application_teardown")
	}
	return nil
}
func (q *applicationQualification) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	routes := map[string]string{
		"/api/v4/snapshot": "snapshot", "/api/v4/registration-authoring/start": "registration.start",
		"/api/v4/registration-authoring/command": "registration.command", "/api/v4/registration-authoring/cancel": "registration.cancel",
		"/api/v4/browser-transactions/review": "transaction.review", "/api/v4/browser-transactions/prepare": "transaction.prepare",
		"/api/v4/browser-transactions/promote": "transaction.promote", "/api/v4/round": "author.round",
		"/api/v4/author/approve": "author.approve", "/api/v4/package/build": "package.build",
	}
	operation, ok := routes[r.URL.Path]
	if !ok {
		w.WriteHeader(400)
		return
	}
	status := http.StatusOK
	if operation == "registration.start" || operation == "registration.cancel" {
		status = http.StatusAccepted
	}
	var body json.RawMessage
	if r.Method == http.MethodPost {
		data, err := io.ReadAll(io.LimitReader(r.Body, MaxRequestBytes+1))
		if err != nil || len(data) > MaxRequestBytes {
			w.WriteHeader(400)
			return
		}
		body = data
		if operation == "registration.command" {
			var command registrationAuthoringCommandRequest
			if json.Unmarshal(body, &command) != nil {
				w.WriteHeader(400)
				return
			}
			if command.Type != "draft" {
				status = http.StatusAccepted
			}
		}
	}
	var response Response
	if q.initial != nil && operation == "snapshot" {
		response = *q.initial
		q.initial = nil
	} else {
		if q.encoder.Encode(registrationControlFrame{Version: ApplicationControlVersion, Operation: operation, Request: body}) != nil {
			w.WriteHeader(500)
			return
		}
		var err error
		response, err = q.read()
		if err != nil {
			w.WriteHeader(500)
			return
		}
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
