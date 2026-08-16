package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

// The zone Scan is worker-read: unlike the dns Scan there is no prober exec and
// no Vantage, and a Batch's observations are stamped at the operator's supply
// instant rather than the worker's read (v1 spec §3.4, CONTEXT.md `Observation`).
// This file holds the worker's completion path for a zone job and the
// dispatcher's fan-out of one job per supplied zone file.

// toZoneObservationParams maps a zone file's restated records to observation
// rows for one Batch. Every row is attributed to the operator's zone-file source
// on the `dns-record` facet, carries no Vantage (a zone file's facts are not a
// function of where they are read from), and is stamped at the record's supply
// instant — never the worker's read — which is what keeps a stale zone file
// legible as stale instead of a current observation of a stale fact.
func toZoneObservationParams(batchID int64, recs []scan.ZoneRecord) []db.InsertObservationParams {
	out := make([]db.InsertObservationParams, 0, len(recs))
	for _, r := range recs {
		// r.Data is always a marshalled zoneValue (RestateZone skips a record it
		// cannot marshal), so it is never empty — no empty-value guard is needed.
		out = append(out, db.InsertObservationParams{
			BatchID:       batchID,
			Facet:         "dns-record",
			SubjectKind:   "name",
			SubjectKey:    r.Name,
			Discriminator: r.Qtype,
			VantageID:     pgtype.Int8{}, // no vantage choice at all
			Source:        scan.ZoneSource,
			Value:         []byte(r.Data),
			ObservedAt:    tstz(r.ObservedAt.UTC()),
		})
	}
	return out
}

// completeZone reads the zone file carried in the job's spec, restates it into
// `dns-record` observations at the supply instant, and commits the Batch, its
// observations and the job's done state in one transaction. A zone read has no
// network step to fail, so there is no retry or dead-letter path here.
func (w *Worker) completeZone(ctx context.Context, job db.ClaimJobRow, spec wire.JobSpec) error {
	zf, err := scan.ZoneScopeFromSpec(spec.Scope)
	if err != nil {
		return fmt.Errorf("decode zone scope: %w", err)
	}
	recs := scan.RestateZone(zf)
	return w.inTx(ctx, func(qtx *db.Queries) error {
		batchID, err := qtx.InsertBatch(ctx, db.InsertBatchParams{
			ScanID:        job.ScanID,
			DispatchID:    job.DispatchID,
			VantageID:     pgtype.Int8{}, // the zone Scan has no vantage
			Kind:          job.Kind,
			Outcome:       "completed",
			Offers:        job.Offers,
			RecordedScope: job.AttemptedScope,
		})
		if err != nil {
			return err
		}
		for _, p := range toZoneObservationParams(batchID, recs) {
			if err := qtx.InsertObservation(ctx, p); err != nil {
				return err
			}
		}
		return qtx.MarkJobDone(ctx, db.MarkJobDoneParams{ID: job.ID, BatchID: pgInt8(batchID)})
	})
}

// zoneFiles reads the latest supplied zone file per name-scope Seed — the scope
// of the zone Scan (v1 spec §3.4). A re-supply is a new row with a new supply
// instant; only the most recent is restated.
func zoneFiles(ctx context.Context, q *db.Queries) ([]scan.ZoneFile, error) {
	rows, err := q.LatestZoneFilesForDispatch(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]scan.ZoneFile, 0, len(rows))
	for _, r := range rows {
		files = append(files, scan.ZoneFile{
			SeedID:     r.SeedID,
			Domain:     r.NameDomain.String,
			SuppliedAt: r.SuppliedAt.Time,
			Content:    r.Content,
		})
	}
	return files, nil
}

// enqueueZoneJob enqueues one worker-read job for one supplied zone file. It
// carries no Vantage and empty offers, and does not retry: MaxAttempts is 1
// because reading a stored file has no transient failure to back off from.
func enqueueZoneJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.ZoneJob) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:seed:%d", scanID, j.SeedID))
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
	_, err = qtx.EnqueueJob(ctx, db.EnqueueJobParams{
		ScanID:         scanID,
		VantageID:      pgtype.Int8{},
		DispatchID:     pgInt8(dispatchID),
		Kind:           scan.ZoneKind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         []byte("{}"),
		Attempt:        1,
		MaxAttempts:    1,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}
