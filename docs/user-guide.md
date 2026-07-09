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

Look for the Snitch icon in the menu bar. Click it for status, **View Details…**, or **History ▸ Open Dashboard…**.

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
- **Latest: …** — disabled preview of the most recent lie (claim type + short quote)
- **View Details…** — open `snitch log --run <id>` for that lie
- **History ▸ Open Dashboard…** — browse runs and lies
- **Quit Snitch Bar** stops the daemon

If Snitch is paused or offline, choose **Start Snitching** to resume.

Disable login auto-start: `SNITCH_MENUBAR=0 ./scripts/install.sh`

### Labeling (coming soon)

Community labeling (Mark Correct / Incorrect, report missed lies, opt-in sync) is **coming soon**. When enabled, shared examples may include the claim sentence, short surrounding context, and claimed→actual — never prompts, code, or paths. See the README “Help train Snitch” section.

## Notifications

When a lie is caught, Snitch Bar can send a macOS notification (title: claim type, body: claimed → actual). Enabled by default; configure under `notifications` in `~/.snitch/config.yaml`. macOS will prompt for notification permission on the first alert.

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

### Reading `snitch log` output

Example:

```text
Run: 8179ec15-9b97-436f-afa1-d0f2e2794533
Verdict: warn
Session: c17d776a-…
Project: /Users/…/snitch
Tool calls: 11
Harness: cursor
Summary: "StrReplace …/menu_test.go" → "new_string not found in file"
  [StrReplace] StrReplace …/menu_test.go → new_string not found in file (sev 2, file)
```

| Field | Meaning |
| ----- | ------- |
| **Run** | One agent **turn** (user message → assistant reply → tools). UUID of the stored run. |
| **Verdict** | Overall outcome for the turn, from the highest claim severity (see below). |
| **Session** | Agent session id (many runs can share one session). |
| **Project** | Working directory Snitch associated with the transcript. |
| **Tool calls** | How many tools the agent invoked in that turn. |
| **Harness** | Which agent platform produced the transcript (`cursor`, `claude`, …). |
| **Prompt** | Truncated user prompt for the turn (when present). |
| **Summary** | Short `claimed → actual` lines for the notable claims. |
| **`[ClaimType]` line** | One claim Snitch checked. Format: `[type] claimed → actual (sev N, verifier)`. |

#### Verdicts

| Verdict | When you see it |
| ------- | ---------------- |
| **pass** | No material contradictions (max severity ≤ 1). |
| **warn** | Partial / medium problems (max severity 2) — e.g. a tool edit that didn’t land, or a softer mismatch. Menu-bar alerts and notifications default to **fail** only. |
| **fail** | High-confidence false claim (max severity 3) — typically prose like “all tests pass” contradicted by evidence. |

#### Severity (`sev`)

| sev | Label | Typical meaning |
| --- | ----- | ---------------- |
| **0** | verified | Claim matches evidence. |
| **1** | minor inaccuracy | Soft mismatch; usually hidden in `snitch log` unless interesting. |
| **2** | partial failure | Real problem, but not always a clear prose lie (often tool/file/shell). → run **warn** |
| **3** | false claim | Strong contradiction. → run **fail** |

#### Claim types (common)

| Type | What Snitch thought the agent claimed |
| ---- | -------------------------------------- |
| **test_pass** | Tests passed / suite is green. |
| **committed** / **pushed** | Git commit or push happened. |
| **file_created** / **file_modified** / **file_deleted** | A file was created, edited, or removed. |
| **no_action** | Prose claimed an action, but the turn had no mutating tools. |
| **stub** | “Done / fully implemented” while written code still has TODO/panic stubs. |
| **Shell**, **StrReplace**, **Write**, **Delete**, … | Tool-call claims — Snitch checked that the tool’s effect matches disk/output. |

`(prose)` after the type means the claim came from assistant text, not only from a tool name.

#### Verifier (the last field)

| Verifier | What it checked |
| -------- | ---------------- |
| **contradiction** | Prose claim vs tools / git / filesystem / lookback. |
| **file** | File tool results vs contents on disk. |
| **shell** | Shell / test / build claims vs command evidence. |
| **consistency** | Same-turn contradictions (e.g. conflicting statements). |
| **subagent** | Parent turn vs merged subagent tool evidence (Cursor). |

#### `claimed → actual`

- **Left** — what the agent said or what the tool implied.
- **Right** — what Snitch found instead.

So `"StrReplace …" → "new_string not found in file"` means the edit tool was used, but the new text never appeared on disk.

### `snitch dashboard`

Interactive TUI with live refresh (`--harness` to filter):

- `tab` — switch runs / lies view
- `v` — cycle verdict filter
- `t` — cycle claim type filter
- `/` — search
- `q` — quit

### `snitch status`

Shows daemon health and enabled harnesses. Use `--detailed` for lie statistics, per-harness run counts, and recent failures.

### `snitch label` (coming soon)

Community labeling and opt-in training sync are **coming soon**. Shared payloads (when enabled) include claim sentence + capped context + claimed→actual; never prompts, code, or paths.

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
  enabled: false            # reserved for upcoming labeling sync
  share_by_default: false
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
