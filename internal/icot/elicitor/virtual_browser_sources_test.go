package elicitor

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authorresult"
	"github.com/OpenUdon/browsertools/authprofile"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/registrationprofile"
	"github.com/OpenUdon/browsertools/registrationreview"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

var virtualBrowserTime = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)

func TestDiscoverVirtualBrowserSourcesComposesAndSelectsDependencies(t *testing.T) {
	input := virtualAuthenticationCapabilityInput(t, "account")
	discovery, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{input}, virtualBrowserTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Candidates) != 2 || discovery.Candidates[0].Kind != browsertransaction.CandidateAuthentication || discovery.Candidates[1].Kind != browsertransaction.CandidateCapability {
		t.Fatalf("candidates = %#v", discovery.Candidates)
	}
	authentication, capability := discovery.Candidates[0], discovery.Candidates[1]
	if authentication.ProvidesSession != "account_session" || capability.RequiresSession != "account_session" || len(capability.Dependencies) != 1 || capability.Dependencies[0] != authentication.ID {
		t.Fatalf("compatibility metadata = auth %#v capability %#v", authentication, capability)
	}
	if len(discovery.Docs) != 2 || len(discovery.Plans) != 2 {
		t.Fatalf("virtual discovery = docs %#v plans %#v", discovery.Docs, discovery.Plans)
	}
	selected, err := SelectVirtualBrowserSources(Session{}, discovery, []string{capability.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.SourcePlan) != 2 {
		t.Fatalf("dependency-closed selection = %#v", selected.SourcePlan)
	}
	for _, plan := range selected.SourcePlan {
		if !strings.HasPrefix(plan.SourcePath, virtualBrowserPrefix) || len(plan.MaterializedContent) == 0 {
			t.Fatalf("selected plan is not in-memory virtual content: %#v", plan)
		}
	}
	input.Sources[0].Source[0] ^= 0xff
	if discovery.Plans[0].MaterializedContent[0] == input.Sources[0].Source[0] {
		t.Fatal("virtual discovery retained caller-owned bytes")
	}
	public, err := json.Marshal(discovery)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range input.Sources {
		if bytes.Contains(public, source.Source) || bytes.Contains(public, source.Review) {
			t.Fatalf("public catalog exposed private source or review: %s", public)
		}
	}
}

func TestVirtualAuthenticationLoweringPreservesTransactionSessionAndBindings(t *testing.T) {
	input := virtualAuthenticationCapabilityInput(t, "account")
	discovery, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{input}, virtualBrowserTime)
	if err != nil {
		t.Fatal(err)
	}
	var authenticationDoc, capabilityDoc APIDocument
	for _, doc := range discovery.Docs {
		if isBrowserAuthenticationDocument(doc) {
			authenticationDoc = doc
		} else if isBrowserActionDocument(doc) {
			capabilityDoc = doc
		}
	}
	if len(authenticationDoc.Operations) != 1 || len(capabilityDoc.Operations) == 0 {
		t.Fatalf("virtual documents = %#v", discovery.Docs)
	}
	action := &rollout.Step{Name: "read_account", Type: "browser", Source: capabilityDoc.RelativePath, Operation: capabilityDoc.Operations[0].OperationID}
	session := Session{Intent: rollout.Intent{Steps: []*rollout.Step{action}}}
	authentication := insertBrowserAuthenticationStep(&session, action, authenticationDoc, &authenticationDoc.Operations[0])
	if authentication == nil || authentication.BrowserSession != input.Transaction.Session || action.BrowserSession != input.Transaction.Session {
		t.Fatalf("lowered session contract = auth %#v action %#v", authentication, action)
	}
	wantBindings := map[string]string{"password": "account_password", "username": "account_username"}
	if !exactBrowserCredentialBindings(authentication.CredentialBindings, []string{"password", "username"}) ||
		authentication.CredentialBindings["password"] != wantBindings["password"] || authentication.CredentialBindings["username"] != wantBindings["username"] ||
		!session.CredentialsSet || strings.Join(session.Credentials, ",") != "account_password,account_username" {
		t.Fatalf("lowered symbolic bindings = step %#v inventory %#v", authentication.CredentialBindings, session.Credentials)
	}
	second := Session{Intent: rollout.Intent{Steps: []*rollout.Step{{Name: "read_account", Type: "browser", Source: capabilityDoc.RelativePath, Operation: capabilityDoc.Operations[0].OperationID}}}}
	secondAuthentication := insertBrowserAuthenticationStep(&second, second.Intent.Steps[0], authenticationDoc, &authenticationDoc.Operations[0])
	firstBytes, _ := json.Marshal(session.Intent)
	secondBytes, _ := json.Marshal(second.Intent)
	if secondAuthentication == nil || !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf("virtual lowering is unstable:\n%s\n%s", firstBytes, secondBytes)
	}
	timeout := 120.0
	authentication.Timeout = &timeout
	session.BrowserAuthenticationApprovals = []string{authentication.Name}
	authentication.BrowserSession = "different_session"
	if issues := browserAuthenticationReadinessIssues(session, discovery.Docs, authentication); !hasBrowserAuthenticationReadinessCode(issues, readinessMissingBrowserAuthenticationSession) {
		t.Fatalf("transaction session drift was not rejected: %#v", issues)
	}
	authentication.BrowserSession = input.Transaction.Session
	authentication.CredentialBindings["password"] = "different_password"
	if issues := browserAuthenticationReadinessIssues(session, discovery.Docs, authentication); !hasBrowserAuthenticationReadinessCode(issues, readinessMissingBrowserCredentialBindings) {
		t.Fatalf("transaction binding drift was not rejected: %#v", issues)
	}
}

