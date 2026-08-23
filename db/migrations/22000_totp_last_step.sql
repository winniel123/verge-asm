-- +goose Up
-- Record the last TOTP counter step this account has successfully authenticated
-- with, so a captured code cannot be replayed within its ~90s validity window
-- (#323, RFC 6238 §5.2). VerifyTOTP is otherwise stateless and accepts steps
-- -1/0/+1, so without this a valid code redeems repeatedly until its window
-- closes — the same single-use discipline recovery codes already hold. NULL means
-- no TOTP login has completed yet (a freshly enrolled account), so the first
-- completion is unconstrained; every later one must present a strictly greater
-- step. Nullable and forward-only: existing accounts carry NULL until their next
-- TOTP sign-in stamps a step.
ALTER TABLE account ADD COLUMN totp_last_step BIGINT;

-- +goose Down
ALTER TABLE account DROP COLUMN totp_last_step;
