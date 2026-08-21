package pii

import (
	"testing"
)

func TestValidateLuhn(t *testing.T) {
	// Valid Visa
	if !ValidateLuhn("4532015112830366") {
		t.Errorf("Expected valid Visa to pass Luhn check")
	}

	// Valid Mastercard (5105105105105100)
	if !ValidateLuhn("5105105105105100") {
		t.Errorf("Expected valid Mastercard to pass Luhn check")
	}

	// Invalid CC (one digit changed)
	if ValidateLuhn("4532015112830367") {
		t.Errorf("Expected invalid CC to fail Luhn check")
	}
}

func TestValidateTCKN(t *testing.T) {
	// Valid algorithmically formed TCKN
	if !ValidateTCKN("10000000146") {
		t.Errorf("Expected 10000000146 to be valid TCKN")
	}

	// Invalid TCKN (starts with 0)
	if ValidateTCKN("01234567890") {
		t.Errorf("Expected TCKN starting with 0 to fail")
	}

	// Invalid TCKN (wrong length)
	if ValidateTCKN("12345") {
		t.Errorf("Expected short TCKN to fail")
	}
}

func TestMaskEmail(t *testing.T) {
	masked := MaskEmail("ahmet.yilmaz@example.com")
	expected := "a***z@example.com"
	if masked != expected {
		t.Errorf("Expected %s, got %s", expected, masked)
	}
}

func TestMaskCreditCard(t *testing.T) {
	masked := MaskCreditCard("4532015112830366")
	expected := "************0366"
	if masked != expected {
		t.Errorf("Expected %s, got %s", expected, masked)
	}
}
