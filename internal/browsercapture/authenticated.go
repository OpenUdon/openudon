package browsercapture

import (
	"context"
	"errors"
	"io"

	"github.com/OpenUdon/openudon/internal/browserauthor"
)

// AuthenticatedCompletion is process-private input to independent profile
// review/import. It has no wire representation and conveys no package authority.
type AuthenticatedCompletion struct {
	Event browserauthor.Event `json:"-"`
}

type authenticationSession interface {
	Events() <-chan browserauthor.Event
	Respond(context.Context, browserauthor.Response) error
	Cancel()
}

// RunAuthenticated uses the existing Browsertools-backed controller. The
// transport is owned and closed here; callers must not log complete frames.
// This internal adapter is not a supported Go-library API.
func RunAuthenticated(ctx context.Context, config browserauthor.Config, in io.ReadCloser, out io.WriteCloser) (AuthenticatedCompletion, error) {
	return runAuthenticated(ctx, config, in, out, func(ctx context.Context, cfg browserauthor.Config) (authenticationSession, error) {
		return browserauthor.Start(ctx, cfg)
	})
}

func runAuthenticated(ctx context.Context, config browserauthor.Config, in io.ReadCloser, out io.WriteCloser,
	start func(context.Context, browserauthor.Config) (authenticationSession, error), options ...authenticationOptions) (AuthenticatedCompletion, error) {
	var option authenticationOptions
	if len(options) > 0 {
		option = options[0]
	}
	config, err := browserauthor.NormalizeConfig(config)
	if err != nil {
		return AuthenticatedCompletion{}, errors.New("browser capture configuration invalid")
	}
	var session authenticationSession
	validate := func(view View, command Command) error {
		if view.Authentication == nil {
			return errWorker
		}
		if command.DiscloseObservation != nil {
			if view.Authentication.Observation == nil || view.State != "exploration" {
				return errInput
			}
			return nil
		}
		if command.Authentication == nil {
			return errInput
		}
		if option.continuation == "continue_current_page" && command.Authentication.Kind == "authenticated" {
			return errInput
		}
		auth := view.Authentication
		return browserauthor.ValidateResponse(config, browserauthor.Event{State: view.State, Phase: view.Phase, Observation: auth.Observation, Approval: auth.Approval, Checkpoint: auth.Checkpoint}, *command.Authentication)
	}
	event, err := drive(ctx, Authenticated, config.Absolute, in, out,
		func(ctx context.Context) (<-chan browserauthor.Event, func(), error) {
			var startErr error
			session, startErr = start(ctx, config)
			if startErr != nil {
				return nil, nil, startErr
			}
			return session.Events(), session.Cancel, nil
		}, reduceAuthentication,
		validate, func(ctx context.Context, command Command) error {
			return session.Respond(ctx, *command.Authentication)
		}, nil, nil, option.complete)
	if err != nil {
		return AuthenticatedCompletion{}, err
	}
	if event.State == "canceled" || event.State == "closed" {
		return AuthenticatedCompletion{}, ErrCanceled
	}
	if event.Result == nil || event.Attestation == nil {
		return AuthenticatedCompletion{}, errWorker
	}
	return AuthenticatedCompletion{Event: event}, nil
}

func reduceAuthentication(event browserauthor.Event) (string, View, bool) {
	view := View{State: event.State, Phase: event.Phase, Authentication: &AuthenticatedView{
		Observation: event.Observation, Approval: event.Approval, Checkpoint: event.Checkpoint,
	}}
	switch {
	case event.Result != nil:
		// Private path, digest and attestation are retained only in-process.
		view.State, view.Phase, view.Authentication = "captured", "completed", nil
		return "result", view, true
	case event.State == "failed" || event.State == "canceled" || event.State == "closed":
		view.Authentication = nil
		view.Diagnostic = "worker_failed"
		if event.State == "canceled" || event.State == "closed" {
			view.Diagnostic = "canceled"
		}
		if event.ErrorCode == "worker_teardown" {
			view.Diagnostic = "worker_teardown"
		}
		return "result", view, true
	case event.Checkpoint != nil:
		return "human_checkpoint", view, false
	case event.Approval != nil:
		return "approval_required", view, false
	case event.Observation != nil:
		return "observation", view, false
	default:
		return "state", view, false
	}
}
