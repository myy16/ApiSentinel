package security

import (
	"github.com/apisentinel/apisentinel/internal/security/pii"
	"github.com/apisentinel/apisentinel/internal/security/secret"
)

type Finding struct {
	Category       string  `json:"category"`
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	Message        string  `json:"message"`
	EvidenceMasked string  `json:"evidence_masked"`
	Confidence     float64 `json:"confidence"`
}

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

// InspectText runs PII and Secret scanners in parallel/sequence
func (e *Engine) Inspect(rawPayload string) []Finding {
	var results []Finding

	// 1. PII Scan
	piiFindings := pii.ScanText(rawPayload)
	for _, f := range piiFindings {
		results = append(results, Finding{
			Category:       "PII",
			Type:           f.Type,
			Severity:       f.Severity,
			Message:        f.Message,
			EvidenceMasked: f.EvidenceMasked,
			Confidence:     f.Confidence,
		})
	}

	// 2. Secret Scan
	secretFindings := secret.ScanText(rawPayload)
	for _, f := range secretFindings {
		results = append(results, Finding{
			Category:       "SECRET",
			Type:           f.Type,
			Severity:       f.Severity,
			Message:        f.Message,
			EvidenceMasked: f.EvidenceMasked,
			Confidence:     f.Confidence,
		})
	}

	return results
}
