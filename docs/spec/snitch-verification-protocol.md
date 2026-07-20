# Snitch Verification Protocol (SVP)

**Version:** 1.0.0-draft
**License:** Apache 2.0
**Status:** Proposed Standard

## Abstract

The Snitch Verification Protocol (SVP) defines an open, platform-agnostic contract for verifying AI coding agent claims against ground truth. Any agent platform, hook system, or IDE can implement SVP to answer the question: **"Did the agent actually do what it claimed?"**

SVP is implemented by the Snitch reference implementation (`github.com/fristovic/snitch`) but is designed to be implementable by anyone — Anthropic, Cursor, OpenAI, enterprise security tools, or independent verifiers.

## Design Principles

1. **Platform-agnostic.** SVP claims and verdicts carry no platform-specific fields. Platform adapters translate their native formats to SVP.
2. **Deterministic where possible.** Regex extraction + deterministic verifiers produce high-precision, auditable results. Classifiers augment but don't replace deterministic verification.
3. **Evidence-backed.** Every verdict includes structured evidence that can be independently verified.
4. **Severity-calibrated.** Not all false claims are equal. Severity levels let consumers decide their own thresholds.
5. **Transport-agnostic.** SVP is a JSON contract over stdin/stdout for hooks, but the same schema works over HTTP, Unix sockets, or file-based batch processing.

## Claim Schema

A claim is a single assertion extracted from an agent's natural language prose.

