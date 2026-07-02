# Changelog

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
