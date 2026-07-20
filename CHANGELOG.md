# Changelog

## 0.5.0 — 2026-07-20

Epistemic verification, calmer flagging when evidence is missing, and Snitch Bar / IPC hardening. Upgrade replaces CLI, `snitchd`, and Snitch Bar together.

### Added

- Per-claim **epistemic** status (`supported`, `contradicted`, `missing`) stored in SQLite (migration `010_claim_epistemic`) with backfill from legacy `verified`
- Draft **Snitch Verification Protocol (SVP)** spec under `docs/spec/`
- Claim **Status** line in `snitch log` / dashboard detail (Supported / Contradicted / Unverified)
- `NewToolCall` preserves harness-native tool names (`raw_name`) while normalizing to Snitch’s canonical vocabulary
- Codex **project cwd** resolved from rollout `session_meta` when path metadata is empty
- Pi **project cwd** decoding that prefers real directory boundaries on disk (embedded dashes in segment names)
- OpenCode session polling can load **only messages after** the last poll cursor
- Subagent transcript parsing uses the **parent harness parser** (not Cursor-only)
- IPC tests for subscribe cleanup and explicit `shared: false` on `set_label`

### Changed

- Verifiers distinguish **contradicted** claims from **missing evidence**; unverifiable shell/test output is usually **WARN (missing)** instead of FAIL
- Run verdicts and stats use epistemic **contradicted** (not “could not verify”) for false-claim counts
- Unified shell/git evidence helpers (`shell_eval.go`); shell verifier no longer delegates through `ContradictionVerifier`
- Snitch Bar **single-flight** daemon start; reconnect no longer spawns overlapping `startWatching` goroutines
- Snitch Bar **stop** only terminates `snitchd` it started — no `lsof` kill of unrelated daemons on the socket
- `set_label` / `add_missed_claim`: explicit **`shared: false`** overrides `telemetry.share_by_default`
- IPC **subscribe** unsubscribes and closes channels when the client disconnects
- Broader **test-file** path heuristics in consistency checks (`__tests__/`, `.spec.`, etc.)
- User guide and harness extension docs updated for epistemic behavior

### Fixed

- IPC subscribe connections could leak subscriber goroutines after disconnect
- False positives when agents claimed success but shell output was not captured (now **missing**, not contradicted, where appropriate)

## 0.4.2 — 2026-07-10

Patch release: fixes false `tool_shell` flags on read-only git commands, plus license and docs updates.

### Fixed

- `tool_shell` no longer treats every `git` command as a commit claim — `git diff`, `git status`, `git log`, etc. verify as normal shell commands instead of failing with “claimed commit but no commit evidence”

### Changed

- License switched from MIT to **Apache License 2.0** (patent grant + NOTICE)
- README marks **Help train Snitch** as coming soon; adds Snitch Bar and notification screenshots
- Dependency updates: cobra, bubbletea, fsnotify; GitHub Actions (checkout, setup-go, goreleaser-action)


Hotfix for Snitch Bar notification authorization after Homebrew upgrades.

### Fixed

- Bundle script adhoc-codesigns `Snitch Bar.app` as `dev.snitch.menubar` with Info.plist bound — fixes `UNErrorDomain Code=1` (NotificationsNotAllowed) when linker-only signatures left `Identifier=a.out`

## 0.4.0 — 2026-07-09

Claim-first UX, clearer findings, and modern macOS notifications. Homebrew/curl upgrades replace CLI + daemon + Snitch Bar together.

### Added

- `internal/claims` display helpers (`FromRecord`, Flagged/Checked, shared arrow format)
- `internal/shellpreview` for compact shell one-liners in UI
- Nested `top_claim` on `run.completed` (includes `source` / `target`)
- IPC `get_latest_top_claim` for menu preview
- SQLite migration `009` + telemetry `0003` normalize tool claim types to `tool_*`
- `no_action` / consistency claims attach real flagged prose (`claim_sentence` / `claim_context`)

### Changed