func TestDiscoverVirtualBrowserSourcesRejectsAuthenticatedReviewDrift(t *testing.T) {
	input := virtualAuthenticationCapabilityInput(t, "account")
	var review authorresult.Review
	if err := json.Unmarshal(input.Sources[0].Review, &review); err != nil {
		t.Fatal(err)
	}
	review.Kind = "capability"
	changed, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	input.Sources[0].Review = changed
	input.Transaction.Candidates[0].ReviewSHA256 = digestVirtualBytes(changed)
	if _, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{input}, virtualBrowserTime); err == nil || !strings.Contains(err.Error(), "identity") {
		t.Fatalf("review drift error = %v", err)
	}
}

func TestDiscoverVirtualBrowserSourcesAcceptsActualBrowsertoolsEncoding(t *testing.T) {
	proof := authorresult.GoalProof{Origin: "https://members.example.test", Path: "/dashboard", Context: "main", Role: "heading", Label: "Dashboard", Matches: 1}
	envelope, err := authorresult.Build(authorresult.BuildRequest{
		ObservedAt: virtualBrowserTime, Title: "Members", Goal: "reach the member dashboard",
		InitialURL: "https://members.example.test/login", DashboardURL: "https://members.example.test/dashboard",
		Origins: []string{"https://members.example.test"}, Contexts: map[string]authorresult.Context{},
		Bounds:        authorresult.Bounds{NavigationTimeoutMS: 20_000, TotalTimeoutMS: 600_000, MaxRequests: 128, MaxResponseBytes: 8 << 20, MaxObservations: 32, MaxCandidates: 32, MaxOutputs: 8},
		GoalPredicate: authorresult.GoalPredicate{Origin: proof.Origin, Path: proof.Path, Context: proof.Context, Role: proof.Role, Label: proof.Label},
		GoalProof:     proof, AuthenticationProof: proof, HumanConfirmed: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	authentication, err := authprofile.Parse(envelope.AuthenticationProfile)
	if err != nil {
		t.Fatal(err)
	}
	capability, err := profile.ParseJSON(envelope.CapabilityProfile)
	if err != nil {
		t.Fatal(err)
	}
	flows := authprofile.SortedFlowNames(authentication)
	if len(flows) != 1 {
		t.Fatalf("producer authentication flows = %v", flows)
	}
	bindings := make([]browsertransaction.CredentialBinding, 0)
	for _, slot := range browserAuthenticationFlowSlots(authentication.Flows[flows[0]]) {
		bindings = append(bindings, browsertransaction.CredentialBinding{Slot: slot, Binding: "binding_" + slot})
	}
	sort.Slice(bindings, func(i, j int) bool { return bindings[i].Slot < bindings[j].Slot })
	authenticationReview, _ := json.Marshal(envelope.AuthenticationReview)
	capabilityReview, _ := json.Marshal(envelope.CapabilityReview)
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: "members", Kind: browsertransaction.KindAuthenticationCapability, State: browsertransaction.StateCandidate,
		Candidates: []browsertransaction.Candidate{
			{Kind: browsertransaction.CandidateAuthentication, Schema: authentication.Profile, SourceSHA256: digestVirtualBytes(envelope.AuthenticationProfile), ReviewSHA256: digestVirtualBytes(authenticationReview)},
			{Kind: browsertransaction.CandidateCapability, Schema: capability.Schema, SourceSHA256: digestVirtualBytes(envelope.CapabilityProfile), ReviewSHA256: digestVirtualBytes(capabilityReview)},
		},
		Provenance:         browsertransaction.Provenance{Producer: "browsertools", ResultVersion: browsertransaction.ResultAuthenticatedAuthoringV2, ResultSHA256: digestVirtualBytes([]byte("private envelope")), ObservedAt: envelope.ObservedAt, ExpiresAt: virtualBrowserTime.Add(time.Hour).Format(time.RFC3339Nano), Origins: append([]string(nil), envelope.Origins...)},
		CredentialBindings: bindings, Session: "members_session",
	}
	discovery, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{{Transaction: transaction, Sources: []VirtualBrowserSourceInput{
		{Kind: browsertransaction.CandidateAuthentication, Flow: flows[0], Source: envelope.AuthenticationProfile, Review: authenticationReview},
		{Kind: browsertransaction.CandidateCapability, Source: envelope.CapabilityProfile, Review: capabilityReview},
	}}}, virtualBrowserTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Candidates) != 2 || discovery.Candidates[0].Flow != flows[0] {
		t.Fatalf("actual producer discovery = %#v", discovery.Candidates)
	}
}

