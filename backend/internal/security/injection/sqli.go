package injection

import (
	"regexp"
	"strings"
	"sync"
)

var (
	sqliOnce sync.Once

	// SQL Injection detection regex patterns
	sqliUnionRegex      *regexp.Regexp
	sqliTautologyRegex  *regexp.Regexp
	sqliStackedRegex    *regexp.Regexp
	sqliTimeBasedRegex  *regexp.Regexp
	sqliCommentRegex    *regexp.Regexp
	sqliErrorBasedRegex *regexp.Regexp
	sqliKeywordCombos   *regexp.Regexp
)

func initSQLiRegexes() {
	sqliOnce.Do(func() {
		// 1. UNION SELECT / ALL SELECT
		sqliUnionRegex = regexp.MustCompile(`(?i)\bUNION\s+(?:ALL\s+)?SELECT\b`)

		// 2. Boolean-based Tautologies (' OR '1'='1, " OR 1=1, ' OR 'a'='a, OR 1=1 --)
		sqliTautologyRegex = regexp.MustCompile(`(?i)(?:'|\"|\b)\s*(?:OR|AND)\s+(?:'[^']+'='[^']+'|\d+=\d+|"[^"]+"="[^"]+")`)

		// 3. Stacked destructive commands (DROP TABLE, TRUNCATE, ALTER TABLE, EXEC xp_cmdshell)
		sqliStackedRegex = regexp.MustCompile(`(?i);\s*(?:DROP\s+(?:TABLE|DATABASE|VIEW)|TRUNCATE\s+TABLE|ALTER\s+TABLE|EXEC(?:UTE)?\s+(?:xp_|sp_))\b`)

		// 4. Time-based blind SQLi (WAITFOR DELAY, SLEEP(N), BENCHMARK(N, ...), pg_sleep(N))
		sqliTimeBasedRegex = regexp.MustCompile(`(?i)\b(?:WAITFOR\s+DELAY|SLEEP\s*\(\s*\d+\s*\)|BENCHMARK\s*\(\s*\d+|PG_SLEEP\s*\(\s*\d+\s*\))`)

		// 5. Dangerous SQL Comment injections with keywords (-- or /* ... */ with quotes/statements)
		sqliCommentRegex = regexp.MustCompile(`(?i)(?:'|\"|` + "`" + `)\s*(?:--|#|/\*.*?\*/)\s*(?:SELECT|UNION|INSERT|UPDATE|DELETE|DROP)?`)

		// 6. Error-based extraction functions (EXTRACTVALUE, UPDATEXML, EXP(~), CTXSYS.DRV_LOAD)
		sqliErrorBasedRegex = regexp.MustCompile(`(?i)\b(?:EXTRACTVALUE|UPDATEXML|LOAD_FILE|INTO\s+(?:OUTFILE|DUMPFILE))\b`)

		// 7. Dangerous keyword combinations (e.g. SELECT ... FROM ... WHERE ... ; INSERT INTO ...)
		sqliKeywordCombos = regexp.MustCompile(`(?i)\b(?:SELECT\s+.*?\s+FROM\s+information_schema|SELECT\s+@@version|SELECT\s+user\(\)|CONCAT\s*\(.*?,\s*0x[0-9a-fA-F]+)\b`)
	})
}

// Finding represents an individual injection finding.
type Finding struct {
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	Message        string  `json:"message"`
	EvidenceMasked string  `json:"evidence_masked"`
	Confidence     float64 `json:"confidence"`
}

// ScanSQLi scans the payload for SQL injection attack patterns.
func ScanSQLi(text string) []Finding {
	initSQLiRegexes()

	var findings []Finding

	// 1. Union-based SQLi
	if sqliUnionRegex.MatchString(text) {
		match := sqliUnionRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "SQLI_UNION_BASED",
			Severity:       "CRITICAL",
			Message:        "SQL Injection attempt detected: UNION SELECT construct",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.95,
		})
	}

	// 2. Boolean-based / Tautology
	if sqliTautologyRegex.MatchString(text) {
		match := sqliTautologyRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "SQLI_TAUTOLOGY",
			Severity:       "HIGH",
			Message:        "SQL Injection attempt detected: Boolean tautology bypass (e.g. OR 1=1)",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.90,
		})
	}

	// 3. Stacked destructive queries
	if sqliStackedRegex.MatchString(text) {
		match := sqliStackedRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "SQLI_STACKED_QUERY",
			Severity:       "CRITICAL",
			Message:        "Destructive stacked SQL query execution attempt detected",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.99,
		})
	}

	// 4. Time-based blind SQLi
	if sqliTimeBasedRegex.MatchString(text) {
		match := sqliTimeBasedRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "SQLI_TIME_BASED",
			Severity:       "HIGH",
			Message:        "Blind SQL Injection attempt detected: Time delay / benchmark function",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.92,
		})
	}

	// 5. Error-based / File I/O extraction
	if sqliErrorBasedRegex.MatchString(text) {
		match := sqliErrorBasedRegex.FindString(text)
		findings = append(findings, Finding{
			Type:           "SQLI_ERROR_BASED",
			Severity:       "HIGH",
			Message:        "SQL Injection attempt detected: Error-based extraction or file access",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.90,
		})
	}

	// 6. Schema/Metadata enumeration
	if sqliKeywordCombos.MatchString(text) {
		match := sqliKeywordCombos.FindString(text)
		findings = append(findings, Finding{
			Type:           "SQLI_METADATA_PROBING",
			Severity:       "HIGH",
			Message:        "SQL Injection attempt detected: Database metadata / version probing",
			EvidenceMasked: maskEvidence(match),
			Confidence:     0.90,
		})
	}

	return findings
}

func maskEvidence(raw string) string {
	if len(raw) <= 8 {
		return raw
	}
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) > 30 {
		return trimmed[:15] + "...[REDACTED]..." + trimmed[len(trimmed)-5:]
	}
	return trimmed
}
