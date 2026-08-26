package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/apisentinel/apisentinel/internal/valkey"
)

type Limiter struct {
	valkeyClient *valkey.Client
}

func NewLimiter(client *valkey.Client) *Limiter {
	return &Limiter{valkeyClient: client}
}

type Result struct {
	Allowed   bool
	Remaining int
	Limit     int
	ResetTime time.Duration
}

// Allow checks if the request is permitted under the rate limit window.
// If limit exceeded, returns Allowed: false with remaining: 0.
func (l *Limiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (Result, error) {
	if l == nil || l.valkeyClient == nil || limit <= 0 {
		return Result{Allowed: true, Remaining: limit, Limit: limit}, nil
	}

	if window <= 0 {
		window = time.Minute
	}

	cacheKey := fmt.Sprintf("ratelimit:%s", key)

	count, ttl, err := l.valkeyClient.RateLimitIncrement(ctx, cacheKey, window)
	if err != nil {
		// Fail open on cache error to avoid blocking legitimate traffic
		return Result{Allowed: true, Remaining: limit, Limit: limit}, nil
	}

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
