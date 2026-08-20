package synthesize

import "testing"

func TestProductionEndpointURLClassifiesUnknownExternalHosts(t *testing.T) {
	for _, value := range []string{"https://api.vendor.com", "HTTPS://api.vendor.com", "https://sandbox.vendor.com", "http://staging.internal.example.io"} {
		if !productionEndpointURL(value) {
			t.Errorf("external endpoint classified non-production: %s", value)
		}
	}
	for _, value := range []string{"http://127.0.0.1:8080", "https://service.example.test", "https://api.example.com"} {
		if productionEndpointURL(value) {
			t.Errorf("reserved endpoint classified production: %s", value)
		}
	}
}
