package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const agentInstructions = `# Panel Agent Setup

Panel can be used as a "skill" inside AI coding agents like Claude Code,
Codex CLI, and others. This lets agents dispatch prompts to a panel of
other agents for second opinions.

## Quick Start

1. Ensure panel is installed and configured:

   panel init --auto
   panel doctor

2. Add panel as a skill in your AI agent's configuration.

### Claude Code

Add to your CLAUDE.md or project instructions:

   When you need a second opinion, use panel:

   panel run "your question here" --json

   Parse the JSON output to read each agent's response.

### Codex CLI

Add a custom command or use panel directly in prompts:

   codex exec "Run: panel run 'your question' --json and summarize the responses"

## Usage in Agent Context

   # Get second opinions with JSON output for parsing
   panel run "Is this the right architecture?" --json

   # Use a specific group of tools
   panel run "Review this approach" -g fast --json

   # Specify read-only mode for safety
   panel run "Analyze this code" -r enforced --json

   # Include file context
   panel run "Review these changes" -c . --json

## Output Structure

Panel writes results to the output directory (default: ./agents/panel/):
  - prompt.md    — the original prompt
  - {tool}.md    — each tool's response
  - {tool}.stderr — each tool's stderr
  - run.json     — manifest with metadata and costs
  - summary.md   — human-readable summary

The --json flag outputs the manifest to stdout for programmatic parsing.
`

func newAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent",
		Short: "Show agent integration instructions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(agentInstructions)
		},
	}
}
