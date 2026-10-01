package browserauthoring

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"time"

	"github.com/OpenUdon/openudon/internal/browsercandidate"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

// ImportedAuthenticationCandidate reconstructs the already approved public
// capture using the existing native review/candidate types. No private result,
// worker, UI, writer, or implicit human approval is involved.
func ImportedAuthenticationCandidate(cfg LiveConfig, transaction browsertransaction.Transaction, authentication, capability, reviews []byte, at time.Time) (*browsercandidate.AuthenticationCapability, error) {
	invalid := errors.New("imported authentication evidence invalid")
	var collection authenticatedAuthoringReviewCollection
	if len(reviews) > liveAuthorResultMaxBytes || evidencefile.DecodeStrict(reviews, &collection) != nil || collection.Version != authenticatedAuthoringReviewVersion {
		return nil, invalid
	}
	var review *authenticatedAuthoringSafeReview
	for i := range collection.Captures {
		if collection.Captures[i].ProfileID == transaction.ID {
			if review != nil {
				return nil, invalid
			}
			review = &collection.Captures[i]
		}
	}
	goalOrigin, goalPath, err := originAndPath(liveGoalURL(cfg))
	if err != nil {
		return nil, invalid
	}
	label := cfg.GoalLabel
	if label == "" {
		label = defaultLiveGoalLabel(goalPath)
	}
	want := liveGoalPredicate{Origin: goalOrigin, Path: goalPath, Context: cfg.GoalContext, Role: cfg.GoalRole, Label: label}
	if review == nil || review.Version != authenticatedAuthoringReviewVersion || review.ProfileID != cfg.ProfileID || review.EnvelopeSHA256 != transaction.Provenance.ResultSHA256 || review.ObservedAt != transaction.Provenance.ObservedAt || review.Goal != cfg.Goal || !reflect.DeepEqual(review.GoalPredicate, want) || review.AuthenticationTarget != filepath.ToSlash(filepath.Join("browser-authentication", transaction.ID+"-auth.json")) || review.CapabilityTarget != filepath.ToSlash(filepath.Join("browser-profiles", transaction.ID+".json")) {
		return nil, invalid
	}
	origins := slices.Clone(cfg.Origins)
	slices.Sort(origins)
	reviewOrigins := slices.Clone(review.Origins)
	slices.Sort(reviewOrigins)
	transactionOrigins := slices.Clone(transaction.Provenance.Origins)
	slices.Sort(transactionOrigins)
	if !slices.Equal(origins, transactionOrigins) || !slices.Equal(origins, reviewOrigins) {
		return nil, invalid
	}
	authentication, err = browsercandidate.CanonicalSourceBytes(authentication)
	if err != nil {
		return nil, invalid
	}
	capability, err = browsercandidate.CanonicalSourceBytes(capability)
	if err != nil {
		return nil, invalid
	}
	authReview, err := json.Marshal(review.AuthenticationReview)
	if err != nil {
		return nil, invalid
	}
	capReview, err := json.Marshal(review.CapabilityReview)
	if err != nil {
		return nil, invalid
	}
	candidate, err := browsercandidate.ComposeAuthenticationCapability(browsercandidate.AuthenticationCapabilityRequest{
		TransactionID: transaction.ID, Flow: "authenticated_goal", Session: transaction.Session, CredentialBindings: transaction.CredentialBindings,
		Authentication: authentication, AuthenticationReview: authReview, Capability: capability, CapabilityReview: capReview,
		ResultSHA256: transaction.Provenance.ResultSHA256, ObservedAt: transaction.Provenance.ObservedAt, Origins: transaction.Provenance.Origins, AssessedAt: at,
	})
	if err != nil {
		return nil, invalid
	}
	reviewed, err := candidate.ReviewedTransaction()
	if err != nil {
		return nil, invalid
	}
	actual, err := browsertransaction.Digest(reviewed)
	if err != nil {
		return nil, invalid
	}
	expected, err := browsertransaction.Digest(transaction)
	if err != nil || actual != expected {
		return nil, invalid
	}
	return candidate, nil
}
