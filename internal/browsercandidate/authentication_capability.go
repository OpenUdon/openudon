package browsercandidate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/uws/browserauthentication"
)

// AuthenticationCapabilityRequest is the path-free output of independently
// validating one Browsertools authenticated-authoring result. Source and
// review bytes must be the exact canonical producer bytes. Bindings and the
// session are symbolic runtime names, never values or a live browser handle.
type AuthenticationCapabilityRequest struct {
	TransactionID        string
	Flow                 string
	Session              string
	CredentialBindings   []browsertransaction.CredentialBinding
	Authentication       []byte
	AuthenticationReview []byte
	Capability           []byte
	CapabilityReview     []byte
	ResultSHA256         string
	ObservedAt           string
	Origins              []string
	AssessedAt           time.Time
}

// AuthenticationCapability is an immutable, path-free BAP+BCP candidate.
// Accessors return defensive copies, and no runtime browser session is held.
type AuthenticationCapability struct {
	transaction          browsertransaction.Transaction
	flow                 string
	authentication       []byte
	authenticationReview []byte
	capability           []byte
	capabilityReview     []byte
}

// ComposeAuthenticationCapability revalidates one compatible BAP+BCP pair
// and constructs its candidate-state transaction.
func ComposeAuthenticationCapability(request AuthenticationCapabilityRequest) (*AuthenticationCapability, error) {
	if request.AssessedAt.IsZero() {
		return nil, errors.New("authentication-capability assessment time is required")
	}
	authentication, err := authprofile.Parse(request.Authentication)
	if err != nil {
		return nil, fmt.Errorf("authentication candidate: %w", err)
	}
	canonicalAuthentication, err := canonicalJSON(request.Authentication)
	if err != nil || !bytes.Equal(canonicalAuthentication, request.Authentication) {
		return nil, errors.New("authentication candidate is not canonical JSON")
	}
	if err := authprofile.ValidateAt(authentication, request.AssessedAt); err != nil {
		return nil, fmt.Errorf("authentication candidate: %w", err)
	}
	capability, err := profile.ParseJSON(request.Capability)
	if err != nil {
		return nil, fmt.Errorf("capability candidate: %w", err)
	}
	canonicalCapability, err := canonicalJSON(request.Capability)
	if err != nil || !bytes.Equal(canonicalCapability, request.Capability) {
		return nil, errors.New("capability candidate is not canonical JSON")
	}
	if !capability.Info.LoginStateRequired {
		return nil, errors.New("capability candidate must require an authenticated session")
	}
	if authentication.Profile != "uws.browser-authentication.1.1" ||
		(capability.Schema != "uws.browser.1.5" && capability.Schema != "uws.browser.1.6" && capability.Schema != "uws.browser.1.7") {
		return nil, errors.New("authentication-capability profile versions are incompatible")
	}
	if len(authentication.Flows) != 1 {
		return nil, errors.New("authentication candidate flow is ambiguous")
	}
	flow := strings.TrimSpace(request.Flow)
	selectedFlow, ok := authentication.Flows[flow]
	if flow == "" || !ok {
		return nil, errors.New("authentication candidate requires one exact flow")
	}
	if err := compatibleContexts(authentication.Contexts, capability.Contexts); err != nil {
		return nil, err
	}
	origins := append([]string(nil), request.Origins...)
	sort.Strings(origins)
	if !sameStrings(origins, authenticationCapabilityOrigins(authentication, capability)) {
		return nil, errors.New("authentication and capability origins do not exactly match transaction provenance")
	}
	observed, err := time.Parse(time.RFC3339Nano, request.ObservedAt)
	if err != nil || observed.Format(time.RFC3339Nano) != request.ObservedAt || observed.After(request.AssessedAt) {
		return nil, errors.New("authentication-capability observation time is invalid")
	}
	authenticationExpiry, err := authprofile.ExpiresAt(authentication)
	if err != nil {
		return nil, err
	}
	capabilityExpiry, err := browserProfileExpiry(capability)
	if err != nil {
		return nil, err
	}
	expires := authenticationExpiry
	if capabilityExpiry.Before(expires) {
		expires = capabilityExpiry
	}
	if !request.AssessedAt.Before(expires) {
		return nil, errors.New("authentication-capability candidate is stale")
	}
	authenticationDigest := candidateDigest(request.Authentication)
	capabilityDigest := candidateDigest(request.Capability)
	if err := validateAuthenticatedReview(request.AuthenticationReview, "authentication", authentication.Profile, authenticationDigest, request.ObservedAt); err != nil {
		return nil, err
	}
	if err := validateAuthenticatedReview(request.CapabilityReview, "capability", capability.Schema, capabilityDigest, request.ObservedAt); err != nil {
		return nil, err
	}
	bindings := append(make([]browsertransaction.CredentialBinding, 0, len(request.CredentialBindings)), request.CredentialBindings...)
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Slot < bindings[j].Slot })
	if !authenticationBindingsCover(bindings, selectedFlow) {
		return nil, errors.New("authentication symbolic credential bindings do not exactly cover the selected flow")
	}
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: request.TransactionID,
		Kind: browsertransaction.KindAuthenticationCapability, State: browsertransaction.StateCandidate,
		Candidates: []browsertransaction.Candidate{
			{Kind: browsertransaction.CandidateAuthentication, Schema: authentication.Profile, SourceSHA256: authenticationDigest, ReviewSHA256: candidateDigest(request.AuthenticationReview)},
			{Kind: browsertransaction.CandidateCapability, Schema: capability.Schema, SourceSHA256: capabilityDigest, ReviewSHA256: candidateDigest(request.CapabilityReview)},
		},
		Provenance: browsertransaction.Provenance{
			Producer: "browsertools", ResultVersion: browsertransaction.ResultAuthenticatedAuthoringV2,
			ResultSHA256: strings.ToLower(strings.TrimSpace(request.ResultSHA256)), ObservedAt: request.ObservedAt,
			ExpiresAt: expires.UTC().Format(time.RFC3339Nano), Origins: origins,
		},
		CredentialBindings: bindings, Session: strings.TrimSpace(request.Session),
	}
	if err := transaction.Validate(); err != nil {
		return nil, fmt.Errorf("authentication-capability transaction candidate is invalid: %w", err)
	}
	if _, err := browsertransaction.CanonicalBytes(transaction); err != nil {
		return nil, err
	}
	return &AuthenticationCapability{
		transaction: transaction, flow: flow,
		authentication: append([]byte(nil), request.Authentication...), authenticationReview: append([]byte(nil), request.AuthenticationReview...),
		capability: append([]byte(nil), request.Capability...), capabilityReview: append([]byte(nil), request.CapabilityReview...),
	}, nil
}

