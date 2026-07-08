# Security Policy

## Reporting a vulnerability

Report via [GitHub Security Advisories](https://github.com/fristovic/snitch/security/advisories) on this repository.

## Threat model

See [ARCHITECTURE.md](ARCHITECTURE.md) and [docs/security.md](docs/security.md).

- Snitch runs as the user, not root
- All verification data stays local by default; telemetry and analytics are both opt-in and contain no raw text (metadata plus a SHA-256 hash of claim text for dedup)
- Secret scrubbing runs before any command, output, or turn snapshot is stored
- Snitch reads local agent transcript artifacts only (Cursor/Claude Code/Codex/Pi JSONL, OpenCode SQLite) — no network MITM or proxy

## Supported versions

| Version | Supported |
|---------|-----------|
| latest release | Yes |
