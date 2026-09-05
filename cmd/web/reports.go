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

// An absent datum draws the design's empty pattern, never a fabricated figure (ADR-0110).

// The Dispatch read is newest-first by id, so a flat cap drops a long range's oldest days.

const reportsDispatchPerWeek = 250

func reportsDispatchLimit(weeks int) int32 {
	return int32(weeks * reportsDispatchPerWeek) // #nosec G115 (weeks bounded by 4-digit-year date parse; weeks*250 well under int32)
}

const (
	reportsHeatWeeks = 12
	reportsHeatDays  = reportsHeatWeeks * 7
)

type reportsPeriod struct {
	Token string
	Label string
	Weeks int
}

func reportsPeriods() []reportsPeriod {
	// The design labels its own twelve-week activity view "Last 7d", so 7d maps to twelve weeks (ADR-0176 §6).
	return []reportsPeriod{
		{Token: "24h", Label: "Last 24h", Weeks: 4},
		{Token: "7d", Label: "Last 7d", Weeks: reportsHeatWeeks},
		{Token: "30d", Label: "Last 30d", Weeks: 26},
		{Token: "90d", Label: "Last 90d", Weeks: 52},
	}
}

const reportsDefaultPeriod = "7d"

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

const reportsCustomPrefix = "custom_"

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

type reportsWindow struct {
	Token string
	Label string
	Weeks int
}

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

// Bucket widths follow the scans-per-day grid, so a series column lines up with a heat cell.

const reportsTrendBucket = 7 * 24 * time.Hour

