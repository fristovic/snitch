-- Telemetry schema: devices + labels. Metadata only — no code, file paths,
-- or claim text ever lands here (claimed_text_hash is a dedup key).

create table if not exists devices (
    device_id       text primary key,          -- sha256(machine-id), client-side
    snitch_version  text not null,
    platforms       text[] not null default '{}',
    first_seen_at   timestamptz not null default now(),
    last_seen_at    timestamptz not null default now()
);

create table if not exists labels (
    id                bigint generated always as identity primary key,
    device_id         text not null,
    snitch_version    text not null,
    run_id            text,                     -- client-local id, opaque
    harness           text,
    model             text,
    claim_type        text,
    verdict           text,                     -- Snitch's original verdict
    label_verdict     text not null,            -- correct | incorrect | added
    claimed_text_hash text,                     -- sha256, dedup only
    labeled_at        timestamptz,
    received_at       timestamptz not null default now()
);

create index if not exists labels_claim_type_idx on labels (claim_type);
create index if not exists labels_hash_idx on labels (claimed_text_hash);

-- Aggregate view for the future public accuracy page: for each claim type and
-- harness, how often users agree with Snitch's verdict.
create or replace view label_accuracy as
select claim_type,
       harness,
       count(*) filter (where label_verdict = 'correct') as agree,
       count(*) filter (where label_verdict in ('correct','incorrect')) as total
from labels
where claim_type is not null
group by claim_type, harness;

-- Row Level Security: edge functions write via service role; nothing is
-- readable through the anon API.
alter table devices enable row level security;
alter table labels enable row level security;
