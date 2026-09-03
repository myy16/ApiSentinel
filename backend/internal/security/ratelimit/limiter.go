package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/apisentinel/apisentinel/internal/valkey"
)

type memoryBucket struct {
	count     int
	resetTime time.Time
}

type Limiter struct {
	valkeyClient *valkey.Client
	mu           sync.Mutex
	memBuckets   map[string]*memoryBucket
}

func NewLimiter(client *valkey.Client) *Limiter {
	return &Limiter{
		valkeyClient: client,
		memBuckets:   make(map[string]*memoryBucket),
	}
}

type Result struct {
	Allowed   bool          `json:"allowed"`
	Remaining int           `json:"remaining"`
	Limit     int           `json:"limit"`
	ResetTime time.Duration `json:"resetTime"`
	IsSpike   bool          `json:"isSpike"`
}

// Allow checks if the request is permitted under the rate limit window.
// Uses Valkey if available; otherwise falls back to a thread-safe in-memory sliding window.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Result, error) {
	if l == nil || limit <= 0 {
		return Result{Allowed: true, Remaining: limit, Limit: limit}, nil
	}

	if window <= 0 {
		window = time.Minute
	}

	if l.valkeyClient != nil {
		cacheKey := fmt.Sprintf("ratelimit:%s", key)
		count, ttl, err := l.valkeyClient.RateLimitIncrement(ctx, cacheKey, window)
		if err == nil {
			remaining := limit - int(count)
			if remaining < 0 {
				remaining = 0
			}
			if count > int64(limit) {
				return Result{
					Allowed:   false,
					Remaining: 0,
					Limit:     limit,
					ResetTime: ttl,
				}, nil
			}
			return Result{
				Allowed:   true,
				Remaining: remaining,
				Limit:     limit,
				ResetTime: ttl,
			}, nil
		}
		// On Valkey error, fall back to in-memory limiter
	}

	// In-Memory Limiter Fallback
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, exists := l.memBuckets[key]
	if !exists || now.After(b.resetTime) {
		b = &memoryBucket{
			count:     1,
			resetTime: now.Add(window),
		}
		l.memBuckets[key] = b
		return Result{
			Allowed:   true,
			Remaining: limit - 1,
			Limit:     limit,
			ResetTime: window,
		}, nil
	}

	b.count++
	remaining := limit - b.count
	if remaining < 0 {
		remaining = 0
	}

	ttl := b.resetTime.Sub(now)
	if ttl < 0 {
		ttl = 0
	}

	if b.count > limit {
		return Result{
			Allowed:   false,
			Remaining: 0,
			Limit:     limit,
			ResetTime: ttl,
		}, nil
	}

	return Result{
		Allowed:   true,
		Remaining: remaining,
		Limit:     limit,
		ResetTime: ttl,
	}, nil
}

// CheckSpikeAnomaly monitors sudden burst traffic over a short sub-window (e.g. 5-10 seconds).
func (l *Limiter) CheckSpikeAnomaly(ctx context.Context, endpointKey string, burstThreshold int, burstWindow time.Duration) (bool, int) {
	if l == nil || burstThreshold <= 0 {
		return false, 0
	}

	if burstWindow <= 0 {
		burstWindow = 10 * time.Second
	}

	spikeKey := fmt.Sprintf("spike:%s", endpointKey)
	res, _ := l.Allow(ctx, spikeKey, burstThreshold, burstWindow)

	// If the burst limit is exceeded, a spike anomaly is flagged
	isSpike := !res.Allowed
	return isSpike, burstThreshold - res.Remaining
}
