package digest_test

import (
	"strings"
	"testing"

	"github.com/OpenUdon/openudon/digest"
)

func TestLegacyDigestContracts(t *testing.T) {
	const abc = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if digest.SHA256([]byte("abc")) != abc {
		t.Fatal("SHA-256 vector changed")
	}
	if !digest.ValidSHA256(" " + strings.ToUpper(abc) + "\n") {
		t.Fatal("legacy whitespace/hex contract changed")
	}
	if digest.ValidSHA256(abc[:63]) || digest.ValidSHA256(strings.Repeat("z", 64)) {
		t.Fatal("malformed digest accepted")
	}
	if !digest.ValidGitObject(abc) || !digest.ValidGitObject(abc[:40]) || digest.ValidGitObject(abc[:39]) {
		t.Fatal("Git object width contract changed")
	}
}
