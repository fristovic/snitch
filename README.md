<p align="center">
  <img src="docs/snitch_logo.png" alt="Snitch logo" width="320">
</p>

<p align="center"><strong>Snitch watches your AI agent so you don't have to.</strong></p>

<p align="center"><a href="https://snitchworks.com">snitchworks.com</a> · <a href="#install">Install</a> · <a href="#help-train-snitch-coming-soon">Help train Snitch</a> · <a href="#roadmap">Roadmap</a></p>

<p align="center">
  <span style="display: inline-block; max-width: 720px; text-align: justify;">
    Snitch is a deterministic prose lie detector for AI coding agents. It watches transcripts from <a href="https://cursor.com">Cursor</a>, <a href="https://claude.com/claude-code">Claude Code</a>, <a href="https://github.com/openai/codex">Codex</a>, <a href="https://pi.dev">Pi</a>, and <a href="https://opencode.ai">OpenCode</a>, extracts high-confidence claims from assistant text ("all tests pass", "I committed this"), and flags claims contradicted by evidence: tool calls (including subagent merges), tool output, filesystem, git, session lookback (3 prior turns), and same-turn consistency.
  </span>
</p>

## Install



### macOS (Homebrew — recommended)

```bash
brew tap fristovic/snitch
brew install snitch
snitch start
```

Snitch Bar opens in the menu bar and starts the lie detector automatically.

### macOS (curl)

Latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/fristovic/snitch/main/scripts/install.sh | bash
```

From a cloned repo:

```bash
./scripts/install.sh
```

After install, open Snitch Bar:

```bash
snitch start
```



### What the installer does

1. Downloads or builds `snitch` CLI and **Snitch Bar.app** (includes `snitchd` inside the app)
2. Installs CLI to `~/.local/bin`
3. Installs **Snitch Bar.app** to `~/.local/share/snitch/`
4. Registers a LaunchAgent to open Snitch Bar at login



## Quick start

### Menu bar (everyday use)

Open Snitch Bar once — it lives in the menu bar with no Dock icon:

```bash
snitch start
```

From the Snitch menu:

| Action | What it does |
| ------ | ------------ |
| **Snitching…** / **Paused** / **Offline** | Current detection status |
| **Start Snitching** / **Stop Snitching** | Turn lie detection on or off |
| **Latest: …** | Preview of the most recent lie (claim type + short quote) |
| **View Details…** | Open Terminal with full details (`snitch log --run <id>`) |
| **History ▸ Open Dashboard…** | Open the interactive TUI (`snitch dashboard`) |
| **Preferences…** | Open `~/.snitch/config.yaml` |
| **Quit Snitch Bar** | Stop the daemon and exit |

When the model lies, the menu bar icon alerts and Snitch Bar may show a Notification Center alert (Snitch app icon). Click **View Details…** for the full verification breakdown, or **History ▸ Open Dashboard…** to browse history.

### Terminal (optional)

```bash
snitch status             # is detection running?
snitch dashboard          # browse runs and lies interactively
snitch log --run <id>     # full detail for one agent turn
snitch doctor             # install checklist
```



## Viewing history: `log` vs `dashboard`

Snitch stores every agent **turn** as a **run** (with a verdict and claims). A **lie** is a high-confidence prose claim inside a run that evidence contradicts.

| View | Best for | What you see |
| ---- | -------- | ------------ |
| **`snitch log --run <id>`** | One agent turn | Full breakdown — verdict, prompt, tool calls, every claim with evidence. |
| **`snitch dashboard`** | Browsing history | Interactive TUI — flip between runs and lies, filter, search, live refresh. |

**Menu bar shortcuts:** **View Details…** runs `snitch log --run <id>` for the latest lie. **History ▸ Open Dashboard…** runs `snitch dashboard`.

```bash
snitch log --run abc12345
snitch dashboard
```



## Commands

### Menu bar (Snitch Bar)

| Item | Description |
| ---- | ----------- |
| **Start / Stop Snitching** | Pause or resume lie detection |
| Alert icon | Flashes when a new lie is caught |
| **Latest: …** | Disabled preview of the most recent lie |
| **View Details…** | Open `snitch log --run <id>` for the latest lie |
| **History ▸ Open Dashboard…** | Open `snitch dashboard` in Terminal |
| **Preferences…** | Edit `~/.snitch/config.yaml` |
| **Quit Snitch Bar** | Stop `snitchd` and exit |

### Terminal (CLI)

| Command | Description |
| ------- | ----------- |
| `snitch start` | Open Snitch Bar |
| `snitch status` | Detection status (`--detailed` for per-harness stats) |
| `snitch log --run <id>` | Full verification detail for one run (`--trace`, `--json`) |
| `snitch log --harness <name>` | List recent runs for one agent platform |
| `snitch dashboard` | Interactive TUI for runs and lies (`--harness` filter) |
| `snitch replay <path>` | Run any transcript through the pipeline offline — measure accuracy on your own sessions |
| `snitch doctor` | Debug install checklist (per-harness) |
| `snitch uninstall` | Remove daemon and binaries (`--purge` for data) |
| `snitch config` | View/set configuration |


Snitch runs passively after install — it reads each enabled agent's local transcripts (see [Supported agents](#supported-agents)); Cursor's `~/.cursor/projects` is watched by default.

### Notifications

When Snitch Bar receives a failed (or optionally warned) run, it posts a macOS Notification Center alert attributed to **Snitch Bar.app** (Snitch head icon). Configure in `~/.snitch/config.yaml`:

```yaml
notifications:
  enabled: true
  on_warn: false
  rate_limit_s: 5
