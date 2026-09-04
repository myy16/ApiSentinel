package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMutateIdempotencyKeys_HeadersAndBody(t *testing.T) {
	headers := map[string]interface{}{
		"Idempotency-Key": "idemp_original_12345",
		"X-Request-Id":    "req_original_67890",
		"Content-Type":    "application/json",
	}

	payload := []byte(`{
		"id": "evt_123456789",
		"event_id": "evt_abc_999",
		"data": {
			"payment_id": "pi_3Mtw",
			"customer": "cus_123",
			"amount": 5000
		}
	}`)

	result := MutateIdempotencyKeys(headers, payload)

	// 1. Headers should be mutated
	if result.Headers["Idempotency-Key"] == "idemp_original_12345" {
		t.Errorf("Idempotency-Key header was not mutated")
	}
	if !strings.HasPrefix(result.Headers["Idempotency-Key"], "idemp_replay_") {
		t.Errorf("Expected Idempotency-Key to preserve prefix, got: %s", result.Headers["Idempotency-Key"])
	}
	if result.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type should remain unchanged, got: %s", result.Headers["Content-Type"])
	}

	// 2. Payload body: idempotency/event fields mutated, business identifiers preserved
	var parsed map[string]interface{}
	if err := json.Unmarshal(result.PayloadBytes, &parsed); err != nil {
		t.Fatalf("Failed to parse mutated payload JSON: %v", err)
	}

	// Business entity "id" should be PRESERVED
	if parsed["id"] != "evt_123456789" {
		t.Errorf("Business ID should be preserved, got: %v", parsed["id"])
	}
	// Event/idempotency key should be MUTATED
	if parsed["event_id"] == "evt_abc_999" {
		t.Errorf("event_id was not mutated")
	}

	dataMap, ok := parsed["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("data object missing in mutated payload")
	}
	// Business entity "payment_id" should be PRESERVED
	if dataMap["payment_id"] != "pi_3Mtw" {
		t.Errorf("payment_id should be preserved as business data, got: %v", dataMap["payment_id"])
	}
	if dataMap["customer"] != "cus_123" {
		t.Errorf("customer field should not be mutated, got: %v", dataMap["customer"])
	}
	if dataMap["amount"] != float64(5000) {
		t.Errorf("amount field should remain 5000, got: %v", dataMap["amount"])
	}

	// 3. Replacements map check (headers + body event_id)
	if len(result.Replacements) < 3 {
		t.Errorf("Expected at least 3 replacements recorded, got %d: %v", len(result.Replacements), result.Replacements)
	}
}
