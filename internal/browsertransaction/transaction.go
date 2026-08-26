// Package browsertransaction defines OpenUdon's value-free browser-profile
// authoring transaction wire. It coordinates existing UWS browser profile
// families; it does not define a new UWS profile or grant browser/runtime
// authority.
package browsertransaction

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	VersionV1 = "openudon.browser-profile-transaction.v1"
	VersionV2 = "openudon.browser-profile-transaction.v2"
	// Version is the immutable legacy default used by unchanged BAP and BRP v1 producers.
	Version                        = VersionV1
	MaxBytes                       = 256 << 10
	ResultAuthenticatedAuthoringV2 = "browsertools.authenticated-authoring.v2"
	ResultRegistrationAuthoringV1  = "browsertools.registration-authoring.v1"
	ResultRegistrationAuthoringV2  = "browsertools.registration-authoring.v2"
	maxJSONDepth                   = 32
)

type Kind string

const (
	KindAuthenticationCapability Kind = "authentication_capability"
	KindRegistration             Kind = "registration"
)

type CandidateKind string

const (
	CandidateAuthentication CandidateKind = "authentication"
	CandidateCapability     CandidateKind = "capability"
	CandidateRegistration   CandidateKind = "registration"
)

type State string

const (
	StateCandidate     State = "candidate"
	StateReviewed      State = "reviewed"
	StatePrepared      State = "prepared"
	StatePromoted      State = "promoted"
	StateCancelled     State = "cancelled"
	StateFailed        State = "failed"
	StateIndeterminate State = "indeterminate"
)

type FailureClass string

const (
	FailureRejected      FailureClass = "rejected"
	FailureConflict      FailureClass = "conflict"
	FailureOperational   FailureClass = "operational"
	FailureIndeterminate FailureClass = "indeterminate"
)

type FailureCode string

const (
	FailureTransactionInvalid     FailureCode = "transaction_invalid"
	FailureCandidateInvalid       FailureCode = "candidate_invalid"
	FailureCandidateStale         FailureCode = "candidate_stale"
	FailureDigestMismatch         FailureCode = "digest_mismatch"
	FailureReviewRejected         FailureCode = "review_rejected"
	FailureWorkspaceConflict      FailureCode = "workspace_conflict"
	FailurePreparationFailed      FailureCode = "preparation_failed"
	FailureQualificationFailed    FailureCode = "qualification_failed"
	FailurePromotionFailed        FailureCode = "promotion_failed"
	FailurePromotionIndeterminate FailureCode = "promotion_indeterminate"
)

// Candidate binds one reviewed profile source and its independently produced
// safe review. Digests use canonical SHA-256 strings and never identify a
// private filesystem location.
type Candidate struct {
	Kind         CandidateKind `json:"kind"`
	Schema       string        `json:"schema"`
	SourceSHA256 string        `json:"source_sha256"`
	ReviewSHA256 string        `json:"review_sha256"`
}

// Provenance binds the private producer result without exposing its path or
// contents. ObservedAt and ExpiresAt are producer-clock lifecycle facts.
type Provenance struct {
	Producer      string   `json:"producer"`
	ResultVersion string   `json:"result_version"`
	ResultSHA256  string   `json:"result_sha256"`
	ObservedAt    string   `json:"observed_at"`
	ExpiresAt     string   `json:"expires_at"`
	Origins       []string `json:"origins"`
}

// CredentialBinding maps one profile slot to a symbolic runtime binding name.
// Neither field may contain a credential value.
type CredentialBinding struct {
	Slot    string `json:"slot"`
	Binding string `json:"binding"`
}

// Preparation identifies the exact prepared package and its independently
// verified qualification report. Presence never means the package is current.
type Preparation struct {
	PackageSHA256       string `json:"package_sha256"`
	QualificationSHA256 string `json:"qualification_sha256"`
}

// Promotion identifies the complete generation made current by a later
// atomic promotion operation.
type Promotion struct {
	GenerationSHA256 string `json:"generation_sha256"`
}

// Failure is deliberately value-free. Human-readable child errors, paths,
// responses, and subprocess output are not part of this wire.
type Failure struct {
	Class FailureClass `json:"class"`
	Code  FailureCode  `json:"code"`
}

