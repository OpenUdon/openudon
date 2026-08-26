package browsertransactioneval

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browserscenario"
)

func TestBuildQualificationReportMapsValidatedClosedEvidence(t *testing.T) {
	bapBCP := testBAPBCPQualificationEvidence()
	brp := testBRPQualificationEvidence()
	report, err := BuildQualificationReport(
		time.Date(2026, 8, 26, 1, 2, 3, 0, time.UTC),
		testReport(t).Repositories,
		bapBCP,
		brp,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(report); err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusPass || report.Summary != (Summary{Total: 18, Passed: 18}) {
		t.Fatalf("qualification summary = %#v, status = %q", report.Summary, report.Status)
	}
	if len(report.Artifacts) != 20 || report.Results[7].ID != GateBRPNetwork || report.Results[7].EvidenceCount != brp.Requests {
		t.Fatalf("qualification evidence mapping is invalid: %#v", report)
	}
	if !report.Posture.SandboxRequired || !report.Posture.SandboxEnabled || !report.Posture.LoopbackOnly ||
		report.Posture.PublicTargetsContacted || report.Posture.RegistrationAuthoringPostRequests != 0 ||
		report.Posture.RegistrationRuntimePostRequests != 1 || !report.Posture.RegistrationSubmitApproved ||
		!report.Posture.AccountCreated || !report.Posture.ExecutorInvokedForRegistration || report.Posture.RegistrationSessionEstablished ||
		report.Posture.ContainsPrivateMaterial || !report.Posture.ValueFree {
		t.Fatalf("qualification posture = %#v", report.Posture)
	}
}

func TestBuildQualificationReportRejectsInvalidProducerEvidence(t *testing.T) {
	bapBCP := testBAPBCPQualificationEvidence()
	brp := testBRPQualificationEvidence()
	bapBCP.TransactionSHA256 = bapBCP.ProducerResultSHA256
	if _, err := BuildQualificationReport(time.Now(), testReport(t).Repositories, bapBCP, brp); err == nil ||
		!strings.Contains(err.Error(), "authenticated qualification evidence is invalid") {
		t.Fatalf("invalid BAP+BCP evidence error = %v", err)
	}

	bapBCP = testBAPBCPQualificationEvidence()
	brp.MutationRequests = 1
	if _, err := BuildQualificationReport(time.Now(), testReport(t).Repositories, bapBCP, brp); err == nil ||
		!strings.Contains(err.Error(), "registration qualification evidence is invalid") {
		t.Fatalf("invalid BRP evidence error = %v", err)
	}
}

func TestBoundedQualificationWriterCapsRetainedOutput(t *testing.T) {
	writer := &boundedQualificationWriter{remaining: 3}
	if count, err := writer.Write([]byte("private-output")); err != nil || count != len("private-output") {
		t.Fatalf("Write = %d, %v", count, err)
	}
	if !writer.exceeded || writer.remaining != 0 || writer.buffer.String() != "pri" {
		t.Fatalf("bounded writer = %#v", writer)
	}
	if count, err := writer.Write([]byte("more")); err != nil || count != 4 || writer.buffer.String() != "pri" {
		t.Fatalf("second Write = %d, %v, %q", count, err, writer.buffer.String())
	}
}

func TestEqualRepositoryRevisionsRequiresExactOrderAndIdentity(t *testing.T) {
	repositories := testReport(t).Repositories
	copy := append([]RepositoryRevision(nil), repositories...)
	if !equalRepositoryRevisions(repositories, copy) {
		t.Fatal("equal repository revisions were rejected")
	}
	copy[1].Published = false
	if equalRepositoryRevisions(repositories, copy) {
		t.Fatal("changed repository identity was accepted")
	}
	if equalRepositoryRevisions(repositories, repositories[:2]) {
		t.Fatal("truncated repository identity was accepted")
	}
}

func TestQualificationRootsResolveDefaultsBesideExplicitOpenUdonRoot(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "openudon")
	for _, name := range []string{"openudon", "browsertools", "uws", "udon", "browserdriver"} {
		if err := os.Mkdir(filepath.Join(parent, name), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	roots, err := qualificationRoots(QualificationOptions{RepoRoot: root})
	if err != nil {
		t.Fatal(err)
	}
	if roots.openudon != root || roots.browsertools != filepath.Join(parent, "browsertools") ||
		roots.uws != filepath.Join(parent, "uws") || roots.udon != filepath.Join(parent, "udon") ||
		roots.browserdriver != filepath.Join(parent, "browserdriver") {
		t.Fatalf("qualification roots = %#v", roots)
	}
}

func TestQualificationStageErrorPreservesOnlySandboxCause(t *testing.T) {
	wrapped := fmt.Errorf("private backend detail: %w", browserscenario.ErrSandboxPrerequisiteUnavailable)
	got := qualificationStageError(wrapped, "closed fallback")
	if !errors.Is(got, browserscenario.ErrSandboxPrerequisiteUnavailable) ||
		got.Error() != browserscenario.ErrSandboxPrerequisiteUnavailable.Error() {
		t.Fatalf("sandbox error = %v", got)
	}
	if got := qualificationStageError(errors.New("private backend detail"), "closed fallback"); got.Error() != "closed fallback" {
		t.Fatalf("generic error = %v", got)
	}
}

func TestAdversarialMakeTargetDoesNotShellExpandBrowsertoolsPath(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(data)
	if !strings.Contains(source, `cd "$${OPENUDON_BROWSERTOOLS_REPO:-../browsertools}"`) ||
		strings.Contains(source, "$(BROWSERTOOLS_REPO)") {
		t.Fatal("browser-transaction-adversarial must consume only the quoted child environment path")
	}
}

func TestQualificationAdversarialEnvironmentClosesControlOverrides(t *testing.T) {
	t.Setenv("OPENUDON_BROWSERTOOLS_REPO", "old")
	t.Setenv("MAKEFLAGS", "-n")
	t.Setenv("MAKEFILES", "private.mk")
	t.Setenv("GO", "true")
	t.Setenv("GOFLAGS", "-run=none")
	t.Setenv("GOENV", "private-go-env")
	environment := qualificationAdversarialEnvironment("literal-$(not-executed)", "/trusted/bin/go")
	values := map[string]string{}
	counts := map[string]int{}
	for _, item := range environment {
		name, value, ok := strings.Cut(item, "=")
		if ok {
			values[name] = value
			counts[name]++
		}
	}
	if values["OPENUDON_BROWSERTOOLS_REPO"] != "literal-$(not-executed)" || values["GOENV"] != "off" ||
		!strings.HasPrefix(values["PATH"], "/trusted/bin"+string(os.PathListSeparator)) || counts["PATH"] != 1 {
		t.Fatalf("closed environment = %#v", values)
	}
	for _, name := range []string{"MAKEFLAGS", "MAKEFILES", "GO", "GOFLAGS"} {
		if _, ok := values[name]; ok {
			t.Fatalf("closed environment retained %s", name)
		}
	}
}

func TestQualificationGitEnvironmentRejectsRepositoryOverrides(t *testing.T) {
	t.Setenv("GIT_DIR", "private.git")
	t.Setenv("GIT_WORK_TREE", "private-tree")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "remote.origin.url")
	t.Setenv("GIT_CONFIG_VALUE_0", "private-origin")
	for _, item := range qualificationGitEnvironment() {
		name, _, ok := strings.Cut(item, "=")
		if ok && (name == "GIT_DIR" || name == "GIT_WORK_TREE" || strings.HasPrefix(name, "GIT_CONFIG_")) {
			t.Fatalf("Git environment retained %s", name)
		}
	}
}

func testBAPBCPQualificationEvidence() browserscenario.BAPBCPQualificationEvidence {
	values := testQualificationDigests("bap-bcp")
	return browserscenario.BAPBCPQualificationEvidence{
		ProducerResultSHA256: values[0], TransactionSHA256: values[1], PreparationSHA256: values[2],
		QualificationSHA256: values[3], GenerationSHA256: values[4], SelectionSHA256: values[5],
		PackageSHA256: values[6], HandoffSHA256: values[7], WorkflowSHA256: values[8], EvidenceCount: 9,
	}
}

func testBRPQualificationEvidence() browserscenario.BRPQualificationEvidence {
	values := testQualificationDigestsN("brp", 11)
	return browserscenario.BRPQualificationEvidence{
		ProducerResultSHA256: values[0], TransactionSHA256: values[1], PreparationSHA256: values[2],
		QualificationSHA256: values[3], GenerationSHA256: values[4], SelectionSHA256: values[5],
		PackageSHA256: values[6], HandoffSHA256: values[7], WorkflowSHA256: values[8],
		AttestationSHA256: values[9], ExecutionReportSHA256: values[10], EvidenceCount: 11,
		Methods: []string{"GET", "HEAD"}, Requests: 2, GETRequests: 1, HEADRequests: 1,
		RuntimePOSTRequests: 1, SubmitApproved: true, SubmitExecuted: true, AccountCreated: true,
		RuntimeSupported: true, ExecutorInvoked: true,
	}
}

func testQualificationDigests(prefix string) []string {
	return testQualificationDigestsN(prefix, 9)
}

func testQualificationDigestsN(prefix string, count int) []string {
	values := make([]string, count)
	for index := range values {
		sum := sha256.Sum256([]byte(prefix + string(rune(index))))
		values[index] = "sha256:" + hex.EncodeToString(sum[:])
	}
	return values
}
