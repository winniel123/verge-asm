package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
)

// The Drift screen is served byte-for-byte from the frozen design-owned
// design-system/templates/drift.tmpl (package v3.8.0, WORKFLOW v4, #562/#563/#564),
// which replaces the repo-authored templates_drift.go const (deleted). The tmpl keeps
// the "drift" + "changeglyph" defines and renders inside the full app chrome
// ({{template "chrome" .}}); it styles against the design token vocabulary, so the
// render opts in with DesignTokens:true (the "head" block inlines tokens/*.css only
// then, as Coverage/Exposure do). drift.tmpl auto-embeds through designfs's existing
// templates/*.tmpl glob, so no designfs.go change is needed.
//
// The tmpl declares the holes driftPage shapes below. KEPT: .Periods[{Token,Label}],
// .Period, .HasEvents, .BatchID, .BatchLabel, .Truncated, .FeedLimit,
// .Kinds[{Change,Family}], .Groups[…], .Movement. NEW (this conversion): .PeriodLabel
// (the trigger label — the active preset's label, or "start – end" for a custom range),
// .Groups[].Collapsed (groups older than the two most recent batches), .TransitionCount,
// and .TransitionDelta (a nullable signed string vs the previous period; "" suppresses
// the chip). Reconciliations SPEC-CHANGE #20 (all ruled): the period control is the
// spec's range picker — preset tokens stay links inside the popover, a custom ISO pair
// submits GET /drift?start=&end= (#20b, resolveDriftWindow); the "Derived · drift"
// microlabel drops (#20c); kind chips toggle and batch groups collapse client-side —
// the view JS ships in the frozen tmpl (ADR-0105), so .Groups always carries the FULL
// period feed, never a server-filtered one; the Movement tally follows the .Kinds
// vocabulary order (#20g). Batch detail routes /runs/{id}.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/drift.tmpl"))

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
	Change  string
	Family  string
	Subject string
	Detail  string
	Time    string
	Reason  string
	Diff    []driftDiffLine
}

// driftBatch is one batch's group of transitions — the unit the timeline groups by
// (#288, ADR-0111). Its Label is the batch kind and how long ago it folded; Meta
// summarises the batch's recorded scope.
type driftBatch struct {
	Label string
	Meta  string
	// Collapsed marks a group whose events start folded away in the timeline —
	// true for groups older than the two most recent batches (SPEC-CHANGE #20). The
	// frozen tmpl draws it closed and its own JS toggles it open client-side; the
	// full feed is always carried, never server-filtered.
	Collapsed bool
	Events    []driftEvent
}

// driftPeriod is one entry of the period selector: the ?period token, its badge
// label, and the lookback window it maps to. `all` maps to a zero duration, read as
// "since the beginning" (no lower bound).
type driftPeriod struct {
	Token  string
	Label  string
	Window time.Duration
}

// driftPeriods is the fixed preset vocabulary the range picker offers, in the design's
// authored order (fixtures.json → drift.periods): 24h, 7d, 30d, 90d. The design package
// is normative for functionality, so this is the design's vocabulary — 7d is the default
// (fixtures.json → drift.period), but it is the SECOND entry, so the default is named
// explicitly rather than taken as the first.
func driftPeriods() []driftPeriod {
	return []driftPeriod{
		{Token: "24h", Label: "Last 24h", Window: 24 * time.Hour},
		{Token: "7d", Label: "Last 7d", Window: 7 * 24 * time.Hour},
		{Token: "30d", Label: "Last 30d", Window: 30 * 24 * time.Hour},
		{Token: "90d", Label: "Last 90d", Window: 90 * 24 * time.Hour},
	}
}

// driftDefaultPeriod is the preset selected when no ?period token is given — 7d, the
// design's default (fixtures.json → drift.period).
const driftDefaultPeriod = "7d"

// resolveDriftPeriod maps the ?period query to a preset, defaulting to driftDefaultPeriod
// for an absent or unrecognised token (a hand-crafted value never widens the window past
// the offered set).
func resolveDriftPeriod(token string) driftPeriod {
	for _, p := range driftPeriods() {
		if p.Token == token {
			return p
		}
	}
	for _, p := range driftPeriods() {
		if p.Token == driftDefaultPeriod {
			return p
		}
	}
	return driftPeriods()[0]
}

