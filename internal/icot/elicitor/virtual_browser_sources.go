package elicitor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/browsertools/registrationreview"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
	"github.com/OpenUdon/uws/browserregistration"
)

const (
	virtualBrowserSourceKind = "browsertools_transaction"
	virtualBrowserPrefix     = "virtual-browser://"
	maxVirtualTransactions   = 128
	maxVirtualReviewBytes    = browsertransaction.MaxBytes
)

// VirtualBrowserSourceInput carries one exact, independently reviewed source
// without a filesystem location. Source and Review are retained only in the
// engine process and never serialized in a snapshot or draft.
type VirtualBrowserSourceInput struct {
	Kind               browsertransaction.CandidateKind `json:"kind"`
	Flow               string                           `json:"flow,omitempty"`
	CleanupDisposition string                           `json:"cleanup_disposition,omitempty"`
	Source             []byte                           `json:"-"`
	Review             []byte                           `json:"-"`
}

// VirtualBrowserTransactionInput is the path-free output of private result
// adoption. Transaction identity determines all public candidate IDs and
// package targets.
type VirtualBrowserTransactionInput struct {
	Transaction browsertransaction.Transaction `json:"transaction"`
	Sources     []VirtualBrowserSourceInput    `json:"-"`
}

// VirtualBrowserCandidate is the value-free catalog view returned to engine
// clients. Digests, symbolic bindings, and session names are metadata; source
// and review bytes and private producer details are deliberately absent.
type VirtualBrowserCandidate struct {
	ID                 string                                 `json:"id"`
	TransactionID      string                                 `json:"transaction_id"`
	TransactionSHA256  string                                 `json:"transaction_sha256"`
	Kind               browsertransaction.CandidateKind       `json:"kind"`
	Schema             string                                 `json:"schema"`
	SourceSHA256       string                                 `json:"source_sha256"`
	ReviewSHA256       string                                 `json:"review_sha256"`
	TargetPath         string                                 `json:"target_path"`
	Title              string                                 `json:"title,omitempty"`
	Flow               string                                 `json:"flow,omitempty"`
	CleanupDisposition string                                 `json:"cleanup_disposition,omitempty"`
	Dependencies       []string                               `json:"dependencies,omitempty"`
	ProvidesSession    string                                 `json:"provides_session,omitempty"`
	RequiresSession    string                                 `json:"requires_session,omitempty"`
	CredentialBindings []browsertransaction.CredentialBinding `json:"credential_bindings,omitempty"`
	Selected           bool                                   `json:"selected"`
}

// VirtualBrowserDiscovery is a validated immutable catalog. Docs and Plans
// hold canonical in-memory source bytes; Candidates is the only public view.
type VirtualBrowserDiscovery struct {
	Candidates []VirtualBrowserCandidate `json:"candidates,omitempty"`
	Docs       []APIDocument             `json:"-"`
	Plans      []SourceMaterialization   `json:"-"`
}

