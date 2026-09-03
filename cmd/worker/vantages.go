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

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/remoteexec"
	"github.com/winniel123/verge-asm/internal/vantage"
)

type vantageKeyStore interface {
	ListVantagesNeedingKey(ctx context.Context) ([]db.Vantage, error)
	SetVantagePublicKey(ctx context.Context, arg db.SetVantagePublicKeyParams) error
}

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
		// The private half never leaves this worker-only volume; only the public half is published.
		if err := store.SetVantagePublicKey(ctx, db.SetVantagePublicKeyParams{
			ID: v.ID, PublicKey: pgtype.Text{String: pub, Valid: true},
		}); err != nil {
			log.Printf("worker: vantage %d: publish public key: %v", v.ID, err)
			continue
		}
		log.Printf("worker: vantage %d: keypair provisioned, public key published", v.ID)
	}
}

func vantageKeyPath(stateDir string, id int64) string {
	return filepath.Join(stateDir, "vantages", strconv.FormatInt(id, 10), "id_ed25519")
}

func ensureVantageKey(stateDir string, id int64) (string, error) {
	keyPath := vantageKeyPath(stateDir, id)

	// Re-deriving matters after a crash: a regenerated key would not match the prober host.
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

// A silent host must not stall the worker's startup sweep or the measurement router.

const dialTimeout = 10 * time.Second

type vantageLatencyStore interface {
	ListVantagesNeedingLatency(ctx context.Context) ([]db.Vantage, error)
	PinVantageHostKey(ctx context.Context, arg db.PinVantageHostKeyParams) error
	SetVantageLatency(ctx context.Context, arg db.SetVantageLatencyParams) error
	SetVantageProbeFacts(ctx context.Context, arg db.SetVantageProbeFactsParams) error
}

type vantageProbe struct {
	rtt   time.Duration
	facts remoteexec.Facts
}

type vantageProber interface {
	Connect(ctx context.Context, v db.Vantage, stateDir string, onFirstUse func(encoded string) error) (vantageProbe, error)
}

func measureVantageLatencies(ctx context.Context, store vantageLatencyStore, prober vantageProber, stateDir string) {
	rows, err := store.ListVantagesNeedingLatency(ctx)
	if err != nil {
		log.Printf("worker: list vantages needing latency: %v", err)
		return
	}
	for _, v := range rows {
		probe, err := prober.Connect(ctx, v, stateDir, func(encoded string) error {
			return store.PinVantageHostKey(ctx, db.PinVantageHostKeyParams{
				ID: v.ID, HostKey: pgtype.Text{String: encoded, Valid: true},
			})
		})
		if err != nil {
			log.Printf("worker: vantage %d: measure connect latency: %v", v.ID, err)
			continue
		}
		// Whole milliseconds is the unit the Dashboard renders, so nothing finer is stored.
		ms := probe.rtt.Milliseconds()
		if ms < 0 {
			ms = 0
		}
		if err := store.SetVantageLatency(ctx, db.SetVantageLatencyParams{
			ID: v.ID, LatencyMs: pgtype.Int4{Int32: int32(ms), Valid: true},
		}); err != nil {
			log.Printf("worker: vantage %d: persist latency: %v", v.ID, err)
			continue
		}
		// A fact that was not observed stays NULL, so its chip collapses rather than fabricating.
		if err := store.SetVantageProbeFacts(ctx, db.SetVantageProbeFactsParams{
			ID:          v.ID,
			Platform:    pgtype.Text{String: probe.facts.Platform.Label, Valid: probe.facts.Platform.Label != ""},
			Egress:      pgtype.Text{String: probe.facts.Egress, Valid: probe.facts.HasEgress},
			DialledAddr: pgtype.Text{String: probe.facts.Dialled, Valid: probe.facts.HasDialled},
		}); err != nil {
			log.Printf("worker: vantage %d: persist probe facts: %v", v.ID, err)
			continue
		}
		log.Printf("worker: vantage %d: connect latency %dms recorded, platform=%q egress=%q dialled=%q",
			v.ID, ms, probe.facts.Platform.Label, probe.facts.Egress, probe.facts.Dialled)
	}
}

type sshProber struct{}

func (sshProber) Connect(ctx context.Context, v db.Vantage, stateDir string, onFirstUse func(encoded string) error) (vantageProbe, error) {
	keyData, err := os.ReadFile(vantageKeyPath(stateDir, v.ID))
	if err != nil {
		return vantageProbe{}, fmt.Errorf("read private key: %w", err)
	}
	addr := net.JoinHostPort(v.Host.String, strconv.FormatInt(int64(v.Port.Int32), 10))

	start := time.Now()
	conn, err := remoteexec.Dial(ctx, remoteexec.Target{
		Addr:            addr,
		Username:        v.Username.String,
		PrivateKey:      keyData,
		HostKeyCallback: vantage.PinningHostKeyCallback(v.HostKey.String, onFirstUse),
		Timeout:         dialTimeout,
	})
	rtt := time.Since(start)
	if err != nil {
		return vantageProbe{}, fmt.Errorf("dial %s: %w", addr, err)
	}
	defer conn.Close()

	// A failed facts read does not fail the connect: the latency is useful on its own.
	facts, err := remoteexec.Inspect(ctx, conn)
	if err != nil {
		log.Printf("worker: vantage %d: inspect facts: %v", v.ID, err)
		return vantageProbe{rtt: rtt}, nil
	}
	return vantageProbe{rtt: rtt, facts: facts}, nil
}
