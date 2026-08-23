package main

import (
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
)

// The Drift screen (#283, ADR-0108/ADR-0110) — the canonical `/drift`, nav item 4
// of 7 and the product's thesis: what moved since last time, grouped by batch, in
// change's own language (appeared / revealed / withdrawn / descoped / returned /
// changed) rather than the severity ramp. The composition is ported verbatim from
// design-system/examples/console/Drift.jsx — the change-kind legend, the two-column
// "By batch" transitions timeline beside a "Movement" summary, the per-event diff
// affordance.
//
// The transition feed is estate-wide and batch-grouped (#288, ADR-0111). A
// transition is a span open/close event, derived on read (ADR-0007); the store's
// ListRecentDriftEvents returns those events across the estate for a period, each
// citing the Batch that caused it, and buildDriftFeed classifies them into the six
// change kinds and groups them by batch. The change vocabulary itself (the legend)
// is definitional, not data, so it always renders even before a second batch has
// folded a value that moved. No change event is ever fabricated (the ported
// example's sample transitions are dropped, not carried): the timeline renders the
// design-system empty-state until the feed is non-empty.

// driftFamily is the drift palette a change kind rides — the `.chip` modifier and
// the `--drift-<family>-*` token triple. Change is its own language: never the
// severity ramp. Mirrors ChangeBadge.jsx's FAMILY map exactly.
func driftFamily(change string) string {
	switch change {
	case "appeared", "revealed", "returned":
		return "gain" // violet — a value entered or widened into sight
	case "withdrawn", "descoped":
		return "loss" // slate — a value left sight
	default:
		return "change" // magenta — a held value moved (changed)
	}
}

// driftKind is one entry of the change vocabulary, shaped for the legend row: the
// kind word and the drift family (chip class) it rides.
type driftKind struct {
	Change string
	Family string
}

// driftKinds is the fixed change vocabulary in the example's order. It is the
// language of drift, not a data read — the legend renders it whether or not any
// transition has yet been folded.
func driftKinds() []driftKind {
	kinds := []string{"appeared", "revealed", "withdrawn", "descoped", "returned", "changed"}
	out := make([]driftKind, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, driftKind{Change: k, Family: driftFamily(k)})
	}
	return out
}

// driftDiffLine is one line of a transition's before/after diff — a removal (the
// prior value, danger-red, a true minus), an addition (the new value, ok-green), or
// an unchanged context line. Always mono.
type driftDiffLine struct {
	Type string // "add" | "remove" | "same"
	Text string
}

// driftEvent is one transition in a batch: its change kind (carrying its drift
// family), the subject it moved, a terse detail, a relative time, an optional
// closure/aperture reason, and an optional before/after diff.
type driftEvent struct {
	Change string
	Family string
	Subject string
	Detail string
	Time   string
	Reason string
	Diff   []driftDiffLine
}

// driftBatch is one batch's group of transitions — the unit the timeline groups by
// (#288, ADR-0111). Its Label is the batch kind and how long ago it folded; Meta
// summarises the batch's recorded scope.
type driftBatch struct {
	Label  string
	Meta   string
	Events []driftEvent
}

// driftPeriod is one entry of the period selector: the ?period token, its badge
// label, and the lookback window it maps to. `all` maps to a zero duration, read as
// "since the beginning" (no lower bound).
type driftPeriod struct {
	Token  string
	Label  string
	Window time.Duration
}

// driftPeriods is the fixed period vocabulary the selector offers, in order. 7d is
// the default (matching the ported example's "Last 7d" badge).
func driftPeriods() []driftPeriod {
	return []driftPeriod{
		{Token: "7d", Label: "Last 7d", Window: 7 * 24 * time.Hour},
		{Token: "30d", Label: "Last 30d", Window: 30 * 24 * time.Hour},
		{Token: "90d", Label: "Last 90d", Window: 90 * 24 * time.Hour},
		{Token: "all", Label: "All time", Window: 0},
	}
}

// resolveDriftPeriod maps the ?period query to a period, defaulting to 7d for an
// absent or unrecognised token (a hand-crafted value never widens the window past
// the offered set).
func resolveDriftPeriod(token string) driftPeriod {
	for _, p := range driftPeriods() {
		if p.Token == token {
			return p
		}
	}
	return driftPeriods()[0]
}

// driftFeedLimit caps how many span events the feed reads and renders in one page,
// so `period=all` on a mature estate cannot load and render an unbounded corpus (the
// thesis screen must never 500 or balloon). The most recent events win — the query
// orders newest-batch-first — and the page states plainly when the cap truncated the
// view rather than dropping rows silently.
const driftFeedLimit int32 = 500