// DiscoverVirtualBrowserSources validates candidate-state transaction inputs
// and constructs deterministic source documents without touching a workspace.
func DiscoverVirtualBrowserSources(inputs []VirtualBrowserTransactionInput, at time.Time) (VirtualBrowserDiscovery, error) {
	if at.IsZero() {
		return VirtualBrowserDiscovery{}, errors.New("virtual browser assessment time is required")
	}
	if len(inputs) > maxVirtualTransactions {
		return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transactions exceed limit %d", maxVirtualTransactions)
	}
	inputs = cloneVirtualBrowserInputs(inputs)
	sort.Slice(inputs, func(i, j int) bool { return inputs[i].Transaction.ID < inputs[j].Transaction.ID })
	result := VirtualBrowserDiscovery{Candidates: []VirtualBrowserCandidate{}, Docs: []APIDocument{}, Plans: []SourceMaterialization{}}
	seenTransactions := map[string]bool{}
	seenCandidates := map[string]bool{}
	seenTargets := map[string]string{}
	for _, input := range inputs {
		transaction := input.Transaction
		if err := transaction.Validate(); err != nil {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction: %w", err)
		}
		if transaction.State != browsertransaction.StateCandidate && transaction.State != browsertransaction.StateReviewed {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction %s is not discoverable in state %s", transaction.ID, transaction.State)
		}
		if seenTransactions[transaction.ID] {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction ID %q is duplicated", transaction.ID)
		}
		seenTransactions[transaction.ID] = true
		expires, err := time.Parse(time.RFC3339Nano, transaction.Provenance.ExpiresAt)
		if err != nil || !at.Before(expires) {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction %s is stale", transaction.ID)
		}
		transactionDigest, err := browsertransaction.Digest(transaction)
		if err != nil {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction %s: %w", transaction.ID, err)
		}
		if len(input.Sources) != len(transaction.Candidates) {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction %s source composition is incomplete", transaction.ID)
		}
		sources := append([]VirtualBrowserSourceInput(nil), input.Sources...)
		sort.Slice(sources, func(i, j int) bool { return sources[i].Kind < sources[j].Kind })
		transactionOrigins := map[string]bool{}
		transactionSlots := map[string]bool{}
		for index, candidate := range transaction.Candidates {
			if sources[index].Kind != candidate.Kind {
				return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction %s source composition does not match its candidates", transaction.ID)
			}
			if len(sources[index].Source) == 0 || len(sources[index].Source) > browsertransaction.MaxBytes || len(sources[index].Review) == 0 || len(sources[index].Review) > maxVirtualReviewBytes {
				return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser candidate %s/%s source or review is empty or oversized", transaction.ID, candidate.Kind)
			}
			if digestVirtualBytes(sources[index].Source) != candidate.SourceSHA256 || digestVirtualBytes(sources[index].Review) != candidate.ReviewSHA256 {
				return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser candidate %s/%s digest mismatch", transaction.ID, candidate.Kind)
			}
			catalogCandidate, plan, doc, err := virtualBrowserMaterialization(transaction, transactionDigest, candidate, sources[index], at)
			if err != nil {
				return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser candidate %s/%s: %w", transaction.ID, candidate.Kind, err)
			}
			if seenCandidates[catalogCandidate.ID] {
				return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser candidate ID %q is duplicated", catalogCandidate.ID)
			}
			seenCandidates[catalogCandidate.ID] = true
			if prior, collision := seenTargets[plan.TargetPath]; collision {
				return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser target %q collides between %s and %s", plan.TargetPath, prior, catalogCandidate.ID)
			}
			seenTargets[plan.TargetPath] = catalogCandidate.ID
			for _, origin := range plan.Origins {
				transactionOrigins[origin] = true
			}
			if catalogCandidate.Flow != "" {
				for _, slot := range plan.FlowCredentialSlots[catalogCandidate.Flow] {
					transactionSlots[slot] = true
				}
			}
			result.Candidates = append(result.Candidates, catalogCandidate)
			result.Plans = append(result.Plans, plan)
			result.Docs = append(result.Docs, doc)
		}
		if !sameVirtualOrigins(transaction.Provenance.Origins, transactionOrigins) {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction %s source origins do not match transaction provenance", transaction.ID)
		}
		if !sameVirtualBindingSlots(transaction.CredentialBindings, transactionSlots) {
			return VirtualBrowserDiscovery{}, fmt.Errorf("virtual browser transaction %s symbolic bindings do not exactly cover source credential slots", transaction.ID)
		}
	}
	sort.Slice(result.Candidates, func(i, j int) bool { return result.Candidates[i].ID < result.Candidates[j].ID })
	sort.Slice(result.Docs, func(i, j int) bool { return result.Docs[i].RelativePath < result.Docs[j].RelativePath })
	result.Plans = normalizeSourcePlan(result.Plans)
	if _, err := virtualBrowserDependencyClosure(result.Candidates, candidateIDs(result.Candidates)); err != nil {
		return VirtualBrowserDiscovery{}, err
	}
	return cloneVirtualBrowserDiscovery(result), nil
}

