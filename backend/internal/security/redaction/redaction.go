package redaction

import (
	"encoding/json"
	"strings"

	"github.com/apisentinel/apisentinel/internal/security/pii"
)

const maskedValue = "[REDACTED]"

var sensitiveKeys = map[string]struct{}{
	"authorization": {}, "cookie": {}, "set-cookie": {}, "password": {}, "passwd": {},
	"secret": {}, "token": {}, "access_token": {}, "refresh_token": {}, "api_key": {},
	"apikey": {}, "client_secret": {}, "private_key": {}, "signature": {}, "x-signature": {},
	"email": {}, "phone": {}, "iban": {}, "credit_card": {}, "card_number": {}, "tckn": {},
}

// Headers returns a copy suitable for persistence and API responses. Sensitive header values are never retained.
func Headers(headers map[string][]string) map[string][]string {
	masked := make(map[string][]string, len(headers))
	for key, values := range headers {
		if isSensitiveKey(key) {
			masked[key] = []string{maskedValue}
			continue
		}
		masked[key] = append([]string(nil), values...)
	}
	return masked
}

// QueryParams returns a copy suitable for persistence and API responses.
func QueryParams(params map[string][]string) map[string][]string {
	masked := make(map[string][]string, len(params))
	for key, values := range params {
		if isSensitiveKey(key) {
			masked[key] = []string{maskedValue}
			continue
		}
		masked[key] = append([]string(nil), values...)
	}
	return masked
}

// Payload redacts sensitive values in JSON. Non-JSON bodies are represented without retaining their content.
func Payload(raw []byte) (masked string, parsedJSON []byte) {
	if len(raw) == 0 {
		return "", nil
	}

	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "[REDACTED: non-JSON payload]", nil
	}

	redactJSON(value)
	encoded, err := json.Marshal(value)
	if err != nil {
		return "[REDACTED: invalid JSON payload]", nil
	}
	return string(encoded), encoded
}

func redactJSON(value any) {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			if isSensitiveKey(key) {
				node[key] = maskedValue
				continue
			}
			if text, ok := child.(string); ok {
				node[key] = redactPII(text)
				continue
			}
			redactJSON(child)
		}
	case []any:
		for index, child := range node {
			if text, ok := child.(string); ok {
				node[index] = redactPII(text)
				continue
			}
			redactJSON(child)
		}
	}
}

func redactPII(text string) string {
	for _, match := range pii.FindCreditCards(text) {
		text = strings.ReplaceAll(text, match, pii.MaskCreditCard(match))
	}
	for _, match := range pii.FindTCKNs(text) {
		text = strings.ReplaceAll(text, match, pii.MaskTCKN(match))
	}
	for _, match := range pii.FindEmails(text) {
		text = strings.ReplaceAll(text, match, pii.MaskEmail(match))
	}
	for _, match := range pii.FindIBANs(text) {
		text = strings.ReplaceAll(text, match, pii.MaskIBAN(match))
	}
	return text
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if _, ok := sensitiveKeys[normalized]; ok {
		return true
	}
	return strings.Contains(normalized, "password") ||
		strings.Contains(normalized, "secret") ||
		strings.Contains(normalized, "token") ||
		strings.Contains(normalized, "api-key") ||
		strings.Contains(normalized, "apikey")
}
