package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/message"
	"github.com/winniel123/verge-asm/internal/signal"
)

// The Reports screen — canonical `/reports` (#285, V2 console map #275). It folds
// the analytics that lived under /exposure and /scans into one period view,
// composed after design-system/examples/console/Reports.jsx (07-console.jpg): a
// KPI band, a time-series card, a by-severity card, a scans-per-day heatmap, a
// recurring-reports table, and a schedule wizard.
//
// The example renders three trend series. The port once held them
// "domain-incompatible" and re-skinned them to honest scalars + empty-states, but
// the design is normative for look AND functionality (ADR-0116; PARITY-CHART.md
// §"The ruling"; SPEC-CHANGE.md collision #3), so the fix is to BUILD each series,
// not drop it. All three are now real derivations, folded in internal/drift/trend.go
// (P0.3, #444) and passed to the template as data the Reports markup (P2.4) paints:
//
//   - Signals-over-time — the "Open signals over time" line and its "Critical +
//     high" companion — is folded from the per-instance first-seen ledger
//     (signal_instance, P0.1) with each rule's severity (internal/signal): weekly
//     incidence and the standing level over the selected range.
//   - Mean-time-to-withdrawal fills the mock's mean-time-to-resolve slot, honest to
//     the domain (signals are withdrawn by the world, never "resolved" by an
//     operator): it is derived from the subject-withdrawal history in the span
//     corpus (ListWithdrawalLifespans → drift.MeanTimeToWithdrawal / WithdrawalSeries),
//     as a KPI scalar and its trend.
//   - Scans-per-day intensities ramp through the shared drift.HeatLevels rule off
//     real Dispatch history (activity volume, not a signal), so the page, the export
//     and any later surface intensify identically.
//
// Report scheduling (the recurring table + wizard) has no dispatch or delivery
// backend, so the UI must not accept a schedule it cannot honour (#344): the table
// empty-states and the "New schedule" wizard is rendered disabled alongside its
// already-disabled sibling controls. This handler reads its data sources read-only
// and owns no mutation.

// reportsDispatchPerWeek budgets the Dispatch read behind the scans-per-day series
// PER WEEK of the selected range, so a wider window reads proportionally more rows
// instead of silently truncating the oldest days at a fixed cap (the read is
// newest-first by id, so a too-small cap drops the early columns of a long range). A
// Dispatch is one fan-out of one Scan; this budget is generous for a busy estate.
const reportsDispatchPerWeek = 250

// reportsDispatchLimit is the bounded Dispatch read size for a given range in weeks,
// scaled off reportsDispatchPerWeek. int32 to match ListDispatchProgress.
func reportsDispatchLimit(weeks int) int32 { return int32(weeks * reportsDispatchPerWeek) }

// reportsHeatWeeks / reportsHeatDays are the heatmap's DEFAULT span: twelve weeks of
// one column per week, seven rows per column, oldest-first. The span is now
// selectable (reportsRangeWeeks); twelve stays the default so an un-parameterised
// request renders exactly as before.
const (
	reportsHeatWeeks = 12
	reportsHeatDays  = reportsHeatWeeks * 7
)

// reportsRangeWeeks is the fixed set of spans the /reports range control offers, in
// weeks. A small honest set — a quarter, the default twelve, a half-year, a year —
// rather than a free from/to pair: the underlying series is whole-day scan volume,
// so a coarse week-granular window is the faithful control. reportsHeatWeeks (12) is
// the default and MUST appear in the set.
var reportsRangeWeeks = []int{4, 12, 26, 52}

// resolveReportsWeeks reads the ?weeks= range param and clamps it to the offered set
// (reportsRangeWeeks), defaulting to reportsHeatWeeks when the param is absent,
// unparseable, or not one of the offered spans. Threading it through reportsPage and
// the export keeps the heatmap span, the "Scans run" window KPI, and the export all
// reading the same range.
func resolveReportsWeeks(r *http.Request) int {
	raw := r.URL.Query().Get("weeks")
	if raw == "" {
		return reportsHeatWeeks
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return reportsHeatWeeks
	}
	for _, w := range reportsRangeWeeks {
		if w == n {
			return n
		}
	}
	return reportsHeatWeeks
}

