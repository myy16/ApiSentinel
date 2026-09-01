package delivery

import (
	"testing"
)

func TestRequestState_Validation(t *testing.T) {
	validStates := []RequestState{
		RequestStateReceived,
		RequestStateVerified,
		RequestStateAccepted,
		RequestStateRejectedSignature,
		RequestStateBlockedPolicy,
	}

	for _, s := range validStates {
		if !s.IsValid() {
			t.Errorf("expected state %s to be valid", s)
		}
	}

	invalid := RequestState("INVALID_STATE")
	if invalid.IsValid() {
		t.Errorf("expected invalid state to return false")
	}

	if !RequestStateAccepted.IsTerminal() || !RequestStateRejectedSignature.IsTerminal() || !RequestStateBlockedPolicy.IsTerminal() {
		t.Errorf("expected terminal request states to return true")
	}

	if RequestStateReceived.IsTerminal() || RequestStateVerified.IsTerminal() {
		t.Errorf("expected non-terminal request states to return false")
	}
}

func TestDeliveryState_Transitions(t *testing.T) {
	tests := []struct {
		from     DeliveryState
		to       DeliveryState
		expected bool
	}{
		{DeliveryStatePending, DeliveryStateProcessing, true},
		{DeliveryStatePending, DeliveryStateDeadLetter, true},
		{DeliveryStatePending, DeliveryStateDelivered, false},

		{DeliveryStateProcessing, DeliveryStateDelivered, true},
		{DeliveryStateProcessing, DeliveryStateRetryWait, true},
		{DeliveryStateProcessing, DeliveryStateDeadLetter, true},
		{DeliveryStateProcessing, DeliveryStatePending, false},

		{DeliveryStateRetryWait, DeliveryStateProcessing, true},
		{DeliveryStateRetryWait, DeliveryStateDeadLetter, true},
		{DeliveryStateRetryWait, DeliveryStateDelivered, false},

		{DeliveryStateDelivered, DeliveryStatePending, true}, // Safe replay
		{DeliveryStateDelivered, DeliveryStateProcessing, false},

		{DeliveryStateDeadLetter, DeliveryStatePending, true}, // Manual re-queue
		{DeliveryStateDeadLetter, DeliveryStateDelivered, false},

		{DeliveryStateNotConfigured, DeliveryStatePending, true},

		// Same state is always allowed
		{DeliveryStateProcessing, DeliveryStateProcessing, true},
	}

	for _, tt := range tests {
		actual := CanTransition(tt.from, tt.to)
		if actual != tt.expected {
			t.Errorf("CanTransition(%s -> %s) = %v; want %v", tt.from, tt.to, actual, tt.expected)
		}

		err := ValidateTransition(tt.from, tt.to)
		if tt.expected && err != nil {
			t.Errorf("ValidateTransition(%s -> %s) returned error: %v", tt.from, tt.to, err)
		}
		if !tt.expected && err == nil {
			t.Errorf("ValidateTransition(%s -> %s) expected error but got nil", tt.from, tt.to)
		}
	}
}
