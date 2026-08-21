package schema

import (
	"testing"
)

func TestJSONSchemaValidator(t *testing.T) {
	testSchema := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"event": { "type": "string" },
			"amount": { "type": "integer", "minimum": 1 },
			"customer": {
				"type": "object",
				"properties": {
					"email": { "type": "string" }
				},
				"required": ["email"]
			}
		},
		"required": ["event", "amount", "customer"]
	}`

	validator, err := NewValidator(testSchema)
	if err != nil {
		t.Fatalf("Failed to compile valid schema: %v", err)
	}

	// 1. Valid payload
	validPayload := []byte(`{
		"event": "payment.success",
		"amount": 2500,
		"customer": { "email": "user@test.com" }
	}`)
	violations, err := validator.Validate(validPayload)
	if err != nil || len(violations) > 0 {
		t.Errorf("Expected valid payload to pass, got violations: %+v, err: %v", violations, err)
	}

	// 2. Invalid payload: missing "amount" and wrong type for "event"
	invalidPayload := []byte(`{
		"event": 12345,
		"customer": {}
	}`)
	violations, err = validator.Validate(invalidPayload)
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if len(violations) == 0 {
		t.Errorf("Expected violations for invalid payload, got none")
	}
}
