# Snitch User Guide

## Requirements

- macOS
- [Cursor](https://cursor.com) with agent transcripts under `~/.cursor/projects/`

## Installation

### Homebrew (recommended)

```bash
brew tap fristovic/snitch
brew install snitch
snitch start
```

The curl installer also registers Snitch Bar to open at login.

### curl installer

```bash
curl -fsSL https://raw.githubusercontent.com/fristovic/snitch/main/scripts/install.sh | bash
snitch start
```

Verify:

```bash
snitch status
```

Look for the Snitch icon in the menu bar. Click it for status, **Show Last Lie**, or **Open Dashboard…**.

## Menu bar (Snitch Bar)

Open Snitch Bar from Terminal with `snitch start`, or from the menu bar after install. Snitch Bar is the main app (no Dock icon). It **starts lie detection automatically** when you open it.

- **Snitching...** status when the lie detector is active
- **Start Snitching** / **Stop Snitching** — pause / resume lie detection
- **Show Last Lie** — open full verification log for the latest lie (`snitch log --run <id>`)
- **Open Dashboard…** — open interactive TUI in Terminal (`snitch dashboard`)
- **Quit Snitch Bar** stops the daemon

If Snitch is paused or offline, choose **Start Snitching** to resume.

Disable login auto-start: `SNITCH_MENUBAR=0 ./scripts/install.sh`

## Notifications

When a lie is caught, `snitchd` can send a macOS notification (title: claim type, body: claimed → actual). Enabled by default; configure under `notifications` in `~/.snitch/config.yaml`. macOS will prompt for notification permission on the first alert.

## How it works

When a Cursor agent turn ends, Snitch:

1. Reads the transcript JSONL for that turn (including `tool_result` blocks when present)
2. Extracts claims from assistant **prose** (not just tool calls)
3. Cross-checks against tool calls, captured shell output (transcript results or Cursor terminal files), files on disk, git, and same-turn consistency
4. Records lies in `~/.snitch/snitch.db`

A **snitch** is a high-confidence prose claim that evidence contradicts.

## Commands

### `snitch log`

Show full verification detail for a single run:

```bash
snitch log --run <id>
snitch log --run <id> --trace
snitch log --run <id> --json
```

Use `snitch dashboard` to browse history and find run IDs.

### `snitch dashboard`

Interactive TUI with live refresh:

- `tab` — switch runs / lies view
- `v` — cycle verdict filter
- `t` — cycle claim type filter
- `/` — search
- `q` — quit

### `snitch status`

Shows daemon health. Use `--detailed` for lie statistics and recent failures.

When no runs exist yet, status prints a hint to trigger a Cursor turn.

### `snitch uninstall`

```bash
snitch uninstall          # remove daemon + binaries
snitch uninstall --purge  # also remove ~/.snitch
```

Homebrew users:

```bash
brew uninstall snitch
```

If you upgraded from an older install that used `brew services start snitch`, stop the legacy service first:

```bash
brew services stop snitch
```

## Configuration

Config file: `~/.snitch/config.yaml`

```yaml
cursor:
  enabled: true
  transcript_watch_path: ~/.cursor/projects
retention:
  max_days: 30
  keep_failures: true
display:
  tui:
    max_runs_visible: 100
    refresh_ms: 500
```

## Limitations

- **High precision, low recall** — only confident contradictions are flagged
- **Same-turn evidence** — work in prior turns is not credited to this turn's prose
- **Shell output** — resolved from `tool_result` blocks when Cursor writes them, else from matching `~/.cursor/projects/*/terminals/*.txt` files
- **Push claims** cap at WARN when no `git push` shell call is visible
- Deterministic only — no semantic/LLM verification
