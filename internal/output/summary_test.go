package output

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/codebeauty/panel/internal/runner"
	"github.com/stretchr/testify/assert"
)

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"success", "✓"},
		{"timeout", "⏱"},
		{"failed", "✗"},
		{"cancelled", "✗"},
		{"unknown", "✗"},
	}
	for _, tt := range tests {
		got := statusIcon(tt.status)
		assert.Equal(t, tt.want, got, "statusIcon(%q)", tt.status)
	}
}

func TestExtractHeadings(t *testing.T) {
	t.Run("basic headings", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "output.md")
		content := `# Introduction
Some text here.

## Architecture Overview
More details.

### Implementation Notes
Even more details.

Regular line without heading.

## Testing Strategy
`
		err := os.WriteFile(path, []byte(content), 0o600)
		assert.NoError(t, err)

		headings := extractHeadings(path)
		assert.Equal(t, []string{
			"Introduction",
			"Architecture Overview",
			"Implementation Notes",
			"Testing Strategy",
		}, headings)
	})

	t.Run("max 10 headings", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "many.md")
		var b strings.Builder
		for i := 0; i < 15; i++ {
			b.WriteString("## Heading ")
			b.WriteString(strings.Repeat("X", i+1))
			b.WriteString("\ntext\n")
		}
		err := os.WriteFile(path, []byte(b.String()), 0o600)
		assert.NoError(t, err)

		headings := extractHeadings(path)
		assert.Len(t, headings, 10)
	})

	t.Run("no headings", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "plain.txt")
		err := os.WriteFile(path, []byte("just plain text\nno headings here\n"), 0o600)
		assert.NoError(t, err)

		headings := extractHeadings(path)
		assert.Empty(t, headings)
	})

	t.Run("missing file", func(t *testing.T) {
		headings := extractHeadings("/nonexistent/path/file.md")
		assert.Nil(t, headings)
	})
}

func TestBuildSummary(t *testing.T) {
	t.Run("successful run with cost", func(t *testing.T) {
		dir := t.TempDir()

		// Write a mock output file
		outputContent := `# Introduction
This is the introduction.

## Architecture Overview
Details about architecture.

Some body text with enough words to count.
`
		err := os.WriteFile(filepath.Join(dir, "claude-opus.md"), []byte(outputContent), 0o600)
		assert.NoError(t, err)

		manifest := &Manifest{
			Version:     1,
			Prompt:      "Review this authentication flow",
			StartedAt:   time.Now().Add(-45 * time.Second),
			CompletedAt: time.Now(),
			Duration:    "45s",
			Platform:    "darwin/arm64",
			Config: ManifestConfig{
				ReadOnly:    "bestEffort",
				Timeout:     300,
				MaxParallel: 3,
			},
			Results: []ManifestResult{
				{
					ToolID:     "claude-opus",
					Status:     "success",
					Duration:   "42.5s",
					ExitCode:   0,
					OutputFile: "claude-opus.md",
					StderrFile: "claude-opus.stderr",
					Cost: &runner.Cost{
						InputTokens:  1234,
						OutputTokens: 567,
						TotalUSD:     0.05,
					},
				},
			},
		}

		summary := BuildSummary(manifest, dir)

		assert.Contains(t, summary, "# Run Summary")
		assert.Contains(t, summary, "**Prompt:** Review this authentication flow")
		assert.Contains(t, summary, "**Tools:** claude-opus")
		assert.Contains(t, summary, "**Policy:** read-only=bestEffort")
		assert.Contains(t, summary, "## Results")
		assert.Contains(t, summary, "### ✓ claude-opus")
		assert.Contains(t, summary, "- Status: success")
		assert.Contains(t, summary, "- Duration: 42.5s")
		assert.Contains(t, summary, "- Word count:")
		assert.Contains(t, summary, "- Key sections:")
		assert.Contains(t, summary, "  - Introduction")
		assert.Contains(t, summary, "  - Architecture Overview")
		assert.Contains(t, summary, "## Cost Summary")
		assert.Contains(t, summary, "| claude-opus | 1234 | 567 | $0.05 |")
	})

	t.Run("failed result", func(t *testing.T) {
		dir := t.TempDir()

		manifest := &Manifest{
			Version:     1,
			Prompt:      "Test prompt",
			StartedAt:   time.Now(),
			CompletedAt: time.Now(),
			Duration:    "15s",
			Config: ManifestConfig{
				ReadOnly: "strict",
			},
			Results: []ManifestResult{
				{
					ToolID:     "gemini-pro",
					Status:     "failed",
					Duration:   "15.2s",
					ExitCode:   1,
					OutputFile: "gemini-pro.md",
					StderrFile: "gemini-pro.stderr",
				},
			},
		}

		summary := BuildSummary(manifest, dir)

		assert.Contains(t, summary, "### ✗ gemini-pro")
		assert.Contains(t, summary, "- Status: failed")
		assert.Contains(t, summary, "- Exit code: 1")
		assert.Contains(t, summary, "- Error: exit code 1")
		assert.NotContains(t, summary, "## Cost Summary")
	})

	t.Run("prompt truncation", func(t *testing.T) {
		dir := t.TempDir()
		longPrompt := strings.Repeat("a", 150)

		manifest := &Manifest{
			Prompt:  longPrompt,
			Config:  ManifestConfig{ReadOnly: "off"},
			Results: []ManifestResult{},
		}

		summary := BuildSummary(manifest, dir)

		assert.Contains(t, summary, "**Prompt:** "+strings.Repeat("a", 100)+"...")
		assert.NotContains(t, summary, strings.Repeat("a", 101))
	})

	t.Run("multiple tools", func(t *testing.T) {
		dir := t.TempDir()

		err := os.WriteFile(filepath.Join(dir, "tool-a.md"), []byte("some output text"), 0o600)
		assert.NoError(t, err)
		err = os.WriteFile(filepath.Join(dir, "tool-b.md"), []byte("other output"), 0o600)
		assert.NoError(t, err)

		manifest := &Manifest{
			Prompt: "Multi-tool test",
			Config: ManifestConfig{ReadOnly: "bestEffort"},
			Results: []ManifestResult{
				{
					ToolID:     "tool-a",
					Status:     "success",
					Duration:   "10s",
					OutputFile: "tool-a.md",
				},
				{
					ToolID:     "tool-b",
					Status:     "success",
					Duration:   "12s",
					OutputFile: "tool-b.md",
				},
			},
		}

		summary := BuildSummary(manifest, dir)

		assert.Contains(t, summary, "**Tools:** tool-a, tool-b")
		assert.Contains(t, summary, "### ✓ tool-a")
		assert.Contains(t, summary, "### ✓ tool-b")
	})
}

func TestWriteSummary(t *testing.T) {
	dir := t.TempDir()
	content := "# Run Summary\nTest content\n"

	err := WriteSummary(dir, content)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(dir, "summary.md"))
	assert.NoError(t, err)
	assert.Equal(t, content, string(data))

	// Verify file permissions
	info, err := os.Stat(filepath.Join(dir, "summary.md"))
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
