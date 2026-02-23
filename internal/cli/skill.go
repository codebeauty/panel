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
name: panel
description: Get parallel second opinions from multiple AI coding agents. Use when the user wants independent reviews, architecture feedback, or a sanity check from other AI models.
---

# Panel — Multi-Agent Review Skill

> **Note:** This is a reference skill template. Your agent system may use a different skill/command format. Adapt the structure and frontmatter below to match your system's conventions — the workflow and phases are what matter.

Fan out a prompt to multiple AI coding agents in parallel and synthesize their responses.

Arguments: $ARGUMENTS

**If no arguments provided**, ask the user what they want reviewed.

---

## Phase 1: Context Gathering

Parse ` + "`$ARGUMENTS`" + ` to understand what the user wants reviewed. Then identify relevant context:

1. **Files mentioned in the prompt**: Use Glob/Grep to find files referenced by name, class, function, or keyword
2. **Recent changes**: Run ` + "`git diff HEAD`" + ` and ` + "`git diff --staged`" + ` to identify what changed
3. **Related code**: Search for key terms from the prompt to identify the most relevant files (up to 5 files)

**Important**: You do NOT need to read and inline every file. Subagents have access to the filesystem and git — they can read files and run git commands themselves. Your job is to *identify* the relevant files and reference them, not to copy their contents into the prompt. See Phase 3 for how to use ` + "`@file`" + ` references.

---

## Phase 2: Agent Selection

1. **Discover available agents, groups, experts, and teams** by running via Bash:
   ` + "```bash" + `
   panel tools list
   panel groups list
   panel experts list
   panel teams list
   ` + "```" + `
   - ` + "`tools list`" + `: configured agents with IDs and binaries
   - ` + "`groups list`" + `: predefined sets of tool IDs
   - ` + "`experts list`" + `: available expert personas (e.g. security, performance, architect)
   - ` + "`teams list`" + `: named groups of experts for cross-product dispatch

2. **MANDATORY: Print the full output of all four commands, then ask the user which to use.**

   **Always print the full output** of all commands as inline text (not inside AskUserQuestion). Just show the raw output so the user sees every tool/group/expert/team. Do NOT reformat or abbreviate it.

   Then ask the user to pick **tools** (or a group):

   **If 4 or fewer agents**: Use AskUserQuestion with ` + "`multiSelect: true`" + `, one option per agent.

   **If more than 4 agents**: AskUserQuestion only supports 4 options. Use these fixed options:
   - Option 1: "All [N] agents" — sends to every configured agent
   - Option 2-4: The first 3 individual agents by ID
   - The user can always select "Other" to type a comma-separated list of agent IDs from the printed list above

   If groups exist, you MAY offer group options (e.g. "Group: smart"), but you MUST expand them to the underlying tool IDs and confirm that expanded list with the user before dispatch. This avoids silently omitting or adding agents.
   If the user says something like "use the smart group", you MUST look up that group in the configured groups list (` + "`panel groups list`" + `). If it exists, use it (via ` + "`--group smart`" + ` or by expanding to tool IDs) and confirm the expanded tool list before dispatch. If it does not exist, tell the user and ask them to choose again — do not guess.

3. **Ask about experts/teams.** After tool selection, ask the user if they want to apply an expert persona or team:

   - **Single expert** (` + "`--expert / -E`" + `): One expert persona applied to all tools. Example: ` + "`-E security`" + ` makes every agent adopt a security reviewer role.
   - **Team** (` + "`--team / -T`" + `): Cross-product dispatch — every tool runs once per expert in the team. Example: 3 tools × 2 experts in team = 6 parallel runs. Composite IDs like ` + "`claude@security`" + `.
   - **None**: No expert, standard dispatch.

   ` + "`--expert`" + ` and ` + "`--team`" + ` are mutually exclusive.

   If the user mentions an expert or team, confirm it exists in the listed output. If teams produce a large cross-product (>8 runs), warn the user before proceeding.

4. Wait for the user's selection before proceeding.

5. **MANDATORY: Confirm the selection before continuing.** After the user picks agents and optionally an expert/team, echo back the exact dispatch plan:

   > Dispatching to: **claude-opus**, **codex-5.3-high**, **gemini-pro**
   > Expert: **security**

   Or for teams:
   > Dispatching to: **claude-opus@security**, **claude-opus@performance**, **codex-5.3-high@security**, **codex-5.3-high@performance** (2 tools × 2 experts)

   Then ask the user to confirm (e.g. "Look good?") before proceeding to Phase 3. This prevents silent tool omissions. If the user corrects the list, update your selection accordingly.

---

## Phase 3: Prompt Assembly

1. **Generate a slug** from the topic (lowercase, hyphens, max 40 chars)
   - "review the auth flow" → ` + "`auth-flow-review`" + `
   - "is this migration safe" → ` + "`migration-safety-review`" + `

