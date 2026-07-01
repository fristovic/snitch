# Security

## Scope

Snitch on macOS reads Cursor agent transcripts from `~/.cursor/projects/`, verifies prose claims locally, and stores results in `~/.snitch/`.

## Data handling

- **Local only** by default — SQLite at `~/.snitch/snitch.db`
- **Scrubbing** — API keys and common secret patterns removed before storage
- **Opt-in analytics** — aggregated stats only; no raw prompts or assistant text

## Permissions

Snitch needs read access to:

- `~/.cursor/projects/**/agent-transcripts/*.jsonl`
- Project directories referenced in transcripts (for file/git checks)

It does not require admin privileges or network interception.

## IPC

The CLI talks to `snitchd` over a Unix domain socket (`~/.snitch/snitch.sock` by default). Only local processes running as your user can connect.

## What Snitch does not do

- No proxy or TLS interception
- No cloud LLM calls for verification
- No transmission of transcript content unless analytics is explicitly enabled

See [SECURITY.md](../SECURITY.md) for vulnerability reporting.
