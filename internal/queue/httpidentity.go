package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
)

// The http-identity Scan (v1 spec §3.3/§3.4, ADR-0011/ADR-0024) is the daily HTTP
// exchange over every reached `Endpoint`. This file reads the reached `Service`
// population — the same current `reachability` spans reading `reached`, per vantage,
// that tls-acceptance reads (reachedServices, shared) — folds it into the per-vantage
// jobs, and enqueues them. There is NO port list read anywhere: the Services carry
// their own ports. It is additive to the hot, cold, zone, dns and tls-acceptance
// fan-outs — a new reader and enqueue path, no change to their dispatch.

// fanOutHTTPIdentity enqueues one http-identity job per Vantage over the Services
// reached from that Vantage. With no reached Service — the shipped state before any
// hot Scan has run — it produces no jobs, a legible empty scope rather than an error.
// This is the dispatch P0.11's first landing (child #686) omitted: it is what actually
// emits the `http-exchange` job that the prober case, the measurer, the drift fold and
// the four HTTP-identity rules were all already wired to consume.
func (d *Dispatcher) fanOutHTTPIdentity(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	// The reached-Service population is re-gated against the CURRENT Custody Estate —
	// the same one the connect-time hot dispatch reads — so an address whose
	// authorising scope/class was withdrawn since it was reached is not re-enumerated
	// (ADR-0079, #742). Only the Estate is needed here; the candidate address set the
	// hot Scan would probe is discarded.
	estate, _, err := hotEstate(ctx, qtx, d.now())
	if err != nil {
		return 0, err
	}
	services, err := reachedServices(ctx, qtx)
	if err != nil {
		return 0, err
	}
	vantages, err := vantageList(ctx, qtx)
	if err != nil {
		return 0, err
	}

	enqueued := 0
	for _, j := range scan.BuildHTTPIdentityJobs(scanID, estate, services, vantages.scanVantages()) {
		if err := enqueueHTTPIdentityJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

func enqueueHTTPIdentityJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.HTTPIdentityJob) error {
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
