# Contributing to Snitch

Thank you for contributing. Read [ARCHITECTURE.md](ARCHITECTURE.md) before opening a PR.
Please follow the [Code of Conduct](CODE_OF_CONDUCT.md).

## Development

macOS required for integration tests that touch Cursor transcript paths.

```bash
go build ./...
go vet ./...
go test -race ./...
gofmt -l .        # must be clean
```

## The two most valuable contributions

### 1. Adding a claim pattern (teaching Snitch a new false-claim pattern)

Full guide: **[docs/extending-patterns.md](docs/extending-patterns.md)**

The short version: one entry in the `claimPatterns` registry in
`internal/verify/prose.go`, with at least 3 example sentences and 2 negative
sentences. `TestClaimPatternRegistry` **fails CI** if a pattern ships without
working examples and negatives — testing is mandatory by construction, not
convention. Patterns must be **high precision**: a pattern that false-positives
on the maintainers' replay corpus is rejected regardless of what it catches.

Validate against your own sessions before opening the PR:

```bash
go run ./cmd/snitch replay ~/.cursor/projects --false-claims-only
```

### 2. Adding a harness (supporting a new AI agent)

Full guide: **[docs/extending-harnesses.md](docs/extending-harnesses.md)**

A harness touches exactly five places (parser, path resolver, descriptor,
config, tests) and requires a realistic fixture plus a replay run against a
real on-device session. Never guess a wire format — verify every field against
real session files.

## Review gates

- `CODEOWNERS` requires maintainer review on the pattern registry, harness
  descriptors, and telemetry code.
- The PR template includes per-contribution checklists; incomplete checklists
  are not reviewed.

## Code style

- No ORM — raw SQL in `internal/record`
- No panics in production paths
- Structured logging with `slog`
- Deterministic verification only — no LLM calls in the verify path
- Privacy is a feature: nothing leaves the machine except opt-in, metadata-only
  telemetry. Any PR that widens what telemetry sends will be rejected.
