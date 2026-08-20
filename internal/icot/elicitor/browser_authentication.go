package elicitor

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/openudon/internal/browserworkflow"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	readinessMissingBrowserAuthenticationFlow    = "missing_browser_authentication_flow"
	readinessMissingBrowserAuthenticationSession = "missing_browser_authentication_session"
	readinessMissingBrowserCredentialBindings    = "missing_browser_authentication_credential_bindings"
	readinessMissingBrowserAuthenticationTimeout = "missing_browser_authentication_timeout"
	readinessUnconfirmedBrowserAuthentication    = "unconfirmed_browser_authentication"
)

var browserBindingNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func browserAuthenticationReadinessIssues(session Session, docs []APIDocument, step *rollout.Step) []ReadinessIssue {
	if step == nil {
		return nil
	}
	name := firstNonEmpty(step.Name, "authenticate")
	slot := "steps." + name
	var issues []ReadinessIssue
	add := func(code, suffix, message, suggested string) {
		issues = append(issues, ReadinessIssue{Code: code, Slot: slot + suffix, Severity: readinessBlocking, Message: message, SuggestedAnswer: suggested})
	}
	doc, operation := browserAuthenticationOperationForStep(session, docs, step)
	if operation == nil {
		suggested := suggestedBrowserAuthenticationFlow(docs)
		if strings.TrimSpace(step.AuthenticationFlow) == "" {
			add(readinessMissingBrowserAuthenticationFlow, ".authentication_flow", "Select one reviewed browser authentication flow. The portable workflow stores only the profile path and flow name, never credentials or session state.", suggested)
		} else {
			add(readinessMissingBrowserAuthenticationFlow, ".authentication_flow", fmt.Sprintf("Authentication flow %q is not present in the selected reviewed profile.", step.AuthenticationFlow), suggested)
		}
		return issues
	}
	if strings.TrimSpace(step.BrowserSession) == "" {
		add(readinessMissingBrowserAuthenticationSession, ".browser_session", "Name the execution-local browser session established by this authentication step.", defaultBrowserSessionName(doc))
	}
	required := browserAuthenticationCredentialSlots(operation)
	if !exactBrowserCredentialBindings(step.CredentialBindings, required) {
		add(readinessMissingBrowserCredentialBindings, ".credential_bindings", "Map every credential slot required by the selected flow to a symbolic runtime binding. Do not enter credential values.", suggestedBrowserCredentialBindings(required))
	}
	if step.Timeout == nil || *step.Timeout <= 0 || *step.Timeout > 600 {
		add(readinessMissingBrowserAuthenticationTimeout, ".timeout", "Set a bounded authentication timeout of no more than 600 seconds so an unattended MFA challenge cannot wait forever.", "120")
	}
	if !stringSliceContains(session.BrowserAuthenticationApprovals, step.Name) {
		add(readinessUnconfirmedBrowserAuthentication, ".authentication_approval", "Browser sign-in and MFA require explicit operation-specific authoring approval. Runtime execution will require a separate exact approval.", "approve "+step.Name)
	}
	return issues
}

func browserAuthenticationOperationForStep(session Session, docs []APIDocument, step *rollout.Step) (APIDocument, *apitools.OperationSummary) {
	if step == nil {
		return APIDocument{}, nil
	}
	ref := stepAPISourceRef(session, step)
	for _, doc := range docs {
		if !isBrowserAuthenticationDocument(doc) || doc.RelativePath != ref {
			continue
		}
		for index := range doc.Operations {
			op := &doc.Operations[index]
			if op.OperationID == strings.TrimSpace(step.AuthenticationFlow) {
				copy := *op
				return doc, &copy
			}
		}
	}
	return APIDocument{}, nil
}

func browserAuthenticationCredentialSlots(operation *apitools.OperationSummary) []string {
	if operation == nil {
		return nil
	}
	var slots []string
	for _, slot := range strings.Split(operation.Extensions["openudon.browser_authentication.credential_slots"], ",") {
		if slot = strings.TrimSpace(slot); slot != "" {
			slots = append(slots, slot)
		}
	}
	return dedupeStrings(slots)
}

