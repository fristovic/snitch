# Snitch Architecture

Snitch is a **lie detector** for AI coding agents. It extracts claims from assistant **prose**, checks them against **evidence** (tool calls + captured output + filesystem + git + consistency), and stores results locally.

**Multi-harness:** Snitch ingests transcripts from five agents — Cursor, Claude Code, Codex, Pi (all JSONL via fsnotify) and OpenCode (SQLite via polling). Each harness provides a parser/reader, a path resolver, a shell-output resolver, and a tool-name normalization map, bundled in a `harness.Descriptor` registry. The verification pipeline is harness-agnostic: every harness normalizes its raw tool names to a canonical internal vocabulary at parse time.

**Menu-bar-first:** Snitch Bar (`cmd/snitchbar`) is the primary app. It owns `snitchd` lifecycle (start/stop), shows status in the menu bar, fires Notification Center alerts on new lies (app-bundle icon), and exposes **View Details…** / **History ▸ Open Dashboard…**. The CLI (`snitch`) is for history, debugging, and power users.

## Data flow

```
Agent transcripts (Cursor/Claude/Codex/Pi JSONL via fsnotify,
                   OpenCode SQLite via polling)
        │
        ▼
 transcript.Watcher  ──► TurnCompleted (prose, tool_use, tool_result, start/end HEAD, file manifest)
        │
        ▼
 capture.Engine      ──► RunCaptured
        │
        ▼
 verify.Engine       ──► enrich context (subagent merge, 3-turn lookback, prose segments)
        │                    │
        │                    ▼
        │              verifiers → SQLite (+ turn snapshots)
        │                    │
        │                    ▼
        │              run.completed (+ top-lie fields)
        ▼
 snitchd IPC ◄──── Snitch Bar (subscribe, status, get_claims, notify.Deliver)
        │
        ▼
 snitch log / dashboard (CLI)
```

## Components

| Package / binary | Role |
|------------------|------|
| `internal/transcript` | Per-harness parsers (Cursor/Claude/Codex/Pi JSONL) + OpenCode SQLite reader, fsnotify watchers, tool-result correlation |
| `internal/harness` | Harness registry: bundles per-platform ingestion (watcher/reader), path resolver, and shell-output resolver |
| `internal/capture` | Turn → run payload, scrub secrets |
| `internal/verify` | Prose extractor + consistency + contradiction + tool verifiers (harness-agnostic; shell output via per-harness resolver) |
| `internal/record` | SQLite runs + claims + user feedback labels |
| `internal/ipc` | Unix socket RPC for CLI and Snitch Bar (incl. `set_label` feedback) |
| `internal/notify` | macOS Notification Center alerts delivered by Snitch Bar (CGO) |
| `cmd/snitchd` | Daemon: watcher, capture, verify, IPC |
| `cmd/snitchbar` | Menu bar app: daemon lifecycle, tray UI, notifications, lie alerts |
| `cmd/snitch` | CLI + TUI (`doctor`, `uninstall`) |

## Menu bar flow

1. LaunchAgent (`com.snitch.menubar`) opens **Snitch Bar.app** at login.
2. Snitch Bar starts bundled `snitchd` (or finds it on PATH) and connects via IPC.
3. Menu shows **Snitching...**, **Start Snitching**, or **Stop Snitching** depending on state.
4. On `subscribe` `run.completed` events, Snitch Bar may call `notify.Deliver` (Notification Center, Snitch app icon) and, for fails, enter alert state.
5. **View Details…** loads the latest lie via IPC (`get_claims`) and opens `snitch log --run <id>` in Terminal.

## Evidence enrichment pipeline

After capture, `verify.BuildVerifyContext` assembles evidence before verifiers run:

1. **Prose segmentation** — split execution vs recap (`### Summary`, `## Summary`, `---`)
2. **Subagent merge** — `transcript.LoadSubagentToolCalls` merges `subagents/*.jsonl` tool calls whose turns overlap the parent window (Cursor; other harnesses return no subagents)
3. **Session lookback** — load up to 3 prior turn payloads from SQLite (`runs.payload_json`)
4. **Harness resolver** — `VerifyContext.ShellOutputResolver` is selected from the registry by the run's harness, so shell-output resolution stays per-platform (Cursor terminal files vs inline tool results)
5. **Severity calibration** — recap claims and low-confidence prose adjust severity after verification

Turn snapshots persist `payload_json`, `start_head`, `end_head`, and `file_manifest_json` for the next turn's baseline.

## Data flywheel (coming soon)

Opt-in community labeling (correct / incorrect feedback, missed-lie reports) is planned. Labels store on the `runs` table in `~/.snitch/snitch.db`. When sharing is enabled (`telemetry.enabled` + share flag), sync may send the claim sentence, capped surrounding context, claimed→actual, and metadata (claim type, harness, model, verdict, label, sentence hash) — never user prompts, code, paths, or full transcripts. The Snitch Bar labeling UI stays behind `flywheelUIEnabled` until the training API ships.

## Prose lie detection

`verify.ExtractProseClaims` uses deterministic regex patterns (high precision, low recall).

`verifiers.ContradictionVerifier` checks prose claims against:

- **Tool calls** in the same turn plus subagent merges and (for eligible claim types) up to 3 prior turns
- **Captured shell output** — inline `tool_result` on the call (Claude Code, Codex, Pi, OpenCode), or Cursor `terminals/*.txt` files matched by command + time window. Resolution is per-harness via the `ShellOutputResolver` interface.
- **Filesystem** for file claims and stub/placeholder bodies
- **Git** `startHEAD..HEAD` for commit claims (baseline captured when the turn starts)

`verifiers.ConsistencyVerifier` checks same-turn internal contradictions (`self_contradiction`, `count_mismatch`, `negation_violation`) with no external oracle.

Tool-call verifiers (`file`, `shell`, `subagent`) provide secondary evidence for actions the agent actually executed. Each harness normalizes its raw tool names to canonical names (e.g. Claude `Bash`→`Shell`, Codex `apply_patch`→`StrReplace`) at parse time, so verifiers only ever see the canonical vocabulary.

## Schema

**runs** — one row per agent turn: verdict, severity, session, project, command.

**claims** — one row per verified claim: `claim_type`, `source` (`prose`|`tool`|`consistency`), claimed text, actual evidence, severity.

## IPC methods

- `status` — daemon health + lie stats
- `get_runs` — filtered run list
- `get_run` — run + claims
- `get_claims` — lie query with filters
- `get_config` / `set_config` — read and update daemon config
- `set_label` — record a user's correct/incorrect verdict on a run
- `add_missed_claim` — record a user-reported false negative
- `subscribe` — live `run.completed` events

## Distribution

- **Homebrew tap** `fristovic/snitch` — auto-published by goreleaser on release tags; **Snitch Bar.app** in cellar (`opt/snitch`)
- **curl installer** — `scripts/install.sh` installs CLI + **Snitch Bar.app** to `~/.local/share/snitch/` and registers menubar LaunchAgent
- Legacy `com.snitch.daemon` LaunchAgent is removed on install/uninstall (daemon lives inside the app bundle)

## Threat model

- Runs as the user, not root
- Data stays local; analytics is opt-in
- Secrets scrubbed before persistence
- No network interception; reads only local agent transcript files and the local OpenCode SQLite DB

## Platform

**macOS only** in this release. Outbound network traffic is opt-in only: telemetry sync (labeled-verdict metadata) and analytics reporting (aggregate metadata). Both are off by default and carry no code, file paths, or claim text. Run dedup by output hash happens in the verify engine before insert.
