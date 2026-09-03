package ai

import (
	"regexp"
	"strings"
)

// PromptSecurityCheck represents the result of a prompt injection inspection.
type PromptSecurityCheck struct {
	IsSafe        bool     `json:"isSafe"`
	RiskScore     float64  `json:"riskScore"` // 0.0 (safe) to 1.0 (malicious)
	ThreatsFound  []string `json:"threatsFound"`
	CleanedPrompt string   `json:"cleanedPrompt"`
}

var (
	// Direct System Override & Jailbreak Patterns
	injectionPatterns = []struct {
		name    string
		pattern *regexp.Regexp
		weight  float64
	}{
		{
			name:    "SYSTEM_INSTRUCTION_OVERRIDE",
			pattern: regexp.MustCompile(`(?i)(?:ignore|disregard|forget|bypass)\s+(?:all\s+)?(?:previous|prior|above)\s+(?:instructions|prompts|rules|commands)`),
			weight:  0.95,
		},
		{
			name:    "ROLE_PLAY_HIJACKING",
			pattern: regexp.MustCompile(`(?i)(?:you\s+are\s+now|act\s+as|pretend\s+to\s+be)\s+(?:DAN|developer\s+mode|unrestricted|god\s+mode|root)`),
			weight:  0.90,
		},
		{
			name:    "SYSTEM_DELIMITER_INJECTION",
			pattern: regexp.MustCompile(`(?i)<\/?(?:system|prompt|assistant|human|context|instruction|admin)>`),
			weight:  0.85,
		},
		{
			name:    "MARKDOWN_ROLE_INJECTION",
			pattern: regexp.MustCompile(`(?i)(?:^|\n)(?:system|assistant|admin)\s*:\s*`),
			weight:  0.80,
		},
		{
			name:    "CREDENTIAL_EXFILTRATION_PROMPT",
			pattern: regexp.MustCompile(`(?i)(?:print|reveal|output|display|show|leak)\s+(?:your\s+)?(?:system\s+prompt|api\s+key|environment\s+variables|jwt_secret|credentials)`),
			weight:  0.90,
		},
		{
			name:    "SECURITY_RULE_BYPASS",
			pattern: regexp.MustCompile(`(?i)(?:disable|turn\s+off|override)\s+(?:content\s+filter|safety\s+guidelines|security\s+checks)`),
			weight:  0.85,
		},
	}
)

// InspectAndNeutralizePrompt inspects text for prompt injection and escapes/neutralizes dangerous sequences.
func InspectAndNeutralizePrompt(input string) PromptSecurityCheck {
	if strings.TrimSpace(input) == "" {
		return PromptSecurityCheck{
			IsSafe:        true,
			RiskScore:     0.0,
			ThreatsFound:  nil,
			CleanedPrompt: "",
		}
	}

	var threats []string
	maxScore := 0.0
	cleaned := input

	for _, item := range injectionPatterns {
		if matches := item.pattern.FindAllString(cleaned, -1); len(matches) > 0 {
			threats = append(threats, item.name)
			if item.weight > maxScore {
				maxScore = item.weight
			}

			// Neutralize by wrapping with explicit harmless tag and neutralizing delimiters
			cleaned = item.pattern.ReplaceAllStringFunc(cleaned, func(match string) string {
				escaped := strings.ReplaceAll(match, "<", "[")
				escaped = strings.ReplaceAll(escaped, ">", "]")
				return "[FILTERED_INJECTION: " + escaped + "]"
			})
		}
	}

	// Also escape raw XML-like delimiters to prevent framing attacks
	cleaned = strings.ReplaceAll(cleaned, "<|im_start|>", "[im_start]")
	cleaned = strings.ReplaceAll(cleaned, "<|im_end|>", "[im_end]")

	isSafe := maxScore < 0.70

	return PromptSecurityCheck{
		IsSafe:        isSafe,
		RiskScore:     maxScore,
		ThreatsFound:  threats,
		CleanedPrompt: cleaned,
	}
}
