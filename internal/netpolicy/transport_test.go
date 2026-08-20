package netpolicy

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
)

type fixedResolver []net.IPAddr

func (r fixedResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) { return r, nil }

type customTransport struct{}

func (customTransport) RoundTrip(*http.Request) (*http.Response, error) { return nil, nil }

func TestSafeHTTPClientRejectsCustomTransport(t *testing.T) {
	if _, err := SafeHTTPClient(&http.Client{Transport: customTransport{}}, Options{}); err == nil {
		t.Fatal("custom transport was accepted")
	}
}

func TestSafeHTTPClientClearsTLSDialBypasses(t *testing.T) {
	base := &http.Transport{
		DialTLS: func(string, string) (net.Conn, error) {
			t.Fatal("DialTLS bypass was invoked")
			return nil, nil
		},
		DialTLSContext: func(context.Context, string, string) (net.Conn, error) {
			t.Fatal("DialTLSContext bypass was invoked")
			return nil, nil
		},
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	client, err := SafeHTTPClient(&http.Client{Transport: base}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	if transport.DialTLS != nil || transport.DialTLSContext != nil {
		t.Fatal("TLS-specific dial callback survived safe transport cloning")
	}
}

func TestSafeHTTPClientRejectsTLSVerificationBypassAndPreservesRedirectPolicy(t *testing.T) {
	if _, err := SafeHTTPClient(&http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}, Options{}); err == nil {
		t.Fatal("TLS verification bypass was accepted")
	}
	called := false
	client, err := SafeHTTPClient(&http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		called = true
		return http.ErrUseLastResponse
	}}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse("https://example.com/source")
	if err := client.CheckRedirect(&http.Request{URL: parsed}, nil); err != http.ErrUseLastResponse || !called {
		t.Fatalf("base redirect policy was not preserved: called=%t err=%v", called, err)
	}
}

func TestUnsafeIPRejectsReservedAndTransitionRanges(t *testing.T) {
	for _, raw := range []string{
		"100.64.0.1", "192.0.2.1", "198.18.0.1", "203.0.113.1", "240.0.0.1",
		"2001:db8::1", "2002:7f00:1::", "2001:0000:4136:e378:8000:63bf:3fff:fdd2",
		"64:ff9b::7f00:1", "64:ff9b:1::1", "::192.168.1.1", "fec0::1", "3fff::1",
	} {
		if !UnsafeIP(net.ParseIP(raw)) {
			t.Errorf("UnsafeIP(%s) = false", raw)
		}
	}
	if UnsafeIP(net.ParseIP("93.184.216.34")) {
		t.Fatal("public example address was classified unsafe")
	}
}

type rebindingResolver struct{ calls int }

func (r *rebindingResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	r.calls++
	if r.calls == 1 {
		return []net.IPAddr{{IP: net.ParseIP("8.8.8.8")}}, nil
	}
	return []net.IPAddr{{IP: net.ParseIP("127.0.0.1")}}, nil
}

func TestDialRejectsAnyUnsafeDNSAnswerBeforeDial(t *testing.T) {
	client, err := SafeHTTPClient(nil, Options{Resolver: fixedResolver{
		{IP: net.ParseIP("93.184.216.34")}, {IP: net.ParseIP("127.0.0.1")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	if _, err := transport.DialContext(context.Background(), "tcp", "example.com:443"); err == nil {
		t.Fatal("mixed public/private DNS answers were accepted")
	}
}

func TestRedirectsAreRevalidated(t *testing.T) {
	client, err := SafeHTTPClient(nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"http://example.com/source", "https://127.0.0.1/source", "https://user@example.com/source"} {
		parsed, err := url.Parse(target)
		if err != nil {
			t.Fatal(err)
		}
		if err := client.CheckRedirect(&http.Request{URL: parsed}, nil); err == nil {
			t.Errorf("unsafe redirect was accepted: %s", target)
		}
	}
	parsed, _ := url.Parse("https://example.com/source")
	if err := client.CheckRedirect(&http.Request{URL: parsed}, nil); err != nil {
		t.Fatalf("safe redirect rejected: %v", err)
	}
}

func TestDialPinsFirstValidatedResolution(t *testing.T) {
	resolver := &rebindingResolver{}
	client, err := SafeHTTPClient(nil, Options{Resolver: resolver, Dialer: &net.Dialer{Timeout: time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	transport := client.Transport.(*http.Transport)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, _ = transport.DialContext(ctx, "tcp", "source.example:443")
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want one validated lookup", resolver.calls)
	}
}
