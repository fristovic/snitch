-- Normalize legacy PascalCase tool claim_type values to tool_* snake_case.
UPDATE claims SET claim_type = 'tool_write' WHERE claim_type = 'Write' AND source = 'tool';
UPDATE claims SET claim_type = 'tool_str_replace' WHERE claim_type = 'StrReplace' AND source = 'tool';
UPDATE claims SET claim_type = 'tool_delete' WHERE claim_type = 'Delete' AND source = 'tool';
UPDATE claims SET claim_type = 'tool_read' WHERE claim_type = 'Read' AND source = 'tool';
UPDATE claims SET claim_type = 'tool_glob' WHERE claim_type = 'Glob' AND source = 'tool';
UPDATE claims SET claim_type = 'tool_shell' WHERE claim_type = 'Shell' AND source = 'tool';
UPDATE claims SET claim_type = 'tool_task' WHERE claim_type = 'Task' AND source = 'tool';
