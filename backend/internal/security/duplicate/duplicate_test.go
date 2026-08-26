package duplicate

import (
	"testing"
)

func TestCalculatePayloadHash(t *testing.T) {
	body1 := []byte(`{"event": "charge.succeeded", "amount": 5000}`)
	body2 := []byte(`{"event": "charge.succeeded", "amount": 5000}`)
	body3 := []byte(`{"event": "charge.failed", "amount": 5000}`)

	hash1 := CalculatePayloadHash(body1)
	hash2 := CalculatePayloadHash(body2)
	hash3 := CalculatePayloadHash(body3)

	if hash1 == "" {
		t.Fatalf("expected non-empty hash")
	}

	if hash1 != hash2 {
		t.Errorf("identical payloads should produce identical hashes: %s != %s", hash1, hash2)
	}

	if hash1 == hash3 {
		t.Errorf("different payloads should produce different hashes: %s == %s", hash1, hash3)
	}

	emptyHash := CalculatePayloadHash([]byte{})
	if emptyHash != "" {
		t.Errorf("empty payload should return empty string, got %s", emptyHash)
	}
}

func TestBuildIdempotencyKey(t *testing.T) {
	key := BuildIdempotencyKey("ep_123", "abc456hash")
	expected := "idemp:ep_123:abc456hash"
	if key != expected {
		t.Errorf("expected %s, got %s", expected, key)
	}
}

func TestCreateDuplicateFinding(t *testing.T) {
	finding := CreateDuplicateFinding("req_original_123", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")

	if finding.Type != "DUPLICATE_WEBHOOK_PAYLOAD" {
		t.Errorf("expected type DUPLICATE_WEBHOOK_PAYLOAD, got %s", finding.Type)
	}

	if finding.OriginalReqID != "req_original_123" {
		t.Errorf("expected original req ID req_original_123, got %s", finding.OriginalReqID)
	}

	if finding.Confidence != 1.0 {
		t.Errorf("expected confidence 1.0, got %f", finding.Confidence)
	}
}
