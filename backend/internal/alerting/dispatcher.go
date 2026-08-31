package alerting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/apisentinel/apisentinel/internal/security/ssrf"
	"github.com/rs/zerolog/log"
)

type ChannelType string

const (
	ChannelSlack    ChannelType = "SLACK"
	ChannelDiscord  ChannelType = "DISCORD"
	ChannelTelegram ChannelType = "TELEGRAM"
	ChannelWebhook  ChannelType = "WEBHOOK"
)

type AlertPayload struct {
	EventID        string `json:"eventId"`
	ProjectName    string `json:"projectName"`
	EndpointName   string `json:"endpointName"`
	Category       string `json:"category"`
	FindingType    string `json:"findingType"`
	Severity       string `json:"severity"`
	PolicyAction   string `json:"policyAction"`
	RequestID      string `json:"requestId"`
	EvidenceMasked string `json:"evidenceMasked"`
	Message        string `json:"message"`
	Timestamp      string `json:"timestamp"`
}

type Dispatcher struct {
	httpClient *http.Client
}

func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		httpClient: ssrf.NewSafeHTTPClient(5 * time.Second),
	}
}

func NewDispatcherWithClient(httpClient *http.Client) *Dispatcher {
	return &Dispatcher{
		httpClient: httpClient,
	}
}

// Dispatch sends security alerts asynchronously to the target channel with exponential backoff (#3.2)
func (d *Dispatcher) Dispatch(ctx context.Context, channelType ChannelType, webhookURL string, payload AlertPayload) error {
	var body []byte
	var err error

	switch channelType {
	case ChannelSlack:
		body, err = formatSlackBlock(payload)
	case ChannelDiscord:
		body, err = formatDiscordEmbed(payload)
	case ChannelTelegram:
		body, err = formatTelegramMessage(payload)
	default:
		body, err = json.Marshal(payload)
	}

	if err != nil {
		return fmt.Errorf("failed to format alert payload: %w", err)
	}

	maxAttempts := 3
	var lastErr error

	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(100*(1<<attempt)) * time.Millisecond
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "ApiSentinel-Security-Alerts/0.1.0")

		resp, err := d.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("alert delivery failed to %s: %w", channelType, err)
			log.Warn().Err(err).Int("attempt", attempt+1).Str("channel", string(channelType)).Msg("Retrying alert webhook delivery")
			continue
		}

		statusCode := resp.StatusCode
		resp.Body.Close()

		if statusCode >= 500 {
			lastErr = fmt.Errorf("alert endpoint returned server error HTTP %d", statusCode)
			log.Warn().Int("status", statusCode).Int("attempt", attempt+1).Str("channel", string(channelType)).Msg("Retrying alert webhook delivery after 5xx")
			continue
		}

		if statusCode >= 400 {
			return fmt.Errorf("alert endpoint returned client error HTTP %d", statusCode)
		}

		log.Info().
			Str("channel", string(channelType)).
			Str("severity", payload.Severity).
			Str("type", payload.FindingType).
			Msg("Security alert successfully dispatched")

		return nil
	}

	return fmt.Errorf("alert delivery failed after %d attempts: %w", maxAttempts, lastErr)
}

// formatSlackBlock formats rich Slack Block Kit layout
func formatSlackBlock(p AlertPayload) ([]byte, error) {
	emoji := "🚨"
	if p.Severity == "CRITICAL" {
		emoji = "🔥"
	}

	blocks := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "header",
				"text": map[string]string{
					"type":  "plain_text",
					"text":  fmt.Sprintf("%s ApiSentinel Güvenlik Uyarısı: [%s] %s", emoji, p.Severity, p.FindingType),
					"emoji": "true",
				},
			},
			{
				"type": "section",
				"fields": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Proje:*\n%s", p.ProjectName)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Endpoint:*\n%s", p.EndpointName)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Politika Kararı:*\n`%s`", p.PolicyAction)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*İstek ID:*\n`%s`", p.RequestID)},
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Açıklama:* %s\n*Maskeli Kanıt:* `%s`", p.Message, p.EvidenceMasked),
				},
			},
			{
				"type": "context",
				"elements": []map[string]string{
					{"type": "mrkdwn", "text": fmt.Sprintf("🛡️ ApiSentinel Realtime Protection • %s", p.Timestamp)},
				},
			},
		},
	}
	return json.Marshal(blocks)
}

// formatDiscordEmbed formats Discord rich embed
func formatDiscordEmbed(p AlertPayload) ([]byte, error) {
	color := 0xef4444 // Red
	if p.Severity == "HIGH" {
		color = 0xf59e0b // Amber
	}

	payload := map[string]interface{}{
		"username":   "ApiSentinel Security Bot",
		"avatar_url": "https://raw.githubusercontent.com/apisentinel/apisentinel/main/docs/logo.png",
		"embeds": []map[string]interface{}{
			{
				"title":       fmt.Sprintf("🚨 Güvenlik Uyarısı: %s (%s)", p.FindingType, p.Severity),
				"description": p.Message,
				"color":       color,
				"fields": []map[string]interface{}{
					{"name": "Proje", "value": p.ProjectName, "inline": true},
					{"name": "Endpoint", "value": p.EndpointName, "inline": true},
					{"name": "Politika", "value": fmt.Sprintf("`%s`", p.PolicyAction), "inline": true},
					{"name": "Maskeli Kanıt", "value": fmt.Sprintf("`%s`", p.EvidenceMasked), "inline": false},
					{"name": "İstek ID", "value": fmt.Sprintf("`%s`", p.RequestID), "inline": true},
				},
				"footer": map[string]string{
					"text": "ApiSentinel Protection Engine",
				},
			},
		},
	}
	return json.Marshal(payload)
}

// formatTelegramMessage formats Telegram markdown
func formatTelegramMessage(p AlertPayload) ([]byte, error) {
	text := fmt.Sprintf(
		"🚨 *ApiSentinel Güvenlik Uyarısı*\n\n"+
			"*Seviye:* `%s`\n"+
			"*Bulgu:* `%s`\n"+
			"*Proje:* %s\n"+
			"*Endpoint:* %s\n"+
			"*Politika:* `%s`\n"+
			"*Maskeli Kanıt:* `%s`\n"+
			"*İstek ID:* `%s`\n\n"+
			"_%s_",
		p.Severity, p.FindingType, p.ProjectName, p.EndpointName, p.PolicyAction, p.EvidenceMasked, p.RequestID, p.Message,
	)

	payload := map[string]string{
		"text":       text,
		"parse_mode": "Markdown",
	}
	return json.Marshal(payload)
}