func TestDiscoverVirtualRegistrationIsSessionFreeAndReviewBound(t *testing.T) {
	input := virtualRegistrationInput(t, "new-account")
	discovery, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{input}, virtualBrowserTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Candidates) != 1 || discovery.Candidates[0].ProvidesSession != "" || discovery.Candidates[0].RequiresSession != "" || len(discovery.Candidates[0].Dependencies) != 0 {
		t.Fatalf("registration candidate = %#v", discovery.Candidates)
	}
	if discovery.Plans[0].Kind != browserRegistrationSourceFamily || discovery.Plans[0].TargetPath != "browser-registration/new-account.json" || discovery.Docs[0].Operations[0].Method != "BROWSER_REGISTRATION" {
		t.Fatalf("registration virtual source = plan %#v doc %#v", discovery.Plans[0], discovery.Docs[0])
	}
	changed := input
	changed.Sources = append([]VirtualBrowserSourceInput(nil), input.Sources...)
	changed.Sources[0].Review = append([]byte(nil), input.Sources[0].Review...)
	changed.Sources[0].Review[0] ^= 0xff
	if _, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{changed}, virtualBrowserTime); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("changed review error = %v", err)
	}
}

func TestVirtualBrowserDiscoveryRejectsStaleVersionAndCollisions(t *testing.T) {
	input := virtualAuthenticationCapabilityInput(t, "account")
	if _, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{input}, transactionExpiry(input.Transaction)); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale error = %v", err)
	}
	unsupported := input
	unsupported.Transaction.Candidates = append([]browsertransaction.Candidate(nil), input.Transaction.Candidates...)
	unsupported.Transaction.Candidates[0].Schema = "uws.browser-authentication.9.9"
	if _, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{unsupported}, virtualBrowserTime); err == nil || !strings.Contains(err.Error(), "schema") {
		t.Fatalf("unsupported version error = %v", err)
	}
	discovery, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{input}, virtualBrowserTime)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateVirtualBrowserDiscoveryAt(discovery, transactionExpiry(input.Transaction)); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("retained stale catalog error = %v", err)
	}
	physical := LocalSourceDiscovery{Plans: []SourceMaterialization{{SourcePath: "/reviewed/profile.json", TargetPath: discovery.Plans[0].TargetPath}}}
	if _, err := MergeVirtualBrowserSources(physical, discovery); err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("target collision error = %v", err)
	}
	if _, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{input, input}, virtualBrowserTime); err == nil || !strings.Contains(err.Error(), "duplicated") {
		t.Fatalf("transaction collision error = %v", err)
	}
}

