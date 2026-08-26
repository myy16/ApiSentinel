package normalize

import (
	"testing"
)

func TestCanonicalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Plain text unchanged",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "Single URL encoding",
			input:    "%27%20OR%201%3D1",
			expected: "' OR 1=1",
		},
		{
			name:     "Double URL encoding",
			input:    "%2527%2520OR%25201%253D1",
			expected: "' OR 1=1",
		},
		{
			name:     "HTML Named Entities",
			input:    "&lt;script&gt;alert(&quot;xss&quot;)&lt;/script&gt;",
			expected: `<script>alert("xss")</script>`,
		},
		{
			name:     "HTML Hex Entities",
			input:    "&#x3C;script&#x3E;alert(1)&#x3C;/script&#x3E;",
			expected: "<script>alert(1)</script>",
		},
		{
			name:     "JSON Unicode Escapes",
			input:    `\u003cscript\u003ealert(\u0027xss\u0027)\u003c/script\u003e`,
			expected: `<script>alert('xss')</script>`,
		},
		{
			name:     "Null byte stripping",
			input:    "admin\x00' OR 1=1",
			expected: "admin' OR 1=1",
		},
		{
			name:     "Mixed Obfuscation: Double URL + HTML + Unicode",
			input:    "%2526lt%253Bscript%2526gt%253B",
			expected: "<script>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Canonicalize(tt.input)
			if got != tt.expected {
				t.Errorf("Canonicalize(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}
