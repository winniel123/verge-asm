-- name: CreateProposerLookup :one
-- Records one operator act — an org-name search — under which a batch of
-- candidate scopes is filed. It is the unit a bulk decline operates over.
INSERT INTO proposer_lookup (query, created_by)
VALUES ($1, $2)
RETURNING id, query, created_by, created_at;

-- name: CreateProposal :one
-- Files one candidate scope a proposer offered. It enters as 'pending' and is
-- read by nothing until it is confirmed into a Seed.
INSERT INTO proposal (lookup_id, source_slug, record_kind, address_cidr, org_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, lookup_id, source_slug, record_kind, address_cidr, org_name, status, confirmed_seed_id, created_at;

-- name: ListPendingProposals :many
-- The pending Proposals the Seeds screen renders, grouped for the caller by
-- lookup so each lookup carries its own bulk-decline act. Only 'pending' rows
-- surface: a confirmed Proposal is already a Seed and a declined one is spent.
SELECT p.id, p.lookup_id, p.source_slug, p.record_kind, p.address_cidr, p.org_name,
       l.query AS lookup_query, l.created_at AS lookup_at, a.username AS lookup_by
FROM proposal p
JOIN proposer_lookup l ON l.id = p.lookup_id
JOIN account a ON a.id = l.created_by
WHERE p.status = 'pending'
ORDER BY l.created_at DESC, l.id DESC, p.id ASC;

-- name: GetPendingProposal :one
-- One pending Proposal, read at the moment of confirmation so the confirm act
-- can copy its scope into a Seed. A Proposal already confirmed or declined does
-- not come back, so a double submit cannot open the gate twice.
SELECT id, lookup_id, source_slug, record_kind, address_cidr, org_name, status, confirmed_seed_id, created_at
FROM proposal
WHERE id = $1 AND status = 'pending';

-- name: ConfirmProposal :execrows
-- Marks a single Proposal confirmed and retains the Seed it became as provenance.
-- Guarded on status = 'pending' so a concurrent or repeated confirm is a no-op
-- rather than a second Seed: confirmation is singular (ADR-0022).
UPDATE proposal
SET status = 'confirmed', confirmed_seed_id = $2
WHERE id = $1 AND status = 'pending';

-- name: DeclineLookup :execrows
-- Declines every still-pending Proposal under one lookup in a single act
-- (ADR-0022: declining may be bulk over a whole lookup). Declining is safe to
-- batch because a pending Proposal is read by nothing, so 'declined' and 'never
-- answered' have the same effect on the gate.
UPDATE proposal
SET status = 'declined'
WHERE lookup_id = $1 AND status = 'pending';

-- name: DeclineProposal :execrows
-- Declines a single still-pending Proposal by id (#21: the Scope decline-many
-- act declines each checked proposal). Guarded on status = 'pending' so a repeat
-- or concurrent decline is a no-op. A declined scope is recorded as an exclusion
-- by the handler, off the row read before the decline.
UPDATE proposal
SET status = 'declined'
WHERE id = $1 AND status = 'pending';
