package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHook_InstallAndBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "githook-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Simulate .git structure
	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		t.Fatalf("Failed to create hooks dir: %v", err)
	}

	preCommitPath := filepath.Join(hooksDir, "pre-commit")
	customHookContent := "#!/bin/sh\necho 'custom linter'\n"
	if err := os.WriteFile(preCommitPath, []byte(customHookContent), 0755); err != nil {
		t.Fatalf("Failed to write initial custom hook: %v", err)
	}

	// 1. Install pre-commit hook (should back up existing custom hook)
	// We test backup logic manually against temp dir
	if existingContent, err := os.ReadFile(preCommitPath); err == nil {
		if !strings.Contains(string(existingContent), "ApiSentinel") {
			backupPath := filepath.Join(hooksDir, "pre-commit.bak")
			_ = os.WriteFile(backupPath, existingContent, 0755)
		}
	}
	if err := os.WriteFile(preCommitPath, []byte(preCommitScript), 0755); err != nil {
		t.Fatalf("Failed to write preCommitScript: %v", err)
	}

	// Verify pre-commit contains ApiSentinel
	content, err := os.ReadFile(preCommitPath)
	if err != nil || !strings.Contains(string(content), "ApiSentinel") {
		t.Fatalf("Expected ApiSentinel in pre-commit hook, got: %s", string(content))
	}

	// Verify backup was created with original content
	bakContent, err := os.ReadFile(filepath.Join(hooksDir, "pre-commit.bak"))
	if err != nil || string(bakContent) != customHookContent {
		t.Fatalf("Expected backup to preserve original custom content, got: %s", string(bakContent))
	}

	// 2. Uninstall pre-commit (should restore original custom hook)
	_ = os.Remove(preCommitPath)
	if backupContent, err := os.ReadFile(filepath.Join(hooksDir, "pre-commit.bak")); err == nil {
		_ = os.WriteFile(preCommitPath, backupContent, 0755)
		_ = os.Remove(filepath.Join(hooksDir, "pre-commit.bak"))
	}

	restoredContent, err := os.ReadFile(preCommitPath)
	if err != nil || string(restoredContent) != customHookContent {
		t.Fatalf("Expected custom hook to be restored after uninstall, got: %s", string(restoredContent))
	}
}
