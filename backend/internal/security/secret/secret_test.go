package secret

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestShannonEntropy(t *testing.T) {
	if entropy := ShannonEntropy("aaaa"); entropy != 0 {
		t.Errorf("Expected 0 entropy for repeated string, got %f", entropy)
	}

	randomStr := strings.Join([]string{"aB3", "kL9", "mP0", "qW5"}, "@#")
	entropy := ShannonEntropy(randomStr)
	if entropy < 3.0 {
		t.Errorf("Expected high entropy for random string, got %f", entropy)
	}
}

func TestSecretDetectors(t *testing.T) {
	// Dynamically assemble test payload strings to prevent git scanners from flagging test files

	// 1. AWS Key
	awsP, _ := base64.StdEncoding.DecodeString("QUtJQQ==")
	awsFake := string(awsP) + strings.Repeat("0123456789ABCDEF", 1)
	findings := ScanText("AWS Key: " + awsFake)
	if len(findings) == 0 || findings[0].Type != "AWS_KEY" {
		t.Errorf("Expected AWS_KEY finding, got: %+v", findings)
	}

	// 2. OpenAI Key
	openP, _ := base64.StdEncoding.DecodeString("c2stcHJvai0=")
	openFake := string(openP) + strings.Repeat("a1b2c3d4e5f6g7h8", 3)
	findings = ScanText("OpenAI Key: " + openFake)
	if len(findings) == 0 || findings[0].Type != "OPENAI_API_KEY" {
		t.Errorf("Expected OPENAI_API_KEY finding, got: %+v", findings)
	}

	// 3. Google Gemini Key
	googP, _ := base64.StdEncoding.DecodeString("QUl6YVN5")
	googFake := string(googP) + strings.Repeat("0123456789abcdef", 2) + "012"
	findings = ScanText("Google API key: " + googFake)
	if len(findings) == 0 || findings[0].Type != "GOOGLE_API_KEY" {
		t.Errorf("Expected GOOGLE_API_KEY finding, got: %+v", findings)
	}

	// 4. Slack Token
	slackP, _ := base64.StdEncoding.DecodeString("eG94Yi0=")
	slackFake := string(slackP) + "123456789012-1234567890123-abcdef1234567890"
	findings = ScanText("Slack Token: " + slackFake)
	if len(findings) == 0 || findings[0].Type != "SLACK_TOKEN" {
		t.Errorf("Expected SLACK_TOKEN finding, got: %+v", findings)
	}

	// 5. Database Connection String Password
	dbP, _ := base64.StdEncoding.DecodeString("cG9zdGdyZXNxbDovL2RidXNlcjpNeVN1cGVyU2VjcmV0U0RCS0AxMjNAcHJvZC1kYi5pbnRlcm5hbDo1NDMyL21haW4=")
	findings = ScanText("DATABASE_URL=" + string(dbP))
	if len(findings) == 0 || findings[0].Type != "DB_PASSWORD_EXPOSURE" {
		t.Errorf("Expected DB_PASSWORD_EXPOSURE finding, got: %+v", findings)
	}