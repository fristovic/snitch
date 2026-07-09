-- 006_missed_claims.sql — user-reported false negatives (v1 data flywheel).
--
-- When an agent made a false claim Snitch missed, the user can report it:
-- what the agent claimed vs what actually happened. These are the
-- highest-value training examples — claims regex can't catch. Claim/actual
-- text is stored locally only. Telemetry (if opted in) shares metadata plus a
-- hash, never the text itself.

CREATE TABLE IF NOT EXISTS missed_claims (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id     TEXT,                      -- optional link to a run
    claimed    TEXT NOT NULL,             -- what the agent said
    actual     TEXT NOT NULL,             -- what actually happened
    shared     INTEGER DEFAULT 0,         -- user opted into sharing
    synced     INTEGER DEFAULT 0,         -- telemetry sync forwarded it
    created_at TEXT NOT NULL
);
