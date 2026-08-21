// Package evidencefile contains the bounded, strict file primitives used by
// review, run, and release evidence readers.
package evidencefile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring/atomicfile"
)

const DefaultMaxBytes int64 = 8 << 20

// ReadRegular reads a bounded regular file without following a final symlink.
func ReadRegular(path string, limit int64) ([]byte, os.FileInfo, error) {
	if strings.TrimSpace(path) == "" {
		return nil, nil, fmt.Errorf("file path is required")
	}
	if limit <= 0 {
		limit = DefaultMaxBytes
	}
	before, err := os.Lstat(path)
	if err != nil {
		return nil, nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("%s must be a regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(data)) > limit {
		return nil, nil, fmt.Errorf("%s exceeds the %d-byte limit", path, limit)
	}
	after, err := f.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !os.SameFile(before, after) || after.Size() != int64(len(data)) {
		return nil, nil, fmt.Errorf("%s changed while it was read", path)
	}
	current, err := os.Lstat(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s changed while it was read: %w", path, err)
	}
	if !current.Mode().IsRegular() || !os.SameFile(after, current) {
		return nil, nil, fmt.Errorf("%s changed while it was read", path)
	}
	return data, after, nil
}

// DecodeStrict decodes exactly one JSON value and rejects unknown fields.
func DecodeStrict(data []byte, out any) error {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("must contain a single JSON value")
		}
		return fmt.Errorf("must contain a single JSON value: %w", err)
	}
	return nil
}

func rejectDuplicateJSONKeys(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := scanJSONValue(dec, 0); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("must contain a single JSON value")
	}
	return nil
}

func scanJSONValue(dec *json.Decoder, depth int) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	if depth >= 64 {
		return fmt.Errorf("JSON nesting exceeds 64 levels")
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			keyToken, err := dec.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("invalid JSON object key")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON object key")
			}
			seen[key] = true
			if err := scanJSONValue(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return fmt.Errorf("invalid JSON object")
		}
	case '[':
		for dec.More() {
			if err := scanJSONValue(dec, depth+1); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim(']') {
			return fmt.Errorf("invalid JSON array")
		}
	default:
		return fmt.Errorf("invalid JSON delimiter")
	}
	return nil
}

func SHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// WriteDigestSidecar atomically writes the canonical digest sidecar for path.
func WriteDigestSidecar(path string, data []byte, mode os.FileMode) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("evidence file path is required")
	}
	line := "sha256:" + SHA256(data) + "  " + filepath.Base(path) + "\n"
	return atomicfile.Write(path+".sha256", []byte(line), mode)
}

func ValidSHA256(value string) bool { return validHex(value, 64) }

// ValidGitObject accepts only complete SHA-1 or SHA-256 Git object IDs.
func ValidGitObject(value string) bool {
	return validHex(value, 40) || validHex(value, 64)
}

func validHex(value string, size int) bool {
	value = strings.TrimSpace(value)
	if len(value) != size {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
