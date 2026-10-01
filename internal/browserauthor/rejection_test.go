package browserauthor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OpenUdon/browsertools/authordiagnostic"
)

func TestJoinedWorkerRejectionV2LegacyAndMalformed(t *testing.T) {
	class := authordiagnostic.Class{Stage: "policy", Reason: "origin_escape"}
	detail := authordiagnostic.Rejection{Boundary: "request", Resource: "script", OriginRelation: "host_mismatch"}
	v2, _ := json.Marshal(authordiagnostic.RecordV2{Version: authordiagnostic.VersionV2, Class: class, Rejection: detail})
	v1, _ := json.Marshal(authordiagnostic.Record{Version: authordiagnostic.Version, Class: class})
	for _, tc := range []struct {
		name, raw, status string
		class             authordiagnostic.Class
		rejection         authordiagnostic.Rejection
	}{
		{"v2", string(v2), "available", class, detail},
		{"v1", string(v1), "available", class, authordiagnostic.UnknownRejection()},
		{"invalid", strings.Replace(string(v2), "host_mismatch", "TOKEN_CANARY", 1), "invalid", authordiagnostic.Class{Stage: "unknown", Reason: "unknown"}, authordiagnostic.NoRejection()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			_ = os.Chmod(root, 0700)
			path := filepath.Join(root, "diagnostic")
			script := filepath.Join(root, "worker")
			body := `#!/bin/sh
umask 077
printf '%s\n' "$2" > "$1"
printf '%s\n' '{"protocol":"browsertools.author-session.v2","type":"hello","capabilities":["chromium","human_credentials","reviewed_mfa_kind","reviewed_outputs","reduced_observation","popup","frame","typed_goal"]}'
IFS= read -r start || exit 2
printf '%s\n' '{"protocol":"browsertools.author-session.v2","type":"diagnostic","diagnostic":{"code":"browser_failure"}}'
exit 1
`
			if os.WriteFile(script, []byte(body), 0700) != nil {
				t.Fatal("fixture")
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			config, err := normalizeConfig(Config{PrivateRoot: root, InitialURL: "http://127.0.0.1:12345/login", DashboardURL: "http://127.0.0.1:12345/dashboard", Goal: "Review", Origins: []string{"http://127.0.0.1:12345"}, ProfileID: "member"})
			if err != nil {
				t.Fatal(err)
			}
			config.backendDiagnosticPath = path
			s, err := startProcess(ctx, config, []string{script, path, tc.raw}, func() {})
			if err != nil {
				t.Fatal(err)
			}
			defer s.Cancel()
			for event := range s.Events() {
				data, _ := json.Marshal(event)
				if strings.Contains(string(data), "origin_relation") || strings.Contains(string(data), "CANARY") {
					t.Fatal("private detail reached protocol")
				}
			}
			status, got := s.BackendDiagnostic()
			if status != tc.status || got != tc.class || s.BackendRejectionDiagnostic() != tc.rejection {
				t.Fatal("joined private diagnostic mismatch", status, got, s.BackendRejectionDiagnostic())
			}
		})
	}
}
