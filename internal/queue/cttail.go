package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

// A log operator may cap a batch below this, so fewer is legal (ct-source-replacement §4.4).

const ctTailBatch = 256

// A busy log outruns one poll, so the delta is read in windows (ct-source-replacement §4.4).

const maxEntriesPerPoll = 16384

func ctTailShrunk(hasCursor bool, cursor, treeSize int64) bool {
	// An append-only tree cannot shrink, so a head below the cursor is a fork or rollback (#1656).
	return hasCursor && treeSize < cursor
}

func ctTailWindow(hasCursor bool, cursor, treeSize int64) (start, end int64) {
	// A log with no cursor is seeded at its head, so the tail never backfills history (#1654).
	if !hasCursor {
		return treeSize, treeSize
	}
	end = treeSize
	if end > cursor+maxEntriesPerPoll {
		end = cursor + maxEntriesPerPoll
	}
	return cursor, end
}

func (w *Worker) WithCTTail(fetcher CTFetcher, throttle CTThrottle) *Worker {
	w.ctTailFetcher = fetcher
	w.ctTailThrottle = throttle
	return w
}

func (w *Worker) completeCTTail(ctx context.Context, job db.ClaimJobRow, spec wire.JobSpec) error {
	// A failed poll leaves the cursor untouched, so it never asserts an absence (ADR-0005).
	if w.ctTailFetcher == nil {
		return fmt.Errorf("queue: ct-tail job %d with no CT tail fetcher configured", job.ID)
	}
	lg, err := scan.CTTailScopeFromSpec(spec.Scope)
	if err != nil {
		return fmt.Errorf("decode ct-tail scope: %w", err)
	}
	if lg.Tiled {
		return w.discardCanceled(job.ID, w.completeCTTailTiled(ctx, job, lg))
	}
	return w.discardCanceled(job.ID, w.completeCTTailRFC(ctx, job, lg))
}

func (w *Worker) reserveCTTailSlot(ctx context.Context, jobID int64) error {
	// A full window outlives the stale-job reaper, so each reservation renews first (#1709).
	if err := w.renewJobLease(ctx, jobID); err != nil {
		return err
	}
	return w.reserveCTSlot(ctx, w.ctTailThrottle)
}

func (w *Worker) completeCTTailRFC(ctx context.Context, job db.ClaimJobRow, lg scan.CTLog) error {
	base := ensureTrailingSlash(lg.URL)

	// The tail reads only forward deltas and never backfills (ct-source-replacement §4).
	cursor, hasCursor := int64(0), true
	if cur, gerr := w.q.GetCTLogCursor(ctx, lg.LogID); gerr == nil {
		cursor = cur.TreeSize
	} else if errors.Is(gerr, pgx.ErrNoRows) {
		hasCursor = false
	} else {
		return fmt.Errorf("ct-tail cursor: %w", gerr)
	}

	if rerr := w.reserveCTTailSlot(ctx, job.ID); rerr != nil {
		return rerr
	}
	status, body, ferr := w.ctTailFetcher.Fetch(ctx, base+"ct/v1/get-sth")
	if ferr != nil || status != 200 {
		// The nil transcript is deliberate: raw-output capture is scoped to crt.sh (#870).
		return w.retryOrDeadLetterCT(ctx, job, nil, ctHTTPCause(ferr, status, "get-sth"))
	}
	sth, perr := scan.ParseSTH(body)
	if perr != nil {
		return w.retryOrDeadLetterCT(ctx, job, nil, perr)
	}

	if ctTailShrunk(hasCursor, cursor, sth.TreeSize) {
		return w.retryOrDeadLetterCT(ctx, job, nil, safeProgress(fmt.Sprintf("CT log STH tree size %d below cursor %d", sth.TreeSize, cursor)))
	}

	start, end := ctTailWindow(hasCursor, cursor, sth.TreeSize)
	var sans []string
	reached := start
	for reached < end {
		reqEnd := reached + ctTailBatch - 1
		if reqEnd > end-1 {
			reqEnd = end - 1
		}
		if rerr := w.reserveCTTailSlot(ctx, job.ID); rerr != nil {
			return rerr
		}
		st, eb, fe := w.ctTailFetcher.Fetch(ctx, getEntriesURL(base, reached, reqEnd))
		if fe != nil || st != 200 {
			return w.retryOrDeadLetterCT(ctx, job, nil, ctHTTPCause(fe, st, "get-entries"))
		}
		entries, pe := scan.ParseLogEntries(eb)
		if pe != nil {
			return w.retryOrDeadLetterCT(ctx, job, nil, pe)
		}
		if len(entries) == 0 {
			break
		}
		for _, e := range entries {
			names, derr := scan.LeafSANs(e.LeafInput, e.ExtraData)
			if derr != nil {
				// A corroborative source's unreadable entry never fails the poll (ADR-0213 §1).
				w.log.Printf("worker: ct-tail job %d skipped a log entry: %v", job.ID, derr)
				continue
			}
			sans = append(sans, names...)
		}
		reached += int64(len(entries))
	}

	seeds, err := ctSeeds(ctx, w.q)
	if err != nil {
		return err
	}
	admissions := scan.AdmitCTNames(sans, seeds)
	return w.admitCTTail(ctx, job, lg, sth.Raw, reached, admissions)
}

