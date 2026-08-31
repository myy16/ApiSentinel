package alerting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatcher_Formatters(t *testing.T) {
	payload := AlertPayload{
		EventID:        "evt_123",
		ProjectName:    "Test Project",
		EndpointName:   "Test Endpoint",
		Category:       "SECRET",
		FindingType:    "AWS_KEY",
		Severity:       "CRITICAL",
		PolicyAction:   "BLOCK",
		RequestID:      "req_456",
		EvidenceMasked: "AKIA****1234",
		Message:        "AWS Access Key detected",
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	// Test Slack block format
	slackBody, err := formatSlackBlock(payload)
	if err != nil {
		t.Fatalf("formatSlackBlock failed: %v", err)
	}
	var slackMap map[string]interface{}
	if err := json.Unmarshal(slackBody, &slackMap); err != nil {
		t.Fatalf("Slack body is not valid JSON: %v", err)
	}

	// Test Discord embed format
	discordBody, err := formatDiscordEmbed(payload)
	if err != nil {
		t.Fatalf("formatDiscordEmbed failed: %v", err)
	}
	var discordMap map[string]interface{}
	if err := json.Unmarshal(discordBody, &discordMap); err != nil {
		t.Fatalf("Discord body is not valid JSON: %v", err)
	}

	// Test Telegram message format
	telegramBody, err := formatTelegramMessage(payload)
	if err != nil {
		t.Fatalf("formatTelegramMessage failed: %v", err)
	}
	var telegramMap map[string]string
	if err := json.Unmarshal(telegramBody, &telegramMap); err != nil {
		t.Fatalf("Telegram body is not valid JSON: %v", err)
	}
}

func TestDispatcher_RetryOn5xx(t *testing.T) {
	var attempts int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	d := NewDispatcherWithClient(srv.Client())
	payload := AlertPayload{
		EventID:      "evt_retry",
		ProjectName:  "Retry Project",
		Severity:     "HIGH",
		FindingType:  "PII_TCKN",
		PolicyAction: "BLOCK",
		Message:      "TCKN detected",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := d.Dispatch(ctx, ChannelWebhook, srv.URL, payload)
	if err != nil {
		t.Fatalf("expected successful dispatch after retries, got error: %v", err)
	}

	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", atomic.LoadInt32(&attempts))
	}
}
