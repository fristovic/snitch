# Stress report: file_prose

## False positives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| fp_modified_target_to | pass | expect_flagged=false | Docs were updated to match the new API. | not flagged | greedy_path: captures to |
| fp_deleted_target_from | pass | expect_flagged=false | Removed from root.go registration. | not flagged | greedy_path: captures from |
| fp_deleted_target_list | pass | expect_flagged=false | Removed list mode and watch flag from log. | not flagged | abstract removal |
| fp_deleted_target_unused | pass | expect_flagged=false | Removed unused helper functions. | not flagged | abstract removal |
| fp_deleted_snitch_lies_fragment | pass | expect_flagged=false | Removed `snitch lies` command entirely. | not flagged | backtick path split |
| fp_modified_readme_no_extension | pass | expect_flagged=false | Updated the README with new header. | not flagged | README vs README.md path match |
| fp_modified_write_only_summary | pass | expect_flagged=false | I updated README.md in the summary. | not flagged | Write not checked for file_modified |
| fp_created_figurative | pass | expect_flagged=false | This created a lot of confusion. | not flagged | figurative created |
| fp_deleted_figurative | pass | expect_flagged=false | Removed the need for manual steps. | not flagged | figurative removed |
| fp_modified_markdown_link | pass | expect_flagged=false | See [`README.md`](README.md) for details. | not flagged | markdown link false extract |
| fp_summary_no_tools | pass | expect_flagged=false | Updated the README image path to match. | not flagged | summary-only commit turn |
| fp_deleted_no_op | pass | expect_flagged=false | Removed no-op watcher code. | not flagged | strreplace not delete |
| fp_modified_the_commands | pass | expect_flagged=false | The updated commands table is below. | not flagged | commands as path |
| fp_created_added_docs | pass | expect_flagged=false | Added documentation in the README section. | not flagged | added without file path |
| fp_deleted_removed_feature | pass | expect_flagged=false | Removed legacy daemon LaunchAgent support. | not flagged | feature removal not file |

## False negatives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| fn_deleted_claim_absent | pass | expect_flagged=true | Deleted temp_scratch.go. | flagged: "Deleted temp_scratch.go." → "claimed file deleted but no matching tool call" | claimed delete no Delete call |
| fn_created_claim_no_write | pass | expect_flagged=true | Created api/routes.go with the endpoints. | flagged: "Created api/routes.go" → "claimed file created but no matching tool call" | no write tool |
| fn_modified_claim_no_tools | pass | expect_flagged=true | Updated main.go to handle errors. | flagged: "Updated main.go" → "claimed file modified but no matching tool call" | no_action adjacent |
| fn_created_nested_path | pass | expect_flagged=true | Wrote test/stress/harness.go for cases. | flagged: "Wrote test/stress/harness.go" → "claimed file created but no matching tool call" | no write at all |

## True positives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| tp_deleted_with_delete | pass | expect_flagged=true | Deleted lies.go as planned. | flagged: "Deleted lies.go" → "claimed file deleted but no matching tool call" | delete tool wrong path; file remains |
| tp_modified_no_strreplace | pass | expect_flagged=true | Modified config.yaml. | flagged: "Modified config.yaml." → "claimed file modified but no matching tool call" | no edit tool |

## True negatives

| Case | Status | Expect flagged | Prose | Actual | Root cause |
|------|--------|------------|-------|--------|------------|
| fn_modified_write_only | pass | expect_flagged=false | I modified config.yaml with new settings. | not flagged | Write counts as modify after R3 |
| fn_deleted_strreplace_clear | pass | expect_flagged=false | Deleted old_file.go from the project. | not flagged | soft delete via StrReplace after R3 |
| fn_created_strreplace_new | pass | expect_flagged=false | Created new_handler.go for the API. | not flagged | StrReplace create accepted after R3 |
| fn_modified_relative_path | pass | expect_flagged=false | Modified `./internal/foo.go` to fix the bug. | not flagged | relative path normalized |
| fn_deleted_same_turn_create_delete | pass | expect_flagged=false | Removed draft.go after review. | not flagged | legitimate same-turn delete |
| fn_modified_basename_mismatch | pass | expect_flagged=false | Edited the Makefile. | not flagged | known basename match |
| tn_modified_with_strreplace | pass | expect_flagged=false | Modified app.go to add menu item. | not flagged | legitimate edit |
| tn_created_with_write | pass | expect_flagged=false | Created stresscase.go with harness types. | not flagged | legitimate create |