// driftSince turns a period into the @since bound the feed query filters on. `all`
// (zero window) reads from the zero instant, so no batch is excluded by age.
func (s *server) driftSince(p driftPeriod) pgtype.Timestamptz {
	if p.Window == 0 {
		return pgtype.Timestamptz{Time: time.Time{}, Valid: true}
	}
	return pgtype.Timestamptz{Time: s.now().UTC().Add(-p.Window), Valid: true}
}

// driftPage renders the Drift screen. It carries the change vocabulary (the legend),
// the batch-grouped transitions for the selected period, and the per-kind movement
// summary. The timeline falls to the empty-state when the feed is empty; the handler
// fabricates no change events. A feed read error degrades to the empty-state rather
// than 500ing the thesis screen.
func (s *server) driftPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	period := resolveDriftPeriod(r.URL.Query().Get("period"))

	var groups []driftBatch
	movement := driftMovement{}
	truncated := false
	if rows, err := s.store.ListRecentDriftEvents(r.Context(), db.ListRecentDriftEventsParams{
		Since: s.driftSince(period), MaxEvents: driftFeedLimit,
	}); err != nil {
		log.Printf("web: drift: list recent drift events: %v", err)
	} else {
		// The cap keeps the newest events (query orders newest-first); a full page is
		// stated as truncated rather than silently dropping the older tail.
		truncated = int32(len(rows)) >= driftFeedLimit
		groups, movement = buildDriftFeed(rows, s.now())
	}

	// Batch detail entry (#311, T16) — opens the Run detail screen (T2, GET /run/{id};
	// id is a Dispatch id) for the most recent batch. Change and batches are distinct
	// feeds: a batch exists as soon as a scan has been dispatched, well before two
	// batches have folded a transition, so the entry is offered whenever a real
	// dispatch exists and omitted otherwise — never a fabricated id. This mirrors the
	// ported example's `onOpenRun && <Button>Batch detail</Button>`.
	batchID, batchLabel := s.latestBatch(r)

	s.render(w, "drift", map[string]any{
		"Title": "Drift", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":  "drift",
		"Kinds":      driftKinds(),
		"Groups":     groups,
		"Movement":   movement,
		"Periods":    driftPeriods(),
		"Period":     period.Token,
		"HasEvents":  len(groups) > 0,
		"Truncated":  truncated,
		"FeedLimit":  driftFeedLimit,
		"BatchID":    batchID,
		"BatchLabel": batchLabel,
	})
}

// driftExport serves the transition feed for the active period as a downloadable CSV
// (#288) — the same reason the Reports export exists: pull the change numbers into a
// sheet or a pipeline without screenshotting. It reads the same estate-wide feed the
// page renders for the same ?period=, so the file mirrors the screen. A viewer reads
// it — an export is a read of the change the page already shows, never a mutation. It
// fabricates nothing: an empty feed produces a header-only file, never invented rows.
// It reads under the same cap the page uses (driftFeedLimit), so an unbounded corpus
// cannot stream an unbounded file; a truncated export is noted in the logs and carries
// a trailing marker row rather than dropping the older tail silently.
func (s *server) driftExport(w http.ResponseWriter, r *http.Request, acct db.Account) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		http.Error(w, "unsupported export format: "+format+" (want csv)", http.StatusBadRequest)
		return
	}

	period := resolveDriftPeriod(r.URL.Query().Get("period"))
	rows, err := s.store.ListRecentDriftEvents(r.Context(), db.ListRecentDriftEventsParams{
		Since: s.driftSince(period), MaxEvents: driftFeedLimit,
	})
	if err != nil {
		s.serverError(w, "drift export: list recent drift events", err)
		return
	}
	if int32(len(rows)) >= driftFeedLimit {
		log.Printf("web: drift export: feed capped at %d events for period=%s; older tail omitted", driftFeedLimit, period.Token)
	}
	s.writeDriftExportCSV(w, period, rows)
}

// latestBatch reads the most recent Dispatch (a batch) so Drift can offer a "Batch
// detail" entry into the Run detail screen. Dispatch is Operational — the read
// records what the system did and never touches the comparison path (ADR-0041). It
// returns a zero id when no scan has been dispatched yet, so the caller offers no
// entry rather than fabricate one; a read error degrades to no entry, never a 500
// on the thesis screen. The list is ordered id DESC, so rows[0] is the latest batch.
func (s *server) latestBatch(r *http.Request) (int64, string) {
	rows, err := s.store.ListDispatchProgress(r.Context(), scansHistoryLimit)
	if err != nil {
		log.Printf("web: drift: latest batch: %v", err)
		return 0, ""
	}
	if len(rows) == 0 {
		return 0, ""
	}
	dv := toDispatchView(rows[0])
	return dv.ID, dv.DispatchedAt
}