// reportsRangeOption is one entry in the range-select control: its weeks value, the
// human label ("last 12 weeks"), and whether it is the active selection.
type reportsRangeOption struct {
	Weeks    int
	Label    string
	Selected bool
}

// reportsRangeOptions builds the select entries for the header range control,
// marking the active span selected.
func reportsRangeOptions(active int) []reportsRangeOption {
	opts := make([]reportsRangeOption, 0, len(reportsRangeWeeks))
	for _, w := range reportsRangeWeeks {
		opts = append(opts, reportsRangeOption{Weeks: w, Label: reportsRangeLabel(w), Selected: w == active})
	}
	return opts
}

// reportsRangeLabel is the terse sentence-case span label used on the KPI caption,
// the heatmap aria-label/legend, and the range-select options ("last 12 weeks").
func reportsRangeLabel(weeks int) string {
	return "last " + strconv.Itoa(weeks) + " weeks"
}

// openSignalsCount is the one honest signal figure the Reports screen and its export
// both show: the current count of firing signals across every rule, a current-state
// census total (never a trend). Building the corpus is the heavier read; on failure
// it returns ok=false so the caller renders the KPI as unavailable rather than a
// fabricated zero. Shared so the page and the export never drift apart.
func (s *server) openSignalsCount(r *http.Request) (count int, ok bool) {
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		log.Printf("web: reports: build signal corpus: %v", err)
		return 0, false
	}
	for _, c := range signal.EvaluateCorpus(corpus) {
		count += len(c.Fired)
	}
	return count, true
}

// reportsTrendBucket is the trend series' bucket width — one WEEK, matching the
// /reports range control's own week granularity (reportsRangeLabel "last N weeks")
// so a signals-over-time / mean-time-to-withdrawal column lines up with a heatmap
// week. The series then carries `weeks` buckets over the selected range.
const reportsTrendBucket = 7 * 24 * time.Hour

// signalRaises reads the per-instance first-seen ledger into the trend fold's input
// (P0.3, #444): one drift.Raise per minted signal_instance, carrying its first-seen
// instant and whether its rule's severity is elevated — critical or high, the
// design's "Critical + high" series. Severity is the RULE's, looked up per instance
// (internal/signal); an unknown rule folds to the calmest level, so it is never
// elevated. The whole never-deleted ledger is read so the standing level counts
// signals raised before the window too.
func (s *server) signalRaises(ctx context.Context) ([]drift.Raise, error) {
	rows, err := s.store.ListSignalInstances(ctx)
	if err != nil {
		return nil, err
	}
	raises := make([]drift.Raise, 0, len(rows))
	for _, row := range rows {
		if !row.FirstSeen.Valid {
			continue
		}
		sev, _ := signal.SeverityFor(row.SignalName)
		raises = append(raises, drift.Raise{
			At:       row.FirstSeen.Time.UTC(),
			Elevated: sev.Rank() <= signal.SevHigh.Rank(),
		})
	}
	return raises, nil
}

// withdrawalLifespans reads the subject-withdrawal ledger since the window's start
// into the trend fold's input (P0.3, #444): one drift.Withdrawal per departure,
// carrying the subject's first appearance and its withdrawal instant, from which
// time-to-withdrawal is derived. A row with an unknown appearance or withdrawal is
// carried through and dropped by the fold (Withdrawal.Duration), never fabricated
// into a zero interval.
func (s *server) withdrawalLifespans(ctx context.Context, since time.Time) ([]drift.Withdrawal, error) {
	rows, err := s.store.ListWithdrawalLifespans(ctx, pgtype.Timestamptz{Time: since, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]drift.Withdrawal, 0, len(rows))
	for _, row := range rows {
		w := drift.Withdrawal{}
		if row.FirstOpened.Valid {
			w.Appeared = row.FirstOpened.Time.UTC()
		}
		if row.WithdrawnAt.Valid {
			w.Withdrawn = row.WithdrawnAt.Time.UTC()
		}
		out = append(out, w)
	}
	return out, nil
}

