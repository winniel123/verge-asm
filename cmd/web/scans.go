package main

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Scans monitor (#245, v1 spec §4.1): a read-only window onto the queue so an
// operator can see a scan is in flight and how far along it is, without querying
// Postgres directly. Dispatch, queue_job and batch are Operational — they record
// what the system did, never what is true of the estate — so this page reads them
// freely and the drift engine never sees this read (CONTEXT.md, ADR-0041).
//
// The monitor read is a viewer act. The on-demand trigger (#252, ask 3 of #245)
// is the one mutation this page hosts: an admin may dispatch an enabled scan now,
// and it appears in flight here. The trigger's handler, guardrails and panel live
// in scantrigger.go; scansPage only assembles the panel's read-side data.

// scansHistoryLimit bounds the Dispatch read. A Dispatch is one fan-out of one
// Scan, so recent history is short; 50 covers every enabled Scan's last several
// cadences without paging.
const scansHistoryLimit = 50

// dispatchView is one Dispatch shaped for the monitor: its Scan kind and tick, the
// per-state job counts folded into a completed / in-flight / total progress, and —
// for an in-flight Dispatch — the per-job detail.
type dispatchView struct {
	ID           int64
	ScanKind     string
	DispatchedAt string
	// Live is the count of jobs a retry has not superseded (total − retried).
	// Completed and InFlight partition it; Percent is completed / live.
	Live      int64
	Completed int64
	InFlight  int64
	Done      int64
	Dead      int64
	Percent   int
	// Active is true while any job is ready or running — the Dispatch is in flight.
	Active bool
	Jobs   []jobView
}

// jobView is one queue job in a Dispatch's drill-down: its kind, live state, the
// attempt it is on (attempt > 1 is a retry in progress), the Vantage it runs at
// where it has one, and its Batch outcome once terminal.
type jobView struct {
	ID          int64
	Kind        string
	State       string
	Attempt     int32
	MaxAttempts int32
	Retrying    bool
	Superseded  bool
	Vantage     string
	Batch       string
}

// scansPage renders the queue monitor: the Scans currently in flight with their
// per-job progress at the top, and recent completed Dispatches beneath as history.
// With nothing dispatched it shows an idle state — a fact and the next action —
// never an error or a blank. A viewer reads it; there is nothing to gate.
func (s *server) scansPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	rows, err := s.store.ListDispatchProgress(ctx, scansHistoryLimit)
	if err != nil {
		s.serverError(w, "scans: list dispatch progress", err)
		return
	}

	var active, history []dispatchView
	activeKinds := make(map[string]bool)
	for _, row := range rows {
		dv := toDispatchView(row)
		if dv.Active {
			activeKinds[row.ScanKind] = true
			jobs, err := s.store.ListJobsForDispatch(ctx, pgtype.Int8{Int64: row.DispatchID, Valid: true})
			if err != nil {
				s.serverError(w, "scans: list jobs for dispatch", err)
				return
			}
			for _, j := range jobs {
				dv.Jobs = append(dv.Jobs, toJobView(j))
			}
			active = append(active, dv)
		} else {
			history = append(history, dv)
		}
	}

	isAdmin := acct.Role == roleAdmin
	data := map[string]any{
		"Title": "Scans", "Account": acct, "IsAdmin": isAdmin,
		"Active": active, "History": history,
		// A meta refresh keeps the in-flight view current as jobs complete, since
		// the page is server-rendered with no client runtime. It runs only while a
		// scan is in flight, so the idle page does not spin.
		"Refresh": len(active) > 0,
	}

	// The admin on-demand trigger panel (#252) rides the same page as the monitor,
	// so pressing it and watching the result stay in one place. Its "in flight"
	// markers reuse the active kinds computed above rather than re-reading the
	// Dispatch corpus. It is built only for an admin — a viewer never sees the
	// panel (the template gates it on IsAdmin) so a viewer never pays its read —
	// and a failed build degrades to an absent panel rather than 500ing the whole
	// read-only monitor a viewer depends on.
	if isAdmin {
		if trigger, err := s.buildTriggerPanel(r, activeKinds); err != nil {
			log.Printf("web: scans: build trigger panel: %v", err)
		} else {
			data["Trigger"] = trigger
		}
	}

	s.render(w, "scans", data)
}

// toDispatchView folds one Dispatch's per-state job counts into progress. Live work
// excludes retried rows — a retry is a fresh job, so each retried row has a
// successor and counting it would double the denominator. Completed is the terminal
// outcomes (done + dead); a dead-lettered job is finished, not pending, so it counts
// toward progress. Percent is 0 while there is no live work rather than a divide.
func toDispatchView(row db.ListDispatchProgressRow) dispatchView {
	live := row.Total - row.Retried
	completed := row.Done + row.Dead
	inFlight := row.Ready + row.Running

	percent := 0
	if live > 0 {
		percent = int(completed * 100 / live)
	}

	dv := dispatchView{
		ID:        row.DispatchID,
		ScanKind:  row.ScanKind,
		Live:      live,
		Completed: completed,
		InFlight:  inFlight,
		Done:      row.Done,
		Dead:      row.Dead,
		Percent:   percent,
		Active:    inFlight > 0,
	}
	if row.CreatedAt.Valid {
		dv.DispatchedAt = row.CreatedAt.Time.UTC().Format("2006-01-02 15:04 UTC")
	}
	return dv
}

// toJobView shapes one queue job for the drill-down. A 'retried' row is a
// superseded attempt the fresh job replaced; a ready-or-running job past its first
// attempt is a retry currently in flight.
func toJobView(j db.ListJobsForDispatchRow) jobView {
	v := jobView{
		ID:          j.ID,
		Kind:        j.Kind,
		State:       j.State,
		Attempt:     j.Attempt,
		MaxAttempts: j.MaxAttempts,
		Superseded:  j.State == "retried",
		Retrying:    j.Attempt > 1 && (j.State == "ready" || j.State == "running"),
	}
	if j.VantageName.Valid {
		v.Vantage = j.VantageName.String
	}
	if j.BatchOutcome.Valid {
		v.Batch = j.BatchOutcome.String
	}
	return v
}
