# panel

A CLI that fans out the same prompt to multiple AI coding agents in parallel and collects their independent responses. Not task splitting — second opinions.

## Install

### Homebrew (recommended)

```bash
brew install codebeauty/homebrew-tap/panel
```

### Go Install

Requires Go 1.25+ and a [released version](https://github.com/codebeauty/panel/releases).

```bash
go install github.com/codebeauty/panel/cmd/panel@latest
```

### Build from Source

```bash
git clone https://github.com/codebeauty/panel.git
cd panel
make build
# Binary at dist/panel
```

> **Note:** panel is macOS only (Apple Silicon and Intel).

## Quick Start

```bash
make build && panel init

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
| `panel config` | Show resolved configuration |
| `panel doctor` | Check configuration and tool availability |
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
```

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

Scans for `claude`, `codex`, `gemini`, and `amp` binaries in PATH, `/opt/homebrew/bin`, `/usr/local/bin`, and `~/.local/bin`. Registers all model variants per adapter with recommended models enabled by default.

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
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `timeout` | 540 | Per-tool timeout in seconds |
| `outputDir` | `./agents/panel` | Base directory for run output |
| `readOnly` | `bestEffort` | `enforced`, `bestEffort`, or `none` |
| `maxParallel` | 4 | Max tools running concurrently |

## Output Structure

Each run creates a timestamped directory:

```
agents/panel/
└── review-auth-flow-1770676882/
    ├── prompt.md              # Original prompt
    ├── run.json               # Manifest with metadata
    ├── summary.md             # Heuristic summary (no LLM)
    ├── claude-opus.md         # Claude's response
    ├── claude-opus.stderr     # Claude's stderr
    ├── gemini-3-pro.md        # Gemini's response
    └── gemini-3-pro.stderr    # Gemini's stderr
```

Use `panel summary latest` to quickly view the most recent result, or `--json` on `panel run` to get the manifest on stdout for programmatic consumption.

## Duplicate Tool Runs

Run the same tool multiple times by repeating its ID:

```bash
panel run -t claude-opus,claude-opus,claude-opus "review this code"
```

Creates three independent runs as `claude-opus`, `claude-opus__2`, `claude-opus__3`.

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for build instructions, adapter details, project structure, and security documentation.
