package browserauthoring

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authprofile"
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
	// Capture applies these native textual normalizations before emitting its
	// review. Preserve the exact start hash while matching that native policy.
	cfg.ProfileID = strings.TrimSpace(cfg.ProfileID)
	cfg.GoalRole = strings.ToLower(strings.TrimSpace(cfg.GoalRole))
	cfg.GoalContext = strings.TrimSpace(cfg.GoalContext)
	cfg.GoalLabel = strings.TrimSpace(cfg.GoalLabel)
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
	origins, err := normalizeBrowserAuthoringOrigins(cfg.Origins)
	if err != nil {
		return nil, invalid
	}
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
	// The public review retains the goal predicate; the native authentication
	// recipe independently retains the original login and dashboard proof.
	// Match both against the approved start rather than trusting a resealed
	// receipt hash to describe the same captured policy.
	auth, err := authprofile.Parse(authentication)
	if err != nil {
		return nil, invalid
	}
	flow, ok := auth.Flows["authenticated_goal"]
	login, _, err := normalizeBrowserAuthoringURL(cfg.URL)
	if err != nil || !ok || len(flow.Sequence) == 0 {
		return nil, invalid
	}
	initial := flow.Sequence[0].Navigate
	if flow.Sequence[0].NavigateTarget != nil {
		initial = flow.Sequence[0].NavigateTarget.URL
	}
	initial, _, err = normalizeBrowserAuthoringURL(initial)
	if err != nil || initial != login {
		return nil, invalid
	}
	dashboardOrigin, dashboardPath, err := originAndPath(cfg.DashboardURL)
	if err != nil || flow.Success.Origin != dashboardOrigin || flow.Success.Path != dashboardPath {
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
