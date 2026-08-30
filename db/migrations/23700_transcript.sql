-- +goose Up
-- The Transcript corpus (raw-job-output spec §1, ADR-0126): verbatim raw job
-- output — stdout, stderr and the exact bytes sent to the prober — captured for
-- operator debugging. It is a NEW, FOURTH Operational corpus beside Dispatch,
-- Message and Delivery, keyed per job (queue_job grain, one row per attempt).
--
-- The fence (spec §1): no derivation may read a Transcript, exactly as none may
-- read a Dispatch. That fence is what makes a wall clock legal on it (§4). Only
-- the worker writes it; only the §6 read handler reads it. The verbatim bytes are
-- the whole volume, so they are retirable INDEPENDENTLY of the lean queue_job
-- record, by the §4 duration dial.
--
-- Overrides ADR-0041 (nothing raw is retained at rest). The at-rest encryption
-- (§5.3, ADR-0053 reversal) lands with a later ticket; this migration lands the
-- store as plain bytea. A job with NO capture is a legible absence — no row —
-- distinct from a captured-but-empty stream (an empty, non-null bytea).
CREATE TABLE transcript (
    -- Keyed on the queue_job id: one Transcript per attempt (spec §1.1). Retry
    -- enqueues a NEW queue_job row, so each attempt keeps its own transcript on
    -- its own row, retired independently by the §4 dial. ON DELETE CASCADE: a
    -- transcript is subordinate to its job and never outlives it.
    queue_job_id BIGINT PRIMARY KEY REFERENCES queue_job (id) ON DELETE CASCADE,

    -- The common frame (spec §1.2): the one job kind, the exec duration, and the
    -- capture instant. duration_ns is the whole exec span in nanoseconds (a Go
    -- time.Duration round-trips exactly as int64). captured_at is stamped by the
    -- producer (w.now()), not defaulted, so the §4 age sweep is reproducible —
    -- mirrors observation.observed_at.
    kind         TEXT NOT NULL,
    duration_ns  BIGINT NOT NULL,
    captured_at  TIMESTAMPTZ NOT NULL,

    -- The closed-union tag (spec §1.2). One variant per producer kind; a struct
    -- with optional fields is barred (CONTEXT.md: every value space is a closed
    -- union). Each variant carries its OWN typed outcome — exited/signalled/
    -- context-cancelled (prober), http/transport-error/context-cancelled (ct),
    -- parsed/decode-error (zone) — held verbatim in outcome as {"kind": ..., ...}.
    variant      TEXT NOT NULL CHECK (variant IN ('prober', 'ct', 'zone')),
    outcome      JSONB NOT NULL,

    -- The verbatim streams (spec §1.4, §3.2), one bytea column per stream — NEVER
    -- JSON-embedded. Each is nullable: a variant fills only the streams it
    -- captured. A NULL stream is one this variant does not carry; an empty bytea
    -- is a captured-but-empty stream. Each can reach its §3.2 store cap
    -- (stdout 4 MiB, stderr 256 KiB, sent-scope 64 KiB), applied by the producer
    -- via head+tail truncate-and-mark. The prober variant maps stdout/stderr/
    -- sent_scope directly; the ct and zone variants reuse the same role columns.
    stdout       BYTEA,
    stderr       BYTEA,
    sent_scope   BYTEA,

    -- Per-stream truncation markers (spec §3.2): {kept, dropped} byte counts per
    -- stream, or a memory-guard-tripped marker when the 64 MiB LimitedBuffer guard
    -- errored the job. Keyed by stream name, e.g.
    -- {"stdout": {"kept": N, "dropped": M}, "stderr": {...}}. Defaults to an empty
    -- object where nothing truncated. The markers describe the bytea streams; they
    -- do not embed them.
    truncation   JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The Transcript retention dial (spec §4, ADR-0126 amends ADR-0041). A duration
-- dial in DAYS, mirroring observation_currency_days — "keep raw output for N
-- days" — added to the retention_settings singleton. It SHIPS BOUNDED at 14 days:
-- this non-zero default IS the ADR-0041 reversal (which shipped 0=unbounded on
-- both retirable corpora), because verbatim bytes are the volume problem on the
-- address-scope installs that motivated retention. 0 == unbounded (explicit
-- operator opt-out, the existing sentinel); the 1-day floor for a positive value
-- is enforced in code (internal/retention), like the observation floor. The
-- ADD COLUMN DEFAULT 14 backfills the existing singleton, so live installs also
-- ship bounded. No coverage-style floor: no derivation reads a Transcript.
ALTER TABLE retention_settings
    ADD COLUMN transcript_currency_days BIGINT NOT NULL DEFAULT 14
        CHECK (transcript_currency_days >= 0);

-- +goose Down
ALTER TABLE retention_settings DROP COLUMN transcript_currency_days;
DROP TABLE transcript;
