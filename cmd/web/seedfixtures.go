package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

// The fixed dev operator the fixture loader seeds so an authenticated screen like
// /inventory is reachable by the pixel-parity harness (which logs in with these
// before it navigates). These are DEV-ONLY credentials — the loader is barred
// outside a VERGE_DEV build — so a well-known password is acceptable and keeps the
// harness login deterministic. The harness references the same two constants.
const (
	devSeedUsername = "operator"
	devSeedPassword = "verge-dev-operator" // #nosec G101 -- not a real credential: a fixed dev-only fixture operator the loader seeds, and the loader is barred outside a VERGE_DEV build
)

// The Inventory fixture loader (#525, P4.0 exact-parity pilot). It is a dev-only
// one-shot: `web -seed-fixtures <path>` reseeds a dev database's open-span corpus
// with the exact spans that make `/inventory` render
// design-system/fixtures/fixtures.json byte-for-byte, then exits before the HTTP
// server starts. It exists only so the pixel-parity harness has a deterministic
// instance to screenshot; it is barred outside a VERGE_DEV build (main.go) and
// touches nothing but the span table.
//
// The 16 spans below ARE the fixture: buildInventory over them reproduces the
// fixture's groups/subjects/facets exactly (the inventory-specific derivations in
// inventory.go do the composing). The slice is the single source of truth — the
// byte-exactness test (inventory_fixture_test.go) folds the same rows through
// buildInventory and asserts deep equality against the parsed fixture JSON, so a
// drift between loader and derivation fails the build rather than the eye.

// fixtureSpan is one open span the loader synthesizes: the subject it sits on, the
// facet timeline it values, the structured value the inventory derivations decode,
// whether it is a Gap, and the UTC date it opened (its Since). vantage_id stays
// NULL — the "vantage 1/3/prober" qualifiers ride the discriminator column, not a
// vantage row.
type fixtureSpan struct {
	kind          string
	key           string
	facet         string
	discriminator string
	value         string // the value JSONB, exactly as stored (curly quotes / arrow are ADDED by the derivation, never stored)
	isGap         bool
	since         string // YYYY-MM-DD; opened_at is this date at T00:00:00Z UTC
}

// inventoryFixtureSpans is the exact open-span corpus the pilot fixture pins. The
// unicode in the values and keys is load-bearing: middle dot · (U+00B7), ellipsis …
// (U+2026), apostrophe ’ (U+2019), em dash — (U+2014). The derivation adds the curly
// quotes “ ” (U+201C/U+201D) around an http-identity title and the arrow → (U+2192)
// before a redirect; those are NOT in the stored value.
var inventoryFixtureSpans = []fixtureSpan{
	// Names.
	{kind: "name", key: "www.acmecorp.io", facet: "resolution", value: `{"rrtype":"A","addresses":["198.51.100.7","198.51.100.8"]}`, since: "2026-07-14"},
	{kind: "name", key: "www.acmecorp.io", facet: "dns-record", value: `{"rrs":[{"type":"CNAME","data":"edge.acmecorp.io"},{"type":"TXT","data":"verge-custody=vg1:9f3k…"}]}`, since: "2026-07-14"},
	{kind: "name", key: "api.acmecorp.io", facet: "resolution", value: `{"rrtype":"A","addresses":["203.0.113.44"]}`, since: "2026-06-02"},
	{kind: "name", key: "api.acmecorp.io", facet: "dns-record", value: `{"rrs":[{"type":"CAA","data":"0 issue “letsencrypt.org”"}]}`, since: "2026-06-02"},
	{kind: "name", key: "mail.acmecorp.io", facet: "resolution", value: `{"rrtype":"A","addresses":["203.0.113.25"]}`, since: "2026-05-19"},
	{kind: "name", key: "mail.acmecorp.io", facet: "dns-record", value: `{}`, isGap: true, since: "2026-08-21"},

	// Services.
	{kind: "service", key: "198.51.100.7:443/tcp", facet: "tls-acceptance", value: `{"outcome":"enumerated","versions":["1.2","1.3"]}`, since: "2026-07-14"},
	{kind: "service", key: "198.51.100.7:443/tcp", facet: "certificate", value: `{"chain":[{"cn":"www.acmecorp.io","not_after":"2026-11-02"},{"cn":"R11","issuer_org":"Let’s Encrypt"}]}`, since: "2026-08-03"},
	{kind: "service", key: "203.0.113.44:22/tcp", facet: "reachability", value: `{"outcome":"answers","ports":["22/tcp"]}`, since: "2026-04-30"},
	{kind: "service", key: "203.0.113.44:22/tcp", facet: "tls-acceptance", value: `{"outcome":"none · plaintext ssh"}`, since: "2026-04-30"},
	{kind: "service", key: "198.51.100.31:8443/tcp", facet: "certificate", value: `{}`, isGap: true, since: "2026-08-19"},

	// Endpoints.
	{kind: "endpoint", key: "www.acmecorp.io · :443 https", facet: "http-identity", value: `{"server":"nginx","status":200,"title":"Acme — sign in"}`, since: "2026-07-14"},
	{kind: "endpoint", key: "grafana.acmecorp.io · :443 https", facet: "http-identity", value: `{"server":"Grafana","status":302,"redirect_location":"/login"}`, since: "2026-06-27"},

	// Addresses. vantage_id stays NULL — the vantage qualifier is the discriminator.
	{kind: "address", key: "198.51.100.7", facet: "reachability", discriminator: "vantage 1", value: `{"outcome":"answers","ports":["443/tcp"]}`, since: "2026-07-14"},
	{kind: "address", key: "198.51.100.7", facet: "reachability", discriminator: "vantage 3", value: `{"outcome":"answers","ports":["443/tcp","8443/tcp"]}`, since: "2026-08-02"},
	{kind: "address", key: "203.0.113.44", facet: "reachability", discriminator: "prober", value: `{"outcome":"answers","ports":["22/tcp"]}`, since: "2026-04-30"},
}