- Product language: “lie” → “claim” / “false claim” in UI, CLI, and IPC
- Status field: `most_common_false_claim_type` (preferred) alongside legacy `top_claim_type`
- Snitch Bar notifications use `UNUserNotificationCenter` (auth + foreground banners)
- Tool claim types are `tool_*` snake_case end-to-end

### Fixed

- Notification title/body race on async main-queue deliver
- Menu “latest claim” no longer depends on changing global `GetClaims` ordering

### Compatibility (upgrade-safe)

- `--lies-only` still works (hidden alias of `--false-claims-only`)
- IPC still accepts `lies_only` on `get_claims`
- `run.completed` still emits flat `top_claim_*` fields alongside nested `top_claim`
- Status still includes `top_claim_type` alongside `most_common_false_claim_type`


## 0.3.2 — 2026-07-09

Flywheel training-payload plumbing for a future opt-in classifier, with the labeling UI still gated off for public builds.

### Added

- Claim sentence + capped ±1–2 sentence context captured at prose extraction and stored on claims
- Opt-in sync fields: `claim_sentence`, `claim_context`, `claimed`, `actual` (scrubbed); `claimed_text_hash` is the sentence hash
- Telemetry server migration + edge function support for training text columns

### Changed

- Docs disclose the exact opt-in training payload and hard exclusions (no prompts, code, paths, or full transcripts)
- Snitch Bar consent/share copy prepared for sentence + context sharing; menu stays behind `flywheelUIEnabled = false`

### Removed

- Public `docs/launch-checklist.md` (local-only via `.gitignore`)

## 0.3.1 — 2026-07-09

Multi-harness lie detection, Snitch Bar–owned notifications with the Snitch app icon, and UX/reliability fixes. Community labeling sync is reserved for a follow-up release.

### Added

- **Multi-harness ingestion** — Claude Code, Codex, Pi (JSONL), and OpenCode (SQLite) alongside Cursor; registry-driven and opt-in per platform (`snitch config set platforms.<name>.enabled true`)
- **Session lookback** — verification can credit evidence from up to three prior turns in the same session (git, shell, file tools, stubs); recap/summary prose is severity-calibrated
- **Subagent merge** — Cursor `subagents/*.jsonl` tool calls overlapping the parent turn window are merged into verification context
- **Data flywheel (coming soon)** — community labeling and anonymous sync reserved for a follow-up release
- `snitch replay <path>` — run transcripts through the verification pipeline offline against a throwaway database
- `snitch log --harness <name>`, `snitch status --detailed` (per-harness run counts), per-harness `snitch doctor` checks
- Idle-flush in the transcript watcher so a session’s final turn is captured without an explicit end marker
- Claim-pattern registry with CI-enforced example/negative tests; contributor guides for adding harnesses and patterns
- OSS hygiene — issue templates, CODEOWNERS, dependabot, Code of Conduct, security policy updates
- Shared `internal/textutil` helpers for consistent truncation across CLI, menu, and notifications
- Bundled **AppIcon.icns** from `assets/snitch_head.png` so Notification Center / Finder show the Snitch head

### Changed

- **Notifications move to Snitch Bar** — macOS alerts are delivered from the app bundle (CGO + `NSUserNotification`) so the Snitch icon is used instead of Script Editor / `osascript`; `snitchd` no longer posts notifications
- Verified-run events carry top-lie fields (`top_claim_type`, `top_claimed`, `top_actual`); Snitch Bar calls `notify.Deliver` with a cached notifications config (no extra `get_config` / `get_run` round-trips)
- Snitch Bar menu redesign — disabled **Latest:** preview, **View Details…**, and **History ▸ Open Dashboard…**
- Single `turnAssembler` encodes every harness’s turn-boundary semantics (watcher, OpenCode reader, and replay)
- Turn snapshots are secret-scrubbed before persistence (previously only combined output was scrubbed)
- Question / conditional / modal phrasing is suppressed for all claim types (fewer false positives)
- Config: legacy top-level `cursor:` block removed; use `platforms.cursor`
- Config structs expose stable lowercase `json` tags for IPC `get_config`
- Dashboard TUI — stacked layout on narrow terminals, single-line list rows, visible selection highlight, shared layout metrics
- IPC scanners use an 8 MiB buffer so large `get_runs` / `get_claims` responses are not truncated
- Watcher catch-up on directory create ingests owned files in that directory only; nested dirs get their own Create events
- `snitch doctor` resolves Snitch Bar.app the same way as `snitch start` (including Homebrew Cellar / `SNITCH_BAR_APP`)
- Bundle script fails loudly if `sips` / `iconutil` cannot build `AppIcon.icns` when the source PNG exists

