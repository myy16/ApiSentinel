package ssrf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	ErrInvalidScheme    = errors.New("only http and https schemes are allowed")
	ErrPrivateIPBlocked = errors.New("outbound requests to private, loopback, or cloud metadata IPs are blocked (SSRF Guard)")
)

// Private/Reserved CIDRs
var privateCIDRs = []*net.IPNet{
	mustParseCIDR("127.0.0.0/8"),     // Loopback
	mustParseCIDR("10.0.0.0/8"),      // RFC 1918 Class A
	mustParseCIDR("172.16.0.0/12"),   // RFC 1918 Class B
	mustParseCIDR("192.168.0.0/16"),  // RFC 1918 Class C
	mustParseCIDR("169.254.0.0/16"),  // Link-Local / Cloud Metadata (AWS/GCP/Azure 169.254.169.254)
	mustParseCIDR("0.0.0.0/8"),       // Current network
	mustParseCIDR("100.64.0.0/10"),   // Carrier-grade NAT
	mustParseCIDR("192.0.0.0/24"),    // IETF Protocol Assignments
	mustParseCIDR("192.0.2.0/24"),    // TEST-NET-1
	mustParseCIDR("198.18.0.0/15"),   // Network benchmark tests
	mustParseCIDR("198.51.100.0/24"), // TEST-NET-2
	mustParseCIDR("203.0.113.0/24"),  // TEST-NET-3
	mustParseCIDR("224.0.0.0/4"),     // Multicast
	mustParseCIDR("240.0.0.0/4"),     // Reserved
	mustParseCIDR("::1/128"),         // IPv6 Loopback
	mustParseCIDR("fc00::/7"),        // IPv6 Unique Local Address
	mustParseCIDR("fe80::/10"),       // IPv6 Link-Local
}

func mustParseCIDR(s string) *net.IPNet {
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return ipnet
}

// IsPrivateIP checks if an IP belongs to private, loopback, or metadata address spaces
func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified()
}

// ValidateURL ensures the target URL is safe for outbound execution
func ValidateURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, ErrInvalidScheme
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return nil, errors.New("empty hostname")
	}

	// Check if local dev forwarding is allowed
	allowLocal := os.Getenv("ALLOW_LOCAL_FORWARDING") == "true" || os.Getenv("ENV") == "development" || os.Getenv("APP_ENV") == "development" || os.Getenv("GIN_MODE") == "debug"

	// Resolve hostname to IP addresses
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %s: %w", hostname, err)
	}

	for _, ip := range ips {
		if IsPrivateIP(ip) {
			if allowLocal && (ip.IsLoopback() || ip.String() == "127.0.0.1" || ip.String() == "::1") {
				continue
			}
			return nil, ErrPrivateIPBlocked
		}
	}

	return parsed, nil
}

// SafeDialContext resolves the target host and validates resolved IPs at connection time
// to prevent DNS rebinding attacks.
func SafeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid network address %q: %w", addr, err)
	}

	allowLocal := os.Getenv("ALLOW_LOCAL_FORWARDING") == "true" || os.Getenv("ENV") == "development" || os.Getenv("APP_ENV") == "development" || os.Getenv("GIN_MODE") == "debug"

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("DNS resolution failed for %s: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP address found for %s", host)
	}

	for _, ip := range ips {
		if IsPrivateIP(ip) {
			if allowLocal && (ip.IsLoopback() || ip.String() == "127.0.0.1" || ip.String() == "::1") {
				continue
			}
			return nil, ErrPrivateIPBlocked
		}
	}

	var dialer net.Dialer
	target := net.JoinHostPort(ips[0].String(), port)
	return dialer.DialContext(ctx, network, target)
}

// NewSafeTransport returns an http.Transport configured with SafeDialContext.
func NewSafeTransport() *http.Transport {
	return &http.Transport{
		DialContext:           SafeDialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// NewSafeHTTPClient returns an *http.Client configured with SSRF protection against DNS rebinding.
func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &http.Client{
		Transport: NewSafeTransport(),
		Timeout:   timeout,
	}
}


