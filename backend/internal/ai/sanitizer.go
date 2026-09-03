package ai

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
)

// SanitizedResult holds the sanitized text along with metadata about what was masked.
type SanitizedResult struct {
	CleanText      string            `json:"cleanText"`
	RedactionCount int               `json:"redactionCount"`
	MaskedTypes    []string          `json:"maskedTypes"`
	DetectedThreat string            `json:"detectedThreat,omitempty"`
	Details        map[string]int    `json:"details"`
}

var (
	// PII Patterns
	emailRegex = regexp.MustCompile(`(?i)[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	phoneRegex = regexp.MustCompile(`(?:\+?\d{1,3}[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}`)
	tcknRegex  = regexp.MustCompile(`\b[1-9]\d{10}\b`)
	ibanRegex  = regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b`)
	cardRegex  = regexp.MustCompile(`\b\d{4}[ -]?\d{4}[ -]?\d{4}[ -]?\d{1,7}\b|\b\d{13,19}\b`)

	// Secret & Token Patterns
	stripeKeyRegex  = regexp.MustCompile(`\b(?:sk|rk)_(?:live|test)_[0-9a-zA-Z]{24,}\b`)
	awsKeyRegex     = regexp.MustCompile(`\b(?:AKIA|ABIA|ACCA|ASIA)[0-9A-Z]{16}\b`)
	githubPatRegex  = regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|ghr)_[0-9a-zA-Z]{36,}\b`)
	jwtRegex        = regexp.MustCompile(`\beyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\b`)
	genericHexToken = regexp.MustCompile(`(?i)\b(?:secret|token|password|auth|api[-_]?key)[\s:="']+([a-f0-9]{32,64})\b`)
	authBearerRegex = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9\-_.~+/]+=*`)
)

// isValidTCKN validates a Turkish National Identity Number (TC Kimlik No).
func isValidTCKN(tckn string) bool {
	if len(tckn) != 11 || tckn[0] == '0' {
		return false
	}
	digits := make([]int, 11)
	for i, c := range tckn {
		d, err := strconv.Atoi(string(c))
		if err != nil {
			return false
		}
		digits[i] = d
	}
	oddSum := digits[0] + digits[2] + digits[4] + digits[6] + digits[8]
	evenSum := digits[1] + digits[3] + digits[5] + digits[7]
	d10 := ((oddSum * 7) - evenSum) % 10
	if d10 < 0 {
		d10 += 10
	}
	if digits[9] != d10 {
		return false
	}
	totalSum := 0
	for i := 0; i < 10; i++ {
		totalSum += digits[i]
	}
	return digits[10] == (totalSum % 10)
}

// isValidLuhn validates credit card numbers using Luhn's Algorithm.
func isValidLuhn(number string) bool {
	clean := strings.ReplaceAll(strings.ReplaceAll(number, " ", ""), "-", "")
	if len(clean) < 13 || len(clean) > 19 {
		return false
	}
	sum := 0
	alternate := false
	for i := len(clean) - 1; i >= 0; i-- {
		digit, err := strconv.Atoi(string(clean[i]))
		if err != nil {
			return false
		}
		if alternate {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		alternate = !alternate
	}
	return sum%10 == 0
}

// SanitizeForAI performs zero-leakage PII and secret sanitization on any text or JSON string before passing to an LLM.
func SanitizeForAI(input string, customKeywords []string) SanitizedResult {
	if input == "" {
		return SanitizedResult{CleanText: "", Details: make(map[string]int)}
	}

	details := make(map[string]int)
	text := input

	// 1. Sanitize JWT Tokens
	if matches := jwtRegex.FindAllString(text, -1); len(matches) > 0 {
		text = jwtRegex.ReplaceAllString(text, "[REDACTED_JWT_TOKEN]")
		details["JWT_TOKEN"] += len(matches)
	}

	// 2. Sanitize Stripe API Keys
	if matches := stripeKeyRegex.FindAllString(text, -1); len(matches) > 0 {
		text = stripeKeyRegex.ReplaceAllString(text, "[REDACTED_STRIPE_KEY]")
		details["STRIPE_KEY"] += len(matches)
	}

	// 3. Sanitize AWS Keys
	if matches := awsKeyRegex.FindAllString(text, -1); len(matches) > 0 {
		text = awsKeyRegex.ReplaceAllString(text, "[REDACTED_AWS_KEY]")
		details["AWS_KEY"] += len(matches)
	}

	// 4. Sanitize GitHub Tokens
	if matches := githubPatRegex.FindAllString(text, -1); len(matches) > 0 {
		text = githubPatRegex.ReplaceAllString(text, "[REDACTED_GITHUB_TOKEN]")
		details["GITHUB_TOKEN"] += len(matches)
	}

	// 5. Sanitize Authorization Bearer Headers
	if matches := authBearerRegex.FindAllString(text, -1); len(matches) > 0 {
		text = authBearerRegex.ReplaceAllString(text, "Bearer [REDACTED_AUTH_TOKEN]")
		details["AUTH_HEADER"] += len(matches)
	}

	// 6. Sanitize Credit Card Numbers with Luhn check
	text = cardRegex.ReplaceAllStringFunc(text, func(match string) string {
		clean := strings.ReplaceAll(strings.ReplaceAll(match, " ", ""), "-", "")
		if isValidLuhn(clean) {
			details["CREDIT_CARD"]++
			return "[REDACTED_CREDIT_CARD]"
		}
		return match
	})

	// 7. Sanitize Turkish TCKN with check digit algorithm
	text = tcknRegex.ReplaceAllStringFunc(text, func(match string) string {
		if isValidTCKN(match) {
			details["TCKN"]++
			return "[REDACTED_TCKN]"
		}
		return match
	})

	// 8. Sanitize IBANs
	if matches := ibanRegex.FindAllString(text, -1); len(matches) > 0 {
		text = ibanRegex.ReplaceAllString(text, "[REDACTED_IBAN]")
		details["IBAN"] += len(matches)
	}

	// 9. Sanitize Emails
	if matches := emailRegex.FindAllString(text, -1); len(matches) > 0 {
		text = emailRegex.ReplaceAllString(text, "[REDACTED_EMAIL]")
		details["EMAIL"] += len(matches)
	}

	// 10. Sanitize Phone Numbers
	if matches := phoneRegex.FindAllString(text, -1); len(matches) > 0 {
		text = phoneRegex.ReplaceAllString(text, "[REDACTED_PHONE]")
		details["PHONE"] += len(matches)
	}

	// 11. Sanitize JSON key-values if valid JSON
	var jsonObj map[string]interface{}
	if err := json.Unmarshal([]byte(text), &jsonObj); err == nil {
		sanitizedMap := sanitizeJSONMap(jsonObj, customKeywords, details)
		if updatedBytes, err := json.Marshal(sanitizedMap); err == nil {
			text = string(updatedBytes)
		}
	} else {
		// Custom keyword sanitization for raw text
		for _, kw := range customKeywords {
			kwTrim := strings.TrimSpace(kw)
			if kwTrim == "" {
				continue
			}
			customRegex := regexp.MustCompile(`(?i)(` + regexp.QuoteMeta(kwTrim) + `[\s:="']+)([^\s"',}]+)`)
			if matches := customRegex.FindAllString(text, -1); len(matches) > 0 {
				text = customRegex.ReplaceAllString(text, `${1}[REDACTED_CUSTOM]`)
				details["CUSTOM_"+strings.ToUpper(kwTrim)] += len(matches)
			}
		}
	}

	totalRedactions := 0
	var maskedTypes []string
	for k, v := range details {
		totalRedactions += v
		maskedTypes = append(maskedTypes, k)
	}

	return SanitizedResult{
		CleanText:      text,
		RedactionCount: totalRedactions,
		MaskedTypes:    maskedTypes,
		Details:        details,
	}
}

