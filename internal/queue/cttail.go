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

// The `ct-tail` Scan is the CT-logs-direct drift tail (spec §4, ADR-0027, ADR-0106). It
// is worker-read like `ct` and `zone` — no prober exec, no Vantage — but unlike `ct`
// (which queries the crt.sh index by name) it reads the logs DIRECTLY and forward-only:
// each poll of a log reads the entries appended since the log's own durable cursor, up
// to a bounded window, decodes the certificate each entry carries, and admits the
// in-scope Names it names — an `admitted_name` row on `authority: inferred` citing the
// Batch, exactly as `ct` does (ADR-0027). It produces no facet, no Signal and no
// timeline; when a certificate appears for a name the operator ALREADY knows, it emits
// an EPHEMERAL drift event, never a durable row (§4.1). A failed poll admits nothing and
// asserts no absence (ADR-0005, ADR-0027 §7). This file holds the forward-delta fetch,
// the cursor advance, the admission write and the drift event. The wire parsers and the
// admission decision are the pure half in internal/scan/cttail.go.

// ctTailBatch is the get-entries batch size the tail requests. A log legitimately
// returns fewer than requested — per-operator caps (Argon 32/request, others 256 —
// §4.4) — so the tail advances by the count returned, never by the count asked for.
const ctTailBatch = 256

// maxEntriesPerPoll bounds how many entries one poll of one log consumes, so a job that
// is far behind (a log first enabled, or one that grew fast between polls) does a
// bounded amount of work and advances its cursor toward the head over successive polls
// rather than reading an unbounded delta in one job. It is a MEASURED BAR and an
// operator dial (§4.4): larger keeps a busy log current, smaller bounds each job harder.
const maxEntriesPerPoll = 16384

// WithCTTail wires the tail's log fetcher onto the Worker. It is separate from NewWorker
// and from WithCT so the measurement-only construction and the `ct` construction stay
// unchanged; a worker with no tail fetcher configured refuses a `ct-tail` job rather
// than silently admitting nothing. The tail reuses the CTFetcher seam — the same
// distinctive User-Agent and unfollowed-redirect (SSRF) guard as the crt.sh fetcher —
// pointed at the RFC 6962 log endpoints instead of crt.sh.
func (w *Worker) WithCTTail(fetcher CTFetcher) *Worker {
	w.ctTailFetcher = fetcher
	return w
}

// completeCTTail is the worker-read path for a `ct-tail` job. It reads the log with the
// client its scope names — the RFC 6962 client (#874) or the static-ct-api tiled client
// (#877) — decided by the Tiled discriminator that travelled from SelectTailLogs. Both
// clients read only the forward delta above the log's durable cursor, admit the in-scope
// Names, and advance the cursor; on any non-200 or malformed body either retries or
// dead-letters and leaves the cursor untouched, so a failed poll never advances past
// unread entries and never asserts an absence (ADR-0005, ADR-0027 §7).
func (w *Worker) completeCTTail(ctx context.Context, job db.ClaimJobRow, spec wire.JobSpec) error {
	if w.ctTailFetcher == nil {
		return fmt.Errorf("queue: ct-tail job %d with no CT tail fetcher configured", job.ID)
	}
	lg, err := scan.CTTailScopeFromSpec(spec.Scope)
	if err != nil {
		return fmt.Errorf("decode ct-tail scope: %w", err)
	}
	if lg.Tiled {
		return w.completeCTTailTiled(ctx, job, lg)
	}
	return w.completeCTTailRFC(ctx, job, lg)
}

