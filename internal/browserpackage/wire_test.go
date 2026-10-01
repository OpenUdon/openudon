package browserpackage

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func requestFixture(t *testing.T, mode string) []byte {
	t.Helper()
	data, err := os.ReadFile("../../docs/fixtures/browser-author-v1/" + mode + "-request.json")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestPublishedRequestsRetainBothModesAndNativeStart(t *testing.T) {
	for _, mode := range []string{"authenticated", "registration"} {
		r, err := DecodeRequest(requestFixture(t, mode))
		if err != nil {
			t.Fatal(mode, err)
		}
		if r.ExpectedTOTP == nil || *r.ExpectedTOTP != (mode == "authenticated") {
			t.Fatal("TOTP/mode lost")
		}
	}
}

func TestPublishedPlanAndPartialWriteResultFixtures(t *testing.T) {
	for _, mode := range []string{"authenticated", "registration"} {
		data, err := os.ReadFile("../../docs/fixtures/browser-author-v1/" + mode + "-plan.json")
		if err != nil {
			t.Fatal(err)
		}
		var p Plan
		if err := json.Unmarshal(data, &p); err != nil {
			t.Fatal(err)
		}
		sealed, err := SealPlan(p)
		if err != nil || p.PlanSHA256 != sealed.PlanSHA256 || p.Ready {
			t.Fatal("fixture grants authority or digest drift", err)
		}
		data, err = os.ReadFile("../../docs/fixtures/browser-author-v1/" + mode + "-result.json")
		if err != nil {
			t.Fatal(err)
		}
		var result Result
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if result.PlanSHA256 != p.PlanSHA256 || result.Outcome != "build_failed" || len(result.Written) == 0 {
			t.Fatal("partial-write outcome lost")
		}
	}
}

func TestRequestRejectsAmbiguityValuesAndPolicySubstitution(t *testing.T) {
	base := requestFixture(t, "authenticated")
	for name, change := range map[string]func(map[string]any){
		"wrong-version":        func(r map[string]any) { r["version"] = "new" },
		"unknown-secret":       func(r map[string]any) { r["password"] = "sentinel-secret" },
		"missing-totp":         func(r map[string]any) { delete(r, "expected_totp") },
		"null-totp":            func(r map[string]any) { r["expected_totp"] = nil },
		"receipt-escape":       func(r map[string]any) { r["receipt_path"] = "expected/browser-capture/../../escape.json" },
		"absolute-receipt":     func(r map[string]any) { r["receipt_path"] = "/tmp/receipt.json" },
		"bad-digest":           func(r map[string]any) { r["transaction_sha256"] = strings.Repeat("a", 64) },
		"registration-on-auth": func(r map[string]any) { r["registration_authority"] = "extra" },
		"mixed-native-mode": func(r map[string]any) {
			mutateStart(r, func(s map[string]any) { s["registration"] = map[string]any{} })
		},
		"native-credential": func(r map[string]any) {
			mutateStart(r, func(s map[string]any) { s["authentication"].(map[string]any)["password"] = "sentinel-secret" })
		},
		"input-value":      func(r map[string]any) { r["input_bindings"] = map[string]any{"name": "sentinel-secret"} },
		"undeclared-input": func(r map[string]any) { r["input_bindings"] = map[string]any{"name": "inputs.name"} },
		"input-default": func(r map[string]any) {
			r["inputs"] = []any{map[string]any{"name": "name", "type": "string", "default": "sentinel-secret"}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			var r map[string]any
			if err := json.Unmarshal(base, &r); err != nil {
				t.Fatal(err)
			}
			change(r)
			data, _ := json.Marshal(r)
			if _, err := DecodeRequest(data); err == nil || strings.Contains(err.Error(), "sentinel-secret") {
				t.Fatal("invalid request accepted or echoed", err)
			}
		})
	}
	for _, data := range [][]byte{
		bytes.Replace(base, []byte(`"kind": "request"`), []byte(`"kind":"request","kind":"request"`), 1),
		append(append([]byte{}, base...), []byte(` {}`)...),
		bytes.Repeat([]byte(" "), MaxRequestBytes+1),
		append([]byte{255}, base...),
	} {
		if _, err := DecodeRequest(data); err == nil {
			t.Fatal("ambiguous/oversized request accepted")
		}
	}
}

func mutateStart(r map[string]any, mutate func(map[string]any)) {
	data, _ := base64.StdEncoding.DecodeString(r["start"].(string))
	var start map[string]any
	_ = json.Unmarshal(data, &start)
	mutate(start)
	data, _ = json.Marshal(start)
	r["start"] = base64.StdEncoding.EncodeToString(data)
}

func TestExactStartBytesSurviveOuterSerialization(t *testing.T) {
	r, err := DecodeRequest(requestFixture(t, "authenticated"))
	if err != nil {
		t.Fatal(err)
	}
	r.Start = append(append([]byte(" \n"), r.Start...), '\n')
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := DecodeRequest(data)
	if err != nil || !bytes.Equal(parsed.Start, r.Start) {
		t.Fatal("native receipt byte binding changed", err)
	}
}

func TestRegistrationRefusesTOTPAndRetainsExplicitAuthority(t *testing.T) {
	var r map[string]any
	if err := json.Unmarshal(requestFixture(t, "registration"), &r); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"expected_totp", "registration_authority", "cleanup_disposition", "action"} {
		copy := map[string]any{}
		for k, v := range r {
			copy[k] = v
		}
		switch field {
		case "expected_totp":
			copy[field] = true
		case "registration_authority":
			delete(copy, field)
		case "cleanup_disposition":
			copy[field] = "none"
		case "action":
			copy[field] = "submit"
		}
		data, _ := json.Marshal(copy)
		if _, err := DecodeRequest(data); err == nil {
			t.Fatal("registration policy widened", field)
		}
	}
}

func TestPlanDigestBindsOwnerRequestInventoryReceiptAndPreview(t *testing.T) {
	p := Plan{Version: Version, Kind: "plan", RequestID: "sample", RequestSHA256: Digest([]byte("request")), InputSHA256: Digest([]byte("input")), ReceiptSHA256: strings.Repeat("a", 64), TransactionSHA256: Digest([]byte("transaction")), Candidates: nil, Operations: nil, Blockers: []string{}}
	p, err := SealPlan(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*Plan){
		func(p *Plan) { p.RequestID = "other" },
		func(p *Plan) { p.RequestSHA256 = Digest([]byte("other request")) },
		func(p *Plan) { p.InputSHA256 = Digest([]byte("changed package")) },
		func(p *Plan) { p.ReceiptSHA256 = strings.Repeat("b", 64) },
		func(p *Plan) { p.TransactionSHA256 = Digest([]byte("other transaction")) },
		func(p *Plan) { p.Ready = true },
	} {
		changed := p
		mutate(&changed)
		sealed, err := SealPlan(changed)
		if err != nil || sealed.PlanSHA256 == p.PlanSHA256 {
			t.Fatal("changed plan kept authority", err)
		}
	}
	sealed, err := SealPlan(p)
	if err != nil || sealed.PlanSHA256 != p.PlanSHA256 {
		t.Fatal("self digest unstable", err)
	}
}
