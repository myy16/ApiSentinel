package ai

import (
	"context"
	"testing"
)

func TestAIExplainer(t *testing.T) {
	explainer := NewExplainer("")

	// 1. Test AWS Key explanation
	exp, err := explainer.ExplainFinding(context.Background(), "SECRET", "AWS_KEY", "CRITICAL", "AKIA****1234", "AWS key exposed", "FULL_LOCAL")
	if err != nil {
		t.Fatalf("ExplainFinding failed: %v", err)
	}
	if exp.FindingType != "AWS_KEY" || len(exp.RemediationSteps) == 0 {
		t.Errorf("Invalid explanation: %+v", exp)
	}
	if exp.Provider != "Dahili Güvenlik Kural Motoru (Tam Yerel / Offline)" {
		t.Errorf("Expected local offline provider string, got: %s", exp.Provider)
	}

	// 2. Test Credit Card explanation
	ccExp, err := explainer.ExplainFinding(context.Background(), "PII", "CREDIT_CARD", "CRITICAL", "************0366", "Card exposed", "MASKED_CLOUD")
	if err != nil {
		t.Fatalf("ExplainFinding for CC failed: %v", err)
	}
	if ccExp.FindingType != "CREDIT_CARD" || len(ccExp.CodeSnippet) == 0 {
		t.Errorf("Invalid CC explanation: %+v", ccExp)
	}
}