// reportsDurationDays renders a duration as the console's terse day figure ("2.4d"),
// the form the mean-time-to-withdrawal KPI reads (Reports.jsx). A sub-day mean still
// renders in days ("0.4d") so the KPI keeps one unit.
func reportsDurationDays(d time.Duration) string {
	return strconv.FormatFloat(d.Hours()/24, 'f', 1, 64) + "d"
}

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

	// The selected range — a week span from the offered set, defaulting to twelve.
	// It sets the heatmap span AND the "Scans run" window KPI so both read the same
	// window; an un-parameterised request stays exactly the twelve-week default.
	weeks := resolveReportsWeeks(r)
	days := weeks * 7

	// Operational activity — the scans-per-day heatmap and the two activity KPIs.
	// A Dispatch read failure degrades to an empty heatmap rather than failing the
	// whole analytics page a viewer depends on.
	cells, heatTotal, scansWindow, activeScans := []heatCell{}, 0, 0, 0
	rows, err := s.store.ListDispatchProgress(ctx, reportsDispatchLimit(weeks))
	if err != nil {
		log.Printf("web: reports: list dispatch progress: %v", err)
	} else {
		cells, heatTotal, scansWindow, activeScans = s.foldScanActivity(rows, days)
	}

	// The headline KPI — the current count of firing signals. It is a current-state
	// census total (not a trend), so it is the one honest signal figure. Building
	// the corpus is the heavier read; on failure the KPI degrades to unavailable
	// rather than 500ing the page.
	openSignals, hasOpenSignals := s.openSignalsCount(r)

	// Signals-over-time — the design's "Open signals over time" line and its
	// "Critical + high" companion (Reports.jsx), folded from the per-instance
	// first-seen ledger over the selected range in weekly buckets (P0.3, #444). A
	// ledger read failure degrades to an empty series rather than failing the page.
	now := s.now().UTC()
	var signalTrend []drift.SignalPoint
	hasSignalTrend := false
	if raises, rerr := s.signalRaises(ctx); rerr != nil {
		log.Printf("web: reports: signal raises: %v", rerr)
	} else {
		signalTrend = drift.SignalsOverTime(raises, now, reportsTrendBucket, weeks)
		for _, p := range signalTrend {
			if p.Standing > 0 || p.Count > 0 {
				hasSignalTrend = true
				break
			}
		}
	}

	// Mean-time-to-withdrawal — the KPI in the mock's mean-time-to-resolve slot, now
	// honest to the domain (signals are withdrawn by the world, never resolved) — and
	// its trend, both derived from the subject-withdrawal history in the span corpus
	// (P0.3, #444). A read failure degrades the KPI to unavailable and the trend to
	// empty rather than 500ing the page.
	windowStart := now.Add(-reportsTrendBucket * time.Duration(weeks))
	var withdrawalTrend []drift.WithdrawalPoint
	mttw, hasMTTW := time.Duration(0), false
	if ws, werr := s.withdrawalLifespans(ctx, windowStart); werr != nil {
		log.Printf("web: reports: withdrawal lifespans: %v", werr)
	} else {
		withdrawalTrend = drift.WithdrawalSeries(ws, now, reportsTrendBucket, weeks)
		mttw, hasMTTW = drift.MeanTimeToWithdrawal(ws)
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

		// Trend series (P0.3, #444) — the datum P2.4 paints. SignalTrend is the
		// signals-over-time line + its critical-high companion; MTTW is the
		// mean-time-to-withdrawal KPI figure and WithdrawalTrend its sparkline. Each
		// carries a Has* flag so an unavailable read renders the design's own empty
		// pattern rather than a fabricated zero.
		"SignalTrend":     signalTrend,
		"HasSignalTrend":  hasSignalTrend,
		"WithdrawalTrend": withdrawalTrend,
		"MTTW":            reportsDurationDays(mttw),
		"HasMTTW":         hasMTTW,

		// Range control + range-aware labels. RangeWeeks drives the export link's
		// carried param; RangeLabel re-skins the twelve-week captions to the active
		// span; RangeOptions renders the header select.
		"RangeWeeks":   weeks,
		"RangeLabel":   reportsRangeLabel(weeks),
		"RangeOptions": reportsRangeOptions(weeks),

		// Recurring reports. Scheduling has no backend yet (#290/#291), so this is
		// empty and the table renders the empty-state; the row-menu "View last
		// delivery" shape is ported now and lights up per report when it lands.
		"Schedules": s.reportScheduleRows(ctx),
	})
}

