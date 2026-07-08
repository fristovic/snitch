// POST /api/v1/telemetry/register — announce a device (opt-in, anonymous).
// Body: { device_id, snitch_version, platforms: [...] }.
import { createClient } from "jsr:@supabase/supabase-js@2";

const supabase = createClient(
  Deno.env.get("SUPABASE_URL")!,
  Deno.env.get("SUPABASE_SERVICE_ROLE_KEY")!,
);

Deno.serve(async (req) => {
  if (req.method !== "POST") {
    return new Response("method not allowed", { status: 405 });
  }
  let body: { device_id?: string; snitch_version?: string; platforms?: string[] };
  try {
    body = await req.json();
  } catch {
    return new Response("invalid json", { status: 400 });
  }
  if (!body.device_id) {
    return new Response("device_id required", { status: 400 });
  }

  const { error } = await supabase.from("devices").upsert({
    device_id: body.device_id.slice(0, 128),
    snitch_version: (body.snitch_version ?? "").slice(0, 64),
    platforms: (body.platforms ?? []).slice(0, 10).map((p) => String(p).slice(0, 32)),
    last_seen_at: new Date().toISOString(),
  }, { onConflict: "device_id" });
  if (error) {
    console.error("register failed", error.message);
    return new Response("internal error", { status: 500 });
  }
  return Response.json({ ok: true });
});
