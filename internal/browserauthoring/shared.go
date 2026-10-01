package browserauthoring

import (
	"io"
	"time"
)

// PlanInput and Plan are the shared inert browser-authoring handoff contract.
type PlanInput = browserAuthoringPlanInput
type Plan = browserAuthoringPlan

// RepeatedFlag retains the terminal's reviewed repeated-string parsing.
type RepeatedFlag = repeatedFlag

func OptionalPlan(input PlanInput) (*Plan, error) { return optionalBrowserAuthoringPlan(input) }
func ClonePlan(plan *Plan) *Plan                  { return cloneBrowserAuthoringPlan(plan) }
func ExampleDirForPlan(value string) string       { return exampleDirForPlan(value) }
func ProviderFromEnv() string                     { return providerFromEnv() }
func FirstNonEmpty(values ...string) string       { return firstNonEmpty(values...) }

// RunWorker dispatches only the existing closed worker/doctor invocations.
func RunWorker(args []string, in io.Reader, out, errOut io.Writer) int {
	return runBundledBrowserWorker(args, in, out, errOut)
}

// RunLive and RunPlan preserve the retained terminal transports over shared
// browser authoring. Neither introduces new approvals or capture authority.
func RunLive(args []string, in io.Reader, out, errOut io.Writer) int {
	return runBrowserAuthorLive(args, in, out, errOut)
}
func RunPlan(args []string, out, errOut io.Writer) int {
	return runBrowserAuthoring(args, out, errOut)
}

// Capture preparation is shared with the retained UI transport. These aliases
// preserve the reviewed configuration and attestation without copying it.
type LiveConfig = liveAuthorConfig
type ProtocolResult = liveProtocolResult
type PreparedImport = preparedAuthenticatedImport

func InputProvided(input PlanInput) bool           { return browserAuthoringInputProvided(input) }
func NormalizeLiveConfig(config *LiveConfig) error { return normalizeLiveAuthorConfig(config) }
func PrepareAttestedImport(config LiveConfig, result ProtocolResult, at time.Time) (PreparedImport, error) {
	return prepareAttestedAuthenticatedAuthoringImport(config, result, at)
}
