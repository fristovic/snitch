# Changelog

## Unreleased

### Added

- **Multi-harness ingestion** — Claude Code, Codex, Pi (JSONL), and OpenCode (SQLite) alongside Cursor; registry-driven, opt-in per platform (`snitch config set platforms.<name>.enabled true`)
- **Data flywheel** — `snitch label <run-id> correct|incorrect`, `snitch label missed`, Snitch Bar 👍/👎 + "Report Missed Lie…" + "Share labels anonymously" checkbox
- **Opt-in telemetry** — metadata-only labeled-verdict sync (claim type, harness, model, verdict, label, claim-text hash) to the training pipeline; off by default
- `snitch replay <path>` — run transcripts through the verification pipeline offline against a throwaway database
- `snitch log --harness <name>`, `snitch status --detailed` per-harness run counts, per-harness `snitch doctor` checks
- Idle-flush in the transcript watcher so a session's final turn is captured even without an explicit end marker
- Claim-pattern registry with CI-enforced example/negative tests; contributor guides for adding harnesses and patterns

### Changed

- Single `turnAssembler` now encodes every harness's turn-boundary semantics (watcher, OpenCode reader, and replay all share it)
- Turn snapshots are secret-scrubbed before persistence (previously only combined output was scrubbed)
- Question/conditional/modal phrasing is suppressed for all claim types (fewer false positives)
- Config: legacy top-level `cursor:` block removed; use `platforms.cursor`

### Fixed

- Last turn of a session was lost for harnesses without a trailing end marker
- OpenCode poll cursor could skip late-completing turns and emit in-progress partials
- `snitch label` required a doubled `label label` invocation
- `daemon.log_level` was loaded but never applied

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
