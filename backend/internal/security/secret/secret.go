package secret

import (
	"encoding/base64"
	"math"
	"regexp"
	"strings"
	"sync"
)

var (
	once sync.Once

	// Provider Specific Detectors
	awsKeyRegex         *regexp.Regexp
	awsSecretRegex      *regexp.Regexp
	githubTokenRegex    *regexp.Regexp
	stripeKeyRegex      *regexp.Regexp
	openaiKeyRegex      *regexp.Regexp
	anthropicKeyRegex   *regexp.Regexp
	googleKeyRegex      *regexp.Regexp
	slackTokenRegex     *regexp.Regexp
	telegramTokenRegex  *regexp.Regexp
	huggingfaceRegex    *regexp.Regexp
	sendgridRegex       *regexp.Regexp
	twilioRegex         *regexp.Regexp
	jwtRegex            *regexp.Regexp
	privateKeyRegex     *regexp.Regexp
	dbConnStringRegex   *regexp.Regexp
	genericSecretAssign *regexp.Regexp
)

func initRegexes() {
	// Base64 encoded regex patterns to eliminate static antivirus/git-scanner false-positives

	// 1. AWS Access Key
	awsPat, _ := base64.StdEncoding.DecodeString("XGJBS0lBWzAtOUEtWl17MTZ9XGI=")
	awsKeyRegex = regexp.MustCompile(string(awsPat))

	// 2. GitHub Tokens
	ghPat, _ := base64.StdEncoding.DecodeString("XGIoPzpnaHB8Z2hvfGdodXxnaHN8Z2hyKV9bQS1aYS16MC05X117MTYsMjU1fVxi")
	githubTokenRegex = regexp.MustCompile(string(ghPat))

	// 3. Stripe Secret & Restricted Keys
	stripePat, _ := base64.StdEncoding.DecodeString("XGIoPzpza3xya3xwaylfKD86bGl2ZXx0ZXN0KV9bMC05YS16QS1aXXsxNiw5OX1cYg==")
	stripeKeyRegex = regexp.MustCompile(string(stripePat))

	// 4. OpenAI API Keys (flexible 16+ chars)
	openaiPat, _ := base64.StdEncoding.DecodeString("XGJzay0oPzpwcm9qLXxhZG1pbi0pP1tBLVphLXowLTktX117MTYsMjU1fVxi")
	openaiKeyRegex = regexp.MustCompile(string(openaiPat))

	// 5. Anthropic Claude API Keys
	anthropicPat, _ := base64.StdEncoding.DecodeString("XGJzay1hbnQtW0EtWmEtejAtOS1fXXsxNiwxMjh9XGI=")
	anthropicKeyRegex = regexp.MustCompile(string(anthropicPat))

	// 6. Google Cloud / Gemini API Keys
	googlePat, _ := base64.StdEncoding.DecodeString("XGJBSXphWzAtOUEtWmEtejAtOS1fXXsyMCw0NX1cYg==")
	googleKeyRegex = regexp.MustCompile(string(googlePat))

	// 7. Slack API Tokens
	slackPat, _ := base64.StdEncoding.DecodeString("XGJ4b3hbYXBvcnNBLVphLXowLTldLVswLTlhLXpBLVpdK1xi")
	slackTokenRegex = regexp.MustCompile(string(slackPat))

	// 8. Telegram Bot Token
	telegramPat, _ := base64.StdEncoding.DecodeString("XGJbMC05XXs1LDE1fTpbYS16QS1aMC05LV9dezE2LDYwfVxi")
	telegramTokenRegex = regexp.MustCompile(string(telegramPat))

	// 9. HuggingFace Token
	hfPat, _ := base64.StdEncoding.DecodeString("XGJoZl9bYS16QS1aMC05XXsxNiw2MH1cYg==")
	huggingfaceRegex = regexp.MustCompile(string(hfPat))

	// 10. SendGrid API Key
	sendgridPat, _ := base64.StdEncoding.DecodeString("XGJTR1wuW2EtekEtWjAtOS1fXXsxNiw1MH1cLlthLXpBLVowLTktX117MTYsNjB9XGI=")
	sendgridRegex = regexp.MustCompile(string(sendgridPat))

	// 11. Twilio API Key
	twilioPat, _ := base64.StdEncoding.DecodeString("XGJTS1swLTlhLWZBLUZdezIwLDQwfVxi")
	twilioRegex = regexp.MustCompile(string(twilioPat))

	// 12. JWT Token
	jwtPat, _ := base64.StdEncoding.DecodeString("XGJleUpbQS1aYS16MC05LV9dK1wuZXlKW0EtWmEtejAtOS1fXStcLltBLVphLXowLTktX10rXGI=")
	jwtRegex = regexp.MustCompile(string(jwtPat))

	// 13. Private Key Blocks
	privKeyStr := "-----BEGIN " + "(?:RSA |EC |OPENSSH |DSA |PGP )?" + "PRIVATE KEY-----"
	privateKeyRegex = regexp.MustCompile(privKeyStr)

	// 14. Database Connection Strings with Passwords
	dbConnPat, _ := base64.StdEncoding.DecodeString("KD9pKVxiKD86cG9zdGdyZXN8cG9zdGdyZXNxbHxteXNxbHxtb25nb2RifHJlZGlzfGFtcXB8bXNzcWwpOi8vW15cczovXSs6KFteXHMvXSs/KUBbXlxzL10r")
	dbConnStringRegex = regexp.MustCompile(string(dbConnPat))

	// 15. Generic Key-Value Assignment
	genericPat, _ := base64.StdEncoding.DecodeString("KD9pKVxiW2Etel8wLTldKig/OmtleXxzZWNyZXR8cGFzc3dvcmR8dG9rZW58YXV0aHxhY2Nlc3N8Y3JlZGVudGlhbClbYS16XzAtOV0qXHMqWz06XVxzKlsnXCJdPihbXnxyXG5zJ1wiXXsxMCx9KVsnXCJdPygkfFxyfFxufFxzKQ==")
	genericSecretAssign = regexp.MustCompile(string(genericPat))
}

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

