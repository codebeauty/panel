package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSkillCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skill",
		Short: "Print slash-command template for AI agent integration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(skillTemplate)
		},
	}
}

const skillTemplate = `---
name: horde
description: Send prompts to multiple AI agents in parallel for independent second opinions. Use when the user wants code reviews, architecture feedback, or a sanity check from other AI models.
---

# Horde — Multi-Agent Review Skill

> **Note:** This is a reference skill template. Your agent system may use a different skill/command format. Adapt the structure and frontmatter below to match your system's conventions — the workflow and phases are what matter.

Send a prompt to multiple AI coding agents in parallel and synthesize their responses.

Arguments: $ARGUMENTS

**If no arguments provided**, ask the user what they want reviewed.

---

## Phase 1: Context Gathering

Parse ` + "`$ARGUMENTS`" + ` to understand what the user wants reviewed. Then identify relevant context:

1. **Files mentioned in the prompt**: Use Glob/Grep to find files referenced by name, class, function, or keyword
2. **Recent changes**: Run ` + "`git diff HEAD`" + ` and ` + "`git diff --staged`" + ` to identify what changed
3. **Related code**: Search for key terms from the prompt to identify the most relevant files (up to 5 files)

**Important**: You do NOT need to read and inline every file. The AI agents that horde invokes have access to the filesystem and git — they can read files and run git commands themselves. Your job is to *identify* the relevant files and reference them, not to copy their contents into the prompt. See Phase 3 for how to use ` + "`@file`" + ` references.

---

## Phase 2: Agent Selection

1. **Discover available agents, agent groups, role presets, and role preset groups** by running via Bash:
   ` + "```bash" + `
   horde agents list
   horde loadouts list
   horde raiders list
   horde squads list
   ` + "```" + `
   - ` + "`agents list`" + `: configured AI CLI agents with IDs and binary paths
   - ` + "`loadouts list`" + `: named groups of agents (predefined sets of agent IDs)
   - ` + "`raiders list`" + `: available role presets (e.g. security reviewer, performance engineer, architect)
   - ` + "`squads list`" + `: named groups of role presets for cross-product runs

2. **MANDATORY: Print the full output of all four commands, then ask the user which to use.**

   **Always print the full output** of all commands as inline text (not inside AskUserQuestion). Just show the raw output so the user sees every option. Do NOT reformat or abbreviate it.

   Then ask the user to pick **agents** (or a loadout):

   **If 4 or fewer agents**: Use AskUserQuestion with ` + "`multiSelect: true`" + `, one option per agent.

   **If more than 4 agents**: AskUserQuestion only supports 4 options. Use these fixed options:
   - Option 1: "All [N] agents" — sends to every configured agent
   - Option 2-4: The first 3 individual agents by ID
   - The user can always select "Other" to type a comma-separated list of agent IDs from the printed list above

   If loadouts exist, you MAY offer loadout options (e.g. "Loadout: smart"), but you MUST expand them to the underlying agent IDs and confirm that expanded list with the user before running. This avoids silently omitting or adding agents.
   If the user says something like "use the smart loadout", you MUST look up that loadout in the configured loadouts list (` + "`horde loadouts list`" + `). If it exists, use it (via ` + "`--loadout smart`" + ` or by expanding to agent IDs) and confirm the expanded agent list before running. If it does not exist, tell the user and ask them to choose again — do not guess.

3. **Ask about role presets.** After agent selection, ask the user if they want to apply a role preset or a group of role presets:

   - **Single role preset** (` + "`--raider / -R`" + `): One persona applied to all agents. Example: ` + "`-R security`" + ` makes every agent respond as a security reviewer.
   - **Role preset group** (` + "`--squad / -S`" + `): Cross-product — every agent runs once per role preset in the group. Example: 3 agents × 2 role presets = 6 parallel runs. Output uses composite IDs like ` + "`claude@security`" + `.
   - **None**: No role preset, agents respond without a persona.

   ` + "`--raider`" + ` and ` + "`--squad`" + ` are mutually exclusive.

   If the user mentions a role preset or group, confirm it exists in the listed output. If a group would produce a large cross-product (>8 runs), warn the user before proceeding.

4. Wait for the user's selection before proceeding.

5. **MANDATORY: Confirm the selection before continuing.** After the user picks agents and optionally a role preset/group, echo back the exact plan:

   > Sending to: **claude-opus**, **codex-5.3-high**, **gemini-pro**
   > Role preset: **security**

   Or for role preset groups:
   > Sending to: **claude-opus@security**, **claude-opus@performance**, **codex-5.3-high@security**, **codex-5.3-high@performance** (2 agents × 2 role presets)

   Then ask the user to confirm (e.g. "Look good?") before proceeding to Phase 3. This prevents silent agent omissions. If the user corrects the list, update your selection accordingly.

---

## Phase 3: Prompt Assembly

1. **Generate a slug** from the topic (lowercase, hyphens, max 40 chars)
   - "review the auth flow" → ` + "`auth-flow-review`" + `
   - "is this migration safe" → ` + "`migration-safety-review`" + `

2. **Create the output directory** via Bash inside your project's output directory (default: ` + "`agents/horde/`" + `) in your current working directory. The directory name MUST always be prefixed with a UNIX timestamp (seconds) so runs are lexically sortable and never collide:
   ` + "```" + `
   <cwd>/<outputDir>/TIMESTAMP-[slug]
   ` + "```" + `
   By default, ` + "`<outputDir>`" + ` is ` + "`agents/horde`" + `, but users can customize it via config (` + "`defaults.outputDir`" + `) or the ` + "`horde raid -o <dir>`" + ` flag.
   For example, if your cwd is ` + "`/Users/me/project`" + `: ` + "`/Users/me/project/agents/horde/1770676882-auth-flow-review`" + `

3. **Write the prompt file** using the Write tool to the directory you just created — ` + "`<cwd>/<outputDir>/TIMESTAMP-[slug]/prompt.md`" + `. Use an absolute path based on your current working directory, NOT a relative path.

   **IMPORTANT:** Do NOT write the prompt file to ` + "`/tmp`" + `, ` + "`~/tmp`" + `, or any temporary directory outside the project. The AI agents are sandboxed to the project directory and will not have access to files outside it. The file MUST be inside the ` + "`<outputDir>`" + ` directory you just created.

   **The invoked agents can read files and use git.** You do NOT need to inline file contents or diff output into the prompt. Instead, use ` + "`@path/to/file`" + ` references to point agents at the relevant files. They will read the files themselves. This keeps the prompt concise and avoids bloating it with copied code.

   Only inline small, critical snippets if they're essential for framing the question (e.g. a specific function signature or error message). For everything else, use ` + "`@file`" + ` references.

` + "```markdown" + `
# Review Request

## Question
[User's original prompt/question from $ARGUMENTS]

## Context

### Files to Review
[List @path/to/file references for each relevant file found in Phase 1]
[e.g. @src/core/executor.ts, @src/adapters/claude.ts]

### Recent Changes
[Brief description of what changed. If a diff is relevant, tell the agent to run ` + "`git diff HEAD`" + ` themselves, or inline only a small critical snippet]

### Related Code
[@path/to/file references for related files discovered via search]

## Instructions
You are providing an independent review. Be critical and thorough.
- Read the referenced files to understand the full context
- Analyze the question in the context provided
- Identify risks, tradeoffs, and blind spots
- Suggest alternatives if you see better approaches
- Be direct and opinionated — don't hedge
- Structure your response with clear headings
` + "```" + `

---

## Phase 4: Run

Run horde via Bash with the prompt file (using the absolute path from Phase 3), passing the user's selected agents and optional role preset:

` + "```bash" + `
horde raid -f <cwd>/<outputDir>/TIMESTAMP-[slug]/prompt.md --agents [comma-separated-agent-ids] --json
` + "```" + `

Examples:
- ` + "`--agents claude-opus,codex-5.3-high,gemini-3-pro`" + `
- ` + "`--loadout smart`" + ` (uses a configured agent group)
- ` + "`--agents claude-opus,codex-5.3-high -R security`" + ` (apply role preset to all agents)
- ` + "`--agents claude-opus,codex-5.3-high -S code-quality`" + ` (cross-product with role preset group)
- ` + "`--agents claude-opus,codex-5.3-high -S code-quality --yes`" + ` (skip confirmation for large cross-products)

Use ` + "`timeout: 600000`" + ` (10 minutes). Horde runs all selected agents in parallel and writes output to the directory shown in the JSON manifest.

**Important**: Use ` + "`-f`" + ` (file mode) so the prompt is sent as-is without wrapping. Use ` + "`--json`" + ` to get structured output for parsing. Add ` + "`--yes`" + ` when using ` + "`--squad`" + ` to skip the interactive confirmation prompt (required for non-TTY agent context).

**Timing**: Sessions commonly take more than 10 minutes. Horde prints each agent's progress status. If a run seems stuck, you can check agent processes in the output.

---

## Phase 5: Read Results

1. **Parse the JSON output** from stdout — it contains the run manifest with status, duration, word count, and output file paths for each agent. Each result entry includes a ` + "`raider`" + ` field when a role preset was used.
2. **Read each agent's response** from the ` + "`.md`" + ` output file in the run directory. When role presets are used, output files use composite IDs (e.g. ` + "`claude@security.md`" + `).
3. **Check ` + "`.stderr`" + ` files** for any agent that failed or returned empty output
4. **Skip empty or error-only responses** — note which agents failed

---

## Phase 6: Synthesize and Present

Combine all agent responses into a synthesis:

` + "```markdown" + `
## Horde Results

**Agents consulted:** [list of agents that responded, noting role presets if used]

**Consensus:** [What most agents agree on — key takeaways]

**Disagreements:** [Where they differ, and reasoning behind each position]

**Key Risks:** [Risks or concerns flagged by any agent]

**Blind Spots:** [Things none of the agents addressed that seem important]

**Recommendation:** [Your synthesized recommendation based on all inputs]

---
Results saved to: [output directory from manifest]
` + "```" + `

When role presets were used, group insights by perspective (e.g. "Security findings" vs "Performance findings") to help the user see how each perspective contributed.

Present this synthesis to the user. Be concise — the individual reports are saved for deep reading.

---

## Phase 7: Action (Optional)

After presenting the synthesis, ask the user what they'd like to address. Offer the top 2-3 actionable items from the synthesis as options. If the user wants to act on findings, plan the implementation before making changes.

---

## Error Handling

- **horde not installed**: Tell the user to install it (` + "`go install github.com/codebeauty/horde/cmd/horde@latest`" + `)
- **No agents configured**: Tell the user to run ` + "`horde wake`" + `
- **Agent fails**: Note it in the synthesis and continue with other agents' results
- **All agents fail**: Report errors from stderr files and suggest checking agent configurations
`
