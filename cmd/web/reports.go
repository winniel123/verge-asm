package main

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Reports screen — canonical `/reports` (#285, V2 console map #275). It folds
// the analytics that lived under /exposure and /scans into one period view,
// composed after design-system/examples/console/Reports.jsx (07-console.jpg): a
// KPI band, a time-series card, a by-severity card, a scans-per-day heatmap, a
// recurring-reports table, and a schedule wizard.
//
// The example is a mock over a severity-scored, trended signal model this product
// does not have. Three of its regions are domain-incompatible and are re-skinned
// to honest current-state facts + design-system empty-states rather than
// fabricated data (CONTEXT.md; ADR-0024):
//
//   - Signals carry NO severity. The census is "deliberately not a severity ramp"
//     (signals.go), so the "By severity" bars have no real series — empty-stated.
//   - A signal census "is never a delta, trend or series" (internal/signal). So
//     "Open signals over time" is not a real series either — empty-stated. The
//     current count of firing signals IS an honest current-state scalar, and that
//     is the one signal figure wired (the headline KPI).
//   - Signals are withdrawn by the world, never "resolved" by an operator, so the
//     example's mean-time-to-resolve KPI is dropped; its slot carries an honest
//     operational scalar (active scans) instead.
//
// The one legitimate series is operational: scans-per-day is activity volume, not
// a signal, so the heatmap is wired from real Dispatch history. Report scheduling
// (the recurring table + wizard) has no backend yet — the table empty-states and
// the wizard renders as an inert preview. Every gap is tracked in a comment on
// issue #285. This handler reads exposure/scans/signals data sources read-only;
// it owns no mutation and adds no store method.

// reportsDispatchLimit bounds the Dispatch read behind the scans-per-day heatmap.
// The heatmap spans twelve weeks, and a Dispatch is one fan-out of one Scan, so a
// generous cap covers that window without paging while staying a bounded read.
const reportsDispatchLimit = 2000

// reportsHeatWeeks / reportsHeatDays are the heatmap's span: twelve weeks of one
// column per week, seven rows per column, oldest-first.
const (
	reportsHeatWeeks = 12
	reportsHeatDays  = reportsHeatWeeks * 7
)

// heatCell is one day in the scans-per-day heatmap: the pre-computed inline
// background (an intensity step on --chart-1, or the sunken step at zero), a
// border, and a hover title. Intensity is folded in the handler so the template
// stays a plain range with no arithmetic. Bg/Border are template.CSS so the
// style-attribute sanitizer emits their color-mix()/var() values verbatim rather
// than neutralizing them to ZgotmplZ.
type heatCell struct {
	Bg     template.CSS
	Border template.CSS
	Title  string
}

func (s *server) reportsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// Operational activity — the scans-per-day heatmap and the two activity KPIs.
	// A Dispatch read failure degrades to an empty heatmap rather than failing the
	// whole analytics page a viewer depends on.
	cells, heatTotal, scansWindow, activeScans := []heatCell{}, 0, 0, 0
	rows, err := s.store.ListDispatchProgress(ctx, reportsDispatchLimit)
	if err != nil {
		log.Printf("web: reports: list dispatch progress: %v", err)
	} else {
		cells, heatTotal, scansWindow, activeScans = s.foldScanActivity(rows)
	}

	// The headline KPI — the current count of firing signals. It is a current-state
	// census total (not a trend), so it is the one honest signal figure. Building
	// the corpus is the heavier read; on failure the KPI degrades to unavailable
	// rather than 500ing the page.
	openSignals, hasOpenSignals := 0, false
	if corpus, err := s.buildSignalCorpus(r); err != nil {
		log.Printf("web: reports: build signal corpus: %v", err)
	} else {
		for _, c := range signal.EvaluateCorpus(corpus) {
			openSignals += len(c.Fired)
		}
		hasOpenSignals = true
	}

	s.render(w, "reports", map[string]any{
		"Title": "Reports", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "reports",

		// KPI band.
		"OpenSignals":    openSignals,
		"HasOpenSignals": hasOpenSignals,
		"ScansWindow":    scansWindow,
		"ActiveScans":    activeScans,

		// Scans-per-day heatmap.
		"Heat":    cells,
		"HasHeat": heatTotal > 0,
	})
}