func virtualBrowserMaterialization(transaction browsertransaction.Transaction, transactionDigest string, candidate browsertransaction.Candidate, input VirtualBrowserSourceInput, at time.Time) (VirtualBrowserCandidate, SourceMaterialization, APIDocument, error) {
	id := transaction.ID + "/" + string(candidate.Kind)
	sourcePath := virtualBrowserPrefix + transaction.ID + "/" + string(candidate.Kind) + "/" + strings.TrimPrefix(candidate.SourceSHA256, "sha256:")
	provenance := "browsertools-transaction:" + transactionDigest
	bindings := append([]browsertransaction.CredentialBinding(nil), transaction.CredentialBindings...)
	public := VirtualBrowserCandidate{
		ID: id, TransactionID: transaction.ID, TransactionSHA256: transactionDigest, Kind: candidate.Kind,
		Schema: candidate.Schema, SourceSHA256: candidate.SourceSHA256, ReviewSHA256: candidate.ReviewSHA256,
		CredentialBindings: bindings,
	}
	materialized := append(append([]byte(nil), input.Source...), '\n')
	materializedSum := sha256.Sum256(materialized)
	plan := SourceMaterialization{
		SourceKind: virtualBrowserSourceKind, SourcePath: sourcePath,
		SHA256: hex.EncodeToString(materializedSum[:]), SourceSHA256: strings.TrimPrefix(candidate.SourceSHA256, "sha256:"),
		Lifecycle: "active", ExpiresAt: transaction.Provenance.ExpiresAt, Provenance: provenance,
		MaterializedContent: materialized,
	}
	if candidate.Kind == browsertransaction.CandidateAuthentication || candidate.Kind == browsertransaction.CandidateCapability {
		if err := validateVirtualAuthenticatedReview(input.Review, candidate, transaction.Provenance.ObservedAt, at); err != nil {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, err
		}
	}
	switch candidate.Kind {
	case browsertransaction.CandidateAuthentication:
		value, canonical, err := canonicalVirtualAuthentication(input.Source, at)
		if err != nil {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, err
		}
		if !bytes.Equal(canonical, input.Source) {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("authentication source is not canonical JSON")
		}
		if value.Profile != candidate.Schema {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("authentication source schema does not match the transaction")
		}
		plan.Kind, plan.ID, plan.TargetPath, plan.Title = browserAuthenticationSourceFamily, transaction.ID+"-auth", filepath.ToSlash(filepath.Join("browser-authentication", transaction.ID+"-auth.json")), value.Info.Title
		plan.OperationCount, plan.Flows, plan.Origins = len(value.Flows), authprofile.SortedFlowNames(value), virtualAuthenticationOrigins(value)
		if expires, err := authprofile.ExpiresAt(value); err != nil {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, err
		} else {
			plan.ExpiresAt = earliestVirtualExpiry(plan.ExpiresAt, expires)
		}
		plan.FlowCredentialSlots = map[string][]string{}
		for _, flow := range plan.Flows {
			plan.FlowCredentialSlots[flow] = browserAuthenticationFlowSlots(value.Flows[flow])
		}
		flow := strings.TrimSpace(input.Flow)
		if _, ok := value.Flows[flow]; flow == "" || !ok {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("authentication candidate requires one exact available flow")
		}
		plan.Provenance = virtualFlowProvenance(plan.Provenance, flow)
		public.TargetPath, public.Title, public.Flow, public.ProvidesSession = plan.TargetPath, plan.Title, flow, transaction.Session
		doc := browserAuthenticationDocument(plan, value)
		for index := range doc.Operations {
			if doc.Operations[index].OperationID != flow {
				continue
			}
			doc.Operations[index].Extensions["openudon.browser_authentication.session"] = transaction.Session
			doc.Operations[index].Extensions["openudon.browser_authentication.credential_bindings"] = virtualCredentialBindings(transaction.CredentialBindings)
		}
		return public, plan, doc, nil
	case browsertransaction.CandidateCapability:
		if strings.TrimSpace(input.Flow) != "" {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("capability candidate must not declare an authentication or registration flow")
		}
		value, canonical, err := canonicalVirtualCapability(input.Source, at)
		if err != nil {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, err
		}
		if !bytes.Equal(canonical, input.Source) {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("capability source is not canonical JSON")
		}
		if value.Schema != candidate.Schema {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("capability source schema does not match the transaction")
		}
		plan.Kind, plan.ID, plan.TargetPath, plan.Title = browserSourceFamily, transaction.ID, filepath.ToSlash(filepath.Join("browser-profiles", transaction.ID+".json")), value.Info.Title
		plan.OperationCount, plan.Actions, plan.Origins, plan.LoginStateRequired = len(value.Actions), value.SortedActionNames(), virtualCapabilityOrigins(value), value.Info.LoginStateRequired
		if expires, err := browserProfileExpiry(value); err != nil {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, err
		} else {
			plan.ExpiresAt = earliestVirtualExpiry(plan.ExpiresAt, expires)
		}
		dependency := transaction.ID + "/" + string(browsertransaction.CandidateAuthentication)
		public.TargetPath, public.Title, public.Dependencies, public.RequiresSession = plan.TargetPath, plan.Title, []string{dependency}, transaction.Session
		return public, plan, browserProfileDocument(plan, value), nil
	case browsertransaction.CandidateRegistration:
		if transaction.State != browsertransaction.StateReviewed {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("registration candidate requires an explicitly reviewed transaction")
		}
		value, canonical, err := canonicalVirtualRegistration(input.Source, input.Review, at)
		if err != nil {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, err
		}
		if !bytes.Equal(canonical, input.Source) {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("registration source is not canonical JSON")
		}
		if value.Profile != candidate.Schema {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("registration source schema does not match the transaction")
		}
		plan.Kind, plan.ID, plan.TargetPath, plan.Title = browserRegistrationSourceFamily, transaction.ID, filepath.ToSlash(filepath.Join("browser-registration", transaction.ID+".json")), value.Info.Title
		plan.OperationCount, plan.Flows, plan.Origins = len(value.Flows), registrationprofile.SortedFlowNames(value), registrationprofile.Origins(value)
		if expires, err := registrationprofile.ExpiresAt(value); err != nil {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, err
		} else {
			plan.ExpiresAt = earliestVirtualExpiry(plan.ExpiresAt, expires)
		}
		plan.FlowCredentialSlots = map[string][]string{}
		for _, flow := range plan.Flows {
			plan.FlowCredentialSlots[flow] = registrationFlowSlots(value.Flows[flow])
		}
		flow := strings.TrimSpace(input.Flow)
		if _, ok := value.Flows[flow]; flow == "" || !ok {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("registration candidate requires one exact available flow")
		}
		cleanup := strings.TrimSpace(input.CleanupDisposition)
		if cleanup != "delete_separately" && cleanup != "retain_dedicated_test_identity" {
			return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("registration candidate cleanup disposition is invalid")
		}
		plan.Provenance = virtualRegistrationPolicyProvenance(plan.Provenance, flow, cleanup)
		plan.ReviewPath = strings.TrimSuffix(plan.TargetPath, filepath.Ext(plan.TargetPath)) + ".review.json"
		plan.MaterializedReview = append(append([]byte(nil), input.Review...), '\n')
		reviewSum := sha256.Sum256(plan.MaterializedReview)
		plan.ReviewSHA256 = hex.EncodeToString(reviewSum[:])
		public.TargetPath, public.Title, public.Flow, public.CleanupDisposition = plan.TargetPath, plan.Title, flow, cleanup
		return public, plan, browserRegistrationDocument(plan, value, flow, transaction.CredentialBindings, cleanup), nil
	default:
		return VirtualBrowserCandidate{}, SourceMaterialization{}, APIDocument{}, errors.New("candidate kind is unsupported")
	}
}

