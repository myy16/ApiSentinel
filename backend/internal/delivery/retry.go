package delivery

import (
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DecisionAction represents the determined action for a delivery attempt.
type DecisionAction string

const (
	ActionDelivered          DecisionAction = "DELIVERED"
	ActionRetryWait          DecisionAction = "RETRY_WAIT"
	ActionDeadLetter         DecisionAction = "DEAD_LETTER"
	ActionCredentialIncident DecisionAction = "CREDENTIAL_INCIDENT"
	ActionConfigAnomaly      DecisionAction = "CONFIG_ANOMALY"
)

// RetryOptions allows customization of retry behavior per endpoint.
type RetryOptions struct {
	MaxRetries          int           // Default: 3
	BaseBackoff         time.Duration // Default: 100ms
	MaxBackoff          time.Duration // Default: 30s
	TreatConflictAsDone bool          // If true, 409 Conflict is treated as Delivered (Idempotent success)
}

// DefaultRetryOptions provides production-grade retry defaults.
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries:          3,
		BaseBackoff:         100 * time.Millisecond,
		MaxBackoff:          30 * time.Second,
		TreatConflictAsDone: false,
	}
}

// EvaluationResult contains the detailed decision outcome of an upstream response.
type EvaluationResult struct {
	Action        DecisionAction `json:"action"`
	NextState     DeliveryState  `json:"nextState"`
	BackoffDelay  time.Duration  `json:"backoffDelay"`
	IsTerminal    bool           `json:"isTerminal"`
	IncidentType  string         `json:"incidentType,omitempty"`
	ReasonSummary string         `json:"reasonSummary"`
}

