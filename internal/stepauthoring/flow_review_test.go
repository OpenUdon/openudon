package stepauthoring

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/elicitor"
)

type fakeDraftReviewer struct {
	response elicitor.DraftReviewResponse
	err      error
	call     func(context.Context, elicitor.DraftReviewRequest)
}

func (f fakeDraftReviewer) ReviewDraft(ctx context.Context, request elicitor.DraftReviewRequest) (elicitor.DraftReviewResponse, error) {
	if f.call != nil {
		f.call(ctx, request)
	}
	return f.response, f.err
}

func TestReviewFlowMatchesPublishedRunnableFixture(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1")
	var request FlowReviewRequest
	readFlowReviewFixture(t, filepath.Join(root, "requests", "flow-review.json"), &request)
	outcome := ReviewFlow(context.Background(), filepath.Join(root, "example"), request, nil)
	if outcome.ExitCode != 0 {
		t.Fatalf("outcome exit=%d: %#v", outcome.ExitCode, outcome.Result)
	}
	assertFlowReviewResultFixture(t, outcome.Result, filepath.Join(root, "results", "flow-review.json"))
}

func TestReviewFlowDistinguishesMissingAndUnsupportedVersions(t *testing.T) {
	request := FlowReviewRequest{Kind: "request", Command: FlowReviewCommand, ModelReview: &FlowReviewModelRequest{}}
	missing := ReviewFlow(nil, t.TempDir(), request, nil)
	if missing.Result.Diagnostics[0].Code != "request.invalid" {
		t.Fatalf("missing version diagnostic = %#v", missing.Result.Diagnostics)
	}
	request.Version = "openudon.step-authoring.v99"
	unsupported := ReviewFlow(nil, t.TempDir(), request, nil)
	if unsupported.Result.Diagnostics[0].Code != "request.unsupported_version" {
		t.Fatalf("unsupported version diagnostic = %#v", unsupported.Result.Diagnostics)
	}
}

func TestReviewFlowUsesConfiguredFakeAndPublishesOnlySafeAdvisory(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = &FlowReviewModelRequest{Enabled: true, Provider: stringPointer("openai"), Model: stringPointer("gpt-test")}
	const finding = "The notification step does not consume the prior summary."
	reviewer := fakeDraftReviewer{response: elicitor.DraftReviewResponse{Issues: []elicitor.DraftReviewIssue{{
		Severity: "error", Code: "disconnected_notification", Message: finding,
		Slot: "steps.notify.with.message", SuggestedAnswer: "message=summary.received_body.text",
		Evidence: "unreturned evidence", GapKind: "disconnected_notification", RemediationAction: "comment_only",
	}}}, call: func(_ context.Context, got elicitor.DraftReviewRequest) {
		if len(got.Credentials) != 0 {
			t.Errorf("model request contains credential names: %#v", got.Credentials)
		}
		if len(got.Steps) != 1 || got.Steps[0].Name != "list_projects" {
			t.Errorf("model request steps = %#v", got.Steps)
		}
		if len(got.Steps) == 1 && (got.Steps[0].Source != "" || got.Steps[0].Operation.Provenance != "") {
			t.Errorf("model request contains local source identity: %#v", got.Steps[0])
		}
	}}
	outcome := ReviewFlow(context.Background(), root, request, func(provider, model string) (DraftReviewer, error) {
		if provider != "openai" || model != "gpt-test" {
			t.Fatalf("factory received provider/model %q/%q", provider, model)
		}
		return reviewer, nil
	})
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result == nil {
		t.Fatalf("outcome = %#v", outcome)
	}
	got := outcome.Result.Result.ModelReview
	if got.Status != "completed" || len(got.Findings) != 1 || got.Findings[0].Message != finding || got.Findings[0].Severity != "warning" {
		t.Fatalf("model review = %#v", got)
	}
	assertFlowReviewResultFixture(t, outcome.Result, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "results", "flow-review-model-finding.json"))
	encoded, _ := json.Marshal(outcome.Result)
	for _, forbidden := range []string{"unreturned evidence", "suggested_answer", "gpt-test", "project_api_key"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Errorf("result disclosed %q: %s", forbidden, encoded)
		}
	}
}

func TestReviewFlowReportsUnavailableProviderWithoutLeakingError(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = &FlowReviewModelRequest{Enabled: true, Provider: stringPointer("openai"), Model: stringPointer("gpt-test")}
	outcome := ReviewFlow(context.Background(), root, request, func(string, string) (DraftReviewer, error) {
		return nil, errors.New("OPENAI_API_KEY=do-not-disclose-this-provider-error")
	})
	if outcome.ExitCode != 0 || outcome.Result.Status != "completed" || outcome.Result.Result.ModelReview.Status != "unavailable" {
		t.Fatalf("outcome = %#v", outcome)
	}
	assertFlowReviewResultFixture(t, outcome.Result, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "results", "flow-review-unavailable.json"))
	encoded, _ := json.Marshal(outcome.Result)
	if strings.Contains(string(encoded), "do-not-disclose") {
		t.Fatalf("provider setup error leaked: %s", encoded)
	}
}

