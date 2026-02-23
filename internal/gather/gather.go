package gather

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const defaultMaxKB = 50

// Gather collects context from the specified file paths and git diffs.
// Files are read first (user content gets priority), then git diffs fill
// remaining budget up to maxKB kilobytes.
func Gather(patterns []string, maxKB int, workDir string) (string, error) {
	if maxKB <= 0 {
		maxKB = defaultMaxKB
	}
	maxBytes := maxKB * 1024

	var parts []string
	totalBytes := 0

	for _, p := range patterns {
		if totalBytes >= maxBytes {
			break
		}
		fullPath := filepath.Join(workDir, p)
		info, err := os.Stat(fullPath)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if int(info.Size()) > maxBytes-totalBytes {
			continue
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("#### %s\n\n```\n%s\n```", p, string(data)))
		totalBytes += len(data)
	}

	if len(parts) > 0 {
		parts = append([]string{"### Files Referenced\n"}, parts...)
	}

	if totalBytes < maxBytes {
		diff := gitDiff(workDir)
		if diff != "" {
			diffBytes := len(diff)
			if totalBytes+diffBytes <= maxBytes {
				parts = append(parts, "### Recent Changes (Git Diff)\n\n```diff\n"+diff+"\n```")
			} else {
				remaining := maxBytes - totalBytes
				truncated := diff[:remaining]
				for !utf8.ValidString(truncated) && len(truncated) > 0 {
					truncated = truncated[:len(truncated)-1]
				}
				parts = append(parts, "### Recent Changes (Git Diff) [truncated]\n\n```diff\n"+truncated+"\n```")
			}
		}
	}

	return strings.Join(parts, "\n\n"), nil
}

func gitDiff(workDir string) string {
	staged := runGit(workDir, "diff", "--staged")
	unstaged := runGit(workDir, "diff")

	var parts []string
	if staged != "" {
		parts = append(parts, staged)
	}
	if unstaged != "" {
		parts = append(parts, unstaged)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

func runGit(workDir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// BuildPrompt wraps a user prompt with optional context.
func BuildPrompt(prompt, context string) string {
	var b strings.Builder
	b.WriteString("# Second Opinion Request\n\n")
	b.WriteString("## Question\n\n")
	b.WriteString(prompt)
	if context != "" {
		b.WriteString("\n\n## Context\n\n")
		b.WriteString(context)
	}
	b.WriteString("\n\n## Instructions\n\n")
	b.WriteString("You are providing an independent second opinion. Be critical and thorough.\n")
	b.WriteString("- Analyze the question in the context provided\n")
	b.WriteString("- Identify risks, tradeoffs, and blind spots\n")
	b.WriteString("- Suggest alternatives if you see better approaches\n")
	b.WriteString("- Be direct and opinionated — don't hedge\n")
	b.WriteString("- Structure your response with clear headings\n")
	b.WriteString("- Keep your response focused and actionable\n")
	return b.String()
}
