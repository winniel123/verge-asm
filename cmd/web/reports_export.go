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

type reportsExportRange struct {
	Weeks int
	Days  int
	From  time.Time
	To    time.Time
}

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

	// PDF is the delivered-report document for the period, never the operational csv/json series.
	if format == "pdf" {
		s.writeReportsExportPDF(ctx, w, rng, win)
		return
	}

	counts := make([]int, days)
	window, active, hasActivity := 0, 0, false
	if rows, err := s.store.ListDispatchProgress(ctx, reportsDispatchLimit(weeks)); err != nil {
		log.Printf("web: reports export: list dispatch progress: %v", err)
	} else {
		// The page folds these same counts, so one bucketing serves both surfaces (ADR-0177, #1349).
		counts, window, active = s.bucketScanActivity(rows, days)
		hasActivity = true
	}

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

func (s *server) writeReportsExportPDF(ctx context.Context, w http.ResponseWriter, rng reportsExportRange, win reportsWindow) {
	// The web image is distroless-static, so the renderer must be pure Go (ADR-0114).
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

type reportsExportFigures struct {
	counts         []int
	window, active int
	hasActivity    bool
	openSignals    int
	hasOpenSignals bool
}

func reportsExportFilename(rng reportsExportRange, ext string) string {
	return "reports-" + rng.From.Format("2006-01-02") + "-to-" + rng.To.Format("2006-01-02") + "." + ext
}

func (rng reportsExportRange) dayDate(i int) time.Time {
	return rng.From.AddDate(0, 0, i)
}

func (s *server) writeReportsExportCSV(w http.ResponseWriter, rng reportsExportRange, fig reportsExportFigures) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+reportsExportFilename(rng, "csv")+`"`)

	cw := csv.NewWriter(w)
	defer cw.Flush()

	_ = cw.Write([]string{"section", "label", "value"})

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

	if fig.hasActivity {
		for i, c := range fig.counts {
			_ = cw.Write([]string{"scans_per_day", rng.dayDate(i).Format("2006-01-02"), strconv.Itoa(c)})
		}
	}
}

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
	OpenSignals *int `json:"open_signals"`
	ScansRun    *int `json:"scans_run"`
	InFlight    *int `json:"in_flight"`
}

type reportsExportDocDay struct {
	Date  string `json:"date"`
	Scans int    `json:"scans"`
}

func (s *server) writeReportsExportJSON(w http.ResponseWriter, rng reportsExportRange, fig reportsExportFigures) {
	doc := reportsExportDoc{
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
