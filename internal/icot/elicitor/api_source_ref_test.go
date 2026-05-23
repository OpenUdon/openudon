package elicitor

import (
	"testing"

	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

func TestAPISourceRefsPreferGenericSource(t *testing.T) {
	session := Session{Intent: rollout.Intent{
		Source:  "google-discovery/gmail.json",
		OpenAPI: "openapi/legacy.yaml",
	}}
	step := &rollout.Step{
		Source:  "aws-smithy/lambda.json",
		OpenAPI: "openapi/step.yaml",
	}

	if got := intentAPISourceRef(session.Intent); got != "google-discovery/gmail.json" {
		t.Fatalf("intentAPISourceRef = %q", got)
	}
	if got := stepAPISourceRef(session, step); got != "aws-smithy/lambda.json" {
		t.Fatalf("stepAPISourceRef = %q", got)
	}
}

func TestAPISourceRefsFallBackToOpenAPI(t *testing.T) {
	session := Session{Intent: rollout.Intent{OpenAPI: "openapi/root.yaml"}}

	if got := intentAPISourceRef(session.Intent); got != "openapi/root.yaml" {
		t.Fatalf("intentAPISourceRef = %q", got)
	}
	if got := stepAPISourceRef(session, &rollout.Step{}); got != "openapi/root.yaml" {
		t.Fatalf("stepAPISourceRef = %q", got)
	}
}

func TestSetAPISourceFromDocPreservesOpenAPIAliasOnlyForOpenAPI(t *testing.T) {
	var session Session
	setIntentAPISourceFromDoc(&session, APIDocument{RelativePath: "google-discovery/gmail.json"})
	if session.Intent.Source != "google-discovery/gmail.json" || session.Intent.OpenAPI != "" {
		t.Fatalf("discovery intent source/openapi = %q/%q", session.Intent.Source, session.Intent.OpenAPI)
	}
	setIntentAPISourceFromDoc(&session, APIDocument{RelativePath: "openapi/weather.yaml"})
	if session.Intent.Source != "openapi/weather.yaml" || session.Intent.OpenAPI != "openapi/weather.yaml" {
		t.Fatalf("openapi intent source/openapi = %q/%q", session.Intent.Source, session.Intent.OpenAPI)
	}

	var step rollout.Step
	setStepAPISourceFromDoc(&step, APIDocument{RelativePath: "aws-smithy/s3.json"})
	if step.Source != "aws-smithy/s3.json" || step.OpenAPI != "" {
		t.Fatalf("smithy step source/openapi = %q/%q", step.Source, step.OpenAPI)
	}
	setStepAPISourceFromDoc(&step, APIDocument{RelativePath: "openapi/support.yaml"})
	if step.Source != "openapi/support.yaml" || step.OpenAPI != "openapi/support.yaml" {
		t.Fatalf("openapi step source/openapi = %q/%q", step.Source, step.OpenAPI)
	}
}