// Transaction is an immutable lifecycle snapshot. A BAP+BCP transaction has
// one symbolic session; a BRP transaction is session-free.
type Transaction struct {
	Version            string              `json:"version"`
	ID                 string              `json:"id"`
	Kind               Kind                `json:"kind"`
	State              State               `json:"state"`
	Candidates         []Candidate         `json:"candidates"`
	Provenance         Provenance          `json:"provenance"`
	CredentialBindings []CredentialBinding `json:"credential_bindings"`
	Session            string              `json:"session,omitempty"`
	Preparation        *Preparation        `json:"preparation,omitempty"`
	Promotion          *Promotion          `json:"promotion,omitempty"`
	Failure            *Failure            `json:"failure,omitempty"`
}

var (
	idPattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)
	symbolPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)
)

// Validate enforces the closed wire, profile composition, lifecycle, and
// value-free provenance invariants. Slice order must already be canonical.
func (transaction Transaction) Validate() error {
	if transaction.Version != VersionV1 && transaction.Version != VersionV2 {
		return errors.New("transaction version is unsupported")
	}
	if !idPattern.MatchString(transaction.ID) {
		return errors.New("transaction ID is invalid")
	}
	if transaction.Kind != KindAuthenticationCapability && transaction.Kind != KindRegistration {
		return errors.New("transaction kind is invalid")
	}
	if err := validateVersionComposition(transaction); err != nil {
		return err
	}
	if !validState(transaction.State) {
		return errors.New("transaction state is invalid")
	}
	if err := validateCandidates(transaction.Kind, transaction.Candidates); err != nil {
		return err
	}
	if err := validateProvenance(transaction.Provenance); err != nil {
		return err
	}
	if err := validateBindings(transaction.CredentialBindings); err != nil {
		return err
	}
	if transaction.Kind == KindRegistration && len(transaction.CredentialBindings) == 0 {
		return errors.New("registration transaction requires at least one symbolic credential binding")
	}
	if transaction.Kind == KindAuthenticationCapability {
		if !symbolPattern.MatchString(transaction.Session) {
			return errors.New("authentication-capability transaction requires a symbolic session")
		}
	} else if transaction.Session != "" {
		return errors.New("registration transaction must not declare a session")
	}
	return validateLifecycle(transaction)
}

func validateVersionComposition(transaction Transaction) error {
	switch transaction.Version {
	case VersionV1:
		if transaction.Kind == KindAuthenticationCapability && transaction.Provenance.ResultVersion != ResultAuthenticatedAuthoringV2 {
			return errors.New("transaction v1 authentication-capability provenance is invalid")
		}
		if transaction.Kind == KindRegistration && transaction.Provenance.ResultVersion != ResultRegistrationAuthoringV1 {
			return errors.New("transaction v1 registration provenance is invalid")
		}
	case VersionV2:
		if transaction.Kind != KindRegistration || transaction.Provenance.ResultVersion != ResultRegistrationAuthoringV2 {
			return errors.New("transaction v2 is restricted to registration-authoring v2 provenance")
		}
	}
	return nil
}

// CanonicalBytes returns the deterministic compact JSON representation. It
// sorts cloned set-like fields and leaves the caller unchanged.
func CanonicalBytes(transaction Transaction) ([]byte, error) {
	if transaction.CredentialBindings == nil {
		return nil, errors.New("transaction must contain an explicit credential_bindings array")
	}
	canonical := transaction
	canonical.Candidates = append([]Candidate(nil), transaction.Candidates...)
	canonical.CredentialBindings = append(make([]CredentialBinding, 0, len(transaction.CredentialBindings)), transaction.CredentialBindings...)
	canonical.Provenance.Origins = append([]string(nil), transaction.Provenance.Origins...)
	sort.Slice(canonical.Candidates, func(i, j int) bool { return canonical.Candidates[i].Kind < canonical.Candidates[j].Kind })
	sort.Slice(canonical.CredentialBindings, func(i, j int) bool {
		if canonical.CredentialBindings[i].Slot == canonical.CredentialBindings[j].Slot {
			return canonical.CredentialBindings[i].Binding < canonical.CredentialBindings[j].Binding
		}
		return canonical.CredentialBindings[i].Slot < canonical.CredentialBindings[j].Slot
	})
	sort.Strings(canonical.Provenance.Origins)
	if err := canonical.Validate(); err != nil {
		return nil, err
	}
	data, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("browser-profile transaction exceeds %d bytes", MaxBytes)
	}
	return data, nil
}

