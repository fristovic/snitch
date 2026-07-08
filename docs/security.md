# Security

## Scope

Snitch on macOS reads local AI agent transcript artifacts (Cursor, Claude Code, Codex, and Pi JSONL files; OpenCode's SQLite database), verifies prose claims locally, and stores results in `~/.snitch/`.

## Data handling

- **Local only** by default — SQLite at `~/.snitch/snitch.db`
- **Scrubbing** — API keys and common secret patterns removed before any command, output, or turn snapshot is stored
- **Opt-in telemetry** — labeled-verdict metadata only (claim type, harness, model, verdict, label, SHA-256 hash of claim text); no code, paths, or claim text
- **Opt-in analytics** — aggregated stats only; no raw prompts or assistant text

## Permissions

Snitch needs read access to:

- The transcript locations of enabled harnesses (e.g. `~/.cursor/projects`, `~/.claude/projects`, `~/.codex/sessions`, `~/.pi/agent/sessions`, `~/.local/share/opencode/opencode.db`)
- Project directories referenced in transcripts (for file/git checks)

It does not require admin privileges or network interception.

## IPC

The CLI talks to `snitchd` over a Unix domain socket (`~/.snitch/snitch.sock` by default). Only local processes running as your user can connect.

## What Snitch does not do

- No proxy or TLS interception
- No cloud LLM calls for verification
- No transmission of transcript content, ever — the opt-in telemetry and analytics channels carry metadata only

See [SECURITY.md](../SECURITY.md) for vulnerability reporting.
