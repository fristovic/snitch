# Snitch v3 — Implementation Plan

> **Stack:** Go 1.24+, SQLite (modernc.org/sqlite), fsnotify, Bubble Tea TUI, Cobra CLI
>
> **Scope:** Refactor from harness-agnostic v2 to Cursor-specific v3. Strip ~1,000 lines of dead code, rewrite transcript sensor and verification engine around Cursor's structured JSONL format. macOS-only for v3. Keep and adapt: event bus, IPC, SQLite store, config, scrubber, CLI/TUI, analytics reporter.

---

## Workstream A: Strip Dead Code

Delete all packages and files that won't exist in the Cursor-only architecture. This must happen first — subsequent workstreams operate on the slimmed codebase.

### A1 — Delete sensor layer
- Remove `internal/sensor/` entirely (process, transcript readers for non-Cursor, network proxy, filesystem, shell)
- Remove `internal/signatures/`
- Remove `cmd/snitch/cmd/proxy.go`
- Remove `internal/sensor/sensor.go`

### A2 — Delete unused verification code
- Remove `internal/verify/extractor.go` (LLM claim extraction)
- Remove `internal/verify/prompt.go` (LLM prompts)
- Remove `internal/verify/verifiers/semantic.go` (LLM verifier)
- Remove `internal/verify/verifiers/http.go`
- Remove `internal/verify/verifiers/llm.go`
- Merge `internal/verify/verifiers/test.go` into `shell.go` (test re-running = shell re-running)

### A3 — Delete unused platform and utility packages
- Remove `internal/repo/`
- Remove `internal/platform/shellprofile_*.go`
- Remove `internal/platform/exec.go`, `exec_windows.go`
- Remove all `*_linux.go` and `*_windows.go` files (macOS-only)
- Remove `internal/sensor/process/`, `internal/sensor/network/`, `internal/sensor/filesystem/`, `internal/sensor/transcript/readers/{claude,codex,aider}.go`

### A4 — Clean up imports and build
- Fix all broken imports in remaining packages
- Remove now-unused dependencies from `go.mod` (`go mod tidy`)
- Verify `go build ./...` succeeds with slimmed tree
- Delete old test files referencing deleted packages

---

## Workstream B: New Transcript Package

Build the single-source Cursor transcript watcher and parser.

### B1 — Create `internal/transcript/watcher.go`
- `CursorWatcher` struct with fsnotify watcher on `~/.cursor/projects/`
- Recursive directory watch with `agent-transcripts/**/*.jsonl` filter
- File offset tracking per transcript path (map[string]int64)
- On CREATE/WRITE events: read new lines, pass to parser, update offset
- Handle directory creation (new projects), file deletion (clean up offsets)
- Emit `TranscriptUpdated` events on the bus

### B2 — Create `internal/transcript/parser.go`
- Parse Cursor JSONL format: `cursorLine` struct as defined in existing `readers/cursor.go`
- Extract `tool_use` blocks into `ToolCall` structs
- Derive project path from transcript path (existing `ProjectCwdFromTranscriptPath` logic)
- Derive run ID from transcript directory name (UUID)
- Track session boundaries (new transcript file = new session)
- Handle malformed lines gracefully (skip, log DEBUG)
- Handle very large lines (10MB scanner buffer, as existing code does)

### B3 — Wire transcript watcher into daemon
- Add `CursorWatcher` to `cmd/snitchd/main.go` bootstrap
- Replace all four v2 sensors with single watcher goroutine
- Remove sensor start/stop loops

---

## Workstream C: Simplified Capture Engine

Remove snapshot diffing and multi-source assembly. Replace with direct transcript-to-claims pipeline.

### C1 — Rewrite `internal/capture/capture.go`
- Remove: `Snapshot` struct, `Delta` struct, `TakeSnapshot()`, `ComputeDelta()`, pre/post state capture
- Remove: `activeRun` tracking, PID correlation, `cwdIndex`, coverage tiers
- New: Listen for `TranscriptUpdated` events
- New: Build `RunPayload` directly from parsed turns and tool calls
- New: Emit `RunCaptured` with structured tool calls (no free-text output assembly)
- Keep: 10MB max size limit, scrubbing

