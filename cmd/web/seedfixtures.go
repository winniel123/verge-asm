package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/queue"
)

const (
	devSeedUsername = "operator"
	devSeedPassword = "verge-dev-operator" // #nosec G101 -- not a real credential: a fixed dev-only fixture operator the loader seeds, and the loader is barred outside a VERGE_DEV build
)

// Containment of a presented address in a declared scope is all that derives a class (#1985).

// A declared scope is a dispatcher input, so this one is TEST-NET-1 rather than a real LAN.

const (
	fixtureVantageInternal = "fixture-internal"
	fixtureVantageInternet = "fixture-internet"
	fixtureAddressScope    = "192.0.2.0/24"
)

type fixtureVantage struct {
	name    string
	dialled string
}

// class ships vestigial on the row and is derived per read from dialled_addr (#709).

var inventoryFixtureVantages = []fixtureVantage{
	{name: fixtureVantageInternal, dialled: "192.0.2.5"},
	{name: fixtureVantageInternet, dialled: "203.0.113.9"},
}

type fixtureSpan struct {
	kind          string
	key           string
	facet         string
	discriminator string
	vantage       string
	value         string
	isGap         bool
	since         string
}

// inventory_fixture_test.go folds this slice against fixtures.json, so a drift fails the build.

// The unicode is load-bearing; the derivation adds the curly quotes and arrow, never stored here.

// A row carries the vantage its facet implies and the source the fold writes (ADR-1985 §2).

var inventoryFixtureSpans = []fixtureSpan{
	{kind: "name", key: "www.acmecorp.io", facet: "resolution", vantage: fixtureVantageInternal, value: `{"rrtype":"A","addresses":["198.51.100.7","198.51.100.8"]}`, since: "2026-07-14"},
	{kind: "name", key: "www.acmecorp.io", facet: "dns-record", discriminator: "TXT", vantage: fixtureVantageInternal, value: `{"rrs":[{"type":"CNAME","data":"edge.acmecorp.io"},{"type":"TXT","data":"v=spf1 -all"}]}`, since: "2026-07-14"},
	{kind: "name", key: "api.acmecorp.io", facet: "resolution", vantage: fixtureVantageInternal, value: `{"rrtype":"A","addresses":["203.0.113.44"]}`, since: "2026-06-02"},
	{kind: "name", key: "api.acmecorp.io", facet: "dns-record", discriminator: "TXT", vantage: fixtureVantageInternal, value: `{"rrs":[{"type":"TXT","data":"v=spf1 -all"}]}`, since: "2026-06-02"},
	{kind: "name", key: "mail.acmecorp.io", facet: "resolution", vantage: fixtureVantageInternal, value: `{"rrtype":"A","addresses":["203.0.113.25"]}`, since: "2026-05-19"},
	{kind: "name", key: "mail.acmecorp.io", facet: "dns-record", discriminator: "MX", vantage: fixtureVantageInternal, value: `{}`, isGap: true, since: "2026-08-21"},

	{kind: "service", key: "198.51.100.7:443/tcp", facet: "tls-acceptance", vantage: fixtureVantageInternet, value: `{"outcome":"enumerated","versions":["1.2","1.3"]}`, since: "2026-07-14"},
	{kind: "service", key: "198.51.100.7:443/tcp", facet: "certificate", vantage: fixtureVantageInternet, value: `{"chain":[{"cn":"www.acmecorp.io","not_after":"2026-11-02"},{"cn":"R11","issuer_org":"Let’s Encrypt"}]}`, since: "2026-08-03"},
	// One port opens one span per vantage, so one row leaves a leg absent (#1962, #1985).
	{kind: "service", key: "203.0.113.44:22/tcp", facet: "reachability", vantage: fixtureVantageInternal, value: `{"outcome":"reached"}`, since: "2026-04-30"},
	{kind: "service", key: "203.0.113.44:22/tcp", facet: "reachability", vantage: fixtureVantageInternet, value: `{"outcome":"not-reached"}`, since: "2026-04-30"},
	{kind: "service", key: "203.0.113.44:22/tcp", facet: "tls-acceptance", vantage: fixtureVantageInternet, value: `{"outcome":"none · plaintext ssh"}`, since: "2026-04-30"},
	{kind: "service", key: "198.51.100.31:8443/tcp", facet: "certificate", vantage: fixtureVantageInternet, value: `{}`, isGap: true, since: "2026-08-19"},

	{kind: "endpoint", key: "www.acmecorp.io · :443 https", facet: "http-identity", vantage: fixtureVantageInternet, value: `{"server":"nginx","status":200,"title":"Acme — sign in"}`, since: "2026-07-14"},
	{kind: "endpoint", key: "grafana.acmecorp.io · :443 https", facet: "http-identity", vantage: fixtureVantageInternet, value: `{"server":"Grafana","status":302,"redirect_location":"/login"}`, since: "2026-06-27"},
}

