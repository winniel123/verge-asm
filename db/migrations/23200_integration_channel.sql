-- +goose Up
-- An installed Integration may bind ONE delivery Channel as its delivery target
-- (collision #39 ruled option b, P0.14). An integration is NOT a channel and holds
-- no delivery target of its own (integration_state is slug/state; Channels are an
-- independent term per migration 20600) — so the binding is a REFERENCE to a Channel,
-- never a fold of one into the integration. The integration adds formatting on top of
-- the channel's transport; the channel still carries the bytes.
--
-- This mirrors the report_schedule → Channel binding (#17, migration 22700): a nullable
-- BIGINT reference, NULL meaning unbound (the freshly-installed default — nothing to
-- deliver a test through, and the drawer's "Send test" stays disabled). The ONE
-- difference is ON DELETE SET NULL rather than RESTRICT: deleting a Channel an
-- integration was bound to degrades that integration to unbound (the honest state the
-- drawer already renders) rather than stranding the delete — an integration is a
-- softer, formatting-layer binding than a report schedule's declared destination.
--
-- Only installed integrations carry a row at all, so only they can bind; an available
-- integration (no row) has nothing to bind and the drawer offers no channel select.
ALTER TABLE integration_state
    ADD COLUMN channel_id BIGINT REFERENCES channel (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE integration_state
    DROP COLUMN channel_id;
