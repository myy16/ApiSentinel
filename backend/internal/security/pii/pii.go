package pii

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	emailRegex = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	phoneRegex = regexp.MustCompile(`(?:\+90|0)?\s*[5]\d{2}\s*\d{3}\s*\d{2}\s*\d{2}`)
	tcknRegex  = regexp.MustCompile(`\b[1-9]\d{10}\b`)
	ccRegex    = regexp.MustCompile(`\b(?:\d[ -]*?){13,19}\b`)
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

// 11-digit Turkish National ID (TCKN) mod11 validation
func ValidateTCKN(tckn string) bool {
	if len(tckn) != 11 || tckn[0] == '0' {
		return false
	}

	digits := make([]int, 11)
	for i := 0; i < 11; i++ {
		d, err := strconv.Atoi(string(tckn[i]))
		if err != nil {
			return false
		}
		digits[i] = d
	}

	sumOdd := digits[0] + digits[2] + digits[4] + digits[6] + digits[8]
	sumEven := digits[1] + digits[3] + digits[5] + digits[7]

	d10 := ((sumOdd * 7) - sumEven) % 10
	if d10 < 0 {
		d10 += 10
	}
	if d10 != digits[9] {
		return false
	}

	var sumTotal int
	for i := 0; i < 10; i++ {
		sumTotal += digits[i]
	}
	d11 := sumTotal % 10

	return d11 == digits[10]
}

func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "******"
	}
	name := parts[0]
	if len(name) <= 2 {
		return name[:1] + "***@" + parts[1]
	}
	return name[:1] + "***" + name[len(name)-1:] + "@" + parts[1]
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
	return strings.Repeat("*", len(cleaned)-4) + cleaned[len(cleaned)-4:]
}

func MaskTCKN(tckn string) string {
	if len(tckn) < 2 {
		return "***********"
	}
	return strings.Repeat("*", len(tckn)-2) + tckn[len(tckn)-2:]
}

func ScanText(text string) []Finding {
	var findings []Finding

	// 1. Credit Card scan with Luhn check
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

	// 2. TCKN scan with mod11 check
	for _, match := range tcknRegex.FindAllString(text, -1) {
		if ValidateTCKN(match) {
			findings = append(findings, Finding{
				Type:           "TCKN",
				Severity:       "HIGH",
				Message:        "Verified Turkish National Identity (TCKN) detected",
				EvidenceMasked: MaskTCKN(match),
				Confidence:     0.99,
			})
		}
	}

	// 3. Email scan
	for _, match := range emailRegex.FindAllString(text, -1) {
		findings = append(findings, Finding{
			Type:           "EMAIL",
			Severity:       "INFO",
			Message:        "Personal email address detected",
			EvidenceMasked: MaskEmail(match),
			Confidence:     0.95,
		})
	}

	return findings
}
