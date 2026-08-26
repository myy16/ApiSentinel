package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/apisentinel/apisentinel/internal/agent"
)

const (
	Version   = "0.2.0"
	BannerArt = `
   ___           _  ____             __   _            __ 
  / _ | ___  _  (_)/ __/ ___  ___   / /_ (_)___  ___  / / 
 / __ |/ _ \| |/ /_\ \  / -_)/ _ \ / __// // _ \/ -_)/ /  
/_/ |_/ .__/|___//___/  \__//_//_/ \__//_//_//_/\__//_/   
     /_/       [ Shift-Left Security & Local Agent ]
`
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "version", "-v", "--version":
		fmt.Printf("ApiSentinel Agent CLI v%s\n", Version)

	case "install-hook":
		runInstallHook()

	case "pre-commit":
		runPreCommit()

	case "scan":
		runScan(os.Args[2:])

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(BannerArt)
	fmt.Printf("ApiSentinel Local Agent & Shift-Left Security CLI (v%s)\n\n", Version)
	fmt.Println("Usage:")
	fmt.Println("  apisentinel scan [path]          Scan local directory or file for secrets and PII")
	fmt.Println("  apisentinel scan --staged        Scan git staged changes before commit")
	fmt.Println("  apisentinel pre-commit           Git pre-commit hook runner (exits 1 if secrets found)")
	fmt.Println("  apisentinel install-hook         Install pre-commit hook into current git repository")
	fmt.Println("  apisentinel version              Show CLI version")
	fmt.Println("")
}

func runInstallHook() {
	targetDir := "."
	if len(os.Args) > 2 {
		targetDir = os.Args[2]
	}

	hookPath, err := agent.InstallGitHook(targetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error installing hook: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ ApiSentinel Git pre-commit hook successfully installed!")
	fmt.Printf("   Hook location: %s\n", hookPath)
	fmt.Println("   Now every `git commit` will automatically scan for exposed secrets before committing.")
}

func runPreCommit() {
	scanner := agent.NewLocalScanner()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("🔍 [ApiSentinel] Scanning git staged files for secrets & credentials...")

	findings, err := scanner.ScanGitStaged(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Warning: git staged scan failed: %v\n", err)
		os.Exit(0) // Don't block if git diff is unavailable
	}

	if len(findings) == 0 {
		fmt.Println("✅ [ApiSentinel] No secrets or credentials found in staged changes. Commit allowed!")
		os.Exit(0)
	}

	// Print violations and BLOCK commit
	fmt.Println("\n🚨 \033[1;31m[ApiSentinel Security Alert] Staged commit contains exposed secrets!\033[0m")
	fmt.Println(strings.Repeat("─", 75))

	hasCritical := false
	for _, f := range findings {
		sevColor := "\033[1;33m" // Yellow
		if f.Finding.Severity == "CRITICAL" || f.Finding.Severity == "HIGH" {
			sevColor = "\033[1;31m" // Red
			hasCritical = true
		}

		fmt.Printf(" %s[%s]\033[0m %s:%d\n", sevColor, f.Finding.Type, f.FilePath, f.LineNumber)
		fmt.Printf("   Message: %s\n", f.Finding.Message)
		fmt.Printf("   Evidence: %s\n", f.Finding.EvidenceMasked)
		fmt.Printf("   Snippet:  %s\n\n", f.MatchedSnippet)
	}
	fmt.Println(strings.Repeat("─", 75))

	if hasCritical {
		fmt.Println("🛑 \033[1;31mCOMMIT BLOCKED: Please remove the secrets above before committing.\033[0m")
		os.Exit(1)
	}

	os.Exit(0)
}

func runScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	staged := fs.Bool("staged", false, "Scan git staged changes only")
	_ = fs.Parse(args)

	if *staged {
		runPreCommit()
		return
	}

	targetPath := "."
	if len(fs.Args()) > 0 {
		targetPath = fs.Args()[0]
	}

	scanner := agent.NewLocalScanner()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fmt.Printf("🔍 Scanning target path: %s ...\n", targetPath)
	startTime := time.Now()

	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Path not found: %s\n", targetPath)
		os.Exit(1)
	}

	var findings []agent.FileFinding
	if fileInfo.IsDir() {
		findings, err = scanner.ScanDirectory(ctx, targetPath)
	} else {
		findings, err = scanner.ScanFile(ctx, targetPath)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Scan error: %v\n", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("⏱️ Scan completed in %v\n\n", duration.Round(time.Millisecond))

	if len(findings) == 0 {
		fmt.Println("✅ Clean! No secrets, credentials, or PII vulnerabilities detected.")
		os.Exit(0)
	}

	fmt.Printf("🚨 Found %d potential security finding(s):\n", len(findings))
	fmt.Println(strings.Repeat("─", 75))

	for _, f := range findings {
		sevColor := "\033[1;33m"
		if f.Finding.Severity == "CRITICAL" || f.Finding.Severity == "HIGH" {
			sevColor = "\033[1;31m"
		}
		fmt.Printf(" %s[%s - %s]\033[0m %s:%d\n", sevColor, f.Finding.Category, f.Finding.Type, f.FilePath, f.LineNumber)
		fmt.Printf("   Message:  %s\n", f.Finding.Message)
		fmt.Printf("   Evidence: %s\n", f.Finding.EvidenceMasked)
		fmt.Printf("   Snippet:  %s\n\n", f.MatchedSnippet)
	}
	fmt.Println(strings.Repeat("─", 75))
}