### Fixed

- Last turn of a session was lost for harnesses without a trailing end marker
- OpenCode poll cursor could skip late-completing turns and emit in-progress partials
- `snitch label` required a doubled `label label` invocation
- `daemon.log_level` was loaded but never applied
- Burst create+write of new session transcripts could be seeded at EOF and skipped
- Dashboard layout cut off text / left large blank gaps on typical terminal sizes
- Dead / stub dashboard project-filter (`p` key) and unreachable verdict display branch removed

## 0.1.4 — 2026-07-03

### Fixed

- Homebrew upgrade failed with `No such file or directory - Snitch Bar.app` because release tarballs flattened the app bundle; install block now accepts flat `Contents/` layout and archives preserve `Snitch Bar.app/` again

## 0.1.3 — 2026-07-03

### Changed

- Removed `snitch lies` CLI command; use `snitch dashboard` (lies tab) to browse lie history
- `snitch log` now requires `--run <id>` (detail-only); list/watch modes removed
- Menu bar: **Browse Lies…** renamed to **Open Dashboard…** (opens `snitch dashboard`); **Show Last Lie** opens `snitch log --run <id>`
- Menu-bar-first docs and messaging: **Start/Stop Snitching** (not "Watching"); lie preview removed from dropdown
- `daemonNotRunning()` and `snitch doctor` point users to Snitch Bar / Start Snitching
- curl install no longer leaves orphan `snitchbar` binary on PATH (app bundle only)
- Homebrew: open `$(brew --prefix)/opt/snitch/Snitch Bar.app`; post_install registers menubar plist and removes legacy daemon agent
- Release archives include **Snitch Bar.app**; legacy `com.snitch.daemon.plist` moved to `install/macos/legacy/`
- ARCHITECTURE.md documents Snitch Bar, notifications, and menu bar flow

## 0.1.1 — 2026-07-03

### Added

- **Snitch Bar** menu bar app (`snitchbar`) — alerts on new lies, no Dock icon
- macOS Notification Center alerts from `snitchd` when a lie is caught (configurable)
- `internal/notify` package and `notifications` config block
- Menubar LaunchAgent (`com.snitch.menubar`) in curl installer (default on)
- **Start/Stop Snitching** — Snitch Bar starts/stops `snitchd`; daemon bundled inside `Snitch Bar.app`

### Changed

- Snitch Bar owns the daemon lifecycle (no separate `brew services` / daemon LaunchAgent)
- Install script and README promote menu bar + `snitch lies` as primary UX
- `log`, `dashboard`, and `doctor` de-emphasized to advanced/debug use
- CI: upgrade `golangci-lint-action` to v9 (Node 24)

## 0.1.0 — 2026-07-02

### Added

- Parse `tool_result` blocks and correlate to `tool_use` by ID
- Real `test_pass` / `command_succeeded` verification from captured shell output (transcript results + Cursor terminal files)
- `stub` detection for placeholder implementations
- Consistency checks: `self_contradiction`, `count_mismatch`, `negation_violation`
- `snitch doctor` and `snitch uninstall` commands
- Homebrew tap auto-publish via goreleaser (`fristovic/homebrew-snitch`)

## 0.0.2 — 2026-07-01

Fix database migration for users upgrading from an older `~/.snitch/snitch.db` schema.

## 0.0.1 — 2026-07-01

Initial public release: Snitch as a Cursor prose lie detector for macOS.

- Watches `~/.cursor/projects/**/agent-transcripts/*.jsonl`
- Extracts high-confidence claims from assistant prose
- Flags contradictions against tool calls, filesystem, and git evidence
- CLI: `snitch lies`, `snitch log`, `snitch status`, `snitch dashboard`
