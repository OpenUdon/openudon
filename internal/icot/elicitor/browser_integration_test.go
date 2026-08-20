package elicitor

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/browsertools/bundle"
	bevidence "github.com/OpenUdon/browsertools/evidence"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/browsertools/registry"
	"github.com/OpenUdon/browsertools/review"
	"github.com/OpenUdon/openudon/internal/authoring"
	"github.com/OpenUdon/openudon/internal/projectwizard"
	rollout "github.com/OpenUdon/openudon/internal/workflowintent"
)

var browserIntegrationTime = time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC)

func TestDiscoverAuthoringSourcesWithBrowserProfileAndAPIPreference(t *testing.T) {
	root := t.TempDir()
	profilePath := filepath.Join(root, "status.browser.json")
	writeBrowserTestFile(t, profilePath, browserProfileFixture(false, false))
	discovery, err := DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "read status", nil, nil, []BrowserSourceInput{{ID: "status", Path: profilePath}}, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(discovery.Docs) != 1 || !isBrowserDocument(discovery.Docs[0]) || discovery.Docs[0].Operations[0].OperationID != "read_status" {
		t.Fatalf("browser discovery = %#v", discovery.Docs)
	}
	if len(discovery.Report.Ambiguous) != 0 || len(discovery.BrowserReport.Ambiguous) != 0 {
		t.Fatalf("cross-family profile remained ambiguous: api=%#v browser=%#v", discovery.Report.Ambiguous, discovery.BrowserReport.Ambiguous)
	}
	if len(discovery.Plans) != 1 || discovery.Plans[0].TargetPath != "browser-profiles/status.json" || discovery.Plans[0].Lifecycle != "active" {
		t.Fatalf("browser source plan = %#v", discovery.Plans)
	}

	openAPIPath := filepath.Join(root, "status.openapi.yaml")
	writeBrowserTestFile(t, openAPIPath, []byte("openapi: 3.0.3\ninfo:\n  title: Status API\n  version: 1.0.0\npaths:\n  /status:\n    get:\n      operationId: read_status\n      responses:\n        '200':\n          description: ok\n"))
	mixed, err := DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "read status", []apitools.LocalSource{{Kind: "openapi", ID: "status-api", Path: openAPIPath}}, nil, []BrowserSourceInput{{ID: "status-ui", Path: profilePath}}, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	preferred := preferredSourceDocument(Session{Boundary: WorkflowBoundary{Outcome: "read status"}}, mixed.Docs)
	if isBrowserDocument(preferred) || preferred.RelativePath == "" {
		t.Fatalf("API source was not preferred for covered capability: %#v", preferred)
	}
}

func TestSourceMaterializationContentRechecksCachedProfileExpiry(t *testing.T) {
	data := []byte("cached browser profile\n")
	digest := sha256.Sum256(data)
	expiresAt := browserIntegrationTime.Add(time.Hour)
	source := SourceMaterialization{
		Kind: browserSourceFamily, ID: "status", SHA256: fmt.Sprintf("%x", digest),
		Lifecycle: "active", ExpiresAt: expiresAt.Format(time.RFC3339), MaterializedContent: data,
	}
	materialized, err := SourceMaterializationContent(source, expiresAt.Add(-time.Nanosecond))
	if err != nil || string(materialized) != string(data) {
		t.Fatalf("current cached profile = %q, %v", materialized, err)
	}
	if _, err := SourceMaterializationContent(source, expiresAt); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected exact-expiry rejection, got %v", err)
	}
}

func TestBrowserProfileWinsOnlyForAPICapabilityGap(t *testing.T) {
	root := t.TempDir()
	profilePath := filepath.Join(root, "status.json")
	openAPIPath := filepath.Join(root, "billing.yaml")
	writeBrowserTestFile(t, profilePath, browserProfileFixture(false, false))
	writeBrowserTestFile(t, openAPIPath, []byte("openapi: 3.0.3\ninfo:\n  title: Billing\n  version: 1.0.0\npaths:\n  /invoice:\n    post:\n      operationId: create_invoice\n      responses:\n        '200':\n          description: ok\n"))
	discovery, err := DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "read status", []apitools.LocalSource{{Kind: "openapi", ID: "billing", Path: openAPIPath}}, nil, []BrowserSourceInput{{ID: "status", Path: profilePath}}, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	preferred := preferredSourceDocument(Session{Boundary: WorkflowBoundary{Outcome: "read status"}}, discovery.Docs)
	if !isBrowserDocument(preferred) {
		t.Fatalf("browser fallback was not selected for uncovered API capability: %#v", preferred)
	}
}

