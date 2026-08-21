package policy

import (
	"testing"

	"github.com/apisentinel/apisentinel/internal/security"
)

func TestPolicyEvaluation(t *testing.T) {
	// 1. Critical secret must BLOCK
	findings := []security.Finding{
		{Category: "SECRET", Type: "AWS_KEY", Severity: "CRITICAL"},
	}
	decision := Evaluate(findings)
	if decision.Action != ActionBlock {
		t.Errorf("Expected ActionBlock for critical secret, got %s", decision.Action)
	}

	// 2. Credit Card must BLOCK
	ccFindings := []security.Finding{
		{Category: "PII", Type: "CREDIT_CARD", Severity: "CRITICAL"},
	}
	ccDecision := Evaluate(ccFindings)
	if ccDecision.Action != ActionBlock {
		t.Errorf("Expected ActionBlock for Credit Card, got %s", ccDecision.Action)
	}

	// 3. Email only must MASK
	emailFindings := []security.Finding{
		{Category: "PII", Type: "EMAIL", Severity: "INFO"},
	}
	emailDecision := Evaluate(emailFindings)
	if emailDecision.Action != ActionMask {
		t.Errorf("Expected ActionMask for Email, got %s", emailDecision.Action)
	}

	// 4. Empty findings must ALLOW
	cleanDecision := Evaluate(nil)
	if cleanDecision.Action != ActionAllow {
		t.Errorf("Expected ActionAllow for clean payload, got %s", cleanDecision.Action)
	}
}
