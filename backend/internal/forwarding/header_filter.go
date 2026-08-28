package forwarding

import "strings"

// allowedForwardHeaders defines headers safe to forward to upstream targets.
// Only headers in this set (case-insensitive) are forwarded; all others are stripped.
var allowedForwardHeaders = map[string]struct{}{
	"content-type":     {},
	"content-length":   {},
	"accept":           {},
	"accept-encoding":  {},
	"accept-language":  {},
	"user-agent":       {},
	"x-request-id":     {},
	"x-correlation-id": {},
	"x-forwarded-for":  {},
	"x-real-ip":        {},
	"idempotency-key":  {},
}

// sensitiveHeaders defines headers that MUST never be forwarded to upstream.
// These are checked even if a future change expands the allowlist.
var sensitiveHeaders = map[string]struct{}{
	"authorization":         {},
	"cookie":                {},
	"set-cookie":            {},
	"x-api-key":             {},
	"x-signature":           {},
	"x-webhook-signature":   {},
	"stripe-signature":      {},
	"x-hub-signature-256":   {},
	"x-shopify-hmac-sha256": {},
	"proxy-authorization":   {},
}

// hopByHopHeaders must be removed per HTTP/1.1 spec (RFC 7230 §6.1).
var hopByHopHeaders = map[string]struct{}{
	"connection":          {},
	"keep-alive":          {},
	"proxy-authenticate":  {},
	"proxy-authorization": {},
	"te":                  {},
	"trailers":            {},
	"transfer-encoding":   {},
	"upgrade":             {},
}

// FilterHeaders returns a new map containing only safe, non-sensitive headers
// that are appropriate for forwarding to an upstream target.
func FilterHeaders(headers map[string]string) map[string]string {
	filtered := make(map[string]string, len(headers))
	for key, value := range headers {
		lower := strings.ToLower(key)

		// Always block sensitive and hop-by-hop headers
		if _, sensitive := sensitiveHeaders[lower]; sensitive {
			continue
		}
		if _, hop := hopByHopHeaders[lower]; hop {
			continue
		}

		// Only forward explicitly allowed headers
		if _, allowed := allowedForwardHeaders[lower]; allowed {
			filtered[key] = value
		}
	}
	return filtered
}
