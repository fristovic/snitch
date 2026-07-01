# Snitch User Guide

## Requirements

- macOS
- [Cursor](https://cursor.com) with agent transcripts under `~/.cursor/projects/`

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/fristovic/snitch/main/scripts/install.sh | bash
```

Verify:

```bash
snitch status
```

## How it works

When a Cursor agent turn ends, Snitch:

1. Reads the transcript JSONL for that turn
2. Extracts claims from assistant **prose** (not just tool calls)
3. Cross-checks against tool calls, files on disk, and git
4. Records lies in `~/.snitch/snitch.db`

A **snitch** is a high-confidence prose claim that evidence contradicts.

## Commands

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

Shows total runs, snitched count, top lie type, projects and sessions seen.

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
- **Push claims** cap at WARN when no `git push` shell call is visible
- **Test output** in the transcript is not re-parsed; "tests pass" without a test command is the flagship check
- Deterministic only — no semantic/LLM verification