// bucketScanActivity is the shared core behind both the heatmap fold and the export:
// it buckets Dispatches into per-day counts over the given span (oldest-first, index
// 0 = the oldest day, last = today, UTC whole-day offsets) and derives the window
// and in-flight totals. The window counts Dispatches dated inside the span; active
// counts those with jobs still ready or running, regardless of date. days is the
// span in whole days (weeks * 7).
func (s *server) bucketScanActivity(rows []db.ListDispatchProgressRow, days int) (counts []int, window, active int) {
	const day = 24 * time.Hour
	today := s.now().UTC().Truncate(day)
	counts = make([]int, days)
	for _, row := range rows {
		if row.Ready+row.Running > 0 {
			active++
		}
		if !row.CreatedAt.Valid {
			continue
		}
		created := row.CreatedAt.Time.UTC().Truncate(day)
		offset := int(today.Sub(created).Hours() / 24)
		if offset < 0 || offset >= days {
			continue
		}
		counts[days-1-offset]++ // index 0 = oldest, last = today
		window++
	}
	return counts, window, active
}

// foldScanActivity buckets recent Dispatches into the scans-per-day heatmap and the
// two activity KPIs over the selected span (days = weeks * 7). Each Dispatch is
// placed by the whole-day offset of its creation from today (UTC), oldest-first; the
// window KPI counts the Dispatches inside the span, and the active KPI those with
// jobs still ready or running. Intensity ramps on --chart-1 in four steps, matching
// HeatmapCalendar. Passing reportsHeatDays reproduces the original twelve-week fold.
func (s *server) foldScanActivity(rows []db.ListDispatchProgressRow, days int) (cells []heatCell, total, window, active int) {
	counts, window, active := s.bucketScanActivity(rows, days)

	for _, c := range counts {
		total += c
	}

	// Intensity steps mirror HeatmapCalendar.jsx: 0/28/48/72/100% of --chart-1
	// mixed into --surface, with the sunken step at zero. The 0..4 level per day is
	// the shared scans-per-day ramp (internal/drift.HeatLevels, P0.3), so the page,
	// the export and any later surface intensify identically off one rule.
	pct := []int{0, 28, 48, 72, 100}
	levels := drift.HeatLevels(counts)
	cells = make([]heatCell, days)
	for i, c := range counts {
		level := levels[i]
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

// reportDeliveryPDF serves the delivered report as a PDF download (#345) — the
// print form of the same Artifact reportDeliveryPage renders on screen, produced
// by internal/message.RenderArtifactPDF (a pure-Go render that runs inside the
// distroless-static web image, no external renderer). It reads the same Artifact
// this handler pair shares, so the download always mirrors what the page shows;
// with no delivery backend yet (#285; #290/#291) that is the empty-state document,
// and when a delivery store lands both handlers fill the same struct with real
// data and the download follows for free. A viewer reads it — a delivered report
// is a record, not a mutation.
func (s *server) reportDeliveryPDF(w http.ResponseWriter, r *http.Request, acct db.Account) {
	// No delivery backing store yet — the zero Artifact renders the empty-state,
	// exactly as the on-screen view does. Never fabricate a document.
	art := message.Artifact{}

	pdf, err := message.RenderArtifactPDF(art)
	if err != nil {
		log.Printf("web: report delivery pdf: render: %v", err)
		http.Error(w, "could not render report PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	// With no delivery backend the document names no delivery window, so the
	// download is the generic report-delivery.pdf. A period-dated name (as the
	// csv/json exports carry) belongs with the delivery backend that gives the
	// Artifact a window to name (#285; #290/#291), not ahead of it.
	w.Header().Set("Content-Disposition", `attachment; filename="report-delivery.pdf"`)
	if _, err := w.Write(pdf); err != nil {
		log.Printf("web: report delivery pdf: write: %v", err)
	}
}
