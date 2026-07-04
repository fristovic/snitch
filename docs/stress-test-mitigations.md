# Lie-type stress test mitigations

**Status (implemented):** R1–R8 and session evidence auditor (priorities 1–6) landed across `internal/verify/`, `internal/transcript/`, and `internal/record/`. The full stress corpus (`go test -tags stress ./test/stress/...`) passes. See [stress-test-matrix.md](stress-test-matrix.md) for post-fix metrics.

Ranked proposals from the stress corpus (`test/stress/`) and live session audit (`70beef85` / run `a6a593bb`). Each item links to representative stress case IDs. Full tables live in [test/stress/reports/](../test/stress/reports/) and the combined matrix is in [stress-test-matrix.md](stress-test-matrix.md).

**Scoring:** Impact (gap count × severity) × confidence / effort. Higher rank = implement first.

---

## Ranked mitigations

| Rank | ID | Proposal | Impact | Effort | Linked cases |
|------|-----|----------|--------|--------|--------------|
| 1 | R2 | Path extraction: require file extension or backtick-quoted path; blocklist stopwords (`to`, `from`, `list`, `the`, `commands`) | Critical — fixes 10+ `file_deleted` FP, 7 `file_modified` FP | Medium | `fp_deleted_target_to`, `fp_deleted_target_from`, `fp_deleted_target_list`, `live_a6a593bb_removed_list_mode`, `live_1cd45c56_updated_to_match`, `live_3b0611a6_updated_commands_unchanged` |
| 2 | R3 | Tool equivalence: treat `Write` as modify; treat empty `StrReplace` as soft delete | Critical — closes `file_modified` FN + `file_deleted` FN | Medium | `fn_modified_write_only`, `fp_modified_write_only_summary`, `live_a6a593bb_removed_noop_watcher`, `fn_deleted_strreplace_clear` |
| 3 | R4 | Stub scope: only scan files **written this turn**; skip markdown code spans / tables | Critical — 5 `stub` FP including live regression | Medium | `live_a6a593bb_all_tasks_done_stub`, `fp_stub_readme_todo_example`, `fp_stub_done_summary` |
| 4 | R1 | Prose segmentation: downweight `### Summary` / recap bullets; verify claims against preceding tool prose only | High — summary-only false flags across families | Medium–High | `fp_summary_no_tools`, `fp_modified_write_only_summary`, `live_1cd45c56_updated_to_match` |
| 5 | R7 | Severity calibration: demote regex-capture FPs to L1 when `looksLikePath(target)` fails | High — reduces noise without losing TP | Low | `fp_deleted_target_unused`, `fp_created_figurative`, `fp_deleted_figurative`, `fp_modified_the_commands` |
| 6 | R8 | Shell output parsers: extend `ParseTestOutput` for pytest/vitest/cargo | High — 5 `test_pass` FN | Medium | `fn_test_pass_pytest`, `fn_test_pass_vitest`, `fn_test_pass_cargo`, `fn_test_pass_task_subagent` |
| 7 | R6 | Subagent attribution: inflate parent `ToolCalls` from `Task` results or skip parent prose when `Task` present | High — shell + consistency FN | High | `fn_test_pass_task_subagent`, `fn_self_subagent_edit`, `fn_pushed_subagent_only` |
| 8 | R5 | Cross-turn policy: document same-turn-only design; optionally ignore recap in final assistant block | Medium — clarifies scope, some FP reduction | Low (docs) / High (code) | `fp_committed_prior_turn`, `fp_pushed_changelog_mention` |
| 9 | — | Consistency regex expansion: word numbers (`five files`), negation variants (`left unchanged`, `avoided tests`) | Medium — 2+ FN per type | Low–Medium | `fn_count_five_files`, `fn_negation_left_unchanged`, `fn_negation_avoided_tests` |
| 10 | — | `reComplete` narrowing: exclude bare `done` in summaries; require implementation context | Medium — stub FP/FN balance | Low | `fp_stub_we_are_done`, `fn_stub_empty_return` |

---

## Detail by research ID

### R1 — Prose segmentation

**Hypothesis:** End-of-turn summary blocks repeat work already credited or claim files without tools in the same turn.

**Evidence:** Commit turns that only run `Shell` still say "updated README" (`fp_summary_no_tools`). Live session `live_1cd45c56_updated_to_match` flags `updated to` from recap prose.

**Proposal:**
1. Split assistant text on `### Summary`, `## Summary`, or horizontal rules.
2. Run file/git claims only on the execution segment, or require tool-call temporal proximity.
3. Tag summary-sourced claims with lower default severity.

**Cases to re-run after fix:** all `live_*` regressions, `fp_summary_no_tools`.

---

### R2 — Path extraction

**Hypothesis:** `[\w./-]+` capture group is too greedy and accepts English words as paths.

**Evidence:** `file_deleted` precision ~23% in matrix — worst type. Captures `to`, `from`, `list`, `no-op`, `snitch` (from backtick split).

