package security

import "context"

// Finding represents a single security observation produced by a Scanner.
type Finding struct {
	Category       string  `json:"category"`
	Type           string  `json:"type"`
	Severity       string  `json:"severity"`
	Message        string  `json:"message"`
	EvidenceMasked string  `json:"evidence_masked"`
	Confidence     float64 `json:"confidence"`
}

// Scanner defines the common pluggable interface for all security inspection modules.
type Scanner interface {
	// Name returns a human-readable identifier for the scanner (e.g. "pii_scanner", "secret_scanner")
	Name() string
	// Category returns the broad classification (e.g. "PII", "SECRET", "INJECTION")
	Category() string
	// Scan analyzes the given payload string within the provided context and returns any findings.
	Scan(ctx context.Context, payload string) []Finding
}
