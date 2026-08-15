// Package pgdb holds the connection setup shared by web and worker: a pgx
// pool for runtime queries, plus a stdlib *sql.DB handle for goose (which
// speaks database/sql, not pgx's native interface).
package pgdb

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Connect opens a pgx pool against databaseURL, pinging once to fail fast
// on a bad DSN or an unreachable database rather than on the first query.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgdb: open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgdb: ping: %w", err)
	}
	return pool, nil
}

// OpenStdlib opens a database/sql handle over the same driver, for tools
// (goose) that require the standard interface rather than pgx's native one.
// Callers must Close it.
func OpenStdlib(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgdb: open stdlib handle: %w", err)
	}
	return db, nil
}