func (w *Worker) completeCTTailTiled(ctx context.Context, job db.ClaimJobRow, lg scan.CTLog) error {
	base := ensureTrailingSlash(lg.URL)

	cursor, hasCursor := int64(0), true
	if cur, gerr := w.q.GetCTLogCursor(ctx, lg.LogID); gerr == nil {
		cursor = cur.TreeSize
	} else if errors.Is(gerr, pgx.ErrNoRows) {
		hasCursor = false
	} else {
		return fmt.Errorf("ct-tail cursor: %w", gerr)
	}

	if rerr := w.reserveCTTailSlot(ctx, job.ID); rerr != nil {
		return rerr
	}
	status, body, ferr := w.ctTailFetcher.Fetch(ctx, base+"checkpoint")
	if ferr != nil || status != 200 {
		return w.retryOrDeadLetterCT(ctx, job, nil, ctHTTPCause(ferr, status, "checkpoint"))
	}
	sth, perr := scan.ParseCheckpoint(body)
	if perr != nil {
		return w.retryOrDeadLetterCT(ctx, job, nil, perr)
	}

	// This shrink check is not the consistency proof, and no signature is verified here.
	if ctTailShrunk(hasCursor, cursor, sth.TreeSize) {
		return w.retryOrDeadLetterCT(ctx, job, nil, safeProgress(fmt.Sprintf("CT log checkpoint tree size %d below cursor %d", sth.TreeSize, cursor)))
	}

	start, end := ctTailWindow(hasCursor, cursor, sth.TreeSize)
	var sans []string
	reached := start
	for reached < end {
		tileIdx := reached / scan.CTTileWidth
		tileBase := tileIdx * scan.CTTileWidth
		// A data tile caps at 256 entries, so that is the batch cap (ct-source-replacement §4.4).
		width := int64(scan.CTTileWidth)
		if tileBase+scan.CTTileWidth > sth.TreeSize {
			width = sth.TreeSize - tileBase
		}
		if rerr := w.reserveCTTailSlot(ctx, job.ID); rerr != nil {
			return rerr
		}
		st, tb, fe := w.ctTailFetcher.Fetch(ctx, dataTileURL(base, tileIdx, width))
		if fe != nil || st != 200 {
			return w.retryOrDeadLetterCT(ctx, job, nil, ctHTTPCause(fe, st, "tile/data"))
		}
		ders, pe := scan.ParseDataTile(tb)
		if pe != nil {
			if cause, permanent := ctTileCause(pe); permanent {
				return w.deadLetterCT(ctx, job, nil, cause)
			}
			return w.retryOrDeadLetterCT(ctx, job, nil, pe)
		}
		offset := int(reached - tileBase)
		if offset >= len(ders) {
			break
		}
		limit := len(ders)
		if tileBase+int64(limit) > end {
			limit = int(end - tileBase)
		}
		for i := offset; i < limit; i++ {
			names, derr := scan.CertSANs(ders[i])
			if derr != nil {
				w.log.Printf("worker: ct-tail job %d skipped a tiled log entry: %v", job.ID, derr)
				continue
			}
			sans = append(sans, names...)
		}
		reached = tileBase + int64(limit)
	}

	seeds, err := ctSeeds(ctx, w.q)
	if err != nil {
		return err
	}
	admissions := scan.AdmitCTNames(sans, seeds)
	return w.admitCTTail(ctx, job, lg, sth.Raw, reached, admissions)
}