// Transaction returns a defensive candidate-state transaction.
func (candidate *AuthenticationCapability) Transaction() browsertransaction.Transaction {
	if candidate == nil {
		return browsertransaction.Transaction{}
	}
	return cloneAuthenticationTransaction(candidate.transaction)
}

// ReviewedTransaction records explicit acceptance without changing identity.
func (candidate *AuthenticationCapability) ReviewedTransaction() (browsertransaction.Transaction, error) {
	if candidate == nil {
		return browsertransaction.Transaction{}, errors.New("authentication-capability candidate is required")
	}
	previous := candidate.Transaction()
	next := cloneAuthenticationTransaction(previous)
	next.State = browsertransaction.StateReviewed
	if err := browsertransaction.ValidateTransition(previous, next); err != nil {
		return browsertransaction.Transaction{}, err
	}
	return next, nil
}

func (candidate *AuthenticationCapability) Flow() string {
	if candidate == nil {
		return ""
	}
	return candidate.flow
}
func (candidate *AuthenticationCapability) Authentication() []byte {
	if candidate == nil {
		return nil
	}
	return append([]byte(nil), candidate.authentication...)
}
func (candidate *AuthenticationCapability) AuthenticationReview() []byte {
	if candidate == nil {
		return nil
	}
	return append([]byte(nil), candidate.authenticationReview...)
}
func (candidate *AuthenticationCapability) Capability() []byte {
	if candidate == nil {
		return nil
	}
	return append([]byte(nil), candidate.capability...)
}
func (candidate *AuthenticationCapability) CapabilityReview() []byte {
	if candidate == nil {
		return nil
	}
	return append([]byte(nil), candidate.capabilityReview...)
}

