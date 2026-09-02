package schema

import (
	"testing"
)

func TestDetectDrift_NoDrift(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"required": ["event", "amount"],
		"properties": {
			"event": {"type": "string"},
			"amount": {"type": "integer"}
		}
	}`)

	payloadJSON := []byte(`{"event": "payment.succeeded", "amount": 100}`)

	report, err := DetectDrift(schemaJSON, payloadJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if report.HasDrift {
		t.Errorf("Expected HasDrift to be false, got true with changes: %v", report.Changes)
	}
	if report.Severity != "NONE" {
		t.Errorf("Expected severity NONE, got %s", report.Severity)
	}
}

func TestDetectDrift_FieldAdded_NonBreaking(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"required": ["event"],
		"properties": {
			"event": {"type": "string"}
		}
	}`)

	payloadJSON := []byte(`{
		"event": "user.signup",
		"new_feature_flag": true,
		"metadata": {"source": "mobile_app"}
	}`)

	report, err := DetectDrift(schemaJSON, payloadJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !report.HasDrift {
		t.Fatalf("Expected HasDrift to be true")
	}
	if report.Severity != "NON_BREAKING" {
		t.Errorf("Expected severity NON_BREAKING, got %s", report.Severity)
	}
	if len(report.Changes) != 2 {
		t.Errorf("Expected 2 added fields, got %d", len(report.Changes))
	}
}

func TestDetectDrift_MissingRequiredField_Breaking(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"required": ["event", "order_id", "currency"],
		"properties": {
			"event": {"type": "string"},
			"order_id": {"type": "string"},
			"currency": {"type": "string"}
		}
	}`)

	payloadJSON := []byte(`{
		"event": "order.completed",
		"order_id": "ord_999"
	}`)

	report, err := DetectDrift(schemaJSON, payloadJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !report.HasDrift {
		t.Fatalf("Expected HasDrift to be true")
	}
	if report.Severity != "BREAKING" {
		t.Errorf("Expected severity BREAKING, got %s", report.Severity)
	}
	if len(report.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(report.Changes))
	}
	if report.Changes[0].ChangeType != ChangeFieldMissing {
		t.Errorf("Expected ChangeFieldMissing, got %s", report.Changes[0].ChangeType)
	}
	if report.Changes[0].Path != "$.currency" {
		t.Errorf("Expected path $.currency, got %s", report.Changes[0].Path)
	}
}

func TestDetectDrift_TypeMismatch_Breaking(t *testing.T) {
	schemaJSON := []byte(`{
		"type": "object",
		"required": ["amount"],
		"properties": {
			"amount": {"type": "integer"}
		}
	}`)

	payloadJSON := []byte(`{
		"amount": "one hundred dollars"
	}`)

	report, err := DetectDrift(schemaJSON, payloadJSON)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !report.HasDrift {
		t.Fatalf("Expected HasDrift to be true")
	}
	if report.Severity != "BREAKING" {
		t.Errorf("Expected severity BREAKING, got %s", report.Severity)
	}
	if len(report.Changes) != 1 {
		t.Fatalf("Expected 1 change, got %d", len(report.Changes))
	}
	if report.Changes[0].ChangeType != ChangeTypeMismatch {
		t.Errorf("Expected ChangeTypeMismatch, got %s", report.Changes[0].ChangeType)
	}
}