func TestBrowserFrontierRequiresSessionPostureAndMutationApproval(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "editor.json")
	writeBrowserTestFile(t, path, browserProfileFixture(true, true))
	discovery, err := DiscoverAuthoringSourcesWithBrowser(context.Background(), filepath.Join(root, "example"), "update record", nil, nil, []BrowserSourceInput{{ID: "editor", Path: path}}, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	doc := discovery.Docs[0]
	session := Session{
		Boundary: WorkflowBoundary{Outcome: "update record", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"updated output is returned"}},
		Project:  projectwizard.Answers{ProjectName: "Editor", Goal: "update record", Fallback: "stop cleanly", Safety: "Execution requires approval and a sandbox proof run.", SideEffectScope: projectwizard.SideEffectAfterApproval},
		Intent: rollout.Intent{
			Workflow: &rollout.WorkflowMeta{Name: "editor", Description: "update record"}, Source: doc.RelativePath,
			Inputs:  []*rollout.Input{{Name: "note", Type: "string", Required: true}},
			Steps:   []*rollout.Step{{Name: "update", Type: "browser", Source: doc.RelativePath, Operation: "update_record", With: map[string]string{"note": "inputs.note"}}},
			Outputs: []*rollout.Output{{Name: "result", From: "update.received_body.status"}},
		},
		SourcePlan: discovery.Plans, BrowserRoute: "browser", SideEffectScope: projectwizard.SideEffectAfterApproval,
		Fallback: "stop cleanly", FallbackSet: true, Safety: "after approval", SafetySet: true, CredentialsSet: true,
	}
	session.Normalize()
	issues := CheckReadiness(session, discovery.Docs)
	if !hasReadinessCode(issues, "missing_browser_session_posture") || !hasReadinessCode(issues, "unconfirmed_browser_mutation") {
		t.Fatalf("browser readiness issues = %#v", issues)
	}
	frontier, err := PlanFrontier(&session, discovery.Docs, issues)
	if err != nil {
		t.Fatal(err)
	}
	if !hasQuestionID(frontier, nodeBrowserSession) || !hasQuestionID(frontier, nodeBrowserApproval) {
		t.Fatalf("browser frontier = %#v", frontier)
	}
	if err := ApplyFrontierRound(&session, []authoring.RoundAnswer{
		{QuestionID: nodeBrowserSession, Value: "opaque-runtime-binding-required"},
		{QuestionID: nodeBrowserApproval, Value: "approve update"},
	}, discovery.Docs); err != nil {
		t.Fatal(err)
	}
	issues = CheckReadiness(session, discovery.Docs)
	if hasReadinessCode(issues, "missing_browser_session_posture") || hasReadinessCode(issues, "unconfirmed_browser_mutation") {
		t.Fatalf("browser decisions were not applied: %#v", issues)
	}
}

