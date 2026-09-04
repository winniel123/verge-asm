-- name: CreateProposerLookup :one
INSERT INTO proposer_lookup (query, created_by)
VALUES ($1, $2)
RETURNING id, query, created_by, created_at;

-- name: CreateProposal :one
INSERT INTO proposal (lookup_id, source_slug, record_kind, address_cidr, org_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, lookup_id, source_slug, record_kind, address_cidr, org_name, status, confirmed_seed_id, created_at;

-- name: ListPendingProposals :many
SELECT p.id, p.lookup_id, p.source_slug, p.record_kind, p.address_cidr, p.org_name,
       l.query AS lookup_query, l.created_at AS lookup_at, a.username AS lookup_by
FROM proposal p
JOIN proposer_lookup l ON l.id = p.lookup_id
JOIN account a ON a.id = l.created_by
WHERE p.status = 'pending'
ORDER BY l.created_at DESC, l.id DESC, p.id ASC;

-- name: GetPendingProposal :one
SELECT id, lookup_id, source_slug, record_kind, address_cidr, org_name, status, confirmed_seed_id, created_at
FROM proposal
WHERE id = $1 AND status = 'pending';

-- name: ConfirmProposal :execrows
UPDATE proposal
SET status = 'confirmed', confirmed_seed_id = $2
WHERE id = $1 AND status = 'pending';

-- name: DeclineLookup :execrows
UPDATE proposal
SET status = 'declined'
WHERE lookup_id = $1 AND status = 'pending';

-- name: DeclineProposal :execrows
UPDATE proposal
SET status = 'declined'
WHERE id = $1 AND status = 'pending';