### C2 — Remove `internal/capture/snapshot.go`
- Entire file deleted. No pre/post snapshots needed when verification is direct.

---

## Workstream D: Verification Engine Rewrite

Replace LLM-based claim extraction with structural tool-call-to-claim mapping. Rewrite verifiers for the new claim types.

### D1 — Rewrite `internal/verify/engine.go`
- Remove: LLM claim extraction pipeline, regex fallback, `RunCaptured` payload with free-text output
- New input: `RunPayload` with `[]ToolCall`
- New pipeline:
  1. Map each `ToolCall` → `Claim` (structural, no LLM)
  2. For each claim: find matching verifier via `CanHandle()`, run `Verify()`
  3. Aggregate results, compute max severity, produce verdict
  4. Emit `RunVerified` event
- Keep: max 3 concurrent verifications

### D2 — Rewrite `internal/verify/verifiers/verifier.go`
- Update `Claim` struct: add `ToolCall` field (string), rename `Action` → `ToolCallType`
- Keep: `Verifier` interface (`Name()`, `CanHandle()`, `Verify()`), `Result` struct

### D3 — Rewrite `internal/verify/verifiers/file.go`
- Handle: `Write`, `StrReplace`, `Read`, `Glob` tool calls
- `Write`/`StrReplace`: stat path, check non-empty content (compare first 10KB if config enabled), check mtime
- `Read`: stat path, check mtime ≤ claim timestamp
- `Glob`: re-run glob pattern, compare result count (loose match)
- Severity: Level3 for missing file on Write, Level2 for content mismatch, Level0 for match

### D4 — Rewrite `internal/verify/verifiers/shell.go`
- Handle: `Shell` tool calls
- Parse command string, validate syntax
- Opt-in re-run (per-project config): execute command with 30s timeout, compare exit codes
- Without re-run: verify command syntax only, mark `pending` verification status
- Severity: Level2 for exit code mismatch, Level1 for syntax issues

### D5 — Keep `internal/verify/verifiers/git.go`
- Already works from v2. Verify git-related claims from agent text.
- Minor adaptation: claims come from agent text turns, not LLM extraction.

### D6 — Create `internal/verify/verifiers/subagent.go`
- Handle: `Task` tool calls (subagent spawns)
- Check: subagent transcript exists at `subagents/<uuid>/<uuid>.jsonl` relative to parent transcript
- Check: subagent transcript is non-empty, contains tool calls
- Severity: Level2 for missing transcript, Level1 for empty transcript

### D7 — Update `internal/verify/severity.go`
- Remove `Level4` (Harmful) — not applicable without semantic understanding
- Keep `Level0`–`Level3`
- Update verdict mapping: PASS (all Level0-1), WARN (any Level2), FAIL (any Level3)

---

## Workstream E: Update Data Model

Adapt SQLite schema and record store for the new data shape.

### E1 — Update migration
- File: `internal/record/migrations/001_initial.sql`
- Remove columns: `coverage`, `command`, `harness_version`
- Add columns: `session_id TEXT`, `transcript_path TEXT`, `tool_call_count INTEGER`
- Rename `action` → `tool_call` in claims table
- Set `harness` default to `'cursor'`
- Remove `extracted_by` column from claims (always structural now)
- Remove `reprompted`, `parent_run_id` (deferred to future)

### E2 — Update `internal/record/models.go`
- Update `Run` struct with new columns
- Update `Claim` struct: `ToolCall string`, remove `ExtractedBy`
- Add `ToolCall` struct for Go-side representation

### E3 — Update `internal/record/store.go`
- Adapt `InsertRun()` and `InsertClaim()` for new fields
- Update query methods: `GetRuns()`, `GetRun()`, filter by `project_path`
- Add `GetRunsByProject(projectPath string)` method

---

## Workstream F: Update CLI, Config, and Docs

Adapt the user-facing surface to the Cursor-only model.

### F1 — Update CLI commands
- Remove: `proxy` command
- Update `status`: show Cursor watcher status (projects watched, transcripts found), remove coverage stats
- Update `log`: show tool call counts, project path, session ID; add `--project` filter
- Keep: `config`, `dashboard` (adapt data displayed)
- Remove `analytics.go` and `analyze.go`? — actually keep `analytics.go`, it's useful. Remove `analyze.go` only.

