// Package presentation derives the shared value-free review surface used by
// browser-transaction frontends. It adds disclosure, never authority.
package presentation

import (
	"github.com/OpenUdon/browsertools/registrationauthor"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/browsertransaction/engine"
)

// Resource augments the exact engine snapshot only with deterministic,
// value-free review disclosures. It does not change the transaction v1 wire.
type Resource struct {
	engine.Snapshot
	Review *Review `json:"review,omitempty"`
}

type Review struct {
	Composition           string                                 `json:"composition"`
	Origins               []string                               `json:"origins"`
	ObservedAt            string                                 `json:"observed_at"`
	ExpiresAt             string                                 `json:"expires_at"`
	FreshnessCheck        string                                 `json:"freshness_check"`
	CredentialBindings    []browsertransaction.CredentialBinding `json:"credential_bindings"`
	Session               string                                 `json:"session,omitempty"`
	RegistrationAuthoring *RegistrationAuthoringDisclosure       `json:"registration_authoring,omitempty"`
}

type RegistrationAuthoringDisclosure struct {
	AccessibilityLabels       string   `json:"accessibility_labels"`
	ObservationStatus         string   `json:"observation_status"`
	NetworkMethods            []string `json:"network_methods"`
	MutationRequestsAllowed   bool     `json:"mutation_requests_allowed"`
	SubmitSupported           bool     `json:"submit_supported"`
	AccountAttemptSupported   bool     `json:"account_attempt_supported"`
	SessionEstablishment      bool     `json:"session_establishment_supported"`
	RuntimeSupported          bool     `json:"runtime_supported"`
	ApprovalSymbol            string   `json:"approval_symbol"`
	ApprovalSymbolIsAuthority bool     `json:"approval_symbol_is_authority"`
}

// New returns the deterministic frontend resource for one defensive engine
// snapshot. Candidate bodies, paths, credentials, and runtime handles have no
// representation here.
func New(snapshot engine.Snapshot) Resource {
	resource := Resource{Snapshot: snapshot}
	if snapshot.Transaction == nil {
		return resource
	}
	transaction := snapshot.Transaction
	review := &Review{
		Composition: "BAP+BCP", Origins: append([]string(nil), transaction.Provenance.Origins...),
		ObservedAt: transaction.Provenance.ObservedAt, ExpiresAt: transaction.Provenance.ExpiresAt,
		FreshnessCheck:     "engine_rechecks_expires_at_before_review_and_prepare",
		CredentialBindings: append(make([]browsertransaction.CredentialBinding, 0, len(transaction.CredentialBindings)), transaction.CredentialBindings...),
		Session:            transaction.Session,
	}
	if transaction.Kind == browsertransaction.KindRegistration {
		review.Composition = "BRP"
		review.RegistrationAuthoring = &RegistrationAuthoringDisclosure{
			AccessibilityLabels: registrationauthorsession.AccessibilityLabelDisclosure,
			ObservationStatus:   "producer_accepted",
			NetworkMethods:      []string{"GET", "HEAD"}, MutationRequestsAllowed: false,
			SubmitSupported: false, AccountAttemptSupported: false, SessionEstablishment: false, RuntimeSupported: false,
			ApprovalSymbol: registrationauthor.ApprovalSymbol, ApprovalSymbolIsAuthority: false,
		}
	}
	resource.Review = review
	return resource
}
