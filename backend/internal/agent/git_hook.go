package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

const preCommitHookContent = `#!/bin/sh
# ApiSentinel Shift-Left Pre-Commit Security Hook
# Automatically blocks secrets, keys, and PII before commit

apisentinel pre-commit
EXIT_CODE=$?

if [ $EXIT_CODE -ne 0 ]; then
    echo ""
    echo "\033[1;31m[ApiSentinel] Commit blocked! Please remove exposed secrets or credentials before committing.\033[0m"
    echo "\033[1;33mTo bypass in emergency (NOT recommended): git commit --no-verify\033[0m"
    exit 1
fi

exit 0
`

// InstallGitHook installs the ApiSentinel pre-commit hook in the current or target repository
func InstallGitHook(repoRoot string) (string, error) {
	if repoRoot == "" {
		repoRoot = "."
	}

	gitDir := filepath.Join(repoRoot, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		return "", fmt.Errorf("'.git' directory not found in %s. Please run inside a git repository", repoRoot)
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create hooks directory: %w", err)
	}

	hookPath := filepath.Join(hooksDir, "pre-commit")
	err := os.WriteFile(hookPath, []byte(preCommitHookContent), 0755)
	if err != nil {
		return "", fmt.Errorf("failed to write pre-commit hook: %w", err)
	}

	return hookPath, nil
}
