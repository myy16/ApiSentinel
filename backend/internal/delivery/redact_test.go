package delivery

import (
	"net/http"
	"strings"
	"testing"
)

func TestRedactHeaders(t *testing.T) {
	headers := http.Header{
		"Content-Type":          []string{"application/json"},
		"Authorization":         []string{"Bearer secret_token_123"},
		"X-Api-Key":             []string{"apk_live_xyz"},
		"Cookie":                []string{"session_id=abc; other=123"},
		"Set-Cookie":            []string{"jwt=secret_jwt"},
		"Stripe-Signature":      []string{"t=12345,v1=signature_hash"},
		"X-Hub-Signature-256":   []string{"sha256=abcdef"},
		"X-Custom-Safe":         []string{"safe-value"},
	}

	sanitized := RedactHeaders(headers)

	if sanitized["Authorization"] != "[REDACTED]" {
		t.Errorf("expected Authorization to be [REDACTED], got %s", sanitized["Authorization"])
	}
	if sanitized["X-Api-Key"] != "[REDACTED]" {
		t.Errorf("expected X-Api-Key to be [REDACTED], got %s", sanitized["X-Api-Key"])
	}
	if sanitized["Cookie"] != "[REDACTED]" {
		t.Errorf("expected Cookie to be [REDACTED], got %s", sanitized["Cookie"])
	}
	if sanitized["Set-Cookie"] != "[REDACTED]" {
		t.Errorf("expected Set-Cookie to be [REDACTED], got %s", sanitized["Set-Cookie"])
	}
	if sanitized["Stripe-Signature"] != "[REDACTED]" {
		t.Errorf("expected Stripe-Signature to be [REDACTED], got %s", sanitized["Stripe-Signature"])
	}
	if sanitized["X-Hub-Signature-256"] != "[REDACTED]" {
		t.Errorf("expected X-Hub-Signature-256 to be [REDACTED], got %s", sanitized["X-Hub-Signature-256"])
	}
	if sanitized["Content-Type"] != "application/json" {
		t.Errorf("expected Content-Type to remain untouched, got %s", sanitized["Content-Type"])
	}
	if sanitized["X-Custom-Safe"] != "safe-value" {
		t.Errorf("expected X-Custom-Safe to remain untouched, got %s", sanitized["X-Custom-Safe"])
	}
}

func TestSanitizeBodySnippet(t *testing.T) {
	// Small body
	small := []byte(`{"status": "ok"}`)
	s1 := SanitizeBodySnippet(small, 2048)
	if s1 != string(small) {
		t.Errorf("expected %s, got %s", string(small), s1)
	}

	// Large body truncation
	large := []byte(strings.Repeat("A", 3000))
	s2 := SanitizeBodySnippet(large, 500)
	if len(s2) > 600 || !strings.Contains(s2, "[TRUNCATED") {
		t.Errorf("expected truncation banner, got length %d", len(s2))
	}

	// Empty body
	s3 := SanitizeBodySnippet(nil, 2048)
	if s3 != "" {
		t.Errorf("expected empty string for nil body")
	}
}
