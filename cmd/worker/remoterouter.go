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

type remoteVantageStore interface {
	GetVantage(ctx context.Context, id int64) (db.Vantage, error)
}

type dialFunc func(ctx context.Context, t remoteexec.Target) (remoteexec.Conn, error)

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
		return wire.ProbeResult{}, false, nil
	}
	v, err := rt.store.GetVantage(ctx, vantageID.Int64)
	if err != nil {
		return wire.ProbeResult{}, false, fmt.Errorf("router: get vantage %d: %w", vantageID.Int64, err)
	}
	// A vantage with no prober endpoint is resolver-only, so its jobs run on the instance host.
	if !v.Host.Valid || v.Host.String == "" {
		return wire.ProbeResult{}, false, nil
	}
	// From here the router owns the outcome, so a failure drives the queue's retry path.
	if !v.HostKey.Valid || v.HostKey.String == "" {
		// Measurement never first-trusts a host: the startup connect pins, this path refuses.
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

	// The result rides the error return, so a failed job still carries its raw output (#867).
	res, err := remoteexec.Probe(ctx, conn, rt.binaries, spec)
	if err != nil {
		return res, true, fmt.Errorf("router: probe vantage %d off-host: %w", v.ID, err)
	}
	return res, true, nil
}
