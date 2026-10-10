// Package browsertransaction preserves the retained CLI through the supported public pure verifier.
package browsertransaction

import (
	public "github.com/OpenUdon/openudon/browsertransaction"
)

const (
	VersionV1                      = public.VersionV1
	VersionV2                      = public.VersionV2
	VersionV3                      = public.VersionV3
	VersionV4                      = public.VersionV4
	Version                        = public.Version
	MaxBytes                       = public.MaxBytes
	ResultAuthenticatedAuthoringV2 = public.ResultAuthenticatedAuthoringV2
	ResultRegistrationAuthoringV1  = public.ResultRegistrationAuthoringV1
	ResultRegistrationAuthoringV2  = public.ResultRegistrationAuthoringV2
	ResultRegistrationAuthoringV3  = public.ResultRegistrationAuthoringV3
	ResultRegistrationAuthoringV4  = public.ResultRegistrationAuthoringV4
	KindAuthenticationCapability   = public.KindAuthenticationCapability
	KindRegistration               = public.KindRegistration
	CandidateAuthentication        = public.CandidateAuthentication
	CandidateCapability            = public.CandidateCapability
	CandidateRegistration          = public.CandidateRegistration
	StateCandidate                 = public.StateCandidate
	StateReviewed                  = public.StateReviewed
	StatePrepared                  = public.StatePrepared
	StatePromoted                  = public.StatePromoted
	StateCancelled                 = public.StateCancelled
	StateFailed                    = public.StateFailed
	StateIndeterminate             = public.StateIndeterminate
	FailureRejected                = public.FailureRejected
	FailureConflict                = public.FailureConflict
	FailureOperational             = public.FailureOperational
	FailureIndeterminate           = public.FailureIndeterminate
	FailureTransactionInvalid      = public.FailureTransactionInvalid
	FailureCandidateInvalid        = public.FailureCandidateInvalid
	FailureCandidateStale          = public.FailureCandidateStale
	FailureDigestMismatch          = public.FailureDigestMismatch
	FailureReviewRejected          = public.FailureReviewRejected
	FailureWorkspaceConflict       = public.FailureWorkspaceConflict
	FailurePreparationFailed       = public.FailurePreparationFailed
	FailureQualificationFailed     = public.FailureQualificationFailed
	FailurePromotionFailed         = public.FailurePromotionFailed
	FailurePromotionIndeterminate  = public.FailurePromotionIndeterminate
)

type Kind = public.Kind
type CandidateKind = public.CandidateKind
type State = public.State
type FailureClass = public.FailureClass
type FailureCode = public.FailureCode
type Candidate = public.Candidate
type Provenance = public.Provenance
type CredentialBinding = public.CredentialBinding
type Preparation = public.Preparation
type Promotion = public.Promotion
type Failure = public.Failure
type Transaction = public.Transaction

func CanonicalBytes(t Transaction) ([]byte, error) { return public.CanonicalBytes(t) }
func Digest(t Transaction) (string, error)         { return public.Digest(t) }
func Decode(data []byte) (Transaction, error)      { return public.Decode(data) }
func DecodeVerified(data []byte, expected string) (Transaction, error) {
	return public.DecodeVerified(data, expected)
}
func ValidateTransition(previous, next Transaction) error {
	return public.ValidateTransition(previous, next)
}
