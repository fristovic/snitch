-- 007_model.sql — model attribution on runs.
--
-- Harnesses that expose the underlying model on assistant messages (Pi today)
-- record it per run. Used for "which model false-claims most" analytics and
-- included in opt-in telemetry labels.

ALTER TABLE runs ADD COLUMN model TEXT;
