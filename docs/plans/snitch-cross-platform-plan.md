# Snitch: Cross-Platform + Monetization Plan

## Current State Summary

**What's built:** Cursor-only daemon. Watches `~/.cursor/projects/**/agent-transcripts/*.jsonl`, parses Cursor's JSONL format (separate lines per content block: `text`, `tool_use`, `tool_result`, `turn_ended`), extracts claims via regex, verifies against 5 verifiers (Contradiction, Consistency, File, Shell, Subagent), stores in SQLite, exposes via Unix socket IPC. CLI with TUI dashboard (`snitch dashboard`), menubar app (`snitchbar`). ~3,400 lines Go, MIT licensed.

**What's hardcoded to Cursor:**
- `internal/transcript/parser.go` — parses Cursor's `cursorMessage` struct with `role`/`message.content[]` where each content block gets its own JSONL line, and `turn_ended` type marks boundaries
- `internal/transcript/watcher.go` — watches `~/.cursor/projects/`, assumes Cursor project-slug-to-filesystem-path decoding (`DecodeProjectSlug`), assumes `agent-transcripts` subdirectory layout
- `internal/transcript/path.go` — `SessionIDFromTranscriptPath`, `ProjectCwdFromTranscriptPath` are Cursor-path-specific
- `internal/capture/capture.go` — hardcodes `Harness: "cursor"` on line 99
- `internal/verify/verifiers/verifier.go` — `ToolCallToClaim` maps Cursor-specific tool names: `Write`, `StrReplace`, `Delete`, `Read`, `Glob`, `Shell`, `Task`
- `cmd/snitchd/main.go` — instantiates `transcript.NewWatcher(bus, cfg.Cursor)`, single platform
- `internal/config/config.go` — `CursorConfig` struct assumes single platform config

**What's already platform-agnostic:**
- `internal/verify/engine.go` — doesn't care about harness source
- `internal/record/` — SQLite schema already has `harness` column
- `internal/event/` — generic pub/sub, no Cursor assumptions
- `internal/ipc/` — generic JSON-RPC, harness-agnostic queries
- `internal/scrub/`, `internal/severity/`, `internal/notify/` — fully generic
- All verifiers (`internal/verify/verifiers/`) — use abstracted `VerifyContext` + `ToolCall`, don't touch Cursor directly except `ToolCallToClaim` name mapping

**Claude Code transcript format (confirmed from ~/.claude/projects/):**
- Location: `~/.claude/projects/<project-slug>/<session-uuid>.jsonl`
- Structure: Each line is a self-contained message. Types: `user`, `assistant`, `system`, `attachment`, `mode`, `file-history-snapshot`
- User messages: `{type:"user", message:{role:"user", content:"text or array"}}`
- Assistant messages: `{type:"assistant", message:{role:"assistant", content:[{type:"thinking",...}, {type:"text",text:"..."}, {type:"tool_use",id,name,input}]}}` — all blocks in ONE line
- Tool results: Embedded in subsequent `type:"user"` messages as `message.content:[{type:"tool_result", tool_use_id, content:[{type:"text",text:"..."}]}]`
- Session boundaries: No explicit turn_ended marker — turns are implicit (user→assistant→user)
- Key difference from Cursor: No separate `tool_result` lines. No `turn_ended` marker. Assistant's thinking+text+tool_use all in one message. Tool names different: `Bash` not `Shell`, `Write`/`Read`/`Edit` for files, `Agent` for subagents, `WebSearch`/`WebFetch`.

**Codex transcript format (to be confirmed during implementation):**
- Codex uses a session directory structure similar to Claude Code
- Expected to produce JSONL with similar message/role/tool_use/tool_result concepts
- Tool names will differ (likely `Shell` or `bash`, `Write`/`Read`, etc.)
- Exact path and schema need discovery during Phase 1.1

---

## Phase 1: Cross-Platform (MIT, open-source)

### Workstream A: Abstract the transcript layer

**Goal A1 — Platform-agnostic parser interface**

Define a `TranscriptParser` interface that returns the same `ParsedLine` struct regardless of platform. The current Cursor parser becomes one implementation.

```go
// internal/transcript/parser.go (refactored)
type TranscriptParser interface {
    ParseLines(path string, fromOffset int64) (lines []ParsedLine, newOffset int64, err error)
}

// ParsedLine stays mostly the same — all platforms have:
// - Role (user/assistant)
// - Text (prose)
// - ToolCalls
// - ToolResults
// - Session boundary markers
type ParsedLine struct {
    Role        string
    Text        string
    ToolCalls   []ToolCall
    ToolResults []ToolResult
    TurnEnded   bool     // true when a turn completes
    TurnStatus  string
}
```

**Goal A2 — Implement Claude Code parser**

New file: `internal/transcript/parser_claude.go`