func virtualCredentialBindings(bindings []browsertransaction.CredentialBinding) string {
	values := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		values = append(values, binding.Slot+"="+binding.Binding)
	}
	return strings.Join(values, ",")
}

func virtualRegistrationPolicyProvenance(provenance, flow, cleanup string) string {
	sum := sha256.Sum256([]byte("flow=" + flow + "\ncleanup=" + cleanup))
	return provenance + ";registration-policy-sha256:" + hex.EncodeToString(sum[:])
}

func canonicalVirtualAuthentication(data []byte, at time.Time) (*authprofile.Profile, []byte, error) {
	value, err := authprofile.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	if err := authprofile.ValidateAt(value, at); err != nil {
		return nil, nil, err
	}
	canonical, err := canonicalVirtualJSON(data)
	return value, canonical, err
}

func canonicalVirtualCapability(data []byte, at time.Time) (*profile.Profile, []byte, error) {
	value, err := profile.ParseJSON(data)
	if err != nil {
		return nil, nil, err
	}
	if err := validateBrowserAuthoringProfile(value); err != nil {
		return nil, nil, err
	}
	expires, err := browserProfileExpiry(value)
	if err != nil || !at.Before(expires) {
		return nil, nil, errors.New("capability source is stale")
	}
	canonical, err := canonicalVirtualJSON(data)
	return value, canonical, err
}

func canonicalVirtualJSON(data []byte) ([]byte, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, errors.New("candidate source contains trailing JSON")
		}
		return nil, err
	}
	return json.Marshal(value)
}

