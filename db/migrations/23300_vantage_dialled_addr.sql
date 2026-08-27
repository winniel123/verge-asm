-- +goose Up
-- The dialled address a prober presented on the connect that pins its host key
-- (P0.8, #710, ADR-0103) — the third presented-address fact the Vantage-class
-- derivation reads, alongside egress (#683) and distinct from every column already
-- on the row:
--
--   dialled_addr — the address the instance actually dialled to reach the prober,
--                  observed off-host as the SSH transport's peer address
--                  (*ssh.Client.RemoteAddr()) on the key-pinning connect. It is the
--                  address an outside observer saw the vantage PRESENT, "known by
--                  construction" (CONTEXT.md `Vantage class`): the instance learns it
--                  locally at Dial without the prober reporting it. It is NOT `host`,
--                  which is operator-typed config and may be a HOSTNAME rather than an
--                  address, may resolve to several addresses of which only one was
--                  dialled, and so cannot stand in for the observed peer (#710).
--
-- Stored as a canonical `netip.Addr` string (family-normalised, `Unmap`ed), so the
-- Vantage-class derivation re-parses it with `netip.ParseAddr` at every classification
-- (exposure.VerifyClass over presentedAddrs). Like `egress`/`platform` it is NULLABLE
-- and set together with them in SetVantageProbeFacts, only from a real successful
-- connect — never fabricated. It is NULL for the resolver-only `local` vantage (no
-- prober) and for a provisioned prober until its first connect lands. A vantage with
-- no presented address (host/egress/dialled all NULL) derives `unverified`, exactly as
-- the vestigial `class` default reads today, so no existing golden moves.
ALTER TABLE vantage ADD COLUMN dialled_addr TEXT;

-- +goose Down
ALTER TABLE vantage DROP COLUMN dialled_addr;
