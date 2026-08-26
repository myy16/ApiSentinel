package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalScanner_ScanFile(t *testing.T) {
	scanner := NewLocalScanner()
	tmpDir := t.TempDir()

	// 1. Create a file containing a simulated high-entropy secret
	testFile := filepath.Join(tmpDir, "config.go")
	content := []byte(`package main

var (
    custom_api_key = "K9#mQ2$zL8!vX4@pW7*jR1"
    appName = "test-service"
)
`)
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	findings, err := scanner.ScanFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("unexpected error scanning file: %v", err)
	}

	if len(findings) == 0 {
		t.Fatalf("expected finding for exposed secret in file, got none")
	}

	if findings[0].LineNumber != 4 {
		t.Errorf("expected line number 4, got %d", findings[0].LineNumber)
	}

	// 2. Clean file
	cleanFile := filepath.Join(tmpDir, "clean.go")
	cleanContent := []byte(`package main
func Hello() string { return "world" }
`)
	if err := os.WriteFile(cleanFile, cleanContent, 0644); err != nil {
		t.Fatalf("failed to write clean file: %v", err)
	}

	cleanFindings, err := scanner.ScanFile(context.Background(), cleanFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cleanFindings) != 0 {
		t.Errorf("clean file produced unexpected findings: %+v", cleanFindings)
	}
}