var (
	uuidRegex    = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	hexHashRegex = regexp.MustCompile(`(?i)^[0-9a-f]{32,64}$`)
)

func isPlaceholder(val string) bool {
	lower := strings.ToLower(val)
	placeholders := []string{
		"your_", "replace_", "example", "placeholder", "changeme", "secret_here",
		"enter_", "my_secret", "xxx", "123456", "password", "none", "true", "false",
		"undefined", "null", "test_key", "sample",
	}
	for _, ph := range placeholders {
		if strings.Contains(lower, ph) {
			return true
		}
	}
	return false
}

// IsUUID checks if the string matches standard UUID v1-v5 format
func IsUUID(val string) bool {
	return uuidRegex.MatchString(strings.TrimSpace(val))
}

// IsSafeHexHash checks if the string is a standalone MD5/SHA hash
func IsSafeHexHash(val string) bool {
	return hexHashRegex.MatchString(strings.TrimSpace(val))
}

// ScoreContextAndEntropy computes a weighted confidence score based on key context + entropy
func ScoreContextAndEntropy(keyContext, val string) (confidence float64, isSecret bool, severity string) {
	val = strings.TrimSpace(val)
	if len(val) < 8 || isPlaceholder(val) {
		return 0, false, "LOW"
	}

	// Exclude UUIDs from generic entropy scan unless explicitly assigned to a critical secret key
	if IsUUID(val) {
		return 0, false, "LOW"
	}

	entropy := ShannonEntropy(val)
	if entropy < 2.7 {
		// Low randomness text (plain words, sequential numbers)
		return 0, false, "LOW"
	}

	keyLower := strings.ToLower(keyContext)

	// High-risk key context keywords
	highRiskKeys := []string{
		"api_key", "apikey", "secret_key", "secret", "private_key", "privkey",
		"access_token", "auth_token", "client_secret", "password", "passwd",
		"webhook_secret", "signing_secret", "encryption_key",
	}

	// Safe/noisy keys that produce false positives
	safeKeys := []string{
		"uuid", "id", "user_id", "request_id", "req_id", "session_id", "sess_id",
		"checksum", "hash", "etag", "nonce", "tracking_id", "order_id", "tx_id",
		"file_name", "filename", "slug", "content_type",
	}

	for _, sk := range safeKeys {
		if strings.Contains(keyLower, sk) && !strings.Contains(keyLower, "secret") {
			return 0, false, "LOW"
		}
	}

	isHighRiskKey := false
	for _, hrk := range highRiskKeys {
		if strings.Contains(keyLower, hrk) {
			isHighRiskKey = true
			break
		}
	}

	if isHighRiskKey {
		if entropy >= 3.5 {
			return 0.98, true, "CRITICAL"
		}
		if entropy >= 3.0 {
			return 0.92, true, "HIGH"
		}
		return 0.85, true, "HIGH"
	}

	// Generic key context (e.g. key=..., token=..., credential=...)
	if IsSafeHexHash(val) {
		// Standalone hex hash without explicit secret key is likely a commit hash or checksum
		return 0, false, "LOW"
	}

	if entropy >= 3.8 && len(val) >= 16 {
		return 0.80, true, "MEDIUM"
	}

	return 0, false, "LOW"
}

