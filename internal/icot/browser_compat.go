package icot

import (
	"context"
	"io"
	"time"

	"github.com/OpenUdon/openudon/internal/browserauthoring"
)

type browserAuthoringPlanInput = browserauthoring.PlanInput
type browserAuthoringPlan = browserauthoring.Plan
type repeatedFlag = browserauthoring.RepeatedFlag

func optionalBrowserAuthoringPlan(input browserAuthoringPlanInput) (*browserAuthoringPlan, error) {
	return browserauthoring.OptionalPlan(input)
}
func cloneBrowserAuthoringPlan(plan *browserAuthoringPlan) *browserAuthoringPlan {
	return browserauthoring.ClonePlan(plan)
}
func exampleDirForPlan(value string) string { return browserauthoring.ExampleDirForPlan(value) }
func providerFromEnv() string               { return browserauthoring.ProviderFromEnv() }
func firstNonEmpty(values ...string) string { return browserauthoring.FirstNonEmpty(values...) }
func runBundledBrowserWorker(args []string, in io.Reader, out, errOut io.Writer) int {
	return browserauthoring.RunWorker(args, in, out, errOut)
}
func runBrowserAuthorLive(args []string, in io.Reader, out, errOut io.Writer) int {
	return browserauthoring.RunLive(args, in, out, errOut)
}
func runBrowserAuthoring(args []string, out, errOut io.Writer) int {
	return browserauthoring.RunPlan(args, out, errOut)
}

// Retained internal callers use the same shared qualification contract.
type BrowserScenarioOutput = browserauthoring.BrowserScenarioOutput
type BrowserScenarioAuthorRequest = browserauthoring.BrowserScenarioAuthorRequest
type BrowserScenarioAuthorResult = browserauthoring.BrowserScenarioAuthorResult
type BrowserScenarioAuthorDiagnostic = browserauthoring.BrowserScenarioAuthorDiagnostic

func RunBrowserScenarioAuthor(ctx context.Context, request BrowserScenarioAuthorRequest) (BrowserScenarioAuthorResult, error) {
	return browserauthoring.RunBrowserScenarioAuthor(ctx, request)
}
func BrowserScenarioFailureDiagnostic(err error) (BrowserScenarioAuthorDiagnostic, bool) {
	return browserauthoring.BrowserScenarioFailureDiagnostic(err)
}

type liveAuthorConfig = browserauthoring.LiveConfig
type liveProtocolResult = browserauthoring.ProtocolResult

func browserAuthoringInputProvided(input browserAuthoringPlanInput) bool {
	return browserauthoring.InputProvided(input)
}
func normalizeLiveAuthorConfig(config *liveAuthorConfig) error {
	return browserauthoring.NormalizeLiveConfig(config)
}
func prepareAttestedAuthenticatedAuthoringImport(config liveAuthorConfig, result liveProtocolResult, at time.Time) (browserauthoring.PreparedImport, error) {
	return browserauthoring.PrepareAttestedImport(config, result, at)
}
