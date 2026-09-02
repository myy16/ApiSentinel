package hmac

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrMissingSignature = errors.New("missing webhook signature header")
	ErrInvalidSignature = errors.New("invalid webhook signature")
	ErrTimestampExpired = errors.New("webhook timestamp expired or in the future")
)

type Provider string

const (
	ProviderGeneric Provider = "GENERIC"
	ProviderStripe  Provider = "STRIPE"
	ProviderGitHub  Provider = "GITHUB"
	ProviderShopify Provider = "SHOPIFY"
)

// Verify verifies a webhook signature based on the configured provider.
func Verify(provider Provider, secret string, rawBody []byte, headers map[string][]string, maxTolerance time.Duration) error {
	if secret == "" {
		return nil // No secret configured, pass-through
	}

	if maxTolerance <= 0 {
		maxTolerance = 5 * time.Minute
	}

	switch provider {
	case ProviderStripe:
		return verifyStripe(secret, rawBody, headers, maxTolerance)
	case ProviderGitHub:
		return verifyGitHub(secret, rawBody, headers)
	case ProviderShopify:
		return verifyShopify(secret, rawBody, headers)
	case ProviderIyzico:
		return verifyIyzico(secret, rawBody, headers)
	default:
		return verifyGeneric(secret, rawBody, headers)
	}
}

// verifyGeneric checks X-Signature or X-Hub-Signature with standard HMAC-SHA256
func verifyGeneric(secret string, rawBody []byte, headers map[string][]string) error {
	sig := getHeader(headers, "x-signature", "x-hub-signature-256", "x-webhook-signature")
	if sig == "" {
		return ErrMissingSignature
	}

	sig = strings.TrimPrefix(sig, "sha256=")

	expectedMac := computeHMACSHA256Hex(secret, rawBody)
	if !hmac.Equal([]byte(sig), []byte(expectedMac)) {
		return ErrInvalidSignature
	}
	return nil
}

// verifyGitHub checks X-Hub-Signature-256: sha256=...
func verifyGitHub(secret string, rawBody []byte, headers map[string][]string) error {
	sig := getHeader(headers, "x-hub-signature-256")
	if sig == "" {
		return ErrMissingSignature
	}

	sig = strings.TrimPrefix(sig, "sha256=")
	expectedMac := computeHMACSHA256Hex(secret, rawBody)

	if !hmac.Equal([]byte(sig), []byte(expectedMac)) {
		return ErrInvalidSignature
	}
	return nil
}

// verifyShopify checks X-Shopify-Hmac-Sha256 (Base64 encoded HMAC-SHA256)
func verifyShopify(secret string, rawBody []byte, headers map[string][]string) error {
	sig := getHeader(headers, "x-shopify-hmac-sha256")
	if sig == "" {
		return ErrMissingSignature
	}

	expectedMac := computeHMACSHA256Base64(secret, rawBody)
	if !hmac.Equal([]byte(sig), []byte(expectedMac)) {
		return ErrInvalidSignature
	}
	return nil
}

// verifyStripe checks Stripe-Signature: t=timestamp,v1=signature
func verifyStripe(secret string, rawBody []byte, headers map[string][]string, maxTolerance time.Duration) error {
	headerVal := getHeader(headers, "stripe-signature")
	if headerVal == "" {
		return ErrMissingSignature
	}

	var timestampStr string
	var signatures []string

	pairs := strings.Split(headerVal, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]
		if key == "t" {
			timestampStr = val
		} else if key == "v1" {
			signatures = append(signatures, val)
		}
	}

	if timestampStr == "" || len(signatures) == 0 {
		return ErrInvalidSignature
	}

	ts, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return ErrInvalidSignature
	}

	// Verify timestamp tolerance to prevent replay attacks
	eventTime := time.Unix(ts, 0)
	now := time.Now()
	if now.Sub(eventTime) > maxTolerance || eventTime.Sub(now) > maxTolerance {
		return ErrTimestampExpired
	}

	// Stripe signature payload is formatted as: timestamp + "." + rawBody
	signedPayload := fmt.Sprintf("%s.%s", timestampStr, string(rawBody))
	expectedMac := computeHMACSHA256Hex(secret, []byte(signedPayload))

	matched := false
	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expectedMac)) {
			matched = true
			break
		}
	}

	if !matched {
		return ErrInvalidSignature
	}
	return nil
}

func computeHMACSHA256Hex(secret string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

func computeHMACSHA256Base64(secret string, data []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func getHeader(headers map[string][]string, keys ...string) string {
	for _, k := range keys {
		for hk, vals := range headers {
			if strings.EqualFold(hk, k) && len(vals) > 0 {
				return vals[0]
			}
		}
	}
	return ""
}
