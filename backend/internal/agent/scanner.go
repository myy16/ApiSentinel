package agent

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/apisentinel/apisentinel/internal/security"
)

// FileFinding represents a security finding located in a specific file and line
type FileFinding struct {
	FilePath       string           `json:"filePath"`
	LineNumber     int              `json:"lineNumber"`
	Finding        security.Finding `json:"finding"`
	MatchedSnippet string           `json:"matchedSnippet"`
}

// LocalScanner performs security scans on local filesystem and git staged changes
type LocalScanner struct {
	engine *security.Engine
}

func NewLocalScanner() *LocalScanner {
	return &LocalScanner{
		engine: security.NewEngine(),
	}
}

// ScanFile scans a single file line by line
func (s *LocalScanner) ScanFile(ctx context.Context, path string) ([]FileFinding, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Check if file is binary
	if isBinaryFile(file) {
		return nil, nil
	}

	_, _ = file.Seek(0, 0)
	scanner := bufio.NewScanner(file)
	var findings []FileFinding
	lineNum := 0

	for scanner.Scan() {
		if ctx.Err() != nil {
			break
		}
		lineNum++
		line := scanner.Text()
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}

		lineFindings := s.engine.Inspect(line)
		for _, f := range lineFindings {
			// In local code scanning, SECRET and PII are critical
			if f.Category == "SECRET" || f.Category == "PII" || f.Category == "INJECTION" {
				snippet := line
				if len(snippet) > 80 {
					snippet = snippet[:77] + "..."
				}
				findings = append(findings, FileFinding{
					FilePath:       path,
					LineNumber:     lineNum,
					Finding:        f,
					MatchedSnippet: strings.TrimSpace(snippet),
				})
			}
		}
	}

	return findings, scanner.Err()
}

// ScanDirectory scans all code files in a directory recursively
func (s *LocalScanner) ScanDirectory(ctx context.Context, rootDir string) ([]FileFinding, error) {
	var allFindings []FileFinding

	ignoredDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		"vendor":       true,
		".next":        true,
		"dist":         true,
		"build":        true,
		".cache":       true,
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			if ignoredDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip large files (> 5MB)
		if info.Size() > 5*1024*1024 {
			return nil
		}

		fileFindings, _ := s.ScanFile(ctx, path)
		if len(fileFindings) > 0 {
			allFindings = append(allFindings, fileFindings...)
		}
		return nil
	})

	return allFindings, err
}

// ScanGitStaged scans files currently staged for commit using git diff --cached
func (s *LocalScanner) ScanGitStaged(ctx context.Context) ([]FileFinding, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--cached", "--unified=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff --cached failed: %s", stderr.String())
	}

	var findings []FileFinding
	scanner := bufio.NewScanner(&stdout)
	currentFile := ""
	currentLine := 1

	for scanner.Scan() {
		line := scanner.Text()

		// Diff file header: +++ b/path/to/file.go
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			continue
		}

		// Diff hunk header: @@ -10,0 +15,5 @@
		if strings.HasPrefix(line, "@@") {
			_, _ = fmt.Sscanf(line, "@@ %*s +%d", &currentLine)
			continue
		}

		// Added or modified line
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			addedContent := strings.TrimPrefix(line, "+")
			lineFindings := s.engine.Inspect(addedContent)
			for _, f := range lineFindings {
				if f.Category == "SECRET" || f.Category == "PII" {
					snippet := addedContent
					if len(snippet) > 80 {
						snippet = snippet[:77] + "..."
					}
					findings = append(findings, FileFinding{
						FilePath:       currentFile,
						LineNumber:     currentLine,
						Finding:        f,
						MatchedSnippet: strings.TrimSpace(snippet),
					})
				}
			}
			currentLine++
		}
	}

	return findings, nil
}

func isBinaryFile(r io.Reader) bool {
	buf := make([]byte, 512)
	n, err := r.Read(buf)
	if err != nil && err != io.EOF {
		return false
	}
	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return true
		}
	}
	return false
}
