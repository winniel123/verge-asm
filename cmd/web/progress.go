package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/winniel123/verge-asm/internal/queue"
)

// --- per-job live progress consumer (#780, SPEC-CHANGE collision #40 producer half) --------
//
// The worker emits ephemeral, redacted per-job progress events over the queue_job_progress
// LISTEN/NOTIFY channel (internal/queue). Web and worker are SEPARATE processes sharing only
// Postgres, so LISTEN/NOTIFY is the one liveness channel that crosses the boundary while
// persisting NOTHING at rest — pg_notify delivers a payload to connected listeners and is gone
// (ADR-0041's corpus separation and the instance-privacy posture stand). This file is the web
// end: a background goroutine LISTENs and appends each event to a bounded, in-memory,
// per-dispatch event log (the hub, lost on restart), and the RunDetail STREAM appends those
// events as live log lines beside the state-derived ones. On a job's conclusion the stream
// stops and the persisted state-derived .Log stands exactly as today — nothing here is written
// back to any store, and the page render's .Log (buildRunView) stays bare state.
//
// Why appended lines, not merged text: the frozen LogViewer (#771, rundetail.tmpl) is
// APPEND-ONLY — its client only ever adds lines at index ≥ its cursor and never re-renders a
// line in place. A dead-letter or completion mutates a job's queue_job row in place, so merging
// a reason onto that existing line would never reach a live viewer; delivering each event as a
// NEW appended line does. The stream's cursor is composite (see streamCursorBase) so the state
// lines and the event lines advance on independent counters — a retry growing the state log
// never shifts an event's position — while a run with no events keeps next == the state-line
// count, exactly the pre-existing transport contract.

// jobProgress mirrors the worker's wire event (internal/queue.jobProgress). The two packages
// keep separate types on purpose — the boundary is a JSON wire, and cmd/web already narrows its
// dependence on internal/db behind the `store` interface rather than sharing generated types —
// and the shared shape is pinned from BOTH ends: queue's TestJobProgressWire asserts the
// emitted tags, this package's TestDecodeProgress asserts they decode, so a drift breaks a test.
type jobProgress struct {
	Dispatch int64  `json:"dispatch"`
	Job      int64  `json:"job"`
	Level    string `json:"level"`
	Text     string `json:"text"`
}

type progressEvents interface {
	ForDispatch(dispatchID int64) []jobProgress
}

const (
	// maxProgressRuns bounds how many dispatches the hub retains events for at once. Events
	// matter only while a run is live and re-derive to bare state at any time, so a small ring
	// is ample; the least-recently-started run is evicted past the cap.
	maxProgressRuns = 256
	// maxEventsPerRun caps one run's event log. The stream's cursor is an index into this log,
	// so events are FROZEN at the cap (further events dropped) rather than evicted from the
	// front — dropping the front would shift every later index and desync the cursor. A real
	// run's event count is bounded by its job rows × transitions, far below this.
	maxEventsPerRun = 1024
)

type progressHub struct {
	mu     sync.Mutex
	byRun  map[int64][]jobProgress
	recent []int64 // dispatch ids in first-seen order, oldest first — the eviction ring
}

func newProgressHub() *progressHub {
	return &progressHub{byRun: map[int64][]jobProgress{}}
}

