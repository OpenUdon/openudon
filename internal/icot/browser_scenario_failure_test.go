package icot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/internal/icot/browserauthor"
)

func TestScenarioAuthorFailureMetadataExcludesPrivateValues(t *testing.T) {
	const canary = "credential-token-page-canary"
	err := scenarioAuthorStageError("result_import", errors.New(canary))
	diagnostic, ok := BrowserScenarioFailureDiagnostic(err)
	if !ok || diagnostic.Phase != "result_import" || diagnostic.Code != "operation_failed" || strings.Contains(err.Error(), canary) {
		t.Fatal("stage failure did not preserve closed metadata")
	}
	for _, code := range []string{canary, "token_canary_0123456789", "", "worker_protocol"} {
		err = scenarioControllerFailure("controller", browserauthor.Event{State: "failed", Phase: canary, ErrorCode: code}, errors.New(canary))
		diagnostic, ok = BrowserScenarioFailureDiagnostic(err)
		if !ok || !diagnostic.Valid() || strings.Contains(err.Error(), canary) || strings.Contains(err.Error(), "token_canary") {
			t.Fatal("worker failure reflected an unreviewed string")
		}
		if code == "worker_protocol" && diagnostic.Code != code {
			t.Fatal("closed worker code was lost")
		}
	}
	if (BrowserScenarioAuthorDiagnostic{Phase: canary, Code: "worker_protocol"}).Valid() || (BrowserScenarioAuthorDiagnostic{Phase: "controller", Code: canary}).Valid() {
		t.Fatal("unknown metadata vocabulary accepted")
	}
}

func TestScenarioAuthorFailurePreservesCancellationAndSpecificPhase(t *testing.T) {
	err := scenarioAuthorStageError("worker_start", context.Canceled)
	err = scenarioAuthorStageError("controller", err)
	diagnostic, ok := BrowserScenarioFailureDiagnostic(err)
	if !ok || diagnostic.Phase != "worker_start" || diagnostic.Code != "operation_canceled" || !errors.Is(err, context.Canceled) {
		t.Fatal("specific phase or cancellation cause was lost")
	}
	if _, ok := BrowserScenarioFailureDiagnostic(errors.New("unclassified private error")); ok {
		t.Fatal("arbitrary error became trusted metadata")
	}
}
