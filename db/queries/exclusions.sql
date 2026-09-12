-- name: CreateNameExclusion :one
INSERT INTO exclusion (kind, name, created_by)
VALUES ($1, $2, $3)
RETURNING id, kind, name, address_cidr, created_by, created_at, proposal_id;

-- name: CreateAddressExclusion :one
-- proposal_id stays NULL, so an undo may not lift what the operator declared (#1799).
INSERT INTO exclusion (kind, address_cidr, created_by)
VALUES ('address', $1, $2)
RETURNING id, kind, name, address_cidr, created_by, created_at, proposal_id;

-- name: CreateDeclinedProposalExclusion :one
-- The decline names the proposal it answered, which is the row the undo may lift (#1799).
INSERT INTO exclusion (kind, address_cidr, created_by, proposal_id)
VALUES ('address', $1, $2, $3)
RETURNING id, kind, name, address_cidr, created_by, created_at, proposal_id;

-- name: ListExclusions :many
SELECT e.id, e.kind, e.name, e.address_cidr, e.created_by, e.created_at, e.proposal_id
FROM exclusion e
ORDER BY e.created_at DESC, e.id DESC;

-- name: ListAddressExclusionCidrs :many
SELECT address_cidr
FROM exclusion
WHERE kind = 'address' AND address_cidr IS NOT NULL
ORDER BY id;

-- name: DeleteExclusion :one
-- The scope rides the act's own RETURNING, so a separate read cannot leave the Act blank.
DELETE FROM exclusion WHERE id = $1
RETURNING kind, name, address_cidr;

-- name: DeleteUnclaimedAddressExclusion :one
-- A data-modifying CTE fires on its own, so nothing need select from lift (#1777).
WITH claim AS (
    SELECT 1 FROM proposal p
    WHERE p.status = 'declined' AND p.address_cidr = $1
), kept AS (
    SELECT proposal_id FROM exclusion
    WHERE kind = 'address' AND address_cidr = $1
), lift AS (
    DELETE FROM exclusion
    WHERE kind = 'address' AND address_cidr = $1
      -- A NULL proposal_id is a standing declaration, which no decline may lift (#1799).
      AND proposal_id IS NOT NULL
      AND NOT EXISTS (SELECT 1 FROM claim)
    RETURNING id
)
-- Every arm reads one snapshot, so kept is the row as it stood before lift (#1777).
SELECT
    EXISTS (SELECT 1 FROM kept WHERE kept.proposal_id IS NULL) AS declared_by_hand,
    -- The two reasons a row stays are independent, so the caller composes them (#1799).
    EXISTS (SELECT 1 FROM claim, kept) AS still_claimed;
