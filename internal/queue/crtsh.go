package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/scan"
	"github.com/winniel123/verge-asm/internal/wire"
)

const maxCTBody = 64 << 20

// Sectigo publicly asked for 5 req/min on crt.sh, and 12s is that spacing (ADR-0005, §2.2).

const crtshInterval = 12 * time.Second

// The free tier allows ten full-domain queries an hour, and the key's tier is unknowable (§2.3).

const certSpotterInterval = 360 * time.Second

// A source that never signals a last page would page forever, so this is a backstop, not a budget.

const maxCTPages = 1000

type CTFetcher interface {
	Fetch(ctx context.Context, url string) (status int, body []byte, err error)
}

// crt.sh asks each client to identify itself by User-Agent (passive-discovery §2.2).

type HTTPCTFetcher struct {
	client    *http.Client
	userAgent string
	bearer    string
}

func NewHTTPCTFetcher(version string) *HTTPCTFetcher {
	return &HTTPCTFetcher{
		client: &http.Client{
			// crt.sh answers legitimately slowly, measured up to 59.6s (passive-discovery §7).
			Timeout: 90 * time.Second,
			// A 3xx could bounce the fetch to an internal host such as IMDS, so no redirect is followed (ADR-0196 §1).
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		userAgent: "verge-asm/" + version + " (+https://github.com/winniel123/verge-asm)",
	}
}

func NewCertSpotterFetcher(version, token string) *HTTPCTFetcher {
	// A secret is held only where its act is performed, so only the worker builds this (ADR-0053).
	f := NewHTTPCTFetcher(version)
	f.bearer = token
	return f
}

func (f *HTTPCTFetcher) Fetch(ctx context.Context, url string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", f.userAgent)
	req.Header.Set("Accept", "application/json")
	if f.bearer != "" {
		req.Header.Set("Authorization", "Bearer "+f.bearer)
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxCTBody))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

type CTThrottle interface {
	Reserve(ctx context.Context) (time.Time, error)
}

// A durable per-source reservation is why --scale worker=N cannot exceed the ceiling (ADR-0005).

type pgCTThrottle struct {
	q        *db.Queries
	source   string
	interval time.Duration
}

func NewCTThrottle(q *db.Queries) CTThrottle {
	return pgCTThrottle{q: q, source: scan.CrtshSource, interval: crtshInterval}
}

func NewCertSpotterThrottle(q *db.Queries) CTThrottle {
	return pgCTThrottle{q: q, source: scan.CertSpotterSource, interval: certSpotterInterval}
}

func (t pgCTThrottle) Reserve(ctx context.Context) (time.Time, error) {
	slot, err := t.q.ReserveCTSlot(ctx, db.ReserveCTSlotParams{
		Source:          t.source,
		IntervalSeconds: t.interval.Seconds(),
	})
	if err != nil {
		return time.Time{}, err
	}
	return slot.Time, nil
}

func (w *Worker) WithCT(fetcher CTFetcher, throttle CTThrottle, source scan.CTSource) *Worker {
	w.ctFetcher = fetcher
	w.ctThrottle = throttle
	w.ctSource = source
	return w
}

func (w *Worker) completeCT(ctx context.Context, job db.ClaimJobRow, spec wire.JobSpec) error {
	if w.ctFetcher == nil {
		return fmt.Errorf("queue: ct job %d with no CT fetcher configured", job.ID)
	}
	src := w.ctSource
	if src == nil {
		src = scan.CrtshCTSource()
	}
	cs, err := scan.CTScopeFromSpec(spec.Scope)
	if err != nil {
		return fmt.Errorf("decode ct scope: %w", err)
	}
	start := w.now()

	adm := scan.NewCTAdmitter(cs.Domain)
	cursor := ""
	var lastURL string
	var lastBody []byte
	var fetchElapsed time.Duration
	sawAny := false
	for page := 0; page < maxCTPages; page++ {
		if w.ctThrottle != nil {
			slot, rerr := w.ctThrottle.Reserve(ctx)
			if rerr != nil {
				return fmt.Errorf("ct throttle: %w", rerr)
			}
			if serr := sleepUntil(ctx, w.now, slot); serr != nil {
				return serr
			}
		}

		url := src.QueryURL(cs.Domain, cursor)
		// Latency must measure the source, not our own spacing, so the wait sits outside it (§3).
		fetchStart := w.now()
		status, body, ferr := w.ctFetcher.Fetch(ctx, url)
		fetchElapsed += w.now().Sub(fetchStart)
		lastURL, lastBody = url, body
		if ferr != nil || status != http.StatusOK {
			// crt.sh returns spurious 404s and 5xxs for domains that do have certificates (§2.2).
			w.recordCTSample(ctx, src.Slug(), false, fetchElapsed, false)
			cause := ferr
			if cause == nil {
				// A status code leaks nothing, unlike a transport error's text, so only this is safe (#780).
				cause = safeProgress(fmt.Sprintf("%s returned HTTP %d", src.DisplayName(), status))
			}
			t := w.buildCTTranscript(job.Kind, start, url, body, ctFetchOutcome(ferr, status))
			return w.retryOrDeadLetterCT(ctx, job, t, cause)
		}

		names, next, perr := src.DecodePage(body)
		if perr != nil {
			w.recordCTSample(ctx, src.Slug(), false, fetchElapsed, false)
			t := w.buildCTTranscript(job.Kind, start, url, body, wire.CTHTTP{Status: status})
			return w.retryOrDeadLetterCT(ctx, job, t, perr)
		}
		adm.Add(countingSeq(names, &sawAny))
		if next == "" || next == cursor || adm.Full() {
			break
		}
		cursor = next
	}
	w.recordCTSample(ctx, src.Slug(), true, fetchElapsed, !sawAny)

	if adm.Full() {
		w.log.Printf("worker: ct job %d for %q reached the admitted-name cap of %d; names beyond the cap were dropped", job.ID, cs.Domain, scan.MaxAdmittedNames)
	}
	t := w.buildCTTranscript(job.Kind, start, lastURL, lastBody, wire.CTHTTP{Status: http.StatusOK})
	return w.admitCT(ctx, job, cs, src.Slug(), adm.Names(), t)
}

func (w *Worker) buildCTTranscript(kind string, start time.Time, url string, body []byte, outcome wire.CTOutcome) wire.Transcript {
	if !w.captureOn() {
		return nil
	}
	return wire.CTTranscript{
		TranscriptFrame: wire.TranscriptFrame{Kind: kind, Duration: w.now().Sub(start)},
		RequestURL:      url,
		ResponseBody:    body,
		Outcome:         outcome,
	}
}

func ctFetchOutcome(ferr error, status int) wire.CTOutcome {
	switch {
	case errors.Is(ferr, context.Canceled) || errors.Is(ferr, context.DeadlineExceeded):
		return wire.CTContextCancelled{}
	case ferr != nil:
		return wire.CTTransportError{Text: ferr.Error()}
	default:
		return wire.CTHTTP{Status: status}
	}
}

func countingSeq(names iter.Seq[string], saw *bool) iter.Seq[string] {
	// The count is the source's raw output, so a source-empty is told from a scope-filtered one (§3).
	return func(yield func(string) bool) {
		for n := range names {
			*saw = true
			if !yield(n) {
				return
			}
		}
	}
}

func (w *Worker) recordCTSample(ctx context.Context, source string, ok bool, latency time.Duration, empty bool) {
	// The sample rides the pool, not the job transaction, so a retry still records its attempt (§3).
	if w.q == nil {
		return
	}
	// A measurement must not change the outcome it measures, so a failed write only logs (§3).
	if err := w.q.InsertCTReliabilitySample(ctx, db.InsertCTReliabilitySampleParams{
		Source:    source,
		Ok:        ok,
		LatencyMs: latency.Milliseconds(),
		Empty:     empty,
	}); err != nil {
		w.log.Printf("worker: ct reliability sample for %q: %v", source, err)
		return
	}
	if err := w.q.TrimCTReliabilitySamples(ctx, db.TrimCTReliabilitySamplesParams{
		Source:    source,
		KeepCount: scan.CTReliabilityWindowSize,
	}); err != nil {
		w.log.Printf("worker: ct reliability trim for %q: %v", source, err)
	}
}

func (w *Worker) retryOrDeadLetterCT(ctx context.Context, job db.ClaimJobRow, t wire.Transcript, cause error) error {
	if exhaustedRetries(job.Attempt, job.MaxAttempts) {
		return w.deadLetterCT(ctx, job, t, cause)
	}
	// A Transcript keys to the attempt that made it, so the failed exchange rides this row (§1.1).
	return w.retry(ctx, job, t, cause)
}

func (w *Worker) admitCT(ctx context.Context, job db.ClaimJobRow, cs scan.CTSeed, source string, names []string, t wire.Transcript) error {
	// A CT source admits without observing, so a completed Batch folds no span (ADR-0027).
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
		for _, n := range names {
			if err := qtx.InsertAdmittedName(ctx, db.InsertAdmittedNameParams{
				Name:    n,
				Source:  source,
				SeedID:  cs.SeedID,
				BatchID: batchID,
			}); err != nil {
				return err
			}
		}
		w.emitJobEvent(ctx, qtx, job, "", countLabel(len(names), "name admitted", "names admitted"))
		if err := w.persistTranscript(ctx, qtx, job.ID, t); err != nil {
			return err
		}
		return markDone(ctx, qtx, job.ID, batchID)
	})
}