### F2 — Update config schema
- File: `internal/config/defaults.go`
- Remove: `sensors.*` config blocks
- Add: `cursor.transcript_watch_path`, `cursor.poll_interval_ms`
- Keep: `daemon`, `verification`, `analytics`, `retention`, `display`
- Remove: `test_verifier` (merged into `shell_verifier`)

### F3 — Update README.md
- Rewrite for Cursor-only focus
- Document: installation (macOS Homebrew), what Snitch watches, how verification works
- Remove: multi-harness claims, proxy setup instructions, shell hook instructions

### F4 — Update installers
- Simplify `install/macos/install.sh`: remove proxy CA step
- Keep: binary copy, LaunchAgent plist, config dir creation

---

## Workstream G: Tests

Rebuild test suite for the simplified architecture.

### G1 — Unit tests for transcript parser
- File: `internal/transcript/parser_test.go`
- Test: real Cursor JSONL fixtures (use existing `test/fixtures/sample_transcripts/cursor.jsonl` + add richer ones)
- Test: malformed lines, empty files, truncated JSON, very long lines
- Test: tool call extraction (Write, Read, Shell, Task, Grep, Glob, StrReplace)
- Test: project path derivation from transcript path

### G2 — Unit tests for verifiers
- File verifier: `internal/verify/verifiers/file_test.go` (update for new claim types)
- Shell verifier: new `shell_test.go`
- Subagent verifier: new `subagent_test.go`
- Git verifier: update existing `git_test.go` (if needed)

### G3 — Integration tests
- Full pipeline: transcript JSONL → parse → verify → SQLite record → IPC query
- Test with real Cursor transcript fixtures from Filip's machine (scrubbed of sensitive paths)

---

## Dependency Order

```
A (strip dead code) ─────────────────────────────────
│
├── B (transcript package) ──────────────────────────
│   │
│   ├── C (capture engine) ──────────────────────────
│   │   │
│   │   └── D (verification engine) ─────────────────
│   │       │
│   │       └── E (data model) ──────────────────────
│   │           │
│   │           └── F (CLI, config, docs) ───────────
│   │               │
│   │               └── G (tests) ─── parallel with F
│   │
│   └── G (transcript parser tests) ─── can start after B2
```

Workstream G can partially overlap: transcript parser tests (G1) can start after B2. Verifier tests (G2) need D. Integration tests (G3) need E.

---

## Independently Shippable Milestones

1. **A only** — Clean compile of stripped codebase. All dead code removed. `go build ./...` succeeds. No functional changes yet.

2. **+ B** — Transcript watcher running. New Cursor sessions detected and parsed. Tool calls extracted. `snitchd` can log "detected session X with N tool calls."

3. **+ C** — Capture engine producing `RunPayload` from parsed transcripts. Events flowing through the bus.

4. **+ D** — Verification engine running. Claims verified against filesystem/git/shell. Verdicts produced. This is the core value proposition — everything before this is plumbing.

5. **+ E + F** — Full pipeline end-to-end. SQLite records, CLI queries, config polished. `snitch status` shows real data. `snitch log --run <id>` shows claim verification.

6. **+ G** — Test suite passing. Ready for release.

---

## What's Intentionally NOT in Scope

| Deferred | Why | When |
|---|---|---|
| **Linux/Windows support** | macOS-only for v3. Cursor transcript paths are macOS-specific for now. | v3.1 |
| **Auto re-prompt** | Was v2 Phase 6. Not useful until verification is battle-tested. | v3.2 |
| **Multiple Cursor profiles** | Watching only default profile. Multi-profile support adds complexity without clear need. | v3.1 |
| **Web dashboard** | TUI only. Web UI is a separate project. | Never (separate project) |
| **Non-Cursor harnesses** | By design. Claude Code, Codex, aider not supported. | Never (Cursor-only is the point) |
| **Network proxy / CA cert** | Cursor transcripts capture everything. No API-level interception needed. | Never |
| **Process signatures** | No process polling. Transcript watcher is the only detection mechanism. | Never |
