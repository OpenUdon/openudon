// Package credentialpolicy centralizes literal-secret detection for generated
// artifacts and LLM-proposed request mappings.
package credentialpolicy

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

var (
	providerPatterns = []*regexp.Regexp{
		regexp.MustCompile(`AIza[0-9A-Za-z_-]{20,}`),
		regexp.MustCompile(`GOCSPX-[0-9A-Za-z_-]{16,}`),
		regexp.MustCompile(`1//[0-9A-Za-z_-]{20,}`),
		regexp.MustCompile(`xox[a-z]-[0-9A-Za-z-]{10,}`),
		regexp.MustCompile(`xapp-[0-9A-Za-z-]{10,}`),
		regexp.MustCompile(`sk-ant-api[0-9A-Za-z_-]*-[0-9A-Za-z_-]{20,}`),
		regexp.MustCompile(`sk-(?:proj-)?[0-9A-Za-z_-]{20,}`),
		regexp.MustCompile(`gh[pousr]_[0-9A-Za-z]{20,}`),
		regexp.MustCompile(`github_pat_[0-9A-Za-z_]{20,}`),
		regexp.MustCompile(`(?:AKIA|ASIA)[0-9A-Z]{16}`),
	}
	jwtPattern          = regexp.MustCompile(`[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)
	bearerPattern       = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/-]{16,}`)
	sensitiveKey        = regexp.MustCompile(`(?i)^(?:[A-Za-z0-9_.-]*(?:api[_-]?key|apikey|app[_-]?id|appid|token|secret|password|authorization|credential|refresh[_-]?token)[A-Za-z0-9_.-]*)$`)
	sensitiveAssignment = regexp.MustCompile(`(?im)["']?([A-Za-z0-9_.-]*(?:api[_-]?key|apikey|app[_-]?id|appid|token|secret|password|authorization|credential|refresh[_-]?token)[A-Za-z0-9_.-]*)["']?[ \t]*[:=][ \t]*([^\r\n]+)`)
	symbolicPattern     = regexp.MustCompile(`^(?:(?:inputs|credentials)\.[A-Za-z_][A-Za-z0-9_-]*|(?:[A-Za-z_][A-Za-z0-9_-]*\.)?(?:received_body|result|output|outputs)(?:\[[0-9]+\]|\.[A-Za-z_][A-Za-z0-9_-]*)*)$`)
	tokenPattern        = regexp.MustCompile(`^[!-~]+$`)
)

func ContainsLikelyValue(data []byte) bool {
	for _, pattern := range providerPatterns {
		if pattern.Match(data) {
			return true
		}
	}
	if bearerPattern.Match(data) {
		return true
	}
	for _, candidate := range jwtPattern.FindAll(data, -1) {
		if isJWT(string(candidate)) {
			return true
		}
	}
	if containsSensitiveJSONValue(data) {
		return true
	}
	for _, match := range sensitiveAssignment.FindAllSubmatch(data, -1) {
		if len(match) < 3 {
			continue
		}
		value := assignmentValue(string(match[2]))
		if !isSafeURLAssignment(string(match[1]), value) && isSensitiveLiteral(value) {
			return true
		}
	}
	return false
}

func assignmentValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Container-valued assignments describe schema or binding structure, not a
	// credential literal. Valid JSON is inspected recursively above, which also
	// catches sensitive scalar values nested inside these containers.
	if raw[0] == '{' || raw[0] == '[' {
		return ""
	}
	if raw[0] == '"' || raw[0] == '\'' {
		quote := raw[0]
		escaped := false
		for i := 1; i < len(raw); i++ {
			if quote == '"' && raw[i] == '\\' && !escaped {
				escaped = true
				continue
			}
			if raw[i] == quote && !escaped {
				return raw[1:i]
			}
			escaped = false
		}
		return strings.Trim(raw, `"'`)
	}
	if end := strings.IndexAny(raw, " \t,#]}"); end >= 0 {
		raw = raw[:end]
	}
	return strings.TrimSpace(raw)
}