**Proposal:**
1. Require `(?:\.\w{1,10})` extension OR path inside backticks with `/` or `.`.
2. Blocklist: `to`, `from`, `list`, `the`, `commands`, `need`, `legacy`, `unused`, `no-op`.
3. Fix backtick capture: prefer longest backtick-delimited span before falling back to bare token.

**Cases to re-run:** entire `file_prose` FP section, all `live_session` FP cases.

---

### R3 — Tool equivalence

**Hypothesis:** `verifyFileClaim` only checks `StrReplace` for modify and `Delete` for delete.

**Evidence:** `fn_modified_write_only` (Write used, claim missed as FN in desired state — currently flags as lie when no StrReplace). `live_a6a593bb_removed_noop_watcher` — StrReplace removal flagged as delete lie.

**Proposal:**
1. `file_modified`: accept `Write` or `StrReplace` targeting path.
2. `file_deleted`: accept `Delete` OR `StrReplace` with empty `new_string` that clears file / removes all content.
3. `file_created`: accept `Write` OR `StrReplace` where file did not exist at turn start.

**Cases to re-run:** `fn_*` file prose negatives, `tp_*` true positives for regression guard.

---

### R4 — Stub scope

**Hypothesis:** `reComplete` matches `done` anywhere; `verifyStub` scans all writes in turn including docs with example stubs.

**Evidence:** `live_a6a593bb_all_tasks_done_stub` — README contains `` `panic("TODO")` `` as documentation.

**Proposal:**
1. Narrow `reComplete`: `all tasks (are )?done` requires nearby implementation context, or exclude when only `Read`/`Glob` tools ran.
2. Restrict stub body scan to files whose content changed this turn (hash at turn start).
3. Strip markdown fenced code blocks before `IsStubBody`.

**Cases to re-run:** `fp_stub_*`, `fn_stub_*`, live stub regression.

---

### R5 — Cross-turn policy

**Hypothesis:** Same-turn-only verification is intentional but confuses users when prose references prior commits.

**Evidence:** `fp_committed_prior_turn` — prose in turn N credits commit from turn N-1 (should not flag per design).

**Proposal:** Document in ARCHITECTURE.md. Optional: add `claim_source_turn` metadata for dashboard. Do not expand scope without explicit product decision.

---

### R6 — Subagent attribution

**Hypothesis:** Parent turn prose claims work done by `Task` subagent without local `ToolCalls`.

**Evidence:** `fn_test_pass_task_subagent`, `fn_self_subagent_edit`, `fn_pushed_subagent_only`.

**Proposal:**
1. Parse `Task` tool results for nested shell/file operations (if present in transcript).
2. Or: suppress action claims in parent turn when `Task` tool invoked and no local evidence.

**Trade-off:** High effort; depends on transcript shape stability.

---

### R7 — Severity calibration

**Hypothesis:** Many path FPs are structurally invalid paths and should not reach FAIL verdict.

**Proposal:** Add `looksLikePath(s string) bool` — requires `.ext`, `/`, or known project-relative pattern. Claims failing check → severity 1 (INFO) or suppressed.

**Cases:** figurative removals, `fp_created_figurative`, stopword captures.

---

### R8 — Evaluation metrics

**Hypothesis:** Need frozen corpus and per-type targets before tuning regexes.

**Proposal:**
1. Freeze `test/stress/` as v1 corpus (141 cases).
2. Targets post-mitigation: precision ≥ 85%, recall ≥ 80% per type.
3. CI `stress` job (`continue-on-error: true`) tracks gap count over time; flip to required when top-3 mitigations land.

---

## Recommended implementation PR (Wave 3)

**Completed in this pass:**

1. **R2 + R7** — `LooksLikePath`, path stopwords, backtick extraction, trailing-period normalization
2. **R3** — `Write`/`StrReplace` equivalence; soft-delete via empty `StrReplace`; disk path from matched tool
3. **R4** — stub scan limited to written tool bodies; markdown table / doc-path guards
4. **R1** — `segmentProse` execution vs recap; recap severity caps; lookback-aware file-claim suppression
5. **R5** — 3-turn session lookback for git/shell/file/stub claims; consistency stays same-turn
6. **R6** — subagent time-window merge into `EffectiveToolCalls`
7. **R8** — framework-specific `ParseTestOutput` chain (go/pytest/vitest/cargo/npm) + relaxed terminal fallback
8. **Confidence** — `Claim.Confidence` scoring + `AdjustSeverity` in engine; `claims.confidence` column

**Remaining (future):**

- **Deep stub analysis** — skeleton types, pre-existing on-disk stubs without writes this turn
- **Terminal-only stress harness** — richer `terminals/*.txt` fixtures in stress corpus

---

## Out of scope (this pass)

- LLM-based verification
- Cross-session lookback (beyond 3 turns in one session)
- Subagent `tool_use_id` mapping (not available in Cursor transcripts today)
- Non-Cursor harnesses
