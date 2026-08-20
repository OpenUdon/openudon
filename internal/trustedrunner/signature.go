package trustedrunner

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

const RunEvidenceSignatureVersion = "openudon.run-evidence-signature.v1"

type RunEvidenceSignature struct {
	Version         string `json:"version"`
	Algorithm       string `json:"algorithm"`
	KeyFingerprint  string `json:"key_fingerprint"`
	EvidenceSHA256  string `json:"evidence_sha256"`
	PublicKeyPEM    string `json:"public_key_pem"`
	SignatureBase64 string `json:"signature_base64"`
}

type VerifyRunEvidenceOptions struct {
	TrustedPublicKey string
	RequireSignature bool
}

func SignaturePath(evidencePath string) string { return evidencePath + ".sig.json" }

func GenerateSigningKey(privatePath, publicPath string) error {
	if strings.TrimSpace(privatePath) == "" || strings.TrimSpace(publicPath) == "" {
		return fmt.Errorf("private and public key paths are required")
	}
	privateResolved, err := canonicalKeyOutputPath(privatePath)
	if err != nil {
		return fmt.Errorf("resolve private key path: %w", err)
	}
	publicResolved, err := canonicalKeyOutputPath(publicPath)
	if err != nil {
		return fmt.Errorf("resolve public key path: %w", err)
	}
	if sameCanonicalPath(privateResolved, publicResolved) {
		return fmt.Errorf("private and public key paths must resolve to distinct files")
	}
	for _, path := range []string{privatePath, publicPath} {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("refusing to overwrite existing key file: %s", path)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}
	publicDER, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return err
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER})
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})
	if err := atomicfile.WriteNew(privatePath, privatePEM, 0o600); err != nil {
		return fmt.Errorf("create private key without overwrite: %w", err)
	}
	if err := atomicfile.WriteNew(publicPath, publicPEM, 0o644); err != nil {
		return fmt.Errorf("create public key without overwrite (private key retained at %s): %w", privatePath, err)
	}
	return nil
}

