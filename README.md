<p align="center">
  <img src="assets/horde.svg" alt="horde" width="400">
</p>

# Horde

One prompt. A horde of raiders. Bring back the goop.

Deploy prompts to multiple AI agents in parallel and collect independent opinions.

Inspired by [counselors](https://github.com/aarondfrancis/counselors) by [Aaron Francis](https://github.com/aarondfrancis), but may go different paths. We'll see :) So, Horde is provided by [Mathias Bachmann](https://bsky.app/profile/designerdrug.net).

## Install

### Homebrew (recommended)

```bash
brew install codebeauty/tap/horde
```

Or as a cask:

```bash
brew install --cask codebeauty/tap/horde
```

> **Note:** for now horde is macOS only (Apple Silicon and Intel).

## Quick Start

```bash
horde wake

horde raid "review this authentication flow for security issues"
```

## Commands

| Command | Description |
|---------|-------------|
| `horde raid [prompt]` | Deploy a prompt to AI agents in parallel |
| `horde summary latest` | Print the most recent run summary |
| `horde summary list` | List recent runs as detailed cards |
| `horde cleanup` | Remove old output directories |
| `horde wake` | Auto-discover installed AI CLIs and write config |
| `horde agents` | Manage configured agents (list, remove, test, discover, rename, add) |
| `horde loadouts` | Manage named loadouts of agents |
| `horde squads` | Manage raider squads (list, create, delete) |
| `horde stash` | Show resolved configuration |
| `horde doctor` | Check configuration and agent availability |
| `horde raiders` | Manage raiders (list, show, create, edit, reset) |
| `horde intel` | Print setup instructions for AI agent integration |
| `horde skill` | Print slash-command template for AI agent integration |
| `horde ls` | Alias for `horde agents list` |

### `horde raid [prompt]`

Deploy a prompt to AI agents in parallel.

```bash
# Send to all enabled agents
horde raid "review this authentication flow for security issues"

# Send to specific agents
horde raid -a claude-opus,gemini-3-pro "is this migration safe?"

# Use a named loadout
horde raid -f prompt.md --loadout smart

# Assign a raider to all agents
horde raid -R security "review this authentication flow"

# Pipe from stdin
echo "explain this error" | horde raid

# Gather context alongside the prompt
horde raid -c . "what does this diff do?"
horde raid -c src/core,src/adapters "review these modules"
```

```
Flags:
  -a, --agents <ids>       Comma-separated agent IDs
  -l, --loadout <name>     Named loadout from config
  -r, --read-only <mode>   enforced, bestEffort, or none
      --timeout <seconds>  Per-agent timeout (default: 540)
  -o, --output <dir>       Output directory override
      --json               Output manifest as JSON
  -f, --file <path>        Read prompt from file
      --dry-run            Show invocations without executing
  -c, --context <paths>    Gather context (comma-separated paths, or "." for git diff)
  -R, --raider <id>        Raider to apply to all agents (overrides per-agent config)
  -S, --squad <name>       Named squad of raiders (cross-product deploy)
      --yes                Skip confirmation prompts
```

`--squad` and `--raider` are mutually exclusive.

When running interactively with multiple agents and no `--agents`/`--loadout` flag, horde shows a numbered list for selection.

### `horde summary`

View run summaries and browse run history.

```bash
horde summary latest              # Print the most recent summary
horde summary latest --path       # Print the run directory path (for scripting)
horde summary latest --json       # Print the run manifest (run.json) instead
horde summary list                # List recent runs as detailed cards
horde summary list --limit 5      # Show only the last 5 runs
horde summary list --json         # Output as JSON array of manifests
```

Both subcommands accept `-o, --output-dir` to override the output directory. Without it, the configured output directory is used (respecting project-level `.horde.json` overrides).

Example output:

```
--- 2026-02-23 00:36 ---
Prompt: Give me a compact idea how to build a SwiftUI app which use ...
Agents: claude (ok 22.171s), gemini (ok 38.357s)
Path:   agents/horde/give-me-a-compact-idea-...-1771803377

--- 2026-02-23 00:07 ---
Prompt: Audit my config for any potential issues
Agents: claude (ok 15.2s), codex (fail 28.1s), gemini (ok 22.4s)
Path:   agents/horde/audit-my-config-...-1771801594
```

### `horde cleanup`

Remove old output directories.

```bash
horde cleanup                        # Remove runs older than 1 day (interactive)
horde cleanup --older-than 2w        # Remove runs older than 2 weeks
horde cleanup --dry-run              # Show what would be removed
horde cleanup --older-than 30m -y    # Remove runs older than 30 minutes, skip confirmation
horde cleanup --json                 # Output results as JSON
```

Supports duration suffixes: `ms`, `s`, `m`, `h`, `d`, `w`. A bare number is interpreted as days.

### `horde wake`

Auto-discover installed AI CLIs and write config.

```bash
horde wake
```

Scans for `claude`, `codex`, `gemini`, and `amp` binaries in PATH, `/opt/homebrew/bin`, `/usr/local/bin`, and `~/.local/bin`. Registers all model variants per adapter with recommended models enabled by default. Also syncs built-in raider presets to the raiders directory (existing customizations are preserved).

### `horde agents`

Manage configured agents.

```bash
horde agents list              # Show all agents (alias: horde ls)
horde agents remove <id>       # Remove an agent
horde agents test [id...]      # Test agents with "Reply OK" prompt
horde agents discover          # Scan for new agents not yet configured
```

### `horde loadouts`

Manage named loadouts of agents.

```bash
horde loadouts create smart --agents claude-opus,gemini-3-pro
horde loadouts list
horde loadouts delete smart
```

### `horde raiders`

Manage raiders — role presets that shape how AI agents respond to the same prompt.

```bash
horde raiders list              # List all raiders (built-in + custom)
horde raiders show security     # Print raider contents
horde raiders create my-raider  # Create custom raider (opens $EDITOR)
horde raiders edit security     # Edit existing raider (opens $EDITOR)
horde raiders reset             # Re-sync built-in presets
horde raiders delete <id>       # Delete a raider (--force to ignore squad refs)
```

Horde ships 6 built-in raiders:

| ID | Role |
|----|------|
| `security` | Security engineer — vulnerabilities, OWASP Top 10, attack surfaces |
| `performance` | Performance engineer — latency, memory, algorithmic complexity |
| `architect` | Software architect — SOLID, coupling, API design, extensibility |
| `reviewer` | Code reviewer — bugs, edge cases, readability, test gaps |
| `devil` | Devil's advocate — challenge assumptions, find flaws, argue the opposite |
| `product` | Product lead — user impact, acceptance criteria, prioritization |

Raiders are markdown files stored in the raiders directory. Create custom raiders by adding `.md` files to that directory or using `horde raiders create`.

#### Creating a custom raider

```bash
# Option 1: Use the create command (opens $EDITOR)
horde raiders create golang-expert

# Option 2: Write the file directly
cat > ~/Library/Application\ Support/horde/raiders/golang-expert.md << 'EOF'
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

The raider ID is the filename without `.md` — use letters, numbers, hyphens, underscores, and dots.

#### Using raiders

```bash
# Apply a raider to all agents for one run
horde raid -R security "review this authentication flow"

# Without -R, no raider is used — the prompt is sent as-is
horde raid "review this code"
```

#### Per-agent raiders in config

Assign default raiders to specific agents in `config.json` so each agent always adopts a different raider role:

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

In this setup, `horde raid "review this code"` sends the prompt to all three agents — Claude answers as a security engineer, Gemini as an architect, and Codex answers without a raider. The `-R` flag overrides all per-agent raiders for a single run.

When a raider is active, the prompt is prepended with the raider's role instructions. Each agent gets its own prompt file (`<agent-id>.prompt.md`) in the output directory. The original `prompt.md` is always preserved without raider modifications.

### `horde squads`

Manage named squads of raiders. A squad is a reusable list of raider IDs. When used with `--squad`, horde creates a cross-product: each agent runs once per raider in the squad.

```bash
horde squads list                                     # List all squads
horde squads create code-review --raiders security,architect,reviewer
horde squads delete code-review
```

#### Using squads

```bash
# Run with a squad — creates agent x raider cross-product
horde raid -S code-review "review this module"

# Combine with agent selection (2 agents x 3 raiders = 6 runs)
horde raid -a claude-opus,gemini-3-pro -S code-review "review this code"
```

Each agent runs independently with each raider. The composite IDs use `@` as separator (e.g., `claude-opus@security`, `gemini-3-pro@architect`). When the cross-product exceeds 8 runs, horde prompts for confirmation (skip with `--yes`).

### `horde skill`

Print a 7-phase slash-command template for AI agent integration.

## Configuration

Global config: `~/Library/Application Support/horde/config.json`

Project overrides: `.horde.json` in the project root (merges with global; read-only mode can only be tightened, not loosened).

```json
{
  "version": 1,
  "defaults": {
    "timeout": 540,
    "outputDir": "./agents/horde",
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
| `timeout` | 540 | Per-agent timeout in seconds |
| `outputDir` | `./agents/horde` | Base directory for run output |
| `readOnly` | `bestEffort` | `enforced`, `bestEffort`, or `none` |
| `maxParallel` | 4 | Max agents running concurrently |

Per-agent fields:

| Field | Description |
|-------|-------------|
| `expert` | Default raider ID for this agent (overridden by `-R` flag) |

## Output Structure

Each run creates a timestamped directory:

```
agents/horde/
  review-auth-flow-1770676882/
    prompt.md              # Original prompt (without raider)
    run.json               # Manifest with metadata
    summary.md             # Heuristic summary (no LLM)
    claude-opus.md         # Claude's response
    claude-opus.stderr     # Claude's stderr
    claude-opus.prompt.md  # Per-agent prompt with raider (if raider used)
    gemini-3-pro.md        # Gemini's response
    gemini-3-pro.stderr    # Gemini's stderr
    claude-opus@security.md         # Squad run: Claude as security raider
    claude-opus@security.prompt.md  # Per-agent prompt with raider
    gemini-3-pro@architect.md       # Squad run: Gemini as architect
    gemini-3-pro@architect.stderr
```

When using `--squad`, output files use composite IDs (`agent@raider`).

Use `horde summary latest` to quickly view the most recent result, or `--json` on `horde raid` to get the manifest on stdout for programmatic consumption.

## Duplicate Agent Runs

Run the same agent multiple times by repeating its ID:

```bash
horde raid -a claude-opus,claude-opus,claude-opus "review this code"
```

Creates three independent runs as `claude-opus`, `claude-opus__2`, `claude-opus__3`.

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for build instructions, adapter details, project structure, and security documentation.

## Why horde?

Same prompt fan-out pattern as [counselors](https://github.com/aarondfrancis/counselors), rewritten in Go.

|                  | horde | counselors |
|------------------|-------|------------|
| Runtime deps     | None  | Node.js 20+|
| Install size     | ~3 MB | ~102 MB    |
| Startup          | ~4 ms | ~120 ms    |
| Peak memory      | ~5 MB | ~65 MB     |

Both deploy to the same AI CLIs — response times depend on the AI provider, not the tool.