// record appends one event to its run's log, registering a new run in the eviction ring and
// evicting the oldest run past the cap. A malformed event with no dispatch/job is dropped, and
// a run already at maxEventsPerRun freezes rather than shifting its cursor indices.
func (h *progressHub) record(ev jobProgress) {
	if ev.Dispatch == 0 || ev.Job == 0 {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	log, ok := h.byRun[ev.Dispatch]
	if !ok {
		h.recent = append(h.recent, ev.Dispatch)
		h.evictLocked()
	}
	if len(log) >= maxEventsPerRun {
		return
	}
	h.byRun[ev.Dispatch] = append(log, ev)
}

func (h *progressHub) evictLocked() {
	for len(h.recent) > maxProgressRuns {
		oldest := h.recent[0]
		h.recent = h.recent[1:]
		delete(h.byRun, oldest)
	}
}

// ForDispatch returns a snapshot copy of a run's events in emit order, or nil when it has none.
// A copy so the stream reads it without holding the lock or racing record.
func (h *progressHub) ForDispatch(dispatchID int64) []jobProgress {
	h.mu.Lock()
	defer h.mu.Unlock()
	log, ok := h.byRun[dispatchID]
	if !ok || len(log) == 0 {
		return nil
	}
	out := make([]jobProgress, len(log))
	copy(out, log)
	return out
}

// decodeProgress parses one NOTIFY payload into an event, reporting ok=false for a malformed or
// incomplete payload so the LISTEN loop drops it rather than recording a phantom.
func decodeProgress(payload []byte) (jobProgress, bool) {
	var ev jobProgress
	if err := json.Unmarshal(payload, &ev); err != nil {
		return jobProgress{}, false
	}
	if ev.Dispatch == 0 || ev.Job == 0 {
		return jobProgress{}, false
	}
	return ev, true
}

// eventStreamLines turns a run's events into stream lines — the SAME tag/level/text shape a
// state line carries, so the frozen client renders them identically. With a numeric ?job the
// events are narrowed to that job, exactly as the state log is. It invents no format: the tag
// is the job's own "#<id>", the level and text are the redacted ones the worker emitted.
func eventStreamLines(events []jobProgress, jobFilter int64, filtered bool) []runStreamLine {
	out := make([]runStreamLine, 0, len(events))
	for _, ev := range events {
		if filtered && ev.Job != jobFilter {
			continue
		}
		out = append(out, runStreamLine{
			Tag:   "#" + strconv.FormatInt(ev.Job, 10),
			Level: ev.Level,
			Text:  ev.Text,
		})
	}
	return out
}

const streamCursorBase = 1_000_000

// encodeStreamCursor packs the two per-source counts into the one numeric cursor the frozen
// client echoes. State stays in the low part (see streamCursorBase); a runaway state count that
// somehow reached the base is clamped so the two counters never collide.
func encodeStreamCursor(events, state int) int {
	if state >= streamCursorBase {
		state = streamCursorBase - 1
	}
	return events*streamCursorBase + state
}

func decodeStreamCursor(after int) (events, state int) {
	if after < 0 {
		return 0, 0
	}
	return after / streamCursorBase, after % streamCursorBase
}

// runProgressListener subscribes to the worker's queue_job_progress channel and records each
// event into the hub, reconnecting with a bounded backoff on any connection loss. It is the one
// path that needs a live pgx connection, so it runs only in production (main.go wires it with
// the pool); tests drive the hub directly. It returns when ctx is done.
func runProgressListener(ctx context.Context, pool *pgxpool.Pool, hub *progressHub, logger *log.Logger) {
	const backoff = 2 * time.Second
	for ctx.Err() == nil {
		if err := listenProgressOnce(ctx, pool, hub); err != nil && ctx.Err() == nil {
			logger.Printf("web: progress listener: %v; retrying in %s", err, backoff)
			select {
			case <-ctx.Done():
			case <-time.After(backoff):
			}
		}
	}
}

// listenProgressOnce holds one LISTEN connection and drains notifications into the hub until the
// connection drops or ctx is done. A dropped connection returns its error so the caller
// reconnects; a missed notification is only a missed ephemeral line (the state-derived .Log
// still stands), so there is no catch-up read.
func listenProgressOnce(ctx context.Context, pool *pgxpool.Pool, hub *progressHub) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "LISTEN "+queue.ProgressChannel); err != nil {
		return err
	}
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if ev, ok := decodeProgress([]byte(n.Payload)); ok {
			hub.record(ev)
		}
	}
}
