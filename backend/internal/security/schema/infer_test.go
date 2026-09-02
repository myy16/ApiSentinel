package schema

import (
	"encoding/json"
	"testing"
)

func TestInferJSONSchema_ComplexObject(t *testing.T) {
	payload := []byte(`{
		"id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11",
		"email": "developer@apisentinel.dev",
		"amount": 1500,
		"price": 29.99,
		"is_active": true,
		"created_at": "2026-09-02T12:00:00Z",
		"website": "https://apisentinel.dev",
		"tags": ["security", "webhook", "gateway"],
		"metadata": {
			"tenant_id": "org_12345",
			"tier": "enterprise"
		}
	}`)

	schemaStr, err := InferJSONSchema(payload)
	if err != nil {
		t.Fatalf("Failed to infer schema: %v", err)
	}

	var schemaMap map[string]interface{}
	if err := json.Unmarshal([]byte(schemaStr), &schemaMap); err != nil {
		t.Fatalf("Inferred schema is invalid JSON: %v", err)
	}

	if schemaMap["type"] != "object" {
		t.Errorf("Expected root type object, got %v", schemaMap["type"])
	}

	props, ok := schemaMap["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing properties map")
	}

	// Validate inferred types and formats
	idProp := props["id"].(map[string]interface{})
	if idProp["type"] != "string" || idProp["format"] != "uuid" {
		t.Errorf("Expected id to be string with uuid format, got %v", idProp)
	}

	emailProp := props["email"].(map[string]interface{})
	if emailProp["type"] != "string" || emailProp["format"] != "email" {
		t.Errorf("Expected email to be string with email format, got %v", emailProp)
	}

	createdAtProp := props["created_at"].(map[string]interface{})
	if createdAtProp["type"] != "string" || createdAtProp["format"] != "date-time" {
		t.Errorf("Expected created_at to be string with date-time format, got %v", createdAtProp)
	}

	websiteProp := props["website"].(map[string]interface{})
	if websiteProp["type"] != "string" || websiteProp["format"] != "uri" {
		t.Errorf("Expected website to be string with uri format, got %v", websiteProp)
	}

	amountProp := props["amount"].(map[string]interface{})
	if amountProp["type"] != "integer" {
		t.Errorf("Expected amount to be integer, got %v", amountProp["type"])
	}

	priceProp := props["price"].(map[string]interface{})
	if priceProp["type"] != "number" {
		t.Errorf("Expected price to be number, got %v", priceProp["type"])
	}

	// Test validation against inferred schema
	validator, err := NewValidator(schemaStr)
	if err != nil {
		t.Fatalf("Failed to compile inferred schema with validator: %v", err)
	}

	violations, err := validator.Validate(payload)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
	if len(violations) > 0 {
		t.Errorf("Original payload should pass inferred schema, got violations: %v", violations)
	}
}
