package runevidence_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"testing"

	"github.com/OpenUdon/openudon/digest"
	"github.com/OpenUdon/openudon/runevidence"
	"github.com/OpenUdon/openudon/udonreport"
)

func fixture(t *testing.T) runevidence.RunEvidence {
	t.Helper()
	data, err := os.ReadFile("../docs/fixtures/broker-execution-v1/evidence-unknown.json")
	if err != nil {
		t.Fatal(err)
	}
	var e runevidence.RunEvidence
	if json.Unmarshal(data, &e) != nil {
		t.Fatal("fixture JSON")
	}
	return e
}

func TestPublishedBrokerEvidenceWire(t *testing.T) {
	e := fixture(t)
	if err := runevidence.Validate(e); err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(e)
	if digest.SHA256(data) != "56dac136a5b15dd6ef2f062d0a7c738130dbca72f6328cf3563055cc62dc696b" {
		t.Fatal("published evidence bytes changed")
	}
	for _, name := range []string{"evidence-downgrade.json", "evidence-wrong-operation.json"} {
		data, err := os.ReadFile("../docs/fixtures/broker-execution-v1/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var changed runevidence.RunEvidence
		if json.Unmarshal(data, &changed) != nil {
			t.Fatal("fixture JSON")
		}
		if runevidence.Validate(changed) == nil {
			t.Fatal("accepted broker downgrade or operation drift")
		}
	}
	if _, err := runevidence.Verify(context.Background(), runevidence.Request{Evidence: data}); !errors.Is(err, runevidence.ErrInvalidEvidence) {
		t.Fatal("missing referenced async bytes accepted")
	}
}

func TestExactReportAndIndependentInventory(t *testing.T) {
	report, err := os.ReadFile("../docs/fixtures/per-step-run-evidence-v3/success.report.json")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile("../docs/fixtures/per-step-run-evidence-v3/expected.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory udonreport.InventoryV5
	if json.Unmarshal(expected, &inventory) != nil {
		t.Fatal("inventory JSON")
	}
	for i := range inventory.Steps {
		inventory.Steps[i].Outcome = "not_started"
	}
	e := fixture(t)
	e.Broker = nil
	e.Version = runevidence.StepVersion
	e.RunID = inventory.RunID
	e.AsyncEvidenceFiles = nil
	observed := udonreport.ObserveV5(inventory, report)
	e.StepExecution = &observed
	for i := range e.Gates {
		if e.Gates[i].Name == "executor_invocation" {
			e.Gates[i].Status = "pass"
		}
	}
	e.Executor.ReportPath = "executor-report.json"
	e.Executor.ReportSHA256 = digest.SHA256(report)
	e.Executor.ReportSize = int64(len(report))
	data, _ := json.Marshal(e)
	identity := runevidence.Identity{RunID: e.RunID, PackageSHA256: e.PackageSHA256, HandoffSHA256: e.HandoffSHA256, ApprovalSHA256: e.ApprovalSHA256, RunConfigSHA256: e.RunConfigSHA256}
	r := runevidence.Request{Evidence: data, Artifacts: map[string][]byte{"executor-report.json": report}, Expected: &identity, ExpectedInventory: &inventory}
	result, err := runevidence.Verify(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if result.Evidence.StepExecution.ReportStatus != "success" {
		t.Fatal("lost validated result")
	}
	changed := inventory
	changed.WorkflowDigest = "sha256:" + digest.SHA256([]byte("changed approved workflow"))
	r.ExpectedInventory = &changed
	if _, err := runevidence.Verify(context.Background(), r); err == nil {
		t.Fatal("accepted unrelated approved inventory")
	}
	r.ExpectedInventory = &inventory
	identity.RunID = "other-attempt"
	if _, err := runevidence.Verify(context.Background(), r); err == nil {
		t.Fatal("accepted stale attempt")
	}
	identity.RunID = e.RunID
	r.Artifacts["executor-report.json"] = []byte("untrusted private report bytes")
	if _, err := runevidence.Verify(context.Background(), r); !errors.Is(err, runevidence.ErrInvalidEvidence) {
		t.Fatal("accepted report byte drift")
	}
}

func TestUnknownEvidenceDoesNotInventResults(t *testing.T) {
	e := fixture(t)
	e.AsyncEvidenceFiles = nil
	data, _ := json.Marshal(e)
	result, err := runevidence.Verify(context.Background(), runevidence.Request{Evidence: data})
	if err != nil {
		t.Fatal(err)
	}
	if result.Evidence.StepExecution.State != "missing" || result.Evidence.StepExecution.Steps[0].Outcome != "unknown" {
		t.Fatal("uncertainty changed")
	}
	e.Browser = &runevidence.BrowserConfig{Protocol: "v11"}
	data, _ = json.Marshal(e)
	if _, err := runevidence.Verify(context.Background(), runevidence.Request{Evidence: data}); !errors.Is(err, runevidence.ErrUnsupportedBrowser) {
		t.Fatal("browser bypassed retained profile")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := runevidence.Verify(ctx, runevidence.Request{}); !errors.Is(err, context.Canceled) {
		t.Fatal("cancellation ignored")
	}
}

func TestSignatureIntegrityAndTrustedKeyAreSeparate(t *testing.T) {
	e := fixture(t)
	e.AsyncEvidenceFiles = nil
	data, _ := json.Marshal(e)
	private := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	public := private.Public().(ed25519.PublicKey)
	der, err := x509.MarshalPKIXPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	hash := sha256.Sum256(data)
	envelope := runevidence.Signature{Version: runevidence.SignatureVersion, Algorithm: "ed25519", KeyFingerprint: "sha256:" + digest.SHA256(der), EvidenceSHA256: hex.EncodeToString(hash[:]), PublicKeyPEM: string(keyPEM), SignatureBase64: base64.StdEncoding.EncodeToString(ed25519.Sign(private, hash[:]))}
	signature, _ := json.Marshal(envelope)
	r := runevidence.Request{Evidence: data, Signature: signature, RequireSignature: true}
	result, err := runevidence.Verify(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if !result.SignatureVerified || result.SignerTrusted {
		t.Fatal("embedded key implied custody")
	}
	r.TrustedPublicKeyPEM = keyPEM
	result, err = runevidence.Verify(context.Background(), r)
	if err != nil || !result.SignerTrusted {
		t.Fatal("trusted key verification failed")
	}
	r.TrustedPublicKeyPEM = []byte("untrusted")
	if _, err := runevidence.Verify(context.Background(), r); err == nil {
		t.Fatal("wrong trusted key accepted")
	}
	r.TrustedPublicKeyPEM = keyPEM
	r.Evidence = append(r.Evidence, ' ')
	if _, err := runevidence.Verify(context.Background(), r); err == nil {
		t.Fatal("signature did not bind exact bytes")
	}
}
