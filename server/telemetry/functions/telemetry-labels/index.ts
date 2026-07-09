// POST /api/v1/telemetry/labels — ingest a batch of labeled verdicts.
// Opt-in training payload may include claim_sentence, claim_context, claimed,
// actual. Never prompts, code, or file paths.
// Body: { device_id, snitch_version, labels: [...] }.
import { createClient } from "jsr:@supabase/supabase-js@2";

const supabase = createClient(
  Deno.env.get("SUPABASE_URL")!,
  Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!,
);

const MAX_BATCH = 200;
const MAX_TEXT = 2000;

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
    run_id: str(l.run_id, 256),
    harness: str(l.harness, 64),
    model: str(l.model, 128),
    claim_type: str(l.claim_type, 64),
    verdict: str(l.verdict, 32),
    label_verdict: str(l.label_verdict, 32) || "unknown",
    claimed_text_hash: str(l.claimed_text_hash, 128),
    claim_sentence: str(l.claim_sentence, MAX_TEXT),
    claim_context: str(l.claim_context, MAX_TEXT),
    claimed: str(l.claimed, MAX_TEXT),
    actual: str(l.actual, MAX_TEXT),
    labeled_at: str(l.labeled_at, 64) || null,
  }));

  const { error } = await supabase.from("labels").insert(rows);
  if (error) {
    console.error("insert labels failed", error.message);
    return new Response("internal error", { status: 500 });
  }
  return Response.json({ ok: true, accepted: rows.length });
});

function str(v: unknown, max: number): string {
  return typeof v === "string" ? v.slice(0, max) : "";
}
