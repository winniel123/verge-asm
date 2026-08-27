-- +goose Up
-- The aperture-widened marker on a span opening (#637, SPEC-CHANGE #32). A drift
-- Transition names an opening `revealed` where a widened aperture opened the
-- timeline — "we started looking, the world did not move" (drift.OpeningKind,
-- ADR-0014) — as distinct from `appeared`, where a subject ENTERED the estate
-- (the world moved). The two are indistinguishable from the span's value alone:
-- both are a first span on a timeline. The signal is WHY the timeline opened —
-- the operator's declared aperture reaching a subject (a Seed covering it), versus
-- the world bringing a subject the aperture had not declared (a resolved address,
-- a CT-admitted name).
--
-- The estate composition (internal/estate) is the one place that knows a subject
-- is declared input — Seed-covered — rather than measured-present; wiring it into
-- the spanfold closure is what lets the fold stamp this marker at open time so the
-- estate-wide drift feed can read `revealed` instead of narrating every first span
-- as `appeared`. It is carried ONLY by a span the fold opened; it is never set on
-- a closure and never rewritten.
--
-- Boolean, NOT NULL, defaulting FALSE: every existing span (and every ordinary
-- world-measured opening) reads FALSE — the honest default, which keeps the feed's
-- `appeared` classification unchanged for everything already recorded.
ALTER TABLE span
    ADD COLUMN opened_aperture BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE span DROP COLUMN opened_aperture;
