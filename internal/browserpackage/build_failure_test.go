package browserpackage

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeBuildFailureReportsCommittedAuthoringAndRefusesReplay(t *testing.T) {
	for _, mode := range []string{"authenticated", "registration"} {
		t.Run(mode, func(t *testing.T) {
			root, request := authorFixture(t, mode, false)
			// Authoring remains writable; only the later generated workflow
			// target conflicts. Exercise the actual native build after commit.
			if err := os.MkdirAll(filepath.Join(root, "workflows", "workflow.hcl"), 0700); err != nil {
				t.Fatal(err)
			}
			request.InputSHA256, _ = InputDigest(context.Background(), root)
			data, _ := json.Marshal(request)
			plan, err := Prepare(context.Background(), root, data)
			if err != nil || !plan.Ready {
				t.Fatal("ready authoring plan unavailable", err)
			}
			var out, diagnostic bytes.Buffer
			code := RunCommand(context.Background(), []string{"apply", "--example", root, "--request", "-", "--expected-plan", plan.PlanSHA256, "--confirmed"}, bytes.NewReader(data), &out, &diagnostic)
			var result Result
			if json.Unmarshal(out.Bytes(), &result) != nil || code != 1 || result.Outcome != "build_failed" || result.QualityStatus != "fail" || len(result.Written) == 0 || result.PlanSHA256 != plan.PlanSHA256 || result.RequestSHA256 != plan.RequestSHA256 {
				t.Fatalf("partial-write evidence missing: code=%d result=%+v", code, result)
			}
			if _, err := os.Stat(filepath.Join(root, "workflows", "intent.hcl")); err != nil {
				t.Fatal("committed authoring lost", err)
			}
			after, err := InputDigest(context.Background(), root)
			if err != nil || after == request.InputSHA256 {
				t.Fatal("partial build incorrectly claims unchanged package", err)
			}
			if _, err := Apply(context.Background(), root, data, plan.PlanSHA256, true); err == nil {
				t.Fatal("partial build permitted replay")
			}
		})
	}
}
