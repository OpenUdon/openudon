package elicitor

import (
	"strconv"
	"strings"

	"github.com/OpenUdon/apitools"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

const (
	readinessMissingBrowserRegistrationFlow     = "missing_browser_registration_flow"
	readinessInvalidBrowserRegistrationContract = "invalid_browser_registration_contract"
	readinessUnconfirmedBrowserRegistration     = "unconfirmed_browser_registration"
)

func isBrowserRegistrationOperationSummary(operation *apitools.OperationSummary) bool {
	return operation != nil && operation.Extensions["openudon.source_family"] == browserRegistrationSourceFamily
}

func isBrowserRegistrationDocument(doc APIDocument) bool {
	for index := range doc.Operations {
		if isBrowserRegistrationOperationSummary(&doc.Operations[index]) {
			return true
		}
	}
	return false
}

func browserRegistrationOperationForStep(session Session, docs []APIDocument, step *rollout.Step) (APIDocument, *apitools.OperationSummary) {
	if step == nil {
		return APIDocument{}, nil
	}
	ref := stepAPISourceRef(session, step)
	for _, doc := range docs {
		if !isBrowserRegistrationDocument(doc) || doc.RelativePath != ref {
			continue
		}
		for index := range doc.Operations {
			operation := &doc.Operations[index]
			if operation.OperationID == strings.TrimSpace(step.RegistrationFlow) {
				copy := *operation
				return doc, &copy
			}
		}
	}
	return APIDocument{}, nil
}

func browserRegistrationReadinessIssues(session Session, docs []APIDocument, step *rollout.Step) []ReadinessIssue {
	if step == nil {
		return nil
	}
	name := firstNonEmpty(step.Name, "register")
	slot := "steps." + name
	add := func(code, suffix, message, suggested string) ReadinessIssue {
		return ReadinessIssue{Code: code, Slot: slot + suffix, Severity: readinessBlocking, Message: message, SuggestedAnswer: suggested}
	}
	_, operation := browserRegistrationOperationForStep(session, docs, step)
	if operation == nil {
		return []ReadinessIssue{add(readinessMissingBrowserRegistrationFlow, ".registration_flow", "Select the one exact reviewed no-submit browser registration flow.", suggestedBrowserRegistrationFlow(docs))}
	}
	if operation.Extensions["openudon.browser_registration.runtime_supported"] != "false" {
		return []ReadinessIssue{add(readinessInvalidBrowserRegistrationContract, ".registration_flow", "The selected registration source does not preserve fail-before-executor runtime policy.", suggestedBrowserRegistrationFlow(docs))}
	}
	expectedBindings, expectedTimeout, validContract := browserRegistrationOperationContract(operation)
	contractMatches := len(expectedBindings) > 0 && exactBrowserCredentialBindingMap(step.CredentialBindings, expectedBindings) &&
		strings.TrimSpace(step.BrowserSession) == "" && strings.TrimSpace(step.AuthenticationFlow) == "" && strings.TrimSpace(step.Operation) == "" &&
		step.DuplicatePrevention == operation.Extensions["openudon.browser_registration.duplicate_prevention"] &&
		step.OnDuplicate == operation.Extensions["openudon.browser_registration.on_duplicate"] &&
		step.AmbiguousOutcome == operation.Extensions["openudon.browser_registration.ambiguous_outcome"] &&
		step.CleanupDisposition == operation.Extensions["openudon.browser_registration.cleanup_disposition"] &&
		validContract && step.Timeout != nil && *step.Timeout == expectedTimeout
	if !contractMatches {
		return []ReadinessIssue{add(readinessInvalidBrowserRegistrationContract, ".registration_contract", "The registration call must preserve the reviewed symbolic bindings, zero-session posture, bounded timeout, and fixed duplicate, ambiguity, and cleanup policy.", suggestedBrowserRegistrationFlow(docs))}
	}
	if strings.TrimSpace(step.RegistrationApproval) != name {
		return []ReadinessIssue{add(readinessUnconfirmedBrowserRegistration, ".registration_approval", "Account creation requires explicit step-scoped authoring approval. Runtime execution remains unsupported and separately gated.", "approve "+name)}
	}
	return nil
}

func browserRegistrationStepFromOperation(doc APIDocument, operation *apitools.OperationSummary) *rollout.Step {
	if operation == nil || !isBrowserRegistrationOperationSummary(operation) {
		return nil
	}
	bindings, timeout, valid := browserRegistrationOperationContract(operation)
	if !valid {
		return nil
	}
	return &rollout.Step{
		Name: camelToSnake(firstNonEmpty(operation.OperationID, doc.ID, "register")), Type: "browser_registration",
		Do: firstNonEmpty(operation.Summary, "Create one account only after exact approval."), Source: doc.RelativePath,
		RegistrationFlow: operation.OperationID, CredentialBindings: bindings,
		DuplicatePrevention: operation.Extensions["openudon.browser_registration.duplicate_prevention"],
		OnDuplicate:         operation.Extensions["openudon.browser_registration.on_duplicate"],
		AmbiguousOutcome:    operation.Extensions["openudon.browser_registration.ambiguous_outcome"],
		CleanupDisposition:  operation.Extensions["openudon.browser_registration.cleanup_disposition"], Timeout: &timeout,
	}
}

func browserRegistrationOperationContract(operation *apitools.OperationSummary) (map[string]string, float64, bool) {
	if operation == nil || !isBrowserRegistrationOperationSummary(operation) {
		return nil, 0, false
	}
	flow := strings.TrimSpace(operation.OperationID)
	cleanup := operation.Extensions["openudon.browser_registration.cleanup_disposition"]
	if !browserBindingNamePattern.MatchString(flow) || operation.ID != flow || operation.Method != "BROWSER_REGISTRATION" || operation.Path != "#/flows/"+flow ||
		operation.Extensions["openudon.browser_registration.runtime_supported"] != "false" ||
		operation.Extensions["openudon.browser_registration.duplicate_prevention"] != "operator_attestation" ||
		operation.Extensions["openudon.browser_registration.on_duplicate"] != "fail" ||
		operation.Extensions["openudon.browser_registration.ambiguous_outcome"] != "stop_without_retry" ||
		(cleanup != "delete_separately" && cleanup != "retain_dedicated_test_identity") {
		return nil, 0, false
	}
	timeout, err := strconv.ParseFloat(operation.Extensions["openudon.browser_registration.timeout_seconds"], 64)
	if err != nil || timeout <= 0 || timeout > 600 {
		return nil, 0, false
	}
	bindings := browserRegistrationTransactionBindings(operation)
	if len(bindings) == 0 {
		return nil, 0, false
	}
	return bindings, timeout, true
}

func browserRegistrationTransactionBindings(operation *apitools.OperationSummary) map[string]string {
	if operation == nil {
		return nil
	}
	raw := strings.TrimSpace(operation.Extensions["openudon.browser_registration.credential_bindings"])
	result := map[string]string{}
	for _, item := range strings.Split(raw, ",") {
		parts := strings.Split(item, "=")
		if len(parts) != 2 || !browserBindingNamePattern.MatchString(parts[0]) || !browserBindingNamePattern.MatchString(parts[1]) || result[parts[0]] != "" {
			return nil
		}
		result[parts[0]] = parts[1]
	}
	required := strings.Split(strings.TrimSpace(operation.Extensions["openudon.browser_registration.credential_slots"]), ",")
	if len(required) == 1 && required[0] == "" {
		required = nil
	}
	if !exactBrowserCredentialBindings(result, required) {
		return nil
	}
	return result
}

func suggestedBrowserRegistrationFlow(docs []APIDocument) string {
	for _, doc := range docs {
		if isBrowserRegistrationDocument(doc) && len(doc.Operations) == 1 {
			return doc.RelativePath + "#" + doc.Operations[0].OperationID
		}
	}
	return ""
}

func selectBrowserRegistrationFlow(answer string, docs []APIDocument) (APIDocument, *apitools.OperationSummary) {
	answer = strings.TrimSpace(answer)
	for _, doc := range docs {
		if !isBrowserRegistrationDocument(doc) {
			continue
		}
		for index := range doc.Operations {
			operation := &doc.Operations[index]
			if answer != operation.OperationID && answer != doc.RelativePath+"#"+operation.OperationID {
				continue
			}
			copy := *operation
			return doc, &copy
		}
	}
	return APIDocument{}, nil
}

func replaceBrowserRegistrationStep(session *Session, target *rollout.Step, doc APIDocument, operation *apitools.OperationSummary) *rollout.Step {
	if session == nil || target == nil {
		return nil
	}
	replacement := browserRegistrationStepFromOperation(doc, operation)
	if replacement == nil {
		return nil
	}
	name, description := target.Name, target.Do
	*target = *replacement
	target.Name = firstNonEmpty(name, replacement.Name)
	target.Do = firstNonEmpty(description, replacement.Do)
	mergeBrowserRegistrationCredentials(session, target)
	return target
}

func mergeBrowserRegistrationCredentials(session *Session, step *rollout.Step) {
	if session == nil || step == nil {
		return
	}
	for _, binding := range step.CredentialBindings {
		session.Credentials = append(session.Credentials, binding)
	}
	session.Credentials = dedupeStrings(session.Credentials)
	session.CredentialsSet = len(session.Credentials) > 0
	session.BrowserRoute = "browser"
	session.BrowserSession = "none"
}