func TestReviewFlowDoesNotPresentModelFailureAsPass(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = &FlowReviewModelRequest{Enabled: true, Provider: stringPointer("openai"), Model: stringPointer("gpt-test")}
	outcome := ReviewFlow(context.Background(), root, request, func(string, string) (DraftReviewer, error) {
		return fakeDraftReviewer{err: errors.New("provider failed with private output")}, nil
	})
	if outcome.ExitCode != 1 || outcome.Result.Status != "failed" || outcome.Result.Result.ModelReview.Status != "failed" {
		t.Fatalf("model failure was not explicit: %#v", outcome)
	}
	assertFlowReviewResultFixture(t, outcome.Result, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "results", "flow-review-failed.json"))
	encoded, _ := json.Marshal(outcome.Result)
	if strings.Contains(string(encoded), "private output") {
		t.Fatalf("provider error leaked: %s", encoded)
	}
}

func TestReviewFlowCancellationDuringModelCallIsBlocked(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = &FlowReviewModelRequest{Enabled: true, Provider: stringPointer("openai"), Model: stringPointer("gpt-test")}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	outcome := ReviewFlow(ctx, root, request, func(string, string) (DraftReviewer, error) {
		return fakeDraftReviewer{err: context.Canceled, call: func(context.Context, elicitor.DraftReviewRequest) { cancel() }}, nil
	})
	if outcome.ExitCode != 4 || outcome.Result.Status != "blocked" || outcome.Result.Result.ModelReview.Status != "failed" {
		t.Fatalf("cancelled model review = %#v", outcome)
	}
	assertFlowReviewResultFixture(t, outcome.Result, filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "results", "flow-review-cancelled.json"))
}

func TestReviewFlowRejectsUnsafeModelFinding(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = &FlowReviewModelRequest{Enabled: true, Provider: stringPointer("openai"), Model: stringPointer("gpt-test")}
	outcome := ReviewFlow(context.Background(), root, request, func(string, string) (DraftReviewer, error) {
		return fakeDraftReviewer{response: elicitor.DraftReviewResponse{Issues: []elicitor.DraftReviewIssue{{
			Code: "unsafe", Message: "Use token=ghp_abcdefghijklmnopqrstuvwxyz1234567890",
		}}}}, nil
	})
	if outcome.ExitCode != 1 || outcome.Result.Status != "failed" || outcome.Result.Result.ModelReview.Status != "failed" || len(outcome.Result.Result.ModelReview.Findings) != 0 {
		t.Fatalf("unsafe model finding was accepted: %#v", outcome)
	}
}

func TestReviewFlowRejectsStaleIntentAndPreservesBytes(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.IntentSHA256 = "sha256:" + strings.Repeat("0", 64)
	path := filepath.Join(root, "workflows", "intent.hcl")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	outcome := ReviewFlow(context.Background(), root, request, nil)
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.ExitCode != 3 || outcome.Result.Status != "conflict" || !reflect.DeepEqual(before, after) {
		t.Fatalf("stale review outcome=%#v, file changed=%t", outcome, !reflect.DeepEqual(before, after))
	}
}

func TestReviewFlowRequiresModelReviewField(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = nil
	outcome := ReviewFlow(context.Background(), root, request, nil)
	if outcome.ExitCode != 2 || outcome.Result.Status != "failed" || outcome.Result.Diagnostics[0].Code != "request.invalid" {
		t.Fatalf("missing model_review field was accepted: %#v", outcome)
	}
}

func TestReviewFlowRejectsProviderFieldsWhenModelReviewIsDisabled(t *testing.T) {
	root := copyFlowReviewExample(t)
	request := flowReviewRequestForExample(t, root)
	request.ModelReview.Provider = stringPointer("")
	outcome := ReviewFlow(context.Background(), root, request, nil)
	if outcome.ExitCode != 2 || outcome.Result.Status != "failed" || outcome.Result.Diagnostics[0].Code != "request.invalid" {
		t.Fatalf("disabled model review accepted an empty provider property: %#v", outcome)
	}
}