2. **Create the output directory** via Bash inside your project's panel output directory (default: ` + "`agents/panel/`" + `) in your current working directory. The directory name MUST always be prefixed with a UNIX timestamp (seconds) so runs are lexically sortable and never collide:
   ` + "```" + `
   <cwd>/<outputDir>/TIMESTAMP-[slug]
   ` + "```" + `
   By default, ` + "`<outputDir>`" + ` is ` + "`agents/panel`" + `, but users can customize it via config (` + "`defaults.outputDir`" + `) or the ` + "`panel run -o <dir>`" + ` flag.
   For example, if your cwd is ` + "`/Users/me/project`" + `: ` + "`/Users/me/project/agents/panel/1770676882-auth-flow-review`" + `

3. **Write the prompt file** using the Write tool to the directory you just created — ` + "`<cwd>/<outputDir>/TIMESTAMP-[slug]/prompt.md`" + `. Use an absolute path based on your current working directory, NOT a relative path.

   **IMPORTANT:** Do NOT write the prompt file to ` + "`/tmp`" + `, ` + "`~/tmp`" + `, or any temporary directory outside the project. Panel agents are sandboxed to the project directory and will not have access to files outside it. The file MUST be inside the ` + "`<outputDir>`" + ` directory you just created.

   **Subagents can read files and use git.** You do NOT need to inline file contents or diff output into the prompt. Instead, use ` + "`@path/to/file`" + ` references to point subagents at the relevant files. They will read the files themselves. This keeps the prompt concise and avoids bloating it with copied code.

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

## Phase 4: Dispatch

Run panel via Bash with the prompt file (using the absolute path from Phase 3), passing the user's selected agents and optional expert/team:

` + "```bash" + `
panel run -f <cwd>/<outputDir>/TIMESTAMP-[slug]/prompt.md --tools [comma-separated-tool-ids] --json
` + "```" + `

Examples:
- ` + "`--tools claude-opus,codex-5.3-high,gemini-3-pro`" + `
- ` + "`--group smart`" + ` (uses the configured group)
- ` + "`--tools claude-opus,codex-5.3-high -E security`" + ` (apply expert to all tools)
- ` + "`--tools claude-opus,codex-5.3-high -T code-quality`" + ` (cross-product with team)
- ` + "`--tools claude-opus,codex-5.3-high -T code-quality --yes`" + ` (skip confirmation for large cross-products)

Use ` + "`timeout: 600000`" + ` (10 minutes). Panel dispatches to the selected agents in parallel and writes results to the output directory shown in the JSON output.

**Important**: Use ` + "`-f`" + ` (file mode) so the prompt is sent as-is without wrapping. Use ` + "`--json`" + ` to get structured output for parsing. Add ` + "`--yes`" + ` when using teams to skip the interactive confirmation prompt (required for non-TTY agent context).

**Timing**: Sessions commonly take more than 10 minutes. Panel prints each tool's progress status. If a run seems stuck, you can check tool processes in the output.

---

## Phase 5: Read Results

1. **Parse the JSON output** from stdout — it contains the run manifest with status, duration, word count, and output file paths for each agent. Each result entry includes an ` + "`expert`" + ` field when an expert was used.
2. **Read each agent's response** from the ` + "`.md`" + ` output file in the run directory. When experts are used, tool IDs are composite (e.g. ` + "`claude@security.md`" + `).
3. **Check ` + "`.stderr`" + ` files** for any agent that failed or returned empty output
4. **Skip empty or error-only reports** — note which agents failed

---

## Phase 6: Synthesize and Present

Combine all agent responses into a synthesis:

` + "```markdown" + `
## Panel Review

**Agents consulted:** [list of agents that responded, noting expert personas if used]

**Consensus:** [What most agents agree on — key takeaways]

**Disagreements:** [Where they differ, and reasoning behind each position]

**Key Risks:** [Risks or concerns flagged by any agent]

**Blind Spots:** [Things none of the agents addressed that seem important]

**Recommendation:** [Your synthesized recommendation based on all inputs]

---
Reports saved to: [output directory from manifest]
` + "```" + `

When experts were used, group insights by expert perspective (e.g. "Security expert findings" vs "Performance expert findings") to help the user see how each perspective contributed.

Present this synthesis to the user. Be concise — the individual reports are saved for deep reading.

---

## Phase 7: Action (Optional)

After presenting the synthesis, ask the user what they'd like to address. Offer the top 2-3 actionable items from the synthesis as options. If the user wants to act on findings, plan the implementation before making changes.

---

## Error Handling

- **panel not installed**: Tell the user to install it (` + "`go install github.com/codebeauty/panel/cmd/panel@latest`" + `)
- **No tools configured**: Tell the user to run ` + "`panel init`" + `
- **Agent fails**: Note it in the synthesis and continue with other agents' results
- **All agents fail**: Report errors from stderr files and suggest checking tool configurations
`
