package main

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
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

// reportsPeriod is one entry of the /reports range picker (SPEC-CHANGE #23b): the
// ?period token, its badge label, and the internal WEEK span the KPI band, trend
// chart, discovery bars and scans-per-day heatmap all read over. The design's period
// vocabulary (fixtures.json → reports.periods) is 24h/7d/30d/90d; each maps to a
// whole-week span for the folds — 7d is the default and maps to the twelve-week heat
// span (reportsHeatWeeks), so an un-parameterised request renders exactly the design's
// default (range_label "Last 7d", range_weeks 12, an 84-cell heat). The design itself
// labels that twelve-week activity view "Last 7d", so this mapping is faithful; the
// wider presets extend the span. The old ?weeks= param and its select retire (#23b).
type reportsPeriod struct {
	Token string
	Label string
	Weeks int
}

// reportsPeriods is the fixed preset vocabulary the range picker offers, in the
// design's authored order (fixtures.json → reports.periods).
func reportsPeriods() []reportsPeriod {
	return []reportsPeriod{
		{Token: "24h", Label: "Last 24h", Weeks: 4},
		{Token: "7d", Label: "Last 7d", Weeks: reportsHeatWeeks},
		{Token: "30d", Label: "Last 30d", Weeks: 26},
		{Token: "90d", Label: "Last 90d", Weeks: 52},
	}
}

// reportsDefaultPeriod is the preset selected when no ?period token is given — 7d, the
// design's default (fixtures.json → reports.period).
const reportsDefaultPeriod = "7d"

// resolveReportsPeriod maps the ?period query to a preset, defaulting to
// reportsDefaultPeriod for an absent or unrecognised token (a hand-crafted value never
// widens the window past the offered set).
func resolveReportsPeriod(token string) reportsPeriod {
	for _, p := range reportsPeriods() {
		if p.Token == token {
			return p
		}
	}
	for _, p := range reportsPeriods() {
		if p.Token == reportsDefaultPeriod {
			return p
		}
	}
	return reportsPeriods()[0]
}

// reportsCustomPrefix marks a custom-range period token — "custom_<start>_<end>" with
// ISO (YYYY-MM-DD) bounds (the drift #20b mechanism). The range popover submits GET
// /reports?start=&end=; the page stamps this stable token into .Period so the export
// links (/reports/export?period=) carry the same window.
const reportsCustomPrefix = "custom_"

// parseReportsCustomToken splits a "custom_<start>_<end>" token back into its ISO bounds.
func parseReportsCustomToken(token string) (start, end string, ok bool) {
	rest, found := strings.CutPrefix(token, reportsCustomPrefix)
	if !found {
		return "", "", false
	}
	start, end, found = strings.Cut(rest, "_")
	if !found || start == "" || end == "" {
		return "", "", false
	}
	return start, end, true
}

// reportsWindow is the resolved reporting window: the stable .Period token and
// .PeriodLabel the tmpl renders, and the internal week span the folds read over.
type reportsWindow struct {
	Token string
	Label string
	Weeks int
}

