package security

import (
	"context"
	"testing"
)

// mockCustomScanner for testing dynamic registration
type mockCustomScanner struct {
	called bool
}

func (m *mockCustomScanner) Name() string     { return "mock_scanner" }
func (m *mockCustomScanner) Category() string { return "CUSTOM" }
func (m *mockCustomScanner) Scan(ctx context.Context, payload string) []Finding {
	m.called = true
	if payload == "trigger_custom" {
		return []Finding{
			{
				Category:       "CUSTOM",
				Type:           "TEST_VIOLATION",
				Severity:       "HIGH",
				Message:        "Custom test trigger found",
				EvidenceMasked: "***",
				Confidence:     1.0,
			},
		}
	}
	return nil
}

func TestEngine_DefaultScanners(t *testing.T) {
	engine := NewEngine()

	scanners := engine.GetScanners()
	if len(scanners) != 3 {
		t.Fatalf("expected 3 default scanners (PII, Secret, Injection), got %d", len(scanners))
	}

	// Test PII detection
	payloadWithEmail := `{"user": "test@example.com"}`
	findings := engine.Inspect(payloadWithEmail)
	if len(findings) == 0 {
		t.Fatalf("expected findings for email, got 0")
	}
	if findings[0].Category != "PII" {
		t.Errorf("expected category PII, got %s", findings[0].Category)
	}

	// Test Secret detection
	payloadWithSecret := `{"aws_key": "AKIAIOSFODNN7EXAMPLE"}`
	findings = engine.Inspect(payloadWithSecret)
	if len(findings) == 0 {
		t.Fatalf("expected findings for AWS key, got 0")
	}
	if findings[0].Category != "SECRET" {
		t.Errorf("expected category SECRET, got %s", findings[0].Category)
	}

	// Test Injection detection
	payloadWithSQLi := `{"query": "admin' OR 1=1 --"}`
	findings = engine.Inspect(payloadWithSQLi)
	if len(findings) == 0 {
		t.Fatalf("expected findings for SQLi, got 0")
	}
	if findings[0].Category != "INJECTION" {
		t.Errorf("expected category INJECTION, got %s", findings[0].Category)
	}
}

func TestEngine_RegisterCustomScanner(t *testing.T) {
	engine := NewEngine()
	mock := &mockCustomScanner{}
	engine.RegisterScanner(mock)

	findings := engine.Inspect("trigger_custom")
	if !mock.called {
		t.Errorf("expected mock scanner to be called")
	}
	if len(findings) == 0 {
		t.Fatalf("expected finding from custom scanner, got 0")
	}
	if findings[0].Category != "CUSTOM" {
		t.Errorf("expected CUSTOM category, got %s", findings[0].Category)
	}
}

func TestEngine_ContextCancellation(t *testing.T) {
	engine := NewEngine()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	findings := engine.InspectWithContext(ctx, "trigger_something")
	if len(findings) != 0 {
		t.Errorf("expected 0 findings when context is cancelled, got %d", len(findings))
	}
}

func TestEngine_ObfuscatedPayloadDetection(t *testing.T) {
	engine := NewEngine()

	// 1. URL encoded SQLi
	urlEncodedSQLi := `{"query": "%27%20OR%201%3D1%20--"}`
	findings := engine.Inspect(urlEncodedSQLi)
	if len(findings) == 0 {
		t.Fatalf("expected finding for URL-encoded SQLi, got none")
	}
	if findings[0].Category != "INJECTION" {
		t.Errorf("expected INJECTION category, got %s", findings[0].Category)
	}

	// 2. HTML entity encoded XSS
	htmlEncodedXSS := `&lt;script&gt;alert(1)&lt;/script&gt;`
	findings = engine.Inspect(htmlEncodedXSS)
	if len(findings) == 0 {
		t.Fatalf("expected finding for HTML-encoded XSS, got none")
	}
	if findings[0].Category != "INJECTION" {
		t.Errorf("expected INJECTION category, got %s", findings[0].Category)
	}

	// 3. Double URL encoded attack
	doubleEncodedSQLi := `{"search": "%2527%2520UNION%2520SELECT%2520null"}`
	findings = engine.Inspect(doubleEncodedSQLi)
	if len(findings) == 0 {
		t.Fatalf("expected finding for double-encoded SQLi, got none")
	}
}

