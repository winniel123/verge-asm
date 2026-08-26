package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/ssh"

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

// vantageKeyPath is the worker-only path the private key for a vantage lives at.
// The private half never leaves this volume; only the public half is published
// to the database, and the same file is read back to dial the prober connect.
func vantageKeyPath(stateDir string, id int64) string {
	return filepath.Join(stateDir, "vantages", strconv.FormatInt(id, 10), "id_ed25519")
}

// ensureVantageKey returns the authorized_keys public line for a vantage,
// generating and writing a fresh private key to the worker volume when none
// exists, or re-deriving the public half from a private key already on disk.
// Re-deriving matters for crash recovery: if the worker wrote the private key
// but died before publishing the public half, the private key must be reused
// rather than regenerated, or the key installed on the prober host would no
// longer match.
func ensureVantageKey(stateDir string, id int64) (string, error) {
	keyPath := vantageKeyPath(stateDir, id)

	data, err := os.ReadFile(keyPath) // #nosec G304 (worker-only key path: constant segments + numeric int64 id via FormatInt, cannot traverse)
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

// dialTimeout bounds the prober connect the latency measurement rides on, so a
// silent host cannot stall the worker's startup sweep.
const dialTimeout = 10 * time.Second

// vantageLatencyStore is the slice of the database the worker's latency
// measurement uses. Narrowing it to an interface lets a test drive the loop with
// an in-memory fake, the same seam key provisioning uses.
type vantageLatencyStore interface {
	ListVantagesNeedingLatency(ctx context.Context) ([]db.Vantage, error)
	PinVantageHostKey(ctx context.Context, arg db.PinVantageHostKeyParams) error
	SetVantageLatency(ctx context.Context, arg db.SetVantageLatencyParams) error
}

// vantageProber measures the round-trip time of the prober connect that pins a
// vantage's host key. The real implementation dials SSH; a test supplies a fake,
// so the orchestration around it runs without a live SSH server — the same seam
// the key-provisioning loop uses. onFirstUse pins the presented host key
// trust-on-first-use; it fires only on the first connect and is a no-op once a
// key is already pinned.
type vantageProber interface {
	Connect(ctx context.Context, v db.Vantage, stateDir string, onFirstUse func(encoded string) error) (time.Duration, error)
}

// measureVantageLatencies records the connect round-trip time for every
// provisioned prober that has a published keypair but no latency yet (P0.5,
// SPEC-CHANGE.md collision #7). The connect it makes is the same one that pins
// the host key trust-on-first-use, so no separate dial is spent: it times
// establishing the SSH connection, pins the host key on first use, and persists
// the round-trip in whole milliseconds — the unit the Dashboard renders. Failure
// on one vantage is logged and skipped, never fatal and never a fabricated value:
// a vantage that could not be reached keeps its NULL latency and the Dashboard
// keeps rendering the pending em dash for it.
func measureVantageLatencies(ctx context.Context, store vantageLatencyStore, prober vantageProber, stateDir string) {
	rows, err := store.ListVantagesNeedingLatency(ctx)
	if err != nil {
		log.Printf("worker: list vantages needing latency: %v", err)
		return
	}
	for _, v := range rows {
		rtt, err := prober.Connect(ctx, v, stateDir, func(encoded string) error {
			return store.PinVantageHostKey(ctx, db.PinVantageHostKeyParams{
				ID: v.ID, HostKey: pgtype.Text{String: encoded, Valid: true},
			})
		})
		if err != nil {
			log.Printf("worker: vantage %d: measure connect latency: %v", v.ID, err)
			continue
		}
		ms := rtt.Milliseconds()
		if ms < 0 {
			ms = 0
		}
		if err := store.SetVantageLatency(ctx, db.SetVantageLatencyParams{
			ID: v.ID, LatencyMs: pgtype.Int4{Int32: int32(ms), Valid: true},
		}); err != nil {
			log.Printf("worker: vantage %d: persist latency: %v", v.ID, err)
			continue
		}
		log.Printf("worker: vantage %d: connect latency %dms recorded", v.ID, ms)
	}
}

// sshProber is the production vantageProber. It dials the prober endpoint over
// SSH with the vantage's private key on the worker volume — the same connect that
// pins the host key trust-on-first-use — and times how long establishing that
// connection takes. It opens the connection only to measure it: no measurement is
// dispatched over it here.
type sshProber struct{}

func (sshProber) Connect(_ context.Context, v db.Vantage, stateDir string, onFirstUse func(encoded string) error) (time.Duration, error) {
	keyData, err := os.ReadFile(vantageKeyPath(stateDir, v.ID))
	if err != nil {
		return 0, fmt.Errorf("read private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return 0, fmt.Errorf("parse private key: %w", err)
	}
	cfg := &ssh.ClientConfig{
		User:            v.Username.String,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: vantage.PinningHostKeyCallback(v.HostKey.String, onFirstUse),
		Timeout:         dialTimeout,
	}
	addr := net.JoinHostPort(v.Host.String, strconv.FormatInt(int64(v.Port.Int32), 10))

	start := time.Now()
	client, err := ssh.Dial("tcp", addr, cfg)
	rtt := time.Since(start)
	if err != nil {
		return 0, fmt.Errorf("dial %s: %w", addr, err)
	}
	_ = client.Close()
	return rtt, nil
}
