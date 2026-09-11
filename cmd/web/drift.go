package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
)

type driftStore interface {
	EarliestBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	ListDispatchProgress(ctx context.Context, limit int32) ([]db.ListDispatchProgressRow, error)
	ListRecentDriftEvents(ctx context.Context, arg db.ListRecentDriftEventsParams) ([]db.ListRecentDriftEventsRow, error)
}

var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/drift.tmpl"))

func driftFamily(change string) string {
	// The change vocabulary rides its own drift palette, never the severity ramp (ADR-0110).
	switch change {
	case "appeared", "revealed", "returned":
		return "gain"
	case "withdrawn", "descoped":
		return "loss"
	default:
		return "change"
	}
}

type driftKind struct {
	Change string
	Family string
}

func driftKinds() []driftKind {
	// The legend is definitional, not data, so it renders before any batch has folded a change.
	kinds := []string{"appeared", "revealed", "withdrawn", "descoped", "returned", "changed"}
	out := make([]driftKind, 0, len(kinds))
	for _, k := range kinds {
		out = append(out, driftKind{Change: k, Family: driftFamily(k)})
	}
	return out
}

type driftDiffLine struct {
	Type string
	Text string
}

type driftEvent struct {
	Change  string
	Family  string
	Subject string
	Detail  string
	Time    string
	Reason  string
	Diff    []driftDiffLine
}

type driftBatch struct {
	Label     string
	Meta      string
	Collapsed bool
	Events    []driftEvent
}

// One preset set for every ported period panel, so a fifth preset cannot reach one screen
// and miss another.

type periodPreset struct {
	Token  string
	Label  string
	Window time.Duration
}

func periodPresets() []periodPreset {
	return []periodPreset{
		{Token: "24h", Label: "Last 24h", Window: 24 * time.Hour},
		{Token: "7d", Label: "Last 7d", Window: 7 * 24 * time.Hour},
		{Token: "30d", Label: "Last 30d", Window: 30 * 24 * time.Hour},
		{Token: "90d", Label: "Last 90d", Window: 90 * 24 * time.Hour},
	}
}

const defaultPeriodPreset = "7d"

func resolvePeriodPreset(token string) periodPreset {
	for _, p := range periodPresets() {
		if p.Token == token {
			return p
		}
	}
	// The design default is the second preset, not the first, so the fallback names it explicitly.
	for _, p := range periodPresets() {
		if p.Token == defaultPeriodPreset {
			return p
		}
	}
	return periodPresets()[0]
}

const driftCustomPrefix = "custom_"

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

// Every ported period panel resolves its custom half the same way, so it sits once.

func resolveCustomWindow(q url.Values) (token, label string, from, until pgtype.Timestamptz, ok bool) {
	start, end := q.Get("start"), q.Get("end")
	if start == "" && end == "" {
		if st, en, found := parseCustomToken(q.Get("period")); found {
			start, end = st, en
		}
	}
	if start == "" || end == "" {
		return "", "", pgtype.Timestamptz{}, pgtype.Timestamptz{}, false
	}
	sd, e1 := time.Parse("2006-01-02", start)
	ed, e2 := time.Parse("2006-01-02", end)
	if e1 != nil || e2 != nil {
		return "", "", pgtype.Timestamptz{}, pgtype.Timestamptz{}, false
	}
	// A reversed pair selects nothing, and an empty state would then read as a fact.
	if ed.Before(sd) {
		return "", "", pgtype.Timestamptz{}, pgtype.Timestamptz{}, false
	}
	return driftCustomPrefix + start + "_" + end,
		start + " – " + end,
		pgtype.Timestamptz{Time: sd.UTC(), Valid: true},
		// The operator's end date is inclusive, so the bound is the next day's start.
		pgtype.Timestamptz{Time: ed.UTC().Add(24 * time.Hour), Valid: true},
		true
}

func (s *server) resolveDriftWindow(r *http.Request) (token, label string, since, until pgtype.Timestamptz) {
	q := r.URL.Query()
	if tk, lb, from, to, ok := resolveCustomWindow(q); ok {
		return tk, lb, from, to
	}
	period := resolvePeriodPreset(q.Get("period"))
	return period.Token, period.Label, s.presetSince(period), pgtype.Timestamptz{}
}

const driftFeedLimit int32 = 500

func (s *server) presetSince(p periodPreset) pgtype.Timestamptz {
	if p.Window == 0 {
		return pgtype.Timestamptz{Time: time.Time{}, Valid: true}
	}
	return pgtype.Timestamptz{Time: s.now().UTC().Add(-p.Window), Valid: true}
}

