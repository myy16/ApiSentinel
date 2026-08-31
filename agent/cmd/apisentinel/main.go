package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/apisentinel/apisentinel/agent/internal/client"
	"github.com/apisentinel/apisentinel/agent/internal/git"
	"github.com/apisentinel/apisentinel/internal/agent"
	securityv1 "github.com/apisentinel/apisentinel/pkg/genproto/proto/security/v1"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	stagedOnly bool
	targetPath string
	serverAddr string = "localhost:50051"
	agentID    string
	agentToken string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "apisentinel",
		Short: "ApiSentinel Local Security & Secret Scanner CLI",
		Long:  "High-performance local git hook scanner and secret detector for developers.",
	}

	var scanCmd = &cobra.Command{
		Use:   "scan",
		Short: "Scan files or git diff for secrets and sensitive data",
		Run:   runScan,
	}
	scanCmd.Flags().BoolVarP(&stagedOnly, "staged", "s", false, "Scan only staged git changes")
	scanCmd.Flags().StringVarP(&targetPath, "path", "p", ".", "Directory or file path to scan")
	scanCmd.Flags().StringVarP(&serverAddr, "server", "S", "localhost:50051", "ApiSentinel Cloud gRPC server address")
	scanCmd.Flags().StringVarP(&agentToken, "token", "t", "", "Agent Authentication Token (or set APISENTINEL_TOKEN env var)")
	scanCmd.Flags().StringVarP(&agentID, "agent-id", "a", "", "Custom Agent ID")

	var hookType string

	var installHookCmd = &cobra.Command{
		Use:   "install-hook",
		Short: "Install ApiSentinel git hook (pre-commit or pre-push) into current repository",
		Run: func(cmd *cobra.Command, args []string) {
			if err := git.InstallHookWithType(hookType); err != nil {
				color.Red("❌ Failed to install git hook: %v", err)
				os.Exit(1)
			}
			color.Green("✅ ApiSentinel %s git hook successfully installed in .git/hooks/%s", hookType, hookType)
		},
	}
	installHookCmd.Flags().StringVarP(&hookType, "type", "T", "pre-push", "Hook type: 'pre-push' (default) or 'pre-commit'")

	var uninstallHookCmd = &cobra.Command{
		Use:   "uninstall-hook",
		Short: "Uninstall ApiSentinel git hook",
		Run: func(cmd *cobra.Command, args []string) {
			if err := git.UninstallHookWithType(hookType); err != nil {
				color.Red("❌ Failed to uninstall git hook: %v", err)
				os.Exit(1)
			}
			color.Green("✅ ApiSentinel %s git hook removed.", hookType)
		},
	}
	uninstallHookCmd.Flags().StringVarP(&hookType, "type", "T", "pre-push", "Hook type: 'pre-push' (default) or 'pre-commit'")

	var connectCmd = &cobra.Command{
		Use:   "connect",
		Short: "Connect Agent to ApiSentinel Cloud via persistent gRPC tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			tok := agentToken
			if tok == "" {
				tok = os.Getenv("APISENTINEL_TOKEN")
			}

			c := client.NewAgentClient(serverAddr, agentID, tok)
			if err := c.Connect(ctx); err != nil {
				color.Red("❌ Agent connection error: %v", err)
				os.Exit(1)
			}
		},
	}
	connectCmd.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "ApiSentinel Cloud gRPC server address")
	connectCmd.Flags().StringVarP(&agentID, "agent-id", "a", "", "Custom Agent ID (defaults to hostname)")
	connectCmd.Flags().StringVarP(&agentToken, "token", "t", "", "Agent Authentication Token (or set APISENTINEL_TOKEN env var)")

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Display local agent and git hook status",
		Run: func(cmd *cobra.Command, args []string) {
			color.Cyan("🛡️  ApiSentinel Local Agent v0.1.0 (Go-Centric)")
			gitRoot, err := git.FindGitRoot()
			if err != nil {
				color.Yellow("⚠️  Not inside a Git repository")
				return
			}
			color.Green("📂 Git Root: %s", gitRoot)

			prePushPath := filepath.Join(gitRoot, ".git", "hooks", "pre-push")
			if _, err := os.Stat(prePushPath); err == nil {
				color.Green("🔒 Pre-push hook: ACTIVE")
			} else {
				color.Yellow("🔓 Pre-push hook: NOT INSTALLED (Run 'apisentinel install-hook --type=pre-push')")
			}

			preCommitPath := filepath.Join(gitRoot, ".git", "hooks", "pre-commit")
			if _, err := os.Stat(preCommitPath); err == nil {
				color.Green("🔒 Pre-commit hook: ACTIVE")
			} else {
				color.Yellow("🔓 Pre-commit hook: NOT INSTALLED (Run 'apisentinel install-hook --type=pre-commit')")
			}
		},
	}

	rootCmd.AddCommand(scanCmd, installHookCmd, uninstallHookCmd, connectCmd, statusCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func shouldIgnoreDir(name string) bool {
	ignored := map[string]bool{
		".git":         true,
		"node_modules": true,
		".next":        true,
		"dist":         true,
		"bin":          true,
		".drizzle":     true,
		"coverage":     true,
	}
	return ignored[name]
}

func shouldIgnoreFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)

	if ext == ".exe" || ext == ".pack" || ext == ".trace" || ext == ".lock" || ext == ".map" {
		return true
	}
	if base == "package-lock.json" || base == "pnpm-lock.yaml" || base == "yarn.lock" {
		return true
	}
	return false
}

