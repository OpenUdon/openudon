package udonreport

import (
	"encoding/json"
	"testing"
)

func TestV4VerificationCodesDoNotChangePublishedV2V3(t *testing.T) {
	for code := range verificationFailureCodes {
		for _, version := range []string{VersionV2, VersionV3, VersionV4} {
			report := Report{Version: version, Status: "error", StartedAt: "2026-09-13T12:00:00Z", FinishedAt: "2026-09-13T12:00:01Z", WorkflowPath: "workflow.uws.yaml", WorkflowFormat: "uws-yaml", WorkDir: ".", ErrorCode: code, ErrorSummary: "Verification stopped."}
			data, _ := json.Marshal(report)
			_, err := Decode(data)
			if (err == nil) != (version == VersionV4) {
				t.Fatalf("version %s code %s acceptance wrong", version, code)
			}
		}
	}
}
