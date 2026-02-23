# Development

## Requirements

- Go 1.26+
- macOS (darwin/arm64)

## Alternative Install Methods

### Go Install

Requires a [released version](https://github.com/codebeauty/panel/releases).

```bash
go install github.com/codebeauty/panel/cmd/panel@latest
```

### Build from Source

```bash
git clone https://github.com/codebeauty/panel.git
cd panel
make build      # -> dist/panel
make install    # -> /usr/local/bin/panel
```

Build with a version tag:

```bash
make build VERSION=1.0.0
```

## Testing

```bash
make test                     # Run tests with race detector
go test ./... -v -race        # Tests directly
go vet ./...                  # Static analysis
make clean                    # Remove binary
```

## Project Structure

```
cmd/panel/main.go             # Entry point
internal/
├── adapter/                  # Adapter interface + Claude/Codex/Gemini/Amp/Cursor/Custom
├── cli/                      # Cobra commands (run, init, tools, groups, summary, cleanup, skill)
├── config/                   # Config types, loading, saving, validation
├── gather/                   # Context gathering (files + git diff)
├── output/                   # Atomic writes, manifest, summary, cleanup scanning
├── runner/                   # Parallel execution, process management
└── ui/                       # Progress display with animated spinner
```

## Supported Adapters

### Claude

```
claude -p --output-format text [--model opus] [read-only flags] <prompt-file>
```

Read-only restricts to: `Read`, `Glob`, `Grep`, `WebFetch`, `WebSearch`.

**Models:**

| ID | Compound ID | Description |
|----|-------------|-------------|
| opus | claude-opus | Opus 4.6 — most capable (recommended) |
| sonnet | claude-sonnet | Sonnet 4.5 — fast and capable |
| haiku | claude-haiku | Haiku 4.5 — fastest, most affordable |

### Codex

```
codex exec [--sandbox read-only] -c web_search=live --skip-git-repo-check <prompt-file>
```

**Models:**

| Compound ID | Description |
|-------------|-------------|
| codex-5.3-high | GPT-5.3 Codex — high reasoning (recommended) |
| codex-5.3-xhigh | GPT-5.3 Codex — xhigh reasoning |
| codex-5.3-medium | GPT-5.3 Codex — medium reasoning |

### Gemini

```
gemini -p "" [--extensions ""] [--allowed-tools ...] --output-format text
```

Prompt delivered via stdin. Read-only restricts to: `read_file`, `list_directory`, `search_file_content`, `glob`, `google_web_search`, `codebase_investigator`.

**Models:**

| Compound ID | Description |
|-------------|-------------|
| gemini-3.1-pro | Gemini 3.1 Pro — latest (recommended) |
| gemini-2.5-pro | Gemini 2.5 Pro — stable GA |
| gemini-3-flash | Gemini 3 Flash — fast |
| gemini-2.5-flash | Gemini 2.5 Flash — fast GA |

### Amp

```
amp -x
```

Prompt delivered via stdin.

**Models:**

| Compound ID | Description |
|-------------|-------------|
| amp-smart | Smart — Opus 4.6, most capable (recommended) |
| amp-deep | Deep — GPT-5.2 Codex, extended thinking |

### Cursor

```
cursor-agent -p --output-format text --trust [--mode ask] <prompt-file>
```

File-based prompt delivery (like Claude). Uses `--mode ask` for read-only enforcement. See [cursor.com/cli](https://cursor.com/cli).

**Models:**

| Compound ID | Description |
|-------------|-------------|
| cursor-opus-4.6-thinking | Claude 4.6 Opus (Thinking) — default (recommended) |
| cursor-composer-1.5 | Composer 1.5 |
| cursor-opus-4.6 | Claude 4.6 Opus |
| cursor-sonnet-4.6-thinking | Claude 4.6 Sonnet (Thinking) |
| cursor-sonnet-4.6 | Claude 4.6 Sonnet |
| cursor-gpt-5.3-codex-xhigh-fast | GPT-5.3 Codex Extra High Fast |
| cursor-gpt-5.3-codex-high | GPT-5.3 Codex High |
| cursor-gemini-3-pro | Gemini 3 Pro |
| cursor-gemini-3-flash | Gemini 3 Flash |
| cursor-grok | Grok |

### Custom

For any CLI tool. Uses `{prompt}` placeholder substitution in extra flags, or `stdin: true` for stdin delivery.

```json
{
  "my-tool": {
    "binary": "/usr/local/bin/my-tool",
    "adapter": "my-tool",
    "extraFlags": ["--query", "{prompt}"],
    "enabled": true,
    "stdin": false
  }
}
```

## Environment Variables

Panel filters environment variables passed to child processes. Only these are forwarded:

- `PATH`, `HOME`, `USER`, `SHELL`, `TERM`
- `ANTHROPIC_API_KEY`, `OPENAI_API_KEY`
- `GEMINI_API_KEY`, `GOOGLE_API_KEY`
- `AMP_API_KEY`
- `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`
- `TMPDIR`

Additionally, `CI=true` and `NO_COLOR=1` are injected into all child processes.

## Security

- Config files must be owner-readable only (`0o600`); group/other-writable configs are rejected
- Output files written with `0o600`, directories with `0o700`
- Config saves use atomic write (temp file + rename)
- Tool names validated against `[a-zA-Z0-9._-]+`
- ANSI escape sequences stripped from captured output
- 10MB per-stream output limit with truncation
- Process groups isolated via `Setpgid` and killed on timeout (SIGTERM, then SIGKILL after 5s)
- Graceful shutdown on Ctrl+C via `signal.NotifyContext`