func (fs fixtureSpan) openedAt() (time.Time, error) {
	t, err := time.Parse("2006-01-02", fs.since)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse since %q: %w", fs.since, err)
	}
	return t, nil
}

func (fs fixtureSpan) source() string { return queue.SourceFor(fs.facet) }

func seedInventoryFixtures(ctx context.Context, pool *pgxpool.Pool, path string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after a successful Commit

	// Dev-only: production never deletes the span corpus (ADR-0041).
	if _, err := tx.Exec(ctx, `DELETE FROM span`); err != nil {
		return fmt.Errorf("reset span corpus: %w", err)
	}

	const upsertVantage = `
		INSERT INTO vantage (name, class, resolver, dialled_addr)
		VALUES ($1, 'unverified', '127.0.0.11:53', $2)
		ON CONFLICT (name) DO UPDATE SET dialled_addr = EXCLUDED.dialled_addr
		RETURNING id`
	vantages := make(map[string]int64, len(inventoryFixtureVantages))
	for _, fv := range inventoryFixtureVantages {
		var id int64
		// class is vestigial and the resolver is docker's embedded DNS (#709, 18800).
		if err := tx.QueryRow(ctx, upsertVantage, fv.name, fv.dialled).Scan(&id); err != nil {
			return fmt.Errorf("seed vantage %s: %w", fv.name, err)
		}
		vantages[fv.name] = id
	}

	if err := seedFixtureAddressScope(ctx, tx); err != nil {
		return err
	}

	const insert = `
		INSERT INTO span
			(subject_kind, subject_key, facet, discriminator, vantage_id, source, value, is_gap, derivation, opened_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '[]'::jsonb, $9)`
	for _, fs := range inventoryFixtureSpans {
		openedAt, err := fs.openedAt()
		if err != nil {
			return err
		}
		vantageID, ok := vantages[fs.vantage]
		if !ok {
			return fmt.Errorf("fixture span %s/%s/%s names unknown vantage %q", fs.kind, fs.key, fs.facet, fs.vantage)
		}
		if _, err := tx.Exec(ctx, insert,
			fs.kind, fs.key, fs.facet, fs.discriminator, vantageID, fs.source(),
			[]byte(fs.value), fs.isGap, openedAt,
		); err != nil {
			return fmt.Errorf("insert %s/%s/%s: %w", fs.kind, fs.key, fs.facet, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func seedFixtureAddressScope(ctx context.Context, tx pgx.Tx) error {
	var accountID int64
	// seedDevOperator runs first, so the estate always holds at least one account.
	if err := tx.QueryRow(ctx, `SELECT id FROM account ORDER BY id LIMIT 1`).Scan(&accountID); err != nil {
		return fmt.Errorf("read seeding account: %w", err)
	}
	const insert = `
		INSERT INTO seed (kind, address_cidr, created_by)
		VALUES ('address', $1::cidr, $2)
		ON CONFLICT (address_cidr) DO NOTHING`
	if _, err := tx.Exec(ctx, insert, fixtureAddressScope, accountID); err != nil {
		return fmt.Errorf("seed address scope: %w", err)
	}
	return nil
}

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
