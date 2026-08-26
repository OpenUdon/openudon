package browsertransactioneval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/internal/browserscenario"
)

func TestReportCanonicalValueFreeRoundTrip(t *testing.T) {
	report := testReport(t)
	first, err := CanonicalBytes(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CanonicalBytes(report)
	if err != nil || !bytes.Equal(first, second) || bytes.Contains(first, []byte("  \"version\"")) {
		t.Fatalf("canonical bytes differ: %v\n%s", err, first)
	}
	path := filepath.Join(t.TempDir(), "qualification.json")
	if err := Write(path, report); err != nil {
		t.Fatal(err)
	}
	verified, err := VerifyFile(path, true)
	if err != nil || verified.Status != StatusPass || verified.Summary != (Summary{Total: len(gateOrder), Passed: len(gateOrder)}) {
		t.Fatalf("VerifyFile = %#v, %v", verified, err)
	}
	for _, forbidden := range []string{"private_root", "result_path", "worker_output", "page_content", "request_content", "cookie", "storage", "credential_value", "account_identifier", "/home/"} {
		if bytes.Contains(first, []byte(forbidden)) {
			t.Fatalf("canonical report retained forbidden class %q: %s", forbidden, first)
		}
	}
}

func TestReportRejectsContractDriftFailureAndDependencyMismatch(t *testing.T) {
	base := testReport(t)
	tests := []struct {
		name   string
		mutate func(*Report)
	}{
		{name: "gate order", mutate: func(report *Report) { report.Results[0], report.Results[1] = report.Results[1], report.Results[0] }},
		{name: "untyped failure", mutate: func(report *Report) {
			report.Results[0].Status = StatusFail
			report.Results[0].FailureCode = "raw browser failure"
		}},
		{name: "false pass code", mutate: func(report *Report) { report.Results[0].FailureCode = "contract_invalid" }},
		{name: "summary", mutate: func(report *Report) { report.Summary.Passed-- }},
		{name: "browsertools revision", mutate: func(report *Report) { report.Repositories[1].Commit = strings.Repeat("a", 40) }},
		{name: "openudon published", mutate: func(report *Report) { report.Repositories[0].Published = true }},
		{name: "artifact order", mutate: func(report *Report) {
			report.Artifacts[0], report.Artifacts[1] = report.Artifacts[1], report.Artifacts[0]
		}},
		{name: "post request", mutate: func(report *Report) { report.Posture.RegistrationAuthoringPostRequests = 1 }},
		{name: "sandbox disabled", mutate: func(report *Report) { report.Posture.SandboxEnabled = false }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copy := cloneReport(t, base)
			test.mutate(copy)
			if err := Validate(copy); err == nil {
				t.Fatal("Validate accepted contract drift")
			}
		})
	}

	failed := cloneReport(t, base)
	failed.Results[3].Status = StatusFail
	failed.Results[3].FailureCode = "package_failed"
	failed.Summary = summarize(failed.Results)
	failed.Status = StatusFail
	if err := Validate(failed); err != nil {
		t.Fatalf("typed failure rejected: %v", err)
	}
}

func TestVerifyFileRejectsTamperUnknownMissingDuplicateNoncanonicalAndSymlink(t *testing.T) {
	report := testReport(t)
	canonical, err := CanonicalBytes(report)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		mutate func([]byte) []byte
	}{
		{name: "unknown", mutate: func(data []byte) []byte {
			return bytes.Replace(data, []byte(`{"version"`), []byte(`{"private_path":"/private/result","version"`), 1)
		}},
		{name: "missing false", mutate: func(data []byte) []byte {
			return bytes.Replace(data, []byte(`,"public_targets_contacted":false`), nil, 1)
		}},
		{name: "duplicate", mutate: func(data []byte) []byte {
			return bytes.Replace(data, []byte(`{"version"`), []byte(`{"version":"`+ReportVersion+`","version"`), 1)
		}},
		{name: "noncanonical", mutate: func(data []byte) []byte { return append([]byte(" "), data...) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "qualification.json")
			writeRawReport(t, path, test.mutate(append([]byte(nil), canonical...)))
			if _, err := VerifyFile(path, false); err == nil {
				t.Fatal("VerifyFile accepted malformed report")
			}
		})
	}

	path := filepath.Join(t.TempDir(), "qualification.json")
	if err := Write(path, report); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(canonical, ' '), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFile(path, false); err == nil || !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("tamper error = %v", err)
	}

	realPath := filepath.Join(t.TempDir(), "real.json")
	if err := Write(realPath, report); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(t.TempDir(), "link.json")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyFile(linkPath, false); err == nil {
		t.Fatal("VerifyFile accepted a symlink")
	}
}

func testReport(t *testing.T) *Report {
	t.Helper()
	lock, err := browserscenario.LoadCompatibilityLock()
	if err != nil {
		t.Fatal(err)
	}
	locked := map[string]browserscenario.LockedRevision{}
	for _, component := range lock.Components {
		locked[component.Name] = component
	}
	repositories := []RepositoryRevision{
		{Name: "openudon", Commit: strings.Repeat("1", 40)},
		{Name: "browsertools", Commit: locked["browsertools"].Commit, ModuleVersion: locked["browsertools"].Version, Published: true},
		{Name: "uws", Commit: locked["uws"].Commit, ModuleVersion: locked["uws"].Version, Published: true},
	}
	artifacts := make([]ArtifactDigest, 0, 2*len(artifactKindOrder))
	for _, caseID := range []string{CaseBAPBCP, CaseBRP} {
		for _, kind := range artifactKindOrder {
			sum := sha256.Sum256([]byte(caseID + "\x00" + kind))
			artifacts = append(artifacts, ArtifactDigest{Case: caseID, Kind: kind, SHA256: "sha256:" + hex.EncodeToString(sum[:])})
		}
	}
	results := make([]GateResult, 0, len(gateOrder))
	for index, id := range gateOrder {
		results = append(results, GateResult{ID: id, Status: StatusPass, EvidenceCount: index + 1})
	}
	report, err := NewReport(BuildRequest{
		GeneratedAt:  time.Date(2026, 8, 26, 1, 2, 3, 0, time.UTC),
		Repositories: repositories,
		Posture: Posture{
			SandboxRequired: true, SandboxEnabled: true, LoopbackOnly: true,
			RegistrationAuthoringMethods: []string{"GET", "HEAD"}, ValueFree: true,
		},
		Artifacts: artifacts, Results: results,
	})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func cloneReport(t *testing.T, report *Report) *Report {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var clone Report
	if err := json.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return &clone
}

func writeRawReport(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	line := "sha256:" + hex.EncodeToString(sum[:]) + "  " + filepath.Base(path) + "\n"
	if err := os.WriteFile(path+".sha256", []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
}
