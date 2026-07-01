-- Lie detector schema upgrades for existing databases.
ALTER TABLE claims ADD COLUMN claim_type TEXT NOT NULL DEFAULT '';
ALTER TABLE claims ADD COLUMN source TEXT NOT NULL DEFAULT 'prose';
UPDATE claims SET claim_type = tool_call WHERE claim_type = '' AND tool_call IS NOT NULL AND tool_call != '';
UPDATE claims SET source = 'tool' WHERE claim_type IN ('Write','StrReplace','Delete','Read','Glob','Shell','Task');
CREATE INDEX IF NOT EXISTS idx_claims_claim_type ON claims(claim_type);
CREATE INDEX IF NOT EXISTS idx_runs_session_id ON runs(session_id);