Parse Claude Code's JSONL format:
- `type:"assistant"` → extract `message.content[].text` for prose, `message.content[].tool_use` for tool calls
- `type:"user"` with `message.content[].tool_result` → extract tool results
- Detect turn boundaries: a `type:"user"` message with actual content (not just tool results) starts a new turn; the preceding assistant + its tool results complete the previous turn
- Handle `thinking` blocks by skipping them (not prose, not actionable)
- Map Claude tool names to Snitch's internal names: `Bash`→`Shell`, `Edit`→`StrReplace`, `Agent`→`Task`

**Goal A3 — Implement Codex parser**

New file: `internal/transcript/parser_codex.go`

Assumes JSONL with message/role/tool_use structure. Exact schema to be confirmed from real Codex session files. Will follow the same pattern as Claude Code parser.

**Goal A4 — Abstract the watcher**

Refactor `internal/transcript/watcher.go` to support multiple platform watchers:

```go
type WatcherConfig struct {
    Name       string   // "cursor", "claude", "codex"
    RootPath   string   // ~/.cursor/projects, ~/.claude/projects, etc.
    Parser     TranscriptParser
    PathDecoder func(string) string  // transcript path → project filesystem path
    FileGlob   string   // e.g. "agent-transcripts/*.jsonl", "*.jsonl"
}
```

Each platform gets its own watcher instance with its own config. The daemon starts all three.

**Goal A5 — Platform-specific path utilities**

New files:
- `internal/transcript/path_cursor.go` — move existing `ProjectCwdFromTranscriptPath`, `DecodeProjectSlug` here
- `internal/transcript/path_claude.go` — Claude uses same slug encoding (`-Users-filipristovic-...`), can reuse `DecodeProjectSlug`; sessions are flat JSONL files named `<uuid>.jsonl` directly under the project directory (no `agent-transcripts/` subdirectory)
- `internal/transcript/path_codex.go` — TBD based on actual Codex layout

### Workstream B: Update the verification pipeline

**Goal B1 — Dynamic tool name mapping**

Replace hardcoded `ToolCallToClaim` in `verifier.go` with a harness-aware mapper:

```go
// Each harness registers its tool-name→ClaimType mapping
var harnessToolMaps = map[string]map[string]ClaimType{
    "cursor": {"Write": "Write", "Shell": "Shell", "StrReplace": "StrReplace", ...},
    "claude": {"Write": "Write", "Bash": "Shell", "Edit": "StrReplace", ...},
    "codex":  {"write_to_file": "Write", "run_shell_command": "Shell", ...},
}

func ToolNameToClaimType(harness, toolName string) (ClaimType, bool) { ... }
```

**Goal B2 — Harness-aware tool call target extraction**

Current `deriveTarget` in parser.go looks for `path`, `command`, `glob_pattern` keys. Claude Code and Codex may use different key names (e.g. `file_path` instead of `path`). Add harness-specific key lookups.

### Workstream C: Configuration

**Goal C1 — Multi-platform config**

Replace `CursorConfig` with a `PlatformConfig` list:

```yaml
# ~/.snitch/config.yaml
platforms:
  cursor:
    enabled: true
    transcript_watch_path: ""  # defaults to ~/.cursor/projects
  claude:
    enabled: true
    transcript_watch_path: ""  # defaults to ~/.claude/projects
  codex:
    enabled: true
    transcript_watch_path: ""  # TBD
```

**Goal C2 — `snitch config` CLI**

Update `snitch config` to support per-platform enable/disable and path override.

### Workstream D: CLI updates

**Goal D1 — Platform filter in queries**

Add `--harness` filter to `snitch lies`, `snitch log`, `snitch dashboard` so users can filter by platform.

**Goal D2 — Status shows all platforms**

```
$ snitch status
  Running: true  Uptime: 2h 14m
  Platforms:
    cursor  ✓ watching (3 projects, 47 runs)
    claude  ✓ watching (5 projects, 128 runs)
    codex   - disabled
```

### Workstream E: Tests

**Goal E1 — Fixtures for each platform**

Add sample JSONL fixtures for Claude Code and Codex in `test/fixtures/sample_transcripts/`.

**Goal E2 — Parser tests**

Unit tests for `parser_claude.go` and `parser_codex.go` covering: assistant with text only, assistant with tool_use, tool_result handling, multi-turn sessions, thinking block exclusion, error handling.

**Goal E3 — Integration test**

Extend `test/integration/pipeline_test.go` to run the full Transcript→Capture→Verify pipeline with Claude Code and Codex fixtures.

---

## Phase 2: Snitchworks (closed-source, paid)

Snitchworks is a separate repository/binary. It reads the same local SQLite databases that Snitch writes, but adds centralized team features. The open-source Snitch daemon remains MIT and unchanged — Snitchworks is an add-on, not a replacement.

### Workstream F: Snitchworks daemon (closed-source)

**Goal F1 — Remote IPC bridge**

A lightweight sidecar daemon (`snitchworksd`) that runs alongside `snitchd` on each developer machine. It:
- Connects to `snitchd`'s Unix socket (subscribe to `run.completed` events)
- Forwards verified runs to the Snitchworks cloud API (encrypted, authenticated)
- Caches locally if offline, syncs when reconnected
- Minimal footprint — the heavy processing stays on the server

