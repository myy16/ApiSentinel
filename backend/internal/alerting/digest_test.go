package alerting

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeliveryDigestAccumulator_ThresholdFlush(t *testing.T) {
	var dispatchedCount int32
	var mu sync.Mutex
	var lastPayload DeliveryAlertPayload

	callback := func(ctx context.Context, p DeliveryAlertPayload) {
		atomic.AddInt32(&dispatchedCount, 1)
		mu.Lock()
		lastPayload = p
		mu.Unlock()
	}

	// Window 10s, threshold 5
	acc := NewDeliveryDigestAccumulator(10*time.Second, 5, callback)
	defer acc.Close()

	// Record 4 failures -> Should NOT trigger flush yet
	for i := 0; i < 4; i++ {
		acc.RecordFailure("proj-1", "Billing Core", "ep-1", "Stripe Inbound", "https://api.internal/hook", 503, "Service Unavailable")
	}

	time.Sleep(50 * time.Millisecond)
	if atomic.LoadInt32(&dispatchedCount) != 0 {
		t.Fatalf("Expected 0 flushes before threshold, got %d", atomic.LoadInt32(&dispatchedCount))
	}

	// Record 5th failure -> MUST trigger immediate digest flush!
	acc.RecordFailure("proj-1", "Billing Core", "ep-1", "Stripe Inbound", "https://api.internal/hook", 503, "Service Unavailable")

	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&dispatchedCount) != 1 {
		t.Fatalf("Expected exactly 1 digest flush, got %d", atomic.LoadInt32(&dispatchedCount))
	}

	mu.Lock()
	payloadCopy := lastPayload
	mu.Unlock()

	if payloadCopy.TotalFailures != 5 {
		t.Errorf("Expected 5 total failures in digest payload, got %d", payloadCopy.TotalFailures)
	}
	if payloadCopy.AlertKind != "DIGEST_SUMMARY" {
		t.Errorf("Expected AlertKind DIGEST_SUMMARY, got %s", payloadCopy.AlertKind)
	}
	if payloadCopy.StatusCode != 503 {
		t.Errorf("Expected StatusCode 503, got %d", payloadCopy.StatusCode)
	}
}

func TestDeliveryDigestAccumulator_WindowExpiryFlush(t *testing.T) {
	var dispatchedCount int32
	var mu sync.Mutex
	var lastPayload DeliveryAlertPayload

	callback := func(ctx context.Context, p DeliveryAlertPayload) {
		atomic.AddInt32(&dispatchedCount, 1)
		mu.Lock()
		lastPayload = p
		mu.Unlock()
	}

	// Window 100ms, threshold 10
	acc := NewDeliveryDigestAccumulator(100*time.Millisecond, 10, callback)
	defer acc.Close()

	// Record 3 failures (below threshold)
	for i := 0; i < 3; i++ {
		acc.RecordFailure("proj-2", "Auth Service", "ep-2", "GitHub Hook", "https://auth.internal/hook", 500, "Internal Server Error")
	}

	// Wait for window expiration
	time.Sleep(250 * time.Millisecond)

	if atomic.LoadInt32(&dispatchedCount) != 1 {
		t.Fatalf("Expected 1 expired window flush, got %d", atomic.LoadInt32(&dispatchedCount))
	}

	mu.Lock()
	payloadCopy := lastPayload
	mu.Unlock()

	if payloadCopy.TotalFailures != 3 {
		t.Errorf("Expected 3 failures in digest payload, got %d", payloadCopy.TotalFailures)
	}
}

func TestDeliveryDigestAccumulator_MultipleEndpointsIsolation(t *testing.T) {
	var ep1Dispatched, ep2Dispatched int32

	callback := func(ctx context.Context, p DeliveryAlertPayload) {
		if p.EndpointID == "ep-1" {
			atomic.AddInt32(&ep1Dispatched, 1)
		} else if p.EndpointID == "ep-2" {
			atomic.AddInt32(&ep2Dispatched, 1)
		}
	}

	acc := NewDeliveryDigestAccumulator(5*time.Second, 3, callback)
	defer acc.Close()

	// ep-1 gets 3 failures (reaches threshold)
	for i := 0; i < 3; i++ {
		acc.RecordFailure("proj-1", "Core", "ep-1", "Stripe", "https://api.internal/hook1", 502, "Bad Gateway")
	}

	// ep-2 gets only 1 failure (does not reach threshold)
	acc.RecordFailure("proj-1", "Core", "ep-2", "Shopify", "https://api.internal/hook2", 504, "Gateway Timeout")

	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&ep1Dispatched) != 1 {
		t.Errorf("Expected ep-1 to be flushed once, got %d", atomic.LoadInt32(&ep1Dispatched))
	}
	if atomic.LoadInt32(&ep2Dispatched) != 0 {
		t.Errorf("Expected ep-2 NOT to be flushed yet, got %d", atomic.LoadInt32(&ep2Dispatched))
	}

	// Manual FlushAll flushes remaining ep-2
	acc.FlushAll()
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&ep2Dispatched) != 1 {
		t.Errorf("Expected ep-2 to be flushed after FlushAll, got %d", atomic.LoadInt32(&ep2Dispatched))
	}
}
