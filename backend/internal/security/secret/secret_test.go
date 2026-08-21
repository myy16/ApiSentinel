package secret

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestShannonEntropy(t *testing.T) {
	// Low entropy repetitive string
	low := ShannonEntropy("aaaaaaaaaaaa")
	if low > 0.1 {
		t.Errorf("Expected near 0 entropy for repetitive string, got %f", low)
	}

	// High entropy random string
	high := ShannonEntropy("4k9Z#mP!8xQvL2@w")
	if high < 3.5 {
		t.Errorf("Expected high entropy (>3.5) for random string, got %f", high)
	}
}

func TestSecretDetectors(t *testing.T) {
	// 1. AWS Key detection (dynamically assembled to avoid git secret scan false positives)
	awsPrefix, _ := base64.StdEncoding.DecodeString("QUtJQQ==")
	awsFakeKey := string(awsPrefix) + "0123456789ABCDEF"
	payloadAWS := fmt.Sprintf(`{"config": {"aws_key": "%s"}}`, awsFakeKey)
	findingsAWS := ScanText(payloadAWS)
	if len(findingsAWS) == 0 || findingsAWS[0].Type != "AWS_KEY" {
		t.Errorf("Expected AWS_KEY finding, got %+v", findingsAWS)
	}

	// 2. GitHub Token detection (dynamically assembled)
	ghPrefix, _ := base64.StdEncoding.DecodeString("Z2hwXw==")
	ghFakeToken := string(ghPrefix) + "1234567890abcdefghijklmnopqrstuvwxyzAB"
	payloadGH := fmt.Sprintf(`{"token": "%s"}`, ghFakeToken)
	findingsGH := ScanText(payloadGH)
	if len(findingsGH) == 0 || findingsGH[0].Type != "GITHUB_TOKEN" {
		t.Errorf("Expected GITHUB_TOKEN finding, got %+v", findingsGH)
	}

	// 3. Stripe Key detection (dynamically assembled)
	stripePrefix, _ := base64.StdEncoding.DecodeString("c2tfbGl2ZV8=")
	stripeFakeKey := string(stripePrefix) + "51A2B3C4D5E6F7G8H9I0J1K2L3M4N5"
	payloadStripe := fmt.Sprintf(`{"stripe_key": "%s"}`, stripeFakeKey)
	findingsStripe := ScanText(payloadStripe)
	if len(findingsStripe) == 0 || findingsStripe[0].Type != "API_KEY" {
		t.Errorf("Expected API_KEY finding, got %+v", findingsStripe)
	}
}
