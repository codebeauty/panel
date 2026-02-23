package output

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var headingRe = regexp.MustCompile(`^#{1,3}\s+(.+)`)

// BuildSummary generates a heuristic (no LLM) markdown summary of a run.
func BuildSummary(manifest *Manifest, runDir string) string {
	var b strings.Builder

	b.WriteString("# Run Summary\n\n")

	// Prompt (truncated to 100 chars)
	prompt := manifest.Prompt
	if len(prompt) > 100 {
		prompt = prompt[:100] + "..."
	}
	fmt.Fprintf(&b, "**Prompt:** %s\n", prompt)

	// Tools list
	toolIDs := make([]string, len(manifest.Results))
	for i, r := range manifest.Results {
		toolIDs[i] = r.ToolID
	}
	fmt.Fprintf(&b, "**Tools:** %s\n", strings.Join(toolIDs, ", "))

	// Policy
	fmt.Fprintf(&b, "**Policy:** read-only=%s\n", manifest.Config.ReadOnly)

	// Results section
	b.WriteString("\n## Results\n")

	for _, r := range manifest.Results {
		icon := statusIcon(r.Status)
		fmt.Fprintf(&b, "\n### %s %s\n", icon, r.ToolID)
		if r.Expert != "" {
			fmt.Fprintf(&b, "- Expert: %s\n", r.Expert)
		}
		fmt.Fprintf(&b, "- Status: %s\n", r.Status)
		fmt.Fprintf(&b, "- Duration: %s\n", r.Duration)

		if r.ExitCode != 0 {
			fmt.Fprintf(&b, "- Exit code: %d\n", r.ExitCode)
		}

		// Read the output file for word count and headings
		outputPath := filepath.Join(runDir, r.OutputFile)
		if content, err := os.ReadFile(outputPath); err == nil {
			words := len(strings.Fields(string(content)))
			fmt.Fprintf(&b, "- Word count: %d\n", words)

			headings := extractHeadings(outputPath)
			if len(headings) > 0 {
				b.WriteString("- Key sections:\n")
				for _, h := range headings {
					fmt.Fprintf(&b, "  - %s\n", h)
				}
			}
		}

		if r.Status != "success" {
			if r.ExitCode != 0 {
				fmt.Fprintf(&b, "- Error: exit code %d\n", r.ExitCode)
			} else {
				fmt.Fprintf(&b, "- Error: %s\n", r.Status)
			}
		}
	}

	// Cost summary table
	hasCost := false
	for _, r := range manifest.Results {
		if r.Cost != nil && (r.Cost.InputTokens > 0 || r.Cost.OutputTokens > 0 || r.Cost.TotalUSD > 0) {
			hasCost = true
			break
		}
	}
	if hasCost {
		b.WriteString("\n## Cost Summary\n")
		b.WriteString("| Tool | Input | Output | Cost |\n")
		b.WriteString("|------|-------|--------|------|\n")
		for _, r := range manifest.Results {
			if r.Cost != nil {
				fmt.Fprintf(&b, "| %s | %d | %d | $%.2f |\n",
					r.ToolID, r.Cost.InputTokens, r.Cost.OutputTokens, r.Cost.TotalUSD)
			}
		}
	}

	return b.String()
}

// WriteSummary writes summary.md atomically to the given directory.
func WriteSummary(dir, content string) error {
	return AtomicWrite(filepath.Join(dir, "summary.md"), []byte(content), 0o600)
}

// extractHeadings reads a file and returns up to 10 markdown headings (h1-h3).
func extractHeadings(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var headings []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := headingRe.FindStringSubmatch(scanner.Text()); matches != nil {
			headings = append(headings, matches[1])
			if len(headings) >= 10 {
				break
			}
		}
	}
	return headings
}

// statusIcon returns a unicode icon for the given result status.
func statusIcon(status string) string {
	switch status {
	case "success":
		return "✓"
	case "timeout":
		return "⏱"
	default:
		return "✗"
	}
}
