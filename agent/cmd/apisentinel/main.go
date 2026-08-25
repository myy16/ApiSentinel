package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/apisentinel/apisentinel/agent/internal/client"
	"github.com/apisentinel/apisentinel/agent/internal/git"
	"github.com/apisentinel/apisentinel/internal/security"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	stagedOnly bool
	targetPath string
	serverAddr string
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

	var installHookCmd = &cobra.Command{
		Use:   "install-hook",
		Short: "Install ApiSentinel pre-push git hook into current repository",
		Run: func(cmd *cobra.Command, args []string) {
			if err := git.InstallHook(); err != nil {
				color.Red("❌ Failed to install git hook: %v", err)
				os.Exit(1)
			}
			color.Green("✅ ApiSentinel pre-push git hook successfully installed in .git/hooks/pre-push")
		},
	}

	var uninstallHookCmd = &cobra.Command{
		Use:   "uninstall-hook",
		Short: "Uninstall ApiSentinel git hook",
		Run: func(cmd *cobra.Command, args []string) {
			if err := git.UninstallHook(); err != nil {
				color.Red("❌ Failed to uninstall git hook: %v", err)
				os.Exit(1)
			}
			color.Green("✅ ApiSentinel git hook removed.")
		},
	}

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
			hookPath := filepath.Join(gitRoot, ".git", "hooks", "pre-push")
			if _, err := os.Stat(hookPath); err == nil {
				color.Green("🔒 Pre-push hook: ACTIVE")
			} else {
				color.Yellow("🔓 Pre-push hook: NOT INSTALLED (Run 'apisentinel install-hook' to protect commits)")
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
	engine := security.NewEngine()
	color.Cyan("🔍 ApiSentinel scanning in progress...")

	var criticalCount int

	if stagedOnly {
		diff, err := git.GetStagedDiff()
		if err != nil {
			color.Red("❌ Failed to read git staged diff: %v", err)
			os.Exit(1)
		}
		if len(diff) == 0 {
			color.Green("✨ No staged changes to scan.")
			return
		}

		findings := engine.Inspect(diff)
		for _, f := range findings {
			if f.Severity == "CRITICAL" || f.Severity == "HIGH" {
				criticalCount++
			}
		}
		printFindings("Git Staged Changes", findings)
	} else {
		err := filepath.Walk(targetPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if info.IsDir() {
				if shouldIgnoreDir(info.Name()) {
					return filepath.SkipDir
				}
				return nil
			}

			if shouldIgnoreFile(path) || info.Size() > 1024*1024 {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			findings := engine.Inspect(string(content))
			if len(findings) > 0 {
				for _, f := range findings {
					if f.Severity == "CRITICAL" || f.Severity == "HIGH" {
						criticalCount++
					}
				}
				printFindings(path, findings)
			}
			return nil
		})

		if err != nil {
			color.Red("❌ Scan error: %v", err)
			os.Exit(1)
		}
	}

	if criticalCount > 0 {
		color.Red("\n🚨 Scan finished: %d CRITICAL / HIGH security finding(s) detected!", criticalCount)
		os.Exit(1)
	} else {
		color.Green("\n✅ Clean! No secrets or critical sensitive data detected.")
	}
}

func printFindings(source string, findings []security.Finding) {
	for _, f := range findings {
		fmt.Println()
		if f.Severity == "CRITICAL" || f.Severity == "HIGH" {
			color.Red("❌ [%s] %s in %s", f.Severity, f.Type, source)
		} else {
			color.Yellow("⚠️  [%s] %s in %s", f.Severity, f.Type, source)
		}
		color.White("   Message:  %s", f.Message)
		color.White("   Evidence: %s", f.EvidenceMasked)
	}
}