func TestBrowserRegistryLocalSuccessAndNetworkBlockers(t *testing.T) {
	registryRoot := filepath.Join(t.TempDir(), "registry")
	value := buildBrowserTestBundle(t)
	if _, err := registry.PublishLocal(context.Background(), registry.PublishOptions{Root: registryRoot, Bundle: value, At: browserIntegrationTime}); err != nil {
		t.Fatal(err)
	}
	report, err := DiscoverBrowserRegistrySources(context.Background(), []string{registryRoot}, "status", "never", false, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Candidates) != 1 || len(report.Docs) != 1 || len(report.Plans) != 1 || report.Plans[0].RegistryCoordinate != "example/status@1.0.0" {
		t.Fatalf("local registry discovery = %#v", report)
	}
	if len(report.Plans[0].MaterializedContent) == 0 {
		t.Fatal("registry profile was not retained in memory for approved materialization")
	}
	if report.Plans[0].TargetPath != "browser-profiles/example-status.json" {
		t.Fatalf("registry target changed without a collision: %q", report.Plans[0].TargetPath)
	}

	denied, err := DiscoverBrowserRegistrySources(context.Background(), []string{"https://registry.example.test"}, "status", "ask", false, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(denied.Blockers) != 1 || denied.Blockers[0].Code != "browser_registry.approval_required" {
		t.Fatalf("approval blocker = %#v", denied.Blockers)
	}
	unsafe, err := DiscoverBrowserRegistrySources(context.Background(), []string{"https://127.0.0.1/registry"}, "status", "allow", true, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(unsafe.Blockers) != 1 || unsafe.Blockers[0].Code != "browser_registry.unsafe_host" {
		t.Fatalf("unsafe-host blocker = %#v", unsafe.Blockers)
	}
}

func TestBrowserRegistryCollidingIDsGetDistinctTargets(t *testing.T) {
	firstRoot := filepath.Join(t.TempDir(), "first")
	secondRoot := filepath.Join(t.TempDir(), "second")
	bundle := buildBrowserTestBundle(t)
	for _, root := range []string{firstRoot, secondRoot} {
		if _, err := registry.PublishLocal(context.Background(), registry.PublishOptions{Root: root, Bundle: bundle, At: browserIntegrationTime}); err != nil {
			t.Fatal(err)
		}
	}
	report, err := DiscoverBrowserRegistrySources(context.Background(), []string{firstRoot, secondRoot}, "status", "never", false, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Plans) != 2 || report.Plans[0].TargetPath == report.Plans[1].TargetPath {
		t.Fatalf("colliding registry plans = %#v", report.Plans)
	}
	targets := []string{report.Plans[0].TargetPath, report.Plans[1].TargetPath}
	sort.Strings(targets)
	if !strings.HasPrefix(targets[0], "browser-profiles/example-status-") || targets[1] != "browser-profiles/example-status.json" {
		t.Fatalf("collision targets = %q, %q", report.Plans[0].TargetPath, report.Plans[1].TargetPath)
	}
	if len(report.Docs) != 2 {
		t.Fatalf("colliding registry documents = %#v", report.Docs)
	}
	planByTarget := map[string]SourceMaterialization{}
	for _, plan := range report.Plans {
		planByTarget[plan.TargetPath] = plan
	}
	for _, doc := range report.Docs {
		plan, ok := planByTarget[doc.RelativePath]
		if !ok || doc.ID != plan.ID {
			t.Fatalf("registry document was not rebuilt from its collision-safe plan: doc=%#v plans=%#v", doc, report.Plans)
		}
		for _, operation := range doc.Operations {
			if operation.DocumentName != plan.ID || operation.DocumentRelativePath != plan.TargetPath {
				t.Fatalf("registry operation retained the colliding path: operation=%#v plan=%#v", operation, plan)
			}
		}
	}
}

func TestBrowserRegistryCollisionAllocatorRechecksGeneratedTarget(t *testing.T) {
	plan := SourceMaterialization{ID: "example-status", TargetPath: "browser-profiles/example-status.json", SourceSHA256: strings.Repeat("a", 64)}
	candidate := BrowserRegistryCandidate{Materialize: plan.TargetPath}
	seen := map[string]bool{plan.TargetPath: true}
	if !assignUniqueBrowserRegistryTarget(&plan, &candidate, seen, "registry-one") {
		t.Fatal("first registry collision was not renamed")
	}
	firstGenerated := plan.TargetPath

	plan = SourceMaterialization{ID: "example-status", TargetPath: "browser-profiles/example-status.json", SourceSHA256: strings.Repeat("a", 64)}
	candidate = BrowserRegistryCandidate{Materialize: plan.TargetPath}
	seen[firstGenerated] = true
	if !assignUniqueBrowserRegistryTarget(&plan, &candidate, seen, "registry-one") {
		t.Fatal("second registry collision was not renamed")
	}
	if plan.TargetPath == firstGenerated || seen[plan.TargetPath] || candidate.Materialize != plan.TargetPath {
		t.Fatalf("generated registry target collided again: plan=%#v candidate=%#v seen=%#v", plan, candidate, seen)
	}
}

func TestBrowserRegistryOperatorTextRemovesControlsAndSecrets(t *testing.T) {
	got := browserRegistryOperatorText(" hostile\n\x1b[2J Bearer abcdefghijklmnopqrstuvwxyz012345 ")
	if strings.ContainsAny(got, "\r\n\x1b") || strings.Contains(got, "abcdefghijklmnopqrstuvwxyz012345") {
		t.Fatalf("unsafe registry text = %q", got)
	}
	blocker := browserRegistryErrorBlocker("registry\n\x1b[2J", errors.New("hostile\n\x1b[2J detail"))
	if strings.ContainsAny(blocker.Registry+blocker.Message, "\r\n\x1b") || strings.Contains(blocker.Message, "hostile") {
		t.Fatalf("unsafe registry blocker = %#v", blocker)
	}
}

func TestBrowserRegistryRemoteApprovalIsSeparateFromAPILookup(t *testing.T) {
	session := Session{
		Boundary: WorkflowBoundary{Outcome: "read UI status", Actor: "operator", Trigger: "on demand", SuccessEvidence: []string{"status is returned"}},
		Project:  projectwizard.Answers{ProjectName: "Status", Goal: "read UI status", Fallback: "stop cleanly", Safety: "read only", SideEffectScope: projectwizard.SideEffectReadOnly},
		Intent:   rollout.Intent{Workflow: &rollout.WorkflowMeta{Name: "status", Description: "read UI status"}, OpenAPI: "openapi/missing.yaml", Steps: []*rollout.Step{{Name: "read", Type: "http", OpenAPI: "openapi/missing.yaml"}}},
		Fallback: "stop cleanly", FallbackSet: true, Safety: "read only", SafetySet: true, SideEffectScope: projectwizard.SideEffectReadOnly,
	}
	session.Normalize()
	if session.Interview.Metadata == nil {
		session.Interview.Metadata = map[string]string{}
	}
	session.Interview.Metadata["network_policy"] = "ask"
	session.Interview.Metadata["browser_registry_configured"] = "true"
	issues := CheckReadiness(session, nil)
	frontier, err := PlanFrontier(&session, nil, issues)
	if err != nil {
		t.Fatal(err)
	}
	if !hasQuestionID(frontier, nodeRemoteLookup) || !hasQuestionID(frontier, nodeBrowserRegistry) {
		t.Fatalf("separate remote approvals missing from frontier: %#v", frontier)
	}
	if err := ApplyFrontierRound(&session, []authoring.RoundAnswer{
		{QuestionID: nodeRemoteLookup, Value: "never"},
		{QuestionID: nodeBrowserRegistry, Value: "allow"},
	}, nil); err != nil {
		t.Fatal(err)
	}
	if session.Interview.Metadata["remote_lookup_decision"] != "never" || session.Interview.Metadata["browser_registry_lookup_decision"] != "allow" {
		t.Fatalf("remote decisions = %#v", session.Interview.Metadata)
	}
}

func TestBrowserRegistryHTTPSBoundedSuccessTimeoutAndEmpty(t *testing.T) {
	registryRoot := filepath.Join(t.TempDir(), "registry")
	if _, err := registry.PublishLocal(context.Background(), registry.PublishOptions{Root: registryRoot, Bundle: buildBrowserTestBundle(t), At: browserIntegrationTime}); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.FileServer(http.Dir(registryRoot)))
	defer server.Close()
	report, err := DiscoverBrowserRegistrySourcesWithOptions(context.Background(), BrowserRegistryDiscoveryOptions{
		Locations: []string{server.URL}, Query: "status", Policy: "allow", Approved: true, At: browserIntegrationTime,
		HTTPClient: server.Client(), AllowUnsafeHosts: true,
	})
	if err != nil || len(report.Candidates) != 1 {
		t.Fatalf("HTTPS registry discovery = %#v, %v", report, err)
	}

	timeoutServer := httptest.NewTLSServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) { <-request.Context().Done() }))
	defer timeoutServer.Close()
	timedOut, err := DiscoverBrowserRegistrySourcesWithOptions(context.Background(), BrowserRegistryDiscoveryOptions{
		Locations: []string{timeoutServer.URL}, Query: "status", Policy: "allow", Approved: true, At: browserIntegrationTime,
		HTTPClient: timeoutServer.Client(), AllowUnsafeHosts: true, Timeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(timedOut.Blockers) != 1 || timedOut.Blockers[0].Code != "browser_registry.timeout" {
		t.Fatalf("timeout blocker = %#v", timedOut.Blockers)
	}

	emptyRoot := filepath.Join(t.TempDir(), "empty")
	if err := os.MkdirAll(emptyRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	writeBrowserTestFile(t, filepath.Join(emptyRoot, registry.IndexName), []byte(`{"version":"browsertools.registry-index.v1","entries":[]}`))
	empty, err := DiscoverBrowserRegistrySources(context.Background(), []string{emptyRoot}, "status", "never", false, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Blockers) != 1 || empty.Blockers[0].Code != "browser_registry.empty" {
		t.Fatalf("empty blocker = %#v", empty.Blockers)
	}
}

type browserRegistryFixedResolver []net.IPAddr

func (r browserRegistryFixedResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	return r, nil
}

func TestBrowserRegistryRejectsMixedDNSAnswersBeforeDownloading(t *testing.T) {
	report, err := DiscoverBrowserRegistrySourcesWithOptions(context.Background(), BrowserRegistryDiscoveryOptions{
		Locations: []string{"https://registry.example.test"}, Query: "status", Policy: "allow", Approved: true,
		At: browserIntegrationTime,
		Resolver: browserRegistryFixedResolver{
			{IP: net.ParseIP("93.184.216.34")},
			{IP: net.ParseIP("127.0.0.1")},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Blockers) != 1 || report.Blockers[0].Code != "browser_registry.unsafe_host" {
		t.Fatalf("mixed-DNS report = %#v", report)
	}
}

func buildBrowserTestBundle(t *testing.T) *bundle.Bundle {
	t.Helper()
	value, err := profile.ParseJSON(browserProfileFixture(false, false))
	if err != nil {
		t.Fatal(err)
	}
	record, err := (&bevidence.RawRecord{Record: bevidence.Record{
		Origin: "https://example.test", ObservationKind: bevidence.ObservationA11ySnapshot,
		ObservedAt: "2026-08-15T00:00:00Z", ActionHint: "read_status",
		CandidateLocators: []bevidence.CandidateLocator{{Role: "status", Name: "Ready"}},
		RedactionStatus:   bevidence.RedactionNotRequired, Provenance: bevidence.Provenance{Tool: "synthetic-fixture", Version: "1"},
	}}).Normalize()
	if err != nil {
		t.Fatal(err)
	}
	reviewBundle, err := review.Build(value, []bevidence.Record{record}, nil, browserIntegrationTime)
	if err != nil {
		t.Fatal(err)
	}
	result, err := bundle.Build(bundle.BuildOptions{
		ID: "example/status", Release: "1.0.0", Source: "reviewed_synthetic_fixture", License: "CC0-1.0",
		Authors: []string{"OpenUdon"}, Profile: value, Review: reviewBundle, Evidence: []bevidence.Record{record}, PublishedAt: browserIntegrationTime,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func browserProfileFixture(mutating, login bool) []byte {
	action := `"read_status":{"description":"Read service status.","sequence":[{"navigate":"/status"},{"wait_for":{"role":"status","name":"Ready"}}],"outputs":{"status":{"type":"string","source":"a11y","locator":{"role":"status","name":"Ready"}}},"sideEffects":["read_only"],"confirmationPolicy":{"required":false}}`
	if mutating {
		action = `"update_record":{"description":"Update a record.","parameters":{"type":"object","properties":{"note":{"type":"string"}},"required":["note"]},"sequence":[{"navigate":"/edit"},{"type_text":{"locator":{"role":"textbox","name":"Note"},"value":"{{note}}"}},{"click":{"locator":{"role":"button","name":"Save"}}}],"outputs":{"status":{"type":"string","source":"a11y","locator":{"role":"status","name":"Saved"}}},"sideEffects":["updates_record"],"confirmationPolicy":{"required":true,"prompt":"Approve updating the selected record."}}`
	}
	return []byte(fmt.Sprintf(`{"profile":"uws.browser.1.5","info":{"title":"Example Browser","origin":"https://example.test","loginStateRequired":%t},"observationKind":"accessibility_snapshot","evidence":{"learnedAt":"2026-08-15T00:00:00Z","source":"synthetic_fixture"},"confidence":"high","expiresAfter":"P30D","verification":{"lastVerifiedAt":"2026-08-15T00:00:00Z","successfulRuns":2,"uiStabilityScore":0.95},"actions":{%s}}`, login, action))
}

func writeBrowserTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