**Goal F2 — Team authentication**

- Developer installs `snitchworksd`, authenticates via CLI (`snitchworks login`)
- Machine gets a team-scoped API key stored in local config
- All forwarded data tagged with team ID + device ID

### Workstream G: Snitchworks cloud (closed-source, SaaS)

**Goal G1 — Team dashboard web app**

Next.js app (following Soothsayers stack: Next.js 15, Tailwind v4, Supabase) that provides:
- **Team overview**: aggregate lie rates across all developer machines, trend charts, top lie types
- **Per-developer view**: individual stats, pattern detection ("dev A's Claude Code lies 3x more than dev B's")
- **Per-project view**: which projects have the highest lie density
- **Lie feed**: real-time stream of caught lies across the team (filterable by severity, type, platform, developer)
- **Configuration management**: push verification policies to all team machines

**Goal G2 — Analytics engine**

- **Model comparison**: "Which model lies least?" — aggregate stats by model (Claude Opus vs GPT-4o vs DeepSeek, extracted from assistant message metadata)
- **Platform comparison**: "Cursor agents lie 12% more than Claude Code agents on `test_pass` claims"
- **Lie pattern detection**: ML-based clustering to find common failure modes "agents that say 'all tests pass' without running them tend to also claim 'I committed this' without committing"
- **Weekly digest emails**: "Your team's AI agents made 847 claims this week, 23 were false (2.7%)"

**Goal G3 — LLM-based semantic verifier (premium feature)**

Regex catches "all tests pass" with high precision but misses subtler lies entirely. The LLM verifier:
- Runs a cheap model (DeepSeek V4 Flash, ~$0.01 per 100 claims) on every prose claim
- Checks: "Does the agent's own tool output actually support this claim?"
- Example catches: "I refactored the auth module to use JWT" → file still uses sessions → regex misses this, LLM catches it
- Higher recall, slightly lower precision than regex
- Team admins set the model + budget

**Goal G4 — Policy engine**

- Define team-wide rules in the dashboard
- Examples:
  - "Any `test_pass` lie → auto-reprompt agent with stricter verification instructions"
  - "Any severity-3 lie → block git commit, notify eng manager"
  - "Developer X's lie rate > 10% → require manual review before PR merge"
- Policies pushed to `snitchworksd` which enforces them locally (git hooks, notifications)

### Workstream H: Monetization

**Goal H1 — Pricing tiers**

| Tier | Price | Features |
|---|---|---|
| **Free** | $0 | Up to 3 developers, 7-day retention, basic dashboard, regex verifiers only |
| **Team** | $12/dev/mo | Unlimited developers, 90-day retention, advanced analytics, model/platform comparison, policy engine, LLM verifier (100 claims/dev/day) |
| **Enterprise** | $25/dev/mo | Everything + SAML/SSO, audit logs, SOC 2, custom retention, priority support, unlimited LLM verifier |

**Goal H2 — Payment integration**

Creem for subscription billing (MoR, 3.9% + $0.40). Webhook fulfillment: team provisioning, license key generation.

**Goal H3 — License key enforcement**

`snitchworksd` validates license keys against Snitchworks API on startup. Grace period for connectivity loss (7 days offline before features degrade). Free tier doesn't require a key — just a team registration.

### Workstream I: Go-to-market

**Goal I1 — Distribution funnel**

1. Developer installs free Snitch via Homebrew → uses it, sees value
2. Shares a lie with their team (built-in share button: "Caught Claude Code in a lie: claimed all tests pass, zero tests ran. 🤥")
3. Team lead sees the viral post / hears about it → signs up for Snitchworks Free
4. Free tier shows aggregate team stats → "3 developers, 47 lies caught this week"
5. Upgrade prompt when they hit the 3-dev limit or try to access premium features

**Goal I2 — Launch assets**

- Snitchworks landing page (snitchworks.dev or similar)
- Documentation site (platform setup guides per agent)
- "State of AI Lies" public benchmark page (refreshed weekly from anonymized aggregate data)

---

## Dependency Order

```
Phase 1:
  A1 (interface) → A2 (Claude parser), A3 (Codex parser), A4 (multi-watcher)
  A4 → B1 (harness-aware tool mapping), B2 (target extraction)
  A1-A4 → C1 (multi-platform config), C2 (CLI)
  A1-A4 → D1 (platform filter), D2 (status)
  A2, A3 → E1 (fixtures), E2 (parser tests)
  All → E3 (integration test)

Phase 2:
  A1-D2 → F1 (IPC bridge), F2 (auth)
  F1-F2 → G1 (dashboard), G2 (analytics)
  G1 → G3 (LLM verifier), G4 (policy engine)
  F1-G4 → H1-H3 (monetization)
  G1-H3 → I1 (funnel), I2 (launch assets)
```

Phase 1 and Phase 2 are independently shippable. Phase 1 can (and should) ship first — broader adoption of the free Snitch engine increases the funnel into Snitchworks.
