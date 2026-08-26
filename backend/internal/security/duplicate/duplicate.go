package duplicate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

// DefaultIdempotencyTTL is the standard sliding window duration to track duplicates
const DefaultIdempotencyTTL = 60 * time.Second

// CalculatePayloadHash computes SHA-256 hash of the normalized payload body
func CalculatePayloadHash(rawBody []byte) string {
	if len(rawBody) == 0 {
		return ""
	}
	trimmed := strings.TrimSpace(string(rawBody))
	hash := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(hash[:])
}

// BuildIdempotencyKey creates the Valkey cache key for an endpoint + payload hash
func BuildIdempotencyKey(endpointID, payloadHash string) string {
	return fmt.Sprintf("idemp:%s:%s", endpointID, payloadHash)
}

// Finding represents an idempotency violation / duplicate event
type Finding struct {
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	Message        string  `json:"message"`
	EvidenceMasked string  `json:"evidence_masked"`
	Confidence     float64 `json:"confidence"`
	OriginalReqID  string  `json:"original_request_id,omitempty"`
}

// CreateDuplicateFinding creates a structured Finding for detected duplicate webhooks
func CreateDuplicateFinding(originalReqID, payloadHash string) Finding {
	evidence := payloadHash
	if len(evidence) > 16 {
		evidence = evidence[:8] + "..." + evidence[len(evidence)-8:]
	}

	msg := "Duplicate webhook payload detected within idempotency window"
	if originalReqID != "" {
		msg = fmt.Sprintf("Duplicate webhook payload detected (Original Request: %s)", originalReqID)
	}

	return Finding{
		Type:           "DUPLICATE_WEBHOOK_PAYLOAD",
		Severity:       "INFO",
		Message:        msg,
		EvidenceMasked: evidence,
		Confidence:     1.0,
		OriginalReqID:  originalReqID,
	}
}
