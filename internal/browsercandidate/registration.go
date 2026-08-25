// Package browsercandidate adopts private Browsertools authoring results into
// value-free OpenUdon browser-profile transactions. Private paths and complete
// producer envelopes remain confined to this package.
package browsercandidate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/browsertools/registrationauthorresult"
	"github.com/OpenUdon/browsertools/registrationauthorsession"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/browsertools/registrationreview"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
)

const maxPrivateResultBytes = browsertransaction.MaxBytes

var registrationResultName = regexp.MustCompile(`^registration-authoring-[0-9a-f]{16}\.json$`)

// RegistrationReview binds the operator-confirmed protocol review to the
// private result that may be adopted after the worker exits cleanly.
type RegistrationReview struct {
	Confirmed          bool
	ProfileID          string
	Flow               string
	SourceSHA256       string
	ReviewedCandidates []registrationauthorsession.ReviewedCandidate
	CleanupDisposition string
	Origins            []string
	Bounds             registrationauthorsession.Bounds
	Observations       int
	MinimumRequests    int
}

// AdoptRegistrationRequest contains only value-free transaction and review
// inputs. CredentialBindings are symbolic runtime names, never values.
type AdoptRegistrationRequest struct {
	TransactionID      string
	CredentialBindings []browsertransaction.CredentialBinding
	Review             RegistrationReview
	AssessedAt         time.Time
}

// Registration is an immutable adopted candidate. Accessors return defensive
// copies so later engine work cannot alter its reviewed identity.
type Registration struct {
	transaction browsertransaction.Transaction
	profileID   string
	flow        string
	source      []byte
	review      []byte
}

// Transaction returns a defensively copied candidate-state transaction.
func (candidate *Registration) Transaction() browsertransaction.Transaction {
	if candidate == nil {
		return browsertransaction.Transaction{}
	}
	result := candidate.transaction
	result.Candidates = append([]browsertransaction.Candidate(nil), result.Candidates...)
	result.Provenance.Origins = append([]string(nil), result.Provenance.Origins...)
	result.CredentialBindings = append([]browsertransaction.CredentialBinding(nil), result.CredentialBindings...)
	return result
}

// ProfileID returns the portable Browsertools profile identity.
func (candidate *Registration) ProfileID() string {
	if candidate == nil {
		return ""
	}
	return candidate.profileID
}

// Flow returns the exact reviewed registration flow.
func (candidate *Registration) Flow() string {
	if candidate == nil {
		return ""
	}
	return candidate.flow
}

// Source returns canonical uws.browser-registration.1.0 JSON.
func (candidate *Registration) Source() []byte {
	if candidate == nil {
		return nil
	}
	return append([]byte(nil), candidate.source...)
}

// Review returns canonical browsertools.registration-review.v1 JSON.
func (candidate *Registration) Review() []byte {
	if candidate == nil {
		return nil
	}
	return append([]byte(nil), candidate.review...)
}

// PrivateInbox anchors one restrictive per-run root before a worker starts and
// remembers every existing result identity. It never exposes a result name.
type PrivateInbox struct {
	path     string
	root     *os.Root
	identity os.FileInfo
	baseline map[string]os.FileInfo
}

// OpenPrivateInbox opens and snapshots an absolute, canonical, mode-0700 root.
func OpenPrivateInbox(privateRoot string) (*PrivateInbox, error) {
	clean := filepath.Clean(privateRoot)
	resolved, resolveErr := filepath.EvalSymlinks(clean)
	before, statErr := os.Lstat(clean)
	if privateRoot != clean || !filepath.IsAbs(clean) || resolveErr != nil || resolved != clean || statErr != nil ||
		!before.IsDir() || before.Mode()&os.ModeSymlink != 0 || before.Mode().Perm() != 0o700 {
		return nil, errors.New("registration private root must be an absolute canonical mode-0700 non-symlink directory")
	}
	root, err := os.OpenRoot(clean)
	if err != nil {
		return nil, errors.New("open registration private root")
	}
	inbox := &PrivateInbox{path: clean, root: root, identity: before}
	if err := inbox.validateRoot(); err != nil {
		_ = root.Close()
		return nil, err
	}
	baseline, err := inbox.scan()
	if err != nil {
		_ = root.Close()
		return nil, err
	}
	inbox.baseline = baseline
	return inbox, nil
}

// Close releases the anchored root.
func (inbox *PrivateInbox) Close() error {
	if inbox == nil || inbox.root == nil {
		return nil
	}
	err := inbox.root.Close()
	inbox.root = nil
	return err
}

