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

func (d *Dispatcher) fanOutEdgeFanout(ctx context.Context, scanID, dispatchID int64) (int, error) {
	// A name whose scope was withdrawn since it resolved contributes no candidate (ADR-0079, #742).
	estate, _, err := hotEstate(ctx, d.q, d.now())
	if err != nil {
		return 0, err
	}
	// An install with neither limb dispatches an empty scope rather than an error (#988).
	jobs := scan.BuildEdgeFanoutJobs(scanID, estate.EdgeFanoutPopulation())
	// A declared address scope has no ceiling, so the jobs stream instead of materializing (ADR-0127).
	return streamEnqueue(ctx, d, jobs, func(ctx context.Context, qtx *db.Queries, j scan.EdgeFanoutJob) error {
		return enqueueEdgeFanoutJob(ctx, qtx, scanID, dispatchID, j)
	})
}

func enqueueEdgeFanoutJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.EdgeFanoutJob) error {
	// The Batch label names the chunk, because this Scan has no vantage or seed to key it on.
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:edges:%d", scanID, j.Chunk))
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
	// A handshake is a network step that transiently fails, so the job retries like a hot one.
	_, err = qtx.EnqueueJob(ctx, db.EnqueueJobParams{
		ScanID:         scanID,
		VantageID:      pgtype.Int8{}, // a default certificate does not vary by vantage (ADR-0129, #954)
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

func toEdgeFanoutRows(jobKind string, batchID int64, measuredAt pgtype.Timestamptz, obs []wire.Observation) ([]db.InsertEdgeFanoutObservationParams, []string) {
	// A line's Kind is the prober's word, so only the dispatcher's kind admits this fold (#773).
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
		// This leaf decides membership and opens no timeline, so its lines carry no facet.
		// A line claiming one was gated on its subject, never on the address read here.
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
		// A fingerprint disagreeing with its material names a certificate no read can join to (#984).
		if o.CertMaterial != nil && o.CertMaterial.Fingerprint != v.Fingerprint {
			dropped = append(dropped, fmt.Sprintf("address %s: fingerprint %q disagrees with its certificate material %q",
				addr, v.Fingerprint, o.CertMaterial.Fingerprint))
			continue
		}
		// The leaf handshakes each distinct address once, so a repeat is a line nothing measured.
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

func foldEdgeFanoutObservations(ctx context.Context, qtx *db.Queries, job db.ClaimJobRow, batchID int64, measuredAt pgtype.Timestamptz, obs []wire.Observation, logger *log.Logger) error {
	rows, dropped := toEdgeFanoutRows(job.Kind, batchID, measuredAt, obs)
	// A malformed line is logged and never fails the job: a prober must not deny the queue (ADR-0001).
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

func pgTextOrNull(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: s, Valid: true}
}

// A second reader could apply a different absence rule, so the web layer reaches this one (#987).

type EdgeFanoutStore interface {
	GetScanByKind(ctx context.Context, kind string) (db.Scan, error)
	ListEdgeFanoutMeasurements(ctx context.Context) ([]db.ListEdgeFanoutMeasurementsRow, error)
	ListEdgeFanoutMeasurementsOver(ctx context.Context, addresses []string) ([]db.ListEdgeFanoutMeasurementsOverRow, error)
	ListCertificateMaterialDER(ctx context.Context, fingerprints []string) ([]db.ListCertificateMaterialDERRow, error)
	ScanHasCompletedBatch(ctx context.Context, kind string) (bool, error)
}

// A bare slice would make nil mean read-everything, so an empty candidate set reads the store.
// A read narrower than the caller needs drops a row, and a missing row reads as pending.

type EdgeFanoutBound struct {
	over    []string
	bounded bool
}

// A declared address scope may be a /8, which no address list can hold (ADR-0127, #1036).

func EdgeFanoutUnbounded() EdgeFanoutBound { return EdgeFanoutBound{} }

func EdgeFanoutOver(addrs []netip.Addr) EdgeFanoutBound {
	// Every comparison in internal/custody is family-agnostic, so a match must not turn on spelling.
	over := make([]string, 0, len(addrs))
	for _, a := range addrs {
		over = append(over, a.Unmap().String())
	}
	return EdgeFanoutBound{over: over, bounded: true}
}

func ReadEdgeFanout(ctx context.Context, q EdgeFanoutStore, bound EdgeFanoutBound) (custody.EdgeFanout, error) {
	s, err := q.GetScanByKind(ctx, scan.EdgeFanoutKind)
	// An absent or disabled Scan yields the zero value, which reaches every target as before ADR-0129.
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return custody.EdgeFanout{}, nil
		}
		return custody.EdgeFanout{}, err
	}
	if !s.Enabled {
		return custody.EdgeFanout{}, nil
	}
	rows, err := readEdgeFanoutRows(ctx, q, bound)
	if err != nil {
		return custody.EdgeFanout{}, err
	}
	// A failed read is the dispatch failing, never a finding, so it must not silently widen the reach.
	material, err := readEdgeFanoutMaterial(ctx, q, rows)
	if err != nil {
		return custody.EdgeFanout{}, err
	}
	// Both limbs share one store since #988, so a declaration-limb row must not lift the floor.
	completed, err := q.ScanHasCompletedBatch(ctx, scan.EdgeFanoutKind)
	if err != nil {
		return custody.EdgeFanout{}, err
	}
	out := toEdgeFanout(completed, rows, material)
	// A bounded map is short, so Partial tells the census to refuse it rather than under-count.
	out.Partial = bound.bounded
	return out, nil
}

func readEdgeFanoutRows(ctx context.Context, q EdgeFanoutStore, bound EdgeFanoutBound) ([]db.ListEdgeFanoutMeasurementsRow, error) {
	if !bound.bounded {
		return q.ListEdgeFanoutMeasurements(ctx)
	}
	// An empty bound could only return rows the caller already knows are none, so no query is issued.
	if len(bound.over) == 0 {
		return nil, nil
	}
	rows, err := q.ListEdgeFanoutMeasurementsOver(ctx, bound.over)
	if err != nil {
		return nil, err
	}
	// sqlc names a row type per query, so copying the narrow rows into the wide type keeps one fold.
	out := make([]db.ListEdgeFanoutMeasurementsRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, db.ListEdgeFanoutMeasurementsRow{
			Address:     r.Address,
			Outcome:     r.Outcome,
			Fingerprint: r.Fingerprint,
		})
	}
	return out, nil
}

func readEdgeFanoutMaterial(ctx context.Context, q EdgeFanoutStore, rows []db.ListEdgeFanoutMeasurementsRow) (map[string][]byte, error) {
	fingerprints := edgeFanoutFingerprints(rows)
	if len(fingerprints) == 0 {
		return nil, nil
	}
	// One CDN edge sits behind thousands of addresses, so a certificate crosses the wire once (#1035).
	material, err := q.ListCertificateMaterialDER(ctx, fingerprints)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(material))
	for _, m := range material {
		out[m.Fingerprint] = m.Der
	}
	return out, nil
}

