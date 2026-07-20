-- Per-claim epistemic verdict (supported | contradicted | missing | stale).
ALTER TABLE claims ADD COLUMN epistemic TEXT NOT NULL DEFAULT '';

UPDATE claims SET epistemic = CASE
  WHEN verified = 1 THEN 'supported'
  WHEN verified = -1 THEN 'contradicted'
  ELSE 'missing'
END WHERE epistemic = '';
