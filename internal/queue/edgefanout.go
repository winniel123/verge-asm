package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/measure/edgefanout"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

// The `edge-fanout` Scan (CONTEXT.md `Scan`, ADR-0129 §6, ticket #983) is the daily
// no-SNI TLS handshake over the custody-extension candidates. This file holds its
// dispatch half — the candidate read and the enqueue — and its recording half, which
// lands one row per measured address in the leaf's own store.
//
// The measurement decides MEMBERSHIP and opens no timeline, so it holds no row in the
// `observation` table: there is no facet, subject or discriminator for that table's
// four-part key to name. toObservationParams already skips a facet-less line, so the
// recording half here is additive to the shared completion path and changes no existing
// fold.

// fanOutEdgeFanout enqueues the edge-fanout jobs for one tick, over the custody-
// extension candidates alone. Those are the direct-A targets — and the apex
// `ALIAS`/`ANAME` flattened to A — of in-zone names the extension would reach.
//
// An instance with no custody extension has an empty candidate set and enqueues no job.
// That is a legible state, not an error: the Dispatch is recorded over an empty scope
// and nothing is probed.
//
// The candidates are read through the same current Custody Estate the hot dispatch
// reads (hotEstate), so a name whose authorising scope was withdrawn since it resolved
// contributes no candidate (ADR-0079, #742). There is no vantage fan-out: a default
// certificate is not a function of vantage, and vantage-varying fan-out is anycast, out
// of v1 (ADR-0129 §5).
func (d *Dispatcher) fanOutEdgeFanout(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	estate, _, err := hotEstate(ctx, qtx, d.now())
	if err != nil {
		return 0, err
	}
	enqueued := 0
	for i, j := range scan.BuildEdgeFanoutJobs(scanID, estate.ExtensionCandidates()) {
		if err := enqueueEdgeFanoutJob(ctx, qtx, scanID, dispatchID, i, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

// enqueueEdgeFanoutJob enqueues one edge-fanout job. It carries NO Vantage — the Scan
// has no vantage dimension — and its Batch label names the chunk of candidates it
// measures, since there is no vantage or seed to key it on. It retries like a hot job:
// a handshake is a network step that can transiently fail.
func enqueueEdgeFanoutJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, chunk int, j scan.EdgeFanoutJob) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:edges:%d", scanID, chunk))
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
		VantageID:      pgtype.Int8{}, // the edge-fanout Scan has no vantage dimension
		DispatchID:     pgInt8(dispatchID),
		Kind:           scan.EdgeFanoutKind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         offersJSON,
		Attempt:        1,
		MaxAttempts:    5,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}

// toEdgeFanoutRows maps an edge-fanout batch's lines to their store rows: one row per
// measured address, carrying the outcome, the served certificate's fingerprint on
// `presented` alone, and the batch instant.
//
// jobKind is the kind the DISPATCHER enqueued, and it is what admits this fold at all.
// A row is written only for a job the dispatcher sent to this leaf — never on the
// strength of a line's self-declared Kind, which is the prober's word and is exactly
// what the recording-side re-gate distrusts (#773). Without that test, a prober handling
// any other job could append one `edge-fanout` line and mint a measurement of an address
// its job never authorised: a dns job's scope denotes names alone, so the address
// dimension has nothing to gate it against, and the veto (#985) would then read an
// answer nothing measured.
//
// It is the pure half of the recording, so the mapping is tested without a database. A
// line this leaf could not have emitted — one carrying a facet (this leaf's lines carry
// none), an outcome outside the closed union, a `presented` with no fingerprint, a
// negative carrying one, a fingerprint disagreeing with the certificate material beside
// it, an address that does not parse — is DROPPED and named in the second return, never
// persisted and never guessed at.
//
// A repeated address in one batch keeps its first row alone. The leaf already
// handshakes each distinct address once, so a repeat is a line the batch had no
// measurement for.
func toEdgeFanoutRows(jobKind string, batchID int64, measuredAt pgtype.Timestamptz, obs []wire.Observation) ([]db.InsertEdgeFanoutObservationParams, []string) {
	if jobKind != scan.EdgeFanoutKind {
		return nil, nil
	}
	var out []db.InsertEdgeFanoutObservationParams
	var dropped []string
	seen := map[netip.Addr]struct{}{}
	for _, o := range obs {
		if o.Kind != edgefanout.Kind {
			continue
		}
		// This leaf's lines carry NO facet — it decides membership and opens no
		// timeline. A line claiming one was gated on its subject rather than on the
		// Address read here (scopegate.go), so its address was never authorised.
		if o.Facet != "" {
			dropped = append(dropped, fmt.Sprintf("address %q carries facet %q", o.Address, o.Facet))
			continue
		}
		addr, err := netip.ParseAddr(normAddr(o.Address))
		if err != nil {
			dropped = append(dropped, fmt.Sprintf("address %q does not parse", o.Address))
			continue
		}
		addr = addr.Unmap()
		v, err := edgefanout.DecodeValue(o.Data)
		if err != nil {
			dropped = append(dropped, fmt.Sprintf("address %s: %v", addr, err))
			continue
		}
		// Where the line carries certificate material, the fingerprint this row stores
		// and the key that material lands under must be the same value — the material's
		// key is recomputed from its own DER (edgefanout.presentedMaterial), so a
		// disagreement means the row would name a certificate the side store does not
		// hold under that key, and #984's SAN read would join to nothing. A presented
		// handshake that carried a chain but no DER carries no material and is not
		// tested here.
		if o.CertMaterial != nil && o.CertMaterial.Fingerprint != v.Fingerprint {
			dropped = append(dropped, fmt.Sprintf("address %s: fingerprint %q disagrees with its certificate material %q",
				addr, v.Fingerprint, o.CertMaterial.Fingerprint))
			continue
		}
		if _, dup := seen[addr]; dup {
			dropped = append(dropped, fmt.Sprintf("address %s measured twice in one batch", addr))
			continue
		}
		seen[addr] = struct{}{}
		out = append(out, db.InsertEdgeFanoutObservationParams{
			BatchID:     batchID,
			Address:     addr.String(),
			Outcome:     string(v.Outcome),
			Fingerprint: pgTextOrNull(v.Fingerprint),
			MeasuredAt:  measuredAt,
		})
	}
	return out, dropped
}

// foldEdgeFanoutObservations writes a completed batch's measured edges into the store,
// inside the batch's own transaction — the outcome and what it measured commit together
// (ADR-0007). It returns at once for a job of any other kind, so every other completion
// path pays one comparison and nothing more.
//
// A dropped line is logged loudly (ADR-0001's fail-loud on a wire mismatch) and never
// fails the job: the legitimate lines in the same batch still commit, and a compromised
// prober cannot turn a malformed line into a queue denial of service.
func foldEdgeFanoutObservations(ctx context.Context, qtx *db.Queries, job db.ClaimJobRow, batchID int64, measuredAt pgtype.Timestamptz, obs []wire.Observation, logger *log.Logger) error {
	rows, dropped := toEdgeFanoutRows(job.Kind, batchID, measuredAt, obs)
	for _, why := range dropped {
		if logger != nil {
			logger.Printf("worker: job %d dropped malformed edge-fanout observation: %s (#773)", job.ID, why)
		}
	}
	for _, r := range rows {
		if err := qtx.InsertEdgeFanoutObservation(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// pgTextOrNull renders an optional text column: the empty string is the absent value, which
// is the NULL fingerprint every negative outcome carries.
func pgTextOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}
