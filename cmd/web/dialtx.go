package main

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/db"
)

// Outside one locked section two submits of a dial move repeat a row or lose one (ADR-1914).

type dialStore interface {
	InDialTx(ctx context.Context, fn func(dialQueries, recorder) error) error
}

// Every statement the critical section may reach, bound to the transaction holding the
// lock. A read taken outside this set is not serialised against the move.

type dialQueries interface {
	LockRetentionSettings(ctx context.Context) (db.LockRetentionSettingsRow, error)
	UpdateCoverageRetentionSettings(
		ctx context.Context, arg db.UpdateCoverageRetentionSettingsParams,
	) (db.UpdateCoverageRetentionSettingsRow, error)
	UpdateRetentionSettings(ctx context.Context, arg db.UpdateRetentionSettingsParams) error

	LockInstanceConfig(ctx context.Context) (db.LockInstanceConfigRow, error)
	SetSeedAddressCap(ctx context.Context, arg db.SetSeedAddressCapParams) error
	SetUpdateCheckEnabled(ctx context.Context, arg db.SetUpdateCheckEnabledParams) error
	SetAPIEnabled(ctx context.Context, arg db.SetAPIEnabledParams) error

	LockZoneCadenceSeconds(ctx context.Context) (int64, error)
	SetZoneCadenceSeconds(ctx context.Context, cadenceSeconds int64) error

	LockDnsCadenceSeconds(ctx context.Context) (int64, error)
	SetDnsCadenceSeconds(ctx context.Context, cadenceSeconds int64) error
}

type pgStore struct {
	*db.Queries
	pool *pgxpool.Pool
}

// The pool arrives as a constructor argument, not a later assignment, so no deployment
// reaches a dial handler with the guard unwired.

func newPgStore(q *db.Queries, pool *pgxpool.Pool) pgStore {
	return pgStore{Queries: q, pool: pool}
}

func (p pgStore) InDialTx(ctx context.Context, fn func(dialQueries, recorder) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The Act commits with the move it records, so a rollback leaves neither behind.
	if err := fn(db.New(tx), txRecorder(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *server) moveDial(
	w http.ResponseWriter, r *http.Request, what string, fn func(dialQueries, recorder) error,
) bool {
	if err := s.dialStore.InDialTx(r.Context(), fn); err != nil {
		s.serverError(w, what, err)
		return false
	}
	return true
}
