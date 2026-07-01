# Snitch Architecture

Snitch is a **lie detector** for Cursor on macOS. It extracts claims from assistant **prose**, checks them against **evidence** (tool calls + filesystem + git), and stores results locally.

## Data flow

```
Cursor transcript JSONL
        │
        ▼
 transcript.Watcher  ──► TurnCompleted (user text, assistant text, tool calls, start HEAD)
        │
        ▼
 capture.Engine      ──► RunCaptured
        │
        ▼
 verify.Engine       ──► prose claims + tool claims → verifiers → SQLite
        │
        ▼
 snitch lies / log / dashboard (IPC)
```

## Components

| Package | Role |
|---------|------|
| `internal/transcript` | fsnotify watcher + JSONL parser for Cursor agent transcripts |
| `internal/capture` | Turn → run payload, scrub secrets, dedupe by output hash |
| `internal/verify` | Prose extractor + contradiction verifier + tool verifiers |
| `internal/record` | SQLite runs + claims |
| `internal/ipc` | Unix socket RPC for CLI |
| `cmd/snitchd` | Daemon: watcher, capture, verify, IPC |
| `cmd/snitch` | CLI + TUI |

## Prose lie detection

`verify.ExtractProseClaims` uses deterministic regex patterns (high precision, low recall).

`verifiers.ContradictionVerifier` checks prose claims against:

- **Tool calls** in the same turn (`Shell`, `Write`, `StrReplace`, `Delete`)
- **Filesystem** for file claims
- **Git** `startHEAD..HEAD` for commit claims (baseline captured when the turn starts)

Tool-call verifiers (`file`, `shell`, `subagent`) provide secondary evidence for actions Cursor actually executed.

## Schema

**runs** — one row per agent turn: verdict, severity, session, project, command.

**claims** — one row per verified claim: `claim_type`, `source` (`prose`|`tool`), claimed text, actual evidence, severity.

## IPC methods

- `status` — daemon health + lie stats
- `get_runs` — filtered run list
- `get_run` — run + claims
- `get_claims` — lie query with filters
- `lie_stats` — aggregate counts by claim type
- `subscribe` — live `run.completed` events

## Threat model

- Runs as the user, not root
- Data stays local; analytics is opt-in
- Secrets scrubbed before persistence
- No network interception; reads only local Cursor transcript files

## Platform

**macOS only** in this release. Cursor transcript layout and paths are assumed.
