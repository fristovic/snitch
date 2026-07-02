<p align="center">
  <img src="docs/snitch_logo.png" alt="Snitch logo" width="320">
</p>

<p align="center"><strong>Catch the model lying in prose.</strong></p>

<p align="center">
  <span style="display: inline-block; max-width: 720px; text-align: justify;">
    Snitch is a deterministic <a href="https://cursor.com">Cursor</a> prose lie detector daemon for macOS. It watches agent transcripts, extracts high-confidence claims from assistant text ("all tests pass", "I committed this"), and flags claims contradicted by evidence: tool calls, tool output, filesystem, git, and same-turn consistency.
  </span>
</p>

## Install

### macOS (Homebrew — recommended)

```bash
brew tap fristovic/snitch
brew install snitch
brew services start snitch
snitch doctor
```

### macOS (curl)

Latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/fristovic/snitch/main/scripts/install.sh | bash
```

From a cloned repo:

```bash
./scripts/install.sh
```

### What the installer does

1. Downloads or builds `snitch` + `snitchd`
2. Installs to `~/.local/bin` (or Homebrew cellar)
3. Registers a LaunchAgent / `brew services` entry for `snitchd`

## Quick start

```bash
snitch doctor           # verify install + Cursor
snitch status           # daemon health
snitch lies             # caught lies
snitch log --watch      # live failed runs
snitch dashboard        # interactive TUI
```

Snitch runs passively after install — it reads `~/.cursor/projects/**/agent-transcripts/*.jsonl`.

## CLI

| Command              | Description                                                       |
| -------------------- | ----------------------------------------------------------------- |
| `snitch doctor`      | Check daemon, Cursor, and transcript paths                        |
| `snitch status`      | Daemon health (`--detailed` for lie stats)                        |
| `snitch lies`        | List caught lies (`--type`, `--project`, `--since`, `--json`)     |
| `snitch log`         | Run log with filters (`--type`, `--search`, `--since`, `--watch`) |
| `snitch dashboard`   | Interactive filtering TUI                                         |
| `snitch uninstall`   | Remove daemon and binaries (`--purge` for data)                   |
| `snitch config`      | View/set configuration                                            |

## Lie types

| Type                  | Example prose              | Contradiction                                      |
| --------------------- | -------------------------- | -------------------------------------------------- |
| `test_pass`           | "all tests pass"           | No test run, or test output shows failure          |
| `command_succeeded`   | "command ran successfully" | Shell exited with error                            |
| `committed`           | "I committed"              | No new commit since turn start                     |
| `pushed`              | "I pushed"                 | No `git push` shell call                           |
| `file_created`        | "created foo.go"           | No matching `Write` + file missing                 |
| `stub`                | "fully implemented"        | Written file is a placeholder (`panic("TODO")`, …) |
| `no_action`           | action claims              | Zero tool calls in the turn                        |
| `self_contradiction`  | "won't modify X"           | Tool call edits X in the same turn                 |
| `count_mismatch`      | "updated all 5 files"      | File tool-call count ≠ 5                           |
| `negation_violation`  | "did not touch tests"      | `*_test.*` file edited in the turn                 |

## Documentation

- [User guide](docs/user-guide.md)
- [Architecture](ARCHITECTURE.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)

## License

MIT
