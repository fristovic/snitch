# Snitch Architecture

Snitch is a **lie detector** for Cursor on macOS. It extracts claims from assistant **prose**, checks them against **evidence** (tool calls + captured output + filesystem + git + consistency), and stores results locally.

**Menu-bar-first:** Snitch Bar (`cmd/snitchbar`) is the primary app. It owns `snitchd` lifecycle (start/stop), shows status in the menu bar, fires alerts on new lies, and exposes **Copy Last Lie** / **Browse lies…**. The CLI (`snitch`) is for history, debugging, and power users.

## Data flow

```
Cursor transcript JSONL (+ terminal files fallback)
        │
        ▼
 transcript.Watcher  ──► TurnCompleted (prose, tool_use, tool_result, start HEAD)
        │
        ▼
 capture.Engine      ──► RunCaptured
        │
        ▼
 verify.Engine       ──► prose + consistency + tool claims → verifiers → SQLite
        │                    │
        │                    ▼
        │              internal/notify (macOS alerts on fail)
        ▼
 snitchd IPC ◄──── Snitch Bar (subscribe, status, loadLatestLie)
        │
        ▼
 snitch lies / log / dashboard (CLI)
```

## Components

| Package / binary | Role |
|------------------|------|
| `internal/transcript` | fsnotify watcher + JSONL parser (`tool_use`, `tool_result`) |
| `internal/capture` | Turn → run payload, scrub secrets, dedupe by output hash |
| `internal/verify` | Prose extractor + consistency + contradiction + tool verifiers |
| `internal/record` | SQLite runs + claims |
| `internal/ipc` | Unix socket RPC for CLI and Snitch Bar |
| `internal/notify` | macOS Notification Center alerts from `snitchd` |
| `cmd/snitchd` | Daemon: watcher, capture, verify, IPC, notifications |
| `cmd/snitchbar` | Menu bar app: daemon lifecycle, tray UI, lie alerts |
| `cmd/snitch` | CLI + TUI (`doctor`, `uninstall`) |

## Menu bar flow

1. LaunchAgent (`com.snitch.menubar`) opens **Snitch Bar.app** at login.
2. Snitch Bar starts bundled `snitchd` (or finds it on PATH) and connects via IPC.
3. Menu shows **Snitching...**, **Start Snitching**, or **Stop Snitching** depending on state.
4. On `subscribe` events with failed runs, icon enters alert state; optional Notification Center alert from `snitchd`.
5. **Copy Last Lie** loads the latest lie via IPC (`loadLatestLie`); lie preview was removed from the dropdown.

## Prose lie detection

`verify.ExtractProseClaims` uses deterministic regex patterns (high precision, low recall).

`verifiers.ContradictionVerifier` checks prose claims against:

- **Tool calls** in the same turn (`Shell`, `Write`, `StrReplace`, `Delete`)
- **Captured shell output** — `tool_result` on the call, or Cursor `terminals/*.txt` matched by command + time window
- **Filesystem** for file claims and stub/placeholder bodies
- **Git** `startHEAD..HEAD` for commit claims (baseline captured when the turn starts)

`verifiers.ConsistencyVerifier` checks same-turn internal contradictions (`self_contradiction`, `count_mismatch`, `negation_violation`) with no external oracle.

Tool-call verifiers (`file`, `shell`, `subagent`) provide secondary evidence for actions Cursor actually executed.

## Schema

**runs** — one row per agent turn: verdict, severity, session, project, command.

**claims** — one row per verified claim: `claim_type`, `source` (`prose`|`tool`|`consistency`), claimed text, actual evidence, severity.

## IPC methods

- `status` — daemon health + lie stats
- `get_runs` — filtered run list
- `get_run` — run + claims
- `get_claims` — lie query with filters
- `lie_stats` — aggregate counts by claim type
- `subscribe` — live `run.completed` events

## Distribution

- **Homebrew tap** `fristovic/snitch` — auto-published by goreleaser on release tags; **Snitch Bar.app** in cellar (`opt/snitch`)
- **curl installer** — `scripts/install.sh` installs CLI + **Snitch Bar.app** to `~/.local/share/snitch/` and registers menubar LaunchAgent
- Legacy `com.snitch.daemon` LaunchAgent is removed on install/uninstall (daemon lives inside the app bundle)

## Threat model

- Runs as the user, not root
- Data stays local; analytics is opt-in
- Secrets scrubbed before persistence
- No network interception; reads only local Cursor transcript files

## Platform

**macOS only** in this release. Cursor transcript layout and paths are assumed.
