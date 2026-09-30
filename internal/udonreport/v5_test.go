package udonreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestV5ConformanceFixtures(t *testing.T) {
	root := filepath.Join("..", "..", "docs", "fixtures", "per-step-run-evidence-v3")
	data, err := os.ReadFile(filepath.Join(root, "expected.json"))
	if err != nil {
		t.Fatal(err)
	}
	var i InventoryV5
	if err := json.Unmarshal(data, &i); err != nil {
		t.Fatal(err)
	}
	for n := range i.Steps {
		i.Steps[n].Outcome = "not_started"
	}
	if err := i.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"dry-run", "success", "failed-read", "failed-write", "interrupted", "missing", "stale", "mismatched", "incomplete-inventory", "malformed"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join(root, name+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var want ObservationV5
			if err := json.Unmarshal(data, &want); err != nil {
				t.Fatal(err)
			}
			if err := want.Validate(); err != nil {
				t.Fatal(err)
			}
			var got ObservationV5
			if name == "dry-run" {
				got = UnknownV5(i, "dry_run")
			} else if name == "missing" {
				got = UnknownV5(i, "missing")
			} else {
				report, err := os.ReadFile(filepath.Join(root, name+".report.json"))
				if err != nil {
					t.Fatal(err)
				}
				got = ObserveV5(i, report)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v; want %+v", got, want)
			}
		})
	}
}

func TestV5RejectsUntrustedRecordAndTimeMutations(t *testing.T) {
	data, err := os.ReadFile("../../docs/fixtures/per-step-run-evidence-v3/success.report.json")
	if err != nil {
		t.Fatal(err)
	}
	base, err := DecodeV5(data)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*ReportV5){
		"comma time":               func(r *ReportV5) { r.StartedAt = "2026-09-30T00:00:00,1Z" },
		"zone out of range":        func(r *ReportV5) { r.StartedAt = "2026-09-30T00:00:00+24:00" },
		"lost timestamp precision": func(r *ReportV5) { r.StartedAt = "2026-09-30T00:00:00.0000000001Z" },
		"duplicate":                func(r *ReportV5) { r.Steps[1].StepID = r.Steps[0].StepID },
		"unknown outcome":          func(r *ReportV5) { r.Steps[0].Outcome = "secret payload" },
		"sensitive code":           func(r *ReportV5) { r.Steps[0].ErrorCode = "secret payload" },
		"start before run":         func(r *ReportV5) { r.Steps[0].StartedAt = "2000-01-01T00:00:00Z" },
		"finish before start":      func(r *ReportV5) { r.Steps[1].FinishedAt = r.StartedAt },
		"too many":                 func(r *ReportV5) { r.Steps = make([]StepV5, 257) },
		"invocation mismatch":      func(r *ReportV5) { r.Steps[1].InvocationID = "other" },
		"success unknown":          func(r *ReportV5) { r.Steps[1].Outcome = "unknown"; r.Steps[1].FinishedAt = "" },
	}
	for name, change := range mutations {
		t.Run(name, func(t *testing.T) {
			r := *base
			r.Steps = append([]StepV5(nil), base.Steps...)
			change(&r)
			data, _ := json.Marshal(r)
			if _, err := DecodeV5(data); err == nil {
				t.Fatal("invalid report accepted")
			}
		})
	}
	for _, bad := range [][]byte{[]byte(strings.Repeat(" ", 256<<10+1)),
		[]byte(strings.Replace(string(data), `"steps":`, `"error_code":"","steps":`, 1)),
		[]byte(strings.Replace(string(data), `"outcome":`, `"error_code":"","outcome":`, 1)),
		[]byte(strings.Replace(string(data), `"version":`, `"VERSION":`, 1)),
		[]byte(strings.Replace(string(data), `"steps":`, `"error_code":null,"steps":`, 1)),
		[]byte(strings.Replace(string(data), `"steps":`, `"inventory_complete":false,"INVENTORY_COMPLETE":true,"steps":`, 1)),
		[]byte(strings.Replace(string(data), `"outcome":`, `"OUTCOME":`, 1)), append(data, []byte(" {}")...), []byte(`{"version":"udon.execution-report.v5","version":"bad"}`), []byte(strings.Replace(string(data), `"version":`, `"payload":"secret", "version":`, 1))} {
		if _, err := DecodeV5(bad); err == nil {
			t.Fatal("untrusted JSON accepted")
		}
	}
}
