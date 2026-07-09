## What

<!-- One paragraph: what this PR does and why. -->

## Checklist (all PRs)

- [ ] `go build ./... && go vet ./... && go test -race ./...` pass
- [ ] `gofmt -l .` is clean

## If this PR adds or changes a claim pattern

See [docs/extending-patterns.md](../docs/extending-patterns.md). Precision over recall — false positives are rejected.

- [ ] Pattern added to the `claimPatterns` registry (not a bespoke extraction loop)
- [ ] At least 3 `Examples` and 2 `Negatives`, and `TestClaimPatternRegistry` passes
- [ ] No overlap with an existing pattern (extended it instead, if close)
- [ ] Replay run over real sessions attached below (`snitch replay <dir> --false-claims-only`) with zero false positives
- [ ] New claim types: verifier support + contradiction test + README "Claim types" row

<details><summary>Replay output</summary>

```
(paste redacted `snitch replay` summary here)
```

</details>

## If this PR adds a harness

See [docs/extending-harnesses.md](../docs/extending-harnesses.md). Format must be verified against real session files, never guessed.

- [ ] Parser + path resolver + descriptor + config entry (all five touchpoints)
- [ ] Realistic fixture in `test/fixtures/sample_transcripts/` (secrets removed)
- [ ] Parser test, `multiharness_test.go` entry, and `watcher_live_test.go` case
- [ ] `snitch replay --harness <name>` run against a real on-device session attached below
- [ ] Disabled by default in `defaults.go`

<details><summary>Replay output</summary>

```
(paste redacted `snitch replay` summary here)
```

</details>
