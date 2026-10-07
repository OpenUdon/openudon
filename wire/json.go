// Package wire supplies bounded, duplicate-rejecting JSON decoding for public trust APIs.
// Error text can include decoder details; consumers redact it before display/storage.
package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const MaxBytes = 8 << 20
const MaxNodes = 100000

// DecodeStrict decodes exactly one JSON value and rejects unknown fields.
func DecodeStrict(data []byte, out any) error {
	return decodeStrict(data, out, false)
}

// DecodeStrictNumbers additionally preserves JSON numeric inputs exactly.
// Use it when caller data participates in canonical request matching.
func DecodeStrictNumbers(data []byte, out any) error {
	return decodeStrict(data, out, true)
}

func decodeStrict(data []byte, out any, numbers bool) error {
	if len(data) > MaxBytes {
		return fmt.Errorf("JSON exceeds byte limit")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if numbers {
		dec.UseNumber()
	}
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
	if err := scanJSONValue(dec, 0, new(int)); err != nil {
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

func scanJSONValue(dec *json.Decoder, depth int, nodes *int) error {
	*nodes++
	if *nodes > MaxNodes {
		return fmt.Errorf("JSON exceeds node limit")
	}
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
			key = strings.ToLower(key)
			if seen[key] {
				return fmt.Errorf("duplicate JSON object key")
			}
			seen[key] = true
			if err := scanJSONValue(dec, depth+1, nodes); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return fmt.Errorf("invalid JSON object")
		}
	case '[':
		for dec.More() {
			if err := scanJSONValue(dec, depth+1, nodes); err != nil {
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
