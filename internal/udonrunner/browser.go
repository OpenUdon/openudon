package udonrunner

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/authoring"
)

var browserBindingPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

var browserDriverEnvironmentAllowlist = map[string]bool{
	"PATH": true, "PATHEXT": true, "HOME": true, "TMPDIR": true, "TMP": true, "TEMP": true,
	"SystemRoot": true, "SYSTEMROOT": true, "WINDIR": true, "COMSPEC": true,
	"DISPLAY": true, "WAYLAND_DISPLAY": true, "XAUTHORITY": true, "DBUS_SESSION_BUS_ADDRESS": true,
	"LANG": true, "LC_ALL": true, "LC_CTYPE": true, "PLAYWRIGHT_BROWSERS_PATH": true,
}

func normalizeBrowserConfig(config *BrowserConfig) {
	if config == nil {
		return
	}
	if config.DriverArgs == nil {
		config.DriverArgs = []string{}
	}
	if config.DriverEnvironment == nil {
		config.DriverEnvironment = []string{}
	}
	if config.CredentialEnvironment == nil {
		config.CredentialEnvironment = []EnvironmentBinding{}
	}
	if config.SessionEnvironment == nil {
		config.SessionEnvironment = []EnvironmentBinding{}
	}
	if config.ApprovedOperations == nil {
		config.ApprovedOperations = []string{}
	}
	if config.ApprovedAuthentication == nil {
		config.ApprovedAuthentication = []string{}
	}
	if config.ApprovedRegistration == nil {
		config.ApprovedRegistration = []string{}
	}
	if config.AttestedRegistration == nil {
		config.AttestedRegistration = []string{}
	}
}

type validatedBrowserConfig struct {
	credentialEnv []string
	sessionEnv    []string
	driverEnv     []string
}

func validateBrowserConfig(config *BrowserConfig, credentials []string, values map[string]string, requireValues, requireDriver bool) (validatedBrowserConfig, error) {
	if config == nil {
		return validatedBrowserConfig{}, nil
	}
	normalizeBrowserConfig(config)
	config.DriverPath = strings.TrimSpace(config.DriverPath)
	config.Protocol = strings.ToLower(strings.TrimSpace(config.Protocol))
	if config.Protocol != "v1" && config.Protocol != "v2" && config.Protocol != "v3" && config.Protocol != "v4" {
		return validatedBrowserConfig{}, fmt.Errorf("run config browser protocol must be v1, v2, v3, or v4")
	}
	if requireDriver && config.DriverPath == "" {
		return validatedBrowserConfig{}, fmt.Errorf("browser workflow execution requires --browser-driver")
	}
	if config.DriverPath != "" {
		if err := rejectControlChars("browser driver path", config.DriverPath); err != nil {
			return validatedBrowserConfig{}, err
		}
		if !filepath.IsAbs(config.DriverPath) {
			return validatedBrowserConfig{}, fmt.Errorf("browser driver path must be absolute: %s", config.DriverPath)
		}
	}
	for _, arg := range config.DriverArgs {
		if err := rejectControlChars("browser driver argument", arg); err != nil {
			return validatedBrowserConfig{}, err
		}
		if authoring.ContainsLikelyCredentialValue([]byte(arg)) {
			return validatedBrowserConfig{}, fmt.Errorf("browser driver arguments must not contain credential values")
		}
	}
	if err := requireSortedUnique("browser driver environment", config.DriverEnvironment, func(value string) bool {
		return browserDriverEnvironmentAllowlist[value]
	}); err != nil {
		return validatedBrowserConfig{}, err
	}
	if err := requireSortedUnique("browser operation approvals", config.ApprovedOperations, browserBindingPattern.MatchString); err != nil {
		return validatedBrowserConfig{}, err
	}
	if err := requireSortedUnique("browser authentication approvals", config.ApprovedAuthentication, browserBindingPattern.MatchString); err != nil {
		return validatedBrowserConfig{}, err
	}
	if err := requireSortedUnique("browser registration approvals", config.ApprovedRegistration, browserBindingPattern.MatchString); err != nil {
		return validatedBrowserConfig{}, err
	}
	if err := requireSortedUnique("browser registration attestations", config.AttestedRegistration, browserBindingPattern.MatchString); err != nil {
		return validatedBrowserConfig{}, err
	}

	declared := make(map[string]bool, len(credentials))
	for _, binding := range credentials {
		declared[strings.TrimSpace(binding)] = true
	}
	credentialEnv, err := validateEnvironmentBindings("browser credential", config.CredentialEnvironment, func(binding string) (string, bool) {
		return CredentialEnvironmentName(binding), declared[binding]
	}, values, requireValues)
	if err != nil {
		return validatedBrowserConfig{}, err
	}
	sessionEnv, err := validateEnvironmentBindings("browser session", config.SessionEnvironment, func(binding string) (string, bool) {
		return sessionEnvName(binding), browserBindingPattern.MatchString(binding)
	}, values, requireValues)
	if err != nil {
		return validatedBrowserConfig{}, err
	}
	if config.Protocol == "v1" && (len(credentialEnv) != 0 || len(sessionEnv) != 0 || len(config.ApprovedAuthentication) != 0 || len(config.ApprovedRegistration) != 0 || len(config.AttestedRegistration) != 0) {
		return validatedBrowserConfig{}, fmt.Errorf("browser authentication and named sessions require protocol v2 or v3")
	}
	if len(config.ApprovedRegistration) != 0 && config.Protocol != "v3" && config.Protocol != "v4" {
		return validatedBrowserConfig{}, fmt.Errorf("browser registration requires protocol v3 dry-run evidence or protocol v4 execution")
	}
	if requireDriver && len(config.ApprovedRegistration) != 0 && config.Protocol != "v4" {
		return validatedBrowserConfig{}, fmt.Errorf("browser registration execution is unsupported by the current external executor contract")
	}
	if config.Protocol == "v4" {
		if len(config.ApprovedRegistration) != 1 || len(config.AttestedRegistration) != 1 || config.ApprovedRegistration[0] != config.AttestedRegistration[0] ||
			len(config.ApprovedOperations) != 0 || len(config.ApprovedAuthentication) != 0 || len(sessionEnv) != 0 || !validPrefixedSHA256(config.RegistrationAttestationSHA256) {
			return validatedBrowserConfig{}, fmt.Errorf("browser protocol v4 requires one exact registration attestation and submit approval without action, authentication, or session authority")
		}
	} else if len(config.AttestedRegistration) != 0 || strings.TrimSpace(config.RegistrationAttestationSHA256) != "" {
		return validatedBrowserConfig{}, fmt.Errorf("browser registration attestation requires protocol v4")
	}
	if requireValues {
		for _, name := range config.DriverEnvironment {
			if strings.TrimSpace(values[name]) == "" {
				return validatedBrowserConfig{}, fmt.Errorf("required browser driver env var is not set: %s", name)
			}
		}
	}
	return validatedBrowserConfig{
		credentialEnv: credentialEnv,
		sessionEnv:    sessionEnv,
		driverEnv:     append([]string(nil), config.DriverEnvironment...),
	}, nil
}

