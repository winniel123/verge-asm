-- +goose Up
-- An Annotation is an operator's declaration, about one `(subject, signal-name)`
-- pair, that a fired rule is an accepted risk on a thing we are still measuring
-- (CONTEXT.md `Annotation`, v1 spec §6.5, ADR-0016/ADR-0073/ADR-0092/ADR-0093).
-- Its whole effect is on the message: a `not-fired` → `fired` `Transition` on an
-- annotated pair is recorded and is not a message. It moves no number — the pair
-- is still measured on its cadence, still inside the rule's `Predicate domain`,
-- and still counted under `fired`.
--
-- It is an operator dial, and every dial in the model is unattributed, so there
-- is NO author column (ADR-0073: an operator dial carries no author however
-- specific its target). It carries NO status and NO expiry — editing one is a new
-- `Annotation`, and an expiry would be a state moving because time passed. What it
-- does carry is the operator's reason in their own words and the instant it was
-- declared: it is the one Declared term holding operator prose and the only one
-- carrying an instant, earned because it cannot be edited (so the instant acquires
-- no successor) and its whole effect is a message that does not fire (so no
-- `Message` anywhere in the model dates the act, ADR-0093).
CREATE TABLE annotation (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    subject_key TEXT NOT NULL,
    signal_name TEXT NOT NULL,
    reason      TEXT NOT NULL,
    declared_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Keyed on the exact `(subject, signal-name)` pair, declared once. It never
-- travels: keyed on the exact subject, a redeploy onto a new address leaves the
-- acceptance matching nothing rather than silencing a subject nobody chose. A
-- returning subject is the same pair, so the mute takes effect again with no
-- operator act (ADR-0092). Re-declaring an existing pair is a duplicate, not an
-- edit — the reason is changed by withdrawing and declaring anew.
CREATE UNIQUE INDEX annotation_pair_key ON annotation (subject_key, signal_name);

-- +goose Down
DROP TABLE annotation;
