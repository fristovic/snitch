// POST /api/v1/telemetry/labels — ingest a batch of labeled verdicts.
// Anonymous, metadata-only. Body: { device_id, snitch_version, labels: [...] }.
import { createClient } from "jsr:@supabase/supabase-js@2";

const supabase = createClient(
  Deno.env.get("SUPABASE_URL")!,
  Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!,
);

const MAX_BATCH = 200;

Deno.serve(async (req) => {
  if (req.method !== "POST") {
    return new Response("method not allowed", { status: 405 });
  }
  let body: {
    device_id?: string;
    snitch_version?: string;
    labels?: Array<Record<string, unknown>>;
  };
  try {
    body = await req.json();
  } catch {
    return new Response("invalid json", { status: 400 });
  }
  if (!body.device_id || !Array.isArray(body.labels) || body.labels.length === 0) {
    return new Response("device_id and labels required", { status: 400 });
  }
  if (body.labels.length > MAX_BATCH) {
    return new Response("batch too large", { status: 413 });
  }

  const rows = body.labels.map((l) => ({
    device_id: body.device_id,
    snitch_version: body.snitch_version ?? "",
    run_id: str(l.run_id),
    harness: str(l.harness),
    model: str(l.model),
    claim_type: str(l.claim_type),
    verdict: str(l.verdict),
    label_verdict: str(l.label_verdict) || "unknown",
    claimed_text_hash: str(l.claimed_text_hash),
    labeled_at: str(l.labeled_at) || null,
  }));

  const { error } = await supabase.from("labels").insert(rows);
  if (error) {
    console.error("insert labels failed", error.message);
    return new Response("internal error", { status: 500 });
  }
  return Response.json({ ok: true, accepted: rows.length });
});

function str(v: unknown): string {
  return typeof v === "string" ? v.slice(0, 256) : "";
}
