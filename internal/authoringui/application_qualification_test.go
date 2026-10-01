package ui

import "testing"

func TestQualificationRejectsNonFixtureAuthority(t *testing.T) {
	for _, test := range []struct {
		origin, target string
		valid          bool
	}{
		{"http://127.0.0.1:12345", "http://127.0.0.1:12345/login", true},
		{"http://127.0.0.1:12345", "http://127.0.0.1:12345/register?action=startnew", true},
		{"http://127.0.0.1:12345@example.test", "http://127.0.0.1:12345@example.test/login", false},
		{"http://127.0.0.1:12345", "http://127.0.0.1:12345@example.test/login", false},
		{"http://127.0.0.1:12345", "http://127.0.0.1:12346/login", false},
		{"http://localhost:12345", "http://localhost:12345/login", false},
		{"http://127.0.0.1:0", "http://127.0.0.1:0/login", false},
		{"http://127.0.0.1:65536", "http://127.0.0.1:65536/login", false},
		{"http://127.0.0.1:12345/", "http://127.0.0.1:12345/login", false},
		{"http://127.0.0.1:12345", "http://127.0.0.1:12345/login#secret", false},
	} {
		if got := qualificationLoopbackURL(test.origin, test.target); got != test.valid {
			t.Fatalf("fixture URL validation = %v, want %v", got, test.valid)
		}
	}
}