```

The first notification triggers the macOS permission prompt for Snitch Bar.

## Lie types


| Type                 | Example prose              | Contradiction                                      |
| -------------------- | -------------------------- | -------------------------------------------------- |
| `test_pass`          | "all tests pass"           | No test run, or test output shows failure          |
| `command_ran`        | "I ran the command"        | No shell tool call in the turn                     |
| `command_succeeded`  | "command ran successfully" | Shell exited with error                            |
| `committed`          | "I committed"              | No new commit since turn start                     |
| `pushed`             | "I pushed"                 | No `git push` shell call                           |
| `file_created`       | "created foo.go"           | No matching `Write` + file missing                 |
| `file_modified`      | "updated foo.go"           | No matching `Write`/`StrReplace` + file missing    |
| `file_deleted`       | "deleted foo.go"           | No matching `Delete`/`StrReplace` + file still present |
| `stub`               | "fully implemented"        | Written file is a placeholder (`panic("TODO")`, …) |
| `no_action`          | action claims              | Zero tool calls in the turn                        |
| `self_contradiction` | "won't modify X"           | Tool call edits X in the same turn                 |
| `count_mismatch`     | "updated all 5 files"      | File tool-call count ≠ 5                           |
| `negation_violation` | "did not touch tests"      | `*_test.*` file edited in the turn                 |



## Session lookback (3 turns)

Snitch persists each turn's full payload (tool calls, git HEAD, file manifest) in SQLite. When verifying recap/summary prose, it can credit evidence from up to **three prior turns** in the same session for:

- `committed` / `pushed` — git shell or HEAD delta in prior turns
- `test_pass` / `command_*` — shell evidence in prior turns
- `file_created` / `file_modified` / `file_deleted` — file tools + manifests in prior turns
- `stub` — placeholder bodies in files written this turn or prior turns

**Same-turn only:** `no_action`, `self_contradiction`, `count_mismatch`, and `negation_violation` never use cross-turn lookback.

Recap segments (`### Summary`, `## Summary`, horizontal rules) are tagged separately: inaccurate recap claims cap at WARN unless there is zero evidence across the current turn plus lookback.

## Supported agents

Snitch watches transcripts from five AI coding agents. Cursor is enabled by default; the others are opt-in.

| Agent | Format | Location | Enable |
| ----- | ------ | -------- | ------ |
| **Cursor** | JSONL | `~/.cursor/projects` | on by default |
| **Claude Code** | JSONL | `~/.claude/projects` | `snitch config set platforms.claude.enabled true` |
| **Codex** | JSONL | `~/.codex/sessions` | `snitch config set platforms.codex.enabled true` |
| **Pi** | JSONL | `~/.pi/agent/sessions` | `snitch config set platforms.pi.enabled true` |
| **OpenCode** | SQLite | `~/.local/share/opencode/opencode.db` | `snitch config set platforms.opencode.enabled true` |

After enabling a platform, restart Snitch (`snitch start`). Each platform's claims, tool calls, and shell output are normalized to a common internal vocabulary, so the verification pipeline works identically across all five.

The dashboard accepts a `--harness` filter to scope to one agent: `snitch dashboard --harness claude`.

## Help train Snitch (coming soon)

A community labeling flywheel — mark whether Snitch was right or wrong, report missed lies, and optionally share training examples — is **coming soon**. Labels stay local by default.

When sharing is enabled (dual opt-in: `telemetry.enabled` + share flag), a shared example may include:

- the **claim sentence** (full assistant sentence containing the match)
- a **short surrounding context** (capped ±1–2 sentences)
- Snitch’s **claimed → actual** pair
- metadata: claim type, harness, model, verdict, your label, and a hash for dedup

**Never shared:** user prompts, full transcripts, source code, file paths, project paths, or shell dumps.

## Roadmap

- **0.3.x (this release):** Multi-harness ingestion (Cursor + Claude Code + Codex + Pi + OpenCode), session lookback, Snitch Bar notifications with app icon.
- **Coming soon:** Community labeling and opt-in sync of claim sentences + short context to train a false-positive filter.
- **Later:** A locally-run false-positive classifier trained on community labels — reduces alert noise by filtering regex hits that aren't genuine claims.
- **Snitchworks:** A paid team layer — centralized dashboard, policy engine, premium semantic claim extraction.

## Limitations

- Deterministic regex extraction only (no LLM claim parsing) — semantic extraction is a later goal
- Lookback is limited to the current agent session (3 turns), not cross-session history
- Subagent tool calls are merged by **time window**, not `tool_use_id` mapping
- Consistency checks remain same-turn only
- File manifests hash paths touched by tool calls at turn end; out-of-band disk changes may be missed



## Documentation

- [User guide](docs/user-guide.md)
- [Architecture](ARCHITECTURE.md)
- [Contributing](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security](SECURITY.md)



## License

MIT

By participating, you agree to follow the [Code of Conduct](CODE_OF_CONDUCT.md).
