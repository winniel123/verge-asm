package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
)

// Reports export — `GET /reports/export` (#291, #23c). It serves the Reports figures
// for the active period as a downloadable file so an operator can pull the numbers into
// a sheet or a pipeline without screenshotting the console. It reuses reportsPage's exact
// data sources and its range param, so the download mirrors what the screen shows for the
// same window.
//
// Three formats ship (the spec SplitButton — CSV primary, JSON + PDF in the menu, #23c),
// chosen by ?format=. CSV and JSON are the operational activity read: the KPI band (open
// signals / scans run / in flight) and the scans-per-day series over the active range. CSV
// is one uniform three-column table (section,label,value) a spreadsheet opens without a
// schema; JSON is the same figures as a structured object with an explicit null where a
// read was unavailable. PDF (#23c, spec-normative, built in #586) is a DIFFERENT read: the
// *delivered-report* document for the period — a drift narrative (appeared / withdrawn
// changes) — recomputed from the period bounds by internal/message.RenderArtifactPDF
// (ADR-0114), the SAME renderer /reports/delivery/pdf uses. The operational (csv/json) and
// delivered (pdf) reads must not be conflated, but both are now served from this one route
// by ?format=. (This corrects an earlier comment that claimed PDF was not offered here.)
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

// reportsExport serves the Reports figures for the active period as CSV, JSON or PDF
// (the spec SplitButton — CSV primary, JSON + PDF in the menu, #23c). The format is
// chosen by ?format=; an absent format defaults to csv, an unrecognised one is a 400.
// The window rides the ?period= token the page carries (#23b), so the download mirrors
// the screen. CSV/JSON are the operational activity series; PDF is the delivered-report
// document recomputed from the period bounds (the same machinery as the delivery PDF).
func (s *server) reportsExport(w http.ResponseWriter, r *http.Request, acct db.Account) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "json" && format != "pdf" {
		http.Error(w, "unsupported export format: "+format+" (want csv, json or pdf)", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	win := resolveReportsWindow(r)
	weeks := win.Weeks
	days := weeks * 7

	const day = 24 * time.Hour
	to := s.now().UTC().Truncate(day)
	rng := reportsExportRange{Weeks: weeks, Days: days, From: to.AddDate(0, 0, -(days - 1)), To: to}

	// PDF (#23c): the delivered-report document for the period, recomputed from the
	// bounds via internal/message.RenderArtifactPDF — the SAME renderer the delivery PDF
	// uses (reportDeliveryPDF), not a stub. It carries the period's signals / severity
	// breakdown / withdrawals recomputed off the never-deleted ledgers.
	if format == "pdf" {
		s.writeReportsExportPDF(ctx, w, rng, win)
		return
	}

	// Scans-per-day series + the two activity KPIs — the same read reportsPage does.
	// A Dispatch read failure marks the activity block unavailable (empty / null KPIs
	// and no per-day rows) rather than exporting a fabricated all-zero series that a
	// reader could not tell from a real zero.
	counts := make([]int, days)
	window, active, hasActivity := 0, 0, false
	if rows, err := s.store.ListDispatchProgress(ctx, reportsDispatchLimit(weeks)); err != nil {
		log.Printf("web: reports export: list dispatch progress: %v", err)
	} else {
		counts, window, active = s.bucketScanActivity(rows, days)
		hasActivity = true
	}

	// The headline KPI — the current count of firing signals, a current-state census
	// (reportsPage's one honest signal figure, shared via openSignalsCount). On a
	// corpus read failure it exports as unavailable (empty / null), never a zero.
	openSignals, hasOpenSignals := s.openSignalsCount(r)

	fig := reportsExportFigures{
		counts: counts, window: window, active: active, hasActivity: hasActivity,
		openSignals: openSignals, hasOpenSignals: hasOpenSignals,
	}
	switch format {
	case "json":
		s.writeReportsExportJSON(w, rng, fig)
	default:
		s.writeReportsExportCSV(w, rng, fig)
	}
}

// writeReportsExportPDF renders the period document as a PDF download (#23c). It builds
// the delivered Artifact for the period's [From, To] bounds — the report name is the
// period label — and recomputes its signals / severity / withdrawals off the same
// ledgers the delivery path reads (reportDeliverySignals / reportDeliveryWithdrawals),
// then renders it with internal/message.RenderArtifactPDF (the pure-Go renderer that
// runs inside the distroless-static web image). A viewer reads it — a report is a
// record, not a mutation. A render failure is a 500; the ledgers degrade to empty
// sections rather than fabricating content.
func (s *server) writeReportsExportPDF(ctx context.Context, w http.ResponseWriter, rng reportsExportRange, win reportsWindow) {
	start, end := rng.From, s.now().UTC()
	art := message.Artifact{
		Title:       "Reports · " + win.Label,
		PeriodStart: rng.From.Format("2006-01-02"),
		PeriodEnd:   rng.To.Format("2006-01-02"),
		GeneratedAt: s.now().UTC().Format(time.RFC3339),
		Format:      "pdf",
	}
	art.Signals, art.SeverityCounts = s.reportDeliverySignals(ctx, start, end)
	art.Withdrawn = s.reportDeliveryWithdrawals(ctx, start, end)

	pdf, err := message.RenderArtifactPDF(art)
	if err != nil {
		log.Printf("web: reports export pdf: render: %v", err)
		http.Error(w, "could not render report PDF", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+reportsExportFilename(rng, "pdf")+`"`)
	if _, err := w.Write(pdf); err != nil {
		log.Printf("web: reports export pdf: write: %v", err)
	}
}

// reportsExportFigures bundles the resolved figures for the export writers: the
// per-day scan series and activity KPIs (valid only when hasActivity), and the
// open-signals census total (valid only when hasOpenSignals). The two availability
// flags keep an unavailable read exporting as empty/null, never a fabricated zero.
type reportsExportFigures struct {
	counts         []int
	window, active int
	hasActivity    bool
	openSignals    int
	hasOpenSignals bool
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
func (s *server) writeReportsExportCSV(w http.ResponseWriter, rng reportsExportRange, fig reportsExportFigures) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+reportsExportFilename(rng, "csv")+`"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"section", "label", "value"})

	// An unavailable read writes an empty value cell, never a zero standing in for a
	// real count.
	intCell := func(v int, ok bool) string {
		if !ok {
			return ""
		}
		return strconv.Itoa(v)
	}
	_ = cw.Write([]string{"summary", "range_weeks", strconv.Itoa(rng.Weeks)})
	_ = cw.Write([]string{"summary", "period_start", rng.From.Format("2006-01-02")})
	_ = cw.Write([]string{"summary", "period_end", rng.To.Format("2006-01-02")})
	_ = cw.Write([]string{"summary", "open_signals", intCell(fig.openSignals, fig.hasOpenSignals)})
	_ = cw.Write([]string{"summary", "scans_run", intCell(fig.window, fig.hasActivity)})
	_ = cw.Write([]string{"summary", "in_flight", intCell(fig.active, fig.hasActivity)})

	// The per-day series is written only when the activity read succeeded — an
	// unavailable read emits no scans_per_day rows rather than a run of fabricated
	// zero-scan days.
	if fig.hasActivity {
		for i, c := range fig.counts {
			_ = cw.Write([]string{"scans_per_day", rng.dayDate(i).Format("2006-01-02"), strconv.Itoa(c)})
		}
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
	// Each KPI is null where its underlying read could not be completed — never a
	// fabricated zero standing in for a real count. OpenSignals tracks the signal
	// census read; ScansRun / InFlight track the Dispatch read.
	OpenSignals *int `json:"open_signals"`
	ScansRun    *int `json:"scans_run"`
	InFlight    *int `json:"in_flight"`
}

type reportsExportDocDay struct {
	Date  string `json:"date"`
	Scans int    `json:"scans"`
}

// writeReportsExportJSON emits the figures as a structured document. open_signals is
// null where the census was unavailable.
func (s *server) writeReportsExportJSON(w http.ResponseWriter, rng reportsExportRange, fig reportsExportFigures) {
	doc := reportsExportDoc{
		// The actual instant the export was produced, not the range end — so two
		// pulls the same day are distinguishable and a freshness check reads true.
		GeneratedAt: s.now().UTC().Format(time.RFC3339),
		Range: reportsExportDocRange{
			Weeks: rng.Weeks, Days: rng.Days,
			From: rng.From.Format("2006-01-02"), To: rng.To.Format("2006-01-02"),
		},
		ScansPerDay: []reportsExportDocDay{},
	}
	if fig.hasOpenSignals {
		n := fig.openSignals
		doc.KPIs.OpenSignals = &n
	}
	// Activity KPIs and the per-day series are null / empty where the Dispatch read
	// failed, never a fabricated zero.
	if fig.hasActivity {
		window, active := fig.window, fig.active
		doc.KPIs.ScansRun, doc.KPIs.InFlight = &window, &active
		doc.ScansPerDay = make([]reportsExportDocDay, len(fig.counts))
		for i, c := range fig.counts {
			doc.ScansPerDay[i] = reportsExportDocDay{Date: rng.dayDate(i).Format("2006-01-02"), Scans: c}
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+reportsExportFilename(rng, "json")+`"`)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		log.Printf("web: reports export: encode json: %v", err)
	}
}
