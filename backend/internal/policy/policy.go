package policy

import (
	"github.com/apisentinel/apisentinel/internal/security"
)

type Action string

const (
	ActionAllow Action = "ALLOW"
	ActionWarn  Action = "WARN"
	ActionAlert Action = "ALERT"
	ActionMask  Action = "MASK"
	ActionBlock Action = "BLOCK"
)

type Decision struct {
	Action         Action             `json:"action"`
	Reason         string             `json:"reason"`
	MatchedFinding *security.Finding  `json:"matched_finding,omitempty"`
}

// Evaluate determines whether a request should be ALLOWED, MASKED, or BLOCKED
func Evaluate(findings []security.Finding) Decision {
	if len(findings) == 0 {
		return Decision{
			Action: ActionAllow,
			Reason: "No security violations detected",
		}
	}

	// 1. Highest priority: BLOCK on CRITICAL secrets
	for _, f := range findings {
		if f.Category == "SECRET" && f.Severity == "CRITICAL" {
			return Decision{
				Action:         ActionBlock,
				Reason:         "Critical secret exposure detected in payload: " + f.Type,
				MatchedFinding: &f,
			}
		}
	}

	// 2. High priority: BLOCK on Credit Card / TCKN exposure
	for _, f := range findings {
		if f.Type == "CREDIT_CARD" || f.Type == "TCKN" {
			return Decision{
				Action:         ActionBlock,
				Reason:         "Sensitive personal identity/financial data detected: " + f.Type,
				MatchedFinding: &f,
			}
		}
	}

	// 3. MASK on Email / Phone
	for _, f := range findings {
		if f.Type == "EMAIL" || f.Type == "PHONE" {
			return Decision{
				Action:         ActionMask,
				Reason:         "Personal contact data detected, applied masking policy",
				MatchedFinding: &f,
			}
		}
	}

	return Decision{
		Action: ActionAllow,
		Reason: "Findings passed default policy filter",
	}
}
