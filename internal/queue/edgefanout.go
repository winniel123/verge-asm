package queue

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/custody"
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

// readEdgeFanout reads the `edge-fanout` Scan's measured result in the form the custody
// extension's reach consumes (#985, ADR-0129 §4). It is the ONE read path from the
// leaf's store to the derivation: every caller that assembles a custody.Estate reaches
// the stored rows through here, so no second reader can apply a different absence rule.
//
// Three reads, in this order:
//
//   - The Scan row. The measurement narrows the reach only where the Scan is IN FORCE.
//     A disabled Scan, and a Scan row that is absent altogether — an install whose
//     migrations predate #983 — both yield the zero custody.EdgeFanout, which reaches
//     every direct-A target exactly as the extension did before ADR-0129. That is half
//     of the fourth absence case.
//   - The measurements. Each address is keyed to the boolean custody.SharedEdge
//     computes over the SAN set of the certificate the edge served. An address with no
//     row gets NO KEY, which is what the derivation reads as *measurement pending* and
//     holds.
//   - Whether a Batch of this Scan has ever completed, and ONLY where the Scan recorded
//     nothing at all. That read is the fourth case's other half — the ERRORED Scan —
//     and toEdgeFanout states why it is needed.
//
// A read that FAILS returns the error rather than an open reach. A failed read is not a
// decision: it is the dispatch failing, and the tick retries. Opening the reach on a
// transient database error would widen the estate on the one signal that says nothing
// was measured.
func readEdgeFanout(ctx context.Context, q *db.Queries) (custody.EdgeFanout, error) {
	s, err := q.GetScanByKind(ctx, scan.EdgeFanoutKind)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return custody.EdgeFanout{}, nil
		}
		return custody.EdgeFanout{}, err
	}
	if !s.Enabled {
		return custody.EdgeFanout{}, nil
	}
	rows, err := q.ListEdgeFanoutMeasurements(ctx)
	if err != nil {
		return custody.EdgeFanout{}, err
	}
	// The second query runs ONLY on an empty store, so the healthy path — where the
	// Scan has measured something — pays one row count and nothing more.
	completed := false
	if len(rows) == 0 {
		completed, err = q.ScanHasCompletedBatch(ctx, scan.EdgeFanoutKind)
		if err != nil {
			return custody.EdgeFanout{}, err
		}
	}
	return toEdgeFanout(completed, rows), nil
}

// toEdgeFanout reduces the stored measurements to the derivation's input. It is the
// pure half of the read, so the absence rule's shape is tested without a database.
//
// Every returned key is an address the Scan MEASURED, and its value is the boolean
// custody.SharedEdge computes over the SAN set the edge served. An address with no row
// gets no key, and that missing key is what the derivation holds on.
//
// AN EMPTY STORE HAS TWO MEANINGS, and completed is what tells them apart. It is the
// ERRORED half of ADR-0129's fourth absence case, and without it hold-then-open has no
// floor at all:
//
//   - The Scan has not run yet (completed false). Its candidates are genuinely
//     *measurement pending*, so they are HELD, bounded by the daily cadence. This is
//     the fresh install and the newly-declared extension, and holding here is exactly
//     what keeps the modal all-CDN install from showing appear-then-withdraw churn.
//   - The Scan RUNS and records nothing (completed true). It is enabled, it has
//     completed a Batch, and the store is still empty. That is not a lag — it is the
//     measurement failing. The commonest cause is a prober binary older than #982,
//     which falls through its kind switch, emits a line the recording fold drops, and
//     completes the Batch clean. So it repeats every tick, forever.
//
// Holding on the second case would withhold the WHOLE custody-extension limb — every
// in-zone direct-A address, on every Scan the gate covers — silently and for good. A
// silently missing estate is the direction ADR-0129 §2 refuses; probing edges it could
// have skipped is the loud direction it accepts. So the errored Scan reaches, which is
// the pre-ADR-0129 behaviour the fourth case names.
//
// One recorded measurement of any kind lifts the floor, and the three negative outcomes
// each record one. So a store that stays empty means the Scan measured NOTHING — never
// that the network was bad — and the reading cannot be reached by a run of failed
// handshakes. An install with no custody extension records nothing either, and reads as
// errored; it holds no extension-covered address, so the reading changes no answer.
func toEdgeFanout(completed bool, rows []db.ListEdgeFanoutMeasurementsRow) custody.EdgeFanout {
	if len(rows) == 0 && completed {
		return custody.EdgeFanout{}
	}
	shared := make(map[netip.Addr]bool, len(rows))
	for _, r := range rows {
		addr, err := netip.ParseAddr(r.Address)
		if err != nil {
			// The writer stores the rendering of a parsed netip.Addr, so this is
			// unreachable in practice. Skipping leaves the address with no key, so it
			// is HELD rather than reached — an address whose row we cannot read is one
			// nothing measured, and withholding the probe is the safe direction.
			continue
		}
		// Only a `presented` handshake can hold identities. The three negatives each
		// measured the address and found none there, which reduces to a fan-out of zero
		// — measured and not-shared, never pending.
		var sans []string
		if r.Outcome == string(edgefanout.Presented) {
			sans = edgeFanoutSANs(r.Der)
		}
		shared[addr.Unmap()] = custody.SharedEdge(sans)
	}
	return custody.EdgeFanout{Enabled: true, Shared: shared}
}

// edgeFanoutSANs decodes one measured edge's default certificate to the dNSName SANs
// the fan-out reduction counts (ADR-0129 §1 as sharpened by the #954 amendment). The
// SAN set is the Observed wire fact; the count over it is the derivation.
//
// It reads the dNSName SANs ALONE and folds in no subject common name. A CN is not a
// SAN, a modern certificate repeats it among them anyway, and a legacy CN-only
// certificate names one identity and could never reach the threshold on its own.
//
// An absent or undecodable DER yields no names, so the edge reduces to a fan-out of
// zero and is reached. A `presented` row whose material never landed — a handshake that
// carried a chain but no leaf DER — is measured and not-shared rather than held, which
// is the loud, wasteful direction ADR-0129 §2 accepts. Holding it instead would withhold
// the address forever on an error that never clears, and a silently missing estate is
// what this derivation exists to avoid.
func edgeFanoutSANs(der []byte) []string {
	if len(der) == 0 {
		return nil
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil
	}
	return cert.DNSNames
}
