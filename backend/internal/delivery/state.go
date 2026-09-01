package delivery

import "fmt"

// RequestState represents the ingestion and security inspection lifecycle of an incoming webhook.
type RequestState string

const (
	// RequestStateReceived means the raw HTTP payload was ingested into the gateway.
	RequestStateReceived RequestState = "RECEIVED"
	// RequestStateVerified means HMAC / signature verification succeeded.
	RequestStateVerified RequestState = "VERIFIED"
	// RequestStateAccepted means all security and policy checks passed; webhook is accepted for processing.
	RequestStateAccepted RequestState = "ACCEPTED"
	// RequestStateRejectedSignature means HMAC signature was invalid, expired, or missing.
	RequestStateRejectedSignature RequestState = "REJECTED_SIGNATURE"
	// RequestStateBlockedPolicy means the request was blocked due to SQLi/XSS/SSRF or strict schema violation.
	RequestStateBlockedPolicy RequestState = "BLOCKED_POLICY"
)

// IsValid checks if the request state is a recognized enum value.
func (s RequestState) IsValid() bool {
	switch s {
	case RequestStateReceived, RequestStateVerified, RequestStateAccepted,
		RequestStateRejectedSignature, RequestStateBlockedPolicy:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the request state represents a final decision.
func (s RequestState) IsTerminal() bool {
	return s == RequestStateAccepted || s == RequestStateRejectedSignature || s == RequestStateBlockedPolicy
}

// DeliveryState represents the upstream forwarding lifecycle of an accepted webhook.
type DeliveryState string

const (
	// DeliveryStateNotConfigured means the endpoint has no active upstream URL configured.
	DeliveryStateNotConfigured DeliveryState = "NOT_CONFIGURED"
	// DeliveryStatePending means the delivery job is queued and waiting for worker pickup.
	DeliveryStatePending DeliveryState = "PENDING"
	// DeliveryStateProcessing means a worker has acquired a lease lock and is currently executing delivery.
	DeliveryStateProcessing DeliveryState = "PROCESSING"
	// DeliveryStateRetryWait means delivery failed with a retryable error and is waiting for backoff timer.
	DeliveryStateRetryWait DeliveryState = "RETRY_WAIT"
	// DeliveryStateDelivered means upstream responded with a successful HTTP 2xx status code.
	DeliveryStateDelivered DeliveryState = "DELIVERED"
	// DeliveryStateDeadLetter means delivery failed permanently (exhausted retries or non-retryable 4xx).
	DeliveryStateDeadLetter DeliveryState = "DEAD_LETTER"
)

// IsValid checks if the delivery state is a recognized enum value.
func (s DeliveryState) IsValid() bool {
	switch s {
	case DeliveryStateNotConfigured, DeliveryStatePending, DeliveryStateProcessing,
		DeliveryStateRetryWait, DeliveryStateDelivered, DeliveryStateDeadLetter:
		return true
	default:
		return false
	}
}

// IsTerminal returns true if the delivery reached an end state (delivered, permanently failed, or unconfigured).
func (s DeliveryState) IsTerminal() bool {
	return s == DeliveryStateDelivered || s == DeliveryStateDeadLetter || s == DeliveryStateNotConfigured
}

// CanTransition validates if moving from one DeliveryState to another is legally permitted.
func CanTransition(from, to DeliveryState) bool {
	if from == to {
		return true
	}
	switch from {
	case DeliveryStatePending:
		return to == DeliveryStateProcessing || to == DeliveryStateDeadLetter
	case DeliveryStateProcessing:
		return to == DeliveryStateDelivered || to == DeliveryStateRetryWait || to == DeliveryStateDeadLetter
	case DeliveryStateRetryWait:
		return to == DeliveryStateProcessing || to == DeliveryStateDeadLetter
	case DeliveryStateDelivered:
		// Terminal, cannot transition unless a safe manual replay resets it to PENDING
		return to == DeliveryStatePending
	case DeliveryStateDeadLetter:
		// Terminal, can be re-queued by manual retry/replay to PENDING
		return to == DeliveryStatePending
	case DeliveryStateNotConfigured:
		return to == DeliveryStatePending
	default:
		return false
	}
}

// ValidateTransition returns an error if a transition is invalid.
func ValidateTransition(from, to DeliveryState) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("illegal delivery state transition from %s to %s", from, to)
	}
	return nil
}