func containsSensitiveJSONValue(data []byte) bool {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return false
	}
	if decoder.Decode(&struct{}{}) == nil {
		return false
	}
	return sensitiveJSONValue(value, false, "")
}

func sensitiveJSONValue(value any, sensitive bool, key string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for childKey, child := range typed {
			// Container names such as credential_bindings and
			// flow_credential_slots describe symbolic metadata. Inspect their
			// children by each child's own key instead of treating every name in
			// the container as a credential value.
			if sensitiveJSONValue(child, sensitiveKey.MatchString(childKey), childKey) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if sensitiveJSONValue(child, false, "") {
				return true
			}
		}
	case string:
		return sensitive && !isSafeURLAssignment(key, typed) && isSensitiveLiteral(typed)
	}
	return false
}

// IsSymbolicReference recognizes only the documented workflow value grammar.
func IsSymbolicReference(value string) bool {
	return symbolicPattern.MatchString(strings.TrimSpace(value))
}

func IsLikelyLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" || IsSymbolicReference(value) {
		return false
	}
	for _, pattern := range providerPatterns {
		if pattern.MatchString(value) {
			return true
		}
	}
	if bearerPattern.MatchString(value) || isJWT(value) {
		return true
	}
	if len(value) < 20 || !tokenPattern.MatchString(value) {
		return false
	}
	var letters, digits, punctuation bool
	for _, r := range value {
		letters = letters || unicode.IsLetter(r)
		digits = digits || unicode.IsDigit(r)
		punctuation = punctuation || (!unicode.IsLetter(r) && !unicode.IsDigit(r))
	}
	classes := 0
	for _, present := range []bool{letters, digits, punctuation} {
		if present {
			classes++
		}
	}
	entropy := shannonEntropy(value)
	return (classes >= 2 && entropy >= 3.5) || (len(value) >= 24 && entropy >= 4.0)
}

func SafeMappingValue(value string) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= 240 && !strings.ContainsAny(value, "\r\n") &&
		!ContainsLikelyValue([]byte(value)) && !IsLikelyLiteral(value)
}

func isSensitiveLiteral(value string) bool {
	value = strings.TrimSpace(value)
	if IsSymbolicReference(value) {
		return false
	}
	if IsLikelyLiteral(value) {
		return true
	}
	if len(value) < 16 || !tokenPattern.MatchString(value) {
		return false
	}
	var letters, digits bool
	for _, r := range value {
		letters = letters || unicode.IsLetter(r)
		digits = digits || unicode.IsDigit(r)
	}
	return letters && digits
}

func isSafeURLAssignment(key, value string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	if !strings.HasSuffix(key, "url") && !strings.HasSuffix(key, "_url") && !strings.HasSuffix(key, "-url") {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Hostname() != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" &&
		(strings.EqualFold(parsed.Scheme, "https") || strings.EqualFold(parsed.Scheme, "http"))
}

func isJWT(candidate string) bool {
	parts := strings.Split(candidate, ".")
	if len(parts) != 3 {
		return false
	}
	header := map[string]any{}
	payload := map[string]any{}
	return decodeBase64JSON(parts[0], &header) && decodeBase64JSON(parts[1], &payload) && len(header) > 0 && len(payload) > 0
}

func decodeBase64JSON(segment string, out any) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(segment)
	if err != nil {
		decoded, err = base64.URLEncoding.DecodeString(segment)
		if err != nil {
			return false
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(decoded))
	decoder.UseNumber()
	return decoder.Decode(out) == nil
}

func shannonEntropy(value string) float64 {
	counts := map[rune]int{}
	var total int
	for _, r := range value {
		counts[r]++
		total++
	}
	if total == 0 {
		return 0
	}
	var entropy float64
	for _, count := range counts {
		p := float64(count) / float64(total)
		entropy -= p * math.Log2(p)
	}
	return entropy
}
