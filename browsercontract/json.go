package browsercontract

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/OpenUdon/openudon/wire"
)

const MaxBytes = 1 << 20

var ErrContract = errors.New("invalid or unproved browser contract")

// CanonicalJSON implements the frozen lossless M51 encoding. It rejects
// ambiguous/invalid Unicode before decoding, preserves number lexemes, sorts
// UTF8 object keys, retains array order and does not escape HTML characters.
func CanonicalJSON(data []byte) ([]byte, error) {
	if len(data) == 0 || len(data) > 8<<20 || !utf8.Valid(data) || !validUnicodeEscapes(data) {
		return nil, ErrContract
	}
	var v any
	if wire.DecodeStrictNumbers(data, &v) != nil {
		return nil, ErrContract
	}
	var out bytes.Buffer
	e := json.NewEncoder(&out)
	e.SetEscapeHTML(false)
	if e.Encode(v) != nil {
		return nil, ErrContract
	}
	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}
func validUnicodeEscapes(data []byte) bool {
	in := false
	for i := 0; i < len(data); i++ {
		if data[i] == '"' {
			in = !in
			continue
		}
		if !in || data[i] != '\\' {
			continue
		}
		i++
		if i >= len(data) {
			return false
		}
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		n, err := strconv.ParseUint(string(data[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if n >= 0xdc00 && n <= 0xdfff {
			return false
		}
		if n >= 0xd800 && n <= 0xdbff {
			if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
				return false
			}
			low, err := strconv.ParseUint(string(data[i+3:i+7]), 16, 16)
			if err != nil || low < 0xdc00 || low > 0xdfff {
				return false
			}
			i += 6
		}
	}
	return true
}
func SHA256(data []byte) string { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func CanonicalDigest(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", ErrContract
	}
	raw, err = CanonicalJSON(raw)
	if err != nil {
		return "", err
	}
	return SHA256(raw), nil
}

// Decode enforces closed exact field names, mandatory fields and non-null
// arrays, in addition to the bounded duplicate/depth/node/Unicode checks.
func Decode(data []byte, out any) error {
	if len(data) > MaxBytes {
		return ErrContract
	}
	canonical, err := CanonicalJSON(data)
	if err != nil {
		return ErrContract
	}
	data = canonical
	if wire.DecodeStrictNumbers(data, out) != nil || closedShape(data, reflect.TypeOf(out)) != nil {
		return ErrContract
	}
	return nil
}
func closedShape(data []byte, t reflect.Type) error {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return ErrContract
	}
	if t == reflect.TypeFor[json.RawMessage]() {
		return nil
	}
	switch t.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if json.Unmarshal(data, &fields) != nil || fields == nil {
			return ErrContract
		}
		allowed := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("json"), ",")
			name := tag[0]
			allowed[name] = true
			raw, ok := fields[name]
			optional := len(tag) > 1 && tag[1] == "omitempty"
			if !ok {
				if !optional {
					return ErrContract
				}
				continue
			}
			if closedShape(raw, f.Type) != nil {
				return ErrContract
			}
		}
		for name := range fields {
			if !allowed[name] {
				return ErrContract
			}
		}
	case reflect.Slice:
		var values []json.RawMessage
		if json.Unmarshal(data, &values) != nil || values == nil {
			return ErrContract
		}
		for _, v := range values {
			if closedShape(v, t.Elem()) != nil {
				return ErrContract
			}
		}
	}
	return nil
}
