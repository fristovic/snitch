# Snitch Telemetry Server

Minimal Supabase project backing the opt-in data flywheel. Two endpoints, two
tables. Nothing else until data volume justifies it.

Served at `https://telemetry.snitchworks.com` (a custom domain in front of the
Supabase Edge Functions gateway).

## Endpoints

| Method | Path | Body |
|--------|------|------|
| POST | `/api/v1/telemetry/labels` | `{ device_id, snitch_version, labels: [...] }` |
| POST | `/api/v1/telemetry/register` | `{ device_id, snitch_version, platforms: [...] }` |

Both are anonymous (no auth) and rate-limited by the gateway. Label entries
carry metadata only: `harness`, `model`, `claim_type`, `verdict`,
`label_verdict`, `claimed_text_hash`, `labeled_at`. Never code, file paths, or
claim text — the hash exists purely for server-side dedup.

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

`label_accuracy` (see migrations) is the view that will power the public
accuracy page once enough labels exist:

```sql
SELECT claim_type, harness, agree, total, ROUND(100.0 * agree / total, 1) AS pct
FROM label_accuracy;
```