func exactBrowserCredentialBindings(bindings map[string]string, required []string) bool {
	if len(bindings) != len(required) {
		return false
	}
	for _, slot := range required {
		binding := strings.TrimSpace(bindings[slot])
		if !browserBindingNamePattern.MatchString(binding) {
			return false
		}
	}
	return true
}

func suggestedBrowserCredentialBindings(slots []string) string {
	values := make([]string, 0, len(slots))
	for _, slot := range slots {
		values = append(values, slot+"="+slot)
	}
	return strings.Join(values, ", ")
}

func suggestedBrowserAuthenticationFlow(docs []APIDocument) string {
	for _, doc := range docs {
		if !isBrowserAuthenticationDocument(doc) || len(doc.Operations) == 0 {
			continue
		}
		return doc.RelativePath + "#" + doc.Operations[0].OperationID
	}
	return ""
}

func defaultBrowserSessionName(doc APIDocument) string {
	name := slugIdent(firstNonEmpty(doc.ID, doc.Title, "browser"))
	if name == "" {
		name = "browser"
	}
	return name + "_session"
}

func browserAuthenticationAvailable(docs []APIDocument) bool {
	for _, doc := range docs {
		if isBrowserAuthenticationDocument(doc) && len(doc.Operations) > 0 {
			return true
		}
	}
	return false
}

func browserActionHasEstablishedSession(session Session, action *rollout.Step) bool {
	return browserworkflow.Analyze(&session.Intent).EstablishedBefore(action)
}

func selectBrowserAuthenticationFlow(answer string, docs []APIDocument) (APIDocument, *apitools.OperationSummary) {
	answer = strings.TrimSpace(answer)
	for _, doc := range docs {
		if !isBrowserAuthenticationDocument(doc) {
			continue
		}
		for index := range doc.Operations {
			op := &doc.Operations[index]
			coordinate := doc.RelativePath + "#" + op.OperationID
			if answer == coordinate || answer == op.OperationID {
				copy := *op
				return doc, &copy
			}
		}
	}
	return APIDocument{}, nil
}

func insertBrowserAuthenticationStep(session *Session, action *rollout.Step, doc APIDocument, operation *apitools.OperationSummary) *rollout.Step {
	if session == nil || operation == nil {
		return nil
	}
	name := uniqueIntentStepName(session.Intent.Steps, "authenticate_"+slugIdent(firstNonEmpty(doc.ID, operation.OperationID)))
	auth := &rollout.Step{
		Name: name, Type: "browser_authentication", Do: firstNonEmpty(operation.Summary, "Establish the browser session."),
		Source: doc.RelativePath, AuthenticationFlow: operation.OperationID, BrowserSession: defaultBrowserSessionName(doc),
	}
	if action != nil {
		action.BrowserSession = auth.BrowserSession
	}
	index := len(session.Intent.Steps)
	for i, step := range session.Intent.Steps {
		if step == action {
			index = i
			break
		}
	}
	session.Intent.Steps = append(session.Intent.Steps, nil)
	copy(session.Intent.Steps[index+1:], session.Intent.Steps[index:])
	session.Intent.Steps[index] = auth
	session.BrowserRoute = "browser"
	session.BrowserSession = "none"
	return auth
}

func uniqueIntentStepName(steps []*rollout.Step, base string) string {
	base = strings.Trim(slugIdent(base), "_")
	if base == "" {
		base = "authenticate_browser"
	}
	used := map[string]bool{}
	for _, step := range steps {
		if step != nil {
			used[step.Name] = true
		}
	}
	if !used[base] {
		return base
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s_%d", base, index)
		if !used[candidate] {
			return candidate
		}
	}
}

func formatBrowserFlowSlots(values map[string][]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"=["+strings.Join(values[key], ",")+"]")
	}
	return strings.Join(parts, ";")
}
