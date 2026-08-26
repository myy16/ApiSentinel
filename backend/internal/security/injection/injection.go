package injection

// ScanText executes all injection detectors (SQLi and XSS) on the payload.
func ScanText(text string) []Finding {
	var findings []Finding

	// Run SQLi scanner
	sqliFindings := ScanSQLi(text)
	if len(sqliFindings) > 0 {
		findings = append(findings, sqliFindings...)
	}

	// Run XSS scanner
	xssFindings := ScanXSS(text)
	if len(xssFindings) > 0 {
		findings = append(findings, xssFindings...)
	}

	return findings
}
