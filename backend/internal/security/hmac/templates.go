package hmac

import (
	"crypto/hmac"
	"fmt"
	"strings"
	"time"
)

const (
	ProviderIyzico Provider = "IYZICO"
)

// ProviderTemplate defines metadata, signature header, encoding, and mock test payload for a webhook provider.
type ProviderTemplate struct {
	ID                      Provider          `json:"id"`
	Name                    string            `json:"name"`
	Description             string            `json:"description"`
	DocsURL                 string            `json:"docsUrl"`
	SignatureHeader         string            `json:"signatureHeader"`
	Algorithm               string            `json:"algorithm"`
	Encoding                string            `json:"encoding"`
	DefaultToleranceSeconds int               `json:"defaultToleranceSeconds"`
	SamplePayload           string            `json:"samplePayload"`
	SampleHeaders           map[string]string `json:"sampleHeaders"`
}

// GetAvailableTemplates returns the catalog of pre-configured webhook provider templates.
func GetAvailableTemplates() []ProviderTemplate {
	return []ProviderTemplate{
		{
			ID:                      ProviderStripe,
			Name:                    "Stripe Webhook",
			Description:             "Stripe ödeme, abonelik ve chargeback olayları. t={timestamp},v1={hmac_sha256} formatında imza doğrulaması ve replay saldırı koruması içerir.",
			DocsURL:                 "https://stripe.com/docs/webhooks/signatures",
			SignatureHeader:         "Stripe-Signature",
			Algorithm:               "HMAC-SHA256",
			Encoding:                "HEX",
			DefaultToleranceSeconds: 300,
			SamplePayload: `{
  "id": "evt_1N4TestPaymentSuccess000",
  "object": "event",
  "api_version": "2023-10-16",
  "created": 1698765432,
  "type": "payment_intent.succeeded",
  "data": {
    "object": {
      "id": "pi_3N4Test987654",
      "object": "payment_intent",
      "amount": 4900,
      "currency": "usd",
      "status": "succeeded"
    }
  }
}`,
			SampleHeaders: map[string]string{
				"Content-Type":     "application/json",
				"Stripe-Signature": "t=1698765432,v1=9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
		},
		{
			ID:                      ProviderIyzico,
			Name:                    "iyzico Ödeme Bildirimi",
			Description:             "iyzico Checkout Form, Abonelik ve BKM Express webhook bildirimleri. X-IYZ-SIGNATURE başlığı ile HMAC-SHA256 doğrulaması yapar.",
			DocsURL:                 "https://dev.iyzipay.com/tr/webhook",
			SignatureHeader:         "X-IYZ-SIGNATURE",
			Algorithm:               "HMAC-SHA256",
			Encoding:                "HEX",
			DefaultToleranceSeconds: 300,
			SamplePayload: `{
  "iyziEventType": "CHECKOUT_FORM_AUTH",
  "iyziEventTime": 1698765432000,
  "paymentId": "12345678",
  "status": "SUCCESS",
  "price": "250.00",
  "paidPrice": "250.00",
  "currency": "TRY",
  "conversationId": "order_conv_987654"
}`,
			SampleHeaders: map[string]string{
				"Content-Type":    "application/json",
				"X-IYZ-SIGNATURE": "a1b2c3d4e5f67890abcdef1234567890abcdef1234567890abcdef1234567890",
			},
		},
		{
			ID:                      ProviderGitHub,
			Name:                    "GitHub Webhooks",
			Description:             "GitHub commit, push, pull request ve release olayları. X-Hub-Signature-256 başlığı ile sha256={hex} imzasını doğrular.",
			DocsURL:                 "https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries",
			SignatureHeader:         "X-Hub-Signature-256",
			Algorithm:               "HMAC-SHA256",
			Encoding:                "HEX",
			DefaultToleranceSeconds: 300,
			SamplePayload: `{
  "ref": "refs/heads/main",
  "repository": {
    "name": "apisentinel",
    "full_name": "apisentinel/apisentinel"
  },
  "pusher": {
    "name": "developer",
    "email": "dev@apisentinel.dev"
  },
  "head_commit": {
    "id": "e6a54162e07eb478b87a918a38a719d3",
    "message": "feat: release delivery control plane"
  }
}`,
			SampleHeaders: map[string]string{
				"Content-Type":        "application/json",
				"X-Hub-Signature-256": "sha256=2c5d88647e3a968a98319f074d6199859f7b1348888b14e9f73f73cd60df768f",
				"X-GitHub-Event":      "push",
			},
		},
		{
			ID:                      ProviderShopify,
			Name:                    "Shopify Webhook",
			Description:             "Shopify e-ticaret sipariş, müşteri ve stok bildirimleri. X-Shopify-Hmac-Sha256 başlığı ile Base64 kodlanmış HMAC-SHA256 doğrulaması yapar.",
			DocsURL:                 "https://shopify.dev/docs/apps/webhooks/configuration/https#step-5-verify-the-webhook",
			SignatureHeader:         "X-Shopify-Hmac-Sha256",
			Algorithm:               "HMAC-SHA256",
			Encoding:                "BASE64",
			DefaultToleranceSeconds: 300,
			SamplePayload: `{
  "id": 820982911946154508,
  "email": "customer@example.com",
  "total_price": "149.99",
  "currency": "USD",
  "financial_status": "paid",
  "line_items": [
    {
      "id": 866550311764710765,
      "title": "Cloud Security Sentinel Subscription",
      "price": "149.99",
      "quantity": 1
    }
  ]
}`,
			SampleHeaders: map[string]string{
				"Content-Type":           "application/json",
				"X-Shopify-Hmac-Sha256":  "4K8k3vF3/8mQJz1gLw+3Fk3F0dF8d1h8w5F2j3k4=",
				"X-Shopify-Topic":        "orders/paid",
				"X-Shopify-Shop-Domain": "store.myshopify.com",
			},
		},
		{
			ID:                      ProviderGeneric,
			Name:                    "Özel / Generic HMAC",
			Description:             "Kendi özel backend veya mikroservis webhook'larınız için standart HMAC-SHA256 imza koruması.",
			DocsURL:                 "https://apisentinel.dev/docs/hmac",
			SignatureHeader:         "X-Signature",
			Algorithm:               "HMAC-SHA256",
			Encoding:                "HEX",
			DefaultToleranceSeconds: 300,
			SamplePayload: `{
  "event": "user.signup",
  "userId": "usr_987654321",
  "timestamp": 1698765432
}`,
			SampleHeaders: map[string]string{
				"Content-Type": "application/json",
				"X-Signature":  "sha256=9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			},
		},
	}
}

// GenerateValidSignature generates a valid signature header value for testing and simulation.
func GenerateValidSignature(provider Provider, secret string, rawBody []byte, timestamp ...int64) string {
	ts := time.Now().Unix()
	if len(timestamp) > 0 && timestamp[0] > 0 {
		ts = timestamp[0]
	}

	switch provider {
	case ProviderStripe:
		signedPayload := fmt.Sprintf("%d.%s", ts, string(rawBody))
		sig := computeHMACSHA256Hex(secret, []byte(signedPayload))
		return fmt.Sprintf("t=%d,v1=%s", ts, sig)
	case ProviderGitHub:
		sig := computeHMACSHA256Hex(secret, rawBody)
		return fmt.Sprintf("sha256=%s", sig)
	case ProviderShopify:
		return computeHMACSHA256Base64(secret, rawBody)
	case ProviderIyzico:
		return computeHMACSHA256Hex(secret, rawBody)
	default:
		sig := computeHMACSHA256Hex(secret, rawBody)
		return fmt.Sprintf("sha256=%s", sig)
	}
}

// verifyIyzico checks X-IYZ-SIGNATURE / x-iyz-signature
func verifyIyzico(secret string, rawBody []byte, headers map[string][]string) error {
	sig := getHeader(headers, "x-iyz-signature", "x-iyzico-signature")
	if sig == "" {
		return ErrMissingSignature
	}

	expectedMac := computeHMACSHA256Hex(secret, rawBody)
	if !hmac.Equal([]byte(strings.ToLower(sig)), []byte(strings.ToLower(expectedMac))) {
		return ErrInvalidSignature
	}
	return nil
}
