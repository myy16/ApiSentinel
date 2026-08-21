package ssrf

import (
	"net"
	"testing"
)

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		expected bool
	}{
		{"127.0.0.1", true},
		{"10.0.1.5", true},
		{"192.168.1.1", true},
		{"172.16.0.10", true},
		{"169.254.169.254", true}, // AWS Metadata
		{"::1", true},
		{"8.8.8.8", false},        // Public Google DNS
		{"1.1.1.1", false},        // Public Cloudflare DNS
	}

	for _, tt := range tests {
		parsed := net.ParseIP(tt.ip)
		if result := IsPrivateIP(parsed); result != tt.expected {
			t.Errorf("IsPrivateIP(%s) = %v; expected %v", tt.ip, result, tt.expected)
		}
	}
}

func TestValidateURL(t *testing.T) {
	// 1. Invalid Schemes
	if _, err := ValidateURL("ftp://example.com"); err != ErrInvalidScheme {
		t.Errorf("Expected ErrInvalidScheme for ftp, got %v", err)
	}

	if _, err := ValidateURL("file:///etc/passwd"); err != ErrInvalidScheme {
		t.Errorf("Expected ErrInvalidScheme for file scheme, got %v", err)
	}

	// 2. Loopback Target -> Must be blocked
	if _, err := ValidateURL("http://127.0.0.1:8080/hook"); err != ErrPrivateIPBlocked {
		t.Errorf("Expected ErrPrivateIPBlocked for 127.0.0.1, got %v", err)
	}

	// 3. Cloud Metadata Target -> Must be blocked
	if _, err := ValidateURL("http://169.254.169.254/latest/meta-data/"); err != ErrPrivateIPBlocked {
		t.Errorf("Expected ErrPrivateIPBlocked for cloud metadata, got %v", err)
	}
}