// fixtureSpanOpenedAt is the opened_at a fixture span carries: its Since date at
// midnight UTC. buildInventory formats OpenedAt back to the "YYYY-MM-DD" Since, so
// the round-trip is exact.
func (fs fixtureSpan) openedAt() (time.Time, error) {
	t, err := time.Parse("2006-01-02", fs.since)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse since %q: %w", fs.since, err)
	}
	return t, nil
}

// seedInventoryFixtures reseeds the dev database's open-span corpus with the exact
// spans the Inventory pilot fixture pins, idempotently: it DELETEs the whole span
// table (dev-only — the corpus is never deleted in production, ADR-0041) and
// re-inserts the 16 open spans, so re-running yields the same state. It is called
// only from the -seed-fixtures one-shot, which main.go gates on VERGE_DEV. The path
// is the fixture it reproduces; it is logged for provenance but the rows are pinned
// in code (the fixture is the contract the test asserts against, not a parse input).
func seedInventoryFixtures(ctx context.Context, pool *pgxpool.Pool, path string) error {
	// DELETE, then INSERT, in one transaction so a mid-run failure leaves the corpus
	// whole rather than half-seeded.
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful Commit

	if _, err := tx.Exec(ctx, `DELETE FROM span`); err != nil {
		return fmt.Errorf("reset span corpus: %w", err)
	}

	const insert = `
		INSERT INTO span
			(subject_kind, subject_key, facet, discriminator, vantage_id, source, value, is_gap, derivation, opened_at)
		VALUES ($1, $2, $3, $4, NULL, 'resolver', $5, $6, '[]'::jsonb, $7)`
	for _, fs := range inventoryFixtureSpans {
		openedAt, err := fs.openedAt()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, insert,
			fs.kind, fs.key, fs.facet, fs.discriminator, []byte(fs.value), fs.isGap, openedAt,
		); err != nil {
			return fmt.Errorf("insert %s/%s/%s: %w", fs.kind, fs.key, fs.facet, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// seedDevOperator makes the fixed dev operator exist so the pixel-parity harness can
// log in and reach the authenticated Inventory screen. It is idempotent: it creates
// the operator only when the instance has no account yet, so a re-seed against an
// instance that already has one (the operator, or a real first-run admin) leaves
// accounts untouched. Dev-only — the caller (main.go) has already gated on VERGE_DEV.
func seedDevOperator(ctx context.Context, pool *pgxpool.Pool) error {
	q := db.New(pool)
	n, err := q.CountAccounts(ctx)
	if err != nil {
		return fmt.Errorf("count accounts: %w", err)
	}
	if n > 0 {
		return nil
	}
	hash, err := auth.HashPassword(devSeedPassword)
	if err != nil {
		return fmt.Errorf("hash dev operator password: %w", err)
	}
	if _, err := q.CreateAccount(ctx, db.CreateAccountParams{
		Username: devSeedUsername, Role: roleAdmin, PasswordHash: hash,
	}); err != nil {
		return fmt.Errorf("create dev operator: %w", err)
	}
	return nil
}
