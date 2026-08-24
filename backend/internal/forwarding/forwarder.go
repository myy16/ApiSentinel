package forwarding

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
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
	Success        bool   `json:"success"`
	StatusCode     int    `json:"statusCode"`
	ResponseBody   string `json:"responseBody"`
	Attempts       int    `json:"attempts"`
	LatencyMs      int64  `json:"latencyMs"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
	SavedToDLQ     bool   `json:"savedToDlq"`
}

type Forwarder struct {
	httpClient *http.Client
}

func NewForwarder() *Forwarder {
	return &Forwarder{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// ForwardRequest forwards the clean webhook payload with exponential retry
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

	var lastErr error
	var respStatus int
	var respBodyStr string
	var totalLatency int64
	startTime := time.Now()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, cfg.TargetURL, bytes.NewReader(body))
		if err != nil {
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

		if err == nil && resp.StatusCode < 500 {
			// Success or client-side valid response (2xx, 3xx, 4xx)
			respStatus = resp.StatusCode
			defer resp.Body.Close()
			bodyBytes, _ := io.ReadAll(resp.Body)
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
			lastErr = fmt.Errorf("upstream returned server error HTTP %d", resp.StatusCode)
		}

		// Exponential Backoff: 200ms, 400ms, 800ms...
		if attempt < maxRetries {
			backoff := time.Duration(math.Pow(2, float64(attempt))*100) * time.Millisecond
			time.Sleep(backoff)
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