// foldScanActivity buckets recent Dispatches into the twelve-week scans-per-day
// heatmap and the two activity KPIs. Each Dispatch is placed by the whole-day
// offset of its creation from today (UTC), oldest-first; the window KPI counts the
// Dispatches inside the span, and the active KPI those with jobs still ready or
// running. Intensity ramps on --chart-1 in four steps, matching HeatmapCalendar.
func (s *server) foldScanActivity(rows []db.ListDispatchProgressRow) (cells []heatCell, total, window, active int) {
	const day = 24 * time.Hour
	today := s.now().UTC().Truncate(day)
	counts := make([]int, reportsHeatDays)
	for _, row := range rows {
		if row.Ready+row.Running > 0 {
			active++
		}
		if !row.CreatedAt.Valid {
			continue
		}
		created := row.CreatedAt.Time.UTC().Truncate(day)
		offset := int(today.Sub(created).Hours() / 24)
		if offset < 0 || offset >= reportsHeatDays {
			continue
		}
		counts[reportsHeatDays-1-offset]++ // index 0 = oldest, last = today
		window++
	}

	max := 1
	for _, c := range counts {
		if c > max {
			max = c
		}
		total += c
	}

	// Intensity steps mirror HeatmapCalendar.jsx: 0/28/48/72/100% of --chart-1
	// mixed into --surface, with the sunken step at zero.
	pct := []int{0, 28, 48, 72, 100}
	cells = make([]heatCell, reportsHeatDays)
	for i, c := range counts {
		level := 0
		if c > 0 {
			level = (c*4 + max - 1) / max // ceil(c/max*4)
			if level < 1 {
				level = 1
			}
			if level > 4 {
				level = 4
			}
		}
		cell := heatCell{Title: pluralScans(c)}
		if level == 0 {
			cell.Bg = template.CSS("var(--sunken)")
			cell.Border = template.CSS("var(--hairline)")
		} else {
			cell.Bg = template.CSS("color-mix(in srgb, var(--chart-1) " + strconv.Itoa(pct[level]) + "%, var(--surface))")
			cell.Border = template.CSS("transparent")
		}
		cells[i] = cell
	}
	return cells, total, window, active
}

func pluralScans(n int) string {
	if n == 1 {
		return "1 scan"
	}
	return strconv.Itoa(n) + " scans"
}

// reportDeliveryPage renders an already-delivered report artifact — the stable
// `/reports/delivery` view Reports' "view last delivery" links to (T17). The
// delivered document itself is rendered by internal/message's RenderArtifact, the
// one canonical rendered form that doubles as the PDF / email spec; this handler
// composes the console chrome around it.
//
// There is no report-scheduling or delivery backend yet (#285; #290/#291 populate
// report content), so no delivered artifact exists to read. Rather than fabricate
// a document, the handler renders an empty Artifact — RenderArtifact draws the
// design-system empty-state inside the delivered-document frame — and the header
// falls back to a generic heading. When #290/#291 land, this handler reads the
// latest delivery for the account and fills the same Artifact struct with real
// data of the same shape; the render path does not change.
func (s *server) reportDeliveryPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// No delivery backing store yet — the zero Artifact renders the empty-state.
	art := message.Artifact{}

	heading := art.Title
	if heading == "" {
		heading = "Report delivery"
	}

	s.render(w, "reportartifact", map[string]any{
		"Title": "Report delivery", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "reports",
		"Heading":   heading,
		"Period":    message.ArtifactPeriod(art),
		"Body":      message.RenderArtifact(art),
	})
}
