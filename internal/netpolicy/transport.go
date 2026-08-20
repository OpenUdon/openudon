// Package netpolicy provides a DNS-pinned HTTP transport for bounded remote
// source discovery. Every redirect is revalidated and every DNS answer must be
// public before a connection is made.
package netpolicy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
)

type Resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

type Options struct {
	AllowUnsafe bool
	Resolver    Resolver
	Dialer      *net.Dialer
}

func SafeHTTPClient(base *http.Client, opts Options) (*http.Client, error) {
	if base == nil {
		base = &http.Client{}
	}
	roundTripper := base.Transport
	if roundTripper == nil {
		roundTripper = http.DefaultTransport
	}
	transport, ok := roundTripper.(*http.Transport)
	if !ok {
		return nil, fmt.Errorf("remote-source HTTP client uses a custom transport that cannot enforce DNS pinning")
	}
	clone := transport.Clone()
	if !opts.AllowUnsafe && clone.TLSClientConfig != nil && clone.TLSClientConfig.InsecureSkipVerify {
		return nil, fmt.Errorf("remote-source HTTP client must verify TLS certificates")
	}
	clone.Proxy = nil
	// A transport-level TLS dial hook bypasses DialContext entirely. Clear both
	// forms so every HTTP and HTTPS connection uses the validated, pinned IP.
	clone.DialTLS = nil
	clone.DialTLSContext = nil
	resolver := opts.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	dialer := opts.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	clone.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, fmt.Errorf("invalid remote address: %w", err)
		}
		answers, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve remote source host: %w", err)
		}
		if len(answers) == 0 {
			return nil, fmt.Errorf("remote source host resolved to no addresses")
		}
		for _, answer := range answers {
			if !opts.AllowUnsafe && UnsafeIP(answer.IP) {
				return nil, fmt.Errorf("remote source host has an unsafe DNS answer")
			}
		}
		ip := answers[0].IP
		dialNetwork := "tcp4"
		if ip.To4() == nil {
			dialNetwork = "tcp6"
		}
		return dialer.DialContext(ctx, dialNetwork, net.JoinHostPort(ip.String(), port))
	}
	baseRedirect := base.CheckRedirect
	client := *base
	client.Transport = clone
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many remote-source redirects")
		}
		if err := ValidateURL(req.URL, opts.AllowUnsafe); err != nil {
			return err
		}
		if baseRedirect != nil {
			return baseRedirect(req, via)
		}
		return nil
	}
	return &client, nil
}

func ValidateURL(parsed *url.URL, allowUnsafe bool) error {
	if parsed == nil || parsed.Hostname() == "" || parsed.User != nil {
		return fmt.Errorf("invalid source URL")
	}
	if !strings.EqualFold(parsed.Scheme, "https") && !(allowUnsafe && strings.EqualFold(parsed.Scheme, "http")) {
		return fmt.Errorf("source URL must use HTTPS")
	}
	host := strings.ToLower(parsed.Hostname())
	if !allowUnsafe && (host == "localhost" || strings.HasSuffix(host, ".localhost")) {
		return fmt.Errorf("unsafe local source host")
	}
	if ip := net.ParseIP(host); ip != nil && !allowUnsafe && UnsafeIP(ip) {
		return fmt.Errorf("unsafe private source host")
	}
	return nil
}

func UnsafeIP(ip net.IP) bool {
	address, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}
	address = address.Unmap()
	if address.Is4() {
		return addressInAnyPrefix(address, unsafeIPv4Prefixes)
	}
	bytes := address.As16()
	if nat64WellKnownPrefix.Contains(address) {
		embedded := netip.AddrFrom4([4]byte{bytes[12], bytes[13], bytes[14], bytes[15]})
		if addressInAnyPrefix(embedded, unsafeIPv4Prefixes) {
			return true
		}
	}
	if bytes[0] == 0x20 && bytes[1] == 0x02 { // 6to4 embeds IPv4 at bits 16..48.
		embedded := netip.AddrFrom4([4]byte{bytes[2], bytes[3], bytes[4], bytes[5]})
		if addressInAnyPrefix(embedded, unsafeIPv4Prefixes) {
			return true
		}
	}
	if bytes[0] == 0x20 && bytes[1] == 0x01 && bytes[2] == 0 && bytes[3] == 0 { // Teredo.
		server := netip.AddrFrom4([4]byte{bytes[4], bytes[5], bytes[6], bytes[7]})
		client := netip.AddrFrom4([4]byte{^bytes[12], ^bytes[13], ^bytes[14], ^bytes[15]})
		if addressInAnyPrefix(server, unsafeIPv4Prefixes) || addressInAnyPrefix(client, unsafeIPv4Prefixes) {
			return true
		}
	}
	return addressInAnyPrefix(address, unsafeIPv6Prefixes)
}

var unsafeIPv4Prefixes = mustPrefixes(
	"0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8",
	"169.254.0.0/16", "172.16.0.0/12", "192.0.0.0/24", "192.0.2.0/24",
	"192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24",
	"203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4",
)

var nat64WellKnownPrefix = netip.MustParsePrefix("64:ff9b::/96")

var unsafeIPv6Prefixes = mustPrefixes(
	"::/96", "::1/128", "64:ff9b:1::/48", "100::/64", "2001::/23", "2001:db8::/32",
	"2002::/16", "3fff::/20", "5f00::/16", "fc00::/7", "fe80::/10", "fec0::/10", "ff00::/8",
)

func mustPrefixes(values ...string) []netip.Prefix {
	result := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		result = append(result, netip.MustParsePrefix(value))
	}
	return result
}

func addressInAnyPrefix(address netip.Addr, prefixes []netip.Prefix) bool {
	for _, prefix := range prefixes {
		if prefix.Contains(address) {
			return true
		}
	}
	return false
}
