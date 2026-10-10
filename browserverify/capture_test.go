package browserverify

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/openudon/browsercontract"
	"github.com/OpenUdon/openudon/browsertransaction"
)

func captureFixture(t *testing.T, version string) ([]byte, CaptureVerifyOptions) {
	t.Helper()
	raw, err := os.ReadFile("../docs/examples/browser-profile-transaction-registration.json")
	if err != nil {
		t.Fatal(err)
	}
	transaction, err := browsertransaction.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	transaction.State = browsertransaction.StateReviewed
	transaction.Version = version
	schema, result := "uws.browser-registration.1.0", browsertransaction.ResultRegistrationAuthoringV1
	switch version {
	case browsertransaction.VersionV2:
		result = browsertransaction.ResultRegistrationAuthoringV2
	case browsertransaction.VersionV3:
		schema, result = "uws.browser-registration.1.1", browsertransaction.ResultRegistrationAuthoringV3
	case browsertransaction.VersionV4:
		schema, result = "uws.browser-registration.1.2", browsertransaction.ResultRegistrationAuthoringV4
	}
	transaction.Provenance.ResultVersion = result
	transaction.Candidates[0].Schema = schema
	source, review := []byte(`{"fixture":"inert byte identity only"}`), []byte(`{"fixture":"value-free review"}`)
	transaction.Candidates[0].SourceSHA256 = "sha256:" + browsercontract.SHA256(source)
	transaction.Candidates[0].ReviewSHA256 = "sha256:" + browsercontract.SHA256(review)
	digest, err := browsertransaction.Digest(transaction)
	if err != nil {
		t.Fatal(err)
	}
	start := browsercontract.SHA256([]byte("independent exact reviewed start"))
	receipt := CaptureReceipt{Version: CaptureImportVersion, StartSHA256: start, Transaction: transaction, Effect: "write", Files: []CaptureFile{{Path: "browser-registration/fixture.json", SHA256: browsercontract.SHA256(source)}, {Path: "expected/registration-review.json", SHA256: browsercontract.SHA256(review)}}}
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	return data, CaptureVerifyOptions{ExpectedReceiptSHA256: browsercontract.SHA256(data), ExpectedStartSHA256: start, ExpectedTransactionSHA256: digest, Files: map[string][]byte{"browser-registration/fixture.json": source, "expected/registration-review.json": review}, At: time.Date(2026, 8, 25, 13, 0, 0, 0, time.UTC)}
}
func TestPublicCaptureReceiptsBindAllFourTransactionVersions(t *testing.T) {
	for _, version := range []string{browsertransaction.VersionV1, browsertransaction.VersionV2, browsertransaction.VersionV3, browsertransaction.VersionV4} {
		t.Run(version, func(t *testing.T) {
			raw, o := captureFixture(t, version)
			r, err := VerifyCaptureReceipt(raw, o)
			if err != nil {
				t.Fatal(err)
			}
			if r.Transaction.Version != version {
				t.Fatal("version projection")
			}
			r.Transaction.CredentialBindings[0].Binding = "changed"
			again, err := VerifyCaptureReceipt(raw, o)
			if err != nil || again.Transaction.CredentialBindings[0].Binding == "changed" {
				t.Fatal("mutable receipt")
			}
		})
	}
}
func TestPublicCaptureReceiptsRefuseForeignTamperedPrivateOrExpiredEvidence(t *testing.T) {
	for name, mutate := range map[string]func(*[]byte, *CaptureVerifyOptions){
		"foreign original start": func(raw *[]byte, o *CaptureVerifyOptions) { o.ExpectedStartSHA256 = strings.Repeat("a", 64) },
		"foreign receipt":        func(raw *[]byte, o *CaptureVerifyOptions) { o.ExpectedReceiptSHA256 = strings.Repeat("a", 64) },
		"foreign transaction": func(raw *[]byte, o *CaptureVerifyOptions) {
			o.ExpectedTransactionSHA256 = "sha256:" + strings.Repeat("a", 64)
		},
		"profile changed": func(raw *[]byte, o *CaptureVerifyOptions) {
			o.Files["browser-registration/fixture.json"] = []byte(`{"changed":true}`)
		},
		"missing review": func(raw *[]byte, o *CaptureVerifyOptions) { delete(o.Files, "expected/registration-review.json") },
		"expired":        func(raw *[]byte, o *CaptureVerifyOptions) { o.At = o.At.Add(48 * time.Hour) },
		"private field": func(raw *[]byte, o *CaptureVerifyOptions) {
			*raw = bytes.Replace(*raw, []byte(`"effect":"write"`), []byte(`"effect":"write","cookie":"private"`), 1)
			o.ExpectedReceiptSHA256 = browsercontract.SHA256(*raw)
		},
		"duplicate start": func(raw *[]byte, o *CaptureVerifyOptions) {
			*raw = bytes.Replace(*raw, []byte(`"effect":"write"`), []byte(`"effect":"write","start_sha256":"`+o.ExpectedStartSHA256+`"`), 1)
			o.ExpectedReceiptSHA256 = browsercontract.SHA256(*raw)
		},
	} {
		t.Run(name, func(t *testing.T) {
			raw, o := captureFixture(t, browsertransaction.VersionV4)
			mutate(&raw, &o)
			if _, err := VerifyCaptureReceipt(raw, o); err == nil {
				t.Fatal("unproved capture accepted")
			}
		})
	}
}
func TestPublicVerificationBytesUseExactProfileAndNoFilesystem(t *testing.T) {
	prof := verificationProfile(t)
	raw, err := json.Marshal(validLiveReport(t, prof))
	if err != nil {
		t.Fatal(err)
	}
	summary, err := InspectBytes(raw, prof, verificationNow())
	if err != nil || !summary.OK {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte(`"profileDigest":"sha256:`), []byte(`"profileDigest":"sha256:a`), 1)
	if _, err := InspectBytes(changed, prof, verificationNow()); err == nil {
		t.Fatal("changed identity accepted")
	}
}