func runScan(cmd *cobra.Command, args []string) {
	scanner := agent.NewLocalScanner()
	color.Cyan("🔍 ApiSentinel scanning in progress...")

	var criticalCount int
	var allFindings []agent.FileFinding

	if stagedOnly {
		findings, err := scanner.ScanGitStaged(cmd.Context())
		if err != nil {
			color.Red("❌ Failed to scan git staged changes: %v", err)
			os.Exit(1)
		}
		allFindings = findings
	} else {
		fi, err := os.Stat(targetPath)
		if err != nil {
			color.Red("❌ Target path error: %v", err)
			os.Exit(1)
		}

		if fi.IsDir() {
			findings, err := scanner.ScanDirectory(cmd.Context(), targetPath)
			if err != nil {
				color.Red("❌ Directory scan error: %v", err)
				os.Exit(1)
			}
			allFindings = findings
		} else {
			findings, err := scanner.ScanFile(cmd.Context(), targetPath)
			if err != nil {
				color.Red("❌ File scan error: %v", err)
				os.Exit(1)
			}
			allFindings = findings
		}
	}

	for _, ff := range allFindings {
		if ff.Finding.Severity == "CRITICAL" || ff.Finding.Severity == "HIGH" {
			criticalCount++
		}
	}

	printFileFindings(allFindings)

	// Sync findings to ApiSentinel Cloud Dashboard if token is configured (#1.1, #1.2, #1.3)
	tok := agentToken
	if tok == "" {
		tok = os.Getenv("APISENTINEL_TOKEN")
	}
	if tok != "" {
		repo, branch, commit := getGitRepoInfo()
		if repo == "" {
			repo = "local-repo"
		}
		protoFindings := convertFileFindingsToProto(allFindings)
		c := client.NewAgentClient(serverAddr, agentID, tok)
		syncCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := c.SyncScanResults(syncCtx, repo, branch, commit, protoFindings)
		if err != nil {
			color.Yellow("⚠️  Failed to sync scan results with Cloud: %v", err)
		} else if !resp.Accepted {
			color.Yellow("⚠️  Cloud rejected scan results: %s", resp.Message)
		} else {
			color.Cyan("☁️  Scan results successfully synced to ApiSentinel Cloud Dashboard (%s)", resp.Action)
		}
	}

	if criticalCount > 0 {
		color.Red("\n🚨 Scan finished: %d CRITICAL / HIGH security finding(s) detected!", criticalCount)
		os.Exit(1)
	} else {
		color.Green("\n✅ Clean! No secrets or critical sensitive data detected.")
	}
}

func getGitRepoInfo() (repo, branch, commit string) {
	root, err := git.FindGitRoot()
	if err == nil && root != "" {
		repo = filepath.Base(root)
	}
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err == nil {
		branch = strings.TrimSpace(string(out))
	}
	commitOut, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err == nil {
		commit = strings.TrimSpace(string(commitOut))
	}
	return
}

func convertFileFindingsToProto(findings []agent.FileFinding) []*securityv1.SecurityFinding {
	var protoList []*securityv1.SecurityFinding
	for _, ff := range findings {
		f := ff.Finding
		var sev securityv1.Severity
		switch f.Severity {
		case "CRITICAL":
			sev = securityv1.Severity_SEVERITY_CRITICAL
		case "HIGH":
			sev = securityv1.Severity_SEVERITY_HIGH
		case "MEDIUM":
			sev = securityv1.Severity_SEVERITY_MEDIUM
		case "LOW":
			sev = securityv1.Severity_SEVERITY_LOW
		default:
			sev = securityv1.Severity_SEVERITY_INFO
		}

		protoList = append(protoList, &securityv1.SecurityFinding{
			Category:       f.Category,
			Type:           f.Type,
			Severity:       sev,
			FieldPath:      fmt.Sprintf("%s:%d", ff.FilePath, ff.LineNumber),
			FilePath:       ff.FilePath,
			LineNumber:     int32(ff.LineNumber),
			Message:        f.Message,
			EvidenceMasked: f.EvidenceMasked,
			Confidence:     f.Confidence,
		})
	}
	return protoList
}

func printFileFindings(findings []agent.FileFinding) {
	if len(findings) == 0 {
		return
	}
	for _, ff := range findings {
		f := ff.Finding
		fmt.Println()
		if f.Severity == "CRITICAL" || f.Severity == "HIGH" {
			color.Red("❌ [%s] %s in %s:%d", f.Severity, f.Type, ff.FilePath, ff.LineNumber)
		} else {
			color.Yellow("⚠️  [%s] %s in %s:%d", f.Severity, f.Type, ff.FilePath, ff.LineNumber)
		}
		color.White("   Message:  %s", f.Message)
		color.White("   Evidence: %s", f.EvidenceMasked)
		if ff.MatchedSnippet != "" {
			color.White("   Snippet:  %s", ff.MatchedSnippet)
		}
	}
}
