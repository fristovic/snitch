-- 005_labels.sql — user feedback labels on runs (v1 data flywheel).
--
-- Each run can carry an optional user verdict (correct/incorrect) plus a flag
-- for whether the user opted to share that verdict with the training pipeline.
-- label_session groups a feedback interaction for dedup. label_synced tracks
-- whether the telemetry sync goroutine has already forwarded this label.
--
-- All ALTER statements are idempotent: isBenignMigrationErr swallows
-- "duplicate column" on re-run, so this migration is safe to apply repeatedly.

ALTER TABLE runs ADD COLUMN label_verdict TEXT;             -- "correct" | "incorrect" | NULL
ALTER TABLE runs ADD COLUMN label_shared INTEGER DEFAULT 0; -- bool: user opted into sharing
ALTER TABLE runs ADD COLUMN label_timestamp TEXT;           -- RFC3339 of when labeled
ALTER TABLE runs ADD COLUMN label_session TEXT;             -- dedup/grouping key
ALTER TABLE runs ADD COLUMN label_synced INTEGER DEFAULT 0; -- bool: telemetry sync forwarded it
