# Snitch → Cursor Extension: Plan

## TL;DR

Turn Snitch into a Cursor plugin that intercepts the agent loop via hooks — blocking lies before execution, auditing all tool calls, and injecting verification findings post-turn. **Ship a thin wrapper now to claim the category. Go full-featured when Cursor fixes the `additional_context` bug.**

---

## Why This Makes Sense

### Cursor is no longer an IDE

Cursor 3 (April 2026) was built from scratch as an agent management console. The traditional editor is a fallback surface. From their own blog:

> *"When we started building Cursor, we forked VS Code instead of building an extension so we could shape our own surface. With Cursor 3, we took that a step further by building this new interface from scratch, centered around agents."*

The New Stack called it *"Cursor's $2 billion bet: The IDE is now a fallback, not the product."*

**This makes Snitch *more* valuable, not less.** When you're managing fleets of autonomous agents instead of writing code, knowing which ones lie to you becomes essential infrastructure — not a nice-to-have.

### The native extension surface is hooks, not VS Code extensions

Cursor's plugin system (launched Feb 2026) bundles hooks, skills, MCP servers, subagents, and rules. Hooks are the integration point for observing and controlling the agent loop — exactly what Snitch does, but natively rather than by scraping transcript files.

A VS Code extension that shows a "lies" sidebar panel would be building on the surface Cursor is actively demoting. A hooks-based plugin is building on the surface Cursor is betting the company on.

### First-mover advantage

`cursor.directory` (83.9K+ developers) and the official Cursor Marketplace have **zero** agent honesty/verification plugins. The closest are:

- **Snyk Evo Agent Guard** — prompt injection / dangerous tool calls (security, not honesty)
- **Semgrep** — code vulnerability scanning
- **Corridor** — code quality feedback

No one is doing claim verification. Snitch would define the category.

---

## Current Snitch vs. Plugin Snitch

| Capability | Current (daemon) | Plugin (hooks) |
|---|---|---|
| Detect lies in agent prose | ✅ Regex extraction from transcripts | ✅ Same regex on `afterAgentResponse` payload |
| Verify claims against tool calls | ✅ Parses JSONL transcripts | ✅ Structured JSON from `postToolUse` — cleaner |
| Verify claims against filesystem | ✅ `os.Stat` etc. | ✅ Hook scripts can run shell commands |
| Verify claims against git | ✅ `git diff startHEAD..HEAD` | ✅ Same, called from hook scripts |
| **Block lies before execution** | ❌ Post-hoc only | ✅ `preToolUse` with `failClosed: true` |
| **Real-time agent feedback** | ❌ | ⚠️ Blocked by `additional_context` bug |
| Cross-session analytics | ✅ SQLite | ✅ Hook scripts maintain SQLite |
| CLI dashboard | ✅ `snitch lies`, TUI | ❌ Would need separate build |
| Distribution | Homebrew tap | Cursor Marketplace + cursor.directory |
| Cloud agent support | ❌ | ⚠️ Partial (limited hook coverage) |

**The plugin adds blocking. It loses the integrated dashboard.** The verification engine can be shared.

---

## Architecture

```
snitch-cursor/                          # New repo or subdirectory in monorepo
├── .cursor-plugin/
│   └── plugin.json                     # Manifest
├── hooks/
│   ├── hooks.json                      # Hook registration
│   ├── pre-tool-use.ts                 # Block shell/write/delete if contradicting
│   ├── post-tool-use.ts                # Audit: log tool_name, input, output, duration
│   ├── after-agent-response.ts         # Extract prose claims (regex)
│   └── stop.ts                         # Run verification, inject followup_message
├── bin/
│   └── snitch-engine                   # Go binary (shared with daemon)
│       ├── verify/                     # Claim extraction + contradiction + consistency
│       └── record/                     # SQLite storage
├── skills/
│   └── snitch.md                       # Teaches agents to self-verify claims
└── README.md
```

### Hook data flow

```
Agent types a response
        │
        ▼
afterAgentResponse  ─── Extract claims from prose ───► SQLite (claims table)
        │
        ▼
Agent calls tool (Shell, Write, etc.)
        │
        ▼
preToolUse  ─── Check: does this contradict a prior claim? ─── BLOCK if yes
        │
        ▼
postToolUse  ─── Log tool_name + input + output ───► SQLite (runs table)
        │
        ▼
Turn ends
        │
        ▼
stop  ─── Run full verification:
          • Contradiction: prose claims vs actual tool calls/output
          • Consistency: internal contradictions, count mismatches
          • Tool: file/shell/subagent verifiers
          ─── Inject findings via followup_message
```

### Why Go binary, not pure TypeScript

The claim extraction and verification logic in `internal/verify/` is ~1,500 lines of Go with regex patterns, consistency checkers, and contradiction verifiers. Rewriting in TypeScript is wasted effort. The binary is called via `Bun.spawn` from hook scripts, stdin/stdout JSON — same IPC model the hooks already use.

---

## Implementation Phases

### Phase 1: Thin wrapper (now)

Ship the minimum to claim the category on `cursor.directory`.

**Deliverables:**
- `.cursor-plugin/plugin.json` with name, description, author
- `hooks/hooks.json` wiring `preToolUse`, `postToolUse`, `stop`
- `hooks/stop.ts` — calls `snitch-engine verify` on turn end, injects `followup_message`
- `bin/snitch-engine` — the existing Go binary, stripped to just `verify` + `record`
- `README.md` with install instructions

