CREATE TABLE IF NOT EXISTS runs (
    id              TEXT PRIMARY KEY,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    session_id      TEXT,
    transcript_path TEXT,
    project_path    TEXT,
    harness         TEXT NOT NULL DEFAULT 'cursor',
    command         TEXT,
    duration_ms     INTEGER,
    output_hash     TEXT,
    tool_call_count INTEGER DEFAULT 0,
    verdict         TEXT NOT NULL DEFAULT 'unverified',
    max_severity    INTEGER,
    claim_count     INTEGER DEFAULT 0,
    verified_claims INTEGER DEFAULT 0,
    false_claims    INTEGER DEFAULT 0,
    device_id       TEXT NOT NULL,
    trace           TEXT
);

CREATE TABLE IF NOT EXISTS claims (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id      TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    claim_type  TEXT NOT NULL,
    source      TEXT NOT NULL DEFAULT 'prose',
    target      TEXT NOT NULL DEFAULT '',
    claimed     TEXT NOT NULL,
    actual      TEXT,
    verified    INTEGER DEFAULT 0,
    severity    INTEGER DEFAULT 0,
    verifier    TEXT,
    evidence    TEXT,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_claims_run_id ON claims(run_id);
CREATE INDEX IF NOT EXISTS idx_claims_claim_type ON claims(claim_type);
CREATE INDEX IF NOT EXISTS idx_claims_severity ON claims(severity);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at);
CREATE INDEX IF NOT EXISTS idx_runs_verdict ON runs(verdict);
CREATE INDEX IF NOT EXISTS idx_runs_project_path ON runs(project_path);
CREATE INDEX IF NOT EXISTS idx_runs_session_id ON runs(session_id);

CREATE TABLE IF NOT EXISTS analytics_queue (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    period_start TEXT NOT NULL,
    period_end   TEXT NOT NULL,
    payload     TEXT NOT NULL,
    sent        INTEGER DEFAULT 0,
    sent_at     TEXT,
    attempt     INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS config (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