func canonicalVirtualRegistration(data, reviewData []byte, at time.Time) (*registrationprofile.Profile, []byte, error) {
	value, err := registrationprofile.Parse(data)
	if err != nil {
		return nil, nil, err
	}
	if err := registrationprofile.ValidateAt(value, at); err != nil {
		return nil, nil, err
	}
	canonical, err := registrationprofile.MarshalJSON(value)
	if err != nil {
		return nil, nil, err
	}
	var review registrationreview.Bundle
	decoder := json.NewDecoder(bytes.NewReader(reviewData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&review); err != nil {
		return nil, nil, fmt.Errorf("decode registration review: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, nil, errors.New("registration review contains trailing JSON")
		}
		return nil, nil, fmt.Errorf("decode registration review trailing data: %w", err)
	}
	if err := registrationreview.Verify(&review, at); err != nil {
		return nil, nil, err
	}
	canonicalReview, err := json.Marshal(&review)
	if err != nil || !bytes.Equal(canonicalReview, reviewData) {
		return nil, nil, errors.New("registration review is not canonical JSON")
	}
	reviewProfile, err := registrationprofile.MarshalJSON(&review.Profile)
	if err != nil || !bytes.Equal(reviewProfile, canonical) {
		return nil, nil, errors.New("registration review does not bind the exact source")
	}
	return value, canonical, nil
}

func validateVirtualAuthenticatedReview(data []byte, candidate browsertransaction.Candidate, observedAt string, at time.Time) error {
	var review authorresult.Review
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&review); err != nil {
		return fmt.Errorf("decode authenticated candidate review: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("authenticated candidate review contains trailing JSON")
		}
		return fmt.Errorf("decode authenticated candidate review trailing data: %w", err)
	}
	canonical, err := json.Marshal(review)
	if err != nil || !bytes.Equal(canonical, data) {
		return errors.New("authenticated candidate review is not canonical JSON")
	}
	wantKind := string(candidate.Kind)
	if review.Schema != "browsertools.authenticated-profile-review.v1" || review.Kind != wantKind || review.ProfileDigest != candidate.SourceSHA256 {
		return errors.New("authenticated candidate review identity or profile digest mismatch")
	}
	assessed, err := time.Parse(time.RFC3339, review.AssessedAt)
	if err != nil || assessed.UTC().Format(time.RFC3339) != review.AssessedAt || assessed.After(at) || review.AssessedAt != observedAt {
		return errors.New("authenticated candidate review assessment time does not match transaction provenance")
	}
	seen := map[string]bool{}
	for _, decision := range review.Decisions {
		decision = strings.TrimSpace(decision)
		if decision == "" || seen[decision] {
			return errors.New("authenticated candidate review decisions are empty or duplicated")
		}
		seen[decision] = true
	}
	if len(seen) == 0 || !seen[candidate.Schema] {
		return errors.New("authenticated candidate review does not bind the candidate schema decision")
	}
	return nil
}

func registrationFlowSlots(flow browserregistration.Flow) []string {
	var slots []string
	for _, step := range flow.Sequence {
		if step.TypeCredential != nil {
			slots = append(slots, step.TypeCredential.Slot)
		}
	}
	return dedupeStrings(slots)
}

func browserRegistrationDocument(plan SourceMaterialization, value *registrationprofile.Profile, selectedFlow string, bindings []browsertransaction.CredentialBinding, cleanup string) APIDocument {
	doc := APIDocument{ID: plan.ID, Path: plan.SourcePath, RelativePath: plan.TargetPath, Title: plan.Title, Description: "Reviewed, no-submit browser registration recipe. Runtime execution remains unsupported."}
	for _, flowName := range registrationprofile.SortedFlowNames(value) {
		if flowName != selectedFlow {
			continue
		}
		flow := value.Flows[flowName]
		effects := append([]string(nil), flow.Effects...)
		sort.Strings(effects)
		doc.Operations = append(doc.Operations, apitools.OperationSummary{
			ID: flowName, OperationID: flowName, Method: "BROWSER_REGISTRATION", Path: "#/flows/" + flowName,
			Summary: firstNonEmpty(flow.Description, "Review an inert account-registration recipe."), DocumentName: plan.ID,
			DocumentPath: plan.SourcePath, DocumentRelativePath: plan.TargetPath, Provenance: plan.Provenance,
			Extensions: map[string]string{
				"openudon.source_family":                             browserRegistrationSourceFamily,
				"openudon.browser_registration.credential_slots":     strings.Join(plan.FlowCredentialSlots[flowName], ","),
				"openudon.browser_registration.effects":              strings.Join(effects, ","),
				"openudon.browser_registration.runtime_supported":    "false",
				"openudon.browser_registration.credential_bindings":  virtualCredentialBindings(bindings),
				"openudon.browser_registration.duplicate_prevention": "operator_attestation",
				"openudon.browser_registration.on_duplicate":         "fail",
				"openudon.browser_registration.ambiguous_outcome":    "stop_without_retry",
				"openudon.browser_registration.cleanup_disposition":  cleanup,
				"openudon.browser_registration.timeout_seconds":      "300",
				"openudon.browser.expires_at":                        plan.ExpiresAt,
			},
		})
	}
	return doc
}

// MergeVirtualBrowserSources appends virtual browser fallback documents after
// validating that no physical/registry target is shadowed.
func MergeVirtualBrowserSources(discovery LocalSourceDiscovery, virtual VirtualBrowserDiscovery) (LocalSourceDiscovery, error) {
	seenTargets := map[string]string{}
	for _, plan := range discovery.Plans {
		seenTargets[filepath.ToSlash(plan.TargetPath)] = plan.SourcePath
	}
	for _, plan := range virtual.Plans {
		target := filepath.ToSlash(plan.TargetPath)
		if prior, exists := seenTargets[target]; exists {
			return LocalSourceDiscovery{}, fmt.Errorf("virtual browser target %q collides with discovered source %s", target, prior)
		}
		seenTargets[target] = plan.SourcePath
	}
	discovery.Docs = append(discovery.Docs, virtual.Docs...)
	sort.SliceStable(discovery.Docs, func(i, j int) bool {
		left, right := apiDocumentPriority(discovery.Docs[i]), apiDocumentPriority(discovery.Docs[j])
		if left != right {
			return left < right
		}
		return discovery.Docs[i].RelativePath < discovery.Docs[j].RelativePath
	})
	mergedPlans := append([]SourceMaterialization(nil), discovery.Plans...)
	for _, plan := range virtual.Plans {
		mergedPlans = append(mergedPlans, cloneVirtualSourcePlan(plan))
	}
	discovery.Plans = normalizeSourcePlan(mergedPlans)
	return discovery, nil
}

