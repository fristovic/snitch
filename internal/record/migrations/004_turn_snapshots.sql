ALTER TABLE runs ADD COLUMN payload_json TEXT;
ALTER TABLE runs ADD COLUMN start_head TEXT;
ALTER TABLE runs ADD COLUMN end_head TEXT;
ALTER TABLE runs ADD COLUMN file_manifest_json TEXT;

ALTER TABLE claims ADD COLUMN confidence INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_runs_session_created ON runs(session_id, created_at);
