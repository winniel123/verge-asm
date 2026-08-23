package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/signal"
)

// Reports export — `GET /reports/export` (#291). It serves the operational Reports
// figures as a downloadable file so an operator can pull the numbers into a sheet or
// a pipeline without screenshotting the console. Two things are exported, both of
// which exist INDEPENDENTLY of report scheduling (which has no backend yet, #290):
// the KPI band (open signals / scans run / in flight) and the scans-per-day series
// over the active range. It reuses reportsPage's exact data sources and its range
// param, so the file mirrors what the screen shows for the same ?weeks=.
//
// Two formats ship: csv and json. CSV is one uniform three-column table
// (section,label,value) carrying both the summary and the per-day series, so a
// spreadsheet opens it without a schema; json is the same figures as a structured
// object with an explicit null where the signal count was unavailable. PDF is
// deliberately NOT offered here: internal/message.RenderArtifact renders the
// *delivered report* document — a drift narrative (appeared / withdrawn changes)
// backed by the delivery store this handler must not depend on — not the operational
// activity series, and there is no HTML-to-PDF machinery in the tree. Wiring PDF
// through it would mean either fabricating a delivery or standing up a renderer, so
// it is scoped out as a follow-on; csv + json satisfy the pull-the-numbers need.
//
// This handler reads exposure/scans/signals data sources read-only; it owns no
// mutation and adds no store method. It fabricates nothing: an unavailable signal
// count exports empty (CSV) / null (JSON), never a zero standing in for a real read.

// reportsExportRange bundles the resolved range for the export: the span in weeks and
// whole days, and the inclusive [From, To] date bounds (UTC whole days, oldest-first).
type reportsExportRange struct {
	Weeks int
	Days  int
	From  time.Time
	To    time.Time
}

// reportsExport serves the Reports figures for the active range as CSV or JSON. The
// format is chosen by ?format=; an absent format defaults to csv, an unrecognised one
// is a 400. The range rides the same ?weeks= param as the page, so the download
// mirrors the screen.
func (s *server) reportsExport(w http.ResponseWriter, r *http.Request, acct db.Account) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" {
		http.Error(w, "unsupported export format: "+format+" (want csv or json)", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	weeks := resolveReportsWeeks(r)
	days := weeks * 7

	// Scans-per-day series + the two activity KPIs — the same read reportsPage does.
	// A Dispatch read failure degrades to an all-zero series rather than failing the
	// download; the file still carries the honest KPI band that did read.
	counts := make([]int, days)
	window, active := 0, 0
	if rows, err := s.store.ListDispatchProgress(ctx, reportsDispatchLimit); err != nil {
		log.Printf("web: reports export: list dispatch progress: %v", err)
	} else {
		counts, window, active = s.bucketScanActivity(rows, days)
	}

	// The headline KPI — the current count of firing signals, a current-state census
	// (reportsPage's one honest signal figure). On a corpus read failure it exports as
	// unavailable (empty / null), never a fabricated zero.
	openSignals, hasOpenSignals := 0, false
	if corpus, err := s.buildSignalCorpus(r); err != nil {
		log.Printf("web: reports export: build signal corpus: %v", err)
	} else {
		for _, c := range signal.EvaluateCorpus(corpus) {
			openSignals += len(c.Fired)
		}
		hasOpenSignals = true
	}

	const day = 24 * time.Hour
	to := s.now().UTC().Truncate(day)
	rng := reportsExportRange{Weeks: weeks, Days: days, From: to.AddDate(0, 0, -(days - 1)), To: to}

	switch format {
	case "json":
		s.writeReportsExportJSON(w, rng, counts, window, active, openSignals, hasOpenSignals)
	default:
		s.writeReportsExportCSV(w, rng, counts, window, active, openSignals, hasOpenSignals)
	}
}

