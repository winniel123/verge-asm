-- +goose Up
-- Prober lifecycle facts observed off-host on the connect that pins the host key
-- (P0.8, #683, ADR-0103). When the worker reaches a provisioned prober over SSH it
-- learns two facts about that position that the VantageCard renders (#26c) and that
-- were fixture stubs until this ticket:
--
--   platform — the remote OS and CPU the prober runs on, read from `uname -s`/
--              `uname -m` on first connection and rendered as the accepted-platform
--              chip. It is also what the arch check matches the pushed binary to
--              (packaging-and-configuration.md §1.5): the instance ships the exact
--              binary for this architecture over SSH, so a mismatch is refused
--              rather than pushed.
--   egress  — the source address the probe leaves from, read from SSH_CLIENT on the
--              connection (the first field, the client address). The instance's own
--              outbound address as an outside host observes it; the card offers it
--              for declaration on /scope so the estate knows its own egress.
--
-- Both are NULLABLE and set together only from a real successful connection — never
-- fabricated. They are NULL for the resolver-only `local` vantage (which has no
-- prober) exactly like the other prober columns, and NULL for a provisioned prober
-- until its first connection lands. Until then the card's platform/egress regions
-- collapse, as they do today.
ALTER TABLE vantage ADD COLUMN platform TEXT;
ALTER TABLE vantage ADD COLUMN egress   TEXT;

-- +goose Down
ALTER TABLE vantage DROP COLUMN egress;
ALTER TABLE vantage DROP COLUMN platform;
