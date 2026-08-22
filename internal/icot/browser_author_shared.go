package icot

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authorsession"
	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

type sharedBrowserAuthorSession interface {
	Events() <-chan browserauthor.Event
	Respond(context.Context, browserauthor.Response) error
	Cancel()
}

var startSharedBrowserAuthorSession = func(ctx context.Context, config browserauthor.Config) (sharedBrowserAuthorSession, error) {
	return browserauthor.Start(ctx, config)
}

var startSharedExternalBrowserAuthorSession = func(ctx context.Context, config browserauthor.Config, executable string) (sharedBrowserAuthorSession, error) {
	return browserauthor.StartExternal(ctx, config, executable)
}

// orchestrateBundledLiveAuthor adapts the terminal interaction to the same
// typed asynchronous controller used by the browser UI. Bundled and expert
// external launches differ only in their executable argument specification.
func orchestrateBundledLiveAuthor(ctx context.Context, cfg liveAuthorConfig, input liveLineReader, out io.Writer, planner livePlanner, provider, model string) (liveProtocolResult, error) {
	goalOrigin, goalPath, err := originAndPath(liveGoalURL(cfg))
	if err != nil {
		return liveProtocolResult{}, err
	}
	controllerConfig := browserauthor.Config{
		PrivateRoot: cfg.PrivateRoot, DriverDir: cfg.DriverDir, InitialURL: cfg.URL, DashboardURL: cfg.DashboardURL,
		Goal: cfg.Goal, Origins: append([]string(nil), cfg.Origins...), ProfileID: cfg.ProfileID,
		GoalPredicate: authorresult.GoalPredicate{Origin: goalOrigin, Path: goalPath, Context: cfg.GoalContext, Role: cfg.GoalRole, Label: cfg.GoalLabel},
	}
	var session sharedBrowserAuthorSession
	if cfg.BundledWorker {
		session, err = startSharedBrowserAuthorSession(ctx, controllerConfig)
	} else {
		session, err = startSharedExternalBrowserAuthorSession(ctx, controllerConfig, cfg.Browsertools)
	}
	if err != nil {
		return liveProtocolResult{}, err
	}
	success := false
	defer func() {
		if !success {
			session.Cancel()
		}
	}()
	disclosureDecided, plannerEnabled := planner == nil, false
	var lastObservation *liveObservation
	for event := range session.Events() {
		if event.ErrorCode != "" || event.State == "failed" || event.State == "canceled" {
			return liveProtocolResult{}, fmt.Errorf("isolated Browsertools worker failed closed")
		}
		if event.Observation != nil {
			observation := liveObservationFromShared(*event.Observation)
			lastObservation = &observation
			printLiveObservation(out, observation)
			if event.Checkpoint == nil {
				goal := liveGoalPredicate{Origin: goalOrigin, Path: goalPath, Context: cfg.GoalContext, Role: cfg.GoalRole, Label: cfg.GoalLabel}
				if liveObservationMatchesGoal(observation, goal) {
					continue
				}
				if planner != nil && !disclosureDecided {
					decision, promptErr := readLiveDecision(input, out, fmt.Sprintf("Allow this run to disclose reduced observations to %s/%s? Type disclose or human: ", provider, model), "disclose", "human")
					if promptErr != nil {
						return liveProtocolResult{}, promptErr
					}
					disclosureDecided, plannerEnabled = true, decision == "disclose"
					if !plannerEnabled {
						fmt.Fprintln(out, "icot: disclosure denied; continuing human-guided")
					}
				}
				plan := livePlan{Kind: "human"}
				if plannerEnabled {
					plan, err = planner.Plan(ctx, cfg.Goal, observation)
					if err != nil || validateLivePlan(plan, observation) != nil {
						fmt.Fprintln(out, "icot: planner action rejected; continuing human-guided")
						plan = livePlan{Kind: "human"}
					}
				}
				if plan.Kind == "human" {
					plan, err = readHumanLivePlan(input, out, observation)
					if err != nil {
						return liveProtocolResult{}, err
					}
				}
				if plan.Kind == "close" {
					return liveProtocolResult{}, fmt.Errorf("operator closed live authoring")
				}
				response := browserauthor.Response{Kind: plan.Kind, CandidateID: plan.CandidateID, URL: plan.URL, Context: plan.Context, POSTBudget: plan.POSTBudget}
				if plan.Kind == "authenticated" {
					choice := cfg.AfterAuthentication
					if choice == "ask_after_authentication" {
						decision, promptErr := readLiveDecision(input, out, "Authentication is complete. Type navigate to open the reviewed dashboard URL, or continue to keep the current page: ", "navigate", "continue")
						if promptErr != nil {
							return liveProtocolResult{}, promptErr
						}
						if decision == "continue" {
							choice = "continue_current_page"
						} else {
							choice = "navigate_absolute"
						}
					}
					if choice == "continue_current_page" {
						response = browserauthor.Response{Kind: "observe", Context: observation.Context}
					}
				}
				if err := session.Respond(ctx, response); err != nil {
					return liveProtocolResult{}, err
				}
				continue
			}
		}
		if event.Approval != nil {
			approval := event.Approval
			fmt.Fprintf(out, "Browsertools requests %q approval: action=%q candidate=%q origin=%q POST-budget=%d\n", approval.Kind, approval.Action, approval.CandidateID, approval.Origin, approval.POSTBudget)
			decision, promptErr := readLiveDecision(input, out, "Type approve for this exact request, or deny: ", "approve", "deny")
			if promptErr != nil {
				return liveProtocolResult{}, promptErr
			}
			if err := session.Respond(ctx, browserauthor.Response{Kind: decision, ApprovalID: approval.ID}); err != nil {
				return liveProtocolResult{}, err
			}
			continue
		}
		if event.Checkpoint != nil {
			checkpoint := event.Checkpoint
			switch checkpoint.Kind {
			case "credential":
				decision, promptErr := readLiveDecision(input, out, "Type continue after entering the value directly in Chromium, or close: ", "continue", "close")
				if promptErr != nil || decision != "continue" {
					return liveProtocolResult{}, fmt.Errorf("operator closed at human input checkpoint")
				}
				err = session.Respond(ctx, browserauthor.Response{Kind: "continue", CandidateID: checkpoint.CandidateID})
			case "mfa":
				fmt.Fprintf(out, "Compatible MFA kinds: %s\n", strings.Join(checkpoint.ChallengeKinds, ", "))
				challengeKind, promptErr := readLiveDecision(input, out, "Type the exact MFA kind you will complete: ", checkpoint.ChallengeKinds...)
				if promptErr != nil {
					return liveProtocolResult{}, promptErr
				}
				decision, promptErr := readLiveDecision(input, out, "Complete that challenge in Chromium or on the paired device, then type continue; or close: ", "continue", "close")
				if promptErr != nil || decision != "continue" {
					return liveProtocolResult{}, fmt.Errorf("operator closed at human MFA checkpoint")
				}
				err = session.Respond(ctx, browserauthor.Response{Kind: "continue", CandidateID: checkpoint.CandidateID, ChallengeKind: challengeKind})
			case "completion":
				if lastObservation == nil {
					return liveProtocolResult{}, fmt.Errorf("completion checkpoint has no current observation")
				}
				outputs, outputErr := readHumanOutputRequests(input, out, *lastObservation, liveAuthorSelectedMaxOutputs)
				if outputErr != nil {
					return liveProtocolResult{}, outputErr
				}
				printLiveOutputSummary(out, outputs, *lastObservation)
				decision, promptErr := readLiveDecision(input, out, "The typed predicate is satisfied. Type confirm to attest goal completion, or deny: ", "confirm", "deny")
				if promptErr != nil || decision != "confirm" {
					return liveProtocolResult{}, fmt.Errorf("human goal completion was denied")
				}
				sharedOutputs := make([]authorsession.OutputRequest, len(outputs))
				for index, output := range outputs {
					sharedOutputs[index] = authorsession.OutputRequest{CandidateID: output.CandidateID, Key: output.Key, Type: output.Type, LocatorMode: output.LocatorMode}
				}
				err = session.Respond(ctx, browserauthor.Response{Kind: "confirm", Confirmed: true, Outputs: sharedOutputs})
			default:
				return liveProtocolResult{}, fmt.Errorf("unexpected human checkpoint")
			}
			if err != nil {
				return liveProtocolResult{}, err
			}
			continue
		}
		if event.Result != nil {
			if event.Attestation == nil {
				return liveProtocolResult{}, fmt.Errorf("isolated Browsertools worker returned no parent attestation")
			}
			success = true
			return liveProtocolResult{ArtifactPath: event.Result.ArtifactPath, Digest: event.Result.Digest, Attestation: event.Attestation}, nil
		}
	}
	return liveProtocolResult{}, fmt.Errorf("isolated Browsertools worker ended without a result")
}

func liveObservationFromShared(observation authorsession.Observation) liveObservation {
	contexts := make(map[string]liveContext, len(observation.Contexts))
	for id, value := range observation.Contexts {
		contexts[id] = liveContext{Kind: value.Kind, Parent: value.Parent, Origin: value.Origin, Path: value.Path, Name: value.Name}
	}
	candidates := make([]liveCandidate, len(observation.Candidates))
	for index, value := range observation.Candidates {
		candidates[index] = liveCandidate{ID: value.ID, Role: value.Role, Label: value.Label, Matches: value.Matches}
	}
	return liveObservation{
		Origin: observation.Origin, Path: observation.Path, Context: observation.Context,
		Contexts: contexts, Candidates: candidates, Diagnostics: append([]string(nil), observation.Diagnostics...),
	}
}
