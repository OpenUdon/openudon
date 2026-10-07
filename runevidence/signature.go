package runevidence

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"strings"

	"github.com/OpenUdon/openudon/wire"
)

const SignatureVersion = "openudon.run-evidence-signature.v1"

type Signature struct {
	Version         string `json:"version"`
	Algorithm       string `json:"algorithm"`
	KeyFingerprint  string `json:"key_fingerprint"`
	EvidenceSHA256  string `json:"evidence_sha256"`
	PublicKeyPEM    string `json:"public_key_pem"`
	SignatureBase64 string `json:"signature_base64"`
}

// VerifySignature verifies exact evidence bytes. Without a trusted key it
// proves embedded-key integrity only, not custody or signer authority.
func VerifySignature(evidence, envelopeBytes, trustedPEM []byte) error {
	if len(evidence) > wire.MaxBytes || len(envelopeBytes) > 1<<20 || len(trustedPEM) > 64<<10 {
		return ErrInvalidEvidence
	}
	var envelope Signature
	if wire.DecodeStrict(envelopeBytes, &envelope) != nil || envelope.Version != SignatureVersion || envelope.Algorithm != "ed25519" {
		return ErrInvalidEvidence
	}
	key, der, ok := parsePublicKey([]byte(envelope.PublicKeyPEM))
	if !ok || fingerprint(der) != envelope.KeyFingerprint {
		return ErrInvalidEvidence
	}
	digest := sha256.Sum256(evidence)
	if hex.EncodeToString(digest[:]) != envelope.EvidenceSHA256 {
		return ErrInvalidEvidence
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.SignatureBase64)
	if err != nil || !ed25519.Verify(key, digest[:], signature) {
		return ErrInvalidEvidence
	}
	if len(trustedPEM) != 0 {
		_, trustedDER, ok := parsePublicKey(trustedPEM)
		if !ok || fingerprint(trustedDER) != envelope.KeyFingerprint {
			return ErrInvalidEvidence
		}
	}
	return nil
}

func parsePublicKey(data []byte) (ed25519.PublicKey, []byte, bool) {
	block, rest := pem.Decode(data)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 || block.Type != "PUBLIC KEY" {
		return nil, nil, false
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, false
	}
	key, ok := parsed.(ed25519.PublicKey)
	return key, block.Bytes, ok
}
func fingerprint(der []byte) string {
	value := sha256.Sum256(der)
	return "sha256:" + hex.EncodeToString(value[:])
}
