package registrationdraft

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestNeutralRegistrationDraftCommandRetainsCanonicalProfileAndRefusesUnsafeSource(t *testing.T) {
	r := CommandRequest{Version: CommandVersion, Start: Start{ProfileVersion: "1.0", Origins: []string{"https://app.example.test"}}, Draft: validRegistrationDraftRequest(), Observation: registrationDraftObservation()}
	data, _ := json.Marshal(r)
	var out, errOut bytes.Buffer
	if code := RunCommand([]string{"--request", "-", "--at", "2026-08-26T12:00:00Z"}, bytes.NewReader(data), &out, &errOut); code != 0 {
		t.Fatalf("draft code=%d err=%s", code, errOut.String())
	}
	var result CommandResult
	if json.Unmarshal(out.Bytes(), &result) != nil || len(result.Profile) == 0 || result.Disclosure == nil || len(result.CandidateIDs) == 0 {
		t.Fatal("canonical draft/disclosure lost", out.String())
	}
	for _, unsafe := range []func(*CommandRequest){func(r *CommandRequest) { r.Version = "unknown" }, func(r *CommandRequest) { r.Start.Origins = []string{"https://other.example.test"} }, func(r *CommandRequest) {
		r.Draft.Flow.Steps[0].Navigate = "https://app.example.test/register?token=private-value"
	}, func(r *CommandRequest) { r.Draft.CallControls.Approval = "automatic" }} {
		changed := r
		changed.Draft.Flow.Steps = append([]Step(nil), r.Draft.Flow.Steps...)
		unsafe(&changed)
		bad, _ := json.Marshal(changed)
		out.Reset()
		errOut.Reset()
		if RunCommand([]string{"--request", "-"}, bytes.NewReader(bad), &out, &errOut) != 2 || out.Len() != 0 || strings.Contains(errOut.String(), "private-value") {
			t.Fatal("unsafe draft published or leaked", out.String(), errOut.String())
		}
	}
}
