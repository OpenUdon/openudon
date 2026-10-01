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
	expectedSession := browserAuthenticationTransactionSession(operation)
	if strings.TrimSpace(step.BrowserSession) == "" || expectedSession != "" && strings.TrimSpace(step.BrowserSession) != expectedSession {
		add(readinessMissingBrowserAuthenticationSession, ".browser_session", "Name the exact execution-local browser session established by this authentication step.", defaultBrowserSessionName(doc, operation))
	}
	required := browserAuthenticationCredentialSlots(operation)
	expectedBindings := browserAuthenticationTransactionBindings(operation)
	bindingsMatch := exactBrowserCredentialBindings(step.CredentialBindings, required)
	if len(expectedBindings) > 0 {
		bindingsMatch = exactBrowserCredentialBindingMap(step.CredentialBindings, expectedBindings)
	}
	if !bindingsMatch {
		suggested := suggestedBrowserCredentialBindings(required)
		if len(expectedBindings) > 0 {
			suggested = formatBrowserCredentialBindingMap(expectedBindings)
		}
		add(readinessMissingBrowserCredentialBindings, ".credential_bindings", "Map every credential slot required by the selected flow to its exact symbolic runtime binding. Do not enter credential values.", suggested)
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

func defaultBrowserSessionName(doc APIDocument, operation *apitools.OperationSummary) string {
	if session := browserAuthenticationTransactionSession(operation); session != "" {
		return session
	}
	name := slugIdent(firstNonEmpty(doc.ID, doc.Title, "browser"))
	if name == "" {
		name = "browser"
	}
	return name + "_session"
}

func browserAuthenticationTransactionSession(operation *apitools.OperationSummary) string {
	if operation == nil {
		return ""
	}
	session := strings.TrimSpace(operation.Extensions["openudon.browser_authentication.session"])
	if !browserBindingNamePattern.MatchString(session) {
		return ""
	}
	return session
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
		Source: doc.RelativePath, AuthenticationFlow: operation.OperationID, BrowserSession: defaultBrowserSessionName(doc, operation),
	}
	if bindings := browserAuthenticationTransactionBindings(operation); len(bindings) > 0 {
		auth.CredentialBindings = bindings
		for _, binding := range bindings {
			session.Credentials = append(session.Credentials, binding)
		}
		session.Credentials = dedupeStrings(session.Credentials)
		session.CredentialsSet = true
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

func browserAuthenticationTransactionBindings(operation *apitools.OperationSummary) map[string]string {
	if operation == nil {
		return nil
	}
	raw := strings.TrimSpace(operation.Extensions["openudon.browser_authentication.credential_bindings"])
	if raw == "" {
		return nil
	}
	result := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		parts := strings.Split(item, "=")
		if len(parts) != 2 || !browserBindingNamePattern.MatchString(parts[0]) || !browserBindingNamePattern.MatchString(parts[1]) || result[parts[0]] != "" {
			return nil
		}
		result[parts[0]] = parts[1]
	}
	if !exactBrowserCredentialBindings(result, browserAuthenticationCredentialSlots(operation)) {
		return nil
	}
	return result
}

func exactBrowserCredentialBindingMap(actual, expected map[string]string) bool {
	if len(actual) != len(expected) {
		return false
	}
	for slot, binding := range expected {
		if actual[slot] != binding {
			return false
		}
	}
	return true
}

func formatBrowserCredentialBindingMap(bindings map[string]string) string {
	slots := make([]string, 0, len(bindings))
	for slot := range bindings {
		slots = append(slots, slot)
	}
	sort.Strings(slots)
	values := make([]string, 0, len(slots))
	for _, slot := range slots {
		values = append(values, slot+"="+bindings[slot])
	}
	return strings.Join(values, ", ")
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
