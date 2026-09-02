package schema

import (
	"testing"
)

func TestExtractSchemaFromOpenAPI_JSONSpec(t *testing.T) {
	openAPISpec := []byte(`{
		"openapi": "3.1.0",
		"info": {
			"title": "Payment Webhook API",
			"version": "1.0.0"
		},
		"webhooks": {
			"newPayment": {
				"post": {
					"requestBody": {
						"required": true,
						"content": {
							"application/json": {
								"schema": {
									"type": "object",
									"required": ["event_id", "amount"],
									"properties": {
										"event_id": {"type": "string"},
										"amount": {"type": "number"}
									}
								}
							}
						}
					}
				}
			}
		}
	}`)

	schemaStr, err := ExtractSchemaFromOpenAPI(openAPISpec, "")
	if err != nil {
		t.Fatalf("Failed to extract schema from OpenAPI 3.1: %v", err)
	}

	validator, err := NewValidator(schemaStr)
	if err != nil {
		t.Fatalf("Failed to compile extracted schema: %v", err)
	}

	validPayload := []byte(`{"event_id":"evt_123","amount":99.5}`)
	violations, err := validator.Validate(validPayload)
	if err != nil || len(violations) > 0 {
		t.Errorf("Expected valid payload to pass, got: %v (violations: %v)", err, violations)
	}

	invalidPayload := []byte(`{"amount":99.5}`)
	violations, err = validator.Validate(invalidPayload)
	if len(violations) == 0 {
		t.Errorf("Expected violations for missing required field event_id")
	}
}

func TestExtractSchemaFromOpenAPI_YAMLSpec(t *testing.T) {
	yamlSpec := []byte(`
openapi: 3.0.3
info:
  title: GitHub Webhook Spec
  version: 1.0.0
paths:
  /webhooks/github:
    post:
      summary: GitHub Push Event
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/PushPayload'
components:
  schemas:
    PushPayload:
      type: object
      required:
        - ref
        - repository
      properties:
        ref:
          type: string
        repository:
          type: object
          properties:
            name:
              type: string
`)

	schemaStr, err := ExtractSchemaFromOpenAPI(yamlSpec, "/webhooks/github")
	if err != nil {
		t.Fatalf("Failed to extract schema from YAML OpenAPI: %v", err)
	}

	validator, err := NewValidator(schemaStr)
	if err != nil {
		t.Fatalf("Failed to compile extracted schema: %v", err)
	}

	validPayload := []byte(`{"ref":"refs/heads/main","repository":{"name":"apisentinel"}}`)
	violations, err := validator.Validate(validPayload)
	if err != nil || len(violations) > 0 {
		t.Errorf("Expected valid payload to pass, got: %v (violations: %v)", err, violations)
	}
}