// resolveReportsWindow resolves the request into the reporting window. A custom range —
// an explicit ?start=&end= from the popover, or a "custom_" ?period token from an export
// link — resolves to an absolute [start, end] window with a stable token, a "start – end"
// label, and a week span derived from the span (rounded up, min one week). Otherwise a
// preset resolves to its span. A malformed custom pair falls back to the preset path, so
// a hand-crafted query never errors the page.
func resolveReportsWindow(r *http.Request) reportsWindow {
	q := r.URL.Query()
	start, end := q.Get("start"), q.Get("end")
	if start == "" && end == "" {
		if st, en, ok := parseReportsCustomToken(q.Get("period")); ok {
			start, end = st, en
		}
	}
	if start != "" && end != "" {
		sd, e1 := time.Parse("2006-01-02", start)
		ed, e2 := time.Parse("2006-01-02", end)
		if e1 == nil && e2 == nil && !ed.Before(sd) {
			days := int(ed.Sub(sd).Hours()/24) + 1
			weeks := (days + 6) / 7
			if weeks < 1 {
				weeks = 1
			}
			return reportsWindow{Token: reportsCustomPrefix + start + "_" + end, Label: start + " – " + end, Weeks: weeks}
		}
	}
	p := resolveReportsPeriod(q.Get("period"))
	return reportsWindow{Token: p.Token, Label: p.Label, Weeks: p.Weeks}
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

// reportsDiscoveryBucket is the discovery series' bucket width — one DAY, matching
// the spec's daily-discovery BarChart (Reports.jsx DISCOVERY is one bar per day).
// The series carries `days` (weeks * 7) buckets over the selected range, so a
// discovery bar lines up column-for-column with a scans-per-day heatmap cell.
const reportsDiscoveryBucket = 24 * time.Hour

// firstAppearances reads the Name/Service first-appearance ledger since a read
// instant into the discovery fold's input (P2.4b, #468): one drift.Appearance per
// subject, carrying its earliest span opened_at (the `appeared` classification) and
// whether it is a service, so the fold derives the per-period count, its name/
// service split, and the daily-discovery series. Read back two windows by the caller
// so the previous equal period is comparable for the vs-previous-period delta.
func (s *server) firstAppearances(ctx context.Context, since time.Time) ([]drift.Appearance, error) {
	rows, err := s.store.ListSubjectFirstAppearances(ctx, pgtype.Timestamptz{Time: since, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]drift.Appearance, 0, len(rows))
	for _, row := range rows {
		if !row.FirstOpened.Valid {
			continue
		}
		out = append(out, drift.Appearance{
			At:      row.FirstOpened.Time.UTC(),
			Service: row.SubjectKind == "service",
		})
	}
	return out, nil
}

// reportsBar is one bar of the daily-discovery BarChart: its height as a percentage
// of the busiest day, whether it is the emphasised last (today) bar, and a hover
// title (the day's discovery count). Height is folded in the handler so the template
// stays arithmetic-free, mirroring BarChart.jsx (flex bars, height ∝ value/max,
// non-last bars dimmed).
type reportsBar struct {
	HeightPct int
	Last      bool
	Title     string
}

// reportsBarChart is the server-rendered form of BarChart.jsx for the "New assets
// discovered" card: the per-day bars plus the two span labels the design draws under
// the axis (oldest edge and today).
type reportsBarChart struct {
	Bars       []reportsBar
	LeftLabel  string
	RightLabel string
}

// buildReportsBarChart folds the daily-discovery series into the bar geometry,
// scaling each bar to a percentage of the busiest day (floored at 1 so an all-zero
// series is every bar at its 2px minimum rather than a divide-by-zero), exactly as
// BarChart.jsx does. The last bar (today) is emphasised; the rest are dimmed.
func buildReportsBarChart(points []drift.DiscoveryPoint, weeks int) reportsBarChart {
	max := 1
	for _, p := range points {
		if p.Count > max {
			max = p.Count
		}
	}
	bars := make([]reportsBar, len(points))
	for i, p := range points {
		bars[i] = reportsBar{
			HeightPct: p.Count * 100 / max,
			Last:      i == len(points)-1,
			Title:     pluralAssets(p.Count),
		}
	}
	return reportsBarChart{
		Bars:       bars,
		LeftLabel:  strconv.Itoa(weeks) + "w ago",
		RightLabel: "today",
	}
}

// pluralAssets renders a discovery count as the console's terse asset figure — the
// bar's hover title. "asset" is the UI collective noun for a watched Name/Service.
func pluralAssets(n int) string {
	if n == 1 {
		return "1 asset"
	}
	return strconv.Itoa(n) + " assets"
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

// reportsSevCount is one severity's open-signal tally for the by-severity bars —
// the SevBars region of Reports.jsx, now a real read off the census (P0.1). Label
// is the microlabel form (CRITICAL … INFO), Sev the token key the template branches
// on to pick the severity ramp fill (a literal var() per level, so the style
// sanitiser sees static CSS rather than a blanked mid-token interpolation), and Pct
// the bar width as a share of the busiest level.
type reportsSevCount struct {
	Label string
	Sev   string
	Count int
	Pct   int
}

// reportsSignalCensus evaluates the signal corpus ONCE and returns the three signal
// figures the Reports screen reads off it: the open-signal total (the "Open signals"
// card value), the per-severity tally the by-severity bars paint (P0.1), and the
// fired (rule, subject) pairs the vs-last-batch open-signals delta reconstructs
// against (P0.2). ok is false on a corpus read failure so every signal region
// degrades to its empty pattern rather than a fabricated zero. It supersedes the
// page's openSignalsCount call (the export still shares that thinner helper).
func (s *server) reportsSignalCensus(r *http.Request) (open int, bySeverity []reportsSevCount, fired []firedSignal, ok bool) {
	corpus, err := s.buildSignalCorpus(r)
	if err != nil {
		log.Printf("web: reports: build signal corpus: %v", err)
		return 0, nil, nil, false
	}
	counts := map[signal.Severity]int{}
	for _, c := range signal.EvaluateCorpus(corpus) {
		sev, _ := signal.SeverityFor(c.Rule)
		for _, m := range c.Fired {
			open++
			counts[sev]++
			fired = append(fired, firedSignal{Rule: c.Rule, Subject: m.Subject})
		}
	}
	max := 1
	for _, sev := range signal.SevOrder {
		if counts[sev] > max {
			max = counts[sev]
		}
	}
	bySeverity = make([]reportsSevCount, 0, len(signal.SevOrder))
	for _, sev := range signal.SevOrder {
		n := counts[sev]
		bySeverity = append(bySeverity, reportsSevCount{
			Label: strings.ToUpper(string(sev)),
			Sev:   string(sev),
			Count: n,
			Pct:   n * 100 / max,
		})
	}
	return open, bySeverity, fired, true
}

// reportsDelta is a KPI card's vs-last-batch delta prepared for the template: the
// signed text ("+3", "−2", "−0.6d"), the semantic tone (good/bad/neutral,
// the DeltaChip colours) and the arrow direction, or Has=false where no previous
// batch exists to compare against — the design's no-delta state (P0.2), never a
// fabricated +0.
type reportsDelta struct {
	Has  bool
	Text string
	Tone string
	Dir  string
}

// signedCount renders a signed integer delta with a true minus and its arrow
// direction: "+3"/up, "−2"/down, "0"/none.
func signedCount(n int) (text, dir string) {
	switch {
	case n > 0:
		return "+" + strconv.Itoa(n), "up"
	case n < 0:
		return "−" + strconv.Itoa(-n), "down"
	default:
		return "0", ""
	}
}

// buildMTTWDelta is the mean-time-to-withdrawal card's delta: this window's mean
// minus the previous equal window's, in days to one place. A shorter time to
// withdrawal is the good direction (the estate cleared its exposure faster), so a
// negative change is toned good.
func buildMTTWDelta(cur, prev time.Duration) reportsDelta {
	diff := math.Round((cur.Hours()/24-prev.Hours()/24)*10) / 10
	switch {
	case diff > 0:
		return reportsDelta{Has: true, Text: "+" + strconv.FormatFloat(diff, 'f', 1, 64) + "d", Tone: "bad", Dir: "up"}
	case diff < 0:
		return reportsDelta{Has: true, Text: "−" + strconv.FormatFloat(-diff, 'f', 1, 64) + "d", Tone: "good", Dir: "down"}
	default:
		return reportsDelta{Has: true, Text: "0d", Tone: "neutral", Dir: ""}
	}
}

// reportsOpenDelta builds the "Open signals" KPI-band delta — the current firing
// census vs those already firing a batch ago (more signals is the bad direction),
// vs-last-batch (P0.2), reconstructed from the same span/first-seen history the
// Dashboard reads. fired is the census the caller already evaluated, so the corpus
// is not re-folded here. Where no previous batch exists, or a read fails, it returns
// Has=false so the card renders its no-delta state rather than a fabricated +0. The
// KPI band's second card (New assets discovered) derives its own count/delta from
// the first-appearance ledger (reportsPage), not from this vs-last-batch helper.
func (s *server) reportsOpenDelta(ctx context.Context, fired []firedSignal) (open reportsDelta) {
	prevAt, ok, err := s.previousBatchInstant(ctx)
	if err != nil {
		log.Printf("web: reports: previous batch instant: %v", err)
		return
	}
	if !ok {
		return
	}
	od, _, derr := s.signalDeltas(ctx, fired, prevAt)
	if derr != nil {
		log.Printf("web: reports: signal deltas: %v", derr)
		return
	}
	otext, odir := signedCount(od.Change())
	tone := "neutral"
	switch {
	case od.Change() > 0:
		tone = "bad"
	case od.Change() < 0:
		tone = "good"
	}
	open = reportsDelta{Has: true, Text: otext, Tone: tone, Dir: odir}
	return
}

// sparkline is a KPI card's inline trend, pre-computed to plain SVG geometry the
// template paints with no arithmetic — the server-rendered form of Sparkline.jsx.
// Colour is the chart series token (never severity); it rides on the struct so a
// card picks --chart-1 or --chart-2 without a second template.
type sparkline struct {
	W, H       int
	Line       string
	Area       string
	DotX, DotY string
	Color      string
}

// f1 formats an SVG coordinate to one decimal place — terse, stable output for the
// points/path strings.
func f1(x float64) string { return strconv.FormatFloat(x, 'f', 1, 64) }

// buildSparkline folds a value series into the Sparkline geometry (polyline, area
// fill, last-point dot) scaled to w×h, mirroring Sparkline.jsx. ok is false for a
// series of fewer than two points, so the card omits the chart (the component's own
// too-short-to-draw form) rather than inventing a shape.
func buildSparkline(data []float64, w, h int, color string) (sparkline, bool) {
	if len(data) < 2 {
		return sparkline{}, false
	}
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	span := max - min
	if span == 0 {
		span = 1
	}
	const pad = 3.0
	fw, fh := float64(w), float64(h)
	px := func(i int) float64 { return pad + float64(i)/float64(len(data)-1)*(fw-pad*2) }
	py := func(v float64) float64 { return pad + (1-(v-min)/span)*(fh-pad*2) }
	var line strings.Builder
	for i, v := range data {
		if i > 0 {
			line.WriteByte(' ')
		}
		line.WriteString(f1(px(i)))
		line.WriteByte(',')
		line.WriteString(f1(py(v)))
	}
	area := "M" + f1(pad) + "," + f1(fh-pad) + " L" + strings.ReplaceAll(line.String(), " ", " L") +
		" L" + f1(px(len(data)-1)) + "," + f1(fh-pad) + " Z"
	return sparkline{
		W: w, H: h, Line: line.String(), Area: area,
		DotX: f1(px(len(data) - 1)), DotY: f1(py(data[len(data)-1])), Color: color,
	}, true
}

// standingSeries lifts the standing (open-at-close) level of each signals-over-time
// bucket into a float series for the "Open signals" card sparkline.
func standingSeries(pts []drift.SignalPoint) []float64 {
	out := make([]float64, len(pts))
	for i, p := range pts {
		out[i] = float64(p.Standing)
	}
	return out
}

// meanDaysSeries lifts each withdrawal bucket's mean time-to-withdrawal (in days)
// into a float series for the MTTW card sparkline, SKIPPING gap buckets (HasMean
// false) so a bucket with no withdrawal draws no fabricated zero point.
func meanDaysSeries(pts []drift.WithdrawalPoint) []float64 {
	out := make([]float64, 0, len(pts))
	for _, p := range pts {
		if p.HasMean {
			out = append(out, p.Mean.Hours()/24)
		}
	}
	return out
}

// reportsGridLine / reportsXLabel are one pre-positioned axis element of the big
// "Open signals over time" chart — every coordinate resolved in the handler so the
// template ranges with no arithmetic.
type reportsGridLine struct {
	X1, X2, Y, LabelX, Label, Stroke string
}
type reportsXLabel struct {
	X, Y, Text string
}

// reportsTimeSeries is the server-rendered form of TimeSeriesChart.jsx for the
// "Open signals over time" card: a fixed viewBox (scaled to width via the SVG),
// nice-stepped y gridlines, sparse x labels, and the two standing series — All open
// (--chart-1) and Critical + high (--chart-2). Every string is paint-ready.
type reportsTimeSeries struct {
	W, H     int
	Grid     []reportsGridLine
	XLabels  []reportsXLabel
	AllOpen  string
	CritHigh string
	// N, LabelsAttr and SeriesJSON feed the frozen tmpl's hover-readout JS (#23e): the
	// point count, a pipe-joined per-point label list (data-labels) and the two series as
	// a JSON array (data-series) the crosshair reads — the same math the polylines draw,
	// no new geometry.
	N          int
	LabelsAttr string
	SeriesJSON string
}

// reportsHoverSeries is one series in the chart's data-series JSON (the hover readout):
// its legend label, its chart colour token, and the per-point standing values.
type reportsHoverSeries struct {
	Label string `json:"label"`
	Color string `json:"color"`
	Data  []int  `json:"data"`
}

// buildReportsTimeSeries folds the signals-over-time buckets into the chart geometry,
// choosing a nice y-axis step exactly as TimeSeriesChart.jsx does. ok is false where
// the standing level never rises above zero, so the card renders the design's empty
// pattern rather than a flat line on a fabricated axis.
func buildReportsTimeSeries(pts []drift.SignalPoint) (reportsTimeSeries, bool) {
	n := len(pts)
	if n < 2 {
		return reportsTimeSeries{}, false
	}
	max0 := 0
	for _, p := range pts {
		if p.Standing > max0 {
			max0 = p.Standing
		}
	}
	if max0 <= 0 {
		return reportsTimeSeries{}, false
	}
	raw := float64(max0) / 4
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	norm := raw / mag
	var stepN float64
	switch {
	case norm <= 1:
		stepN = 1
	case norm <= 2:
		stepN = 2
	case norm <= 5:
		stepN = 5
	default:
		stepN = 10
	}
	step := stepN * mag
	top := step
	if c := math.Ceil(float64(max0)/step) * step; c > top {
		top = c
	}
	const (
		W, H       = 860, 230
		PL, PR     = 40, 8
		PT, PB     = 10, 22
		labelX     = PL - 8
		axisRightX = W - PR
	)
	x := func(i int) float64 { return PL + float64(i)/float64(n-1)*(W-PL-PR) }
	y := func(v float64) float64 { return PT + (1-v/top)*(H-PT-PB) }
	var grid []reportsGridLine
	for v := 0.0; v <= top+1e-9; v += step {
		yy := f1(y(v))
		stroke := "var(--sunken)"
		if v == 0 {
			stroke = "var(--hairline)"
		}
		grid = append(grid, reportsGridLine{
			X1: strconv.Itoa(PL), X2: strconv.Itoa(axisRightX), Y: yy,
			LabelX: strconv.Itoa(labelX), Label: strconv.Itoa(int(v + 0.5)), Stroke: stroke,
		})
	}
	build := func(get func(p drift.SignalPoint) int) string {
		var b strings.Builder
		for i, p := range pts {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(f1(x(i)))
			b.WriteByte(',')
			b.WriteString(f1(y(float64(get(p)))))
		}
		return b.String()
	}
	xLabelY := strconv.Itoa(H - 6)
	xLabels := []reportsXLabel{
		{X: f1(x(0)), Y: xLabelY, Text: strconv.Itoa(n) + "w ago"},
		{X: f1(x(n - 1)), Y: xLabelY, Text: "now"},
	}
	// The hover-readout inputs (#23e): a label per point and the two standing series as a
	// JSON array the crosshair JS reads (data-labels / data-series). The labels index the
	// buckets (weekly), enough for the tooltip's per-point title; the values are the same
	// standing figures the polylines plot.
	allData := make([]int, n)
	critData := make([]int, n)
	labels := make([]string, n)
	for i, p := range pts {
		allData[i] = p.Standing
		critData[i] = p.StandingElevated
		labels[i] = "wk " + strconv.Itoa(i+1)
	}
	seriesJSON, _ := json.Marshal([]reportsHoverSeries{
		{Label: "All open", Color: "var(--chart-1)", Data: allData},
		{Label: "Critical + high", Color: "var(--chart-2)", Data: critData},
	})
	return reportsTimeSeries{
		W: W, H: H, Grid: grid, XLabels: xLabels,
		AllOpen:    build(func(p drift.SignalPoint) int { return p.Standing }),
		CritHigh:   build(func(p drift.SignalPoint) int { return p.StandingElevated }),
		N:          n,
		LabelsAttr: strings.Join(labels, "|"),
		SeriesJSON: string(seriesJSON),
	}, true
}

func (s *server) reportsPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// VERGE_DEV pixel-parity path (#588): serve the pinned fixtures.json reports slice so
	// the seeded instance renders byte-for-byte what the golden composes (as the sibling
	// screens do). The exact spark/series geometry, the 84-cell heat, the delta chips and
	// the schedule rows are the design's curated fixture — they cannot be reconstructed
	// from live derivations without fabricating domain data, which SPEC-CHANGE forbids. A
	// real deployment (devMode == false) falls through to the honest live reads below.
	if s.devMode {
		s.render(w, "reports", s.reportsFixtureData(acct))
		return
	}

	// The selected period — a preset token (SPEC-CHANGE #23b) or a custom ISO range —
	// resolving to a WEEK span that sets the heatmap span, the trend-series window AND the
	// export window so all read the same period; an un-parameterised request stays the
	// design's default (7d → twelve weeks).
	window := resolveReportsWindow(r)
	weeks := window.Weeks
	days := weeks * 7

	// The range's window bounds, shared by the trend folds (P0.3) and the discovery
	// count (P2.4b): `windowStart` opens the selected period, `doubleStart` the equal
	// period before it — the ledgers are read back to doubleStart so each card's
	// vs-previous-period delta is comparable, then split at windowStart.
	now := s.now().UTC()
	windowStart := now.Add(-reportsTrendBucket * time.Duration(weeks))
	doubleStart := now.Add(-reportsTrendBucket * time.Duration(2*weeks))

	// Operational activity — the scans-per-day heatmap. A Dispatch read failure
	// degrades to an empty heatmap rather than failing the whole analytics page a
	// viewer depends on. The window/in-flight scalars the export carries are folded
	// in reports_export.go; the band no longer shows them (the spec's band is three
	// trend cards, not operational counters — no added affordance, ADR-0116).
	cells, heatTotal := []heatCell{}, 0
	if rows, err := s.store.ListDispatchProgress(ctx, reportsDispatchLimit(weeks)); err != nil {
		log.Printf("web: reports: list dispatch progress: %v", err)
	} else {
		cells, heatTotal, _, _ = s.foldScanActivity(rows, days)
	}

	// Signal census — the open-signal total (the "Open signals" card value), the
	// per-severity bars (P0.1) and the fired pairs the open-signals delta reads
	// against — all off ONE corpus evaluation. A failure degrades every signal region
	// to its empty pattern rather than a fabricated zero.
	openSignals, bySeverity, fired, hasOpenSignals := s.reportsSignalCensus(r)

	// Open-signals vs-last-batch delta for the KPI band's first card (P0.2).
	openDelta := s.reportsOpenDelta(ctx, fired)

	// New assets discovered — the KPI band's second card (P2.4b, #468). The count of
	// Name/Service subjects that FIRST appeared in the selected range, its
	// vs-previous-period signed delta, its name/service split, and the daily-discovery
	// bar series (Reports.jsx DISCOVERY). Derived from the span corpus's per-subject
	// first appearance (MIN(opened_at), the `appeared` classification): the ledger is
	// read back two windows so the previous equal window is comparable, then split at
	// the window start. A read failure degrades the card to the ReportCard's empty
	// pattern rather than 500ing the page (PARITY-CHART acceptance #7).
	discoveryCount, discoveryNames, discoverySvcs := 0, 0, 0
	var discoveryDelta reportsDelta
	var discoveryBars reportsBarChart
	hasDiscovery := false
	if apps, aerr := s.firstAppearances(ctx, doubleStart); aerr != nil {
		log.Printf("web: reports: first appearances: %v", aerr)
	} else {
		cur := drift.DiscoveryCount(apps, windowStart, now)
		prev := drift.DiscoveryCount(apps, doubleStart, windowStart)
		dtext, ddir := signedCount(cur.Total - prev.Total)
		discoveryDelta = reportsDelta{Has: true, Text: dtext, Tone: "neutral", Dir: ddir}
		discoveryBars = buildReportsBarChart(drift.DiscoverySeries(apps, now, reportsDiscoveryBucket, days), weeks)
		discoveryCount, discoveryNames, discoverySvcs = cur.Total, cur.Names, cur.Services
		hasDiscovery = true
	}

	// Signals-over-time — the design's "Open signals over time" line and its
	// "Critical + high" companion (Reports.jsx), folded from the per-instance
	// first-seen ledger over the selected range in weekly buckets (P0.3, #444). It
	// paints both the big trend chart and the "Open signals" card sparkline. A ledger
	// read failure degrades to an empty series rather than failing the page.
	var signalPoints []drift.SignalPoint
	hasSignalTrend := false
	if raises, rerr := s.signalRaises(ctx); rerr != nil {
		log.Printf("web: reports: signal raises: %v", rerr)
	} else {
		signalPoints = drift.SignalsOverTime(raises, now, reportsTrendBucket, weeks)
		for _, p := range signalPoints {
			if p.Standing > 0 || p.Count > 0 {
				hasSignalTrend = true
				break
			}
		}
	}
	signalSpark, hasSignalSpark := buildSparkline(standingSeries(signalPoints), 300, 46, "var(--chart-1)")
	signalSeries, hasSignalSeries := buildReportsTimeSeries(signalPoints)
	hasSignalSpark = hasSignalSpark && hasSignalTrend

	// Mean-time-to-withdrawal — the KPI in the mock's mean-time-to-resolve slot, now
	// honest to the domain (signals are withdrawn by the world, never resolved) — its
	// sparkline, and a vs-previous-window delta (Reports.jsx shows "−0.6d"). The
	// window mean averages withdrawals that occurred in the selected range; the delta
	// compares it against the previous equal window, so the ledger is read back two
	// windows and split. A read failure degrades the KPI to unavailable, the trend to
	// empty and the delta away, rather than 500ing the page (P0.3, #444).
	var withdrawalPoints []drift.WithdrawalPoint
	mttw, hasMTTW := "—", false
	var mttwDelta reportsDelta
	if ws, werr := s.withdrawalLifespans(ctx, doubleStart); werr != nil {
		log.Printf("web: reports: withdrawal lifespans: %v", werr)
	} else {
		withdrawalPoints = drift.WithdrawalSeries(ws, now, reportsTrendBucket, weeks)
		var recent, prior []drift.Withdrawal
		for _, wd := range ws {
			if wd.Withdrawn.Before(windowStart) {
				prior = append(prior, wd)
			} else {
				recent = append(recent, wd)
			}
		}
		if m, ok := drift.MeanTimeToWithdrawal(recent); ok {
			mttw, hasMTTW = reportsDurationDays(m), true
			if pm, pok := drift.MeanTimeToWithdrawal(prior); pok {
				mttwDelta = buildMTTWDelta(m, pm)
			}
		}
	}
	mttwSpark, hasMTTWSpark := buildSparkline(meanDaysSeries(withdrawalPoints), 300, 46, "var(--chart-2)")

	s.render(w, "reports", map[string]any{
		"Title": "Reports", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "reports", "DesignTokens": true,

		// KPI band — the three trend cards of Reports.jsx. Each renders real computed
		// values with its vs-last-batch delta (P0.2) and inline trend (P0.3), and
		// degrades to the ReportCard's own no-delta / no-chart form where a read is
		// unavailable — never a fabricated figure (PARITY-CHART acceptance #7).
		"OpenSignals":    openSignals,
		"HasOpenSignals": hasOpenSignals,
		"OpenDelta":      openDelta,
		"OpenSpark":      signalSpark,
		"HasOpenSpark":   hasSignalSpark,

		// New assets discovered (P2.4b, #468): the per-period count, its name/service
		// split caption, the vs-previous-period delta, and the daily-discovery bars.
		"DiscoveryCount":    discoveryCount,
		"DiscoveryNames":    discoveryNames,
		"DiscoveryServices": discoverySvcs,
		"HasDiscovery":      hasDiscovery,
		"DiscoveryDelta":    discoveryDelta,
		"DiscoveryBars":     discoveryBars,

		"MTTW":        mttw,
		"HasMTTW":     hasMTTW,
		"MTTWDelta":   mttwDelta,
		"MTTWSpark":   mttwSpark,
		"HasMTTWSpark": hasMTTWSpark,

		// "Open signals over time" — the big trend chart.
		"SignalSeries":    signalSeries,
		"HasSignalSeries": hasSignalSeries,

		// By-severity bars (P0.1). Rendered only where signals are firing; an all-clear
		// estate draws the design's empty pattern.
		"BySeverity":  bySeverity,
		"HasSeverity": hasOpenSignals && openSignals > 0,

		// Scans-per-day heatmap.
		"Heat":    cells,
		"HasHeat": heatTotal > 0,

		// Period picker + period-aware labels (#23b). Periods renders the preset links,
		// Period is the active token the export links carry, PeriodLabel/RangeLabel re-skin
		// the header and captions to the active period, and RangeWeeks drives the heat
		// legend's "N weeks ago" (the fixed twelve-week activity span).
		"RangeWeeks":  weeks,
		"RangeLabel":  window.Label,
		"Periods":     reportsPeriods(),
		"Period":      window.Token,
		"PeriodLabel": window.Label,

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

// reportDeliveryPage renders the account's latest delivered report artifact — the
// stable `/reports/delivery` view Reports' "view last delivery" links to (T17). The
// delivered document itself is rendered by internal/message's RenderArtifact, the
// one canonical rendered form that doubles as the PDF / email spec; this handler
// composes the console chrome around it.
//
// With the report_delivery receipts store in place (#291/T2) the handler reads the
// most-recent non-failed delivery and recomputes its contents from the run's period
// bounds (T1 ruling: the receipt snapshots nothing). Where no schedule has ever
// delivered, the zero Artifact renders the design-system empty-state inside the
// delivered-document frame (ADR-0110) and the header falls back to a generic
// heading — nothing is fabricated.
func (s *server) reportDeliveryPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	art := s.reportDeliveryArtifact(r.Context())

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
// this handler pair shares (reportDeliveryArtifact), so the download always mirrors
// what the page shows; where no schedule has delivered that is the empty-state
// document. A viewer reads it — a delivered report is a record, not a mutation.
func (s *server) reportDeliveryPDF(w http.ResponseWriter, r *http.Request, acct db.Account) {
	art := s.reportDeliveryArtifact(r.Context())

	pdf, err := message.RenderArtifactPDF(art)
	if err != nil {
		log.Printf("web: report delivery pdf: render: %v", err)
		http.Error(w, "could not render report PDF", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+reportDeliveryPDFName(art)+`"`)
	if _, err := w.Write(pdf); err != nil {
		log.Printf("web: report delivery pdf: write: %v", err)
	}
}

// reportDeliveryPDFName is the PDF download name: a period-dated
// report-<start>-to-<end>.pdf where the delivery names a window (mirroring the
// operational export's ISO-dated filename, reportsExportFilename), else the generic
// report-delivery.pdf for the empty-state document that names no window.
func reportDeliveryPDFName(a message.Artifact) string {
	if a.PeriodStart != "" && a.PeriodEnd != "" {
		return "report-" + a.PeriodStart + "-to-" + a.PeriodEnd + ".pdf"
	}
	return "report-delivery.pdf"
}

// reportDeliveryArtifact resolves the account's latest delivered report and builds
// the delivered Artifact for its period bounds. The stable /reports/delivery route
// names no schedule (reportDeliveryHref is a constant), so the view opens the single
// most-recent non-failed delivery across every schedule — the receipt the "View last
// delivery" affordance points at. Where no schedule has delivered, the zero Artifact
// renders the design-system empty-state (ADR-0110); nothing is fabricated.
//
// The receipt snapshots no content (T2): the artifact recomputes its signals,
// severity breakdown and withdrawals from the run's [period_start, period_end] bounds
// at render time (T1 ruling), reading the never-deleted first-seen and withdrawal
// ledgers. A list/read failure degrades to the empty-state or an empty section rather
// than 500ing the view a viewer depends on.
func (s *server) reportDeliveryArtifact(ctx context.Context) message.Artifact {
	schedules, err := s.store.ListReportSchedules(ctx)
	if err != nil {
		log.Printf("web: report delivery: list schedules: %v", err)
		return message.Artifact{}
	}
	var (
		best  db.ReportDelivery
		sched db.ReportSchedule
		found bool
	)
	for _, sc := range schedules {
		// The newest non-failed run of each schedule; pgx.ErrNoRows is the genuine
		// never-delivered state and contributes no candidate. Any other read error
		// skips this schedule rather than failing the whole view.
		del, err := s.store.GetLatestReportDelivery(ctx, sc.ID)
		switch {
		case err == nil:
			if !found || del.ID > best.ID {
				best, sched, found = del, sc, true
			}
		case !errors.Is(err, pgx.ErrNoRows):
			log.Printf("web: report delivery: latest for schedule %d: %v", sc.ID, err)
		}
	}
	if !found {
		return message.Artifact{}
	}
	return s.buildReportDeliveryArtifact(ctx, sched, best)
}

// buildReportDeliveryArtifact fills the delivered Artifact for one run: its identity
// (the schedule name and delivered format), its period window and delivery number,
// the receipt footer (the delivery instant and destination HOST only — never the raw
// delivery-target URL where a token an operator embedded would sit, ADR-0081), and
// the period's recomputed signals / severity breakdown / withdrawals (T1 ruling).
func (s *server) buildReportDeliveryArtifact(ctx context.Context, sc db.ReportSchedule, del db.ReportDelivery) message.Artifact {
	art := message.Artifact{
		Title:      sc.Name,
		Format:     sc.Format,
		DeliveryNo: int(del.DeliveryNo),
	}
	if del.PeriodStart.Valid {
		art.PeriodStart = del.PeriodStart.Time.UTC().Format("2006-01-02")
	}
	if del.PeriodEnd.Valid {
		art.PeriodEnd = del.PeriodEnd.Time.UTC().Format("2006-01-02")
	}
	if del.GeneratedAt.Valid {
		art.GeneratedAt = del.GeneratedAt.Time.UTC().Format(time.RFC3339)
	}
	// The receipt footer: the delivery instant and the destination host only. A run
	// that generated without leaving (delivered_at NULL) names no channel, so the
	// footer reads "not delivered" (artifactReceipt).
	if del.DeliveredAt.Valid {
		art.Delivered = del.DeliveredAt.Time.UTC().Format(time.RFC3339)
		art.ChannelHost = deliveryTargetHost(sc.DeliveryTarget)
	}
	// Recompute the period's contents from its bounds — the receipt stores none.
	if del.PeriodStart.Valid && del.PeriodEnd.Valid {
		start, end := del.PeriodStart.Time.UTC(), del.PeriodEnd.Time.UTC()
		art.Signals, art.SeverityCounts = s.reportDeliverySignals(ctx, start, end)
		art.Withdrawn = s.reportDeliveryWithdrawals(ctx, start, end)
	}
	return art
}

// reportDeliverySignals recomputes the delivery period's "new this week" signals
// table and its "open signals by severity" breakdown from the never-deleted
// first-seen ledger (signal_instance): every signal instance whose first-seen falls
// in [start, end], carrying its rule's severity (P0.1, internal/signal). The table is
// ordered most-urgent first, the ramp order (SevOrder); the breakdown carries one
// entry per severity level present. A ledger read failure degrades both to empty
// rather than failing the view.
func (s *server) reportDeliverySignals(ctx context.Context, start, end time.Time) ([]message.ArtifactSignal, []message.ArtifactSeverityCount) {
	rows, err := s.store.ListSignalInstances(ctx)
	if err != nil {
		log.Printf("web: report delivery: list signal instances: %v", err)
		return nil, nil
	}
	counts := map[signal.Severity]int{}
	var sigs []message.ArtifactSignal
	for _, row := range rows {
		if !row.FirstSeen.Valid {
			continue
		}
		at := row.FirstSeen.Time.UTC()
		if at.Before(start) || at.After(end) {
			continue
		}
		// A signal's severity is exactly its rule's severity; an unknown rule folds to
		// the calmest level (SeverityFor), never manufacturing urgency.
		sev, _ := signal.SeverityFor(row.SignalName)
		counts[sev]++
		sigs = append(sigs, message.ArtifactSignal{
			Severity: string(sev),
			Signal:   signalTitle(row.SignalName),
			Asset:    row.SubjectKey,
			Raised:   strings.ToLower(at.Format("Jan 2")),
		})
	}
	sort.SliceStable(sigs, func(i, j int) bool {
		return signal.Severity(sigs[i].Severity).Rank() < signal.Severity(sigs[j].Severity).Rank()
	})
	var bySeverity []message.ArtifactSeverityCount
	for _, sev := range signal.SevOrder {
		if n := counts[sev]; n > 0 {
			bySeverity = append(bySeverity, message.ArtifactSeverityCount{Level: string(sev), Count: n})
		}
	}
	return sigs, bySeverity
}

// reportDeliveryWithdrawals recomputes the period's "withdrawn by the world" section
// from the subject-withdrawal ledger: every subject whose withdrawal instant falls in
// [start, end]. Withdrawal rides the drift vocabulary (never the severity ramp). A
// read failure degrades to an empty section rather than failing the view.
func (s *server) reportDeliveryWithdrawals(ctx context.Context, start, end time.Time) []message.ArtifactChange {
	rows, err := s.store.ListWithdrawalLifespans(ctx, pgtype.Timestamptz{Time: start, Valid: true})
	if err != nil {
		log.Printf("web: report delivery: list withdrawal lifespans: %v", err)
		return nil
	}
	var out []message.ArtifactChange
	for _, row := range rows {
		if !row.WithdrawnAt.Valid {
			continue
		}
		at := row.WithdrawnAt.Time.UTC()
		if at.Before(start) || at.After(end) {
			continue
		}
		out = append(out, message.ArtifactChange{
			Change:  "withdrawn",
			Subject: row.SubjectKey,
			Detail:  strings.ToLower(at.Format("Jan 2")),
		})
	}
	return out
}

// deliveryTargetHost renders a schedule's delivery target as its destination host
// only — never the raw URL, where a token an operator embedded in the path or query
// would sit (mirrors the message panel's host-only rule, ADR-0081). A target that
// does not parse to a host names no channel, so the receipt shows the instant alone.
func deliveryTargetHost(target string) string {
	if u, err := url.Parse(target); err == nil && u.Host != "" {
		return u.Host
	}
	return ""
}