// reportsExportFilename is the download name: reports-<from>-to-<to>.<ext>, ISO dates
// so a directory of exports sorts by period.
func reportsExportFilename(rng reportsExportRange, ext string) string {
	return "reports-" + rng.From.Format("2006-01-02") + "-to-" + rng.To.Format("2006-01-02") + "." + ext
}

// dayDate returns the calendar date of the i-th series bucket (index 0 = oldest).
func (rng reportsExportRange) dayDate(i int) time.Time {
	return rng.From.AddDate(0, 0, i)
}

// writeReportsExportCSV emits the figures as one uniform section,label,value table —
// the summary band first, then one row per day of the scans-per-day series. An
// unavailable signal count writes an empty value cell, never a zero.
func (s *server) writeReportsExportCSV(w http.ResponseWriter, rng reportsExportRange, counts []int, window, active, openSignals int, hasOpenSignals bool) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+reportsExportFilename(rng, "csv")+`"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"section", "label", "value"})

	openVal := ""
	if hasOpenSignals {
		openVal = strconv.Itoa(openSignals)
	}
	_ = cw.Write([]string{"summary", "range_weeks", strconv.Itoa(rng.Weeks)})
	_ = cw.Write([]string{"summary", "period_start", rng.From.Format("2006-01-02")})
	_ = cw.Write([]string{"summary", "period_end", rng.To.Format("2006-01-02")})
	_ = cw.Write([]string{"summary", "open_signals", openVal})
	_ = cw.Write([]string{"summary", "scans_run", strconv.Itoa(window)})
	_ = cw.Write([]string{"summary", "in_flight", strconv.Itoa(active)})

	for i, c := range counts {
		_ = cw.Write([]string{"scans_per_day", rng.dayDate(i).Format("2006-01-02"), strconv.Itoa(c)})
	}
}

// reportsExportDoc is the JSON export shape: the resolved range, the KPI band (with a
// null open_signals where the census read failed), and the per-day series.
type reportsExportDoc struct {
	GeneratedAt string                `json:"generated_at"`
	Range       reportsExportDocRange `json:"range"`
	KPIs        reportsExportDocKPIs  `json:"kpis"`
	ScansPerDay []reportsExportDocDay `json:"scans_per_day"`
}

type reportsExportDocRange struct {
	Weeks int    `json:"weeks"`
	Days  int    `json:"days"`
	From  string `json:"from"`
	To    string `json:"to"`
}

type reportsExportDocKPIs struct {
	// OpenSignals is null where the signal census could not be read — never a
	// fabricated zero standing in for a real count.
	OpenSignals *int `json:"open_signals"`
	ScansRun    int  `json:"scans_run"`
	InFlight    int  `json:"in_flight"`
}

type reportsExportDocDay struct {
	Date  string `json:"date"`
	Scans int    `json:"scans"`
}

// writeReportsExportJSON emits the figures as a structured document. open_signals is
// null where the census was unavailable.
func (s *server) writeReportsExportJSON(w http.ResponseWriter, rng reportsExportRange, counts []int, window, active, openSignals int, hasOpenSignals bool) {
	doc := reportsExportDoc{
		GeneratedAt: rng.To.Format(time.RFC3339),
		Range: reportsExportDocRange{
			Weeks: rng.Weeks, Days: rng.Days,
			From: rng.From.Format("2006-01-02"), To: rng.To.Format("2006-01-02"),
		},
		KPIs:        reportsExportDocKPIs{ScansRun: window, InFlight: active},
		ScansPerDay: make([]reportsExportDocDay, len(counts)),
	}
	if hasOpenSignals {
		n := openSignals
		doc.KPIs.OpenSignals = &n
	}
	for i, c := range counts {
		doc.ScansPerDay[i] = reportsExportDocDay{Date: rng.dayDate(i).Format("2006-01-02"), Scans: c}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+reportsExportFilename(rng, "json")+`"`)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		log.Printf("web: reports export: encode json: %v", err)
	}
}
