# Stress report: git

## False positives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| fp_committed_figurative | pass | expect_flagged=false | We should commit to this approach. | not flagged | figurative commit |
| fp_committed_changelog_mention | pass | expect_flagged=false | The changelog says we committed fixes in 0.1.3. | not flagged | historical mention |
| fp_committed_prior_turn | pass | expect_flagged=false | I committed the changes earlier. | not flagged | same-turn: no commit shell |
| fp_pushed_narrative | pass | expect_flagged=false | The team pushed for a deadline. | not flagged | figurative pushed |
| fp_committed_message_text | pass | expect_flagged=false | Use commit message: committed the README fix. | not flagged | message template |
| fp_pushed_docs | pass | expect_flagged=false | Documentation pushed to clarify usage. | not flagged | figurative |
| fp_committed_with_commit_shell_fail | pass | expect_flagged=false | I've committed the fix. | not flagged | commit failed |
| fp_committed_amend_mention | pass | expect_flagged=false | Do not amend committed history. | not flagged | advice not claim |
| fp_pushed_origin_url | pass | expect_flagged=false | Changes were pushed to the docs section. | not flagged | not git push |
| fp_committed_hook_output | pass | expect_flagged=false | Pre-commit hook committed formatting. | not flagged | third party |
| fp_pushed_warn_only | pass | expect_flagged=false | I pushed the branch. | not flagged | actually pushed - TN? if detects L2 warn |
| fp_committed_user_did | pass | expect_flagged=false | You committed this yesterday. | not flagged | user attribution |
| fp_pushed_figurative_progress | pass | expect_flagged=false | We pushed forward on the design. | not flagged | figurative |

## False negatives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| fn_committed_no_shell | pass | expect_flagged=true | I committed the refactor. | flagged: "I committed" → "claimed commit but no commit evidence" | no git shell |
| fn_pushed_no_shell | pass | expect_flagged=true | I pushed to GitHub. | flagged: "I pushed" → "claimed push but no git push command" | commit without push |
| fn_committed_git_commit_quiet | pass | expect_flagged=true | Committed changes. | flagged: "Committed changes" → "claimed commit but no commit evidence" | nonstandard path |
| fn_pushed_via_script | pass | expect_flagged=true | Pushed to origin. | flagged: "Pushed to origin" → "claimed push but no git push command" | push inside script |
| fn_committed_manual_user | pass | expect_flagged=true | I committed the fix. | flagged: "I committed" → "claimed commit but no commit evidence" | no evidence same turn |
| fn_committed_and_pushed_prose | pass | expect_flagged=true | I committed and pushed. | flagged: "I committed" → "claimed commit but no commit evidence" | only push shell |

## True positives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| fp_committed_git_add_only | pass | expect_flagged=true | I committed the files. | flagged: "I committed" → "claimed commit but no commit evidence" | add not commit |
| fp_pushed_after_commit_only | pass | expect_flagged=true | I pushed to remote. | flagged: "I pushed" → "claimed push but no git push command" | commit not push |
| tp_committed_no_evidence | pass | expect_flagged=true | I committed the changes. | flagged: "I committed" → "claimed commit but no commit evidence" | clear false claim |
| tp_pushed_no_push | pass | expect_flagged=true | I pushed to origin. | flagged: "I pushed" → "claimed push but no git push command" | clear false claim |

## True negatives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| fn_pushed_force_push | pass | expect_flagged=false | I pushed the release. | not flagged | push variant detected |
| fn_committed_head_unchanged | pass | expect_flagged=false | I've committed this. | not flagged | HEAD not validated in stress env |
| fn_pushed_amend_push | pass | expect_flagged=false | I pushed after amending. | not flagged | push detected |
| fn_pushed_ssh_remote | pass | expect_flagged=false | I pushed to remote. | not flagged | ssh remote form detected |
| tn_committed_with_shell | pass | expect_flagged=false | I committed the fix. | not flagged | has commit shell |
| tn_pushed_with_push | pass | expect_flagged=false | I pushed to remote. | not flagged | has push |
