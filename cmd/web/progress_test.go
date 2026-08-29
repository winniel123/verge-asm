package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// --- per-job live progress consumer (#780, collision #40) ---------------------------------

// eventStreamLines turns a run's ephemeral events into wire lines carrying the job's own tag
// and the redacted level/text, in emit order. A numeric ?job narrows them to that job; the
// unfiltered case keeps them all.
func TestEventStreamLines(t *testing.T) {
	events := []jobProgress{
		{Dispatch: 5, Job: 701, Level: "warn", Text: "attempt 1 failed · crt.sh returned HTTP 502 · retrying"},
		{Dispatch: 5, Job: 702, Level: "error", Text: "dead-lettered after 5 attempts · crt.sh returned HTTP 502"},
		{Dispatch: 5, Job: 701, Text: "12 observations"},
	}

	// Unfiltered: every event, in order, with the job's tag.
	all := eventStreamLines(events, 0, false)
	if len(all) != 3 {
		t.Fatalf("unfiltered should keep every event: %+v", all)
	}
	if all[0].Tag != "#701" || all[0].Level != "warn" || all[0].Text != "attempt 1 failed · crt.sh returned HTTP 502 · retrying" {
		t.Errorf("first event line wrong: %+v", all[0])
	}
	if all[1].Tag != "#702" || all[1].Level != "error" {
		t.Errorf("dead-letter event line wrong: %+v", all[1])
	}

	// Filtered to job 701: only its two events, order preserved.
	only := eventStreamLines(events, 701, true)
	if len(only) != 2 || only[0].Tag != "#701" || only[1].Text != "12 observations" {
		t.Errorf("job filter wrong: %+v", only)
	}
	if got := eventStreamLines(events, 999, true); len(got) != 0 {
		t.Errorf("unknown job filter should keep nothing: %+v", got)
	}
	if got := eventStreamLines(nil, 0, false); len(got) != 0 {
		t.Errorf("no events should yield no lines: %+v", got)
	}
}

// The composite cursor packs (events, state) into one number: state in the low part so a
// first-poll cursor (the bare state-line count, events 0) round-trips, and events in the high
// part so the two advance independently. Encode/decode are inverses across the range.
func TestStreamCursor(t *testing.T) {
	cases := []struct{ events, state int }{{0, 0}, {0, 7}, {3, 2}, {1, 0}, {5, 999}}
	for _, c := range cases {
		e, s := decodeStreamCursor(encodeStreamCursor(c.events, c.state))
		if e != c.events || s != c.state {
			t.Errorf("round-trip (%d,%d) → (%d,%d)", c.events, c.state, e, s)
		}
	}
	// A bare state count (no events) encodes to itself — the pre-producer contract the frozen
	// client's initial cursor relies on.
	if got := encodeStreamCursor(0, 7); got != 7 {
		t.Errorf("state-only cursor must equal the state count: %d", got)
	}
	// A negative cursor is treated as the origin, not a panic.
	if e, s := decodeStreamCursor(-1); e != 0 || s != 0 {
		t.Errorf("negative cursor should decode to origin: (%d,%d)", e, s)
	}
}

// decodeProgress accepts a well-formed payload and rejects malformed or incomplete ones so the
// LISTEN loop never records a phantom.
func TestDecodeProgress(t *testing.T) {
	ev, ok := decodeProgress([]byte(`{"dispatch":5,"job":701,"level":"warn","text":"retrying"}`))
	if !ok || ev.Dispatch != 5 || ev.Job != 701 || ev.Level != "warn" || ev.Text != "retrying" {
		t.Fatalf("valid payload: ok=%v ev=%+v", ok, ev)
	}
	for _, bad := range []string{
		`not json`,
		`{"dispatch":5}`,         // no job
		`{"job":701}`,            // no dispatch
		`{"dispatch":0,"job":0}`, // both zero
	} {
		if _, ok := decodeProgress([]byte(bad)); ok {
			t.Errorf("malformed payload accepted: %s", bad)
		}
	}
}

// The hub appends events per dispatch in emit order, returns a snapshot copy, freezes a run at
// the per-run cap (rather than shifting cursor indices), and evicts the least-recently-started
// run past the run cap so it cannot grow without bound.
func TestProgressHub(t *testing.T) {
	h := newProgressHub()

	// A malformed event (no dispatch/job) is dropped.
	h.record(jobProgress{Text: "orphan"})
	if h.ForDispatch(0) != nil {
		t.Error("orphan event should not be recorded")
	}

	// Order preserved; a job's later event is a NEW line, not a supersession.
	h.record(jobProgress{Dispatch: 5, Job: 701, Text: "attempt 1 failed"})
	h.record(jobProgress{Dispatch: 5, Job: 702, Text: "other"})
	h.record(jobProgress{Dispatch: 5, Job: 701, Text: "dead-lettered"})
	got := h.ForDispatch(5)
	if len(got) != 3 || got[0].Text != "attempt 1 failed" || got[1].Job != 702 || got[2].Text != "dead-lettered" {
		t.Fatalf("append order wrong: %+v", got)
	}
	// The returned slice is a copy: mutating it does not affect the hub.
	got[0] = jobProgress{Text: "mutated"}
	if h.ForDispatch(5)[0].Text != "attempt 1 failed" {
		t.Error("ForDispatch must return a snapshot copy")
	}

	// Per-run cap: further events are dropped (frozen), keeping the cursor indices stable.
	h2 := newProgressHub()
	for i := 0; i < maxEventsPerRun+50; i++ {
		h2.record(jobProgress{Dispatch: 9, Job: 1, Text: "x"})
	}
	if n := len(h2.ForDispatch(9)); n != maxEventsPerRun {
		t.Errorf("per-run cap: got %d events, want %d", n, maxEventsPerRun)
	}

	// Run eviction: start more runs than the cap; the earliest is gone, the newest retained.
	for i := int64(1); i <= maxProgressRuns+10; i++ {
		h.record(jobProgress{Dispatch: 1000 + i, Job: 1, Text: "x"})
	}
	if h.ForDispatch(1001) != nil {
		t.Error("oldest run past the cap should be evicted")
	}
	if h.ForDispatch(1000+maxProgressRuns+10) == nil {
		t.Error("newest run should be retained")
	}
}

