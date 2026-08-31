package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const preCommitScript = `#!/bin/sh
# ApiSentinel Security Scanner Pre-Commit Hook
echo "🔍 [ApiSentinel] Scanning staged files for secrets and sensitive data..."

apisentinel scan --staged
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
    echo "❌ [ApiSentinel] Git commit BLOCKED! Critical secrets or sensitive data detected."
    echo "💡 Fix the leaks or review with 'apisentinel scan --staged'."
    exit 1
fi

echo "✅ [ApiSentinel] Security scan passed cleanly. Proceeding with commit."
exit 0
`

const prePushScript = `#!/bin/sh
# ApiSentinel Security Scanner Pre-Push Hook
echo "🔍 [ApiSentinel] Scanning commits for secrets and sensitive data..."

apisentinel scan --staged
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
    echo "❌ [ApiSentinel] Git push BLOCKED! Critical secrets or sensitive data detected."
    echo "💡 Fix the leaks or review with 'apisentinel scan'."
    exit 1
fi

echo "✅ [ApiSentinel] Security scan passed cleanly. Proceeding with push."
exit 0
`

func FindGitRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func InstallHook() error {
	return InstallHookWithType("pre-push")
}

func InstallHookWithType(hookType string) error {
	hookType = strings.ToLower(strings.TrimSpace(hookType))
	if hookType != "pre-commit" && hookType != "pre-push" {
		return fmt.Errorf("invalid hook type: %s (must be 'pre-commit' or 'pre-push')", hookType)
	}

	gitRoot, err := FindGitRoot()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hookPath := filepath.Join(hooksDir, hookType)

	// If existing hook file exists, check if it's already ApiSentinel. If not, back it up (#11)
	if existingContent, err := os.ReadFile(hookPath); err == nil {
		if !strings.Contains(string(existingContent), "ApiSentinel") {
			timestamp := time.Now().Format("20060102150405")
			backupTimestampPath := filepath.Join(hooksDir, fmt.Sprintf("%s.bak.%s", hookType, timestamp))
			backupDefaultPath := filepath.Join(hooksDir, fmt.Sprintf("%s.bak", hookType))
			_ = os.WriteFile(backupTimestampPath, existingContent, 0755)
			_ = os.WriteFile(backupDefaultPath, existingContent, 0755)
		}
	}

	scriptContent := prePushScript
	if hookType == "pre-commit" {
		scriptContent = preCommitScript
	}

	if err := os.WriteFile(hookPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to write git hook: %w", err)
	}

	return nil
}

func UninstallHook() error {
	return UninstallHookWithType("pre-push")
}

func UninstallHookWithType(hookType string) error {
	hookType = strings.ToLower(strings.TrimSpace(hookType))
	if hookType != "pre-commit" && hookType != "pre-push" {
		return fmt.Errorf("invalid hook type: %s (must be 'pre-commit' or 'pre-push')", hookType)
	}

	gitRoot, err := FindGitRoot()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	hookPath := filepath.Join(hooksDir, hookType)

	if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove git hook: %w", err)
	}

	// If a backup exists, restore it (#11)
	backupDefaultPath := filepath.Join(hooksDir, fmt.Sprintf("%s.bak", hookType))
	if backupContent, err := os.ReadFile(backupDefaultPath); err == nil {
		_ = os.WriteFile(hookPath, backupContent, 0755)
		_ = os.Remove(backupDefaultPath)
	}

	return nil
}

func GetStagedDiff() (string, error) {
	out, err := exec.Command("git", "diff", "--cached").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
