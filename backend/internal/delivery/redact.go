package delivery

import (
	"net/http"
	"strings"
	"unicode/utf8"
)

// DefaultMaxBodySnippetBytes defines the maximum response snippet size to store in attempts (2 KB).
const DefaultMaxBodySnippetBytes = 2048

// sensitiveHeaderKeys list headers that must be masked before storage in audit/attempt logs.
var sensitiveHeaderKeys = map[string]bool{
	"authorization":          true,
	"proxy-authorization":    true,
	"x-api-key":              true,
	"api-key":                true,
	"apikey":                 true,
	"cookie":                 true,
	"set-cookie":             true,
	"x-auth-token":           true,
	"x-access-token":         true,
	"stripe-signature":       true,
	"x-hub-signature":        true,
	"x-hub-signature-256":    true,
	"x-shopify-hmac-sha256":  true,
	"x-iyz-signature":        true,
	"x-iyz-signature-v3":     true,
	"webhook-signature":      true,
}

// RedactHeaders produces a sanitized map of headers where sensitive keys are masked with [REDACTED].
func RedactHeaders(headers http.Header) map[string]string {
	if headers == nil {
		return make(map[string]string)
	}

	sanitized := make(map[string]string, len(headers))
	for k, vals := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(k))
		joinedVals := strings.Join(vals, ", ")

		if sensitiveHeaderKeys[lowerKey] {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = joinedVals
		}
	}
	return sanitized
}

// RedactHeaderMap performs redaction on a plain map[string]string of headers.
func RedactHeaderMap(headers map[string]string) map[string]string {
	if headers == nil {
		return make(map[string]string)
	}

	sanitized := make(map[string]string, len(headers))
	for k, v := range headers {
		lowerKey := strings.ToLower(strings.TrimSpace(k))
		if sensitiveHeaderKeys[lowerKey] {
			sanitized[k] = "[REDACTED]"
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}

// SanitizeBodySnippet trims response body to maxBytes and ensures valid UTF-8 boundary.
func SanitizeBodySnippet(rawBody []byte, maxBytes int) string {
	if len(rawBody) == 0 {
		return ""
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBodySnippetBytes
	}

	isTruncated := false
	if len(rawBody) > maxBytes {
		rawBody = rawBody[:maxBytes]
		isTruncated = true
	}

	// Ensure we do not cut in the middle of a multi-byte UTF-8 rune
	for len(rawBody) > 0 && !utf8.Valid(rawBody) {
		rawBody = rawBody[:len(rawBody)-1]
	}

	res := string(rawBody)
	if isTruncated {
		res += "\n... [TRUNCATED - MAX 2KB REACHED]"
	}

	return res
}
