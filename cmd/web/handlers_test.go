package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// fakeStore is an in-memory store used across the web handler tests, standing
// in for a live Postgres.
type fakeStore struct {
	hb    db.Heartbeat
	hbErr error

	accounts map[int64]db.Account
	byName   map[string]int64
	nextID   int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{accounts: map[int64]db.Account{}, byName: map[string]int64{}, nextID: 1}
}

func (f *fakeStore) RecordHeartbeat(context.Context) (db.Heartbeat, error) {
	return f.hb, f.hbErr
}

func (f *fakeStore) CountAccounts(context.Context) (int64, error) {
	return int64(len(f.accounts)), nil
}

func (f *fakeStore) CreateAccount(_ context.Context, arg db.CreateAccountParams) (db.Account, error) {
	if _, taken := f.byName[arg.Username]; taken {
		return db.Account{}, &pgconn.PgError{Code: "23505", Message: "duplicate key"}
	}
	acct := db.Account{
		ID: f.nextID, Username: arg.Username, Role: arg.Role, PasswordHash: arg.PasswordHash,
		CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
	}
	f.accounts[acct.ID] = acct
	f.byName[acct.Username] = acct.ID
	f.nextID++
	return acct, nil
}

func (f *fakeStore) GetAccountByUsername(_ context.Context, username string) (db.Account, error) {
	id, ok := f.byName[username]
	if !ok {
		return db.Account{}, pgx.ErrNoRows
	}
	return f.accounts[id], nil
}

func (f *fakeStore) GetAccountByID(_ context.Context, id int64) (db.Account, error) {
	acct, ok := f.accounts[id]
	if !ok {
		return db.Account{}, pgx.ErrNoRows
	}
	return acct, nil
}

func (f *fakeStore) SetTOTPSecret(_ context.Context, arg db.SetTOTPSecretParams) error {
	acct, ok := f.accounts[arg.ID]
	if !ok {
		return pgx.ErrNoRows
	}
	acct.TotpSecret = arg.TotpSecret
	acct.TotpEnabled = false
	f.accounts[arg.ID] = acct
	return nil
}

func (f *fakeStore) ConfirmTOTP(_ context.Context, id int64) error {
	acct, ok := f.accounts[id]
	if !ok || !acct.TotpSecret.Valid {
		return pgx.ErrNoRows
	}
	acct.TotpEnabled = true
	f.accounts[id] = acct
	return nil
}

// testKey is a fixed 32-byte session signing key for tests.
var testKey = []byte("0123456789abcdef0123456789abcdef")

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC) }
}

func TestHealthzOK(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	f := newFakeStore()
	f.hb = db.Heartbeat{ID: 1, CheckedAt: pgtype.Timestamptz{Time: now, Valid: true}}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newServer(f, testKey, "", fixedClock()).handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		Status    string    `json:"status"`
		CheckedAt time.Time `json:"checked_at"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ok" || !body.CheckedAt.Equal(now) {
		t.Fatalf("unexpected body: %+v", body)
	}
}

func TestHealthzDBError(t *testing.T) {
	f := newFakeStore()
	f.hbErr = errors.New("connection refused")

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	newServer(f, testKey, "", fixedClock()).handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}
