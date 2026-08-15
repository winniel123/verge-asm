package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/vantage"
)

// vantageKeyStore is the slice of the database the worker's key provisioning
// uses. Narrowing it to an interface lets a test drive the loop with an
// in-memory fake, the same seam the web handlers use.
type vantageKeyStore interface {
	ListVantagesNeedingKey(ctx context.Context) ([]db.Vantage, error)
	SetVantagePublicKey(ctx context.Context, arg db.SetVantagePublicKeyParams) error
}

// provisionVantageKeys generates the SSH keypair for every provisioned vantage
// that has none yet: the private half is written to the worker-only state
// volume and never leaves it, and only the public half is published back to the
// database for the web surface to render. Failure on one vantage is logged and
// skipped, not fatal — the worker has no dispatch work yet for this to gate.
//
// This is the whole of the prober keypair story in this ticket: no measurement
// is dispatched over the connection, and the host key is pinned on the first
// real connect a later ticket wires (internal/vantage.PinningHostKeyCallback).
func provisionVantageKeys(ctx context.Context, store vantageKeyStore, stateDir string) {
	rows, err := store.ListVantagesNeedingKey(ctx)
	if err != nil {
		log.Printf("worker: list vantages needing key: %v", err)
		return
	}
	for _, v := range rows {
		pub, err := ensureVantageKey(stateDir, v.ID)
		if err != nil {
			log.Printf("worker: vantage %d: ensure key: %v", v.ID, err)
			continue
		}
		if err := store.SetVantagePublicKey(ctx, db.SetVantagePublicKeyParams{
			ID: v.ID, PublicKey: pgtype.Text{String: pub, Valid: true},
		}); err != nil {
			log.Printf("worker: vantage %d: publish public key: %v", v.ID, err)
			continue
		}
		log.Printf("worker: vantage %d: keypair provisioned, public key published", v.ID)
	}
}

// ensureVantageKey returns the authorized_keys public line for a vantage,
// generating and writing a fresh private key to the worker volume when none
// exists, or re-deriving the public half from a private key already on disk.
// Re-deriving matters for crash recovery: if the worker wrote the private key
// but died before publishing the public half, the private key must be reused
// rather than regenerated, or the key installed on the prober host would no
// longer match.
func ensureVantageKey(stateDir string, id int64) (string, error) {
	keyPath := filepath.Join(stateDir, "vantages", strconv.FormatInt(id, 10), "id_ed25519")

	data, err := os.ReadFile(keyPath)
	switch {
	case err == nil:
		return vantage.PublicKeyFromPrivatePEM(data)
	case !errors.Is(err, fs.ErrNotExist):
		return "", err
	}

	kp, err := vantage.Generate()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(keyPath, kp.PrivatePEM, 0o600); err != nil {
		return "", err
	}
	return kp.PublicKey, nil
}
