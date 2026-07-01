# Changelog

## Unreleased

## 0.0.2 — 2026-07-01

Fix database migration for users upgrading from an older `~/.snitch/snitch.db` schema.

## 0.0.1 — 2026-07-01

Initial public release: Snitch as a Cursor prose lie detector for macOS.

- Watches `~/.cursor/projects/**/agent-transcripts/*.jsonl`
- Extracts high-confidence claims from assistant prose
- Flags contradictions against tool calls, filesystem, and git evidence
- CLI: `snitch lies`, `snitch log`, `snitch status`, `snitch dashboard`