```json
{
  "claim_id": "550e8400-e29b-41d4-a716-446655440000",
  "harness": "claude",
  "session_id": "abc123",
  "turn_number": 5,
  "claim_type": "test_pass",
  "sentence": "All tests pass.",
  "context": "I ran the test suite and all tests pass.",
  "tool_calls": [
    {
      "tool_name": "Bash",
      "tool_input": {
        "command": "npm test"
      }
    }
  ],
  "tool_outputs": [
    {
      "tool_name": "Bash",
      "exit_code": 1,
      "stdout": "14 tests run, 3 failed",
      "stderr": "FAIL: TestFoo"
    }
  ],
  "filesystem_state": {
    "start_head": "abc123def",
    "end_head": "abc123def",
    "file_manifest": ["src/main.go", "src/lib.go"]
  },
  "timestamp": "2026-07-20T12:00:00Z"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `claim_id` | UUID string | Yes | Unique identifier for this claim |
| `harness` | string | Yes | Platform identifier: `cursor`, `claude`, `codex`, `pi`, `opencode` |
| `session_id` | string | Yes | Agent session identifier |
| `turn_number` | integer | No | Turn number within the session (for lookback verification) |
| `claim_type` | string | Yes | One of the [Claim Types](#claim-types) |
| `sentence` | string | Yes | The specific sentence containing the claim |
| `context` | string | No | Surrounding paragraph for context |
| `tool_calls` | array | Yes | Tool calls in this turn (for contradiction detection) |
| `tool_outputs` | array | Yes | Tool outputs in this turn (for evidence) |
| `filesystem_state` | object | No | Git HEAD and file manifest at turn start/end |
| `timestamp` | ISO 8601 | Yes | When the claim was emitted |

### Claim Types

| Type | Example Prose | What Snitch Checks |
|------|--------------|-------------------|
| `test_pass` | "All tests pass" | Shell exit code ≠ 0, test failures in output |
| `command_ran` | "I ran the command" | Shell tool call exists in the turn |
| `command_succeeded` | "Command ran successfully" | Shell exit code ≠ 0 |
| `committed` | "I committed the changes" | Git HEAD delta between turn start and end |
| `pushed` | "I pushed to origin" | Git push shell call in the turn |
| `file_created` | "Created handler.go" | Write tool call + file exists on disk |
| `file_modified` | "Updated handler.go" | Write/Edit tool call + file modified |
| `file_deleted` | "Deleted handler.go" | Delete tool call + file absent from disk |
| `stub` | "Fully implemented" | Written file contains placeholder (TODO, panic, pass) |
| `no_action` | Any action claim | Zero mutating tool calls in the turn |
| `self_contradiction` | "Won't modify X" while tool call edits X | Internal consistency check |
| `count_mismatch` | "Updated all 5 files" | Tool call count ≠ claimed count |
| `negation_violation` | "Did not touch tests" | Test file edited in the turn |

## Verdict Schema

A verdict is the result of verifying a claim.

```json
{
  "verdict": "fail",
  "severity": 3,
  "claim_type": "test_pass",
  "contradiction": {
    "type": "shell_exit_code",
    "expected": "exit code 0",
    "actual": "exit code 1"
  },
  "evidence": {
    "source": "tool_output",
    "detail": "npm test exited with code 1. 3 of 14 tests failed."
  },
  "classifier_verdict": "genuine",
  "classifier_confidence": 0.97,
  "verification_method": "contradiction",
  "timestamp": "2026-07-20T12:00:01Z"
}
```

### Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `verdict` | string | Yes | `pass` (claim verified), `fail` (claim contradicted), `warn` (suspicious), `ambiguous` (could not determine) |
| `severity` | integer | Yes | 0-4 (see [Severity Taxonomy](#severity-taxonomy)) |
| `claim_type` | string | Yes | Echo of the claim type |
| `contradiction` | object | No | Present on `fail` verdicts. Describes what was expected vs. actual |
| `evidence` | object | No | Structured evidence supporting the verdict |
| `classifier_verdict` | string | No | If classifier loaded: `genuine` or `false_positive` |
| `classifier_confidence` | float | No | 0.0-1.0 confidence score (if classifier loaded) |
| `verification_method` | string | Yes | Which verifier produced this verdict: `contradiction`, `consistency`, `file`, `shell`, `subagent` |
| `timestamp` | ISO 8601 | Yes | When verification completed |

### Severity Taxonomy

| Level | Name | Definition | Recommended Action |
|-------|------|------------|-------------------|
| 0 | PASS | Claim verified — no contradiction found | None |
| 1 | INFO | No contradiction but could not fully verify | Log for later review |
| 2 | WARN | Suspicious but not definitively false | Alert developer, allow action |
| 3 | FAIL | High-confidence false claim | Block action, alert developer |
| 4 | CRITICAL | Dangerous false claim (e.g., "committed" but no git change) | Block action, alert team, require manual review |

## Verification Methods

### 1. Contradiction Detection
Compares prose claims against tool output, filesystem state, and git history.

**Evidence sources:**
- Shell exit codes (test_pass, command_succeeded)
- File existence checks (file_created, file_modified, file_deleted)
- Git HEAD delta (committed, pushed)
- Tool call presence (command_ran, no_action)

### 2. Consistency Checking
Detects internal contradictions within a single turn without external evidence.

**Checks:**
- `self_contradiction`: claims "won't modify X" but tool call edits X
- `count_mismatch`: claims "updated 5 files" but tool call count ≠ 5
- `negation_violation`: claims "did not touch tests" but test file edited

### 3. File Verification
Validates file operations actually modified the filesystem.

**Checks:**
- File exists after Write tool call
- Edit reflected in file contents
- Delete removed the file from disk
- Written file is not a stub/placeholder (stub detection)

### 4. Shell Verification
Validates shell command output against claims.

**Checks:**
- Exit code matches claim (0 for success, non-zero for failure)
- Output contains expected strings
- Command actually ran (shell tool call exists)

### 5. Subagent Verification
Merges subagent transcript tool calls by time window for cross-turn evidence.

## Hook Integration Contract

All platform hook bindings MUST conform to this contract:

### Input (stdin)
SVP Claim JSON (see [Claim Schema](#claim-schema)).

### Output (stdout)
SVP Verdict JSON (see [Verdict Schema](#verdict-schema)). Optional — the exit code is the primary signal.

### Exit Codes

| Code | Meaning | Consumer Action |
|------|---------|----------------|
| 0 | PASS — no false claims detected at or above threshold | Allow action to proceed |
| 2 | BLOCK — one or more false claims at severity ≥ threshold | Block action, show stderr to agent |
| 1 | ERROR — non-blocking error (hook misconfigured, timeout, etc.) | Allow action, log warning |

### Stderr
Human-readable explanation. On exit code 2, stderr is shown to the agent.

### Threshold
Configurable via `~/.snitch/config.yaml`:
```yaml
hooks:
  block_threshold: 3  # Minimum severity to trigger exit code 2
