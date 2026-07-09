-- Normalize legacy PascalCase tool claim_type values to tool_* snake_case.
UPDATE labels SET claim_type = 'tool_write' WHERE claim_type = 'Write';
UPDATE labels SET claim_type = 'tool_str_replace' WHERE claim_type = 'StrReplace';
UPDATE labels SET claim_type = 'tool_delete' WHERE claim_type = 'Delete';
UPDATE labels SET claim_type = 'tool_read' WHERE claim_type = 'Read';
UPDATE labels SET claim_type = 'tool_glob' WHERE claim_type = 'Glob';
UPDATE labels SET claim_type = 'tool_shell' WHERE claim_type = 'Shell';
UPDATE labels SET claim_type = 'tool_task' WHERE claim_type = 'Task';