// driftCustomPrefix marks a custom-range period token — "custom_<start>_<end>" with ISO
// (YYYY-MM-DD) bounds. The range popover submits GET /drift?start=&end= (#20b); the page
// stamps this stable token into .Period so the Export CSV link (/drift/export?period=)
// carries the same window and the file mirrors the screen.
const driftCustomPrefix = "custom_"

// parseCustomToken splits a "custom_<start>_<end>" period token back into its ISO bounds.
func parseCustomToken(token string) (start, end string, ok bool) {
	rest, found := strings.CutPrefix(token, driftCustomPrefix)
	if !found {
		return "", "", false
	}
	start, end, found = strings.Cut(rest, "_")
	if !found || start == "" || end == "" {
		return "", "", false
	}
	return start, end, true
}

// resolveDriftWindow resolves the request's period into the feed window: the .Period
// token and .PeriodLabel the tmpl renders, plus the query bounds. A custom range — an
// explicit ?start=&end= from the popover form (#20b), or a "custom_" ?period token from
// the export link — resolves to an absolute [start, end] window with a stable token and
// a "start – end" label. Otherwise a preset resolves to its relative lookback (upper
// bound left invalid, i.e. up to now). A malformed custom pair falls back to the preset
// path, so a hand-crafted query never errors the thesis screen.
func (s *server) resolveDriftWindow(r *http.Request) (token, label string, since, until pgtype.Timestamptz) {
	q := r.URL.Query()
	start, end := q.Get("start"), q.Get("end")
	if start == "" && end == "" {
		if st, en, ok := parseCustomToken(q.Get("period")); ok {
			start, end = st, en
		}
	}
	if start != "" && end != "" {
		sd, e1 := time.Parse("2006-01-02", start)
		ed, e2 := time.Parse("2006-01-02", end)
		if e1 == nil && e2 == nil {
			return driftCustomPrefix + start + "_" + end,
				start + " – " + end,
				pgtype.Timestamptz{Time: sd.UTC(), Valid: true},
				// The end date is inclusive: bound at the start of the following day.
				pgtype.Timestamptz{Time: ed.UTC().Add(24 * time.Hour), Valid: true}
		}
	}
	period := resolveDriftPeriod(q.Get("period"))
	return period.Token, period.Label, s.driftSince(period), pgtype.Timestamptz{}
}

// filterDriftRowsUntil drops the raw span events whose batch instant is at or after the
// custom range's exclusive upper bound. The feed query has no upper-bound parameter (it
// reads from @since to now), so a custom range's end is applied here on the read side.
// Preset periods pass an invalid `until` and skip this entirely.
func filterDriftRowsUntil(rows []db.ListRecentDriftEventsRow, until time.Time) []db.ListRecentDriftEventsRow {
	out := make([]db.ListRecentDriftEventsRow, 0, len(rows))
	for _, row := range rows {
		if row.BatchAt.Valid && row.BatchAt.Time.Before(until) {
			out = append(out, row)
		}
	}
	return out
}

// driftFeedLimit caps how many span events the feed reads and renders in one page,
// so the widest preset (90d) on a mature estate cannot load and render an unbounded
// corpus (the thesis screen must never 500 or balloon). The most recent events win —
// the query orders newest-batch-first — and the page states plainly when the cap
// truncated the view rather than dropping rows silently.
const driftFeedLimit int32 = 500

