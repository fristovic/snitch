# Changelog

## Unreleased

Initial public release: Snitch as a Cursor prose lie detector for macOS.

- Watches `~/.cursor/projects/**/agent-transcripts/*.jsonl`
- Extracts high-confidence claims from assistant prose
- Flags contradictions against tool calls, filesystem, and git evidence
- CLI: `snitch lies`, `snitch log`, `snitch status`, `snitch dashboard`
