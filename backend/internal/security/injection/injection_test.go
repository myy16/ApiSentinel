package injection

import (
	"testing"
)

func TestSQLiDetection(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		expectedType string
	}{
		{
			name:         "Union Select attack",
			payload:      `{"search": "admin' UNION SELECT null, username, password FROM users --"}`,
			expectedType: "SQLI_UNION_BASED",
		},
		{
			name:         "Tautology Boolean Bypass",
			payload:      `{"username": "admin' OR 1=1 --", "password": "x"}`,
			expectedType: "SQLI_TAUTOLOGY",
		},
		{
			name:         "Stacked Query Drop Table",
			payload:      `{"id": "10; DROP TABLE customers;"}`,
			expectedType: "SQLI_STACKED_QUERY",
		},
		{
			name:         "Time-based Blind Sleep",
			payload:      `{"filter": "1' AND SLEEP(5) --"}`,
			expectedType: "SQLI_TIME_BASED",
		},
		{
			name:         "Error-based ExtractValue",
			payload:      `{"query": "AND EXTRACTVALUE(1, CONCAT(0x7e, @@version))"}`,
			expectedType: "SQLI_ERROR_BASED",
		},
		{
			name:         "Information Schema Probing",
			payload:      `{"param": "SELECT table_name FROM information_schema.tables"}`,
			expectedType: "SQLI_METADATA_PROBING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := ScanSQLi(tt.payload)
			if len(findings) == 0 {
				t.Fatalf("expected SQLi finding for '%s', got none", tt.name)
			}
			found := false
			for _, f := range findings {
				if f.Type == tt.expectedType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected finding type %s, got %+v", tt.expectedType, findings)
			}
		})
	}
}

func TestXSSDetection(t *testing.T) {
	tests := []struct {
		name         string
		payload      string
		expectedType string
	}{
		{
			name:         "Script Tag Injection",
			payload:      `<script>alert('XSS')</script>`,
			expectedType: "XSS_SCRIPT_TAG",
		},
		{
			name:         "Inline Event Handler (onerror)",
			payload:      `<img src="invalid.jpg" onerror="alert(document.cookie)">`,
			expectedType: "XSS_EVENT_HANDLER",
		},
		{
			name:         "JavaScript URI Scheme",
			payload:      `<a href="javascript:eval('malicious')">Click</a>`,
			expectedType: "XSS_JAVASCRIPT_URI",
		},
		{
			name:         "Dangerous Iframe Injection",
			payload:      `<iframe src="http://evil.com"></iframe>`,
			expectedType: "XSS_DANGEROUS_ELEMENT",
		},
		{
			name:         "DOM Access Cookie Theft",
			payload:      `{"hook": "var token = document.cookie;"}`,
			expectedType: "XSS_DOM_MANIPULATION",
		},
		{
			name:         "Data URI Executable HTML",
			payload:      `<object data="data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==">`,
			expectedType: "XSS_DATA_URI_PAYLOAD",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := ScanXSS(tt.payload)
			if len(findings) == 0 {
				t.Fatalf("expected XSS finding for '%s', got none", tt.name)
			}
			found := false
			for _, f := range findings {
				if f.Type == tt.expectedType {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected finding type %s, got %+v", tt.expectedType, findings)
			}
		})
	}
}

func TestCleanPayloads_NoFalsePositives(t *testing.T) {
	cleanPayloads := []string{
		`{"event": "order.completed", "amount": 100, "customer": "John Doe"}`,
		`{"notes": "Please select the options that you want from the drop down menu."}`,
		`{"description": "This is a clean description with punctuation & normal English text."}`,
		`{"url": "https://api.stripe.com/v1/charges", "status": "succeeded"}`,
	}

	for i, p := range cleanPayloads {
		findings := ScanText(p)
		if len(findings) > 0 {
			t.Errorf("clean payload #%d generated false positive finding: %+v", i, findings)
		}
	}
}
