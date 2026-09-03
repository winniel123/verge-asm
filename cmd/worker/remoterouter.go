package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/remoteexec"
	"github.com/winniel123/verge-asm/internal/vantage"
	"github.com/winniel123/verge-asm/internal/wire"
)

func endpointAddr(v db.Vantage) string {
	return net.JoinHostPort(v.Host.String, strconv.FormatInt(int64(v.Port.Int32), 10))
}

// remoteVantageStore is the slice of the database the off-host router reads: the one
// vantage the claimed job belongs to. Narrowed to an interface so a test drives the
// router with an in-memory fake, no live database.
type remoteVantageStore interface {
	GetVantage(ctx context.Context, id int64) (db.Vantage, error)
}

// dialFunc establishes a Conn to a prober. It is the production remoteexec.Dial in the
// wiring and an in-memory fake in tests, so the router's decision logic — off-host vs
// local, pinned-or-refuse — runs with no live SSH server.
type dialFunc func(ctx context.Context, t remoteexec.Target) (remoteexec.Conn, error)

// remoteProberRouter implements queue.VantageRouter: it decides whether a job runs
// off-host and, when it does, pushes the measurement binary to the prober and exec's
// it there (ADR-0103, #683). A provisioned prober (host set) measures from its OWN
// position; the resolver-only `local` vantage — and any job with no vantage — falls
// through to the worker's local ExecProber. The private key is read from the same
// worker-only volume the latency sweep uses, and the host key MUST already be pinned:
// measurement never trusts an unpinned host, so an un-pinned prober is refused (a
// transient error) rather than silently first-trusted on the measurement path.
type remoteProberRouter struct {
	store    remoteVantageStore
	binaries remoteexec.BinaryProvider
	stateDir string
	dial     dialFunc
	log      *log.Logger
}

func newRemoteProberRouter(store remoteVantageStore, binaries remoteexec.BinaryProvider, stateDir string, logger *log.Logger) *remoteProberRouter {
	return &remoteProberRouter{store: store, binaries: binaries, stateDir: stateDir, dial: remoteexec.Dial, log: logger}
}

func (rt *remoteProberRouter) ProbeVantage(ctx context.Context, vantageID pgtype.Int8, spec wire.JobSpec) (wire.ProbeResult, bool, error) {
	if !vantageID.Valid {
		return wire.ProbeResult{}, false, nil // no vantage (e.g. a worker-read kind) — local
	}
	v, err := rt.store.GetVantage(ctx, vantageID.Int64)
	if err != nil {
		return wire.ProbeResult{}, false, fmt.Errorf("router: get vantage %d: %w", vantageID.Int64, err)
	}
	// A resolver-only vantage has no prober endpoint: its jobs run on the instance host
	// via the local prober, exactly as before this router existed.
	if !v.Host.Valid || v.Host.String == "" {
		return wire.ProbeResult{}, false, nil
	}
	// This is a provisioned prober, so its measurement belongs off-host. From here the
	// router owns the outcome (handled=true); a failure is a transient measurement error
	// that drives the same retry/dead-letter path a local probe error does.
	if !v.HostKey.Valid || v.HostKey.String == "" {
		// The host key is pinned on the worker's startup connect. Until then, refuse
		// rather than silently first-trust a host on the measurement path.
		return wire.ProbeResult{}, true, fmt.Errorf("router: vantage %d host key not pinned yet", v.ID)
	}
	keyData, err := os.ReadFile(vantageKeyPath(rt.stateDir, v.ID))
	if err != nil {
		return wire.ProbeResult{}, true, fmt.Errorf("router: read private key for vantage %d: %w", v.ID, err)
	}
	addr := endpointAddr(v)
	conn, err := rt.dial(ctx, remoteexec.Target{
		Addr:            addr,
		Username:        v.Username.String,
		PrivateKey:      keyData,
		HostKeyCallback: vantage.PinningHostKeyCallback(v.HostKey.String, nil),
		Timeout:         dialTimeout,
	})
	if err != nil {
		return wire.ProbeResult{}, true, fmt.Errorf("router: dial vantage %d (%s): %w", v.ID, addr, err)
	}
	defer conn.Close()

	// Probe carries the verbatim off-host exchange back in the ProbeResult's Transcript
	// on every outcome (#867). The result rides the error return too, so a failed or
	// decode-failed off-host job keeps its raw output onto the retry/dead-letter tx —
	// exactly when it is most wanted.
	res, err := remoteexec.Probe(ctx, conn, rt.binaries, spec)
	if err != nil {
		return res, true, fmt.Errorf("router: probe vantage %d off-host: %w", v.ID, err)
	}
	return res, true, nil
}
