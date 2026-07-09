# Adding a Claim Pattern

Every lie Snitch catches starts with a regex pattern in the claim registry:
[`internal/verify/prose.go`](../internal/verify/prose.go) → `claimPatterns`.
If you've watched an agent lie about something Snitch missed, you can teach it
the pattern. This is the single highest-leverage contribution to the project.

## The one rule: precision over recall

**A pattern that produces false positives will be rejected, no matter how many
real lies it would catch.** A missed lie costs nothing — the agent got away
with it once. A false alarm costs trust — the user stops believing Snitch, and
then every catch is worthless. Recall is optional. Precision is not.

## How to add a pattern

One entry in the `claimPatterns` table. That's the whole code change:

```go
{
	Type:  verifiers.ClaimTestPass,
	Regex: regexp.MustCompile(`(?i)\b(lint(?:er)? (?:passes|is clean)|no lint (?:errors|warnings))\b`),
	Examples: []string{
		"The linter passes with zero warnings.",
		"Lint is clean after the refactor.",
		"There are no lint errors left.",
	},
	Negatives: []string{
		"Does the linter pass on your branch?",
		"If the lint passes, we can merge.",
	},
},
```

Fields:

| Field | Requirement |
|-------|-------------|
| `Type` | An existing `verifiers.ClaimType`, or a new one (see below) |
| `Regex` | Case-insensitive, word-boundary anchored, as narrow as you can make it |
| `TargetIdx` | Submatch index of the target (file path etc.), 0 for none |
| `Examples` | **Minimum 3** sentences that MUST produce the claim |
| `Negatives` | **Minimum 2** sentences that MUST NOT — questions, hypotheticals, instructions, reported speech |

## Testing is mandatory — and enforced by CI

`TestClaimPatternRegistry` in
[`prose_registry_test.go`](../internal/verify/prose_registry_test.go) iterates
every pattern and **fails the build** if:

- it has fewer than 3 `Examples` or 2 `Negatives`,
- any example fails to produce its claim through the full extraction pipeline
  (including suppression heuristics), or
- any negative produces a claim.

You cannot merge an untested pattern. There is no override.

```bash
go test ./internal/verify/ -run TestClaimPatternRegistry
```

If your negative sentence still fires, don't delete the negative — improve the
regex or extend the suppression heuristics in
[`prose_suppress.go`](../internal/verify/prose_suppress.go) (questions,
conditionals, and modal phrasing are already suppressed generically for all
claim types).

## Validate against real sessions before opening the PR

Run the replay tool over your own session history and check your pattern
doesn't fire on innocent prose:

```bash
go run ./cmd/snitch replay ~/.cursor/projects --lies-only
go run ./cmd/snitch replay --harness claude ~/.claude/projects --lies-only
```

The PR template asks you to paste a summary of this run. Maintainers will run
the same replay against their own corpora — **a pattern that false-positives
on the maintainers' replay corpus is rejected**, with the offending sentence
quoted so you can tighten the regex and resubmit.

## New claim types

If no existing `ClaimType` fits:

1. Add the constant in [`verifiers/verifier.go`](../internal/verify/verifiers/verifier.go).
2. Teach a verifier to check it — usually `ContradictionVerifier` in
   [`verifiers/contradiction.go`](../internal/verify/verifiers/contradiction.go).
   A pattern whose claims no verifier can contradict is dead weight.
3. Add a contradiction test in `verifiers/contradiction_test.go` proving the
   claim can be both verified and refuted.
4. Document the type in the README's "Lie types" table.

## Review and approval

Pattern changes require maintainer approval (enforced via `CODEOWNERS` on
`internal/verify/prose*.go`). The review checklist:

- [ ] Examples/negatives present and meaningful (not trivial variations)
- [ ] Regex is anchored and narrow — no bare keywords
- [ ] No overlap with an existing pattern (extend that one instead)
- [ ] Replay run attached to the PR, no false positives
- [ ] New claim types come with a verifier + contradiction test

Once merged, your pattern's real-world precision is tracked through the data
flywheel (coming soon) — user Mark Correct / Mark Incorrect labels on its verdicts. Patterns whose community labels
show sustained false positives get reverted.
