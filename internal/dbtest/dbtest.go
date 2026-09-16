// Package dbtest opens a real PostgreSQL for a test, so a rule that lives in
// SQL alone can be proved by running the statement rather than by matching its
// text (#2166).
package dbtest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	"github.com/winniel123/verge-asm/db/migrations"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/pgdb"
)

const URLEnv = "VERGE_TEST_DATABASE_URL"

const migrateLockKey int64 = 2166

const lockWait = 2 * time.Minute

var migrateOnce struct {
	sync.Once
	err error
}

func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	// Not DATABASE_URL: this package applies migrations and writes rows, and a
	// developer and every container already point that name at a live instance.
	url := os.Getenv(URLEnv)
	if url == "" {
		t.Skipf("%s is unset, so no database is reachable", URLEnv)
	}

	migrateOnce.Do(func() { migrateOnce.err = migrate(url) })
	if migrateOnce.err != nil {
		t.Fatalf("dbtest: %v", migrateOnce.err)
	}

	pool, err := pgdb.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("dbtest: connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func Tx(t *testing.T) pgx.Tx {
	t.Helper()

	tx, err := Pool(t).Begin(context.Background())
	if err != nil {
		t.Fatalf("dbtest: begin: %v", err)
	}
	// One database serves every case, so a case that committed would be read by
	// the next one.
	t.Cleanup(func() {
		if err := tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Errorf("dbtest: rollback: %v", err)
		}
	})
	return tx
}

func Queries(t *testing.T) (*db.Queries, pgx.Tx) {
	t.Helper()
	tx := Tx(t)
	// The generated queries, so a case exercises the shipped statement (#2166).
	return db.New(tx), tx
}

func migrate(url string) error {
	sqlDB, err := pgdb.OpenStdlib(url)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	unlock, err := lock(sqlDB)
	if err != nil {
		return err
	}
	defer unlock()

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(sqlDB, "."); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

func lock(sqlDB *sql.DB) (func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), lockWait)
	defer cancel()

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("take migration lock %d: %w", migrateLockKey, err)
	}
	// `go test ./...` runs package binaries concurrently, so two of them can
	// reach goose.Up against one database at the same time. The deadline turns a
	// peer that never releases into a named failure rather than a hang.
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", migrateLockKey); err != nil {
		conn.Close()
		return nil, fmt.Errorf("take migration lock %d: %w", migrateLockKey, err)
	}

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), lockWait)
		defer cancel()
		// Close returns the connection to the pool rather than ending the
		// session, so a session-level lock outlives it unless released here.
		_, _ = conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", migrateLockKey)
		conn.Close()
	}, nil
}
