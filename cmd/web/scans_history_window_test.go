package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
)

// busyMonitorStore seeds an admin plus `inFlight` in-flight dispatches and `concluded`
// completed ones, newest first, the order both monitor reads return. The in-flight ids
// run above the concluded ones, so a shared newest-first window would list the in-flight
// burst and evict the completed rows — the bug #962 removes.
func busyMonitorStore(t *testing.T, inFlight, concluded int) *fakeStore {
	t.Helper()
	f := newFakeStore()
	seedAccount(t, f, "admin", roleAdmin, "hunter2hunter2")
	tick := time.Date(2026, 8, 22, 14, 0, 0, 0, time.UTC)
	rows := make([]db.ListDispatchProgressRow, 0, inFlight+concluded)
	for i := range inFlight {
		id := int64(9000 - i)
		rows = append(rows, progressRow(id, "standard", tick, 4, 1, 1, 2, 0, 0))
	}
	for i := range concluded {
		id := int64(1000 - i)
		rows = append(rows, progressRow(id, "dns", tick, 3, 0, 0, 3, 0, 0))
	}
	f.dispatchProgress = rows
	return f
}

// The history window is dedicated (#962, SPEC §3): a burst of in-flight dispatches far
// past the old shared 50-row read no longer evicts a single completed row.
func TestScansHistorySurvivesBusyQueue(t *testing.T) {
	f := busyMonitorStore(t, 60, 3)
	base := startWithTrigger(t, f, &fakeTrigger{})
	ac := login(t, base, "admin", "hunter2hunter2")
	page := getBody(t, ac, base+"/scans", http.StatusOK)

	for _, id := range []int64{1000, 999, 998} {
		if !strings.Contains(page, fmt.Sprintf(`href="/runs/%d"`, id)) {
			t.Errorf("completed dispatch %d evicted from history by the in-flight burst", id)
		}
	}
	if strings.Contains(page, "No dispatches yet") {
		t.Errorf("history rendered its empty state with three completed dispatches; body: %s", page)
	}
	// A listed row must resolve. Run detail reads the same two windows the monitor lists
	// from, so a history row a busy queue pushed past the old shared 50 still has a page,
	// and so does an in-flight dispatch deep in the uncapped Active read. renderMissingRun
	// answers 404, so the status assertion alone catches an unresolvable row.
	for _, id := range []int64{1000, 998, 8941} {
		getBody(t, ac, fmt.Sprintf("%s/runs/%d", base, id), http.StatusOK)
	}
}

// Truncation is detected with LIMIT N+1: the read fetches scansHistoryLimit + 1 rows,
// the page shows scansHistoryLimit, and the extra row raises the callout (#962, SPEC §3).
func TestScansHistoryTruncationCallout(t *testing.T) {
	want := "Showing the " + strconv.Itoa(scansHistoryLimit) + " most recent dispatches."
	cases := []struct {
		name      string
		concluded int
		truncated bool
	}{
		{"at the cap", scansHistoryLimit, false},
		{"one past the cap", scansHistoryLimit + 1, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := busyMonitorStore(t, 0, tc.concluded)
			base := startWithTrigger(t, f, &fakeTrigger{})
			ac := login(t, base, "admin", "hunter2hunter2")
			page := getBody(t, ac, base+"/scans", http.StatusOK)

			if got := strings.Contains(page, want); got != tc.truncated {
				t.Errorf("callout present = %v, want %v with %d concluded dispatches",
					got, tc.truncated, tc.concluded)
			}
			// Visible depth stays exactly the cap: the extra row is the signal, never a row.
			if n := strings.Count(page, `href="/runs/`); n != scansHistoryLimit {
				t.Errorf("history rendered %d rows, want %d", n, scansHistoryLimit)
			}
			// The callout is stateless — no link and no pager, so the meta refresh
			// re-renders it identically.
			if strings.Contains(page, "?page=") {
				t.Errorf("history grew a pager; body: %s", page)
			}
		})
	}
}

// The stop / terminate dialogs and their POSTs look their target up in the uncapped
// Active read (#962), so a dispatch in flight behind a long queue still ends. Under the
// old shared 50-row read, id 8941 sat past the window and the act refused it.
func TestScansStopTerminateReachPastTheOldWindow(t *testing.T) {
	const deep = 8941 // the 60th of 60 in-flight dispatches seeded newest-first

	t.Run("dialogs", func(t *testing.T) {
		f := busyMonitorStore(t, 60, 0)
		base := startWithTrigger(t, f, &fakeTrigger{})
		ac := login(t, base, "admin", "hunter2hunter2")

		stopPage := getBody(t, ac, fmt.Sprintf("%s/settings?tab=scans&stop=%d", base, deep), http.StatusOK)
		if !strings.Contains(stopPage, `action="/scans/stop"`) {
			t.Errorf("stop dialog missing for a deeply queued in-flight dispatch")
		}
		termPage := getBody(t, ac, fmt.Sprintf("%s/settings?tab=scans&terminate=%d", base, deep), http.StatusOK)
		if !strings.Contains(termPage, `action="/scans/terminate"`) {
			t.Errorf("terminate dialog missing for a deeply queued in-flight dispatch")
		}
	})

	for _, act := range []struct{ path, status string }{
		{"/scans/stop", "stopped"},
		{"/scans/terminate", "terminated"},
	} {
		t.Run(act.status, func(t *testing.T) {
			f := busyMonitorStore(t, 60, 0)
			base := startWithTrigger(t, f, &fakeTrigger{})
			ac := login(t, base, "admin", "hunter2hunter2")

			resp := postForm(t, ac, base+act.path, url.Values{"id": {strconv.Itoa(deep)}})
			resp.Body.Close()
			if resp.StatusCode != http.StatusSeeOther {
				t.Fatalf("%s: status = %d, want 303", act.path, resp.StatusCode)
			}
			if got := f.dispatchStatus[deep]; got != act.status {
				t.Errorf("%s: dispatch status = %q, want %q", act.path, got, act.status)
			}
		})
	}
}
