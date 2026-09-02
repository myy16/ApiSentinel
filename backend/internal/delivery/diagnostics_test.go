package delivery

import (
	"errors"
	"testing"
)

func TestDiagnoseAttempt_AllCategories(t *testing.T) {
	tests := []struct {
		name             string
		statusCode       int
		err              error
		targetURL        string
		respSnippet      string
		expectedCategory DiagnosticCategory
		expectedSeverity string
	}{
		{
			name:             "200 OK -> Success",
			statusCode:       200,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			respSnippet:      `{"status":"ok"}`,
			expectedCategory: DiagSuccess,
			expectedSeverity: "INFO",
		},
		{
			name:             "SSRF Blocked -> Private IP",
			statusCode:       0,
			err:              errors.New("SSRF: private IP address blocked: 127.0.0.1"),
			targetURL:        "http://127.0.0.1:8080/hook",
			expectedCategory: DiagSSRFBlocked,
			expectedSeverity: "CRITICAL",
		},
		{
			name:             "DNS Resolution Failed -> no such host",
			statusCode:       0,
			err:              errors.New("dial tcp: lookup non-existent-api.domain.internal: no such host"),
			targetURL:        "https://non-existent-api.domain.internal/hook",
			expectedCategory: DiagDNSLookupFailed,
			expectedSeverity: "CRITICAL",
		},
		{
			name:             "Connection Refused -> port closed",
			statusCode:       0,
			err:              errors.New("dial tcp 10.0.0.5:8080: connect: connection refused"),
			targetURL:        "http://10.0.0.5:8080/hook",
			expectedCategory: DiagConnectionRefused,
			expectedSeverity: "CRITICAL",
		},
		{
			name:             "TLS Certificate Expired",
			statusCode:       0,
			err:              errors.New("tls: failed to verify certificate: x509: certificate has expired"),
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagTLSCertError,
			expectedSeverity: "CRITICAL",
		},
		{
			name:             "HTTP Timeout (408)",
			statusCode:       408,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagTimeout,
			expectedSeverity: "WARNING",
		},
		{
			name:             "Network Timeout / Context Deadline Exceeded",
			statusCode:       0,
			err:              errors.New("net/http: request canceled (Client.Timeout exceeded while awaiting headers)"),
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagTimeout,
			expectedSeverity: "WARNING",
		},
		{
			name:             "401 Unauthorized",
			statusCode:       401,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagAuthFailure,
			expectedSeverity: "CRITICAL",
		},
		{
			name:             "403 Forbidden",
			statusCode:       403,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagForbidden,
			expectedSeverity: "CRITICAL",
		},
		{
			name:             "404 Not Found",
			statusCode:       404,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagNotFound,
			expectedSeverity: "CRITICAL",
		},
		{
			name:             "422 Unprocessable Entity",
			statusCode:       422,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			respSnippet:      `{"error":"missing required field 'order_id'"}`,
			expectedCategory: DiagSchemaValidationError,
			expectedSeverity: "WARNING",
		},
		{
			name:             "429 Rate Limited",
			statusCode:       429,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagRateLimited,
			expectedSeverity: "WARNING",
		},
		{
			name:             "500 Internal Server Error",
			statusCode:       500,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagServerError,
			expectedSeverity: "WARNING",
		},
		{
			name:             "503 Service Unavailable",
			statusCode:       503,
			err:              nil,
			targetURL:        "https://api.internal/hook",
			expectedCategory: DiagServerError,
			expectedSeverity: "WARNING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diag := DiagnoseAttempt(tt.statusCode, tt.err, tt.targetURL, tt.respSnippet)
			if diag.Category != tt.expectedCategory {
				t.Errorf("Expected category %s, got %s", tt.expectedCategory, diag.Category)
			}
			if diag.Severity != tt.expectedSeverity {
				t.Errorf("Expected severity %s, got %s", tt.expectedSeverity, diag.Severity)
			}
			if diag.Title == "" {
				t.Errorf("Expected non-empty title")
			}
			if diag.SuggestedAction == "" {
				t.Errorf("Expected non-empty suggested action")
			}
		})
	}
}