**What it does:** Post-turn verification only. Agent finishes a turn → Snitch checks for lies → injects findings into the next turn. Same capability as the daemon, but via hooks instead of file watching.

**Does not include:** Real-time blocking (`preToolUse` requires a running daemon/SQLite, hook scripts are stateless between invocations — need to solve state persistence first). Dashboard (separate concern).

### Phase 2: Real-time blocking (when `additional_context` is fixed)

**Blocked on:** Cursor fixing the `additional_context` injection bug (acknowledged March 2026, no ETA).

**Adds:**
- Stateful SQLite connection shared across hook invocations (or use a small sidecar daemon)
- `preToolUse` matcher for `Shell|Write|Delete` — checks pending claims before allowing execution
- `additional_context` injection on `postToolUse` — "I noticed you claimed X but the file actually contains Y"

### Phase 3: Full plugin (when marketplace review speeds up)

**Adds:**
- `skills/snitch.md` — teaches agents to self-verify before making claims
- `afterAgentResponse` claim extraction inline (not just in `stop`)
- Web dashboard (Next.js, reads the same SQLite)
- Official Cursor Marketplace submission
- Cloud agent support (to the extent hooks allow)

---

## The Critical Bug: `additional_context`

**Status:** Confirmed by Cursor staff, unfixed since at least March 2026.

Three separate forum threads document it:
- [#155689](https://forum.cursor.com/t/155689) (Mar 23, 569 views, 18 likes) — Original report with smoke test proving context isn't surfaced
- [#156157](https://forum.cursor.com/t/156157) (Mar 29) — Duplicate; staff confirmed and closed
- [#158168](https://forum.cursor.com/t/158168) (Apr 16) — Additional confirmation

Staff response (Mar 29, 2026):

> *"This is a confirmed bug. additional_context in postToolUse hooks is accepted and logged correctly, but it isn't actually delivered to the model context. For non-MCP tools, the hook response is discarded fire-and-forget. For MCP, only updated_mcp_tool_output is used. The only hook where additional_context works end-to-end is sessionStart."*

**Impact on Snitch:** Without `additional_context` working on `postToolUse`, you cannot tell the agent mid-turn that it made a false claim. The feedback loop is post-turn only (`stop` hook → `followup_message`). This means the plugin's "block before execution" capability can prevent damage, but the gentler "hey, you said X but Y is true, try again" correction path doesn't exist yet.

**Workaround in the meantime:** Use the `stop` hook's `followup_message` for post-turn correction. Use `preToolUse` with `permission: "deny"` for hard blocking. Accept that mid-turn feedback is unavailable.

---

## Marketplace Distribution

Two channels, very different timelines:

| | cursor.directory | Official Cursor Marketplace |
|---|---|---|
| Listing | Self-serve, immediate | Submit via `marketplace-publishing@cursor.com` |
| Review time | None (community) | Weeks to months (ZoomInfo: 1+ month, no response) |
| Audience | 83.9K+ developers | Full Cursor user base |
| Verification | Community ratings | Official Cursor team review |
| Plugin format | Same manifest | Same manifest |

**Strategy:** List on `cursor.directory` immediately (Phase 1). Submit to official marketplace in parallel, but don't wait for it.

---

## Is It Worth It? — The Honest Answer

**Yes, but phased and patient.** The strategic argument is strong: Cursor is betting the company on agents, agents lie, and no one else is building verification. The hooks API is the right integration point — it's where Cursor is investing, not legacy VS Code extensions.

**But the ecosystem is immature.** The one hook feature that would make the plugin *better* than the daemon is broken with no fix timeline. Marketplace review is glacial for indies. Cloud agent hook coverage is incomplete.

**The right move:** Ship the thin wrapper now (Phase 1) to claim the category and learn how hooks behave in practice. Keep the daemon as the primary product. When `additional_context` is fixed, that's the signal to invest heavily in Phase 2.

**What NOT to do:** Rewrite Snitch as a VS Code extension. That's building on the surface Cursor is demoting. The IDE panel showing lies would be a tombstone before it shipped.

---

## References

- [Cursor Hooks Docs](https://cursor.com/docs/hooks) — Full API: 19 hook types, stdin/stdout JSON protocol, matchers, `failClosed`
- [Cursor Plugins Reference](https://cursor.com/docs/reference/plugins) — Manifest format, component discovery, marketplace
- [Cursor Plugin GitHub](https://github.com/cursor/plugins) — Official plugin spec + examples (Thermos, pstack, continual-learning)
- [Cursor 3 Blog Post](https://cursor.com/blog/cursor-3) — Agent-first interface, IDE as fallback
- [Cursor Marketplace Blog](https://cursor.com/blog/marketplace) — Plugin system launch (Feb 17, 2026)
- [The New Stack: Cursor 3 Demotes IDE](https://thenewstack.io/cursor-3-demotes-ide/) — External analysis
- [Forum: additional_context bug #156157](https://forum.cursor.com/t/156157) — Staff confirmed, no ETA
- [Forum: ZoomInfo submission 1+ month no response](https://forum.cursor.com/t/164579) — Marketplace review bottleneck
- [cursor.directory](https://cursor.directory/) — Community plugin listing (83.9K+ devs)
