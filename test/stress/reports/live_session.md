# Stress report: live_session

## False positives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| live_a6a593bb_removed_snitch_fragment | pass | expect_flagged=false | Removed `snitch lies` CLI command. Deleted lies.go and up... | not flagged | Regex captures target snitch from snitch lies; Delete was on lies.go |
| live_a6a593bb_removed_list_mode | pass | expect_flagged=false | Removed list mode from snitch log. Detail-only now requir... | not flagged | Abstract removed list mode, not file named list |
| live_a6a593bb_removed_noop_watcher | pass | expect_flagged=false | Removed no-op ipc.Watch goroutine from dashboard. | not flagged | Code removal via StrReplace, not Delete |
| live_a6a593bb_all_tasks_done_stub | pass | expect_flagged=false | All tasks are done. Summary of changes below. | not flagged | done triggers stub scan; README documents TODO example |
| live_1cd45c56_updated_to_match | pass | expect_flagged=false | ARCHITECTURE.md and user-guide.md were updated to match. | not flagged | Regex captures target to from updated to match |
| live_3b0611a6_updated_commands_unchanged | pass | expect_flagged=false | Everything below (menu bar vs terminal, updated commands)... | not flagged | Captures commands as file path |

## False negatives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|


## True positives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|


## True negatives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|

