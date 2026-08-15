package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

// The tls-acceptance Scan (v1 spec §3.4, ADR-0028) is the weekly enumeration over
// every open `Service`. This file reads that open `Service` population — the current
// `reachability` spans reading `reached`, per vantage — folds it into the
// per-vantage jobs, and enqueues them. There is NO port list read anywhere: the
// Services carry their own ports, so the aperture is the open-Service set and the
// declared candidate set, never a port tier (ADR-0028). It is additive to the hot,
// cold, zone and dns fan-outs — a new reader and enqueue path, no change to their
// dispatch.

// fanOutTLSAcceptance enqueues one tls-acceptance job per Vantage over the Services
// reached from that Vantage. With no reached Service — the shipped state before any
// hot Scan has run — it produces no jobs, a legible empty scope rather than an error.
func (d *Dispatcher) fanOutTLSAcceptance(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	services, err := reachedServices(ctx, qtx)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, qtx)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, j := range scan.BuildTLSAcceptanceJobs(scanID, services, vantages.scanVantages()) {
		if err := enqueueTLSAcceptanceJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

// reachedServices reads the open `Service` population from the current reachability
// spans and parses each `address:port/tcp` subject key back to an address and port.
// A span with no vantage row, or a key that does not parse, is skipped — the
// enumeration never fabricates a target it cannot name.
func reachedServices(ctx context.Context, q *db.Queries) ([]scan.ReachedService, error) {
	rows, err := q.ListReachedServices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]scan.ReachedService, 0, len(rows))
	for _, r := range rows {
		if !r.VantageID.Valid {
			continue // a Service with no vantage row — the enumeration is per-vantage
		}
		ap, ok := parseServiceKey(r.ServiceKey)
		if !ok {
			continue
		}
		out = append(out, scan.ReachedService{
			VantageID: r.VantageID.Int64,
			Address:   ap.Addr().String(),
			Port:      ap.Port(),
		})
	}
	return out, nil
}

// parseServiceKey folds a `address:port/tcp` Service subject key back to its
// `(Address, port)`. It is the inverse of the ServiceKey the connect-outcome leaf
// renders. A key without the `/tcp` transport suffix, or one whose address:port does
// not parse, is rejected rather than guessed at.
func parseServiceKey(key string) (netip.AddrPort, bool) {
	base, ok := strings.CutSuffix(key, "/tcp")
	if !ok {
		return netip.AddrPort{}, false
	}
	ap, err := netip.ParseAddrPort(base)
	if err != nil {
		return netip.AddrPort{}, false
	}
	return ap, true
}

// enqueueTLSAcceptanceJob enqueues one tls-acceptance job for one Vantage. Its
// recorded scope carries the open Services and the candidate set by content; its
// offers carry the candidate set. It retries like a hot job — an enumeration is a
// network step that can transiently fail.
func enqueueTLSAcceptanceJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.TLSAcceptanceJob) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:vantage:%d", scanID, j.VantageID))
	if err != nil {
		return err
	}
	specJSON, err := json.Marshal(spec)
	if err != nil {
		return err
	}
	scopeJSON, err := j.AttemptedScope()
	if err != nil {
		return err
	}
	offersJSON, err := j.OffersJSON()
	if err != nil {
		return err
	}
	_, err = qtx.EnqueueJob(ctx, db.EnqueueJobParams{
		ScanID:         scanID,
		VantageID:      pgInt8(j.VantageID),
		DispatchID:     pgInt8(dispatchID),
		Kind:           j.Kind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         offersJSON,
		Attempt:        1,
		MaxAttempts:    5,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}
