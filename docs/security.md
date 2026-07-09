# Security

## Scope

Snitch on macOS reads local AI agent transcript artifacts (Cursor, Claude Code, Codex, and Pi JSONL files; OpenCode's SQLite database), verifies prose claims locally, and stores results in `~/.snitch/`.

## Data handling

- **Local only** by default — SQLite at `~/.snitch/snitch.db`
- **Scrubbing** — API keys and common secret patterns removed before any command, output, or turn snapshot is stored
- **Opt-in labeling sync** (coming soon; off by default) — when both `telemetry.enabled` and a share flag are set, shared examples may include:
  - claim sentence (full assistant sentence containing the match)
  - capped surrounding context (±1–2 sentences)
  - Snitch’s claimed → actual pair
  - metadata: claim type, harness, model, verdict, user label, claim-sentence hash
  - **Never:** user prompts, full transcripts, source code, file paths, project paths, or shell dumps
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
- No transmission of user prompts, source code, or full transcripts — even when labeling sync is enabled

See [SECURITY.md](../SECURITY.md) for vulnerability reporting.