// AdoptNewRegistration requires exactly one new private result since the
// inbox snapshot and adopts it without returning its name or path.
func (inbox *PrivateInbox) AdoptNewRegistration(request AdoptRegistrationRequest) (*Registration, error) {
	if inbox == nil || inbox.root == nil {
		return nil, errors.New("registration private inbox is unavailable")
	}
	if err := inbox.validateRoot(); err != nil {
		return nil, err
	}
	current, err := inbox.scan()
	if err != nil {
		return nil, err
	}
	for name, before := range inbox.baseline {
		after, ok := current[name]
		if !ok || !sameFileState(before, after) {
			return nil, errors.New("existing private registration result changed during the worker run")
		}
	}
	newNames := make([]string, 0, 1)
	for name := range current {
		if _, existed := inbox.baseline[name]; !existed {
			newNames = append(newNames, name)
		}
	}
	if len(newNames) != 1 {
		return nil, errors.New("registration worker must create exactly one new private result")
	}
	data, digest, err := inbox.readStable(newNames[0])
	if err != nil {
		return nil, err
	}
	if newNames[0] != resultName(digest) {
		return nil, errors.New("private registration result name does not match its exact digest")
	}
	return adoptRegistration(data, digest, request)
}

func (inbox *PrivateInbox) validateRoot() error {
	if inbox == nil || inbox.root == nil {
		return errors.New("registration private inbox is unavailable")
	}
	anchored, anchorErr := inbox.root.Stat(".")
	current, pathErr := os.Lstat(inbox.path)
	if anchorErr != nil || pathErr != nil || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 ||
		current.Mode().Perm() != 0o700 || !os.SameFile(inbox.identity, anchored) || !os.SameFile(anchored, current) {
		return errors.New("registration private root changed during the worker run")
	}
	return nil
}

func (inbox *PrivateInbox) scan() (map[string]os.FileInfo, error) {
	directory, err := inbox.root.Open(".")
	if err != nil {
		return nil, errors.New("scan registration private root")
	}
	defer directory.Close()
	entries, err := directory.ReadDir(4097)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, errors.New("scan registration private root")
	}
	if len(entries) > 4096 {
		return nil, errors.New("registration private root entry limit exceeded")
	}
	results := make(map[string]os.FileInfo)
	for _, entry := range entries {
		name := entry.Name()
		if !registrationResultName.MatchString(name) {
			continue
		}
		info, err := inbox.root.Lstat(name)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 ||
			info.Mode().Perm() != 0o600 || info.Size() <= 0 || info.Size() > maxPrivateResultBytes {
			return nil, errors.New("private registration result identity is invalid")
		}
		results[name] = info
	}
	return results, nil
}

var privateResultReadHook = func() {}

func (inbox *PrivateInbox) readStable(name string) ([]byte, string, error) {
	before, err := inbox.root.Lstat(name)
	if err != nil || !before.Mode().IsRegular() || before.Mode().Perm() != 0o600 || before.Size() <= 0 || before.Size() > maxPrivateResultBytes {
		return nil, "", errors.New("private registration result identity is invalid")
	}
	file, err := inbox.root.Open(name)
	if err != nil {
		return nil, "", errors.New("open private registration result")
	}
	defer file.Close()
	opened, statErr := file.Stat()
	if statErr != nil || !opened.Mode().IsRegular() || opened.Mode().Perm() != 0o600 || !os.SameFile(before, opened) {
		return nil, "", errors.New("private registration result changed during open")
	}
	data, readErr := io.ReadAll(io.LimitReader(file, maxPrivateResultBytes+1))
	if readErr != nil || len(data) == 0 || len(data) > maxPrivateResultBytes {
		return nil, "", errors.New("read private registration result")
	}
	privateResultReadHook()
	after, afterErr := file.Stat()
	pathAfter, pathErr := inbox.root.Lstat(name)
	if afterErr != nil || pathErr != nil || !sameFileState(opened, after) || !sameFileState(after, pathAfter) || after.Size() != int64(len(data)) {
		return nil, "", errors.New("private registration result changed during read")
	}
	if err := inbox.validateRoot(); err != nil {
		return nil, "", err
	}
	return data, digest(data), nil
}

