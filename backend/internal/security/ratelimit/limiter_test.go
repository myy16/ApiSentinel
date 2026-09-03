package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestLimiter_InMemoryRateLimit(t *testing.T) {
	limiter := NewLimiter(nil)
	ctx := context.Background()

	key := "test-ip-123"
	limit := 3
	window := 100 * time.Millisecond

	// 1st request -> allowed
	res, err := limiter.Allow(ctx, key, limit, window)
	if err != nil || !res.Allowed || res.Remaining != 2 {
		t.Fatalf("Expected 1st request allowed with 2 remaining, got allowed=%v, rem=%d", res.Allowed, res.Remaining)
	}

	// 2nd request -> allowed
	res, _ = limiter.Allow(ctx, key, limit, window)
	if !res.Allowed || res.Remaining != 1 {
		t.Fatalf("Expected 2nd request allowed with 1 remaining, got allowed=%v, rem=%d", res.Allowed, res.Remaining)
	}

	// 3rd request -> allowed
	res, _ = limiter.Allow(ctx, key, limit, window)
	if !res.Allowed || res.Remaining != 0 {
		t.Fatalf("Expected 3rd request allowed with 0 remaining, got allowed=%v, rem=%d", res.Allowed, res.Remaining)
	}

	// 4th request -> blocked (429)
	res, _ = limiter.Allow(ctx, key, limit, window)
	if res.Allowed {
		t.Fatalf("Expected 4th request blocked, but was allowed")
	}

	// Wait for window reset
	time.Sleep(120 * time.Millisecond)

	// 5th request -> allowed again
	res, _ = limiter.Allow(ctx, key, limit, window)
	if !res.Allowed {
		t.Fatalf("Expected request allowed after window expiry, but was blocked")
	}
}

func TestLimiter_SpikeAnomalyDetection(t *testing.T) {
	limiter := NewLimiter(nil)
	ctx := context.Background()

	endpointKey := "ep_spike_test"
	burstLimit := 2
	burstWindow := 100 * time.Millisecond

	// 1st and 2nd burst
	isSpike, _ := limiter.CheckSpikeAnomaly(ctx, endpointKey, burstLimit, burstWindow)
	if isSpike {
		t.Fatalf("Expected no spike on 1st request")
	}

	isSpike, _ = limiter.CheckSpikeAnomaly(ctx, endpointKey, burstLimit, burstWindow)
	if isSpike {
		t.Fatalf("Expected no spike on 2nd request")
	}

	// 3rd burst -> spike detected!
	isSpike, _ = limiter.CheckSpikeAnomaly(ctx, endpointKey, burstLimit, burstWindow)
	if !isSpike {
		t.Fatalf("Expected spike anomaly detected on 3rd burst request")
	}
}
