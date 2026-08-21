package secret

import (
	"math"
	"regexp"
	"strings"
)

var (
	awsKeyRegex      = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	githubTokenRegex = regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9_]{36,255}\b`)
	stripeKeyRegex   = regexp.MustCompile(`\b(?:sk|pk)_(?:live|test)_[0-9a-zA-Z]{24,99}\b`)
	jwtRegex         = regexp.MustCompile(`\beyJ[A-Za-z0-9-_]+\.eyJ[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\b`)
	privateKeyRegex  = regexp.MustCompile(`-----BEGIN (?:RSA |EC |OPENSSH |DSA )?PRIVATE KEY-----`)
)

type Finding struct {
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	FieldPath      string  `json:"field_path"`
	Message        string  `json:"message"`
	EvidenceMasked string  `json:"evidence_masked"`
	Confidence     float64 `json:"confidence"`
}

// Calculate Shannon entropy of a string
func ShannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	counts := make(map[rune]float64)
	for _, r := range s {
		counts[r]++
	}
	var entropy float64
	length := float64(len(s))
	for _, count := range counts {
		freq := count / length
		entropy -= freq * math.Log2(freq)
	}
	return entropy
}

func MaskSecret(secret string) string {
	if len(secret) <= 8 {
		return "********"
	}
	prefix := secret[:4]
	suffix := secret[len(secret)-4:]
	return prefix + strings.Repeat("*", len(secret)-8) + suffix
}

func ScanText(text string) []Finding {
	var findings []Finding

	// 1. AWS Key
	for _, match := range awsKeyRegex.FindAllString(text, -1) {
		findings = append(findings, Finding{
			Type:           "AWS_KEY",
			Severity:       "CRITICAL",
			Message:        "AWS Access Key ID detected in payload",
			EvidenceMasked: MaskSecret(match),
			Confidence:     0.99,
		})
	}

	// 2. GitHub Token
	for _, match := range githubTokenRegex.FindAllString(text, -1) {
		findings = append(findings, Finding{
			Type:           "GITHUB_TOKEN",
			Severity:       "CRITICAL",
			Message:        "GitHub Personal Access Token detected",
			EvidenceMasked: MaskSecret(match),
			Confidence:     0.99,
		})
	}

	// 3. Stripe Secret/Publishable Key
	for _, match := range stripeKeyRegex.FindAllString(text, -1) {
		findings = append(findings, Finding{
			Type:           "API_KEY",
			Severity:       "CRITICAL",
			Message:        "Stripe Live/Test API Key detected",
			EvidenceMasked: MaskSecret(match),
			Confidence:     0.99,
		})
	}

	// 4. Private Key Block
	if privateKeyRegex.MatchString(text) {
		findings = append(findings, Finding{
			Type:           "PRIVATE_KEY",
			Severity:       "CRITICAL",
			Message:        "Unencrypted Private Key block detected",
			EvidenceMasked: "-----BEGIN PRIVATE KEY...[REDACTED]-----",
			Confidence:     1.0,
		})
	}

	// 5. JWT Token
	for _, match := range jwtRegex.FindAllString(text, -1) {
		findings = append(findings, Finding{
			Type:           "JWT_EXPOSURE",
			Severity:       "HIGH",
			Message:        "JSON Web Token (JWT) exposed in payload",
			EvidenceMasked: MaskSecret(match),
			Confidence:     0.95,
		})
	}

	return findings
}
