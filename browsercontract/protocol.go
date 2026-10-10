package browsercontract

// Protocols selects the frozen per-leaf preferred protocol pair. It does not
// imply native runtime support; the consuming runtime must independently admit.
func Protocols(profile string) (outer, inner string) {
	switch profile {
	case "uws.browser-authentication.1.0":
		return "udon.browser-driver.v2", ""
	case "uws.browser-authentication.1.1":
		return "udon.browser-driver.v3", ""
	case "uws.browser.1.5", "uws.browser.1.6", "uws.browser.1.7":
		return "udon.browser-driver.v3", "udon.browser-driver.v2"
	case "uws.browser.1.8", "uws.browser.1.9":
		return "udon.browser-driver.v10", "udon.browser-driver.v3"
	case "uws.browser.1.10":
		return "udon.browser-driver.v11", "udon.browser-driver.v4"
	case "uws.browser-registration.1.0":
		return "udon.browser-driver.v4", ""
	case "uws.browser-registration.1.1":
		return "udon.browser-driver.v5", ""
	case "uws.browser-registration.1.2":
		return "udon.browser-driver.v6", ""
	}
	return "", ""
}
