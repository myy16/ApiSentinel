package forwarding

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/apisentinel/apisentinel/internal/security/ssrf"
	"github.com/rs/zerolog/log"
)

type Config struct {
	EndpointID string            `json:"endpointId"`
	TargetURL  string            `json:"targetUrl"`
	MaxRetries int               `json:"maxRetries"`
	TimeoutMs  int               `json:"timeoutMs"`
	Headers    map[string]string `json:"headers"`
	Enabled    bool              `json:"enabled"`
}

type DeliveryResult struct {
	Success      bool   `json:"success"`
	StatusCode   int    `json:"statusCode"`
	ResponseBody string `json:"responseBody"`
	Attempts     int    `json:"attempts"`
	LatencyMs    int64  `json:"latencyMs"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	SavedToDLQ   bool   `json:"savedToDlq"`
}

type Forwarder struct {
	httpClient *http.Client
}

func NewForwarder() *Forwarder {
	return &Forwarder{
		httpClient: ssrf.NewSafeHTTPClient(10 * time.Second),
	}
}

// ForwardRequest forwards the clean webhook payload with exponential retry + jitter
func (f *Forwarder) ForwardRequest(ctx context.Context, cfg Config, method string, headers map[string]string, body []byte) (*DeliveryResult, error) {
	// 1. SSRF Validation
	_, err := ssrf.ValidateURL(cfg.TargetURL)
	if err != nil {
		return &DeliveryResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("SSRF Guard blocked upstream URL: %v", err),
			SavedToDLQ:   true,
		}, err
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}

	// Use per-request timeout from config if specified
	timeoutMs := cfg.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 10000 // default 10s
	}

	var lastErr error
	var respStatus int
	var respBodyStr string
	var totalLatency int64
	startTime := time.Now()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Create per-attempt context with config-driven timeout
		attemptCtx, attemptCancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)

		req, err := http.NewRequestWithContext(attemptCtx, method, cfg.TargetURL, bytes.NewReader(body))
		if err != nil {
			attemptCancel()
			return nil, err
		}

		// Set headers
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		for k, v := range cfg.Headers {
			req.Header.Set(k, v)
		}
		req.Header.Set("X-ApiSentinel-Forwarded", "true")
		req.Header.Set("X-ApiSentinel-Attempt", fmt.Sprintf("%d", attempt))

		attemptStart := time.Now()
		resp, err := f.httpClient.Do(req)
		latency := time.Since(attemptStart).Milliseconds()
		totalLatency += latency
		attemptCancel()

		if err == nil && resp.StatusCode < 500 {
			// Success or client-side valid response (2xx, 3xx, 4xx)
			respStatus = resp.StatusCode
			bodyBytes, _ := io.ReadAll(resp.Body)
			resp.Body.Close() // Close immediately instead of defer inside loop
			respBodyStr = string(bodyBytes)

			log.Info().
				Str("targetUrl", cfg.TargetURL).
				Int("status", respStatus).
				Int("attempt", attempt).
				Msg("Webhook successfully forwarded upstream")

			return &DeliveryResult{
				Success:      respStatus >= 200 && respStatus < 300,
				StatusCode:   respStatus,
				ResponseBody: respBodyStr,
				Attempts:     attempt,
				LatencyMs:    totalLatency,
				SavedToDLQ:   false,
			}, nil
		}

		if err != nil {
			lastErr = err
			log.Warn().Err(err).Int("attempt", attempt).Str("targetUrl", cfg.TargetURL).Msg("Upstream delivery failed, retrying...")
		} else {
			respStatus = resp.StatusCode
			resp.Body.Close() // Close error response body immediately
			lastErr = fmt.Errorf("upstream returned server error HTTP %d", resp.StatusCode)
		}

		// Exponential Backoff with Jitter: base * 2^attempt * (1 + random 0-30%)
		if attempt < maxRetries {
			baseBackoff := math.Pow(2, float64(attempt)) * 100
			jitter := baseBackoff * 0.3 * rand.Float64()
			backoff := time.Duration(baseBackoff+jitter) * time.Millisecond

			// Context-aware sleep: abort backoff if context is cancelled
			select {
			case <-time.After(backoff):
				// continue to next retry
			case <-ctx.Done():
				return &DeliveryResult{
					Success:      false,
					StatusCode:   respStatus,
					Attempts:     attempt,
					LatencyMs:    time.Since(startTime).Milliseconds(),
					ErrorMessage: "context cancelled during retry backoff",
					SavedToDLQ:   true,
				}, ctx.Err()
			}
		}
	}

	errMsg := ""
	if lastErr != nil {
		errMsg = lastErr.Error()
	}

	log.Error().
		Str("targetUrl", cfg.TargetURL).
		Int("attempts", maxRetries).
		Msg("Upstream delivery failed all retries, recording in Dead Letter Queue (DLQ)")

	return &DeliveryResult{
		Success:      false,
		StatusCode:   respStatus,
		ResponseBody: respBodyStr,
		Attempts:     maxRetries,
		LatencyMs:    time.Since(startTime).Milliseconds(),
		ErrorMessage: errMsg,
		SavedToDLQ:   true,
	}, lastErr
}
