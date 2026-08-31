package queue

import (
	"context"
	"encoding/json"
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

// The `ct` Scan is worker-read (ADR-0106): like `zone` there is no prober exec
// and no Vantage, but unlike every other Scan its source ADMITS WITHOUT OBSERVING
// (ADR-0027) — a completed Batch produces no observation, no span and no facet,
// only the set of `Name`s the certificates carry, each written as an
// `admitted_name` row citing the Batch. And unlike `zone` (a stored-file read
// that cannot fail transiently) the fetch is a network step against a source
// measured ~50% reliable (ADR-0027 §7), so it retries and dead-letters like the
// prober-exec path, and a non-200 admits nothing and never an absence (ADR-0005).
// This file holds the throttled fetcher, the throttle, the dispatcher's fan-out
// and the worker's completion path.

// maxCTBody bounds a crt.sh response read into memory. The documented 999-row cap
// already bounds the answer, but a defensive ceiling keeps a misbehaving or
// oversized response from exhausting the worker.
const maxCTBody = 64 << 20 // 64 MiB

// crtshInterval is the instance-wide spacing between crt.sh requests: 12s, the
// 5 req/min ceiling the operator asked for (ADR-0005, passive-discovery §2.2).
const crtshInterval = 12 * time.Second

// certSpotterInterval is the instance-wide spacing between Cert Spotter requests.
// It is sized to the free authenticated tier's documented cap — 10 full-domain
// queries per hour, i.e. one every 360s (research §3.3) — because the worker cannot
// know which paid tier the operator holds, so it spaces to the floor. A paid tier
// tolerates far more; #879 re-measures this against the reliability bar (spec §3).
const certSpotterInterval = 360 * time.Second

// maxCTPages bounds how many pages one CT job fetches for a single name-scope
// domain. crt.sh is single-shot (one page); Cert Spotter paginates by cursor until
// a page comes back empty. The cap is a backstop against a source that never
// returns an empty page (a non-advancing cursor is already caught by the
// next==cursor guard): a legitimate estate's per-domain issuance history sits far
// below it, so reaching it signals an oversized or misbehaving answer.
const maxCTPages = 1000

// CTFetcher fetches a crt.sh URL, returning the HTTP status and body. It is an
// injected seam so the worker is driven by a fake in tests and never touches
// crt.sh under test — the proposer/delivery Doer pattern, one level up (the whole
// fetch is faked, not just the transport).
type CTFetcher interface {
	Fetch(ctx context.Context, url string) (status int, body []byte, err error)
}

// HTTPCTFetcher is the production fetcher. It sends a distinctive User-Agent
// identifying verge-asm (the operator asked for one — passive-discovery §2.2) and
// bounds each attempt with a timeout wide enough for crt.sh's legitimately slow
// responses, measured up to ~60s (§7).
type HTTPCTFetcher struct {
	client    *http.Client
	userAgent string
	// bearer is the Authorization: Bearer credential sent with each request, or ""
	// for a keyless source (crt.sh sends none). Cert Spotter's operator key rides
	// here — worker-only (ADR-0053, spec §2.4), set through NewCertSpotterFetcher,
	// which is constructed only in the worker process.
	bearer string
}

// NewHTTPCTFetcher builds the production fetcher for a keyless source (crt.sh).
// version identifies the running build in the User-Agent, per the operator's
// request for an identifiable client.
func NewHTTPCTFetcher(version string) *HTTPCTFetcher {
	return &HTTPCTFetcher{
		client: &http.Client{
			Timeout: 90 * time.Second,
			// Redirects are not followed: a 3xx returns its own response unfollowed,
			// so a compromised or MITM'd source cannot bounce the fetch to an
			// arbitrary internal host (blind SSRF — IMDS at 169.254.169.254 or an
			// RFC-1918 address). The caller treats any non-200 as transient failure
			// and admits nothing, so an unfollowed 3xx is handled like any other
			// non-200 (ADR-0027 §7). Same idiom as httpexchange.NetExchanger.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		},
		userAgent: "verge-asm/" + version + " (+https://github.com/winniel123/verge-asm)",
	}
}

// NewCertSpotterFetcher builds the production fetcher for the Cert Spotter API,
// carrying the operator's API token as a Bearer credential (spec §2.4). It shares
// the crt.sh fetcher's no-redirect SSRF guard and timeout, adding only the auth
// header. The token stays worker-only: this constructor is called only in the
// worker process, so the web process never holds it.
func NewCertSpotterFetcher(version, token string) *HTTPCTFetcher {
	f := NewHTTPCTFetcher(version)
	f.bearer = token
	return f
}

// Fetch performs one GET against a crt.sh URL. A transport error returns status 0;
// the caller treats any non-200 — transport error or HTTP error alike — as
// transient failure, never as an empty result (ADR-0027 §7).
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

// CTThrottle reserves the next instant a crt.sh fetch may start, instance-wide.
// It is an interface so a test injects a no-op and a real run reserves a slot in
// Postgres (ADR-0005: the throttle is per-source across the whole instance, not
// worker memory).
type CTThrottle interface {
	Reserve(ctx context.Context) (time.Time, error)
}

// pgCTThrottle is the Postgres reservation throttle. Each Reserve atomically
// claims the next free slot in the source's own ct_throttle row, so
// `--scale worker=N` cannot exceed the ceiling. The row is keyed by source slug:
// each CT source reserves on its own interval, and today crt.sh is the only one.
type pgCTThrottle struct {
	q        *db.Queries
	source   string
	interval time.Duration
}

// NewCTThrottle builds the production throttle over pool for crt.sh at the 5
// req/min ceiling (12 s spacing, ADR-0005), the keyless fallback source.
func NewCTThrottle(q *db.Queries) CTThrottle {
	return pgCTThrottle{q: q, source: scan.CrtshSource, interval: crtshInterval}
}

// NewCertSpotterThrottle builds the production throttle for Cert Spotter on its own
// slug and interval (spec §2.5). It reserves against the `certspotter` row of the
// per-source ct_throttle table (seeded by migration 24100), separate from crt.sh's
// row, so each source spaces on its own cadence.
func NewCertSpotterThrottle(q *db.Queries) CTThrottle {
	return pgCTThrottle{q: q, source: scan.CertSpotterSource, interval: certSpotterInterval}
}

// Reserve claims the next slot for this source and returns the instant the fetch
// may start.
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

// WithCT wires the CT fetcher, throttle and selected bulk source onto the Worker.
// source is the one `ct` source active this config (spec §2.3): crt.sh when no
// operator key is set, Cert Spotter when it is. It decides the query URL, the
// response decode, and the slug stamped on admissions, so the fetcher and the
// source always agree. WithCT is separate from NewWorker so the measurement-only
// worker construction (and its tests) stay unchanged; a worker with no CT fetcher
// configured refuses a `ct` job rather than silently admitting nothing.
func (w *Worker) WithCT(fetcher CTFetcher, throttle CTThrottle, source scan.CTSource) *Worker {
	w.ctFetcher = fetcher
	w.ctThrottle = throttle
	w.ctSource = source
	return w
}

// completeCT is the worker-read path for a `ct` job: throttle, fetch, and either
// admit (on a well-formed 200) or retry/dead-letter (on anything else). It writes
// no observation and folds nothing into the Span corpus — CT admits without
// observing (ADR-0027).
func (w *Worker) completeCT(ctx context.Context, job db.ClaimJobRow, spec wire.JobSpec) error {
	if w.ctFetcher == nil {
		return fmt.Errorf("queue: ct job %d with no CT fetcher configured", job.ID)
	}
	src := w.ctSource
	if src == nil {
		// Default to crt.sh so a worker wired with a fetcher but no explicit source
		// (older construction, a test) still runs the keyless path unchanged.
		src = scan.CrtshCTSource()
	}
	cs, err := scan.CTScopeFromSpec(spec.Scope)
	if err != nil {
		return fmt.Errorf("decode ct scope: %w", err)
	}

	// Fetch the source's answer for this domain, following its pagination cursor.
	// crt.sh is single-shot (one page); Cert Spotter pages by cursor until a page
	// comes back empty. Every candidate feeds the shared admitter, which caps the
	// admission across all pages (spec §2.6). Each page reserves its own throttle
	// slot, so the per-source spacing bounds the whole paginated fetch.
	adm := scan.NewCTAdmitter(cs.Domain)
	cursor := ""
	// The reliability bar (spec §3, #879) measures each bulk-by-name query. fetchElapsed
	// accumulates the wall time on the wire across this query's pages, excluding the
	// throttle wait, so latency reflects the source and not our own spacing. sawAny
	// records whether the source returned any certificate name across all pages, so a
	// successful-but-zero query is recorded as the false-empty limb. One sample per
	// query attempt, so a retry is its own sample.
	var fetchElapsed time.Duration
	sawAny := false
	for page := 0; page < maxCTPages; page++ {
		// Reserve the next instance-wide slot and wait for it before going on the
		// wire (ADR-0005). The reservation is durable in Postgres; the wait is local.
		if w.ctThrottle != nil {
			slot, rerr := w.ctThrottle.Reserve(ctx)
			if rerr != nil {
				return fmt.Errorf("ct throttle: %w", rerr)
			}
			if serr := sleepUntil(ctx, w.now, slot); serr != nil {
				return serr
			}
		}

		fetchStart := w.now()
		status, body, ferr := w.ctFetcher.Fetch(ctx, src.QueryURL(cs.Domain, cursor))
		fetchElapsed += w.now().Sub(fetchStart)
		if ferr != nil || status != http.StatusOK {
			// Any non-200 is transient failure, never an empty result: a CT index
			// returns spurious 404s and 5xxs for domains that demonstrably have
			// certificates (ADR-0027 §7).
			w.recordCTSample(ctx, src.Slug(), false, fetchElapsed, false)
			cause := ferr
			if cause == nil {
				// A non-200 status carries no source detail — only the code — so it is
				// marked safe to surface verbatim in the live stream (#780, collision
				// #40). A transport error (ferr) is NOT marked and stays redacted.
				cause = safeProgress(fmt.Sprintf("%s returned HTTP %d", src.DisplayName(), status))
			}
			return w.retryOrDeadLetterCT(ctx, job, cause)
		}

		names, next, perr := src.DecodePage(body)
		if perr != nil {
			// A 200 whose body is not well-formed is not evidence of anything (§7) —
			// treat it as transient, not as "no certificates".
			w.recordCTSample(ctx, src.Slug(), false, fetchElapsed, false)
			return w.retryOrDeadLetterCT(ctx, job, perr)
		}
		// Count the raw candidate names this page carried as the admitter pulls them,
		// before the scope/wildcard/cap filter, so sawAny reflects whether the SOURCE
		// returned anything — a source empty, not a scope-filtered empty (spec §3).
		adm.Add(countingSeq(names, &sawAny))
		// Stop when the source signals no more pages (crt.sh always; Cert Spotter on
		// an empty page), when the cursor fails to advance, or once the admitter has
		// filled the cap and further pages could admit nothing more (#741).
		if next == "" || next == cursor || adm.Full() {
			break
		}
		cursor = next
	}
	// A well-formed 200 (or paginated run) is a successful sample; empty is true when
	// the source returned no certificate name at all — the false-empty limb (spec §3).
	w.recordCTSample(ctx, src.Slug(), true, fetchElapsed, !sawAny)

	if adm.Full() {
		// The admitter stopped at the ceiling: this domain carried at least
		// MaxAdmittedNames in-scope names and any beyond it were dropped rather than
		// admitted (#741). Legitimate estates sit far below the cap, so reaching it
		// signals an oversized or hostile answer worth an operator's notice.
		w.log.Printf("worker: ct job %d for %q reached the admitted-name cap of %d; names beyond the cap were dropped", job.ID, cs.Domain, scan.MaxAdmittedNames)
	}
	return w.admitCT(ctx, job, cs, src.Slug(), adm.Names())
}

// countingSeq wraps a candidate-name sequence to set *saw true the moment it yields
// its first name, without consuming the sequence twice: the admitter still pulls
// every name through it (spec §3, #879). It counts the SOURCE's raw output, before
// the scope/wildcard/cap filter, so a query that the source answered empty is told
// apart from one whose names the scope filter dropped — only the former is a
// false-empty.
func countingSeq(names iter.Seq[string], saw *bool) iter.Seq[string] {
	return func(yield func(string) bool) {
		for n := range names {
			*saw = true
			if !yield(n) {
				return
			}
		}
	}
}

// recordCTSample records one bulk-by-name query as a reliability sample and trims the
// source's window to CTReliabilityWindowSize (spec §3, #879). It is best-effort: a
// sample write that fails is logged and never fails the job, because the measurement
// must not change the outcome of the query it measures. It writes on w.q (the pool),
// not the job transaction, so the sample records the attempt independent of whether
// the job goes on to commit, retry, or dead-letter.
func (w *Worker) recordCTSample(ctx context.Context, source string, ok bool, latency time.Duration, empty bool) {
	if w.q == nil {
		return
	}
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

// retryOrDeadLetterCT enqueues a fresh attempt while the retry budget remains and
// dead-letters an empty scope once it is spent — the same rule the prober path
// applies, over the two transient-failure cases a CT fetch has (a non-200 and an
// unparseable 200). Either way the failed fetch admits nothing and asserts no
// absence (ADR-0005, ADR-0027 §7).
func (w *Worker) retryOrDeadLetterCT(ctx context.Context, job db.ClaimJobRow, cause error) error {
	if exhaustedRetries(job.Attempt, job.MaxAttempts) {
		return w.deadLetterCT(ctx, job, cause)
	}
	// A CT retry carries no transcript yet — the crt.sh producer capture is #870.
	// Until then it passes an absent transcript, so no row is written.
	return w.retry(ctx, job, nil, cause)
}

// admitCT writes the completed Batch, its admissions and the job's done state in
// one transaction. Each admitted Name cites this Batch and terminates at the
// covering Seed (ADR-0027). There is no observation and no span-fold — CT admits
// without observing, so a completed CT Batch moves no timeline.
func (w *Worker) admitCT(ctx context.Context, job db.ClaimJobRow, cs scan.CTSeed, source string, names []string) error {
	return w.runJobTx(ctx, job.ID, func(qtx *db.Queries) error {
		batchID, err := qtx.InsertBatch(ctx, db.InsertBatchParams{
			ScanID:        job.ScanID,
			DispatchID:    job.DispatchID,
			VantageID:     pgtype.Int8{}, // the ct Scan has no vantage
			Kind:          job.Kind,
			Outcome:       "completed",
			Offers:        job.Offers,
			RecordedScope: job.AttemptedScope, // the domain we queried, by content
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
		// Ephemeral per-job progress (#780, collision #40): a completed CT fetch rides the count
		// of names it admitted onto the live stream — the count alone, never the names.
		w.emitJobEvent(ctx, qtx, job, "", countLabel(len(names), "name admitted", "names admitted"))
		return markDone(ctx, qtx, job.ID, batchID)
	})
}

// deadLetterCT records a CT Batch whose scope is empty — never the queried domain
// — and marks the job dead, together. An empty scope licenses no absence: a failed
// fetch of an append-only, corroborative source says nothing about what exists
// (ADR-0005, ADR-0027).
func (w *Worker) deadLetterCT(ctx context.Context, job db.ClaimJobRow, cause error) error {
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
		// Ephemeral per-job progress (#780, collision #40): the redacted dead-letter reason —
		// the crt.sh non-200 marked safe above — rides the live stream so the operator sees why.
		w.emitJobEvent(ctx, qtx, job, "error", deadLetterLabel(job.Attempt, cause))
		return markDead(ctx, qtx, job.ID, batchID)
	})
}

// sleepUntil waits until t, or returns early if ctx is cancelled. now is the
// worker's injectable clock, so a test with a no-op throttle (slot == now) never
// sleeps.
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

// fanOutCT enqueues one CT job per name-scope Seed, gated on the selected CT
// source being enabled (ADR-0106). Exactly one CT source is active per config,
// selected at worker wire-time by the presence of the operator key (spec §2.3):
// crt.sh when absent, Cert Spotter when set. The gate consults only the selected
// source's slug, so the standby source never fans out even where its own
// source_state is on. The gate's ship-default is true because the selection has
// already happened — the operator toggles the selected source OFF in source_state
// to fire over an empty scope (a legible zero-job state, like `zone` with no file);
// there is no auto-fallback to the other source.
func (d *Dispatcher) fanOutCT(ctx context.Context, qtx *db.Queries, scanID, dispatchID int64) (int, error) {
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

// sourceEnabled reports whether the named source may run: its operator override
// from source_state where one exists, and the source's shipped default otherwise
// (crt.sh ships on — ADR-0003 unencumbered). It is per-slug so the `ct` fan-out
// consults it for the selected source, and a second CT source keys its own gate
// off the same table with no new read path. The read is confined to the
// dispatcher, where the CT scope is drawn (ADR-0106).
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

// ctSeeds reads the name-scope Seeds the CT Scan queries — id and registrable
// domain, one crt.sh query per Seed (ADR-0106).
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

// enqueueCTJob enqueues one worker-read CT job for one name-scope Seed. Unlike the
// zone job it retries (MaxAttempts 5, like the prober path): a crt.sh fetch has
// transient failure to back off from, where a stored-file read does not.
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
		ScanID:         scanID,
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