func TestMergeVirtualBrowserSourcesPreservesAPIFirstPreference(t *testing.T) {
	virtual, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{virtualAuthenticationCapabilityInput(t, "account")}, virtualBrowserTime)
	if err != nil {
		t.Fatal(err)
	}
	api := APIDocument{ID: "api", RelativePath: "openapi/api.json", Title: "Account API"}
	merged, err := MergeVirtualBrowserSources(LocalSourceDiscovery{Docs: []APIDocument{api}}, virtual)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Docs) != 3 || merged.Docs[0].ID != "api" || apiDocumentPriority(merged.Docs[0]) >= apiDocumentPriority(merged.Docs[1]) {
		t.Fatalf("API-first order = %#v", merged.Docs)
	}
}

func TestVirtualBrowserDependencyTraversalRejectsMissingCycleAndDuplicates(t *testing.T) {
	candidates := []VirtualBrowserCandidate{
		{ID: "a", Dependencies: []string{"b"}},
		{ID: "b", Dependencies: []string{"c"}},
		{ID: "c"},
	}
	closure, err := virtualBrowserDependencyClosure(candidates, []string{"a"})
	if err != nil || strings.Join(closure, ",") != "a,b,c" {
		t.Fatalf("nested closure = %v, %v", closure, err)
	}
	if _, err := virtualBrowserDependencyClosure(candidates, []string{"missing"}); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatalf("missing dependency error = %v", err)
	}
	cyclic := append([]VirtualBrowserCandidate(nil), candidates...)
	cyclic[2].Dependencies = []string{"a"}
	if _, err := virtualBrowserDependencyClosure(cyclic, []string{"a"}); err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("cycle error = %v", err)
	}
	if _, err := virtualBrowserDependencyClosure(candidates, []string{" a", "a"}); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("duplicate selection error = %v", err)
	}
}

func TestRequireFreshVirtualBrowserSourcesRejectsReplacement(t *testing.T) {
	discovery, err := DiscoverVirtualBrowserSources([]VirtualBrowserTransactionInput{virtualRegistrationInput(t, "new-account")}, virtualBrowserTime)
	if err != nil {
		t.Fatal(err)
	}
	selected := append([]SourceMaterialization(nil), discovery.Plans...)
	selected[0].MaterializedContent = nil
	if err := RequireFreshVirtualBrowserSources(selected, discovery.Plans); err != nil {
		t.Fatal(err)
	}
	selected[0].SourceSHA256 = strings.Repeat("f", 64)
	if err := RequireFreshVirtualBrowserSources(selected, discovery.Plans); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("replacement error = %v", err)
	}
}

func virtualAuthenticationCapabilityInput(t *testing.T, id string) VirtualBrowserTransactionInput {
	t.Helper()
	authentication, err := authprofile.Parse(browserAuthenticationFixture())
	if err != nil {
		t.Fatal(err)
	}
	authenticationBytes, err := authprofile.MarshalJSON(authentication)
	if err != nil {
		t.Fatal(err)
	}
	authenticationBytes = canonicalVirtualTestJSON(t, authenticationBytes)
	capability, err := profile.ParseYAML(browserProfileFixture(true, false))
	if err != nil {
		t.Fatal(err)
	}
	capabilityBytes, err := json.Marshal(capability)
	if err != nil {
		t.Fatal(err)
	}
	capabilityBytes = canonicalVirtualTestJSON(t, capabilityBytes)
	assessedAt := virtualBrowserTime.Add(-time.Hour).Format(time.RFC3339)
	authenticationReview, err := json.Marshal(authorresult.Review{
		Schema: "browsertools.authenticated-profile-review.v1", Kind: "authentication", ProfileDigest: digestVirtualBytes(authenticationBytes),
		AssessedAt: assessedAt, Decisions: []string{"human_credentials", authentication.Profile},
	})
	if err != nil {
		t.Fatal(err)
	}
	capabilityReview, err := json.Marshal(authorresult.Review{
		Schema: "browsertools.authenticated-profile-review.v1", Kind: "capability", ProfileDigest: digestVirtualBytes(capabilityBytes),
		AssessedAt: assessedAt, Decisions: []string{"selected_trace", capability.Schema},
	})
	if err != nil {
		t.Fatal(err)
	}
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: id, Kind: browsertransaction.KindAuthenticationCapability, State: browsertransaction.StateCandidate,
		Candidates: []browsertransaction.Candidate{
			{Kind: browsertransaction.CandidateAuthentication, Schema: authentication.Profile, SourceSHA256: digestVirtualBytes(authenticationBytes), ReviewSHA256: digestVirtualBytes(authenticationReview)},
			{Kind: browsertransaction.CandidateCapability, Schema: capability.Schema, SourceSHA256: digestVirtualBytes(capabilityBytes), ReviewSHA256: digestVirtualBytes(capabilityReview)},
		},
		Provenance: browsertransaction.Provenance{
			Producer: "browsertools", ResultVersion: browsertransaction.ResultAuthenticatedAuthoringV2,
			ResultSHA256: digestVirtualBytes([]byte("private result")), ObservedAt: virtualBrowserTime.Add(-time.Hour).Format(time.RFC3339Nano),
			ExpiresAt: virtualBrowserTime.Add(24 * time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://example.test", "https://login.example.test"},
		},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "password", Binding: "account_password"}, {Slot: "username", Binding: "account_username"}}, Session: id + "_session",
	}
	return VirtualBrowserTransactionInput{Transaction: transaction, Sources: []VirtualBrowserSourceInput{
		{Kind: browsertransaction.CandidateAuthentication, Flow: "member_login_push", Source: authenticationBytes, Review: authenticationReview},
		{Kind: browsertransaction.CandidateCapability, Source: capabilityBytes, Review: capabilityReview},
	}}
}