func adoptRegistration(data []byte, resultDigest string, request AdoptRegistrationRequest) (*Registration, error) {
	if !request.Review.Confirmed {
		return nil, errors.New("registration candidate requires explicit human review")
	}
	if request.AssessedAt.IsZero() || request.AssessedAt.Location() != time.UTC || request.AssessedAt.Nanosecond() != 0 {
		return nil, errors.New("registration assessment time must be whole-second UTC")
	}
	result, err := registrationauthorresult.Decode(data, request.AssessedAt)
	if err != nil {
		return nil, fmt.Errorf("registration result rejected: %w", err)
	}
	canonicalResult, err := registrationauthorresult.MarshalDeterministic(result)
	if err != nil || !bytes.Equal(data, canonicalResult) {
		return nil, errors.New("registration result is not exact canonical producer output")
	}
	reconstructedDigest, err := registrationauthorresult.Digest(result)
	if err != nil || reconstructedDigest != resultDigest {
		return nil, errors.New("registration result digest does not match reconstructed output")
	}
	if result.Schema != registrationauthorresult.Schema || result.Provenance.Producer != "browsertools" ||
		result.Provenance.ResultVersion != registrationauthorresult.Schema ||
		result.Provenance.SessionVersion != registrationauthorsession.Protocol {
		return nil, errors.New("registration result provenance is unsupported")
	}
	profile, err := registrationprofile.Parse(result.Candidate.Source)
	if err != nil {
		return nil, errors.New("registration candidate source is invalid")
	}
	source, err := registrationprofile.MarshalJSON(profile)
	if err != nil || !bytes.Equal(source, result.Candidate.Source) {
		return nil, errors.New("registration candidate source is not canonical")
	}
	sourceDigest := digest(source)
	if result.Candidate.Schema != "uws.browser-registration.1.0" || result.Candidate.SourceDigest != sourceDigest ||
		request.Review.SourceSHA256 != sourceDigest {
		return nil, errors.New("registration candidate source digest or schema changed after review")
	}
	createdAt, err := time.Parse(time.RFC3339, result.CreatedAt)
	if err != nil || createdAt.Location() != time.UTC || createdAt.Nanosecond() != 0 {
		return nil, errors.New("registration result creation time is invalid")
	}
	review, err := registrationreview.Build(profile, createdAt)
	if err != nil {
		return nil, errors.New("rebuild registration review")
	}
	reviewBytes, err := json.Marshal(review)
	if err != nil {
		return nil, errors.New("encode reconstructed registration review")
	}
	producerReviewBytes, err := json.Marshal(result.Candidate.Review)
	if err != nil || !bytes.Equal(reviewBytes, producerReviewBytes) || result.Candidate.ReviewDigest != digest(reviewBytes) {
		return nil, errors.New("registration review does not match independent reconstruction")
	}
	if request.Review.ProfileID != result.Candidate.ProfileID || request.Review.Flow != result.Flow.Name ||
		request.Review.CleanupDisposition != result.CallPolicy.CleanupDisposition ||
		!equalStrings(request.Review.Origins, result.Origins) ||
		!equalReviewedCandidates(request.Review.ReviewedCandidates, result.ReviewedCandidates) ||
		request.Review.Bounds != result.Bounds || request.Review.Observations != result.Observations ||
		request.Review.MinimumRequests <= 0 || result.Network.Requests < request.Review.MinimumRequests {
		return nil, errors.New("registration result does not match the explicit human review")
	}
	if !equalStrings(result.Origins, registrationprofile.Origins(profile)) {
		return nil, errors.New("registration result origins do not match the canonical profile")
	}
	if result.Network.Methods == nil || !equalStrings(result.Network.Methods, []string{"GET", "HEAD"}) ||
		result.Network.Requests != result.Network.GETRequests+result.Network.HEADRequests ||
		result.Network.MutationRequests != 0 || result.Network.SubmitExecuted || result.Network.AccountAttempted ||
		result.Network.SessionEstablished || result.Network.RuntimeSupported || result.Flow.Submit.Executed {
		return nil, errors.New("registration result claims unsupported browser or runtime authority")
	}
	bindings := append([]browsertransaction.CredentialBinding(nil), request.CredentialBindings...)
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Slot < bindings[j].Slot })
	if !bindingsCoverSlots(bindings, result.Flow.CredentialSlots) {
		return nil, errors.New("registration symbolic credential bindings do not exactly cover the reviewed slots")
	}
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: request.TransactionID,
		Kind: browsertransaction.KindRegistration, State: browsertransaction.StateCandidate,
		Candidates: []browsertransaction.Candidate{{
			Kind: browsertransaction.CandidateRegistration, Schema: result.Candidate.Schema,
			SourceSHA256: sourceDigest, ReviewSHA256: result.Candidate.ReviewDigest,
		}},
		Provenance: browsertransaction.Provenance{
			Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1,
			ResultSHA256: resultDigest, ObservedAt: result.ObservedAt, ExpiresAt: result.ExpiresAt,
			Origins: append([]string(nil), result.Origins...),
		},
		CredentialBindings: bindings,
	}
	if err := transaction.Validate(); err != nil {
		return nil, fmt.Errorf("registration transaction candidate is invalid: %w", err)
	}
	if _, err := browsertransaction.CanonicalBytes(transaction); err != nil {
		return nil, fmt.Errorf("canonicalize registration transaction candidate: %w", err)
	}
	return &Registration{
		transaction: transaction, profileID: result.Candidate.ProfileID, flow: result.Flow.Name,
		source: append([]byte(nil), source...), review: append([]byte(nil), reviewBytes...),
	}, nil
}

func bindingsCoverSlots(bindings []browsertransaction.CredentialBinding, slots []registrationauthorresult.CredentialSlot) bool {
	if len(bindings) != len(slots) {
		return false
	}
	for index := range bindings {
		if bindings[index].Slot != slots[index].Slot || strings.TrimSpace(bindings[index].Binding) == "" {
			return false
		}
	}
	return true
}

func equalReviewedCandidates(left, right []registrationauthorsession.ReviewedCandidate) bool {
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

func sameFileState(left, right os.FileInfo) bool {
	return left != nil && right != nil && os.SameFile(left, right) && left.Mode() == right.Mode() &&
		left.Size() == right.Size() && left.ModTime().Equal(right.ModTime())
}

func resultName(resultDigest string) string {
	return "registration-authoring-" + strings.TrimPrefix(resultDigest, "sha256:")[:16] + ".json"
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func equalStrings(left, right []string) bool {
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