// ValidateVirtualBrowserDiscoveryAt rechecks the retained in-memory source
// identities and earliest source/transaction expiry before every refresh.
func ValidateVirtualBrowserDiscoveryAt(discovery VirtualBrowserDiscovery, at time.Time) error {
	if at.IsZero() {
		return errors.New("virtual browser assessment time is required")
	}
	if len(discovery.Candidates) != len(discovery.Plans) {
		return errors.New("virtual browser catalog candidate and plan counts differ")
	}
	byTarget := map[string]bool{}
	byID := map[string]bool{}
	for _, candidate := range discovery.Candidates {
		if candidate.ID == "" || byID[candidate.ID] || candidate.TargetPath == "" || byTarget[candidate.TargetPath] {
			return errors.New("virtual browser catalog candidate targets are invalid or duplicated")
		}
		byID[candidate.ID] = true
		byTarget[candidate.TargetPath] = true
	}
	seenPlans := map[string]bool{}
	for _, plan := range discovery.Plans {
		if !byTarget[plan.TargetPath] || seenPlans[plan.TargetPath] || !strings.HasPrefix(plan.SourcePath, virtualBrowserPrefix) || plan.SourceKind != virtualBrowserSourceKind {
			return errors.New("virtual browser catalog plan identity is invalid")
		}
		seenPlans[plan.TargetPath] = true
		if err := validateBrowserMaterializationFreshness(plan, at); err != nil {
			return err
		}
		if len(plan.MaterializedContent) == 0 {
			return errors.New("virtual browser catalog materialization is empty")
		}
		sum := sha256.Sum256(plan.MaterializedContent)
		if hex.EncodeToString(sum[:]) != plan.SHA256 {
			return errors.New("virtual browser catalog materialization digest mismatch")
		}
		if plan.Kind == browserRegistrationSourceFamily {
			if plan.ReviewPath == "" || plan.ReviewSHA256 == "" || len(plan.MaterializedReview) == 0 {
				return errors.New("virtual browser registration review materialization is incomplete")
			}
			reviewSum := sha256.Sum256(plan.MaterializedReview)
			if hex.EncodeToString(reviewSum[:]) != plan.ReviewSHA256 {
				return errors.New("virtual browser registration review materialization digest mismatch")
			}
		} else if plan.ReviewPath != "" || plan.ReviewSHA256 != "" || len(plan.MaterializedReview) != 0 {
			return errors.New("non-registration virtual source carries registration review materialization")
		}
	}
	return nil
}

// RequireFreshVirtualBrowserSources rejects a resumed selection unless its
// exact transaction-derived plan exists in the current in-memory catalog.
func RequireFreshVirtualBrowserSources(selected, available []SourceMaterialization) error {
	byTarget := map[string]SourceMaterialization{}
	for _, plan := range available {
		byTarget[plan.TargetPath] = plan
	}
	for _, source := range selected {
		if !strings.HasPrefix(source.SourcePath, virtualBrowserPrefix) {
			continue
		}
		candidate, ok := byTarget[source.TargetPath]
		if !ok || candidate.Kind != source.Kind || candidate.ID != source.ID || candidate.SourcePath != source.SourcePath || candidate.SHA256 != source.SHA256 || candidate.SourceSHA256 != source.SourceSHA256 || candidate.ReviewPath != source.ReviewPath || candidate.ReviewSHA256 != source.ReviewSHA256 || candidate.Provenance != source.Provenance {
			return fmt.Errorf("selected virtual browser source %s is stale or unavailable", source.ID)
		}
	}
	return nil
}