// driftSince turns a preset into the @since bound the feed query filters on. A
// zero-window preset (none in the current vocabulary) reads from the zero instant, so
// no batch is excluded by age; the offered presets (24h–90d) all carry a real window.
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
	// VERGE_DEV pixel-parity path (#563). The frozen drift.tmpl renders the batch-grouped
	// transition timeline, the range picker, the movement tally and the diff affordance —
	// a curated corpus (3 groups / 7 events incl. one diff, the movement tally, the +2
	// transition delta) whose exact events, groups and delta are the design's, not a
	// live-estate read. Reproducing them from the live span derivations would mean
	// fabricating domain data, which SPEC-CHANGE forbids — so, exactly as the Exposure /
	// Coverage screens pin their dev fixture and serve it under devMode with a drift test
	// (TestDriftFixtureMatchesPackage), drift serves the pinned fixtures.json → drift slice
	// here so the seeded candidate renders byte-for-byte what the golden composes. A real
	// deployment (devMode == false) falls through to the honest live feed below.
	if s.devMode {
		s.render(w, r, "drift", s.driftFixtureData(acct))
		return
	}

	// The period window: a preset (relative lookback) or a custom ISO [start, end] range
	// (#20b). token + label feed .Period / .PeriodLabel; until bounds a custom range.
	token, periodLabel, since, until := s.resolveDriftWindow(r)

	var groups []driftBatch
	movement := driftMovement{}
	truncated := false
	if rows, err := s.store.ListRecentDriftEvents(r.Context(), db.ListRecentDriftEventsParams{
		Since: since, MaxEvents: driftFeedLimit,
	}); err != nil {
		log.Printf("web: drift: list recent drift events: %v", err)
	} else {
		if until.Valid {
			rows = filterDriftRowsUntil(rows, until.Time)
		}
		// The cap keeps the newest events (query orders newest-first); a full page is
		// stated as truncated rather than silently dropping the older tail.
		truncated = int32(len(rows)) >= driftFeedLimit // #nosec G115 (len(rows) capped at driftFeedLimit=500 via query MaxEvents)
		groups, movement = buildDriftFeed(rows, s.now())
	}

	// Groups older than the two most recent batches start collapsed (SPEC-CHANGE #20);
	// the full feed is always carried and the tmpl's JS toggles them open client-side.
	// .TransitionCount is this period's total folded transitions (the .Movement sum).
	transitionCount := 0
	for i := range groups {
		transitionCount += len(groups[i].Events)
		if i >= 2 {
			groups[i].Collapsed = true
		}
	}

	// Batch detail entry (#311, T16) — opens the Run detail screen (screen 9, GET
	// /runs/{id}; id is a Dispatch id) for the most recent batch. Change and batches are
	// distinct feeds: a batch exists as soon as a scan has been dispatched, well before
	// two batches have folded a transition, so the entry is offered whenever a real
	// dispatch exists and omitted otherwise — never a fabricated id.
	batchID, batchLabel := s.latestBatch(r)

	s.render(w, r, "drift", map[string]any{
		"Title": "Drift", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "drift",
		// drift.tmpl styles against the design token vocabulary; the "head" block inlines
		// tokens/*.css only when this datum is set (as Coverage/Exposure do).
		"DesignTokens":    true,
		"Kinds":           driftKinds(),
		"Groups":          groups,
		"Movement":        movement,
		"Periods":         driftPeriods(),
		"Period":          token,
		"PeriodLabel":     periodLabel,
		"HasEvents":       len(groups) > 0,
		"Truncated":       truncated,
		"FeedLimit":       driftFeedLimit,
		"BatchID":         batchID,
		"BatchLabel":      batchLabel,
		"TransitionCount": transitionCount,
		// The vs-previous-period delta chip: this window's transition count minus the
		// immediately preceding equal-length window's, signed ("+2"/"−1") or "0" — an
		// empty string suppresses the chip when no complete previous window exists (an
		// install younger than 2× the window), never a fabricated "+0" (the P0.2 degrade,
		// collision #36 ruling (a), #690). The design fixture carries "+2".
		"TransitionDelta": s.transitionDelta(r.Context(), since, until, transitionCount),
	})
}