// completeCTTailRFC reads one RFC 6962 log's forward delta (#874): fetch get-sth for the
// current tree size, read positions [cursor, min(size, cursor+maxEntriesPerPoll)) in
// get-entries batches, decode each MerkleTreeLeaf, and admit the in-scope Names.
func (w *Worker) completeCTTailRFC(ctx context.Context, job db.ClaimJobRow, lg scan.CTLog) error {
	base := ensureTrailingSlash(lg.URL)

	// The forward cursor: the position the last poll read up to. No row yet means the
	// log has never been polled — start at position 0 and read the current delta from
	// the origin forward (never below it afterwards — the §4 invariant). The read is
	// outside the terminal tx; the advance is inside it.
	start := int64(0)
	if cur, gerr := w.q.GetCTLogCursor(ctx, lg.LogID); gerr == nil {
		start = cur.TreeSize
	} else if !errors.Is(gerr, pgx.ErrNoRows) {
		return fmt.Errorf("ct-tail cursor: %w", gerr)
	}

	status, body, ferr := w.ctTailFetcher.Fetch(ctx, base+"ct/v1/get-sth")
	if ferr != nil || status != 200 {
		return w.retryOrDeadLetterCT(ctx, job, ctHTTPCause(ferr, status, "get-sth"))
	}
	sth, perr := scan.ParseSTH(body)
	if perr != nil {
		return w.retryOrDeadLetterCT(ctx, job, perr)
	}

	// The forward window is [start, end): from the cursor up to the head, capped at
	// maxEntriesPerPoll so one poll does bounded work. reached tracks how far the poll
	// actually consumed — the position the cursor advances to.
	end := sth.TreeSize
	if end > start+maxEntriesPerPoll {
		end = start + maxEntriesPerPoll
	}
	var sans []string
	reached := start
	for reached < end {
		reqEnd := reached + ctTailBatch - 1
		if reqEnd > end-1 {
			reqEnd = end - 1
		}
		st, eb, fe := w.ctTailFetcher.Fetch(ctx, getEntriesURL(base, reached, reqEnd))
		if fe != nil || st != 200 {
			return w.retryOrDeadLetterCT(ctx, job, ctHTTPCause(fe, st, "get-entries"))
		}
		entries, pe := scan.ParseLogEntries(eb)
		if pe != nil {
			return w.retryOrDeadLetterCT(ctx, job, pe)
		}
		if len(entries) == 0 {
			break // no progress possible: stop rather than spin, and admit what we have
		}
		for _, e := range entries {
			names, derr := scan.LeafSANs(e.LeafInput, e.ExtraData)
			if derr != nil {
				// One malformed or unrecognised entry does not fail the poll: the tail
				// reads a corroborative source, so it skips the entry and reads on.
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

// completeCTTailTiled reads one static-ct-api log's forward delta (#877, §4.3). A tiled
// log serves no get-sth and no get-entries; it serves static files under monitoring_url.
// The poll fetches `checkpoint` (the signed tree head) for the current tree size, then
// reads the entries in [cursor, min(size, cursor+maxEntriesPerPoll)) from the
// `tile/data/<N>` files that cover them, one tile per fetch. A data tile holds at most
// CTTileWidth (256) entries, so the tile granularity IS the tiled batch cap (§4.4) — no
// separate request-size dial. The forward-only invariant holds exactly as in the RFC
// path: the cursor starts at the last position read and never moves below it.
func (w *Worker) completeCTTailTiled(ctx context.Context, job db.ClaimJobRow, lg scan.CTLog) error {
	base := ensureTrailingSlash(lg.URL)

	start := int64(0)
	if cur, gerr := w.q.GetCTLogCursor(ctx, lg.LogID); gerr == nil {
		start = cur.TreeSize
	} else if !errors.Is(gerr, pgx.ErrNoRows) {
		return fmt.Errorf("ct-tail cursor: %w", gerr)
	}

	status, body, ferr := w.ctTailFetcher.Fetch(ctx, base+"checkpoint")
	if ferr != nil || status != 200 {
		return w.retryOrDeadLetterCT(ctx, job, ctHTTPCause(ferr, status, "checkpoint"))
	}
	sth, perr := scan.ParseCheckpoint(body)
	if perr != nil {
		return w.retryOrDeadLetterCT(ctx, job, perr)
	}

	// Opportunistic append-only guard (§4.4): a checkpoint whose tree is SMALLER than the
	// cursor is a shrink — a log fork or rollback, which an append-only tree must never
	// do. It is keyless and near-free (the checkpoint is already fetched), so it runs each
	// poll, but it is NOT the full consistency proof: signature and inclusion verification
	// stay deferred with #874's, the checkpoint kept verbatim to enable them later. On a
	// shrink the poll admits nothing and leaves the cursor untouched, exactly like any
	// other transient anomaly.
	if sth.TreeSize < start {
		return w.retryOrDeadLetterCT(ctx, job, safeProgress(fmt.Sprintf("CT log checkpoint tree size %d below cursor %d", sth.TreeSize, start)))
	}

	end := sth.TreeSize
	if end > start+maxEntriesPerPoll {
		end = start + maxEntriesPerPoll
	}
	var sans []string
	reached := start
	for reached < end {
		tileIdx := reached / scan.CTTileWidth
		tileBase := tileIdx * scan.CTTileWidth
		// The tile's server-side width: a full tile below the head, or the partial width
		// of the head tile. The partial `.p/<W>` suffix names exactly this width.
		width := int64(scan.CTTileWidth)
		if tileBase+scan.CTTileWidth > sth.TreeSize {
			width = sth.TreeSize - tileBase
		}
		st, tb, fe := w.ctTailFetcher.Fetch(ctx, dataTileURL(base, tileIdx, width))
		if fe != nil || st != 200 {
			return w.retryOrDeadLetterCT(ctx, job, ctHTTPCause(fe, st, "tile/data"))
		}
		ders, pe := scan.ParseDataTile(tb)
		if pe != nil {
			return w.retryOrDeadLetterCT(ctx, job, pe)
		}
		// Skip the leaves at or below the cursor within this (possibly re-fetched head)
		// tile, and stop at the poll window's end. A tile that returns fewer leaves than
		// the cursor expects makes no progress: stop rather than spin.
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
				// One malformed entry does not fail the poll: the tail reads a
				// corroborative source, so it skips the entry and reads on.
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

// dataTileURL builds a static-ct-api data-tile request. A tile below the head is full and
// served at `tile/data/<path>`; the head tile is partial and served at
// `tile/data/<path>.p/<W>` with its current width W (1..255). The path segments come from
// scan.DataTilePath (§4.3).
func dataTileURL(base string, tileIdx, width int64) string {
	p := scan.DataTilePath(tileIdx)
	if width < scan.CTTileWidth {
		return fmt.Sprintf("%stile/data/%s.p/%d", base, p, width)
	}
	return fmt.Sprintf("%stile/data/%s", base, p)
}

// admitCTTail writes the completed Batch, its admissions, the advanced cursor and the
// job's done state in one transaction. Each admitted Name cites this Batch and
// terminates at its covering Seed (ADR-0027). The cursor advances to reached with the
// current signed head, so the next poll continues forward from here. There is no
// observation and no span-fold — the tail admits without observing. When a certificate
// appears for a Name the operator already knew, an ephemeral drift event rides the live
// stream — the count of such names only, never the names themselves (#780).
func (w *Worker) admitCTTail(ctx context.Context, job db.ClaimJobRow, lg scan.CTLog, signedHead []byte, reached int64, admissions []scan.CTAdmission) error {
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		// The set of Names the operator already knew, read BEFORE this batch's inserts:
		// the seed domains and every previously-admitted Name. A certificate for a Name
		// in this set is drift (new issuance for a tracked name); a Name not in it is a
		// first discovery, admitted but not drift.
		known, err := knownNameSet(ctx, qtx)
		if err != nil {
			return err
		}
		batchID, err := qtx.InsertBatch(ctx, db.InsertBatchParams{
			ScanID:        job.ScanID,
			DispatchID:    job.DispatchID,
			VantageID:     pgtype.Int8{}, // the ct-tail Scan has no vantage
			Kind:          job.Kind,
			Outcome:       "completed",
			Offers:        job.Offers,
			RecordedScope: job.AttemptedScope, // the log we polled, by content
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
		// Ephemeral per-job progress (#780): the admitted count, and — the tail's whole
		// point — the drift signal when new issuance lands on a known name.
		w.emitJobEvent(ctx, qtx, job, "", countLabel(len(admissions), "name admitted", "names admitted"))
		if drift > 0 {
			w.emitJobEvent(ctx, qtx, job, "warn", ctDriftLabel(drift))
		}
		return markDone(ctx, qtx, job.ID, batchID)
	})
}

// knownNameSet is the set of Names the operator already knows: the name-scope Seed
// domains and every distinct previously-admitted Name. Read inside the job tx before
// the batch's own inserts, so it reflects the pre-batch estate and the tail's re-poll of
// an already-seen name reads as drift rather than discovery.
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

// ctDriftLabel is the redacted text a drift event rides: a bare count of new
// certificates observed for names the operator already tracks — never the names.
func ctDriftLabel(n int) string {
	return fmt.Sprintf("%s for known names", countLabel(n, "new certificate", "new certificates"))
}

// ctHTTPCause turns a tail fetch outcome into a stream-safe failure cause. A transport
// error stays redacted to a generic phrase; a non-200 carries only the endpoint and the
// code — no source detail — so it is marked safe to surface verbatim (#780).
func ctHTTPCause(ferr error, status int, endpoint string) error {
	if ferr != nil {
		return ferr
	}
	return safeProgress(fmt.Sprintf("CT log %s returned HTTP %d", endpoint, status))
}

// ensureTrailingSlash normalises a log base URL to end in a slash so the RFC 6962
// endpoint paths (ct/v1/get-sth, ct/v1/get-entries) append cleanly. The embedded
// log_list.json already ends every url in a slash; this is defensive.
func ensureTrailingSlash(u string) string {
	if strings.HasSuffix(u, "/") {
		return u
	}
	return u + "/"
}

// getEntriesURL builds an RFC 6962 get-entries request for the inclusive position range
// [start, end]. The log returns the entries at those positions (possibly fewer, capped
// per operator), which the caller decodes and counts.
func getEntriesURL(base string, start, end int64) string {
	return fmt.Sprintf("%sct/v1/get-entries?start=%d&end=%d", base, start, end)
}

// fanOutCTTail enqueues one tail job per followed CT log, gated on the `ct-tail` source
// being enabled (§4.4). The source ships OFF (DefaultOn: false), so until the operator
// opts in this produces no jobs — a legible zero-job state, like `ct` with its source
// toggled off or `zone` with no file. The Scan is the Declared schedule and ships
// enabled; the source toggle is ADR-0003 consent. The log-set is the RFC 6962 logs from
// the embedded log_list.json that are readable and cover now (§4.3).
func (d *Dispatcher) fanOutCTTail(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
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

// enqueueCTTailJob enqueues one worker-read tail job for one log. Like the `ct` job it
// retries (MaxAttempts 5): a log poll is a network step with transient failure to back
// off from. The batch token keys the job to its log so concurrent fan-outs stay
// idempotent per (scan, log).
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
