package authoring

import (
	"github.com/OpenUdon/openudon/handoff"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
)

type ReviewState = handoff.ReviewState
type ReviewHandoff = handoff.ReviewHandoff
type ReviewHandoffInput = handoff.ReviewHandoffInput
type ReviewApprovalState = handoff.ReviewApprovalState
type ReviewOwnerSplit = handoff.ReviewOwnerSplit
type ReviewExecutionPolicy = handoff.ReviewExecutionPolicy
type ReviewCredentialBindings = handoff.ReviewCredentialBindings
type ReviewTrustedRunner = handoff.ReviewTrustedRunner
type ReviewHandoffOptions = handoff.ReviewHandoffOptions
type ReviewHandoffValidationOptions = handoff.ReviewHandoffValidationOptions

const ReviewHandoffVersion = handoff.ReviewHandoffVersion
const LegacyReviewHandoffVersion = handoff.LegacyReviewHandoffVersion
const ReviewStateGenerated = handoff.ReviewStateGenerated
const ReviewStateValidated = handoff.ReviewStateValidated
const ReviewStateReviewRequired = handoff.ReviewStateReviewRequired
const ReviewStateApprovedForSandbox = handoff.ReviewStateApprovedForSandbox
const ReviewStateApprovedForProduction = handoff.ReviewStateApprovedForProduction
const ReviewStateRejected = handoff.ReviewStateRejected

func NewReviewHandoff(options ReviewHandoffOptions) ReviewHandoff {
	return handoff.NewReviewHandoff(options)
}
func DefaultReviewStateMachine() []ReviewApprovalState { return handoff.DefaultReviewStateMachine() }
func DefaultReviewExecutionPolicy(sideEffectful bool) ReviewExecutionPolicy {
	return handoff.DefaultReviewExecutionPolicy(sideEffectful)
}
func ReviewStateNames() []string { return handoff.ReviewStateNames() }
func ReviewStateMachineHasRequiredStates(states []ReviewApprovalState) bool {
	return handoff.ReviewStateMachineHasRequiredStates(states)
}
func ValidateReviewHandoff(manifest ReviewHandoff, options ...ReviewHandoffValidationOptions) []Diagnostic {
	return handoff.ValidateReviewHandoff(manifest, options...)
}
func ReviewHandoffSelfDigest(manifest ReviewHandoff, path string) (string, error) {
	return handoff.ReviewHandoffSelfDigest(manifest, path)
}

func cleanReviewHandoffInputPath(path string) (string, bool) {
	clean, err := packageartifacts.CleanRelativePath(path)
	return clean, err == nil
}
