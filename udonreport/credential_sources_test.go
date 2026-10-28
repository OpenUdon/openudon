package udonreport

import (
	"fmt"
	"testing"
)

func TestCredentialSourcesStrictContract(t *testing.T) {
	for _, version := range []string{VersionV3, VersionV4} {
		for _, tc := range []struct {
			name, member string
			valid        bool
		}{
			{"absent", "", true},
			{"empty", `,"credential_sources":[]`, true},
			{"valid", `,"credential_sources":[{"binding":"B","source":"env:EXAMPLE"}]`, true},
			{"null array", `,"credential_sources":null`, false},
			{"null item", `,"credential_sources":[null]`, false},
			{"wrong type", `,"credential_sources":{}`, false},
			{"missing binding", `,"credential_sources":[{"source":"env:B"}]`, false},
			{"missing source", `,"credential_sources":[{"binding":"B"}]`, false},
			{"empty binding", `,"credential_sources":[{"binding":"","source":"env:B"}]`, false},
			{"empty source", `,"credential_sources":[{"binding":"B","source":""}]`, false},
			{"unknown member", `,"credential_sources":[{"binding":"B","source":"env:B","extra":1}]`, false},
			{"member aliases", `,"credential_sources":[{"BINDING":"B","SOURCE":"env:B"}]`, false},
			{"field alias", `,"CREDENTIAL_SOURCES":[]`, false},
			{"null member", `,"credential_sources":[{"binding":null,"source":"env:B"}]`, false},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				data := []byte(fmt.Sprintf(`{"version":%q,"status":"error","started_at":"2026-10-28T00:00:00Z","finished_at":"2026-10-28T00:00:01Z","workflow_path":"workflow","workflow_format":"uws-json","workdir":".","error_code":"driver_error","error_summary":"failed"%s}`, version, tc.member))
				_, err := Decode(data)
				if (err == nil) != tc.valid {
					t.Fatalf("Decode error = %v; valid = %v", err, tc.valid)
				}
				if tc.valid && FailureCode(data) != "driver_error" {
					t.Fatalf("lost failure code: %s", FailureCode(data))
				}
			})
		}
	}
	for _, key := range []string{"credential_sources", "CREDENTIAL_SOURCES"} {
		data := []byte(fmt.Sprintf(`{"version":%q,"status":"success","started_at":"2026-10-28T00:00:00Z","finished_at":"2026-10-28T00:00:01Z","workflow_path":"workflow","workflow_format":"uws-json","workdir":".",%q:[]}`, VersionV2, key))
		if _, err := Decode(data); err == nil {
			t.Fatalf("v2 accepted %s", key)
		}
	}
}
