-- 008_claim_training_text.sql — sentence + context for opt-in flywheel training.
-- Additive columns. isBenignMigrationErr swallows duplicate-column on re-run.

ALTER TABLE claims ADD COLUMN claim_sentence TEXT;
ALTER TABLE claims ADD COLUMN claim_context TEXT;