func virtualRegistrationInput(t *testing.T, id string) VirtualBrowserTransactionInput {
	t.Helper()
	value, err := registrationprofile.Parse([]byte(`profile: uws.browser-registration.1.0
info:
  title: Synthetic registration
  applicationOrigins: [https://app.example.test]
  registrationOrigins: [https://app.example.test]
observationKind: accessibility_snapshot
evidence: {learnedAt: "2026-08-25T00:00:00Z", source: synthetic_fixture}
confidence: high
expiresAfter: P30D
verification: {lastVerifiedAt: "2026-08-25T00:00:00Z"}
credentialSlots:
  identifier: {kind: identifier}
flows:
  create_account:
    sequence:
      - navigate: https://app.example.test/register
      - type_credential: {locator: {role: textbox}, slot: identifier}
      - submit: {locator: {role: button, name: Register}}
      - wait_for: {locator: {role: status}}
    effects: [creates_account]
    confirmationPolicy: {required: true}
    success: {origin: https://app.example.test, locator: {role: status}}
`))
	if err != nil {
		t.Fatal(err)
	}
	source, err := registrationprofile.MarshalJSON(value)
	if err != nil {
		t.Fatal(err)
	}
	review, err := registrationreview.Build(value, virtualBrowserTime)
	if err != nil {
		t.Fatal(err)
	}
	reviewBytes, err := json.Marshal(review)
	if err != nil {
		t.Fatal(err)
	}
	transaction := browsertransaction.Transaction{
		Version: browsertransaction.Version, ID: id, Kind: browsertransaction.KindRegistration, State: browsertransaction.StateCandidate,
		Candidates: []browsertransaction.Candidate{{Kind: browsertransaction.CandidateRegistration, Schema: value.Profile, SourceSHA256: digestVirtualBytes(source), ReviewSHA256: digestVirtualBytes(reviewBytes)}},
		Provenance: browsertransaction.Provenance{
			Producer: "browsertools", ResultVersion: browsertransaction.ResultRegistrationAuthoringV1,
			ResultSHA256: digestVirtualBytes([]byte("registration result")), ObservedAt: virtualBrowserTime.Add(-time.Hour).Format(time.RFC3339Nano),
			ExpiresAt: virtualBrowserTime.Add(24 * time.Hour).Format(time.RFC3339Nano), Origins: []string{"https://app.example.test"},
		},
		CredentialBindings: []browsertransaction.CredentialBinding{{Slot: "identifier", Binding: "registration_identifier"}},
	}
	return VirtualBrowserTransactionInput{Transaction: transaction, Sources: []VirtualBrowserSourceInput{{Kind: browsertransaction.CandidateRegistration, Flow: "create_account", Source: source, Review: reviewBytes}}}
}

func transactionExpiry(transaction browsertransaction.Transaction) time.Time {
	value, _ := time.Parse(time.RFC3339Nano, transaction.Provenance.ExpiresAt)
	return value
}

func canonicalVirtualTestJSON(t *testing.T, data []byte) []byte {
	t.Helper()
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}
