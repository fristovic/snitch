# Snitch

**Catch the model lying in prose.**

Snitch is a macOS daemon for [Cursor](https://cursor.com). It watches agent transcripts, extracts high-confidence claims from assistant text ("all tests pass", "I committed this"), and flags claims contradicted by evidence: tool calls, filesystem, and git.

No LLM. No proxy. Local SQLite only.

## Install

### macOS (one command)

```bash
curl -fsSL https://raw.githubusercontent.com/fristovic/snitch/main/scripts/install.sh | bash
```

From a cloned repo:

```bash
./scripts/install.sh
```

### macOS (Homebrew)

```bash
brew install --formula ./packaging/homebrew/snitch.rb
```

### What the installer does

1. Downloads or builds `snitch` + `snitchd`
2. Installs to `~/.local/bin`
3. Registers a LaunchAgent for `snitchd`

## Quick start

```bash
snitch status          # daemon + lie stats
snitch lies            # caught lies
snitch log --watch     # live failed runs
snitch dashboard       # interactive TUI
```

Snitch runs passively after install — it reads `~/.cursor/projects/**/agent-transcripts/*.jsonl`.

## CLI

| Command | Description |
|---------|-------------|
| `snitch status` | Daemon health and lie statistics |
| `snitch lies` | List caught lies (`--type`, `--project`, `--since`, `--json`) |
| `snitch log` | Run log with filters (`--type`, `--search`, `--since`, `--watch`) |
| `snitch dashboard` | Interactive filtering TUI |
| `snitch config` | View/set configuration |

## Lie types

| Type | Example prose | Contradiction |
|------|---------------|---------------|
| `test_pass` | "all tests pass" | No test `Shell` command in the turn |
| `committed` | "I committed" | No new commit since turn start |
| `pushed` | "I pushed" | No `git push` shell call |
| `file_created` | "created foo.go" | No matching `Write` + file missing |
| `no_action` | action claims | Zero tool calls in the turn |

## Documentation

- [User guide](docs/user-guide.md)
- [Architecture](ARCHITECTURE.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)

## License

MIT