func (w *Worker) deadLetterCT(ctx context.Context, job db.ClaimJobRow, t wire.Transcript, cause error) error {
	// A failed fetch of a corroborative source asserts no absence, so the scope is empty (ADR-0005).
	w.log.Printf("worker: ct job %d dead-lettered after %d attempts: %v", job.ID, job.Attempt, cause)
	empty, err := scan.EmptyCTScope()
	if err != nil {
		return err
	}
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		batchID, err := qtx.InsertBatch(ctx, db.InsertBatchParams{
			ScanID:        job.ScanID,
			DispatchID:    job.DispatchID,
			VantageID:     pgtype.Int8{},
			Kind:          job.Kind,
			Outcome:       "dead-lettered",
			Offers:        job.Offers,
			RecordedScope: empty,
		})
		if err != nil {
			return err
		}
		w.emitJobEvent(ctx, qtx, job, "error", deadLetterLabel(job.Attempt, cause))
		if err := w.persistTranscript(ctx, qtx, job.ID, t); err != nil {
			return err
		}
		return markDead(ctx, qtx, job.ID, batchID)
	})
}

func sleepUntil(ctx context.Context, now func() time.Time, t time.Time) error {
	d := t.Sub(now())
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (d *Dispatcher) fanOutCT(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
	// Toggling the selected source off fires over an empty scope; there is no fallback (ADR-0003).
	enabled, err := sourceEnabled(ctx, qtx, d.selectedCTSource(), true)
	if err != nil {
		return 0, err
	}
	if !enabled {
		return 0, nil
	}
	seeds, err := ctSeeds(ctx, qtx)
	if err != nil {
		return 0, err
	}
	enqueued := 0
	for _, j := range scan.BuildCTJobs(scanID, seeds) {
		if err := enqueueCTJob(ctx, qtx, scanID, dispatchID, j); err != nil {
			return 0, err
		}
		enqueued++
	}
	return enqueued, nil
}

func sourceEnabled(ctx context.Context, q *db.Queries, slug string, shipDefault bool) (bool, error) {
	states, err := q.ListSourceStates(ctx)
	if err != nil {
		return false, err
	}
	for _, s := range states {
		if s.Slug == slug {
			return s.Enabled, nil
		}
	}
	return shipDefault, nil
}

func ctSeeds(ctx context.Context, q *db.Queries) ([]scan.CTSeed, error) {
	rows, err := q.ListNameSeeds(ctx)
	if err != nil {
		return nil, err
	}
	seeds := make([]scan.CTSeed, 0, len(rows))
	for _, r := range rows {
		if r.NameDomain.Valid {
			seeds = append(seeds, scan.CTSeed{SeedID: r.ID, Domain: r.NameDomain.String})
		}
	}
	return seeds, nil
}

func enqueueCTJob(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64, j scan.CTJob) error {
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
		ScanID: scanID,
		// A ct job runs no prober exec, so it rides no Vantage (ADR-0106).
		VantageID:      pgtype.Int8{},
		DispatchID:     pgInt8(dispatchID),
		Kind:           scan.CTKind,
		Spec:           specJSON,
		AttemptedScope: scopeJSON,
		Offers:         []byte("{}"),
		Attempt:        1,
		MaxAttempts:    5,
		RunAfter:       tstz(time.Now().UTC()),
	})
	return err
}
