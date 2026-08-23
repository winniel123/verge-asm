-- +goose Up
-- A report_schedule is the operator's Declared assertion that a report should be
-- exported on a cadence (Reports screen wizard, #285/#290). It is Declared and
-- carries no timeline: a schedule re-declared with different contents is created
-- anew, never an existing one recomputed. Scheduling produces no Observation and
-- never touches the comparison path — it is a delivery intent the operator sets,
-- read back onto the "Recurring reports" table.
--
-- Tenancy follows the estate's own per-record convention (seed, proposal, channel):
-- this build is single-estate, so a schedule is not account-scoped but is
-- attributed to the admin who declared it via created_by. created_by is NOT NULL,
-- matching seed/channel, so a schedule is a durable record of who set it. There is
-- no per-schedule delivery backing store yet (#291/T3), so "last sent" is resolved
-- from the Message/Delivery corpus at read time rather than stamped here — this
-- table holds only the declared intent.
--
-- sections is the small set of report sections the export includes. It is modelled
-- as JSONB — the estate's one precedent for a structured list column (message.census,
-- span.value, observation.value are all JSONB) — carrying a JSON array of section
-- keys. There is no text[] anywhere in the schema, so JSONB keeps the column type
-- vocabulary uniform.
CREATE TABLE report_schedule (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name            TEXT NOT NULL,
    -- The report sections the export includes, a JSON array of section keys
    -- (e.g. ["summary-kpis","new-assets"]). Empty array where none were chosen.
    sections        JSONB NOT NULL DEFAULT '[]',
    -- The delivery cadence token the wizard offers (6h / daily / weekly / monthly /
    -- custom). A plain label the table renders in mono; the dispatcher that would run
    -- it on cadence is a later ticket.
    cadence         TEXT NOT NULL,
    -- The export format (pdf / csv), rendered as the row's format badge.
    format          TEXT NOT NULL,
    -- Where the export is delivered — a free-form target the operator names (an
    -- address, a note, or empty for download-only). Kept as declared; there is no
    -- channel binding yet.
    delivery_target TEXT NOT NULL DEFAULT '',
    created_by      BIGINT NOT NULL REFERENCES account (id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The Reports screen lists every schedule newest-first; a plain index over the
-- ordering key keeps that read cheap.
CREATE INDEX report_schedule_recent ON report_schedule (id DESC);

-- +goose Down
DROP TABLE report_schedule;