```

## Platform Adapter Responsibilities

Each platform adapter MUST:
1. Translate the platform's native hook JSON into an SVP Claim
2. Map platform-specific tool names to SVP canonical tool names (see [Canonical Tool Names](#canonical-tool-names))
3. Call the SVP verifier (reference implementation or compatible)
4. Map the SVP Verdict exit code back to the platform's blocking mechanism

Each platform adapter MAY:
1. Inject the verdict as context into the agent (PostToolUse/Stop)
2. Customize severity thresholds per platform
3. Add platform-specific metadata to the claim

## Canonical Tool Names

| Canonical | Cursor | Claude Code | Codex CLI | Pi | OpenCode |
|-----------|--------|-------------|-----------|----|----|
| `Shell` | `Shell` | `Bash` | `shell_command` | `bash` | `bash` |
| `Write` | `Write` | `Write` | `write_to_file` | `write` | `write` |
| `Read` | `Read` | `Read` | `read_file` | `read` | `read` |
| `StrReplace` | `StrReplace` | `Edit` | `apply_patch` | `edit` | `edit` |
| `Delete` | `Delete` | — | — | — | — |
| `Glob` | `Glob` | `Glob` | `search` | `grep` / `glob` | `grep` / `glob` |
| `Task` | `Task` | `Agent` | `task` | `task` | `task` |
| `WebFetch` | — | `WebSearch` / `WebFetch` | — | — | — |

## Session Lookback

For recap/summary prose (tagged segments like `### Summary`, `## Summary`, horizontal rules), SVP verifiers MAY credit evidence from up to **3 prior turns** in the same session for:
- `committed` / `pushed` — git evidence in prior turns
- `test_pass` / `command_*` — shell evidence in prior turns
- `file_created` / `file_modified` / `file_deleted` — file tools + manifests in prior turns
- `stub` — placeholder bodies in files written in current or prior turns

Same-turn-only types (not eligible for lookback): `no_action`, `self_contradiction`, `count_mismatch`, `negation_violation`.

## Classifier Integration

An optional false-positive classifier can sit between claim extraction and deterministic verification:

```
prose → regex extraction → classifier (if loaded) → deterministic verifiers
                                    │
                                    ▼
                          regex hits classified as:
                          genuine → pass to verifiers
                          false_positive → discard
```

The classifier is NOT part of the SVP spec. It is an implementation detail of the reference implementation. The SVP contract is: claims arrive at the verifier, verdicts are returned. Whether the claim was pre-filtered by a classifier is opaque to the protocol.

## Security Considerations

1. **Claims may contain sensitive data.** Platform adapters should strip secrets (API keys, tokens) before transmitting claims.
2. **Stderr may be shown to the agent.** On exit code 2, stderr is injected into the agent's context. Avoid leaking sensitive information.
3. **Hook timeout is critical.** A slow verifier blocks the agent. Recommended timeout: 10 seconds.
4. **Fail-open vs fail-closed.** Configurable. Default: fail-open (timeout or error → allow action). Production teams may prefer fail-closed for critical hooks.

## Conformance

An implementation conforms to SVP if it:
1. Accepts SVP Claim JSON on stdin
2. Returns exit code 0 (pass) or 2 (block)
3. Outputs SVP Verdict JSON on stdout (optional but recommended)
4. Maps platform tool names to canonical names per the [Canonical Tool Names](#canonical-tool-names) table
5. Produces deterministic, auditable verdicts with structured evidence

The Snitch reference implementation (`snitch verify`) is the canonical implementation. Other implementations are valid if they conform to this specification.

## Versioning

SVP follows semantic versioning. The version is embedded in the claim/verdict JSON as an optional `svp_version` field. Breaking changes to the claim or verdict schema require a major version bump.

## References

- OWASP Top 10 for Agentic Applications 2026 — [ASI09: Human-Agent Trust Exploitation](https://genai.owasp.org/resource/owasp-top-10-for-agentic-applications-for-2026/)
- NIST AI Agent Standards Initiative — [February 2026](https://www.nist.gov/artificial-intelligence/ai-agent-standards-initiative)
- Cloud Security Alliance Agentic Trust Framework — [February 2026](https://cloudsecurityalliance.org/blog/2026/02/02/the-agentic-trust-framework-zero-trust-governance-for-ai-agents)
- Claude Code Hooks Reference — [code.claude.com/docs/en/hooks](https://code.claude.com/docs/en/hooks)
- Cursor Hooks — [cursor.com/docs/hooks](https://cursor.com/docs/hooks)
- Codex CLI Hooks — [GitHub: openai/codex#14882](https://github.com/openai/codex/issues/14882)
- Pi Extension API — [deepwiki.com/earendil-works/pi](https://deepwiki.com/earendil-works/pi/6.1-extension-api-and-lifecycle-events)
- OpenCode Plugin SDK — [opencode.ai/docs/plugins](https://opencode.ai/docs/plugins/)
