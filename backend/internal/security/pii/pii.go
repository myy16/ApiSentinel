package pii

import (
	"regexp"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	phoneRegex = regexp.MustCompile(`\b(?:\+?90|0)?[5][0-9]{9}\b`)
	tcknRegex  = regexp.MustCompile(`\b[1-9]\d{10}\b`)
	ccRegex    = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
	ibanRegex  = regexp.MustCompile(`\bTR\d{2}[0-9A-Z]{5}[0-9A-Z]{17}\b`)
)

type Finding struct {
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	FieldPath      string  `json:"field_path"`
	Message        string  `json:"message"`
	EvidenceMasked string  `json:"evidence_masked"`
	Confidence     float64 `json:"confidence"`
}

// Luhn algorithm for valid credit cards
func ValidateLuhn(number string) bool {
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, number)

	if len(cleaned) < 13 || len(cleaned) > 19 {
		return false
	}

	var sum int
	alternate := false

	for i := len(cleaned) - 1; i >= 0; i-- {
		n := int(cleaned[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}

	return sum%10 == 0
}

// TCKN Validation algorithm
func ValidateTCKN(tckn string) bool {
	if len(tckn) != 11 || tckn[0] == '0' {
		return false
	}

	var digits [11]int
	for i, r := range tckn {
		if r < '0' || r > '9' {
			return false
		}
		digits[i] = int(r - '0')
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

	var totalSum int
	for i := 0; i < 10; i++ {
		totalSum += digits[i]
	}
	if digits[10] != totalSum%10 {
		return false
	}

	return true
}

func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "******"
	}
	name := parts[0]
	domain := parts[1]
	if len(name) <= 2 {
		return name[:1] + "***@" + domain
	}
	return name[:1] + "***" + name[len(name)-1:] + "@" + domain
}

func MaskCreditCard(cc string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, cc)
	if len(cleaned) < 4 {
		return "************"
	}
	last4 := cleaned[len(cleaned)-4:]
	return strings.Repeat("*", len(cleaned)-4) + last4
}

func MaskTCKN(tckn string) string {
	if len(tckn) != 11 {
		return "***********"
	}
	return "*********" + tckn[9:]
}

func ScanText(text string) []Finding {
	var findings []Finding

	// 1. Credit Cards (Luhn Algorithm)
	for _, match := range ccRegex.FindAllString(text, -1) {
		if ValidateLuhn(match) {
			findings = append(findings, Finding{
				Type:           "CREDIT_CARD",
				Severity:       "CRITICAL",
				Message:        "Valid credit card number detected in payload",
				EvidenceMasked: MaskCreditCard(match),
				Confidence:     0.99,
			})
		}
	}

	// 2. Turkish National ID (TCKN Algorithm)
	for _, match := range tcknRegex.FindAllString(text, -1) {
		if ValidateTCKN(match) {
			findings = append(findings, Finding{
				Type:           "TCKN",
				Severity:       "HIGH",
				Message:        "Verified Turkish National Identity (TCKN) detected",
				EvidenceMasked: MaskTCKN(match),
				Confidence:     0.95,
			})
		}
	}

	// 3. Turkish & International IBAN (mod97 Algorithm)
	for _, match := range ibanRegex.FindAllString(text, -1) {
		if ValidateIBAN(match) {
			findings = append(findings, Finding{
				Type:           "IBAN",
				Severity:       "HIGH",
				Message:        "Valid Turkish IBAN bank account number detected",
				EvidenceMasked: MaskIBAN(match),
				Confidence:     0.99,
			})
		}
	}

	// 4. Email Addresses
	for _, match := range emailRegex.FindAllString(text, -1) {
		findings = append(findings, Finding{
			Type:           "EMAIL",
			Severity:       "INFO",
			Message:        "Personal email address detected",
			EvidenceMasked: MaskEmail(match),
			Confidence:     0.90,
		})
	}

	return findings
}