func (s *server) driftPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "drift", s.driftFixtureData(acct))
		return
	}

	token, periodLabel, since, until := s.resolveDriftWindow(r)

	// A 90d window on a mature estate is unbounded, so the feed reads under a cap (ADR-0178 §1).
	rows, err := s.driftStore.ListRecentDriftEvents(r.Context(), db.ListRecentDriftEventsParams{
		Since: since, Until: until, MaxEvents: driftFeedLimit,
	})
	if err != nil {
		// An empty feed reads as no drift, so Drift's own subject is loud (ADR-0168 §4, #1424).
		s.serverError(w, "drift: list recent drift events", err)
		return
	}
	truncated := int32(len(rows)) >= driftFeedLimit // #nosec G115 (len(rows) capped at driftFeedLimit=500 via query MaxEvents)
	groups, movement := buildDriftFeed(rows, s.now())

	transitionCount := 0
	// JS toggles a group open, so the whole period feed ships, never a filtered one (ADR-0178 §4).
	for i := range groups {
		transitionCount += len(groups[i].Events)
		if i >= 2 {
			groups[i].Collapsed = true
		}
	}

	// A batch exists at dispatch, long before two fold a transition, so it can precede a feed.
	batchID, batchLabel := s.latestBatch(r)

	s.render(w, r, "drift", pageData(acct, "Drift", "drift", map[string]any{
		"Kinds":           driftKinds(),
		"Groups":          groups,
		"Movement":        movement,
		"Periods":         periodPresets(),
		"Period":          token,
		"PeriodLabel":     periodLabel,
		"HasEvents":       len(groups) > 0,
		"Truncated":       truncated,
		"FeedLimit":       driftFeedLimit,
		"BatchID":         batchID,
		"BatchLabel":      batchLabel,
		"TransitionCount": transitionCount,
		"TransitionDelta": s.transitionDelta(r.Context(), since, until, transitionCount),
	}))
}

func (s *server) transitionDelta(ctx context.Context, since, until pgtype.Timestamptz, currentCount int) string {
	if !since.Valid {
		return ""
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

	earliest, err := s.driftStore.EarliestBatchTime(ctx)
	if err != nil {
		log.Printf("web: drift: earliest batch time: %v", err)
		return ""
	}
	// A window the estate never saw is unknown, not zero, so the chip is suppressed (ADR-0110).
	if !earliest.Valid || earliest.Time.After(prevStart) {
		return ""
	}

	rows, err := s.driftStore.ListRecentDriftEvents(ctx, db.ListRecentDriftEventsParams{
		Since: pgtype.Timestamptz{Time: prevStart, Valid: true}, Until: since, MaxEvents: driftFeedLimit,
	})
	if err != nil {
		log.Printf("web: drift: previous-window drift events: %v", err)
		return ""
	}
	return driftTransitionDelta(currentCount, rows, earliest, prevStart, s.now())
}

func driftTransitionDelta(currentCount int, prevRows []db.ListRecentDriftEventsRow, earliest pgtype.Timestamptz, prevStart, now time.Time) string {
	if !earliest.Valid || earliest.Time.After(prevStart) {
		return ""
	}
	// The classifier drops events it cannot narrate, so a raw row count would not match the screen.
	prevGroups, _ := buildDriftFeed(prevRows, now)
	prevCount := 0
	for i := range prevGroups {
		prevCount += len(prevGroups[i].Events)
	}
	text, _ := signedCount(currentCount - prevCount)
	return text
}

func (s *server) driftExport(w http.ResponseWriter, r *http.Request, acct db.Account) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		http.Error(w, "unsupported export format: "+format+" (want csv)", http.StatusBadRequest)
		return
	}

	token, _, since, until := s.resolveDriftWindow(r)
	rows, err := s.driftStore.ListRecentDriftEvents(r.Context(), db.ListRecentDriftEventsParams{
		Since: since, Until: until, MaxEvents: driftFeedLimit,
	})
	if err != nil {
		s.serverError(w, "drift export: list recent drift events", err)
		return
	}
	if int32(len(rows)) >= driftFeedLimit { // #nosec G115 G706 (len(rows) capped at driftFeedLimit=500 via query MaxEvents; token below is a constant preset or time.Parse-validated YYYY-MM-DD range — no CR/LF)
		log.Printf("web: drift export: feed capped at %d events for period=%s; older tail omitted", driftFeedLimit, token)
	}
	s.writeDriftExportCSV(w, token, rows)
}

func (s *server) latestBatch(r *http.Request) (int64, string) {
	rows, err := s.driftStore.ListDispatchProgress(r.Context(), scansHistoryLimit)
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
