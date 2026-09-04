package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/auth"
	"github.com/winniel123/verge-asm/internal/db"
)

const (
	devSeedUsername = "operator"
	devSeedPassword = "verge-dev-operator" // #nosec G101 -- not a real credential: a fixed dev-only fixture operator the loader seeds, and the loader is barred outside a VERGE_DEV build
)

type fixtureSpan struct {
	kind          string
	key           string
	facet         string
	discriminator string
	value         string
	isGap         bool
	since         string
}

// inventory_fixture_test.go folds this slice against fixtures.json, so a drift fails the build.

// The unicode is load-bearing; the derivation adds the curly quotes and arrow, never stored here.

var inventoryFixtureSpans = []fixtureSpan{
	{kind: "name", key: "www.acmecorp.io", facet: "resolution", value: `{"rrtype":"A","addresses":["198.51.100.7","198.51.100.8"]}`, since: "2026-07-14"},
	{kind: "name", key: "www.acmecorp.io", facet: "dns-record", value: `{"rrs":[{"type":"CNAME","data":"edge.acmecorp.io"},{"type":"TXT","data":"verge-custody=vg1:9f3k…"}]}`, since: "2026-07-14"},
	{kind: "name", key: "api.acmecorp.io", facet: "resolution", value: `{"rrtype":"A","addresses":["203.0.113.44"]}`, since: "2026-06-02"},
	{kind: "name", key: "api.acmecorp.io", facet: "dns-record", value: `{"rrs":[{"type":"CAA","data":"0 issue “letsencrypt.org”"}]}`, since: "2026-06-02"},
	{kind: "name", key: "mail.acmecorp.io", facet: "resolution", value: `{"rrtype":"A","addresses":["203.0.113.25"]}`, since: "2026-05-19"},
	{kind: "name", key: "mail.acmecorp.io", facet: "dns-record", value: `{}`, isGap: true, since: "2026-08-21"},

	{kind: "service", key: "198.51.100.7:443/tcp", facet: "tls-acceptance", value: `{"outcome":"enumerated","versions":["1.2","1.3"]}`, since: "2026-07-14"},
	{kind: "service", key: "198.51.100.7:443/tcp", facet: "certificate", value: `{"chain":[{"cn":"www.acmecorp.io","not_after":"2026-11-02"},{"cn":"R11","issuer_org":"Let’s Encrypt"}]}`, since: "2026-08-03"},
	{kind: "service", key: "203.0.113.44:22/tcp", facet: "reachability", value: `{"outcome":"answers","ports":["22/tcp"]}`, since: "2026-04-30"},
	{kind: "service", key: "203.0.113.44:22/tcp", facet: "tls-acceptance", value: `{"outcome":"none · plaintext ssh"}`, since: "2026-04-30"},
	{kind: "service", key: "198.51.100.31:8443/tcp", facet: "certificate", value: `{}`, isGap: true, since: "2026-08-19"},

	{kind: "endpoint", key: "www.acmecorp.io · :443 https", facet: "http-identity", value: `{"server":"nginx","status":200,"title":"Acme — sign in"}`, since: "2026-07-14"},
	{kind: "endpoint", key: "grafana.acmecorp.io · :443 https", facet: "http-identity", value: `{"server":"Grafana","status":302,"redirect_location":"/login"}`, since: "2026-06-27"},

	{kind: "address", key: "198.51.100.7", facet: "reachability", discriminator: "vantage 1", value: `{"outcome":"answers","ports":["443/tcp"]}`, since: "2026-07-14"},
	{kind: "address", key: "198.51.100.7", facet: "reachability", discriminator: "vantage 3", value: `{"outcome":"answers","ports":["443/tcp","8443/tcp"]}`, since: "2026-08-02"},
	{kind: "address", key: "203.0.113.44", facet: "reachability", discriminator: "prober", value: `{"outcome":"answers","ports":["22/tcp"]}`, since: "2026-04-30"},

	{kind: "address", key: "104.18.22.90", facet: "reachability", discriminator: "vantage 1", value: `{"outcome":"gap","cause":"blanket-responder","reason":"this address answers on all ports — it is a proxy edge, not your origin"}`, isGap: true, since: "2026-08-19"},
}

func (fs fixtureSpan) openedAt() (time.Time, error) {
	t, err := time.Parse("2006-01-02", fs.since)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse since %q: %w", fs.since, err)
	}
	return t, nil
}

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