func (s *server) signalRaises(ctx context.Context) ([]drift.Raise, error) {
	// The whole ledger is read unwindowed, so the standing level counts signals raised before it.
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

const reportsDiscoveryBucket = 24 * time.Hour

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

type reportsBar struct {
	HeightPct int
	Last      bool
	Title     string
}

type reportsBarChart struct {
	Bars       []reportsBar
	LeftLabel  string
	RightLabel string
}

// Past a month of daily bars the row overflowed its card, so a longer range is aggregated.

const reportsMaxBars = 31

func aggregateDiscoveryBars(points []drift.DiscoveryPoint) []drift.DiscoveryPoint {
	if len(points) <= reportsMaxBars {
		return points
	}
	perBucket := 7
	for (len(points)+perBucket-1)/perBucket > reportsMaxBars {
		perBucket += 7
	}
	out := make([]drift.DiscoveryPoint, 0, (len(points)+perBucket-1)/perBucket)
	for i := 0; i < len(points); i += perBucket {
		bucket := drift.DiscoveryPoint{Start: points[i].Start}
		for j := i; j < i+perBucket && j < len(points); j++ {
			bucket.Count += points[j].Count
		}
		out = append(out, bucket)
	}
	return out
}

func buildReportsBarChart(points []drift.DiscoveryPoint, weeks int) reportsBarChart {
	points = aggregateDiscoveryBars(points)
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

func pluralAssets(n int) string {
	if n == 1 {
		return "1 asset"
	}
	return strconv.Itoa(n) + " assets"
}

func reportsDurationDays(d time.Duration) string {
	return strconv.FormatFloat(d.Hours()/24, 'f', 1, 64) + "d"
}

type heatCell struct {
	Bg     template.CSS
	Border template.CSS
	Title  string
}

// A value spliced mid-token is blanked, so the template picks a whole literal var() per level.

type reportsSevCount struct {
	Label string
	Sev   string
	Count int
	Pct   int
}

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

type reportsDelta struct {
	Has  bool
	Text string
	Tone string
	Dir  string
}

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

type sparkline struct {
	W, H       int
	Line       string
	Area       string
	DotX, DotY string
	Color      string
}

func f1(x float64) string { return strconv.FormatFloat(x, 'f', 1, 64) }

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

func standingSeries(pts []drift.SignalPoint) []float64 {
	out := make([]float64, len(pts))
	for i, p := range pts {
		out[i] = float64(p.Standing)
	}
	return out
}

func meanDaysSeries(pts []drift.WithdrawalPoint) []float64 {
	out := make([]float64, 0, len(pts))
	for _, p := range pts {
		if p.HasMean {
			out = append(out, p.Mean.Hours()/24)
		}
	}
	return out
}

type reportsGridLine struct {
	X1, X2, Y, LabelX, Label, Stroke string
}
type reportsXLabel struct {
	X, Y, Text string
}

type reportsTimeSeries struct {
	W, H       int
	Grid       []reportsGridLine
	XLabels    []reportsXLabel
	AllOpen    string
	CritHigh   string
	N          int
	LabelsAttr string
	SeriesJSON string
}

type reportsHoverSeries struct {
	Label string `json:"label"`
	Color string `json:"color"`
	Data  []int  `json:"data"`
}

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
		stroke := "var(--row-sep)"
		if v == 0 {
			stroke = "var(--border-default)"
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

	if s.devMode {
		s.render(w, r, "reports", s.reportsFixtureData(acct))
		return
	}

	// The export link carries this token, so the page and the export read the same window (ADR-0176 §4).
	window := resolveReportsWindow(r)
	weeks := window.Weeks
	days := weeks * 7

	now := s.now().UTC()
	windowStart := now.Add(-reportsTrendBucket * time.Duration(weeks))
	doubleStart := now.Add(-reportsTrendBucket * time.Duration(2*weeks))

	// A failed analytics read degrades its own region; the page a viewer depends on still renders.
	cells, heatTotal := []heatCell{}, 0
	if rows, err := s.store.ListDispatchProgress(ctx, reportsDispatchLimit(weeks)); err != nil {
		log.Printf("web: reports: list dispatch progress: %v", err)
	} else {
		cells, heatTotal, _, _ = s.foldScanActivity(rows, days)
	}

	openSignals, bySeverity, fired, hasOpenSignals := s.reportsSignalCensus(r)

	openDelta := s.reportsOpenDelta(ctx, fired)

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

	var withdrawalPoints []drift.WithdrawalPoint
	mttw, hasMTTW := "—", false
	var mttwDelta reportsDelta

	// Nothing is "resolved" in this vocabulary: the world withdraws a signal (v1 spec §5.3).
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

	s.render(w, r, "reports", map[string]any{
		"Title": "Reports", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive": "reports",

		"OpenSignals":    openSignals,
		"HasOpenSignals": hasOpenSignals,
		"OpenDelta":      openDelta,
		"OpenSpark":      signalSpark,
		"HasOpenSpark":   hasSignalSpark,

		"DiscoveryCount":    discoveryCount,
		"DiscoveryNames":    discoveryNames,
		"DiscoveryServices": discoverySvcs,
		"HasDiscovery":      hasDiscovery,
		"DiscoveryDelta":    discoveryDelta,
		"DiscoveryBars":     discoveryBars,

		"MTTW":         mttw,
		"HasMTTW":      hasMTTW,
		"MTTWDelta":    mttwDelta,
		"MTTWSpark":    mttwSpark,
		"HasMTTWSpark": hasMTTWSpark,

		"SignalSeries":    signalSeries,
		"HasSignalSeries": hasSignalSeries,

		"BySeverity":  bySeverity,
		"HasSeverity": hasOpenSignals && openSignals > 0,

		"Heat":    cells,
		"HasHeat": heatTotal > 0,

		"RangeWeeks":  weeks,
		"RangeLabel":  window.Label,
		"Periods":     reportsPeriods(),
		"Period":      window.Token,
		"PeriodLabel": window.Label,

		"Schedules": s.reportScheduleRows(ctx),
	})
}

func (s *server) bucketScanActivity(rows []db.ListDispatchProgressRow, days int) (counts []int, window, active int) {
	const day = 24 * time.Hour
	today := s.now().UTC().Truncate(day)

	// A silent day still gets a cell; emission is never restricted to days with activity (#759).
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
		counts[days-1-offset]++
		window++
	}
	return counts, window, active
}

func (s *server) foldScanActivity(rows []db.ListDispatchProgressRow, days int) (cells []heatCell, total, window, active int) {
	counts, window, active := s.bucketScanActivity(rows, days)

	for _, c := range counts {
		total += c
	}

	pct := []int{0, 28, 48, 72, 100}
	levels := drift.HeatLevels(counts)
	cells = make([]heatCell, days)
	for i, c := range counts {
		level := levels[i]
		cell := heatCell{Title: pluralScans(c)}
		if level == 0 {
			// In dark mode a zero cell's fill sits ~5/255 off the ground, so only its border draws it.
			cell.Bg = template.CSS("var(--surface-sunken)")
			cell.Border = template.CSS("var(--row-sep)")
		} else {
			cell.Bg = template.CSS("color-mix(in srgb, var(--chart-1) " + strconv.Itoa(pct[level]) + "%, var(--surface))") // #nosec G203 (constant color-mix CSS; only interpolant is strconv.Itoa of a fixed 0-4 index)
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

func (s *server) reportDeliveryPage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	if s.devMode {
		s.render(w, r, "reportartifact", s.reportartifactFixtureData(acct, r.URL.Query().Get("variant")))
		return
	}

	art, scheduleID, live := s.reportDeliveryArtifact(r.Context())

	heading := art.Title
	if heading == "" {
		heading = "Report delivery"
	}

	var scheduleHole any
	if live {
		scheduleHole = scheduleID
	}

	s.render(w, r, "reportartifact", map[string]any{
		"Title": "Report delivery", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":  "reports",
		"Heading":    heading,
		"Period":     message.ArtifactPeriod(art),
		"ScheduleID": scheduleHole,
		"Doc":        message.BuildArtifactDoc(art),
	})
}

func (s *server) reportDeliveryPDF(w http.ResponseWriter, r *http.Request, acct db.Account) {
	art, _, _ := s.reportDeliveryArtifact(r.Context())

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

func reportDeliveryPDFName(a message.Artifact) string {
	if a.PeriodStart != "" && a.PeriodEnd != "" {
		return "report-" + a.PeriodStart + "-to-" + a.PeriodEnd + ".pdf"
	}
	return "report-delivery.pdf"
}

func (s *server) reportDeliveryArtifact(ctx context.Context) (message.Artifact, int64, bool) {
	schedules, err := s.store.ListReportSchedules(ctx)
	if err != nil {
		log.Printf("web: report delivery: list schedules: %v", err)
		return message.Artifact{}, 0, false
	}
	var (
		best  db.ReportDelivery
		sched db.ReportSchedule
		found bool
	)
	for _, sc := range schedules {
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
		return message.Artifact{}, 0, false
	}
	return s.buildReportDeliveryArtifact(ctx, sched, best), sched.ID, true
}

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
	if del.DeliveredAt.Valid {
		art.Delivered = del.DeliveredAt.Time.UTC().Format(time.RFC3339)
		art.ChannelHost = deliveryTargetHost(sc.DeliveryTarget)
	}
	// The receipt snapshots no content, so the artifact recomputes from its bounds (ADR-0118).
	if del.PeriodStart.Valid && del.PeriodEnd.Valid {
		start, end := del.PeriodStart.Time.UTC(), del.PeriodEnd.Time.UTC()
		art.Signals, art.SeverityCounts = s.reportDeliverySignals(ctx, start, end)
		art.Withdrawn = s.reportDeliveryWithdrawals(ctx, start, end)
	}
	return art
}

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

func deliveryTargetHost(target string) string {
	// An operator's embedded token rides in the target URL (ADR-0114 #1456).
	if u, err := url.Parse(target); err == nil && u.Host != "" {
		return u.Host
	}
	return ""
}