func sanitizeJSONMap(m map[string]interface{}, customKeys []string, details map[string]int) map[string]interface{} {
	sensitiveKeyPatterns := []string{
		"password", "secret", "token", "apikey", "api_key", "cvv", "cvc", "pin",
		"private_key", "access_token", "refresh_token", "client_secret",
	}

	res := make(map[string]interface{})
	for k, v := range m {
		kLower := strings.ToLower(k)
		isSensitive := false

		for _, pat := range sensitiveKeyPatterns {
			if strings.Contains(kLower, pat) {
				isSensitive = true
				break
			}
		}

		if !isSensitive {
			for _, ck := range customKeys {
				if strings.EqualFold(k, ck) {
					isSensitive = true
					break
				}
			}
		}

		if isSensitive {
			res[k] = "[REDACTED_SECRET]"
			details["SENSITIVE_KEY_"+strings.ToUpper(k)]++
			continue
		}

		switch val := v.(type) {
		case map[string]interface{}:
			res[k] = sanitizeJSONMap(val, customKeys, details)
		case []interface{}:
			var arr []interface{}
			for _, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					arr = append(arr, sanitizeJSONMap(itemMap, customKeys, details))
				} else if strVal, ok := item.(string); ok {
					subRes := SanitizeForAI(strVal, customKeys)
					arr = append(arr, subRes.CleanText)
				} else {
					arr = append(arr, item)
				}
			}
			res[k] = arr
		case string:
			subRes := SanitizeForAI(val, customKeys)
			res[k] = subRes.CleanText
		default:
			res[k] = v
		}
	}
	return res
}