func edgeFanoutFingerprints(rows []db.ListEdgeFanoutMeasurementsRow) []string {
	seen := make(map[string]struct{}, len(rows))
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		// A negative outcome names no certificate, and an absent fingerprint stores as NULL or as empty.
		if !r.Fingerprint.Valid || r.Fingerprint.String == "" {
			continue
		}
		if _, dup := seen[r.Fingerprint.String]; dup {
			continue
		}
		seen[r.Fingerprint.String] = struct{}{}
		out = append(out, r.Fingerprint.String)
	}
	return out
}

func toEdgeFanout(completed bool, rows []db.ListEdgeFanoutMeasurementsRow, material map[string][]byte) custody.EdgeFanout {
	// Which limb a key came from is decided where this map is read, never here (ADR-0129, #956).
	shared := make(map[netip.Addr]bool, len(rows))
	verdict := make(map[string]bool, len(material))
	// An address with no row gets no key, and the derivation reads that missing key as pending.
	for _, r := range rows {
		addr, err := netip.ParseAddr(r.Address)
		if err != nil {
			// A row that will not parse measured nothing, so withholding the address is the safe direction.
			continue
		}
		var edge bool
		// A negative found nothing at the address: not-shared, never pending (ADR-0163 §1).
		if r.Outcome == string(edgefanout.Presented) {
			var fingerprint string
			if r.Fingerprint.Valid {
				fingerprint = r.Fingerprint.String
			}
			v, derived := verdict[fingerprint]
			if !derived {
				// No captured material is a fan-out of zero, reached, never pending (ADR-0163 §1).
				v = custody.SharedEdge(edgeFanoutSANs(material[fingerprint]))
				verdict[fingerprint] = v
			}
			edge = v
		}
		shared[addr.Unmap()] = edge
	}
	// The errored floor is per limb, and only the estate holds the candidate set resolving it (#1018).
	return custody.EdgeFanout{Enabled: true, BatchCompleted: completed, Shared: shared}
}

func edgeFanoutSANs(der []byte) []string {
	if len(der) == 0 {
		return nil
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil
	}
	// A CN is not a SAN, is repeated among them in practice, and alone never reaches the threshold.
	return cert.DNSNames
}
