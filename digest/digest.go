// Package digest owns the unchanged value-only SHA-256 and object-ID helpers
// used by OpenUdon trust wires. Hashes describe bytes, never approval authority.
package digest

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func SHA256(data []byte) string        { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func ValidSHA256(value string) bool    { return validHex(value, 64) }
func ValidGitObject(value string) bool { return validHex(value, 40) || validHex(value, 64) }
func validHex(value string, size int) bool {
	value = strings.TrimSpace(value)
	if len(value) != size {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
