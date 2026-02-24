<p align="center">
  <img src="assets/panel.svg" alt="panel" width="400">
</p>

# Panel

A second opinion is good. A panel of experts is better. A panel of expert AIs. One prompt in, independent opinions out.

Inspired by [counselors](https://github.com/aarondfrancis/counselors) by [Aaron Francis](https://github.com/aarondfrancis), but may go different paths. We'll see :) So, Panel is provided by [Mathias Bachmann](https://bsky.app/profile/designerdrug.net).

## Install

### Homebrew (recommended)

```bash
brew install codebeauty/tap/panel
```

Or as a cask:

```bash
brew install --cask codebeauty/tap/panel
```

> **Note:** for now panel is macOS only (Apple Silicon and Intel).

## Quick Start

```bash
panel init

panel run "review this authentication flow for security issues"
```

## Commands

| Command | Description |
|---------|-------------|
| `panel run [prompt]` | Dispatch a prompt to AI tools in parallel |
| `panel summary latest` | Print the most recent run summary |
| `panel summary list` | List recent runs as detailed cards |
| `panel cleanup` | Remove old output directories |
| `panel init` | Auto-discover installed AI CLIs and write config |
| `panel tools` | Manage configured tools (list, remove, test, discover, rename, add) |
| `panel groups` | Manage named groups of tools |
| `panel teams` | Manage expert teams (list, create, delete) |
| `panel config` | Show resolved configuration |
| `panel doctor` | Check configuration and tool availability |
| `panel experts` | Manage experts (list, show, create, edit, reset) |
| `panel agent` | Print setup instructions for AI agent integration |
| `panel skill` | Print slash-command template for AI agent integration |
| `panel ls` | Alias for `panel tools list` |

### `panel run [prompt]`

Dispatch a prompt to AI tools in parallel.

```bash
# Send to all enabled tools
panel run "review this authentication flow for security issues"

# Send to specific tools
panel run -t claude-opus,gemini-3-pro "is this migration safe?"

# Use a named group
panel run -f prompt.md --group smart

# Assign an expert to all tools
panel run -E security "review this authentication flow"

# Pipe from stdin
echo "explain this error" | panel run

# Gather context alongside the prompt
panel run -c . "what does this diff do?"
panel run -c src/core,src/adapters "review these modules"
```

```
Flags:
  -t, --tools <ids>        Comma-separated tool IDs
  -g, --group <name>       Named group from config
  -r, --read-only <mode>   enforced, bestEffort, or none
      --timeout <seconds>  Per-tool timeout (default: 540)
  -o, --output <dir>       Output directory override
      --json               Output manifest as JSON
  -f, --file <path>        Read prompt from file
      --dry-run            Show invocations without executing
  -c, --context <paths>    Gather context (comma-separated paths, or "." for git diff)
  -E, --expert <id>        Expert to apply to all tools (overrides per-tool config)
  -T, --team <name>        Named team of experts (cross-product dispatch)
      --yes                Skip confirmation prompts
```

`--team` and `--expert` are mutually exclusive.

When running interactively with multiple tools and no `--tools`/`--group` flag, panel shows a numbered list for selection.

### `panel summary`

View run summaries and browse run history.

```bash
panel summary latest              # Print the most recent summary
panel summary latest --path       # Print the run directory path (for scripting)
panel summary latest --json       # Print the run manifest (run.json) instead
panel summary list                # List recent runs as detailed cards
panel summary list --limit 5      # Show only the last 5 runs
panel summary list --json         # Output as JSON array of manifests
```

Both subcommands accept `-o, --output-dir` to override the output directory. Without it, the configured output directory is used (respecting project-level `.panel.json` overrides).

Example output:

```
─── 2026-02-23 00:36 ───
Prompt: Give me a compact idea how to build a SwiftUI app which use ...
Tools:  claude (✓ 22.171s), gemini (✓ 38.357s)
Path:   agents/panel/give-me-a-compact-idea-...-1771803377

─── 2026-02-23 00:07 ───
Prompt: Audit my config for any potential issues
Tools:  claude (✓ 15.2s), codex (✗ 28.1s), gemini (✓ 22.4s)
Path:   agents/panel/audit-my-config-...-1771801594
```

### `panel cleanup`

Remove old output directories.

```bash
panel cleanup                        # Remove runs older than 1 day (interactive)
panel cleanup --older-than 2w        # Remove runs older than 2 weeks
panel cleanup --dry-run              # Show what would be removed
panel cleanup --older-than 30m -y    # Remove runs older than 30 minutes, skip confirmation
panel cleanup --json                 # Output results as JSON
```

Supports duration suffixes: `ms`, `s`, `m`, `h`, `d`, `w`. A bare number is interpreted as days.

### `panel init`

Auto-discover installed AI CLIs and write config.

```bash
panel init
```

Scans for `claude`, `codex`, `gemini`, and `amp` binaries in PATH, `/opt/homebrew/bin`, `/usr/local/bin`, and `~/.local/bin`. Registers all model variants per adapter with recommended models enabled by default. Also syncs built-in expert presets to `~/.config/panel/experts/` (existing customizations are preserved).

### `panel tools`

Manage configured tools.

```bash
panel tools list              # Show all tools (alias: panel ls)
panel tools remove <id>       # Remove a tool
panel tools test [id...]      # Test tools with "Reply OK" prompt
panel tools discover          # Scan for new tools not yet configured
```

### `panel groups`

Manage named groups of tools.

```bash
panel groups create smart --tools claude-opus,gemini-3-pro
panel groups list
panel groups delete smart
```

### `panel experts`

Manage experts — expert roles that shape how AI tools respond to the same prompt.

```bash
panel experts list              # List all experts (built-in + custom)
panel experts show security     # Print expert contents
panel experts create my-expert  # Create custom expert (opens $EDITOR)
panel experts edit security     # Edit existing expert (opens $EDITOR)
panel experts reset             # Re-sync built-in presets
panel experts delete <id>    # Delete an expert (--force to ignore team refs)
```

Panel ships 6 built-in experts:

| ID | Role |
|----|------|
| `security` | Security engineer — vulnerabilities, OWASP Top 10, attack surfaces |
| `performance` | Performance engineer — latency, memory, algorithmic complexity |
| `architect` | Software architect — SOLID, coupling, API design, extensibility |
| `reviewer` | Code reviewer — bugs, edge cases, readability, test gaps |
| `devil` | Devil's advocate — challenge assumptions, find flaws, argue the opposite |
| `product` | Product lead — user impact, acceptance criteria, prioritization |

Experts are markdown files stored in `~/.config/panel/experts/`. Create custom experts by adding `.md` files to that directory or using `panel experts create`.

#### Creating a custom expert

```bash
# Option 1: Use the create command (opens $EDITOR)
panel experts create golang-expert

# Option 2: Write the file directly
cat > ~/.config/panel/experts/golang-expert.md << 'EOF'
You are a senior Go developer reviewing for idiomatic patterns.

Focus on:
- Error handling (wrapping, sentinel errors, checking)
- Concurrency safety (goroutine leaks, race conditions, mutex usage)
- Interface design (small, consumer-defined)
- Naming conventions and package structure
- Performance (allocations, buffer reuse, unnecessary copies)

Be specific: point to the exact line, explain why it's wrong, and provide a corrected snippet.
EOF
```

The expert ID is the filename without `.md` — use letters, numbers, hyphens, underscores, and dots.

#### Using experts

```bash
# Apply an expert to all tools for one run
panel run -E security "review this authentication flow"

# Without -E, no expert is used — the prompt is sent as-is
panel run "review this code"
```

#### Per-tool experts in config

Assign default experts to specific tools in `config.json` so each tool always adopts a different expert role:

```json
{
  "tools": {
    "claude-opus": {
      "binary": "/usr/local/bin/claude",
      "adapter": "claude",
      "extraFlags": ["--model", "opus"],
      "enabled": true,
      "expert": "security"
    },
    "gemini-3-pro": {
      "binary": "/usr/local/bin/gemini",
      "adapter": "gemini",
      "extraFlags": ["--model", "gemini-2.5-pro"],
      "enabled": true,
      "expert": "architect"
    },
    "codex": {
      "binary": "/usr/local/bin/codex",
      "adapter": "codex",
      "enabled": true
    }
  }
}
```

In this setup, `panel run "review this code"` sends the prompt to all three tools — Claude answers as a security engineer, Gemini as an architect, and Codex answers without an expert. The `-E` flag overrides all per-tool experts for a single run.

When an expert is active, the prompt is prepended with the expert's role instructions. Each tool gets its own prompt file (`<tool-id>.prompt.md`) in the output directory. The original `prompt.md` is always preserved without expert modifications.

### `panel teams`

Manage named teams of experts. A team is a reusable list of expert IDs. When used with `--team`, panel creates a cross-product: each tool runs once per expert in the team.

```bash
panel teams list                                     # List all teams
panel teams create code-review --experts security,architect,reviewer
panel teams delete code-review
```

#### Using teams

```bash
# Run with a team — creates tool × expert cross-product
panel run -T code-review "review this module"

# Combine with tool selection (2 tools × 3 experts = 6 runs)
panel run -t claude-opus,gemini-3-pro -T code-review "review this code"
```

Each tool runs independently with each expert. The composite IDs use `@` as separator (e.g., `claude-opus@security`, `gemini-3-pro@architect`). When the cross-product exceeds 8 runs, panel prompts for confirmation (skip with `--yes`).

### `panel skill`

Print a 7-phase slash-command template for AI agent integration.

## Configuration

Global config: `~/Library/Application Support/panel/config.json`

Project overrides: `.panel.json` in the project root (merges with global; read-only mode can only be tightened, not loosened).

```json
{
  "version": 1,
  "defaults": {
    "timeout": 540,
    "outputDir": "./agents/panel",
    "readOnly": "bestEffort",
    "maxParallel": 4
  },
  "tools": {
    "claude-opus": {
      "binary": "/usr/local/bin/claude",
      "adapter": "claude",
      "extraFlags": ["--model", "opus"],
      "enabled": true
    }
  },
  "groups": {
    "smart": ["claude-opus", "gemini-3-pro"]
  },
  "teams": {
    "code-review": ["security", "architect", "reviewer"]
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `timeout` | 540 | Per-tool timeout in seconds |
| `outputDir` | `./agents/panel` | Base directory for run output |
| `readOnly` | `bestEffort` | `enforced`, `bestEffort`, or `none` |
| `maxParallel` | 4 | Max tools running concurrently |

Per-tool fields:

| Field | Description |
|-------|-------------|
| `expert` | Default expert ID for this tool (overridden by `-E` flag) |

## Output Structure

Each run creates a timestamped directory:

```
agents/panel/
└── review-auth-flow-1770676882/
    ├── prompt.md              # Original prompt (without expert)
    ├── run.json               # Manifest with metadata
    ├── summary.md             # Heuristic summary (no LLM)
    ├── claude-opus.md         # Claude's response
    ├── claude-opus.stderr     # Claude's stderr
    ├── claude-opus.prompt.md  # Per-tool prompt with expert (if expert used)
    ├── gemini-3-pro.md        # Gemini's response
    ├── gemini-3-pro.stderr    # Gemini's stderr
    ├── claude-opus@security.md         # Team run: Claude as security expert
    ├── claude-opus@security.prompt.md  # Per-tool prompt with expert
    ├── gemini-3-pro@architect.md       # Team run: Gemini as architect
    └── gemini-3-pro@architect.stderr
```

When using `--team`, output files use composite IDs (`tool@expert`).

Use `panel summary latest` to quickly view the most recent result, or `--json` on `panel run` to get the manifest on stdout for programmatic consumption.

## Duplicate Tool Runs

Run the same tool multiple times by repeating its ID:

```bash
panel run -t claude-opus,claude-opus,claude-opus "review this code"
```

Creates three independent runs as `claude-opus`, `claude-opus__2`, `claude-opus__3`.

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for build instructions, adapter details, project structure, and security documentation.

## Why panel?

Same prompt fan-out pattern as [counselors](https://github.com/aarondfrancis/counselors), rewritten in Go.

|                  | panel | counselors |
|------------------|-------|------------|
| Runtime deps     | None  | Node.js 20+|
| Install size     | ~3 MB | ~102 MB    |
| Startup          | ~4 ms | ~120 ms    |
| Peak memory      | ~5 MB | ~65 MB     |

Both dispatch to the same AI CLIs — response times depend on the AI provider, not the tool.
