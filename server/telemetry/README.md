# Snitch Telemetry Server

Optional Supabase project for the upcoming opt-in labeling flywheel. Two
endpoints, two tables.

Intended hostname: `https://telemetry.snitchworks.com` (custom domain in front
of Supabase Edge Functions).

## Endpoints

| Method | Path | Body |
|--------|------|------|
| POST | `/api/v1/telemetry/labels` | `{ device_id, snitch_version, labels: [...] }` |
| POST | `/api/v1/telemetry/register` | `{ device_id, snitch_version, platforms: [...] }` |

Both are anonymous (no auth) and rate-limited by the gateway. Sharing is
**fully opt-in** on the client (`telemetry.enabled` + share flag).

### What a shared label may include

- Metadata: `harness`, `model`, `claim_type`, Snitch `verdict`, user `label_verdict`, `claimed_text_hash`, timestamps
- Training text (for the future false-positive classifier): `claim_sentence` (full sentence containing the match), `claim_context` (capped ±1–2 surrounding sentences), `claimed` / `actual` (Snitch’s claimed→actual pair)

### What is never collected

User prompts, assistant transcripts beyond the capped claim window, source code, file paths, project paths, or shell output dumps.

## Deploy

```bash
supabase init            # once, in this directory
supabase db push         # applies migrations/
supabase functions deploy telemetry-labels
supabase functions deploy telemetry-register
# then map telemetry.snitchworks.com → the functions gateway, with rewrites:
#   /api/v1/telemetry/labels   → /functions/v1/telemetry-labels
#   /api/v1/telemetry/register → /functions/v1/telemetry-register
```

## Accuracy aggregates

`label_accuracy` (see migrations) is the view that will power a public
accuracy page once enough labels exist:

```sql
SELECT claim_type, harness, agree, total, ROUND(100.0 * agree / total, 1) AS pct
FROM label_accuracy;
```
