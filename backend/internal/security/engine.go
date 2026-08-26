package security

import (
	"context"
	"sync"

	"github.com/apisentinel/apisentinel/internal/security/injection"
	"github.com/apisentinel/apisentinel/internal/security/normalize"
	"github.com/apisentinel/apisentinel/internal/security/pii"
	"github.com/apisentinel/apisentinel/internal/security/secret"
)

// Engine coordinates execution across registered security scanners.
type Engine struct {
	mu       sync.RWMutex
	scanners []Scanner
}

// NewEngine creates a new Security Engine preloaded with default scanners.
func NewEngine() *Engine {
	e := &Engine{
		scanners: make([]Scanner, 0),
	}
	// Register default core scanners
	e.RegisterScanner(&piiScannerAdapter{})
	e.RegisterScanner(&secretScannerAdapter{})
	e.RegisterScanner(&injectionScannerAdapter{})
	return e
}

// RegisterScanner adds a new scanner to the engine in a thread-safe manner.
func (e *Engine) RegisterScanner(s Scanner) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.scanners = append(e.scanners, s)
}

// GetScanners returns a copy of currently registered scanners.
func (e *Engine) GetScanners() []Scanner {
	e.mu.RLock()
	defer e.mu.RUnlock()
	dst := make([]Scanner, len(e.scanners))
	copy(dst, e.scanners)
	return dst
}

// Inspect runs all registered scanners against the raw payload.
func (e *Engine) Inspect(rawPayload string) []Finding {
	return e.InspectWithContext(context.Background(), rawPayload)
}

// InspectWithContext runs all registered scanners with context cancellation and payload canonicalization support.
func (e *Engine) InspectWithContext(ctx context.Context, rawPayload string) []Finding {
	e.mu.RLock()
	scanners := e.scanners
	e.mu.RUnlock()

	var results []Finding
	seen := make(map[string]bool)

	addFindings := func(findings []Finding) {
		for _, f := range findings {
			key := f.Category + ":" + f.Type + ":" + f.Message
			if !seen[key] {
				seen[key] = true
				results = append(results, f)
			}
		}
	}

	// 1. Scan raw payload directly
	for _, scanner := range scanners {
		if ctx.Err() != nil {
			break
		}
		addFindings(scanner.Scan(ctx, rawPayload))
	}

	// 2. Scan canonicalized (decoded/de-obfuscated) payload if different
	canonical := normalize.Canonicalize(rawPayload)
	if canonical != rawPayload && ctx.Err() == nil {
		for _, scanner := range scanners {
			if ctx.Err() != nil {
				break
			}
			addFindings(scanner.Scan(ctx, canonical))
		}
	}

	return results
}

// --- Built-in Scanner Adapters ---

type piiScannerAdapter struct{}

func (p *piiScannerAdapter) Name() string     { return "pii_scanner" }
func (p *piiScannerAdapter) Category() string { return "PII" }
func (p *piiScannerAdapter) Scan(ctx context.Context, payload string) []Finding {
	raw := pii.ScanText(payload)
	findings := make([]Finding, len(raw))
	for i, f := range raw {
		findings[i] = Finding{
			Category:       "PII",
			Type:           f.Type,
			Severity:       f.Severity,
			Message:        f.Message,
			EvidenceMasked: f.EvidenceMasked,
			Confidence:     f.Confidence,
		}
	}
	return findings
}

type secretScannerAdapter struct{}

func (s *secretScannerAdapter) Name() string     { return "secret_scanner" }
func (s *secretScannerAdapter) Category() string { return "SECRET" }
func (s *secretScannerAdapter) Scan(ctx context.Context, payload string) []Finding {
	raw := secret.ScanText(payload)
	findings := make([]Finding, len(raw))
	for i, f := range raw {
		findings[i] = Finding{
			Category:       "SECRET",
			Type:           f.Type,
			Severity:       f.Severity,
			Message:        f.Message,
			EvidenceMasked: f.EvidenceMasked,
			Confidence:     f.Confidence,
		}
	}
	return findings
}

type injectionScannerAdapter struct{}

func (i *injectionScannerAdapter) Name() string     { return "injection_scanner" }
func (i *injectionScannerAdapter) Category() string { return "INJECTION" }
func (i *injectionScannerAdapter) Scan(ctx context.Context, payload string) []Finding {
	raw := injection.ScanText(payload)
	findings := make([]Finding, len(raw))
	for idx, f := range raw {
		findings[idx] = Finding{
			Category:       "INJECTION",
			Type:           f.Type,
			Severity:       f.Severity,
			Message:        f.Message,
			EvidenceMasked: f.EvidenceMasked,
			Confidence:     f.Confidence,
		}
	}
	return findings
}