func dataTileURL(base string, tileIdx, width int64) string {
	p := scan.DataTilePath(tileIdx)
	if width < scan.CTTileWidth {
		return fmt.Sprintf("%stile/data/%s.p/%d", base, p, width)
	}
	return fmt.Sprintf("%stile/data/%s", base, p)
}

func (w *Worker) admitCTTail(ctx context.Context, job db.ClaimJobRow, lg scan.CTLog, signedHead []byte, reached int64, admissions []scan.CTAdmission) error {
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		// Read after this batch's inserts, every admitted name would read as drift, not discovery.
		known, err := knownNameSet(ctx, qtx)
		if err != nil {
			return err
		}
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
		drift := 0
		for _, a := range admissions {
			if err := qtx.InsertAdmittedName(ctx, db.InsertAdmittedNameParams{
				Name:    a.Name,
				Source:  scan.CTTailSource,
				SeedID:  a.SeedID,
				BatchID: batchID,
			}); err != nil {
				return err
			}
			if _, isKnown := known[a.Name]; isKnown {
				drift++
			}
		}
		if err := qtx.AdvanceCTLogCursor(ctx, db.AdvanceCTLogCursorParams{
			LogID:      lg.LogID,
			TreeSize:   reached,
			SignedHead: signedHead,
		}); err != nil {
			return err
		}
		// A durable signal needs a new facet, so drift is ephemeral (ct-source-replacement §4.1).
		w.emitJobEvent(ctx, qtx, job, "", countLabel(len(admissions), "name admitted", "names admitted"))
		if drift > 0 {
			w.emitJobEvent(ctx, qtx, job, "warn", ctDriftLabel(drift))
		}
		return markDone(ctx, qtx, job.ID, batchID)
	})
}

func knownNameSet(ctx context.Context, q *db.Queries) (map[string]struct{}, error) {
	domains, err := nameSeedDomains(ctx, q)
	if err != nil {
		return nil, err
	}
	admitted, err := admittedNames(ctx, q)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(domains)+len(admitted))
	for _, d := range domains {
		set[d] = struct{}{}
	}
	for _, n := range admitted {
		set[n] = struct{}{}
	}
	return set, nil
}

func ctDriftLabel(n int) string {
	// A count leaks nothing where a name would, so the drift line never carries one (#780).
	return fmt.Sprintf("%s for known names", countLabel(n, "new certificate", "new certificates"))
}

// An unknown entry type withholds the leaf's length, so every refetch fails identically (ADR-0191).

func ctTileCause(err error) (cause error, permanent bool) {
	var unsupported *scan.UnsupportedEntryTypeError
	if !errors.As(err, &unsupported) {
		return err, false
	}
	return safeProgress(fmt.Sprintf("cannot read this log: unsupported tile entry type %d", unsupported.EntryType)), true
}

func ctHTTPCause(ferr error, status int, endpoint string) error {
	if ferr != nil {
		return ferr
	}
	return safeProgress(fmt.Sprintf("CT log %s returned HTTP %d", endpoint, status))
}

func ensureTrailingSlash(u string) string {
	if strings.HasSuffix(u, "/") {
		return u
	}
	return u + "/"
}

func getEntriesURL(base string, start, end int64) string {
	return fmt.Sprintf("%sct/v1/get-entries?start=%d&end=%d", base, start, end)
}

func (d *Dispatcher) fanOutCTTail(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	// The Scan ships enabled and the source off: the toggle is consent, not schedule (ADR-0003).
	enabled, err := sourceEnabled(ctx, qtx, scan.CTTailSource, false)
	if err != nil {
		return 0, err
	}
	if !enabled {
		return 0, nil
	}
	logs, err := scan.SelectTailLogs(d.now())
	if err != nil {
		return 0, err
	}
	enqueued := 0
	for _, j := range scan.BuildCTTailJobs(scanID, logs) {
		if err := enqueueCTTailJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

func enqueueCTTailJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.CTTailJob) error {
	spec, err := j.JobSpec(fmt.Sprintf("scan:%d:log:%s", scanID, j.Log.LogID))
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
		Kind:           scan.CTTailKind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         []byte("{}"),
		Attempt:        1,
		MaxAttempts:    5,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}
