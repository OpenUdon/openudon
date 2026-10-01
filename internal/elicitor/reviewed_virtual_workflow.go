package elicitor

import (
	"errors"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

// ReviewedVirtualWorkflow proposes one exact captured recipe using the same
// native operation/session/credential lowering as interactive authoring. Its
// approval fields describe the proposal; only a later confirmed writer may
// commit it. This function is pure and grants no executor authority.
func ReviewedVirtualWorkflow(seed Session, discovery VirtualBrowserDiscovery, flow, action string, bindings map[string]string) (Session, error) {
	var ids []string
	var authDoc, selected APIDocument
	var authOperation, selectedOperation *apitools.OperationSummary
	for _, candidate := range discovery.Candidates {
		ids = append(ids, candidate.ID)
		for _, doc := range discovery.Docs {
			if doc.RelativePath != candidate.TargetPath {
				continue
			}
			for i := range doc.Operations {
				op := doc.Operations[i]
				switch candidate.Kind {
				case browsertransaction.CandidateAuthentication:
					if op.OperationID == flow && candidate.Flow == flow {
						authDoc = doc
						authOperation = &op
					}
				case browsertransaction.CandidateCapability:
					if op.OperationID == action {
						selected = doc
						selectedOperation = &op
					}
				case browsertransaction.CandidateRegistration:
					if op.OperationID == flow && candidate.Flow == flow {
						selected = doc
						selectedOperation = &op
					}
				}
			}
		}
	}
	if selectedOperation == nil {
		return Session{}, errors.New("native browser operation selection required")
	}
	next, err := SelectVirtualBrowserSources(seed, discovery, ids)
	if err != nil {
		return Session{}, err
	}
	step := stepFromOperation(selected, selectedOperation)
	if step == nil {
		return Session{}, errors.New("native browser operation unavailable")
	}
	step.Source = selected.RelativePath
	step.With = bindings
	// Native authoring requires a workflow result. Export the operation result
	// without inventing a new selector, response field, or runtime value.
	next.Intent.Outputs = []*rollout.Output{{Name: "result", From: step.Name}}
	next.Intent.Source = selected.RelativePath
	next.Intent.Steps = []*rollout.Step{step}
	next.BrowserRoute = "browser"
	next.BrowserSession = "none"
	if step.Type == "browser_registration" {
		step.RegistrationApproval = step.Name
		mergeBrowserRegistrationCredentials(&next, step)
	} else {
		if authOperation == nil {
			return Session{}, errors.New("native authentication flow selection required")
		}
		auth := insertBrowserAuthenticationStep(&next, step, authDoc, authOperation)
		if auth == nil {
			return Session{}, errors.New("native authentication lowering failed")
		}
		timeout := 120.0
		auth.Timeout = &timeout
		next.BrowserAuthenticationApprovals = []string{auth.Name}
		if strings.EqualFold(selectedOperation.Extensions["openudon.browser.side_effect"], "true") {
			next.BrowserApprovals = []string{step.Name}
		}
	}
	next.Normalize()
	return next, nil
}