func cloneAuthenticationTransaction(value browsertransaction.Transaction) browsertransaction.Transaction {
	value.Candidates = append([]browsertransaction.Candidate(nil), value.Candidates...)
	value.Provenance.Origins = append([]string(nil), value.Provenance.Origins...)
	value.CredentialBindings = append(make([]browsertransaction.CredentialBinding, 0, len(value.CredentialBindings)), value.CredentialBindings...)
	return value
}

func canonicalJSON(data []byte) ([]byte, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("candidate contains trailing JSON")
	}
	return json.Marshal(value)
}

func validateAuthenticatedReview(data []byte, kind, schema, sourceDigest, observedAt string) error {
	var review authorresult.Review
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&review); err != nil {
		return fmt.Errorf("%s review: %w", kind, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s review contains trailing JSON", kind)
	}
	canonical, err := json.Marshal(review)
	if err != nil || !bytes.Equal(canonical, data) {
		return fmt.Errorf("%s review is not canonical JSON", kind)
	}
	if review.Schema != "browsertools.authenticated-profile-review.v1" || review.Kind != kind || review.ProfileDigest != sourceDigest || review.AssessedAt != observedAt {
		return fmt.Errorf("%s review identity does not match the candidate", kind)
	}
	foundSchema := false
	seen := map[string]bool{}
	for _, decision := range review.Decisions {
		if strings.TrimSpace(decision) == "" || seen[decision] {
			return fmt.Errorf("%s review decisions are invalid", kind)
		}
		seen[decision] = true
		foundSchema = foundSchema || decision == schema
	}
	if !foundSchema {
		return fmt.Errorf("%s review does not bind schema %s", kind, schema)
	}
	return nil
}

func compatibleContexts(authentication map[string]browserauthentication.Context, capability map[string]profile.Context) error {
	for id, authContext := range authentication {
		if capabilityContext, ok := capability[id]; ok {
			converted := browserauthentication.Context{Kind: capabilityContext.Kind, Parent: capabilityContext.Parent, Origin: capabilityContext.Origin, Path: capabilityContext.Path, Name: capabilityContext.Name}
			if !reflect.DeepEqual(authContext, converted) {
				return fmt.Errorf("browser context %q differs between authentication and capability candidates", id)
			}
		}
	}
	return nil
}

func authenticationCapabilityOrigins(authentication *authprofile.Profile, capability *profile.Profile) []string {
	set := map[string]bool{}
	for _, value := range authprofile.Origins(authentication) {
		set[value] = true
	}
	for _, value := range capability.Info.Origin {
		set[value] = true
	}
	for _, context := range capability.Contexts {
		set[context.Origin] = true
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func authenticationBindingsCover(bindings []browsertransaction.CredentialBinding, flow browserauthentication.Flow) bool {
	set := map[string]bool{}
	for _, step := range flow.Sequence {
		if step.TypeCredential != nil {
			set[step.TypeCredential.Slot] = true
		}
		if step.Challenge != nil && step.Challenge.Slot != "" {
			set[step.Challenge.Slot] = true
		}
	}
	if len(bindings) != len(set) {
		return false
	}
	for _, binding := range bindings {
		if !set[binding.Slot] {
			return false
		}
	}
	return true
}

func browserProfileExpiry(value *profile.Profile) (time.Time, error) {
	verified, err := time.Parse(time.RFC3339, value.Verification.LastVerifiedAt)
	if err != nil {
		return time.Time{}, err
	}
	return value.ExpiresAfter.AddTo(verified)
}

func candidateDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
