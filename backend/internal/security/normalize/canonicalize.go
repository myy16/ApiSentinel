package normalize

import (
	"html"
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// Canonicalize applies multi-pass decoding, HTML unescaping, unicode normalizing,
// and control-character stripping to defeat WAF evasion techniques.
func Canonicalize(input string) string {
	if input == "" {
		return ""
	}

	result := input

	// 1. Strip null bytes and non-printable binary control characters (except tab, newline, carriage return)
	result = stripControlCharacters(result)

	// 2. Multi-pass URL Decoding (up to 3 passes to unwrap double/triple encoding)
	result = multiPassURLDecode(result, 3)

	// 3. HTML Entity Unescaping (&lt; -> <, &#x3C; -> <, etc.)
	result = html.UnescapeString(result)

	// 4. JSON / JS Unicode Unescaping (\u0027 -> ', \x27 -> ')
	result = unescapeUnicodeSequences(result)

	// 5. Secondary URL decode pass (in case HTML entities revealed new %-encodings)
	if strings.Contains(result, "%") {
		if decoded, err := url.QueryUnescape(result); err == nil && decoded != result {
			result = decoded
		}
	}

	return result
}

// multiPassURLDecode iteratively unwraps URL encoding until stable or maxPasses reached
func multiPassURLDecode(s string, maxPasses int) string {
	curr := s
	for i := 0; i < maxPasses; i++ {
		if !strings.Contains(curr, "%") && !strings.Contains(curr, "+") {
			break
		}
		decoded, err := url.QueryUnescape(curr)
		if err != nil || decoded == curr {
			break
		}
		curr = decoded
	}
	return curr
}

// unescapeUnicodeSequences handles \uXXXX and \xXX escape sequences
func unescapeUnicodeSequences(s string) string {
	if !strings.Contains(s, `\u`) && !strings.Contains(s, `\x`) {
		return s
	}

	var sb strings.Builder
	sb.Grow(len(s))

	i := 0
	for i < len(s) {
		// Handle \uXXXX (4 hex digits)
		if i+5 < len(s) && s[i] == '\\' && (s[i+1] == 'u' || s[i+1] == 'U') {
			hexStr := s[i+2 : i+6]
			if r, err := strconv.ParseInt(hexStr, 16, 32); err == nil {
				sb.WriteRune(rune(r))
				i += 6
				continue
			}
		}

		// Handle \xXX (2 hex digits)
		if i+3 < len(s) && s[i] == '\\' && (s[i+1] == 'x' || s[i+1] == 'X') {
			hexStr := s[i+2 : i+4]
			if r, err := strconv.ParseInt(hexStr, 16, 32); err == nil {
				sb.WriteRune(rune(r))
				i += 4
				continue
			}
		}

		sb.WriteByte(s[i])
		i++
	}

	return sb.String()
}

// stripControlCharacters removes null bytes and dangerous control chars while preserving newlines & tabs
func stripControlCharacters(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))

	for _, r := range s {
		if r == 0 {
			// Null byte stripped
			continue
		}
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		sb.WriteRune(r)
	}

	return sb.String()
}
