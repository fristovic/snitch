-- 0002_training_text.sql — opt-in claim sentence/context for FP-classifier training.
-- Additive; safe to apply after 0001_init.

alter table labels add column if not exists claim_sentence text;
alter table labels add column if not exists claim_context text;
alter table labels add column if not exists claimed text;
alter table labels add column if not exists actual text;
