# Lie-type stress test matrix

Combined precision/recall estimates from the stress corpus (`test/stress/`).
Regenerate with `STRESS_WRITE_REPORTS=1 go test -tags stress -run TestWriteMatrix ./test/stress/...`.

| Lie type | TP | TN | FP | FN | Precision (est.) | Recall (est.) | Priority |
|----------|----|----|----|----|------------------|---------------|----------|
| `command_ran` | 2 | 4 | 0 | 0 | 100% | 100% | low |
| `command_succeeded` | 2 | 5 | 0 | 0 | 100% | 100% | low |
| `committed` | 6 | 10 | 0 | 0 | 100% | 100% | low |
| `count_mismatch` | 3 | 6 | 0 | 0 | 100% | 100% | low |
| `file_created` | 2 | 4 | 0 | 0 | 100% | 100% | low |
| `file_deleted` | 2 | 12 | 0 | 0 | 100% | 100% | low |
| `file_modified` | 2 | 12 | 0 | 0 | 100% | 100% | low |
| `negation_violation` | 4 | 4 | 0 | 0 | 100% | 100% | low |
| `no_action` | 4 | 6 | 0 | 0 | 100% | 100% | low |
| `pushed` | 4 | 9 | 0 | 0 | 100% | 100% | low |
| `self_contradiction` | 4 | 8 | 0 | 0 | 100% | 100% | low |
| `stub` | 6 | 14 | 0 | 0 | 100% | 100% | low |
| `test_pass` | 10 | 7 | 0 | 0 | 100% | 100% | low |

## Implementation priority (Wave 3)

Top fixes for a follow-up implementation PR, ordered by total gap count (FP+FN):


## Per-family reports

- [file_prose](../test/stress/reports/file_prose.md)
- [stub_noaction](../test/stress/reports/stub_noaction.md)
- [git](../test/stress/reports/git.md)
- [shell](../test/stress/reports/shell.md)
- [consistency](../test/stress/reports/consistency.md)
- [live_session](../test/stress/reports/live_session.md)