func canonicalKeyOutputPath(path string) (string, error) {
	abs, err := filepath.Abs(strings.TrimSpace(path))
	if err != nil {
		return "", err
	}
	current := filepath.Clean(abs)
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(abs), nil
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func sameCanonicalPath(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func SignRunEvidenceFile(evidencePath, privateKeyPath string) (string, error) {
	privatePEM, err := loadSigningKeyPEM(privateKeyPath)
	if err != nil {
		return "", err
	}
	if len(privatePEM) == 0 {
		return "", fmt.Errorf("signing key path is required")
	}
	return signRunEvidence(evidencePath, privatePEM)
}

func loadSigningKeyPEM(privateKeyPath string) ([]byte, error) {
	privateKeyPath = strings.TrimSpace(privateKeyPath)
	if privateKeyPath == "" {
		return nil, nil
	}
	privatePEM, _, err := evidencefile.ReadRegular(privateKeyPath, 64<<10)
	if err != nil {
		return nil, fmt.Errorf("read signing key: %w", err)
	}
	if _, _, _, err := parsePrivateKey(privatePEM); err != nil {
		return nil, err
	}
	return append([]byte(nil), privatePEM...), nil
}

func signRunEvidence(evidencePath string, privatePEM []byte) (string, error) {
	evidence, _, err := evidencefile.ReadRegular(evidencePath, evidencefile.DefaultMaxBytes)
	if err != nil {
		return "", err
	}
	var document RunEvidence
	if err := evidencefile.DecodeStrict(evidence, &document); err != nil {
		return "", err
	}
	if document.Version != RunEvidenceVersion {
		return "", fmt.Errorf("only %s evidence can be signed", RunEvidenceVersion)
	}
	privateKey, publicKey, publicDER, err := parsePrivateKey(privatePEM)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(evidence)
	publicPEM, err := publicKeyPEM(publicKey)
	if err != nil {
		return "", err
	}
	envelope := RunEvidenceSignature{
		Version: RunEvidenceSignatureVersion, Algorithm: "ed25519",
		KeyFingerprint: fingerprint(publicDER), EvidenceSHA256: hex.EncodeToString(digest[:]),
		PublicKeyPEM: string(publicPEM), SignatureBase64: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, digest[:])),
	}
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return "", err
	}
	path := SignaturePath(evidencePath)
	if err := atomicfile.Write(path, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func verifyRunEvidenceSignature(evidencePath string, evidence []byte, opts VerifyRunEvidenceOptions) error {
	data, _, err := evidencefile.ReadRegular(SignaturePath(evidencePath), 1<<20)
	if err != nil {
		if os.IsNotExist(err) && !opts.RequireSignature && strings.TrimSpace(opts.TrustedPublicKey) == "" {
			return nil
		}
		if os.IsNotExist(err) {
			return fmt.Errorf("run evidence signature is required")
		}
		return fmt.Errorf("read run evidence signature: %w", err)
	}
	var envelope RunEvidenceSignature
	if err := evidencefile.DecodeStrict(data, &envelope); err != nil {
		return fmt.Errorf("run evidence signature must be valid JSON: %w", err)
	}
	if envelope.Version != RunEvidenceSignatureVersion || envelope.Algorithm != "ed25519" {
		return fmt.Errorf("unsupported run evidence signature envelope")
	}
	publicKey, publicDER, err := parsePublicKey([]byte(envelope.PublicKeyPEM))
	if err != nil {
		return fmt.Errorf("parse embedded public key: %w", err)
	}
	if fingerprint(publicDER) != envelope.KeyFingerprint {
		return fmt.Errorf("run evidence signature key fingerprint mismatch")
	}
	digest := sha256.Sum256(evidence)
	if hex.EncodeToString(digest[:]) != envelope.EvidenceSHA256 {
		return fmt.Errorf("run evidence signature digest mismatch")
	}
	signature, err := base64.StdEncoding.DecodeString(envelope.SignatureBase64)
	if err != nil || !ed25519.Verify(publicKey, digest[:], signature) {
		return fmt.Errorf("run evidence signature verification failed")
	}
	if trustedPath := strings.TrimSpace(opts.TrustedPublicKey); trustedPath != "" {
		trustedPEM, _, err := evidencefile.ReadRegular(trustedPath, 64<<10)
		if err != nil {
			return fmt.Errorf("read trusted public key: %w", err)
		}
		_, trustedDER, err := parsePublicKey(trustedPEM)
		if err != nil {
			return fmt.Errorf("parse trusted public key: %w", err)
		}
		if fingerprint(trustedDER) != envelope.KeyFingerprint {
			return fmt.Errorf("run evidence signer does not match the trusted public key")
		}
	}
	return nil
}

func parsePrivateKey(data []byte) (ed25519.PrivateKey, ed25519.PublicKey, []byte, error) {
	block, rest := pem.Decode(data)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 || block.Type != "PRIVATE KEY" {
		return nil, nil, nil, fmt.Errorf("signing key must be one PKCS#8 PEM private key")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse PKCS#8 signing key: %w", err)
	}
	privateKey, ok := parsed.(ed25519.PrivateKey)
	if !ok {
		return nil, nil, nil, fmt.Errorf("signing key must be Ed25519")
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	der, err := x509.MarshalPKIXPublicKey(publicKey)
	return privateKey, publicKey, der, err
}

func parsePublicKey(data []byte) (ed25519.PublicKey, []byte, error) {
	block, rest := pem.Decode(data)
	if block == nil || len(strings.TrimSpace(string(rest))) != 0 || block.Type != "PUBLIC KEY" {
		return nil, nil, fmt.Errorf("public key must be one PKIX PEM public key")
	}
	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, nil, err
	}
	publicKey, ok := parsed.(ed25519.PublicKey)
	if !ok {
		return nil, nil, fmt.Errorf("public key must be Ed25519")
	}
	return publicKey, block.Bytes, nil
}

func publicKeyPEM(key ed25519.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}), nil
}

func fingerprint(der []byte) string {
	digest := sha256.Sum256(der)
	return "sha256:" + hex.EncodeToString(digest[:])
}