// SelectVirtualBrowserSources replaces the selected virtual subset with the
// dependency closure of candidateIDs while preserving ordinary sources.
func SelectVirtualBrowserSources(session Session, discovery VirtualBrowserDiscovery, candidateIDs []string) (Session, error) {
	closure, err := virtualBrowserDependencyClosure(discovery.Candidates, candidateIDs)
	if err != nil {
		return Session{}, err
	}
	byCandidate := map[string]SourceMaterialization{}
	for index, candidate := range discovery.Candidates {
		if index >= len(discovery.Plans) {
			return Session{}, errors.New("virtual browser catalog plan index is incomplete")
		}
		// Plans are independently sorted by target, so bind through target rather
		// than relying on candidate order.
		for _, plan := range discovery.Plans {
			if plan.TargetPath == candidate.TargetPath {
				byCandidate[candidate.ID] = plan
				break
			}
		}
	}
	next, err := cloneSession(session)
	if err != nil {
		return Session{}, err
	}
	previous := append([]SourceMaterialization(nil), next.SourcePlan...)
	kept := next.SourcePlan[:0]
	for _, plan := range next.SourcePlan {
		if !strings.HasPrefix(plan.SourcePath, virtualBrowserPrefix) {
			kept = append(kept, plan)
		}
	}
	for _, id := range closure {
		plan, ok := byCandidate[id]
		if !ok {
			return Session{}, fmt.Errorf("virtual browser candidate %q has no source plan", id)
		}
		kept = append(kept, cloneVirtualSourcePlan(plan))
	}
	next.SourcePlan = normalizeSourcePlan(kept)
	invalidateChangedVirtualSourceApprovals(&next, previous, next.SourcePlan)
	return next, nil
}

func invalidateChangedVirtualSourceApprovals(session *Session, previous, current []SourceMaterialization) {
	if session == nil {
		return
	}
	previousByTarget := virtualSourceIdentities(previous)
	currentByTarget := virtualSourceIdentities(current)
	changed := map[string]bool{}
	for target, identity := range previousByTarget {
		if currentByTarget[target] != identity {
			changed[target] = true
		}
	}
	for target, identity := range currentByTarget {
		if previousByTarget[target] != identity {
			changed[target] = true
		}
	}
	walkSteps(session.Intent.Steps, func(step *rollout.Step) {
		if step == nil || !changed[filepath.ToSlash(stepAPISourceRef(*session, step))] {
			return
		}
		switch strings.ToLower(strings.TrimSpace(step.Type)) {
		case "browser_registration":
			step.RegistrationApproval = ""
		case "browser_authentication":
			session.BrowserAuthenticationApprovals = removeString(session.BrowserAuthenticationApprovals, step.Name)
		case "browser":
			session.BrowserApprovals = removeString(session.BrowserApprovals, step.Name)
		}
	})
}

func virtualSourceIdentities(sources []SourceMaterialization) map[string]string {
	result := map[string]string{}
	for _, source := range sources {
		if !strings.HasPrefix(source.SourcePath, virtualBrowserPrefix) {
			continue
		}
		target := filepath.ToSlash(source.TargetPath)
		result[target] = strings.Join([]string{
			source.Kind, source.ID, source.SourcePath, source.SHA256, source.SourceSHA256,
			source.ReviewPath, source.ReviewSHA256, source.Provenance,
		}, "\x00")
	}
	return result
}

func virtualBrowserDependencyClosure(candidates []VirtualBrowserCandidate, requested []string) ([]string, error) {
	byID := map[string]VirtualBrowserCandidate{}
	for _, candidate := range candidates {
		if candidate.ID == "" || byID[candidate.ID].ID != "" {
			return nil, errors.New("virtual browser candidate IDs are invalid or duplicated")
		}
		byID[candidate.ID] = candidate
	}
	requested = append([]string(nil), requested...)
	for index := range requested {
		requested[index] = strings.TrimSpace(requested[index])
	}
	sort.Strings(requested)
	for index := 1; index < len(requested); index++ {
		if requested[index] == requested[index-1] {
			return nil, fmt.Errorf("virtual browser candidate %q was selected more than once", requested[index])
		}
	}
	state := map[string]uint8{}
	selected := map[string]bool{}
	var visit func(string, int) error
	visit = func(id string, depth int) error {
		if depth > 32 {
			return errors.New("virtual browser dependency depth exceeds 32")
		}
		candidate, ok := byID[id]
		if !ok {
			return fmt.Errorf("virtual browser candidate %q is unavailable", id)
		}
		if state[id] == 1 {
			return fmt.Errorf("virtual browser dependency cycle includes %q", id)
		}
		if state[id] == 2 {
			selected[id] = true
			return nil
		}
		state[id] = 1
		dependencies := append([]string(nil), candidate.Dependencies...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency, depth+1); err != nil {
				return err
			}
		}
		state[id] = 2
		selected[id] = true
		return nil
	}
	for _, id := range requested {
		if err := visit(id, 0); err != nil {
			return nil, err
		}
	}
	result := make([]string, 0, len(selected))
	for id := range selected {
		result = append(result, id)
	}
	sort.Strings(result)
	return result, nil
}

