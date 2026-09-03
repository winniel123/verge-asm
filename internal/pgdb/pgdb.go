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

func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgdb: open pool: %w", err)
	}
	// pgxpool.New is lazy, so a bad DSN or an unreachable database surfaces on the first query.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgdb: ping: %w", err)
	}
	return pool, nil
}

func OpenStdlib(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("pgdb: open stdlib handle: %w", err)
	}
	return db, nil
}