// fakeProgress is a hub stand-in for the HTTP-level stream test: it returns a fixed event list.
type fakeProgress struct {
	byRun map[int64][]jobProgress
}

func (f fakeProgress) ForDispatch(id int64) []jobProgress { return f.byRun[id] }

// End-to-end: with the hub wired, the stream APPENDS the run's ephemeral event lines after its
// state lines — the crt.sh-502 retry reason and the dead-letter cause reach the append-only
// viewer as new lines — while the transport contract ({lines,next,done}) holds. A retry event
// carries the job's tag and warn level; a run with no hub (every other stream test) appends
// nothing, so the enrichment is purely additive.
func TestRunStreamEnrichedByHub(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(90, "ct", tick, 2, 1, 0, 1, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		90: {
			{ID: 900, Kind: "ct", State: "dead", Attempt: 5, MaxAttempts: 5},
			{ID: 901, Kind: "ct", State: "running", Attempt: 1, MaxAttempts: 5},
		},
	}

	srv := newServer(f, testKey, "", fixedClock())
	srv.progress = fakeProgress{byRun: map[int64][]jobProgress{
		90: {
			{Dispatch: 90, Job: 900, Level: "warn", Text: "attempt 4 failed · crt.sh returned HTTP 502 · retrying"},
			{Dispatch: 90, Job: 900, Level: "error", Text: "dead-lettered after 5 attempts · crt.sh returned HTTP 502"},
		},
	}}
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)

	ac := login(t, ts.URL, "admin", "hunter2hunter2")
	// From cursor 0: 2 state lines then 2 appended event lines.
	got := getStream(t, ac, ts.URL+"/run/90/stream?after=0")
	if len(got.Lines) != 4 {
		t.Fatalf("expected 2 state + 2 event lines, got %+v", got)
	}
	// The event lines carry the redacted reasons the append-only viewer can render live.
	if got.Lines[2].Tag != "#900" || !strings.Contains(got.Lines[2].Text, "crt.sh returned HTTP 502") || got.Lines[2].Level != "warn" {
		t.Errorf("retry event line wrong: %+v", got.Lines[2])
	}
	if got.Lines[3].Level != "error" || !strings.Contains(got.Lines[3].Text, "dead-lettered after 5 attempts") {
		t.Errorf("dead-letter event line wrong: %+v", got.Lines[3])
	}
	// The state lines stay bare — the reasons ride only the appended event lines.
	if strings.Contains(got.Lines[0].Text, "crt.sh") || strings.Contains(got.Lines[1].Text, "crt.sh") {
		t.Errorf("state lines must stay bare state: %+v", got.Lines[:2])
	}

	// The cursor round-trips: re-polling at next returns nothing new (running job → not done).
	got2 := getStream(t, ac, ts.URL+"/run/90/stream?after="+strconv.Itoa(got.Next))
	if len(got2.Lines) != 0 || got2.Done {
		t.Errorf("re-poll at next should be empty and not done: %+v", got2)
	}
}

// The hub enrichment does NOT leak into the page-render .Log: buildRunView's log stays bare
// state, so on conclusion the persisted state-derived log stands (nothing new at rest).
func TestPageLogStaysBareState(t *testing.T) {
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")

	tick := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	f.dispatchProgress = []db.ListDispatchProgressRow{
		progressRow(91, "ct", tick, 1, 1, 0, 0, 0, 0),
	}
	f.jobsByDispatch = map[int64][]db.ListJobsForDispatchRow{
		91: {{ID: 910, Kind: "ct", State: "ready", Attempt: 1, MaxAttempts: 5,
			VantageName: pgtype.Text{}}},
	}

	srv := newServer(f, testKey, "", fixedClock())
	srv.progress = fakeProgress{byRun: map[int64][]jobProgress{
		91: {{Dispatch: 91, Job: 910, Text: "should not appear on the page"}},
	}}
	ts := httptest.NewServer(srv.handler())
	t.Cleanup(ts.Close)

	ac := login(t, ts.URL, "admin", "hunter2hunter2")
	page := getBody(t, ac, ts.URL+"/run/91", http.StatusOK)
	if strings.Contains(page, "should not appear on the page") {
		t.Error("page-render .Log must stay bare state — hub enrichment leaked into the static log")
	}
}