func cloneVirtualBrowserInputs(inputs []VirtualBrowserTransactionInput) []VirtualBrowserTransactionInput {
	result := make([]VirtualBrowserTransactionInput, len(inputs))
	for index, input := range inputs {
		result[index].Transaction = input.Transaction
		result[index].Transaction.Candidates = append([]browsertransaction.Candidate(nil), input.Transaction.Candidates...)
		result[index].Transaction.Provenance.Origins = append([]string(nil), input.Transaction.Provenance.Origins...)
		if input.Transaction.CredentialBindings != nil {
			result[index].Transaction.CredentialBindings = append(make([]browsertransaction.CredentialBinding, 0, len(input.Transaction.CredentialBindings)), input.Transaction.CredentialBindings...)
		}
		result[index].Sources = make([]VirtualBrowserSourceInput, len(input.Sources))
		for sourceIndex, source := range input.Sources {
			result[index].Sources[sourceIndex] = VirtualBrowserSourceInput{Kind: source.Kind, Flow: source.Flow, CleanupDisposition: source.CleanupDisposition, Source: append([]byte(nil), source.Source...), Review: append([]byte(nil), source.Review...)}
		}
	}
	return result
}

func cloneVirtualBrowserDiscovery(discovery VirtualBrowserDiscovery) VirtualBrowserDiscovery {
	result := VirtualBrowserDiscovery{Candidates: make([]VirtualBrowserCandidate, len(discovery.Candidates)), Docs: append([]APIDocument(nil), discovery.Docs...), Plans: make([]SourceMaterialization, len(discovery.Plans))}
	for index, candidate := range discovery.Candidates {
		result.Candidates[index] = candidate
		result.Candidates[index].Dependencies = append([]string(nil), candidate.Dependencies...)
		result.Candidates[index].CredentialBindings = append([]browsertransaction.CredentialBinding(nil), candidate.CredentialBindings...)
	}
	for index := range discovery.Plans {
		result.Plans[index] = cloneVirtualSourcePlan(discovery.Plans[index])
	}
	return result
}

func cloneVirtualSourcePlan(plan SourceMaterialization) SourceMaterialization {
	result := plan
	result.Actions = append([]string(nil), plan.Actions...)
	result.Flows = append([]string(nil), plan.Flows...)
	result.Origins = append([]string(nil), plan.Origins...)
	result.MaterializedContent = append([]byte(nil), plan.MaterializedContent...)
	result.MaterializedReview = append([]byte(nil), plan.MaterializedReview...)
	if plan.FlowCredentialSlots != nil {
		result.FlowCredentialSlots = make(map[string][]string, len(plan.FlowCredentialSlots))
		for flow, slots := range plan.FlowCredentialSlots {
			result.FlowCredentialSlots[flow] = append([]string(nil), slots...)
		}
	}
	return result
}

func candidateIDs(candidates []VirtualBrowserCandidate) []string {
	result := make([]string, len(candidates))
	for index := range candidates {
		result[index] = candidates[index].ID
	}
	return result
}

func digestVirtualBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func earliestVirtualExpiry(transactionExpiry string, sourceExpiry time.Time) string {
	transactionTime, err := time.Parse(time.RFC3339Nano, transactionExpiry)
	if err == nil && transactionTime.Before(sourceExpiry) {
		return transactionTime.Format(time.RFC3339Nano)
	}
	return sourceExpiry.UTC().Format(time.RFC3339)
}

func sameVirtualOrigins(expected []string, actual map[string]bool) bool {
	if len(expected) != len(actual) {
		return false
	}
	for _, origin := range expected {
		if !actual[origin] {
			return false
		}
	}
	return true
}

func sameVirtualBindingSlots(bindings []browsertransaction.CredentialBinding, slots map[string]bool) bool {
	if len(bindings) != len(slots) {
		return false
	}
	for _, binding := range bindings {
		if !slots[binding.Slot] {
			return false
		}
	}
	return true
}

func virtualFlowProvenance(provenance, flow string) string {
	sum := sha256.Sum256([]byte(flow))
	return provenance + ";flow-sha256:" + hex.EncodeToString(sum[:])
}

func virtualAuthenticationOrigins(value *authprofile.Profile) []string {
	origins := append([]string(nil), authprofile.Origins(value)...)
	if value != nil {
		for _, context := range value.Contexts {
			origins = append(origins, context.Origin)
		}
	}
	return sortedUniqueBrowserStrings(origins)
}

func virtualCapabilityOrigins(value *profile.Profile) []string {
	if value == nil {
		return nil
	}
	origins := append([]string(nil), value.Info.Origin...)
	for _, context := range value.Contexts {
		origins = append(origins, context.Origin)
	}
	return sortedUniqueBrowserStrings(origins)
}
