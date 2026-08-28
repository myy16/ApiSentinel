package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
	gitRoot, err := FindGitRoot()
	if err != nil {
		return err
	}

	hooksDir := filepath.Join(gitRoot, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hookPath := filepath.Join(hooksDir, "pre-push")

	// If existing hook file exists, check if it's already ApiSentinel. If not, back it up (#24)
	if existingContent, err := os.ReadFile(hookPath); err == nil {
		if !strings.Contains(string(existingContent), "ApiSentinel") {
			backupPath := filepath.Join(hooksDir, "pre-push.bak")
			_ = os.WriteFile(backupPath, existingContent, 0755)
		}
	}

	if err := os.WriteFile(hookPath, []byte(prePushScript), 0755); err != nil {
		return fmt.Errorf("failed to write git hook: %w", err)
	}

	return nil
}

func UninstallHook() error {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return err
	}

	hookPath := filepath.Join(gitRoot, ".git", "hooks", "pre-push")
	if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove git hook: %w", err)
	}

	return nil
}

func GetStagedDiff() (string, error) {
	// Scan all staged files across the repository without exclusions
	out, err := exec.Command("git", "diff", "--cached").Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}
