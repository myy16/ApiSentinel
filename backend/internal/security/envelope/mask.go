package envelope

import (
	"net/url"
	"strings"
)

// MaskWebhookURL masks sensitive tokens, paths, and secrets in Webhook URLs
// (Discord tokens, Slack webhook keys, Telegram bot tokens, etc.)
func MaskWebhookURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		if len(rawURL) > 10 {
			return rawURL[:6] + "********"
		}
		return "********"
	}

	lowerHost := strings.ToLower(u.Host)

	// Discord Webhook: https://discord.com/api/webhooks/<id>/<token>
	if strings.Contains(lowerHost, "discord.com") {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 4 { // api, webhooks, id, token
			return fmtURL(u.Scheme, u.Host, "/"+strings.Join(parts[:3], "/")+"/****")
		}
	}

	// Slack Webhook: https://hooks.slack.com/services/T00/B00/XXXXX
	if strings.Contains(lowerHost, "slack.com") {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 4 { // services, T, B, X
			return fmtURL(u.Scheme, u.Host, "/"+strings.Join(parts[:3], "/")+"/****")
		}
	}

	// Telegram Bot: https://api.telegram.org/bot<token>/sendMessage
	if strings.Contains(lowerHost, "telegram.org") {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 2 && strings.HasPrefix(parts[0], "bot") {
			action := ""
			if len(parts) > 1 {
				action = "/" + parts[len(parts)-1]
			}
			return fmtURL(u.Scheme, u.Host, "/bot****"+action)
		}
	}

	// Generic Webhook: mask query params and last path component if long
	if len(u.RawQuery) > 0 {
		u.RawQuery = "token=****"
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) > 1 {
		lastIdx := len(parts) - 1
		if len(parts[lastIdx]) > 6 {
			parts[lastIdx] = parts[lastIdx][:2] + "****"
		}
		u.Path = "/" + strings.Join(parts, "/")
	}

	return u.String()
}

func fmtURL(scheme, host, path string) string {
	if scheme == "" {
		scheme = "https"
	}
	return scheme + "://" + host + path
}

// MaskHeaderValue masks sensitive authorization and secret header values
func MaskHeaderValue(headerName, headerValue string) string {
	if headerValue == "" {
		return ""
	}
	lowerName := strings.ToLower(headerName)

	if lowerName == "authorization" {
		if strings.HasPrefix(headerValue, "Bearer ") {
			token := strings.TrimPrefix(headerValue, "Bearer ")
			if len(token) > 6 {
				return "Bearer " + token[:3] + "********"
			}
			return "Bearer ********"
		}
		if strings.HasPrefix(headerValue, "Basic ") {
			return "Basic ********"
		}
		return "Bearer ********"
	}

	if strings.Contains(lowerName, "key") || strings.Contains(lowerName, "token") || strings.Contains(lowerName, "secret") || strings.Contains(lowerName, "auth") {
		if len(headerValue) > 6 {
			return headerValue[:3] + "********"
		}
		return "********"
	}

	if len(headerValue) > 12 {
		return headerValue[:4] + "********"
	}
	return "********"
}