// Digest returns the SHA-256 digest of CanonicalBytes.
func Digest(transaction Transaction) (string, error) {
	data, err := CanonicalBytes(transaction)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// Decode strictly decodes one JSON document and validates its canonical
// ordering and semantic invariants.
func Decode(data []byte) (Transaction, error) {
	var transaction Transaction
	if len(data) > MaxBytes {
		return Transaction{}, fmt.Errorf("browser-profile transaction exceeds %d bytes", MaxBytes)
	}
	if !utf8.Valid(data) {
		return Transaction{}, errors.New("browser-profile transaction must be valid UTF-8")
	}
	if err := rejectDuplicateJSONNames(data); err != nil {
		return Transaction{}, fmt.Errorf("decode browser-profile transaction: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&transaction); err != nil {
		return Transaction{}, fmt.Errorf("decode browser-profile transaction: %w", err)
	}
	if err := requireEOF(decoder); err != nil {
		return Transaction{}, err
	}
	if err := transaction.Validate(); err != nil {
		return Transaction{}, err
	}
	return transaction, nil
}

// DecodeVerified rejects a valid but changed transaction when its canonical
// digest does not match the separately retained expected digest.
func DecodeVerified(data []byte, expectedSHA256 string) (Transaction, error) {
	transaction, err := Decode(data)
	if err != nil {
		return Transaction{}, err
	}
	actual, err := Digest(transaction)
	if err != nil {
		return Transaction{}, err
	}
	if !validDigest(expectedSHA256) || actual != strings.ToLower(expectedSHA256) {
		return Transaction{}, errors.New("browser-profile transaction digest mismatch")
	}
	return transaction, nil
}

// ValidateTransition proves that immutable transaction identity/provenance
// facts did not change and that the lifecycle edge is allowed.
func ValidateTransition(previous, next Transaction) error {
	if err := previous.Validate(); err != nil {
		return fmt.Errorf("previous transaction: %w", err)
	}
	if err := next.Validate(); err != nil {
		return fmt.Errorf("next transaction: %w", err)
	}
	previousIdentity := previous
	nextIdentity := next
	previousIdentity.State, nextIdentity.State = "", ""
	previousIdentity.Preparation, nextIdentity.Preparation = nil, nil
	previousIdentity.Promotion, nextIdentity.Promotion = nil, nil
	previousIdentity.Failure, nextIdentity.Failure = nil, nil
	previousBytes, err := canonicalIdentityBytes(previousIdentity)
	if err != nil {
		return fmt.Errorf("canonicalize previous transaction identity: %w", err)
	}
	nextBytes, err := canonicalIdentityBytes(nextIdentity)
	if err != nil {
		return fmt.Errorf("canonicalize next transaction identity: %w", err)
	}
	if !bytes.Equal(previousBytes, nextBytes) {
		return errors.New("transaction immutable identity or provenance changed")
	}
	if previous.Preparation != nil {
		if next.Preparation == nil || *previous.Preparation != *next.Preparation {
			return errors.New("transaction preparation changed or disappeared")
		}
	} else if next.Preparation != nil && next.State != StatePrepared {
		return errors.New("transaction preparation appeared outside a reviewed-to-prepared transition")
	}
	if previous.Promotion != nil {
		if next.Promotion == nil || *previous.Promotion != *next.Promotion {
			return errors.New("transaction promotion changed or disappeared")
		}
	}
	if !allowedTransition(previous.State, next.State) {
		return fmt.Errorf("transaction transition %s -> %s is not allowed", previous.State, next.State)
	}
	return nil
}

func canonicalIdentityBytes(transaction Transaction) ([]byte, error) {
	transaction.State = StateCandidate
	transaction.Preparation = nil
	transaction.Promotion = nil
	transaction.Failure = nil
	return CanonicalBytes(transaction)
}

func validateCandidates(kind Kind, candidates []Candidate) error {
	want := []CandidateKind{CandidateAuthentication, CandidateCapability}
	if kind == KindRegistration {
		want = []CandidateKind{CandidateRegistration}
	}
	if len(candidates) != len(want) {
		return errors.New("transaction candidate composition is invalid")
	}
	for index, candidate := range candidates {
		if candidate.Kind != want[index] {
			return errors.New("transaction candidates are not canonical or do not match the transaction kind")
		}
		if !validCandidateSchema(candidate.Kind, candidate.Schema) {
			return errors.New("transaction candidate schema is invalid")
		}
		if !validDigest(candidate.SourceSHA256) || !validDigest(candidate.ReviewSHA256) {
			return errors.New("transaction candidate digest is invalid")
		}
	}
	return nil
}

func validCandidateSchema(kind CandidateKind, schema string) bool {
	switch kind {
	case CandidateAuthentication:
		return schema == "uws.browser-authentication.1.0" || schema == "uws.browser-authentication.1.1"
	case CandidateCapability:
		return schema == "uws.browser.1.5" || schema == "uws.browser.1.6" || schema == "uws.browser.1.7"
	case CandidateRegistration:
		return schema == "uws.browser-registration.1.0"
	default:
		return false
	}
}

func validateProvenance(provenance Provenance) error {
	if provenance.Producer != "browsertools" || !validResultVersion(provenance.ResultVersion) || !validDigest(provenance.ResultSHA256) {
		return errors.New("transaction provenance is invalid")
	}
	observed, err := time.Parse(time.RFC3339Nano, provenance.ObservedAt)
	if err != nil || observed.Format(time.RFC3339Nano) != provenance.ObservedAt {
		return errors.New("transaction observed_at must be canonical RFC3339")
	}
	expires, err := time.Parse(time.RFC3339Nano, provenance.ExpiresAt)
	if err != nil || expires.Format(time.RFC3339Nano) != provenance.ExpiresAt || !expires.After(observed) {
		return errors.New("transaction expires_at must be canonical RFC3339 after observed_at")
	}
	if len(provenance.Origins) == 0 || len(provenance.Origins) > 32 {
		return errors.New("transaction provenance must contain one through 32 origins")
	}
	previous := ""
	for _, origin := range provenance.Origins {
		if origin <= previous || !validOrigin(origin) {
			return errors.New("transaction origins are invalid, duplicated, or not canonical")
		}
		previous = origin
	}
	return nil
}

func validateBindings(bindings []CredentialBinding) error {
	if bindings == nil || len(bindings) > 32 {
		return errors.New("transaction must contain an explicit array of at most 32 symbolic credential bindings")
	}
	previousSlot := ""
	for _, binding := range bindings {
		if !symbolPattern.MatchString(binding.Slot) || !symbolPattern.MatchString(binding.Binding) || binding.Slot <= previousSlot {
			return errors.New("transaction credential bindings are invalid, duplicated, or not canonical")
		}
		previousSlot = binding.Slot
	}
	return nil
}

func validateLifecycle(transaction Transaction) error {
	if transaction.Preparation != nil && (!validDigest(transaction.Preparation.PackageSHA256) || !validDigest(transaction.Preparation.QualificationSHA256)) {
		return errors.New("transaction preparation digest is invalid")
	}
	if transaction.Promotion != nil && !validDigest(transaction.Promotion.GenerationSHA256) {
		return errors.New("transaction promotion digest is invalid")
	}
	if transaction.Failure != nil && !validFailure(*transaction.Failure) {
		return errors.New("transaction failure is invalid")
	}
	switch transaction.State {
	case StateCandidate, StateReviewed:
		if transaction.Preparation != nil || transaction.Promotion != nil || transaction.Failure != nil {
			return errors.New("candidate/reviewed transaction has later lifecycle fields")
		}
	case StatePrepared:
		if transaction.Preparation == nil || transaction.Promotion != nil || transaction.Failure != nil {
			return errors.New("prepared transaction lifecycle fields are invalid")
		}
	case StatePromoted:
		if transaction.Preparation == nil || transaction.Promotion == nil || transaction.Failure != nil {
			return errors.New("promoted transaction lifecycle fields are invalid")
		}
	case StateCancelled:
		if transaction.Promotion != nil || transaction.Failure != nil {
			return errors.New("cancelled transaction lifecycle fields are invalid")
		}
	case StateFailed:
		if transaction.Promotion != nil || transaction.Failure == nil || transaction.Failure.Class == FailureIndeterminate {
			return errors.New("failed transaction lifecycle fields are invalid")
		}
	case StateIndeterminate:
		if transaction.Preparation == nil || transaction.Promotion != nil || transaction.Failure == nil || transaction.Failure.Class != FailureIndeterminate || transaction.Failure.Code != FailurePromotionIndeterminate {
			return errors.New("indeterminate transaction lifecycle fields are invalid")
		}
	}
	return nil
}

func validState(state State) bool {
	switch state {
	case StateCandidate, StateReviewed, StatePrepared, StatePromoted, StateCancelled, StateFailed, StateIndeterminate:
		return true
	default:
		return false
	}
}

func validFailure(failure Failure) bool {
	switch failure.Class {
	case FailureRejected:
		return failure.Code == FailureTransactionInvalid || failure.Code == FailureCandidateInvalid || failure.Code == FailureCandidateStale || failure.Code == FailureDigestMismatch || failure.Code == FailureReviewRejected
	case FailureConflict:
		return failure.Code == FailureWorkspaceConflict
	case FailureOperational:
		return failure.Code == FailurePreparationFailed || failure.Code == FailureQualificationFailed || failure.Code == FailurePromotionFailed
	case FailureIndeterminate:
		return failure.Code == FailurePromotionIndeterminate
	default:
		return false
	}
}

func allowedTransition(previous, next State) bool {
	switch previous {
	case StateCandidate:
		return next == StateReviewed || next == StateCancelled || next == StateFailed
	case StateReviewed:
		return next == StatePrepared || next == StateCancelled || next == StateFailed
	case StatePrepared:
		return next == StatePromoted || next == StateIndeterminate || next == StateCancelled || next == StateFailed
	case StateIndeterminate:
		return next == StatePrepared || next == StatePromoted || next == StateCancelled || next == StateFailed
	default:
		return false
	}
}

func validDigest(value string) bool {
	if len(value) != len("sha256:")+sha256.Size*2 || value != strings.ToLower(value) || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(value, "sha256:"))
	return err == nil
}

func validResultVersion(value string) bool {
	return value == ResultAuthenticatedAuthoringV2 || value == ResultRegistrationAuthoringV1 || value == ResultRegistrationAuthoringV2
}

func validOrigin(value string) bool {
	if utf8.RuneCountInString(value) > 1024 {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return false
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		return false
	}
	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return false
	}
	canonicalHostname := hostname
	if ip := net.ParseIP(hostname); ip != nil {
		canonicalHostname = ip.String()
	} else if strings.Contains(hostname, ":") || !asciiOnly(hostname) {
		return false
	}
	if scheme == "http" && hostname != "localhost" {
		ip := net.ParseIP(hostname)
		if ip == nil || !ip.IsLoopback() {
			return false
		}
	}
	port := parsed.Port()
	if port != "" {
		number, err := strconv.Atoi(port)
		defaultPort := scheme == "https" && number == 443 || scheme == "http" && number == 80
		if err != nil || number < 1 || number > 65535 || strconv.Itoa(number) != port || defaultPort {
			return false
		}
	}
	canonicalHost := canonicalHostname
	if strings.Contains(canonicalHostname, ":") {
		canonicalHost = "[" + canonicalHostname + "]"
	}
	if port != "" {
		canonicalHost = net.JoinHostPort(canonicalHostname, port)
	}
	canonical := scheme + "://" + canonicalHost
	return value == canonical
}

func asciiOnly(value string) bool {
	for _, character := range value {
		if character > 127 {
			return false
		}
	}
	return true
}

func rejectDuplicateJSONNames(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := scanJSONValue(decoder, 0); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return fmt.Errorf("contains trailing JSON token %v", token)
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, depth int) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if depth >= maxJSONDepth {
		return fmt.Errorf("JSON nesting exceeds %d levels", maxJSONDepth)
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			nameToken, err := decoder.Token()
			if err != nil {
				return err
			}
			name, ok := nameToken.(string)
			if !ok {
				return errors.New("JSON object member name is not a string")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON field %q", name)
			}
			seen[name] = true
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim('}') {
			return errors.New("JSON object is not closed")
		}
	case '[':
		for decoder.More() {
			if err := scanJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil || end != json.Delim(']') {
			return errors.New("JSON array is not closed")
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delimiter)
	}
	return nil
}

func requireEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("browser-profile transaction contains multiple JSON values")
		}
		return fmt.Errorf("decode browser-profile transaction trailer: %w", err)
	}
	return nil
}
