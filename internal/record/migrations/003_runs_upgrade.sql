-- Upgrade legacy runs tables from pre-lie-detector schemas.
ALTER TABLE runs ADD COLUMN session_id TEXT;
ALTER TABLE runs ADD COLUMN transcript_path TEXT;
ALTER TABLE runs ADD COLUMN project_path TEXT;
ALTER TABLE runs ADD COLUMN tool_call_count INTEGER DEFAULT 0;
ALTER TABLE runs ADD COLUMN trace TEXT;
CREATE INDEX IF NOT EXISTS idx_runs_project_path ON runs(project_path);
CREATE INDEX IF NOT EXISTS idx_runs_session_id ON runs(session_id);
