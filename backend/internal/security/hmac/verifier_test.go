package hmac

import (
	"fmt"
	"testing"
	"time"
)

func TestVerify_Generic(t *testing.T) {
	secret := "whsec_test_secret_12345"
	payload := []byte(`{"event": "payment.success", "amount": 100}`)

	sig := computeHMACSHA256Hex(secret, payload)
	headers := map[string][]string{
		"X-Signature": {"sha256=" + sig},
	}

	// Valid signature
	err := Verify(ProviderGeneric, secret, payload, headers, 0)
	if err != nil {
		t.Fatalf("expected signature to be valid, got: %v", err)
	}

	// Tampered payload
	tamperedPayload := []byte(`{"event": "payment.success", "amount": 999999}`)
	err = Verify(ProviderGeneric, secret, tamperedPayload, headers, 0)
	if err != ErrInvalidSignature {
		t.Fatalf("expected ErrInvalidSignature for tampered payload, got: %v", err)
	}
}

func TestVerify_GitHub(t *testing.T) {
	secret := "github_webhook_secret_abc"
	payload := []byte(`{"action": "push", "ref": "refs/heads/main"}`)

	sig := computeHMACSHA256Hex(secret, payload)
	headers := map[string][]string{
		"X-Hub-Signature-256": {"sha256=" + sig},
	}

	err := Verify(ProviderGitHub, secret, payload, headers, 0)
	if err != nil {
		t.Fatalf("expected GitHub signature to pass, got: %v", err)
	}
}

func TestVerify_Shopify(t *testing.T) {
	secret := "shopify_shared_secret"
	payload := []byte(`{"id": 12345, "email": "customer@example.com"}`)

	sig := computeHMACSHA256Base64(secret, payload)
	headers := map[string][]string{
		"X-Shopify-Hmac-Sha256": {sig},
	}

	err := Verify(ProviderShopify, secret, payload, headers, 0)
	if err != nil {
		t.Fatalf("expected Shopify signature to pass, got: %v", err)
	}
}

func TestVerify_Stripe(t *testing.T) {
	secret := "whsec_stripe_test_key"
	payload := []byte(`{"id": "evt_123", "type": "charge.captured"}`)
	nowTs := time.Now().Unix()

	signedPayload := fmt.Sprintf("%d.%s", nowTs, string(payload))
	sig := computeHMACSHA256Hex(secret, []byte(signedPayload))

	headers := map[string][]string{
		"Stripe-Signature": {fmt.Sprintf("t=%d,v1=%s", nowTs, sig)},
	}

	// Valid Stripe signature
	err := Verify(ProviderStripe, secret, payload, headers, 5*time.Minute)
	if err != nil {
		t.Fatalf("expected Stripe signature to pass, got: %v", err)
	}

	// Expired Stripe signature (replay attack simulation)
	expiredTs := time.Now().Add(-10 * time.Minute).Unix()
	expiredSignedPayload := fmt.Sprintf("%d.%s", expiredTs, string(payload))
	expiredSig := computeHMACSHA256Hex(secret, []byte(expiredSignedPayload))

	expiredHeaders := map[string][]string{
		"Stripe-Signature": {fmt.Sprintf("t=%d,v1=%s", expiredTs, expiredSig)},
	}

	err = Verify(ProviderStripe, secret, payload, expiredHeaders, 5*time.Minute)
	if err != ErrTimestampExpired {
		t.Fatalf("expected ErrTimestampExpired for old signature, got: %v", err)
	}
}
