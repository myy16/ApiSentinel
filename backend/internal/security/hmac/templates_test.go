package hmac

import (
	"testing"
	"time"
)

func TestProviderTemplates_GetAvailableTemplates(t *testing.T) {
	templates := GetAvailableTemplates()
	if len(templates) < 5 {
		t.Fatalf("Expected at least 5 provider templates, got %d", len(templates))
	}

	foundStripe := false
	foundIyzico := false
	foundGitHub := false
	foundShopify := false
	foundGeneric := false

	for _, tmpl := range templates {
		if tmpl.ID == ProviderStripe {
			foundStripe = true
			if tmpl.SignatureHeader != "Stripe-Signature" {
				t.Errorf("Expected Stripe-Signature, got %s", tmpl.SignatureHeader)
			}
		}
		if tmpl.ID == ProviderIyzico {
			foundIyzico = true
			if tmpl.SignatureHeader != "X-IYZ-SIGNATURE" {
				t.Errorf("Expected X-IYZ-SIGNATURE, got %s", tmpl.SignatureHeader)
			}
		}
		if tmpl.ID == ProviderGitHub {
			foundGitHub = true
		}
		if tmpl.ID == ProviderShopify {
			foundShopify = true
		}
		if tmpl.ID == ProviderGeneric {
			foundGeneric = true
		}
	}

	if !foundStripe || !foundIyzico || !foundGitHub || !foundShopify || !foundGeneric {
		t.Errorf("Missing one or more expected provider templates")
	}
}

func TestProviderTemplates_GenerateAndVerifyAllProviders(t *testing.T) {
	secret := "whsec_super_secret_test_key_12345"
	payload := []byte(`{"event":"payment.succeeded","amount":5000}`)

	providers := []Provider{
		ProviderStripe,
		ProviderIyzico,
		ProviderGitHub,
		ProviderShopify,
		ProviderGeneric,
	}

	for _, p := range providers {
		t.Run(string(p), func(t *testing.T) {
			sig := GenerateValidSignature(p, secret, payload)
			if sig == "" {
				t.Fatalf("Expected non-empty signature for %s", p)
			}

			// Format headers based on provider
			headers := make(map[string][]string)
			switch p {
			case ProviderStripe:
				headers["Stripe-Signature"] = []string{sig}
			case ProviderIyzico:
				headers["X-IYZ-SIGNATURE"] = []string{sig}
			case ProviderGitHub:
				headers["X-Hub-Signature-256"] = []string{sig}
			case ProviderShopify:
				headers["X-Shopify-Hmac-Sha256"] = []string{sig}
			default:
				headers["X-Signature"] = []string{sig}
			}

			// Verify
			err := Verify(p, secret, payload, headers, 5*time.Minute)
			if err != nil {
				t.Fatalf("Verification failed for provider %s: %v", p, err)
			}

			// Tamper payload -> must fail
			tamperedPayload := []byte(`{"event":"payment.succeeded","amount":999999}`)
			err = Verify(p, secret, tamperedPayload, headers, 5*time.Minute)
			if err == nil {
				t.Fatalf("Expected verification failure on tampered payload for %s", p)
			}
		})
	}
}
