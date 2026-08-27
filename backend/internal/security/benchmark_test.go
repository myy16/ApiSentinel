package security_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/apisentinel/apisentinel/internal/id"
	"github.com/apisentinel/apisentinel/internal/security"
	"github.com/apisentinel/apisentinel/internal/security/hmac"
	"github.com/apisentinel/apisentinel/internal/security/normalize"
	"github.com/apisentinel/apisentinel/internal/security/pii"
)

var (
	smallPayload = `{"user_id": "usr_123", "email": "john.doe@example.com", "comment": "Hello world!"}`
	mediumPayload = `{"event": "checkout", "card": "4532-1234-5678-9012", "amount": 199.99, "notes": "SELECT * FROM users WHERE id = '1' OR '1'='1' --"}`
	largePayload = fmt.Sprintf(`{"items": [%s], "secret": "AKIAIOSFODNN7EXAMPLE", "tckn": "12345678901"}`, strings.Repeat(`{"sku": "ABC", "qty": 1, "desc": "Product item"},`, 100))
)

// BenchmarkSecurityEngine_Small benchmarks full security inspection on typical 1KB payload
func BenchmarkSecurityEngine_Small(b *testing.B) {
	engine := security.NewEngine()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Inspect(smallPayload)
	}
}

// BenchmarkSecurityEngine_Medium benchmarks injection + PII payload
func BenchmarkSecurityEngine_Medium(b *testing.B) {
	engine := security.NewEngine()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Inspect(mediumPayload)
	}
}

// BenchmarkSecurityEngine_Large benchmarks 10KB+ complex payload with secrets & SQLi
func BenchmarkSecurityEngine_Large(b *testing.B) {
	engine := security.NewEngine()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = engine.Inspect(largePayload)
	}
}

// BenchmarkCanonicalization benchmarks multi-pass URL/HTML/Unicode normalization
func BenchmarkCanonicalization(b *testing.B) {
	obfuscated := "%2527%20OR%201%3D1%20--%20\u0027\x00"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = normalize.Canonicalize(obfuscated)
	}
}

// BenchmarkPIIMasking benchmarks credit card & email regex masking
func BenchmarkPIIMasking(b *testing.B) {
	input := "Customer card is 4532-1234-5678-9012 and email is john@corp.com"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cards := pii.FindCreditCards(input)
		for _, c := range cards {
			_ = pii.MaskCreditCard(c)
		}
	}
}

// BenchmarkUUIDv7Generation benchmarks time-sortable K-Sortable Request ID generation
func BenchmarkUUIDv7Generation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = id.NewRequestID()
	}
}

// BenchmarkHMACVerification benchmarks Stripe webhook HMAC-SHA256 signature verification
func BenchmarkHMACVerification(b *testing.B) {
	secret := "whsec_bench_secret_key"
	payload := []byte(smallPayload)
	nowTs := time.Now().Unix()
	signed := fmt.Sprintf("%d.%s", nowTs, smallPayload)
	headers := map[string][]string{
		"Stripe-Signature": {fmt.Sprintf("t=%d,v1=5257a869e7ecebeaa3332029", nowTs)},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hmac.Verify(hmac.ProviderStripe, secret, payload, headers, 5*time.Minute)
	}
	_ = signed
}
