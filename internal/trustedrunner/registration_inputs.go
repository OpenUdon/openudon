package trustedrunner

import (
	"fmt"
	"github.com/OpenUdon/openudon/internal/udonrunner"
	"net"
	"net/url"
)

// This is a trusted external Udon handoff. No form endpoint is proxied through
// iCoT, and no private input data or input digest enters the package.
func configurePrivateRegistration(browser *udonrunner.BrowserConfig, endpoint string) error {
	if endpoint == "" {
		return nil
	}
	if browser == nil || browser.Protocol != "v5" {
		return fmt.Errorf("a prepared private form service requires registration 1.1 execution")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "http" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Port() == "" || net.ParseIP(parsed.Hostname()) == nil || !net.ParseIP(parsed.Hostname()).IsLoopback() {
		return fmt.Errorf("private registration form service must be an exact loopback URL")
	}
	browser.RegistrationInputUI = false
	browser.RegistrationInputService = endpoint
	browser.RegistrationInputTokenEnv = "UDON_REGISTRATION_INPUT_TOKEN"
	browser.RegistrationInputExpectedEnv = "UDON_REGISTRATION_INPUT_EXPECTED"
	return nil
}
