package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// IdempotencyHeaderNames contains standard HTTP headers used for idempotency and request tracing.
var IdempotencyHeaderNames = []string{
	"idempotency-key",
	"x-idempotency-key",
	"x-request-id",
	"request-id",
	"x-correlation-id",
	"stripe-request-id",
}

// IdempotencyBodyFieldNames contains JSON field names typically carrying unique idempotency or event identifiers.
var IdempotencyBodyFieldNames = []string{
	"id",
	"event_id",
	"idempotency_key",
	"idempotencyKey",
	"nonce",
	"request_id",
	"requestId",
	"tx_id",
	"transaction_id",
	"payment_id",
	"paymentId",
	"conversation_id",
	"conversationId",
}

// MutationResult contains the mutated headers, payload, and a log of replaced keys.
type MutationResult struct {
	Headers      map[string]string `json:"headers"`
	PayloadBytes []byte            `json:"payloadBytes"`
	PayloadJSON  string            `json:"payloadJson"`
	Replacements map[string]string `json:"replacements"` // "header.Idempotency-Key": "new_val", "body.event_id": "new_val"
}

// MutateIdempotencyKeys inspects headers and payload, replacing existing unique keys with fresh unique UUID tokens.
func MutateIdempotencyKeys(rawHeaders map[string]interface{}, rawPayload []byte) MutationResult {
	replacements := make(map[string]string)
	mutatedHeaders := make(map[string]string)

	// 1. Mutate Headers
	for k, v := range rawHeaders {
		strVal := ""
		switch val := v.(type) {
		case string:
			strVal = val
		case []interface{}:
			if len(val) > 0 {
				if s, ok := val[0].(string); ok {
					strVal = s
				}
			}
		}

		lowerKey := strings.ToLower(k)
		if isIdempotencyHeader(lowerKey) {
			newVal := generateFreshToken(strVal)
			mutatedHeaders[k] = newVal
			replacements["header."+k] = fmt.Sprintf("%s -> %s", strVal, newVal)
		} else if strVal != "" {
			mutatedHeaders[k] = strVal
		}
	}

	// Always ensure an Idempotency-Key header is present if missing
	hasIdemp := false
	for k := range mutatedHeaders {
		if strings.ToLower(k) == "idempotency-key" {
			hasIdemp = true
			break
		}
	}
	if !hasIdemp {
		newKey := "idemp_" + strings.ReplaceAll(uuid.New().String(), "-", "")
		mutatedHeaders["Idempotency-Key"] = newKey
		replacements["header.Idempotency-Key"] = "(yeni eklendi) -> " + newKey
	}

	// 2. Mutate JSON Payload
	var parsedBody interface{}
	if err := json.Unmarshal(rawPayload, &parsedBody); err != nil || parsedBody == nil {
		// Non-JSON or empty payload
		return MutationResult{
			Headers:      mutatedHeaders,
			PayloadBytes: rawPayload,
			PayloadJSON:  string(rawPayload),
			Replacements: replacements,
		}
	}

	mutateNode("$", parsedBody, replacements)

	mutatedBytes, err := json.Marshal(parsedBody)
	if err != nil {
		mutatedBytes = rawPayload
	}

	return MutationResult{
		Headers:      mutatedHeaders,
		PayloadBytes: mutatedBytes,
		PayloadJSON:  string(mutatedBytes),
		Replacements: replacements,
	}
}

func isIdempotencyHeader(key string) bool {
	for _, h := range IdempotencyHeaderNames {
		if key == h {
			return true
		}
	}
	return false
}

func isIdempotencyField(field string) bool {
	lower := strings.ToLower(field)
	for _, f := range IdempotencyBodyFieldNames {
		if strings.ToLower(f) == lower {
			return true
		}
	}
	return false
}

func mutateNode(currentPath string, node interface{}, replacements map[string]string) {
	switch val := node.(type) {
	case map[string]interface{}:
		for k, child := range val {
			childPath := currentPath + "." + k
			if isIdempotencyField(k) {
				if oldStr, ok := child.(string); ok && oldStr != "" {
					newStr := generateFreshToken(oldStr)
					val[k] = newStr
					replacements["body."+childPath] = fmt.Sprintf("%s -> %s", oldStr, newStr)
					continue
				}
			}
			mutateNode(childPath, child, replacements)
		}
	case []interface{}:
		for _, item := range val {
			mutateNode(currentPath+"[]", item, replacements)
		}
	}
}

func generateFreshToken(original string) string {
	freshUUID := strings.ReplaceAll(uuid.New().String(), "-", "")

	// If original had a common prefix like evt_, ch_, ord_, pi_
	if strings.Contains(original, "_") {
		parts := strings.SplitN(original, "_", 2)
		prefix := parts[0]
		if len(prefix) <= 8 {
			return fmt.Sprintf("%s_replay_%s", prefix, freshUUID[:16])
		}
	}

	return fmt.Sprintf("replay_%s", freshUUID[:20])
}