// transitionDelta computes the Drift page's vs-previous-period chip (collision #36
// ruling (a), #690): the selected window's transition count minus the immediately
// preceding equal-length window's, rendered signed ("+2"/"−1") or "0". It returns the
// empty string — which suppresses the chip — when no complete previous window exists to
// compare against: the estate's earliest batch is younger than the preceding window's
// start (install younger than 2× the window), so there is nothing to compare against
// and no baseline is fabricated (the P0.2 vs-last-batch HasDeltas-false degrade). A read
// error degrades the same way rather than 500ing the thesis screen.
//
// The preceding window is [since−length, since); its transition count is read and
// classified through the SAME estate-wide feed path (ListRecentDriftEvents →
// buildDriftFeed) the current .TransitionCount is derived from, so both windows count
// identically across all six kinds. Presets and custom ISO ranges alike resolve here to
// a [since, hi] window: hi is the custom range's exclusive upper bound, else now.
func (s *server) transitionDelta(ctx context.Context, since, until pgtype.Timestamptz, currentCount int) string {
	if !since.Valid {
		return "" // no lower bound (no offered preset resolves this) — nothing to compare
	}
	hi := s.now().UTC()
	if until.Valid {
		hi = until.Time
	}
	length := hi.Sub(since.Time)
	if length <= 0 {
		return ""
	}
	prevStart := since.Time.Add(-length)

	// No complete previous window: the estate must have been observing since at or before
	// the preceding window's start. Otherwise the preceding equal window predates the
	// first batch — suppress rather than compare against a window never witnessed.
	earliest, err := s.store.EarliestBatchTime(ctx)
	if err != nil {
		log.Printf("web: drift: earliest batch time: %v", err)
		return ""
	}
	if !earliest.Valid || earliest.Time.After(prevStart) {
		return ""
	}

	// The preceding window's rows: the query's Since bounds the lower edge (prevStart);
	// filterDriftRowsUntil trims batches at or after the current window's start (since),
	// leaving exactly [prevStart, since) — the same read + until-trim the custom-range
	// current window uses.
	rows, err := s.store.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		Since: pgtype.Timestamptz{Time: prevStart, Valid: true}, MaxEvents: driftFeedLimit,
	})
	if err != nil {
		log.Printf("web: drift: previous-window drift events: %v", err)
		return ""
	}
	rows = filterDriftRowsUntil(rows, since.Time)
	return driftTransitionDelta(currentCount, rows, earliest, prevStart, s.now())
}

// driftTransitionDelta renders the vs-previous-period chip string from the current
// window's transition count and the preceding window's raw feed rows. It classifies the
// preceding rows through the same buildDriftFeed the current count uses, so both windows
// count identically, then renders the signed difference ("+2"/"−1"/"0"). It returns the
// empty string (chip suppressed) when the estate's earliest batch is missing or younger
// than the preceding window's start — no complete previous window, no fabricated
// baseline. Split from the store-touching method above so the compare is unit-testable.
func driftTransitionDelta(currentCount int, prevRows []db.ListRecentDriftEventsRow, earliest pgtype.Timestamptz, prevStart, now time.Time) string {
	if !earliest.Valid || earliest.Time.After(prevStart) {
		return ""
	}
	prevGroups, _ := buildDriftFeed(prevRows, now)
	prevCount := 0
	for i := range prevGroups {
		prevCount += len(prevGroups[i].Events)
	}
	text, _ := signedCount(currentCount - prevCount)
	return text
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

	// The export reads the same window the page renders for the same ?period= — a preset
	// or a custom_ range token (#20b) — so the file mirrors the screen.
	token, _, since, until := s.resolveDriftWindow(r)
	rows, err := s.store.ListRecentDriftEvents(r.Context(), db.ListRecentDriftEventsParams{
		Since: since, MaxEvents: driftFeedLimit,
	})
	if err != nil {
		s.serverError(w, "drift export: list recent drift events", err)
		return
	}
	if until.Valid {
		rows = filterDriftRowsUntil(rows, until.Time)
	}
	if int32(len(rows)) >= driftFeedLimit { // #nosec G115 G706 (len(rows) capped at driftFeedLimit=500 via query MaxEvents; token below is a constant preset or time.Parse-validated YYYY-MM-DD range — no CR/LF)
		log.Printf("web: drift export: feed capped at %d events for period=%s; older tail omitted", driftFeedLimit, token)
	}
	s.writeDriftExportCSV(w, token, rows)
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
