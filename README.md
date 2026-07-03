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
| **Show Last Lie** | Open Terminal with full details for the most recent lie (`snitch log --run <id>`) |
| **Open Dashboard…** | Open Terminal with the interactive TUI (`snitch dashboard`) |
| **Preferences…** | Open `~/.snitch/config.yaml` |
| **Quit Snitch Bar** | Stop the daemon and exit |

When the model lies, the menu bar icon alerts and macOS may show a notification. Click **Show Last Lie** for the full verification breakdown, or **Open Dashboard…** to browse history.

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

**Menu bar shortcuts:** **Show Last Lie** runs `snitch log --run <id>` for the latest lie. **Open Dashboard…** runs `snitch dashboard`.

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
| **Show Last Lie** | Open `snitch log --run <id>` for the latest lie |
| **Open Dashboard…** | Open `snitch dashboard` in Terminal |
| **Preferences…** | Edit `~/.snitch/config.yaml` |
| **Quit Snitch Bar** | Stop `snitchd` and exit |

### Terminal (CLI)

| Command | Description |
| ------- | ----------- |
| `snitch start` | Open Snitch Bar |
| `snitch status` | Detection status (`--detailed` for stats) |
| `snitch log --run <id>` | Full verification detail for one run (`--trace`, `--json`) |
| `snitch dashboard` | Interactive TUI for runs and lies |
| `snitch doctor` | Debug install checklist |
| `snitch uninstall` | Remove daemon and binaries (`--purge` for data) |
| `snitch config` | View/set configuration |


Snitch runs passively after install — it reads `~/.cursor/projects/**/agent-transcripts/*.jsonl`.

### Notifications

When `snitchd` catches a lie, macOS Notification Center can alert you (enabled by default). Configure in `~/.snitch/config.yaml`:

```yaml
notifications:
  enabled: true
  on_warn: false
  rate_limit_s: 5
```

The first notification triggers the macOS permission prompt.

## Lie types


| Type                 | Example prose              | Contradiction                                      |
| -------------------- | -------------------------- | -------------------------------------------------- |
| `test_pass`          | "all tests pass"           | No test run, or test output shows failure          |
| `command_succeeded`  | "command ran successfully" | Shell exited with error                            |
| `committed`          | "I committed"              | No new commit since turn start                     |
| `pushed`             | "I pushed"                 | No `git push` shell call                           |
| `file_created`       | "created foo.go"           | No matching `Write` + file missing                 |
| `stub`               | "fully implemented"        | Written file is a placeholder (`panic("TODO")`, …) |
| `no_action`          | action claims              | Zero tool calls in the turn                        |
| `self_contradiction` | "won't modify X"           | Tool call edits X in the same turn                 |
| `count_mismatch`     | "updated all 5 files"      | File tool-call count ≠ 5                           |
| `negation_violation` | "did not touch tests"      | `*_test.*` file edited in the turn                 |




## Documentation

- [User guide](docs/user-guide.md)
- [Architecture](ARCHITECTURE.md)
- [Contributing](CONTRIBUTING.md)
- [Security](SECURITY.md)



## License

MIT
