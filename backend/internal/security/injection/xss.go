package injection

import (
	"regexp"
	"sync"
)

var (
	xssOnce sync.Once

	// XSS detection patterns
	xssScriptTagRegex    *regexp.Regexp
	xssEventHandlerRegex *regexp.Regexp
	xssJavascriptURIRegex *regexp.Regexp
	xssIframeObjectRegex *regexp.Regexp
	xssDOMAccessRegex    *regexp.Regexp
	xssDataHTMLRegex     *regexp.Regexp
)

func initXSSRegexes() {
	xssOnce.Do(func() {
		// 1. Script tag injection (<script>...</script>, <script src=...>, <SCRIPT/XSS>)
		xssScriptTagRegex = regexp.MustCompile(`(?i)<\s*script[^>]*>`)

		// 2. Inline Event Handlers (onload=, onerror=, onclick=, onmouseover=, onfocus=, onblur=, etc.)
		xssEventHandlerRegex = regexp.MustCompile(`(?i)\bon(?:error|load|click|mouseover|mouseenter|focus|blur|change|submit|keydown|keypress|keyup|pointerdown)\s*=\s*['"]?[^'">\s]+`)

		// 3. Javascript / VBScript URI pseudoprotocols (href="javascript:...", src="javascript:alert(1)")
		xssJavascriptURIRegex = regexp.MustCompile(`(?i)(?:href|src|action|data)\s*=\s*['"]?\s*(?:javascript|vbscript|livescript):`)

		// 4. Dangerous embedding elements (<iframe>, <object>, <embed>, <applet>, <base>)
		xssIframeObjectRegex = regexp.MustCompile(`(?i)<\s*(?:iframe|object|embed|applet|base)[^>]*>`)

		// 5. DOM/Cookie manipulation in payload (document.cookie, document.location, window.location, eval(...))
		xssDOMAccessRegex = regexp.MustCompile(`(?i)\b(?:document\.cookie|document\.location|window\.location|eval\s*\(|setTimeout\s*\(\s*['"]|Function\s*\()\b`)

		// 6. Data URI with HTML/SVG executable payload (data:text/html, data:image/svg+xml)
		xssDataHTMLRegex = regexp.MustCompile(`(?i)data:\s*(?:text\/html|image\/svg\+xml)\s*;\s*base64\s*,`)
	})
}

// ScanXSS scans the payload for Cross-Site Scripting (XSS) injection patterns.
func ScanXSS(text string) []Finding {
	initXSSRegexes()

	var findings []Finding

	// 1. Script Tag
	if xssScriptTagRegex.MatchString(text) {
		match := xssScriptTagRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "XSS_SCRIPT_TAG",
			Severity:       "HIGH",
			Message:        "Cross-Site Scripting (XSS) attempt: <script> tag injection detected",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.98,
		})
	}

	// 2. Event Handler Injection
	if xssEventHandlerRegex.MatchString(text) {
		match := xssEventHandlerRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "XSS_EVENT_HANDLER",
			Severity:       "HIGH",
			Message:        "Cross-Site Scripting (XSS) attempt: HTML inline event handler injection (e.g. onerror/onload)",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.95,
		})
	}

	// 3. JavaScript URI Scheme
	if xssJavascriptURIRegex.MatchString(text) {
		match := xssJavascriptURIRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "XSS_JAVASCRIPT_URI",
			Severity:       "HIGH",
			Message:        "Cross-Site Scripting (XSS) attempt: javascript: URI scheme execution",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.95,
		})
	}

	// 4. Iframe / Object Embedding
	if xssIframeObjectRegex.MatchString(text) {
		match := xssIframeObjectRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "XSS_DANGEROUS_ELEMENT",
			Severity:       "MEDIUM",
			Message:        "Dangerous HTML embedding tag (<iframe/object/embed>) detected",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.88,
		})
	}

	// 5. DOM Access / Cookie Theft
	if xssDOMAccessRegex.MatchString(text) {
		match := xssDOMAccessRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "XSS_DOM_MANIPULATION",
			Severity:       "HIGH",
			Message:        "Suspicious DOM/Cookie access construct detected (e.g. document.cookie, eval)",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.90,
		})
	}

	// 6. Data URI Executable HTML/SVG
	if xssDataHTMLRegex.MatchString(text) {
		match := xssDataHTMLRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "XSS_DATA_URI_PAYLOAD",
			Severity:       "HIGH",
			Message:        "Executable data: URI payload (text/html or image/svg) detected",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.92,
		})
	}

	return findings
}
