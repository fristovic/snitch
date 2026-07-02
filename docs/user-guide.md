# Snitch User Guide

## Requirements

- macOS
- [Cursor](https://cursor.com) with agent transcripts under `~/.cursor/projects/`

## Installation

### Homebrew (recommended)

```bash
brew tap fristovic/snitch
brew install snitch
brew services start snitch
snitch doctor
```

### curl installer

```bash
curl -fsSL https://raw.githubusercontent.com/fristovic/snitch/main/scripts/install.sh | bash
```

Verify:

```bash
snitch doctor
snitch status
```

## How it works

When a Cursor agent turn ends, Snitch:

1. Reads the transcript JSONL for that turn (including `tool_result` blocks when present)
2. Extracts claims from assistant **prose** (not just tool calls)
3. Cross-checks against tool calls, captured shell output (transcript results or Cursor terminal files), files on disk, git, and same-turn consistency
4. Records lies in `~/.snitch/snitch.db`

A **snitch** is a high-confidence prose claim that evidence contradicts.

## Commands

### `snitch doctor`

Check daemon, binaries, LaunchAgent, Cursor install, and transcript watch path.

### `snitch lies`

List caught lies:

```bash
snitch lies
snitch lies --type test_pass
snitch lies --project ~/code/myapp --since 24h
snitch lies --json
```

### `snitch log`

Show runs (failed by default):

```bash
snitch log
snitch log --all
snitch log --run <id>
snitch log --type committed --search "refactor"
snitch log --watch
```

### `snitch dashboard`

Interactive TUI with live updates:

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
brew services stop snitch
brew uninstall snitch
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