func TestValidFlowReviewJSONPreservesRequiredOptionPresence(t *testing.T) {
	tests := []struct {
		name string
		json string
		want bool
	}{
		{name: "local review", json: `{"model_review":{"enabled":false}}`, want: true},
		{name: "explicit model review", json: `{"model_review":{"enabled":true,"provider":"openai","model":"gpt-test"}}`, want: true},
		{name: "missing options", json: `{}`, want: false},
		{name: "null options", json: `{"model_review":null}`, want: false},
		{name: "null enabled", json: `{"model_review":{"enabled":null}}`, want: false},
		{name: "empty disabled provider", json: `{"model_review":{"enabled":false,"provider":""}}`, want: false},
		{name: "null disabled model", json: `{"model_review":{"enabled":false,"model":null}}`, want: false},
		{name: "missing enabled provider", json: `{"model_review":{"enabled":true,"model":"gpt-test"}}`, want: false},
		{name: "null enabled provider", json: `{"model_review":{"enabled":true,"provider":null,"model":"gpt-test"}}`, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ValidFlowReviewRequestJSON([]byte(test.json)); got != test.want {
				t.Fatalf("valid=%t, want %t", got, test.want)
			}
		})
	}
}

func TestReviewFlowDoesNotSendCredentialShapedSourceContext(t *testing.T) {
	root := copyFlowReviewExample(t)
	sourcePath := filepath.Join(root, "openapi", "project-api.yaml")
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	unsafeDescription := "description: ghp_abcdefghijklmnopqrstuvwxyz1234567890\n"
	updated := strings.Replace(string(source), "      summary: List projects.\n", "      summary: List projects.\n      "+unsafeDescription, 1)
	if updated == string(source) {
		t.Fatal("could not add credential-shaped operation context")
	}
	if err := os.WriteFile(sourcePath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = &FlowReviewModelRequest{Enabled: true, Provider: stringPointer("openai"), Model: stringPointer("gpt-test")}
	factoryCalled := false
	outcome := ReviewFlow(context.Background(), root, request, func(string, string) (DraftReviewer, error) {
		factoryCalled = true
		return fakeDraftReviewer{}, nil
	})
	if factoryCalled || outcome.Result.Result.ModelReview.Status != "unavailable" {
		t.Fatalf("credential-shaped context reached model factory: called=%t result=%#v", factoryCalled, outcome)
	}
	encoded, _ := json.Marshal(outcome.Result)
	if strings.Contains(string(encoded), "ghp_") {
		t.Fatalf("credential-shaped source text leaked: %s", encoded)
	}
}

func TestReviewFlowSkipsOversizedModelContext(t *testing.T) {
	root := copyFlowReviewExample(t)
	intentPath := filepath.Join(root, "workflows", "intent.hcl")
	intent, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(intent), "A local example for deterministic step-check conformance.", strings.Repeat("a", MaxModelReviewInputBytes+1), 1)
	if updated == string(intent) {
		t.Fatal("could not prepare large review context")
	}
	if err := os.WriteFile(intentPath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	request := flowReviewRequestForExample(t, root)
	request.ModelReview = &FlowReviewModelRequest{Enabled: true, Provider: stringPointer("openai"), Model: stringPointer("gpt-test")}
	factoryCalled := false
	outcome := ReviewFlow(context.Background(), root, request, func(string, string) (DraftReviewer, error) {
		factoryCalled = true
		return fakeDraftReviewer{}, nil
	})
	if factoryCalled || outcome.Result.Result.ModelReview.Status != "unavailable" || len(outcome.Result.Diagnostics) == 0 || outcome.Result.Diagnostics[0].Code != "model_review.context_too_large" {
		t.Fatalf("oversized model context was not bounded: called=%t outcome=%#v", factoryCalled, outcome)
	}
}

func flowReviewRequestForExample(t *testing.T, root string) FlowReviewRequest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, "workflows", "intent.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	return FlowReviewRequest{Version: WireVersion, Kind: "request", Command: FlowReviewCommand, IntentSHA256: sourceDigest(data), ModelReview: &FlowReviewModelRequest{Enabled: false}}
}

func stringPointer(value string) *string { return &value }

func copyFlowReviewExample(t *testing.T) string {
	t.Helper()
	fixture := filepath.Join("..", "..", "docs", "examples", "step-authoring", "v1", "example")
	root := t.TempDir()
	for _, relative := range []string{"workflows/intent.hcl", "openapi/project-api.yaml"} {
		data, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(relative)))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func readFlowReviewFixture(t *testing.T, path string, target any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		t.Fatal(err)
	}
}

func assertFlowReviewResultFixture(t *testing.T, actual FlowReviewWireResult, path string) {
	t.Helper()
	got, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	var want any
	var gotValue any
	readFlowReviewFixture(t, path, &want)
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gotValue, want) {
		t.Fatalf("flow-review result differs from fixture\ngot:  %s\nwant: %#v", got, want)
	}
}