func validateEnvironmentBindings(label string, bindings []EnvironmentBinding, expected func(string) (string, bool), values map[string]string, requireValues bool) ([]string, error) {
	result := make([]string, 0, len(bindings))
	previous := ""
	seenEnv := map[string]bool{}
	for _, binding := range bindings {
		name := strings.TrimSpace(binding.Name)
		environment := strings.TrimSpace(binding.Environment)
		want, ok := expected(name)
		if !ok || !browserBindingPattern.MatchString(name) || environment != want || (previous != "" && name <= previous) || seenEnv[environment] {
			return nil, fmt.Errorf("%s environment mappings must be unique, sorted, symbolic, and canonical", label)
		}
		if requireValues && strings.TrimSpace(values[environment]) == "" {
			return nil, fmt.Errorf("required %s env var is not set: %s", label, environment)
		}
		previous = name
		seenEnv[environment] = true
		result = append(result, environment)
	}
	return result, nil
}

func requireSortedUnique(label string, values []string, valid func(string) bool) error {
	for index, value := range values {
		if strings.TrimSpace(value) != value || !valid(value) || (index > 0 && values[index-1] >= value) {
			return fmt.Errorf("%s must be unique, sorted, and valid", label)
		}
	}
	return nil
}

func sessionEnvName(binding string) string {
	return normalizedEnvironmentName("UDON_BROWSER_SESSION_", binding)
}

// CredentialEnvironmentName returns the canonical Udon credential variable.
func CredentialEnvironmentName(binding string) string {
	return normalizedEnvironmentName("UDON_CREDENTIAL_", binding)
}

// SessionEnvironmentName returns the canonical Udon browser-session variable.
func SessionEnvironmentName(binding string) string { return sessionEnvName(binding) }

func normalizedEnvironmentName(prefix, binding string) string {
	var b strings.Builder
	b.WriteString(prefix)
	lastUnderscore := false
	for _, ch := range strings.TrimSpace(binding) {
		if (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') {
			b.WriteRune(ch)
			lastUnderscore = false
		} else if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.TrimRight(strings.ToUpper(b.String()), "_")
}

func availableBrowserDriverEnvironment(values map[string]string) []string {
	result := make([]string, 0, len(browserDriverEnvironmentAllowlist))
	for name := range browserDriverEnvironmentAllowlist {
		if values[name] != "" {
			result = append(result, name)
		}
	}
	sort.Strings(result)
	return result
}

// AvailableBrowserDriverEnvironment returns the sorted names from the fixed
// launcher allowlist that are present in env. Values are never returned.
func AvailableBrowserDriverEnvironment(env []string) []string {
	if env == nil {
		env = os.Environ()
	}
	return availableBrowserDriverEnvironment(environmentMap(env))
}

// ValidateBrowserEvidenceConfig validates a value-free persisted browser
// contract without requiring the referenced runtime values or executable to
// be present on the verification host.
func ValidateBrowserEvidenceConfig(config *BrowserConfig, credentials []string) error {
	if config == nil {
		return nil
	}
	copy := *config
	copy.DriverArgs = append([]string(nil), config.DriverArgs...)
	copy.DriverEnvironment = append([]string(nil), config.DriverEnvironment...)
	copy.CredentialEnvironment = append([]EnvironmentBinding(nil), config.CredentialEnvironment...)
	copy.SessionEnvironment = append([]EnvironmentBinding(nil), config.SessionEnvironment...)
	copy.ApprovedOperations = append([]string(nil), config.ApprovedOperations...)
	copy.ApprovedAuthentication = append([]string(nil), config.ApprovedAuthentication...)
	copy.ApprovedRegistration = append([]string(nil), config.ApprovedRegistration...)
	copy.AttestedRegistration = append([]string(nil), config.AttestedRegistration...)
	_, err := validateBrowserConfig(&copy, credentials, map[string]string{}, false, false)
	return err
}

func validPrefixedSHA256(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != len("sha256:")+64 || !strings.HasPrefix(value, "sha256:") || value != strings.ToLower(value) {
		return false
	}
	for _, ch := range strings.TrimPrefix(value, "sha256:") {
		if ch < '0' || ch > '9' && ch < 'a' || ch > 'f' {
			return false
		}
	}
	return true
}
