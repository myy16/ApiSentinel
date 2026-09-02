package delivery

import (
	"fmt"
	"strings"
)

// DiagnosticCategory represents the root classification of a delivery attempt failure.
type DiagnosticCategory string

const (
	DiagSuccess               DiagnosticCategory = "SUCCESS"
	DiagDNSLookupFailed       DiagnosticCategory = "DNS_LOOKUP_FAILED"
	DiagConnectionRefused     DiagnosticCategory = "CONNECTION_REFUSED"
	DiagTLSCertError          DiagnosticCategory = "TLS_CERT_ERROR"
	DiagTimeout               DiagnosticCategory = "HTTP_TIMEOUT"
	DiagAuthFailure           DiagnosticCategory = "HTTP_AUTH_FAILURE"
	DiagForbidden             DiagnosticCategory = "HTTP_FORBIDDEN"
	DiagNotFound              DiagnosticCategory = "HTTP_NOT_FOUND"
	DiagSchemaValidationError DiagnosticCategory = "SCHEMA_VALIDATION_ERROR"
	DiagRateLimited           DiagnosticCategory = "RATE_LIMITED"
	DiagServerError           DiagnosticCategory = "SERVER_INTERNAL_ERROR"
	DiagSSRFBlocked           DiagnosticCategory = "SSRF_BLOCKED"
	DiagUnknown               DiagnosticCategory = "UNKNOWN_ERROR"
)

// DiagnosticResult contains actionable root cause analysis and a fix recommendation.
type DiagnosticResult struct {
	Category        DiagnosticCategory `json:"category"`
	Severity        string             `json:"severity"` // "CRITICAL", "WARNING", "INFO"
	Title           string             `json:"title"`
	RootCause       string             `json:"rootCause"`
	SuggestedAction string             `json:"suggestedAction"`
	QuickFixSnippet string             `json:"quickFixSnippet,omitempty"`
	DocLink         string             `json:"docLink,omitempty"`
}

