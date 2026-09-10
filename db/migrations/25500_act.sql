-- +goose Up
-- The Act corpus (audit-act spec §4.1): one recorded act by one principal on
-- this instance. It is a NEW, FIFTH Operational corpus beside Dispatch,
-- Message, Delivery and Transcript.
--
-- This withdraws a refusal rather than filling a gap. #127 ruled an operator-act
-- record out of v1 and ADR-0073 §1 made the refusal total. The demand side —
-- 18 sentences the SPEC enumerates at §8 · C — is what reopens it.
--
-- The predicate has four limbs (§1): the estate's declaration, who may act on
-- this instance, directing the instance to act on the network, and a disclosure
-- from a corpus the model seals. 61 classes satisfy it; 23 acts are exempt with
-- a stated reason each (§2.3).
--
-- The Operational fence holds: no derivation may read an Act, exactly as none
-- may read a Dispatch or a Transcript. That fence is what makes a wall clock
-- legal on it, and what keeps operator identity out of the comparison path.
CREATE TABLE act (
    id          BIGSERIAL PRIMARY KEY,

    -- The recording instant, never the act's own (§4.1). now() is the idiom all
    -- four sibling corpora share, and the caller cannot forge it. The recorder
    -- is not atomic with the act it records (§7.6), so the two can differ.
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- The Actor union's discriminator (§3). Two members, never a nullable
    -- account_id, because null is a shape and not a meaning (ADR-0126's idiom).
    -- It discriminates on HOW the principal proved themselves to the instance —
    -- a session or a grant — never on who the row is about.
    --
    -- 'system' is absent on purpose (§3.4). A migration is not a principal and
    -- writes no Act, so the variant would be uninhabited: a closed union exists
    -- to make the encoder exhaustive, and an unexercised variant makes an
    -- unreachable state compile.
    actor_kind  TEXT NOT NULL CHECK (actor_kind IN ('account','grant_holder')),

    -- account: {account_id, username_snapshot}. grant_holder: the Grant union —
    -- setup token, password-reset link, or invite{invite_id}. A grant carries
    -- its id exactly when a second Act names the same grant (§3.3), which is
    -- true of the invite alone.
    --
    -- The username is a captured VALUE and not a join, and there is NO FK to
    -- account (§5.4). Copy the shipped attribution columns and no admin who has
    -- ever acted could be removed at all, because those columns restrict. An FK
    -- from an Operational record into the identity table also re-couples what
    -- the Operational fence exists to keep apart. username is UNIQUE but
    -- reusable after a delete, so the row carries the id AND the name.
    actor       JSONB NOT NULL,

    -- The closed-union tag over 61 variants (§4), one per act class, each
    -- naming its own typed subject. An action enum beside a subject union was
    -- weighed and rejected: it makes an illegal pair expressible.
    --
    -- NO CHECK constraint, and this diverges from transcript's 3-token CHECK
    -- deliberately (§4.1). A 61-token constraint needs a migration per act
    -- class and fails at runtime rather than in CI. The exhaustive Go encoder
    -- and §7's AST conformance gate hold the set instead.
    action      TEXT NOT NULL,

    -- The typed payload, rendered at read time and never a frozen sentence
    -- (§4.1). Price stated rather than hidden: a later copy change re-renders
    -- history, which is right against a corpus that generates no UPDATE.
    --
    -- No variant carries a list-valued subject: one Act per subject, never one
    -- per request. POST /coverage/retention therefore writes two rows (§2.2).
    -- Nothing in the render path touches a store, so a withdrawn subject still
    -- renders (§4.2) and an Act may outlive its subject by design (§4.3).
    subject     JSONB NOT NULL
);

-- The reader is a server-side date range over an unbounded, never-deleted
-- corpus (§5.1, §6.2), so the range scan needs its own index rather than a
-- growing sequential scan. Descending matches the newest-first render.
CREATE INDEX act_created_at_idx ON act (created_at DESC, id DESC);

-- +goose Down
DROP TABLE act;
