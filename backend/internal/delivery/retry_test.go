package delivery

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestEvaluateResponse_TableDriven(t *testing.T) {
	opts := DefaultRetryOptions()
	opts.MaxRetries = 3
	opts.BaseBackoff = 50 * time.Millisecond
	opts.MaxBackoff = 5 * time.Second

	tests := []struct {
		name           string
		statusCode     int
		err            error
		attempt        int
		headers        http.Header
		overrideOpts   *RetryOptions
		expectedAction DecisionAction
		expectedState  DeliveryState
		expectedTerm   bool
	}{
		{
			name:           "200 OK -> Delivered",
			statusCode:     200,
			attempt:        1,
			expectedAction: ActionDelivered,
			expectedState:  DeliveryStateDelivered,
			expectedTerm:   true,
		},
		{
			name:           "201 Created -> Delivered",
			statusCode:     201,
			attempt:        1,
			expectedAction: ActionDelivered,
			expectedState:  DeliveryStateDelivered,
			expectedTerm:   true,
		},
		{
			name:           "500 Internal Server Error Attempt 1 -> RetryWait",
			statusCode:     500,
			attempt:        1,
			expectedAction: ActionRetryWait,
			expectedState:  DeliveryStateRetryWait,
			expectedTerm:   false,
		},
		{
			name:           "503 Service Unavailable Attempt 3 (Max) -> DeadLetter",
			statusCode:     503,
			attempt:        3,
			expectedAction: ActionDeadLetter,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
		{
			name:           "429 Too Many Requests with Retry-After 2s -> RetryWait",
			statusCode:     429,
			attempt:        1,
			headers:        http.Header{"Retry-After": []string{"2"}},
			expectedAction: ActionRetryWait,
			expectedState:  DeliveryStateRetryWait,
			expectedTerm:   false,
		},
		{
			name:           "408 Request Timeout Attempt 1 -> RetryWait",
			statusCode:     408,
			attempt:        1,
			expectedAction: ActionRetryWait,
			expectedState:  DeliveryStateRetryWait,
			expectedTerm:   false,
		},
		{
			name:           "401 Unauthorized -> Credential Incident",
			statusCode:     401,
			attempt:        1,
			expectedAction: ActionCredentialIncident,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
		{
			name:           "403 Forbidden -> Credential Incident",
			statusCode:     403,
			attempt:        1,
			expectedAction: ActionCredentialIncident,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
		{
			name:           "404 Not Found -> Config Anomaly",
			statusCode:     404,
			attempt:        1,
			expectedAction: ActionConfigAnomaly,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
		{
			name:           "400 Bad Request -> DeadLetter",
			statusCode:     400,
			attempt:        1,
			expectedAction: ActionDeadLetter,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
		{
			name:           "422 Unprocessable Entity -> DeadLetter",
			statusCode:     422,
			attempt:        1,
			expectedAction: ActionDeadLetter,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
		{
			name:           "409 Conflict with TreatConflictAsDone = false -> DeadLetter",
			statusCode:     409,
			attempt:        1,
			expectedAction: ActionDeadLetter,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
		{
			name:       "409 Conflict with TreatConflictAsDone = true -> Delivered",
			statusCode: 409,
			attempt:    1,
			overrideOpts: &RetryOptions{
				MaxRetries:          3,
				TreatConflictAsDone: true,
			},
			expectedAction: ActionDelivered,
			expectedState:  DeliveryStateDelivered,
			expectedTerm:   true,
		},
		{
			name:           "Network Timeout Error Attempt 1 -> RetryWait",
			err:            errors.New("dial tcp: i/o timeout"),
			attempt:        1,
			expectedAction: ActionRetryWait,
			expectedState:  DeliveryStateRetryWait,
			expectedTerm:   false,
		},
		{
			name:           "Network Error Attempt 3 (Max) -> DeadLetter",
			err:            errors.New("connection refused"),
			attempt:        3,
			expectedAction: ActionDeadLetter,
			expectedState:  DeliveryStateDeadLetter,
			expectedTerm:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			currentOpts := opts
			if tt.overrideOpts != nil {
				currentOpts = *tt.overrideOpts
			}

			res := EvaluateResponse(tt.statusCode, tt.err, tt.attempt, tt.headers, currentOpts)

			if res.Action != tt.expectedAction {
				t.Errorf("expected action %s, got %s", tt.expectedAction, res.Action)
			}
			if res.NextState != tt.expectedState {
				t.Errorf("expected next state %s, got %s", tt.expectedState, res.NextState)
			}
			if res.IsTerminal != tt.expectedTerm {
				t.Errorf("expected isTerminal %v, got %v", tt.expectedTerm, res.IsTerminal)
			}
		})
	}
}

func TestCalculateBackoff(t *testing.T) {
	base := 100 * time.Millisecond
	max := 10 * time.Second

	// Attempt 1: ~100ms
	d1 := CalculateBackoff(1, base, max)
	if d1 < base || d1 > 200*time.Millisecond {
		t.Errorf("unexpected attempt 1 delay: %v", d1)
	}

	// Attempt 2: ~200ms
	d2 := CalculateBackoff(2, base, max)
	if d2 < 200*time.Millisecond || d2 > 350*time.Millisecond {
		t.Errorf("unexpected attempt 2 delay: %v", d2)
	}

	// Attempt 10 should cap at max
	d10 := CalculateBackoff(10, base, max)
	if d10 > max {
		t.Errorf("backoff exceeded max: %v > %v", d10, max)
	}
}

func TestParseRetryAfter(t *testing.T) {
	base := 100 * time.Millisecond
	max := 10 * time.Second

	// Numeric seconds
	h1 := http.Header{"Retry-After": []string{"3"}}
	d1 := ParseRetryAfter(h1, 1, base, max)
	if d1 != 3*time.Second {
		t.Errorf("expected 3s from Retry-After, got %v", d1)
	}

	// Cap at max
	h2 := http.Header{"Retry-After": []string{"60"}}
	d2 := ParseRetryAfter(h2, 1, base, max)
	if d2 != max {
		t.Errorf("expected %v, got %v", max, d2)
	}

	// Empty header fallback
	h3 := http.Header{}
	d3 := ParseRetryAfter(h3, 1, base, max)
	if d3 < base {
		t.Errorf("expected at least base delay, got %v", d3)
	}
}
