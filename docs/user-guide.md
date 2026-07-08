# Snitch User Guide

## Requirements

- macOS
- At least one supported AI coding agent:
  - [Cursor](https://cursor.com) (`~/.cursor/projects/`) — enabled by default
  - [Claude Code](https://claude.com/claude-code) (`~/.claude/projects/`)
  - [Codex](https://github.com/openai/codex) (`~/.codex/sessions/`)
  - [Pi](https://pi.dev) (`~/.pi/agent/sessions/`)
  - [OpenCode](https://opencode.ai) (`~/.local/share/opencode/opencode.db`)

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
snitch doctor
```

Look for the Snitch icon in the menu bar. Click it for status, **Show Last Lie**, or **Open Dashboard…**.

## Enabling more agents

Cursor is watched by default. Enable the others and restart:

```bash
snitch config set platforms.claude.enabled true
snitch config set platforms.codex.enabled true
snitch config set platforms.pi.enabled true
snitch config set platforms.opencode.enabled true
snitch start
```

`snitch doctor` shows every harness's status and watch path.

## Menu bar (Snitch Bar)

Open Snitch Bar from Terminal with `snitch start`, or from the menu bar after install. Snitch Bar is the main app (no Dock icon). It **starts lie detection automatically** when you open it.

- **Snitching...** status when the lie detector is active
- **Start Snitching** / **Stop Snitching** — pause / resume lie detection
- **Show Last Lie** — open full verification log for the latest lie (`snitch log --run <id>`)
- **👍 / 👎** — mark the latest verdict correct or incorrect (trains Snitch)
- **Report Missed Lie…** — report a lie Snitch didn't catch
- **Share labels anonymously** — opt-in checkbox for metadata-only telemetry
- **Open Dashboard…** — open interactive TUI in Terminal (`snitch dashboard`)
- **Quit Snitch Bar** stops the daemon

If Snitch is paused or offline, choose **Start Snitching** to resume.

Disable login auto-start: `SNITCH_MENUBAR=0 ./scripts/install.sh`

## Notifications

When a lie is caught, `snitchd` can send a macOS notification (title: claim type, body: claimed → actual). Enabled by default; configure under `notifications` in `~/.snitch/config.yaml`. macOS will prompt for notification permission on the first alert.

## How it works

When an agent turn ends (any enabled harness), Snitch:

1. Reads the turn from the agent's local transcript artifacts (JSONL files, or SQLite for OpenCode)
2. Extracts claims from assistant **prose** (not just tool calls)
3. Cross-checks against tool calls, captured shell output (inline tool results, or Cursor terminal files), files on disk, git, session lookback (up to 3 prior turns), and same-turn consistency
4. Records lies in `~/.snitch/snitch.db`

A **snitch** is a high-confidence prose claim that evidence contradicts.

## Commands

### `snitch log`

Show full verification detail for a single run, or list recent runs per harness:

```bash
snitch log --run <id>
snitch log --run <id> --trace
snitch log --run <id> --json
snitch log --harness claude
```

Use `snitch dashboard` to browse history and find run IDs.

### `snitch dashboard`

Interactive TUI with live refresh (`--harness` to filter):

- `tab` — switch runs / lies view
- `v` — cycle verdict filter
- `t` — cycle claim type filter
- `/` — search
- `q` — quit

### `snitch status`

Shows daemon health and enabled harnesses. Use `--detailed` for lie statistics, per-harness run counts, and recent failures.

### `snitch label`

Mark a verdict correct or incorrect — every label trains Snitch:

```bash
snitch label <run-id> correct
snitch label <run-id> incorrect --share
snitch label missed --claimed "what the agent said" --actual "what happened"
```

### `snitch replay`

Run transcripts through the verification pipeline offline (throwaway database,
no daemon needed). Useful for measuring accuracy on your own sessions or
validating a new harness:

```bash
snitch replay ~/.cursor/projects --lies-only
snitch replay --harness claude ~/.claude/projects
```

### `snitch doctor`

Install checklist plus per-harness watch-path checks.

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
platforms:
  cursor:
    enabled: true
    transcript_watch_path: ~/.cursor/projects
  claude:
    enabled: false
    transcript_watch_path: ~/.claude/projects
  opencode:
    enabled: false
    transcript_watch_path: ~/.local/share/opencode/opencode.db
retention:
  max_days: 30
  keep_failures: true
telemetry:
  enabled: false            # opt-in metadata sync
  share_by_default: false   # labels marked shareable automatically
display:
  tui:
    max_runs_visible: 100
    refresh_ms: 500
```

## Limitations

- **High precision, low recall** — only confident contradictions are flagged
- **Session lookback is limited** — recap prose can credit evidence from up to 3 prior turns in the same session; older work is not credited
- **Shell output** — resolved from inline tool results (all harnesses), else from matching `~/.cursor/projects/*/terminals/*.txt` files (Cursor)
- **Push claims** cap at WARN when no `git push` shell call is visible
- Deterministic only — no semantic/LLM verification
