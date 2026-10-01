package browsercapture

import (
	"context"
	"errors"
	"io"

	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browserauthor"
	"github.com/OpenUdon/openudon/internal/browsercandidate"
)

type registrationSession interface {
	Events() <-chan browserauthor.RegistrationEvent
	Send(context.Context, browserauthor.RegistrationCommand) error
	ValidateDecision(string, int, browserauthor.RegistrationCommand) error
	TerminalEvent() (browserauthor.RegistrationEvent, bool)
	Cancel()
}

// RegistrationCompletion keeps the native independently reconstructed
// candidate process-private until exact review/package transaction admission.
type RegistrationCompletion struct {
	Candidate *browsercandidate.Registration `json:"-"`
}

// RunRegistration consumes fixedStart only after the caller has obtained
// approval for its exact initial URL/origins/profile/bounds. Later commands
// require protocol review cards. New capture uses native v4, including explicit
// verification approval; legacy controllers keep their original defaults.
func RunRegistration(ctx context.Context, config browserauthor.RegistrationConfig, fixedStart browserauthor.RegistrationCommand, in io.ReadCloser, out io.WriteCloser) (RegistrationCompletion, error) {
	return runRegistration(ctx, config, fixedStart, in, out, func(ctx context.Context, cfg browserauthor.RegistrationConfig) (registrationSession, error) {
		return browserauthor.StartRegistration(ctx, cfg)
	})
}

func runRegistration(ctx context.Context, config browserauthor.RegistrationConfig, fixedStart browserauthor.RegistrationCommand, in io.ReadCloser, out io.WriteCloser,
	start func(context.Context, browserauthor.RegistrationConfig) (registrationSession, error), complete ...func(context.Context, browserauthor.RegistrationEvent) (*profileImport, error)) (RegistrationCompletion, error) {
	var admission func(context.Context, browserauthor.RegistrationEvent) (*profileImport, error)
	if len(complete) > 0 {
		admission = complete[0]
	}
	if config.Protocol != registrationauthorsession.ProtocolV4 {
		return RegistrationCompletion{}, errors.New("browser capture registration protocol unsupported")
	}
	config, err := browserauthor.NormalizeRegistrationConfig(config)
	if err != nil {
		return RegistrationCompletion{}, errors.New("browser capture configuration invalid")
	}
	fixedStart, err = browserauthor.NormalizeRegistrationStart(config.Protocol, fixedStart)
	if err != nil {
		return RegistrationCompletion{}, errors.New("browser capture registration authority invalid")
	}
	var session registrationSession
	started := false
	validate := func(view View, command Command) error {
		if view.Registration == nil {
			return errInput
		}
		if command.DiscloseObservation != nil {
			if view.State != "observation" || view.Registration.Observation == nil {
				return errInput
			}
			return nil
		}
		if command.Registration == nil {
			return errInput
		}
		generation := 0
		if view.Registration.Observation != nil {
			generation = view.Registration.Observation.Generation
		}
		return session.ValidateDecision(view.State, generation, registrationCommand(*command.Registration))
	}
	event, err := drive(ctx, Registration, config.Absolute, in, out,
		func(ctx context.Context) (<-chan browserauthor.RegistrationEvent, func(), error) {
			var startErr error
			session, startErr = start(ctx, config)
			if startErr != nil {
				return nil, nil, startErr
			}
			return session.Events(), session.Cancel, nil
		}, reduceRegistration, validate,
		func(ctx context.Context, command Command) error {
			return session.Send(ctx, registrationCommand(*command.Registration))
		},
		func(ctx context.Context, event browserauthor.RegistrationEvent) error {
			if event.State != "ready" {
				return nil
			}
			if started {
				return errWorker
			}
			started = true
			return session.Send(ctx, fixedStart)
		}, func() (browserauthor.RegistrationEvent, bool) { return session.TerminalEvent() }, admission)
	if err != nil {
		return RegistrationCompletion{}, err
	}
	if event.State == "closed" || event.State == "canceled" {
		return RegistrationCompletion{}, ErrCanceled
	}
	if event.Candidate == nil || event.State != "candidate" {
		return RegistrationCompletion{}, errWorker
	}
	return RegistrationCompletion{Candidate: event.Candidate}, nil
}

func registrationCommand(command RegistrationCommand) browserauthor.RegistrationCommand {
	native := browserauthor.RegistrationCommand{Type: command.Type, Method: command.Method, URL: command.URL, Confirmed: command.Confirmed, Verification: command.Verification, VerificationCandidateID: command.CandidateID,
		Preview: command.Preview, CandidateIDs: command.CandidateIDs, StepCandidates: command.StepCandidates, Flow: command.Flow, CleanupDisposition: command.CleanupDisposition, CredentialBindings: command.CredentialBindings}
	if command.Profile != "" {
		native.Profile = []byte(command.Profile)
	}
	return native
}

func reduceRegistration(event browserauthor.RegistrationEvent) (string, View, bool) {
	view := View{State: event.State, Phase: event.Phase, Registration: &RegistrationView{Observation: event.Observation, Bounds: event.Bounds, VerificationAuthority: event.VerificationAuthority}}
	if len(event.Previews) > 0 {
		latest := event.Previews[len(event.Previews)-1]
		view.Registration.Preview = &latest
	}
	switch {
	case event.Candidate != nil:
		view.State, view.Phase, view.Registration = "captured", "completed", nil
		return "result", view, true
	case event.State == "failed" || event.State == "canceled":
		view.Registration = nil
		view.Diagnostic = "worker_failed"
		if event.ErrorCode == "worker_teardown" {
			view.Diagnostic = "worker_teardown"
		}
		if registrationauthorsession.ValidTerminalDiagnostic(event.Diagnostic) {
			view.Diagnostic = event.Diagnostic
		}
		// Containment failure takes precedence over the preceding diagnostic.
		if event.ErrorCode == "worker_teardown" {
			view.Diagnostic = "worker_teardown"
		}
		return "result", view, true
	case event.State == "closed":
		return "state", view, true
	case view.Registration.Preview != nil:
		return "preview", view, false
	case event.Observation != nil:
		return "observation", view, false
	default:
		return "state", view, false
	}
}
