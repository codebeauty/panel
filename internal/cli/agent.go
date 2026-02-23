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

   # Apply an expert persona to all tools
   panel run "Review this code" -E security --json

   # Use a named team (cross-product: every tool × every expert)
   panel run "Review this code" -T security-focused --json

## Experts

Experts are role presets that shape how each agent responds. When an expert
is applied, its role definition is prepended to the prompt so the agent
adopts that persona.

   # List available experts
   panel experts list

   # Show an expert's role definition
   panel experts show security

   # Apply one expert to all tools (--expert / -E)
   panel run "Review auth flow" -E security --json

Built-in experts: security, performance, architect, reviewer, devil, product.
Create custom experts with: panel experts create <id>

## Teams

Teams are named groups of experts. When you use --team / -T, panel creates
a cross-product: every selected tool runs once per expert in the team.
For example, 3 tools × 2 experts = 6 parallel runs.

   # List configured teams
   panel teams list

   # Create a team
   panel teams create code-quality --experts reviewer,security,performance

   # Use a team (cross-product dispatch)
   panel run "Review this PR" -T code-quality --json

   # Combine with tool/group selection
   panel run "Review this PR" --tools claude,codex -T code-quality --json

Note: --expert and --team are mutually exclusive.

The manifest (run.json) includes the expert ID for each result entry,
and composite tool IDs use the format {tool}@{expert} (e.g. claude@security).

## Output Structure

Panel writes results to the output directory (default: ./agents/panel/):
  - prompt.md            — the original prompt
  - {tool}.md            — each tool's response
  - {tool}.stderr        — each tool's stderr
  - {tool}.prompt.md     — per-tool prompt (when experts are used)
  - run.json             — manifest with metadata, costs, and expert IDs
  - summary.md           — human-readable summary

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
