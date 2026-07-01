# Security Policy

## Reporting a vulnerability

Report via [GitHub Security Advisories](https://github.com/fristovic/snitch/security/advisories) on this repository.

## Threat model

See [ARCHITECTURE.md](ARCHITECTURE.md) and [docs/security.md](docs/security.md).

- Snitch runs as the user, not root
- All verification data stays local by default; analytics is opt-in and contains no raw text
- Secret scrubbing runs before any command or output is stored
- Snitch reads local Cursor transcript files only — no network MITM or proxy

## Supported versions

| Version | Supported |
|---------|-----------|
| 0.2.x   | Yes       |
