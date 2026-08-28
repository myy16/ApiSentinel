package valkey

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

type Client struct {
	rdb *redis.Client
}

func New(addr string) (*Client, error) {
	var opts *redis.Options

	if strings.HasPrefix(addr, "redis://") || strings.HasPrefix(addr, "rediss://") || strings.HasPrefix(addr, "valkey://") {
		parsed, err := redis.ParseURL(strings.Replace(addr, "valkey://", "redis://", 1))
		if err != nil {
			return nil, fmt.Errorf("failed to parse Valkey URL: %w", err)
		}
		opts = parsed
	} else {
		opts = &redis.Options{
			Addr: addr,
		}
	}

	// Connection Pool & Timeouts
	opts.PoolSize = 20
	opts.MinIdleConns = 5
	opts.DialTimeout = 5 * time.Second
	opts.ReadTimeout = 3 * time.Second
	opts.WriteTimeout = 3 * time.Second
	opts.MaxRetries = 3
	opts.MinRetryBackoff = 100 * time.Millisecond
	opts.MaxRetryBackoff = 500 * time.Millisecond

	rdb := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Valkey: %w", err)
	}

	log.Info().Str("addr", addr).Msg("Connected to Valkey")
	return &Client{rdb: rdb}, nil
}

func (c *Client) Close() error {
	return c.rdb.Close()
}

func (c *Client) PublishStream(ctx context.Context, stream string, values map[string]interface{}) (string, error) {
	return c.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: values,
	}).Result()
}

func (c *Client) PublishEvent(ctx context.Context, channel string, message interface{}) error {
	return c.rdb.Publish(ctx, channel, message).Err()
}

func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channels...)
}

// CheckAndSetIdempotency checks if a payload hash key exists using atomic SETNX with TTL.
// Returns (isDuplicate bool, originalRequestID string, err error)
func (c *Client) CheckAndSetIdempotency(ctx context.Context, key, requestID string, ttl time.Duration) (bool, string, error) {
	if c == nil || c.rdb == nil {
		return false, "", nil
	}

	ok, err := c.rdb.SetNX(ctx, key, requestID, ttl).Result()
	if err != nil {
		return false, "", err
	}

	if !ok {
		// Key already exists! Fetch the original request ID that created it
		origID, _ := c.rdb.Get(ctx, key).Result()
		return true, origID, nil
	}

	return false, "", nil
}

// RateLimitIncrement atomically increments counter for a rate-limit key and sets window TTL on first access.
func (c *Client) RateLimitIncrement(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error) {
	pipe := c.rdb.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	ttlCmd := pipe.TTL(ctx, key)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, 0, err
	}

	count := incrCmd.Val()
	ttl := ttlCmd.Val()

	if count == 1 || ttl < 0 {
		c.rdb.Expire(ctx, key, window)
		ttl = window
	}

	return count, ttl, nil
}