// EvaluateResponse analyzes an HTTP status code or network error and produces a deterministic retry decision.
func EvaluateResponse(statusCode int, err error, currentAttempt int, headers http.Header, opts RetryOptions) EvaluationResult {
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	if opts.BaseBackoff <= 0 {
		opts.BaseBackoff = 100 * time.Millisecond
	}
	if opts.MaxBackoff <= 0 {
		opts.MaxBackoff = 30 * time.Second
	}

	// 1. Network / DNS / Timeout Errors
	if err != nil {
		if currentAttempt >= opts.MaxRetries {
			return EvaluationResult{
				Action:        ActionDeadLetter,
				NextState:     DeliveryStateDeadLetter,
				IsTerminal:    true,
				ReasonSummary: "network error and max retries exhausted: " + err.Error(),
			}
		}
		delay := CalculateBackoff(currentAttempt, opts.BaseBackoff, opts.MaxBackoff)
		return EvaluationResult{
			Action:        ActionRetryWait,
			NextState:     DeliveryStateRetryWait,
			BackoffDelay:  delay,
			IsTerminal:    false,
			ReasonSummary: "network error, scheduled for retry: " + err.Error(),
		}
	}

	// 2. Successful 2xx responses
	if statusCode >= 200 && statusCode < 300 {
		return EvaluationResult{
			Action:        ActionDelivered,
			NextState:     DeliveryStateDelivered,
			IsTerminal:    true,
			ReasonSummary: "upstream responded with successful status " + strconv.Itoa(statusCode),
		}
	}

	// 3. 409 Conflict (Optional Idempotency Success)
	if statusCode == http.StatusConflict && opts.TreatConflictAsDone {
		return EvaluationResult{
			Action:        ActionDelivered,
			NextState:     DeliveryStateDelivered,
			IsTerminal:    true,
			ReasonSummary: "upstream returned 409 Conflict, treated as idempotent delivery",
		}
	}

	// 4. Rate Limiting (429 Too Many Requests)
	if statusCode == http.StatusTooManyRequests {
		if currentAttempt >= opts.MaxRetries {
			return EvaluationResult{
				Action:        ActionDeadLetter,
				NextState:     DeliveryStateDeadLetter,
				IsTerminal:    true,
				ReasonSummary: "upstream rate limit 429 and max retries exhausted",
			}
		}

		delay := ParseRetryAfter(headers, currentAttempt, opts.BaseBackoff, opts.MaxBackoff)
		return EvaluationResult{
			Action:        ActionRetryWait,
			NextState:     DeliveryStateRetryWait,
			BackoffDelay:  delay,
			IsTerminal:    false,
			ReasonSummary: "upstream rate limited (429), waiting for retry",
		}
	}

	// 5. Retryable 4xx errors: 408 (Request Timeout), 425 (Too Early)
	if statusCode == http.StatusRequestTimeout || statusCode == 425 {
		if currentAttempt >= opts.MaxRetries {
			return EvaluationResult{
				Action:        ActionDeadLetter,
				NextState:     DeliveryStateDeadLetter,
				IsTerminal:    true,
				ReasonSummary: "transient 4xx error (" + strconv.Itoa(statusCode) + ") and max retries exhausted",
			}
		}
		delay := CalculateBackoff(currentAttempt, opts.BaseBackoff, opts.MaxBackoff)
		return EvaluationResult{
			Action:        ActionRetryWait,
			NextState:     DeliveryStateRetryWait,
			BackoffDelay:  delay,
			IsTerminal:    false,
			ReasonSummary: "transient upstream error (" + strconv.Itoa(statusCode) + "), retrying",
		}
	}

	// 6. Non-retryable Authentication/Authorization errors: 401, 403
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return EvaluationResult{
			Action:        ActionCredentialIncident,
			NextState:     DeliveryStateDeadLetter,
			IsTerminal:    true,
			IncidentType:  "CREDENTIAL_OR_PERMISSION_FAILURE",
			ReasonSummary: "upstream rejected credentials (HTTP " + strconv.Itoa(statusCode) + ")",
		}
	}

	// 7. Non-retryable Not Found: 404
	if statusCode == http.StatusNotFound {
		return EvaluationResult{
			Action:        ActionConfigAnomaly,
			NextState:     DeliveryStateDeadLetter,
			IsTerminal:    true,
			IncidentType:  "UPSTREAM_URL_NOT_FOUND",
			ReasonSummary: "upstream endpoint not found (HTTP 404); check configured target URL",
		}
	}

	// 8. Other 4xx Client errors: 400 Bad Request, 422 Unprocessable Entity, etc.
	if statusCode >= 400 && statusCode < 500 {
		return EvaluationResult{
			Action:        ActionDeadLetter,
			NextState:     DeliveryStateDeadLetter,
			IsTerminal:    true,
			ReasonSummary: "upstream rejected request with non-retryable client error HTTP " + strconv.Itoa(statusCode),
		}
	}

	// 9. Server 5xx errors: 500, 502, 503, 504
	if statusCode >= 500 {
		if currentAttempt >= opts.MaxRetries {
			return EvaluationResult{
				Action:        ActionDeadLetter,
				NextState:     DeliveryStateDeadLetter,
				IsTerminal:    true,
				ReasonSummary: "upstream server error HTTP " + strconv.Itoa(statusCode) + " and max retries exhausted",
			}
		}
		delay := CalculateBackoff(currentAttempt, opts.BaseBackoff, opts.MaxBackoff)
		return EvaluationResult{
			Action:        ActionRetryWait,
			NextState:     DeliveryStateRetryWait,
			BackoffDelay:  delay,
			IsTerminal:    false,
			ReasonSummary: "upstream server error HTTP " + strconv.Itoa(statusCode) + ", retrying",
		}
	}

	// Fallback for unexpected status codes
	return EvaluationResult{
		Action:        ActionDeadLetter,
		NextState:     DeliveryStateDeadLetter,
		IsTerminal:    true,
		ReasonSummary: "unexpected status code HTTP " + strconv.Itoa(statusCode),
	}
}

// CalculateBackoff computes exponential backoff with full jitter:
// delay = min(maxBackoff, base * 2^attempt) + jitter
func CalculateBackoff(attempt int, base, max time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	multiplier := math.Pow(2, float64(attempt-1))
	backoff := time.Duration(float64(base) * multiplier)
	if backoff > max {
		backoff = max
	}
	// Full Jitter: 0 to 25% random addition
	jitter := time.Duration(rand.Float64() * 0.25 * float64(backoff))
	total := backoff + jitter
	if total > max {
		return max
	}
	return total
}

// ParseRetryAfter inspects the Retry-After header (seconds or HTTP-Date) or falls back to backoff.
func ParseRetryAfter(headers http.Header, attempt int, base, max time.Duration) time.Duration {
	if headers == nil {
		return CalculateBackoff(attempt, base, max)
	}

	val := strings.TrimSpace(headers.Get("Retry-After"))
	if val == "" {
		return CalculateBackoff(attempt, base, max)
	}

	// Check if integer seconds
	if seconds, err := strconv.Atoi(val); err == nil && seconds > 0 {
		dur := time.Duration(seconds) * time.Second
		if dur > max {
			return max
		}
		return dur
	}

	// Check if HTTP-Date format (RFC1123 / RFC850 / ANSIC)
	if targetTime, err := http.ParseTime(val); err == nil {
		diff := time.Until(targetTime)
		if diff > 0 {
			if diff > max {
				return max
			}
			return diff
		}
	}

	return CalculateBackoff(attempt, base, max)
}