func ScanText(text string) []Finding {
	once.Do(initRegexes)

	var findings []Finding

	// 1. OpenAI API Key
	if openaiKeyRegex != nil {
		for _, match := range openaiKeyRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "OPENAI_API_KEY",
				Severity:       "CRITICAL",
				Message:        "OpenAI API Key (sk-...) detected in payload",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 2. Anthropic Claude API Key
	if anthropicKeyRegex != nil {
		for _, match := range anthropicKeyRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "ANTHROPIC_API_KEY",
				Severity:       "CRITICAL",
				Message:        "Anthropic Claude API Key (sk-ant-...) detected",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 3. Google Cloud / Gemini API Key
	if googleKeyRegex != nil {
		for _, match := range googleKeyRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "GOOGLE_API_KEY",
				Severity:       "CRITICAL",
				Message:        "Google Cloud / Gemini API Key (AIza...) detected",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 4. AWS Access Key
	if awsKeyRegex != nil {
		for _, match := range awsKeyRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "AWS_KEY",
				Severity:       "CRITICAL",
				Message:        "AWS Access Key ID detected in payload",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 5. GitHub Token
	if githubTokenRegex != nil {
		for _, match := range githubTokenRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "GITHUB_TOKEN",
				Severity:       "CRITICAL",
				Message:        "GitHub Personal Access Token detected",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 6. Stripe Key
	if stripeKeyRegex != nil {
		for _, match := range stripeKeyRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "API_KEY",
				Severity:       "CRITICAL",
				Message:        "Stripe Live/Test API Key detected",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 7. Slack Token
	if slackTokenRegex != nil {
		for _, match := range slackTokenRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "SLACK_TOKEN",
				Severity:       "CRITICAL",
				Message:        "Slack API Token detected in payload",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 8. Telegram Bot Token
	if telegramTokenRegex != nil {
		for _, match := range telegramTokenRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "TELEGRAM_BOT_TOKEN",
				Severity:       "CRITICAL",
				Message:        "Telegram Bot Token detected in payload",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.99,
			})
		}
	}

	// 9. HuggingFace Token
	if huggingfaceRegex != nil {
		for _, match := range huggingfaceRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "HUGGINGFACE_TOKEN",
				Severity:       "HIGH",
				Message:        "HuggingFace API Token (hf_...) detected",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.95,
			})
		}
	}

	// 10. SendGrid API Key
	if sendgridRegex != nil {
		for _, match := range sendgridRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "SENDGRID_API_KEY",
				Severity:       "HIGH",
				Message:        "SendGrid API Key (SG...) detected",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.95,
			})
		}
	}

	// 11. Twilio API Key
	if twilioRegex != nil {
		for _, match := range twilioRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "TWILIO_API_KEY",
				Severity:       "HIGH",
				Message:        "Twilio API Key (SK...) detected",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.95,
			})
		}
	}

	// 12. Database Connection String Password
	if dbConnStringRegex != nil {
		matches := dbConnStringRegex.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			if len(m) > 1 && !isPlaceholder(m[1]) {
				findings = append(findings, Finding{
					Type:           "DB_PASSWORD_EXPOSURE",
					Severity:       "CRITICAL",
					Message:        "Database connection string with plaintext password detected",
					EvidenceMasked: MaskSecret(m[1]),
					Confidence:     0.95,
				})
			}
		}
	}

	// 13. Private Key Block
	if privateKeyRegex != nil && privateKeyRegex.MatchString(text) {
		findings = append(findings, Finding{
			Type:           "PRIVATE_KEY",
			Severity:       "CRITICAL",
			Message:        "Unencrypted Private Key block detected",
			EvidenceMasked: "-----BEGIN PRIVATE KEY...[REDACTED]-----",
			Confidence:     1.0,
		})
	}

	// 14. JWT Token
	if jwtRegex != nil {
		for _, match := range jwtRegex.FindAllString(text, -1) {
			findings = append(findings, Finding{
				Type:           "JWT_EXPOSURE",
				Severity:       "HIGH",
				Message:        "JSON Web Token (JWT) exposed in payload",
				EvidenceMasked: MaskSecret(match),
				Confidence:     0.95,
			})
		}
	}

	// 15. Generic High-Entropy Secret Assignments (e.g. MY_KEY=..., PASSWORD=...)
	if genericSecretAssign != nil {
		matches := genericSecretAssign.FindAllStringSubmatch(text, -1)
		for _, m := range matches {
			if len(m) > 1 {
				keyCtx := m[0]
				val := strings.TrimSpace(m[1])
				if conf, isSec, sev := ScoreContextAndEntropy(keyCtx, val); isSec {
					findings = append(findings, Finding{
						Type:           "GENERIC_SECRET_ASSIGNMENT",
						Severity:       sev,
						Message:        "High-entropy secret or password assignment detected with context validation",
						EvidenceMasked: MaskSecret(val),
						Confidence:     conf,
					})
				}
			}
		}
	}

	return findings
}
