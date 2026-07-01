# Contributing to Snitch

Thank you for contributing. Read [ARCHITECTURE.md](ARCHITECTURE.md) before opening a PR.

## Development

macOS required for integration tests that touch Cursor transcript paths.

```bash
go build ./...
go vet ./...
go test -race ./...
```

## Adding prose claim patterns

Edit `internal/verify/prose.go`. Patterns must be **high precision** — false positives destroy trust. Add tests in `prose_test.go` and contradiction tests in `verifiers/contradiction_test.go`.

## Code style

- No ORM — raw SQL in `internal/record`
- No panics in production paths
- Structured logging with `slog`
- Deterministic verification only — no LLM calls in the verify path