// DiagnoseAttempt analyzes the status code, network error, and response snippet to produce an actionable diagnosis.
func DiagnoseAttempt(statusCode int, netErr error, targetURL, respSnippet string) DiagnosticResult {
	if netErr == nil && statusCode >= 200 && statusCode < 300 {
		return DiagnosticResult{
			Category:        DiagSuccess,
			Severity:        "INFO",
			Title:           "Başarılı İletim",
			RootCause:       "Upstream sunucu isteği başarıyla kabul etti (2xx OK).",
			SuggestedAction: "Ek bir aksiyon gerekmemektedir.",
		}
	}

	errStr := ""
	if netErr != nil {
		errStr = strings.ToLower(netErr.Error())
	}

	// 1. SSRF or Private IP Blocking
	if strings.Contains(errStr, "ssrf") || strings.Contains(errStr, "private ip") || strings.Contains(errStr, "loopback") {
		return DiagnosticResult{
			Category:        DiagSSRFBlocked,
			Severity:        "CRITICAL",
			Title:           "SSRF / Özel Ağ Engeli",
			RootCause:       fmt.Sprintf("Hedef URL (%s) özel IP / localhost aralığında olduğu için güvenlik kalkanı tarafından engellendi.", targetURL),
			SuggestedAction: "İletim hedefini genel erişime açık (public domain) bir HTTPS adresine güncelleyin veya yerel test için ApiSentinel CLI tünelini kullanın.",
			DocLink:         "https://apisentinel.dev/docs/security/ssrf-protection",
		}
	}

	// 2. DNS Resolution Failures
	if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "lookup") {
		return DiagnosticResult{
			Category:        DiagDNSLookupFailed,
			Severity:        "CRITICAL",
			Title:           "DNS Alan Adı Çözümlenemedi",
			RootCause:       fmt.Sprintf("Hedef sunucunun DNS kaydı bulunamadı (%s). Alan adı yazım hatası veya DNS kesintisi mevcut.", targetURL),
			SuggestedAction: "Endpoint yapılandırmasındaki Upstream URL alan adını doğrulayın. Alan adınızın public DNS sunucularında çözümlendiğinden emin olun.",
			QuickFixSnippet: fmt.Sprintf("nslookup $(echo %s | awk -F/ '{print $3}')", targetURL),
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#dns",
		}
	}

	// 3. Connection Refused / Server Unreachable
	if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connect: connection refused") {
		return DiagnosticResult{
			Category:        DiagConnectionRefused,
			Severity:        "CRITICAL",
			Title:           "Bağlantı Reddedildi (Port Kapalı)",
			RootCause:       fmt.Sprintf("Hedef sunucuya (%s) TCP seviyesinde ulaşılamadı. Port dinlenmiyor veya sunucu çökmüş.", targetURL),
			SuggestedAction: "Upstream sunucunuzun ayakta ve belirtilen portta dinlemede olduğunu kontrol edin. Güvenlik duvarı (Firewall) kurallarını gözden geçirin.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#connection-refused",
		}
	}

	// 4. TLS/SSL Certificate Errors
	if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls") || strings.Contains(errStr, "x509") {
		return DiagnosticResult{
			Category:        DiagTLSCertError,
			Severity:        "CRITICAL",
			Title:           "SSL/TLS Sertifika Hatası",
			RootCause:       "Hedef sunucunun SSL sertifikası geçersiz, süresi dolmuş veya güvenilmeyen bir CA tarafından imzalanmış.",
			SuggestedAction: "Upstream sunucunuzun SSL/TLS sertifikasını yenileyin (örn: Let's Encrypt / Certbot) veya geçerli bir sertifika zinciri tanımlayın.",
			QuickFixSnippet: fmt.Sprintf("openssl s_client -connect $(echo %s | awk -F/ '{print $3}'):443 -servername $(echo %s | awk -F/ '{print $3}')", targetURL, targetURL),
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#ssl-tls",
		}
	}

	// 5. Timeouts (Network or Upstream Latency)
	if statusCode == 408 || statusCode == 504 || strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return DiagnosticResult{
			Category:        DiagTimeout,
			Severity:        "WARNING",
			Title:           "İstek Zaman Aşımı (Timeout)",
			RootCause:       "Upstream sunucunuz belirlenen zaman aşımı süresi içinde HTTP yanıtı döndürmedi.",
			SuggestedAction: "Upstream sunucunuzun webhook işleme mantığını asenkron (arka plan kuyruğuna atıp 200 dönen) hale getirin veya Forwarding ayarlarından timeout_ms süresini artırın.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#timeouts",
		}
	}

	// 6. 401 Unauthorized
	if statusCode == 401 {
		return DiagnosticResult{
			Category:        DiagAuthFailure,
			Severity:        "CRITICAL",
			Title:           "Yetkilendirme Hatası (401 Unauthorized)",
			RootCause:       "Upstream sunucunuz isteği reddetti. İletilen Authorization başlığı eksik, geçersiz veya API anahtarının süresi dolmuş.",
			SuggestedAction: "Forwarding ayarlarından upstream sunucunuza iletilen Özel Başlıkları (Custom Headers: Authorization / X-API-Key) kontrol edin.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#auth",
		}
	}

	// 7. 403 Forbidden
	if statusCode == 403 {
		return DiagnosticResult{
			Category:        DiagForbidden,
			Severity:        "CRITICAL",
			Title:           "Erişim Engellendi (403 Forbidden)",
			RootCause:       "Upstream sunucunuz ApiSentinel IP adresini veya isteğin kaynağını engelledi (WAF, IP Allowlist veya CORS/Origin kuralı).",
			SuggestedAction: "Upstream sunucunuzdaki WAF / Cloudflare / Nginx kurallarını inceleyerek ApiSentinel teslimat IP'sine izin verin.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#forbidden",
		}
	}

	// 8. 404 Not Found
	if statusCode == 404 {
		return DiagnosticResult{
			Category:        DiagNotFound,
			Severity:        "CRITICAL",
			Title:           "Upstream URL Bulunamadı (404 Not Found)",
			RootCause:       fmt.Sprintf("Upstream hedef URL'si (%s) sunucuda mevcut değil veya rota yolu (path) yanlış yazılmış.", targetURL),
			SuggestedAction: "Endpoint ayarlarındaki Upstream URL yolunu (örn: /api/webhooks/stripe) ve HTTP metodunu doğrulayın.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#404",
		}
	}

	// 9. 422 Unprocessable Entity / 400 Bad Request
	if statusCode == 422 || statusCode == 400 {
		return DiagnosticResult{
			Category:        DiagSchemaValidationError,
			Severity:        "WARNING",
			Title:           "Şema Doğrulama Hatası (400 / 422)",
			RootCause:       fmt.Sprintf("Upstream sunucu gönderilen payload formatını kabul etmedi. Yanıt: %s", truncateText(respSnippet, 120)),
			SuggestedAction: "Payload içeriğindeki zorunlu alanları veya Redaction ayarlarını (REDACTED vs RAW mod) inceleyin.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#schema-validation",
		}
	}

	// 10. 429 Too Many Requests (Rate Limited)
	if statusCode == 429 {
		return DiagnosticResult{
			Category:        DiagRateLimited,
			Severity:        "WARNING",
			Title:           "Hız Sınırı Aşıldı (429 Rate Limited)",
			RootCause:       "Upstream sunucunuz gelen istek yoğunluğunu kaldıramayarak hız sınırlaması (Rate Limit) uyguladı.",
			SuggestedAction: "ApiSentinel otomatik olarak Retry-After başlığına göre bekleyip tekrar deneyecektir. Upstream sunucunuzun rate limit kotasını artırabilirsiniz.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#rate-limiting",
		}
	}

	// 11. 500, 502, 503 Server Error
	if statusCode >= 500 {
		return DiagnosticResult{
			Category:        DiagServerError,
			Severity:        "WARNING",
			Title:           fmt.Sprintf("Upstream Sunucu Hatası (%d)", statusCode),
			RootCause:       fmt.Sprintf("Upstream sunucu iç hata verdi veya ters vekil (Nginx/Envoy) upstream bağlantısı kuramadı (%d).", statusCode),
			SuggestedAction: "Upstream sunucu loglarını kontrol edin (Exception / Unhandled Error). ApiSentinel exponential backoff ile yeniden denemeye devam edecektir.",
			DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting#server-errors",
		}
	}

	// 12. Fallback Unknown
	return DiagnosticResult{
		Category:        DiagUnknown,
		Severity:        "WARNING",
		Title:           "Bilinmeyen İletim Hatası",
		RootCause:       fmt.Sprintf("İletim esnasında beklenmeyen bir hata oluştu: %s (Status: %d)", errStr, statusCode),
		SuggestedAction: "Sunucu ve ağ bağlantılarını kontrol edip Replay Lab üzerinden isteği yeniden ateşlemeyi deneyin.",
		DocLink:         "https://apisentinel.dev/docs/delivery/troubleshooting",
	}
}

func truncateText(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
