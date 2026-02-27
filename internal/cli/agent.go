package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const agentInstructions = `# Horde — AI Agent Integration

Horde sends the same prompt to multiple AI coding CLIs in parallel and
collects their independent responses. Use it from inside Claude Code,
Codex CLI, Gemini CLI, or any agent that can run shell commands.

## Quick Start

1. Ensure horde is installed and configured:

   horde wake --auto
   horde doctor

2. Add horde as a tool in your AI agent's configuration.

### Claude Code

Add to your CLAUDE.md or project instructions:

   When you need a second opinion, use horde:

   horde raid "your question here" --json

   Parse the JSON output to read each agent's response.

### Codex CLI

Add a custom command or use horde directly in prompts:

   codex exec "Run: horde raid 'your question' --json and summarize the responses"

## CLI Reference

   # Send a prompt to all configured agents, get JSON manifest on stdout
   horde raid "Is this the right architecture?" --json

   # Send to a specific named group of agents
   horde raid "Review this approach" -l fast --json

   # Enforce read-only mode (agents cannot modify files)
   horde raid "Analyze this code" -r enforced --json

   # Include file and git diff context
   horde raid "Review these changes" -c . --json

   # Apply a role preset to all agents (e.g. security reviewer persona)
   horde raid "Review this code" -R security --json

   # Apply a group of role presets (cross-product: each agent × each role)
   horde raid "Review this code" -S security-focused --json

## Role Presets (--raider / -R)

Role presets define a persona that shapes how each agent responds. When
applied, the role definition is prepended to the prompt so the agent
adopts that perspective (e.g. security reviewer, performance engineer).

   # List available role presets
   horde raiders list

   # Show a role preset's definition
   horde raiders show security

   # Apply one role preset to all agents
   horde raid "Review auth flow" -R security --json

Built-in presets: security, performance, architect, reviewer, devil, product.
Create custom presets: horde raiders create <id>

## Role Preset Groups (--squad / -S)

A squad is a named group of role presets. With --squad, horde runs a
cross-product: every selected agent runs once per role preset in the group.
For example, 3 agents × 2 role presets = 6 parallel runs.

   # List configured groups
   horde squads list

   # Create a group of role presets
   horde squads create code-quality --raiders reviewer,security,performance

   # Use a group (cross-product)
   horde raid "Review this PR" -S code-quality --json

   # Combine with agent selection
   horde raid "Review this PR" --agents claude,codex -S code-quality --json

Note: --raider and --squad are mutually exclusive.

The manifest (run.json) includes the role preset ID for each result entry.
Composite IDs use the format {agent}@{role} (e.g. claude@security).

## Output Structure

Horde writes results to the output directory (default: ./agents/horde/):
  - prompt.md            — the original prompt
  - {agent}.md           — each agent's response
  - {agent}.stderr       — each agent's stderr
  - {agent}.prompt.md    — per-agent prompt (when a role preset is applied)
  - run.json             — manifest with metadata, costs, and role IDs
  - summary.md           — human-readable summary

The --json flag outputs the manifest to stdout for programmatic parsing.
`

func newAgentCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "intel",
		Aliases: []string{"agent"},
		Short:   "Show agent integration instructions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(agentInstructions)
		},
	}
}
