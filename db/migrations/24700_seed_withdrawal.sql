-- +goose Up
-- A tombstone for a withdrawn address `Seed` (ADR-0134 §2, ticket #1040).
--
-- Withdrawing a `Seed` is the second of the two narrowing acts `CONTEXT.md` names.
-- An exclusion is the first, and #1032 discharged it by reading the live
-- `exclusion` corpus: the mover is a row the membership fold can still read after
-- the act. A `Seed` withdrawal destroys the mover in the same statement that
-- performs the act, so nothing in the database would name what moved.
--
-- That matters because the fold must name the mover. `composeAddressWithdrawals`
-- already refuses to close a span it cannot attribute to a declared act, because a
-- closure with no mover is a withdrawal the operator cannot trace back to their
-- own act. This table is that mover, standing in for the `exclusion` row the other
-- act reads live.
--
-- The `seed` delete stays a hard DELETE, so every existing reader of `seed` is
-- untouched and the FK cascade `TestSeedForeignKeysCascadeOnDelete` guards is
-- unchanged. A soft delete would put a filter on every one of those readers, and a
-- missed filter re-admits a withdrawn scope silently.
--
-- ADDRESS ONLY. An address `Seed`'s display scope IS its CIDR, so the CIDR alone
-- carries both the mover's identity and the message's firing site, and no display
-- string is stored. The name limb is a different message contract and is out of
-- scope (ADR-0134 §7).
--
-- `created_by` records the actor, as `seed` and `exclusion` do: a withdrawal is the
-- counterpart act to a declaration, so the two are recorded the same way. The actor
-- stops here. It never reaches the span and never reaches the message, so ADR-0087's
-- refusal to record an actor on a CLOSURE is not weakened.
--
-- It is NULLABLE and ON DELETE SET NULL, which is where it parts from `seed` and
-- `exclusion`. Those two carry a NOT NULL FK that deliberately refuses to remove an
-- account which authored a live declaration, and the operator can lift the refusal
-- by removing the declaration. A tombstone is not removable by any operator act, so
-- the same FK would pin the account for ever, with no cleanup path even after the
-- row is spent. The attribution is worth keeping while the account exists and is
-- not worth making a member undeletable.
--
-- No unique index on the CIDR. Withdraw a scope, declare it again and withdraw it
-- again, and each act is its own dated fact.
CREATE TABLE seed_withdrawal (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    address_cidr      CIDR NOT NULL,
    created_by        BIGINT REFERENCES account (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- The batch that spent the tombstone (ADR-0134 §5). The fold stamps it once
    -- the withdrawal has taken everything it can take, and the listing query
    -- filters a stamped row out. Idempotency does not depend on the stamp — the
    -- three survivor rules give the same by-construction idempotency the exclusion
    -- act has — but the stamp bounds the read, satisfies ADR-0041 (a spent row is
    -- a row nothing may still read), and records WHICH batch performed the
    -- withdrawal, mirroring ADR-0111 at the declared input.
    --
    -- LIVENESS depends on the stamp, which is why it is not written eagerly. Two
    -- of ADR-0134 §4's survivors are TRANSIENT: a citing resolution goes away, and
    -- a custody extension is turned off. The exclusion twin re-reads its live
    -- `exclusion` row every batch, so it acts whenever a survivor lapses. A
    -- tombstone gets one shot, and spending it while its ground is still held
    -- would leave those addresses open for ever with no mover — the very leak this
    -- table exists to close. So the fold spends a row only once no open timeline
    -- remains under its CIDR (SpendSeedWithdrawals).
    --
    -- `consumed_at` is the filter, never `consumed_batch_id`. The FK sets the
    -- batch id NULL if its batch ever goes, and filtering on the id would then
    -- resurrect a spent tombstone and withdraw the ground a second time.
    consumed_at       TIMESTAMPTZ,
    consumed_batch_id BIGINT REFERENCES batch (id) ON DELETE SET NULL,

    CONSTRAINT seed_withdrawal_consumed_shape CHECK (
        consumed_at IS NOT NULL OR consumed_batch_id IS NULL
    )
);

-- The fold reads the unspent rows on every completed job, so the read is bounded
-- by what is still owed rather than by the whole history of withdrawals.
CREATE INDEX seed_withdrawal_pending_idx ON seed_withdrawal (id) WHERE consumed_at IS NULL;

-- +goose Down
DROP TABLE seed_withdrawal;
