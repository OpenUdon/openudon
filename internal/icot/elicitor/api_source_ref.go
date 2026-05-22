package elicitor

import (
	"strings"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func intentAPISourceRef(intent rollout.Intent) string {
	return strings.TrimSpace(firstNonEmpty(intent.Source, intent.OpenAPI))
}

func stepAPISourceRef(session Session, step *rollout.Step) string {
	if step == nil {
		return intentAPISourceRef(session.Intent)
	}
	return strings.TrimSpace(firstNonEmpty(step.Source, step.OpenAPI, session.Intent.Source, session.Intent.OpenAPI))
}

func setIntentAPISourceFromDoc(session *Session, doc APIDocument) {
	if session == nil || strings.TrimSpace(doc.RelativePath) == "" {
		return
	}
	session.Intent.Source = doc.RelativePath
	if isOpenAPIDocument(doc) {
		session.Intent.OpenAPI = doc.RelativePath
	}
}

func setStepAPISourceFromDoc(step *rollout.Step, doc APIDocument) {
	if step == nil || strings.TrimSpace(doc.RelativePath) == "" {
		return
	}
	step.Source = doc.RelativePath
	if isOpenAPIDocument(doc) {
		step.OpenAPI = doc.RelativePath
	}
}
