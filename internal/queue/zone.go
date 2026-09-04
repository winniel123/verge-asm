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

func toZoneObservationParams(batchID int64, recs []scan.ZoneRecord) []db.InsertObservationParams {
	// Stamped at the operator's supply instant, so a stale zone file reads as stale (v1-spec §3.4).
	out := make([]db.InsertObservationParams, 0, len(recs))
	for _, r := range recs {
		// RestateZone drops a record it cannot marshal, so no empty-value guard is needed here.
		out = append(out, db.InsertObservationParams{
			BatchID:       batchID,
			Facet:         "dns-record",
			SubjectKind:   "name",
			SubjectKey:    r.Name,
			Discriminator: r.Qtype,
			VantageID:     pgtype.Int8{},
			Source:        scan.ZoneSource,
			Value:         []byte(r.Data),
			ObservedAt:    tstz(r.ObservedAt.UTC()),
		})
	}
	return out
}

func (w *Worker) completeZone(ctx context.Context, job db.ClaimJobRow, spec wire.JobSpec) error {
	zf, err := scan.ZoneScopeFromSpec(spec.Scope)
	if err != nil {
		return fmt.Errorf("decode zone scope: %w", err)
	}
	start := w.now()
	recs, skipped := scan.RestateZone(zf)

	var zt wire.Transcript
	// The zone file is already stored, so the transcript never copies it (raw-job-output §1.3).
	if w.captureOn() {
		zt = wire.ZoneTranscript{
			TranscriptFrame: wire.TranscriptFrame{Kind: job.Kind, Duration: w.now().Sub(start)},
			Restated:        len(recs),
			Skipped:         skipped,
			Outcome:         wire.ZoneParsed{},
		}
	}

	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		batchID, err := qtx.InsertBatch(ctx, db.InsertBatchParams{
			ScanID:        job.ScanID,
			DispatchID:    job.DispatchID,
			VantageID:     pgtype.Int8{},
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
		if err := markDone(ctx, qtx, job.ID, batchID); err != nil {
			return err
		}
		return w.persistTranscript(ctx, qtx, job.ID, zt)
	})
}

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
	// Reading a stored file has no transient failure to back off from, so the job never retries.
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
