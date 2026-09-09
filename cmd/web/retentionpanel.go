package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/retention"
	"github.com/winniel123/verge-asm/internal/seed"
)

// A dial's only justification is the projection, so it is not shown apart from it (ADR-0081).

type retentionPanelStore interface {
	CountHeldObservations(ctx context.Context, exactLimit int64) (db.CountHeldObservationsRow, error)
	EarliestBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
	ListCoveringScanKinds(ctx context.Context) ([]string, error)
	ListDerivationBreaks(ctx context.Context, rowLimit int64) ([]db.ListDerivationBreaksRow, error)
	ListEnabledScans(ctx context.Context) ([]db.Scan, error)
	ListFacetSourceFloors(ctx context.Context) ([]db.ListFacetSourceFloorsRow, error)
	UpdateRetentionSettings(ctx context.Context, arg db.UpdateRetentionSettingsParams) error
}

// Ground is not a value the operator may pick and be corrected for (ADR-0081).

type retentionStopView struct {
	Value    int64
	Label    string
	Selected bool
}

type retentionDialView struct {
	Field       string
	Title       string
	Unit        string
	FloorText   string
	FloorHint   string
	HasFloor    bool
	Stops       []retentionStopView
	Terminal    retentionStopView
	Value       int64
	HorizonText string
}

type retentionPairView struct {
	Facet     string
	Source    string
	FloorText string
	Rows      string
	Uncovered string
	Covered   bool
}

type retentionClampView struct {
	Label   string
	Names   string
	At      string
	ISO     string
	Binding bool
	Inert   bool
	Kind    string
}

type retentionProjectionView struct {
	Held           string
	HeldBytes      string
	HeldEstimated  bool
	HasDenominator bool
	PerYear        string
	PerYearBytes   string
	Denominator    string
	Reason         string
	Withheld       string
}

// A failed read withholds its section and says so, never an empty state (#989).

type retentionPanelView struct {
	Withheld       string
	Observation    retentionDialView
	Dispatch       retentionDialView
	Pairs          []retentionPairView
	PairsWithheld  string
	Clamps         []retentionClampView
	ClampsWithheld string
	Projection     retentionProjectionView
	AnyBounded     bool
	IsAdmin        bool
	UpdatedAt      string
	UpdatedBy      string
}

// A ladder is a rendering position above a computed floor, never a default (ADR-0038).

var observationLadder = []int64{30, 90, 180, 365, 730}

var dispatchLadder = []int64{4, 8, 13, 26, 52}

const breakClampLimit = 200

// Above the cap the count is the corpus scan the panel is bounding, so it stops there (#1768).

const heldCountExactLimit = 100_000

// Three of the panel's reads scan a whole corpus, so one budget bounds the page (#1692).

const retentionPanelBudget = 2 * time.Second

const notResolved = "did not resolve on this load. Nothing is shown rather than a guessed zero."

// A statistic far under the capped count was collected when the table was much smaller (#1783).

const heldCountUnpriceable = "The projection is withheld. This corpus is too large to count on a page load, and the table's live-row statistic sits far below the rows already counted, so no figure here would estimate the corpus. A vacuum or an analyse refreshes that statistic."

// An unfloored dial names the Scan set it came up empty in, and the two differ (ADR-0081).

const (
	unflooredObservation = "No Scan bounds a row yet, so nothing floors this dial."
	unflooredDispatch    = "No enabled Scan supplies a cadence, so nothing floors this dial."
)

func floorText(f retention.FloorExpression) string {
	// A multiple and a named Scan cannot go stale the way a day count already has (ADR-0038).
	if !f.Bounded() {
		return "no enabled Scan supplies a cadence, so no floor is in force"
	}
	if label := cadenceLabel(f.CadenceSeconds); label != "" {
		return fmt.Sprintf("%d × cadence(%s), %s", f.Multiple, f.ScanKind, label)
	}
	return fmt.Sprintf("%d × cadence(%s)", f.Multiple, f.ScanKind)
}

func scanCadences(rows []db.Scan, coveringKinds []string) []retention.ScanCadence {
	covering := make(map[string]bool, len(coveringKinds))
	for _, k := range coveringKinds {
		covering[k] = true
	}
	out := make([]retention.ScanCadence, 0, len(rows))
	for _, r := range rows {
		out = append(out, retention.ScanCadence{
			Kind: r.Kind, CadenceSeconds: r.CadenceSeconds, Covers: covering[r.Kind],
		})
	}
	return out
}

func humanDays(days int64) string {
	switch {
	case days <= 0:
		return "keep everything"
	case days == 1:
		return "1 day"
	case days%365 == 0 && days >= 365:
		y := days / 365
		if y == 1 {
			return "1 year"
		}
		return fmt.Sprintf("%d years", y)
	case days%30 == 0 && days >= 60:
		return fmt.Sprintf("%d months", days/30)
	default:
		return fmt.Sprintf("%d days", days)
	}
}

func humanCadences(n int64) string {
	if n <= 0 {
		return "keep everything"
	}
	if n == 1 {
		return "1 cadence"
	}
	return fmt.Sprintf("%d cadences", n)
}

func buildDial(field, title, unit string, floor retention.FloorExpression, floorUnits int64, ladder []int64, value int64, label func(int64) string, unfloored string) retentionDialView {
	stops := retention.DialStops(floorUnits, ladder)
	// The handle parks on a labelled stop, so the control is never valueless (ADR-0081).
	view := retentionDialView{
		Field:     field,
		Title:     title,
		Unit:      unit,
		FloorText: floorText(floor),
		HasFloor:  floor.Bounded(),
		Value:     value,
		Terminal:  retentionStopView{Value: 0, Label: "keep everything", Selected: value <= 0},
	}
	if floor.Bounded() {
		view.FloorHint = fmt.Sprintf("Below this the dial changes no row. Move it by changing %s's cadence, not by arguing with this control.", floor.ScanKind)
	} else {
		view.FloorHint = unfloored
	}
	for _, s := range stops {
		view.Stops = append(view.Stops, retentionStopView{Value: s, Label: label(s), Selected: value == s})
	}
	switch {
	case value <= 0:
		view.HorizonText = "The handle is parked on keep everything. No number ships because none is derivable — ninety days is invented and a year is invented."
	default:
		view.HorizonText = fmt.Sprintf("The dial buys %s. A row whose own bound outlives that is kept anyway; the floor is what the product will not sell below.", label(value))
	}
	return view
}

type pairKey struct{ facet, source string }

func pairViews(rows []db.ListFacetSourceFloorsRow) []retentionPairView {
	pairs := make([]retention.PairFloor, 0, len(rows))
	uncovered := make(map[pairKey]int64, len(rows))
	for _, r := range rows {
		uncovered[pairKey{r.Facet, r.Source}] = r.UncoveredRows
		pairs = append(pairs, retention.PairFloor{
			Facet:  r.Facet,
			Source: r.Source,
			Rows:   r.RowsHeld,
			Floor: retention.FloorExpression{
				Multiple: retention.FloorCadences, ScanKind: r.ScanKind, CadenceSeconds: r.TightestCadence,
			},
		})
	}
	retention.SortPairFloors(pairs)
	out := make([]retentionPairView, 0, len(pairs))
	for _, p := range pairs {
		v := retentionPairView{
			Facet: p.Facet, Source: p.Source, Rows: humanRows(p.Rows), Covered: p.Floor.Bounded(),
		}
		if p.Floor.Bounded() {
			v.FloorText = floorText(p.Floor)
		} else {
			// An undefined bound is not an expired one, so the row is never retired (ADR-0094).
			v.FloorText = "no covering Scan — the bound is undefined, so nothing here is retired"
		}
		if n := uncovered[pairKey{p.Facet, p.Source}]; n > 0 && p.Floor.Bounded() {
			v.Uncovered = fmt.Sprintf("%s of these have no covering Scan and are never retired", humanRows(n))
		}
		out = append(out, v)
	}
	return out
}

func humanRows(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	lead := len(s) % 3
	if lead > 0 {
		b.WriteString(s[:lead])
	}
	for i := lead; i < len(s); i += 3 {
		if b.Len() > 0 {
			b.WriteByte(',')
		}
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func breakClamps(rows []db.ListDerivationBreaksRow) []retention.Clamp {
	// One row per leaf: the most recent instant that leaf's version moved.
	seen := make(map[string]bool, len(rows))
	out := make([]retention.Clamp, 0, len(rows))
	for _, r := range rows {
		if !r.OpenedAt.Valid {
			continue
		}
		for _, leaf := range retention.MovedLeaves(r.Previous, r.Derivation) {
			if seen[leaf] {
				continue
			}
			seen[leaf] = true
			out = append(out, retention.Clamp{
				Kind: retention.ClampBreak, Label: "Break", Leaf: leaf, At: r.OpenedAt.Time, Bounded: true,
			})
		}
	}
	return out
}

func clampViews(rows []retention.ClampRow, now time.Time) []retentionClampView {
	out := make([]retentionClampView, 0, len(rows))
	for _, r := range rows {
		v := retentionClampView{
			Label: r.Label, Names: r.Leaf, Binding: r.Binding, Inert: r.Inert,
		}
		switch r.Kind {
		case retention.ClampBatch:
			v.Kind = "batch"
		case retention.ClampBreak:
			v.Kind = "break"
		default:
			v.Kind = "retention"
		}
		if r.Bounded {
			v.At = relativeAge(now.Sub(r.At))
			v.ISO = r.At.UTC().Format(time.RFC3339)
		} else {
			v.At = "binds at no instant"
		}
		out = append(out, v)
	}
	return out
}

func relativeAge(d time.Duration) string {
	days := int64(d.Hours() / 24)
	switch {
	case days <= 0:
		return "today"
	case days == 1:
		return "1 day ago"
	case days < 60:
		return fmt.Sprintf("%d days ago", days)
	case days < 730:
		return fmt.Sprintf("%d months ago", days/30)
	default:
		return fmt.Sprintf("%d years ago", days/365)
	}
}

func declaredAddresses(rows []*netip.Prefix) (count int64, text string, tooLarge bool) {
	// An undeclared scope is the >99% install, so the projection renders anyway (#47).
	total := new(big.Int)
	for _, p := range rows {
		if p == nil {
			continue
		}
		total.Add(total, seed.AddressCount(*p))
	}
	if total.Sign() == 0 {
		return 0, "", false
	}
	if !total.IsInt64() {
		return 0, total.String(), true
	}
	return total.Int64(), total.String(), false
}

func projectionReason(hasDenominator, declared, tooLarge bool) string {
	switch {
	case hasDenominator:
		return ""
	case !declared:
		return "This install declares no address scope, so this projection has no denominator: it shows what is held and never a forecast of what you have."
	default:
		return "Your declared address scope is too large to count here, so this projection has no denominator: it shows what is held and never a forecast."
	}
}

func (s *server) retentionPanel(ctx context.Context, isAdmin bool) retentionPanelView {
	view := retentionPanelView{IsAdmin: isAdmin}
	ctx, cancel := context.WithTimeout(ctx, retentionPanelBudget)
	defer cancel()

	settings, err := s.retentionPanelStore.GetRetentionSettings(ctx)
	if err != nil {
		log.Printf("web: coverage: retention settings: %v", err)
		view.Withheld = "The retention dials " + notResolved
		return view
	}
	if settings.UpdatedAt.Valid {
		view.UpdatedAt = settings.UpdatedAt.Time.UTC().Format("2006-01-02 15:04 MST")
	}
	scanRows, err := s.retentionPanelStore.ListEnabledScans(ctx)
	if err != nil {
		// A dial drawn with no floor would offer the ground as a stop (ADR-0081).
		log.Printf("web: coverage: enabled scans: %v", err)
		view.Withheld = "The enabled Scan set, which every floor derives from, " + notResolved
		return view
	}
	coveringKinds, err := s.retentionPanelStore.ListCoveringScanKinds(ctx)
	if err != nil {
		// An unread cover would floor the dial on a Scan bounding nothing (ADR-0081).
		log.Printf("web: coverage: covering scans: %v", err)
		view.Withheld = "The covering Scan set, which the observation floor derives from, " + notResolved
		return view
	}
	scans := scanCadences(scanRows, coveringKinds)
	obsFloor := retention.ObservationFloor(scans)
	dispFloor := retention.DispatchFloor(scans)

	view.Observation = buildDial("observation_currency_days", "Observation currency", "days",
		obsFloor, obsFloor.Days(), observationLadder, settings.ObservationCurrencyDays, humanDays,
		unflooredObservation)
	view.Dispatch = buildDial("dispatch_cadence_multiple", "Dispatch retention", "cadences",
		dispFloor, retention.FloorCadences, dispatchLadder, settings.DispatchCadenceMultiple, humanCadences,
		unflooredDispatch)
	view.AnyBounded = settings.ObservationCurrencyDays > 0 || settings.DispatchCadenceMultiple > 0

	if rows, perr := s.retentionPanelStore.ListFacetSourceFloors(ctx); perr == nil {
		view.Pairs = pairViews(rows)
	} else {
		log.Printf("web: coverage: facet-source floors: %v", perr)
		view.PairsWithheld = "The facet-source floors " + notResolved
	}

	now := s.now()
	clamps := make([]retention.Clamp, 0, 8)
	var clampsWithheld bool
	if at, berr := s.retentionPanelStore.EarliestBatchTime(ctx); berr == nil {
		if at.Valid {
			clamps = append(clamps, retention.Clamp{
				Kind: retention.ClampBatch, Label: "First batch", At: at.Time, Bounded: true,
			})
		}
	} else {
		log.Printf("web: coverage: earliest batch: %v", berr)
		clampsWithheld = true
	}
	if rows, kerr := s.retentionPanelStore.ListDerivationBreaks(ctx, breakClampLimit); kerr == nil {
		clamps = append(clamps, breakClamps(rows)...)
	} else {
		log.Printf("web: coverage: derivation breaks: %v", kerr)
		clampsWithheld = true
	}
	if clampsWithheld {
		// A list missing a Break would hand the ink to the wrong row, so none is shown.
		view.ClampsWithheld = "The clamp list " + notResolved
	} else {
		obsAt, obsBounded := dialInstant(now, settings.ObservationCurrencyDays*retention.SecondsPerDay)
		clamps = append(clamps, retention.Clamp{
			Kind: retention.ClampRetention, Label: "Observation retention", At: obsAt, Bounded: obsBounded,
		})
		dispAt, dispBounded := retention.Cutoff(now, settings.DispatchCadenceMultiple, dispFloor.CadenceSeconds)
		clamps = append(clamps, retention.Clamp{
			Kind: retention.ClampRetention, Label: "Dispatch retention", At: dispAt, Bounded: dispBounded,
		})
		view.Clamps = clampViews(retention.OrderClamps(clamps), now)
	}

	counts, cerr := s.retentionPanelStore.CountHeldObservations(ctx, heldCountExactLimit)
	if cerr != nil {
		log.Printf("web: coverage: held observations: %v", cerr)
		view.Projection.Withheld = "The projection " + notResolved
		return view
	}
	addrRows, aerr := s.retentionPanelStore.ListAddressScopeCidrs(ctx)
	if aerr != nil {
		log.Printf("web: coverage: address scope: %v", aerr)
		view.Projection.Withheld = "The declared address scope, which the projection prices, " + notResolved
		return view
	}
	declared, declaredText, tooLarge := declaredAddresses(addrRows)
	held, estimated, priced := heldRows(counts, heldCountExactLimit)
	if !priced {
		view.Projection.Withheld = heldCountUnpriceable
		return view
	}
	p := retention.Project(held, declared, rowsPerAddressPerYear(scans))
	overflowed := declared > 0 && !p.HasDenominator
	view.Projection = retentionProjectionView{
		Held:           humanRows(p.RowsHeld),
		HeldBytes:      humanBytes(p.BytesHeld),
		HeldEstimated:  estimated,
		HasDenominator: p.HasDenominator,
		Denominator:    declaredText,
		Reason:         projectionReason(p.HasDenominator, declaredText != "", tooLarge || overflowed),
	}
	if p.HasDenominator {
		view.Projection.PerYear = humanRows(p.RowsPerYear)
		view.Projection.PerYearBytes = humanBytes(p.BytesPerYear)
	}
	return view
}

func heldRows(row db.CountHeldObservationsRow, exactLimit int64) (rows int64, estimated, priced bool) {
	if row.CountedRows <= exactLimit {
		return row.CountedRows, false, true
	}
	if row.EstimatedRows < exactLimit*4/5 {
		// Autovacuum leaves reltuples short by at most a fifth, so a wider gap is stale (#1783).
		return 0, false, false
	}
	if row.EstimatedRows < row.CountedRows {
		return row.CountedRows, true, true
	}
	return row.EstimatedRows, true, true
}

func dialInstant(now time.Time, dialSeconds int64) (time.Time, bool) {
	if dialSeconds <= 0 {
		return time.Time{}, false
	}
	return now.Add(-time.Duration(dialSeconds) * time.Second), true
}

func rowsPerAddressPerYear(scans []retention.ScanCadence) int64 {
	const secondsPerYear = 365 * retention.SecondsPerDay
	var total int64
	// Arithmetic over the enabled cadences, never a forecast about the estate (ADR-0081).
	for _, s := range scans {
		if s.CadenceSeconds > 0 {
			total += secondsPerYear / s.CadenceSeconds
		}
	}
	return total
}

func (s *server) updateCoverageRetention(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()
	settings, err := s.retentionPanelStore.GetRetentionSettings(ctx)
	if err != nil {
		s.serverError(w, "retention settings", err)
		return
	}
	scanRows, err := s.retentionPanelStore.ListEnabledScans(ctx)
	if err != nil {
		// With no floor to raise to, a write could persist the ground (ADR-0081).
		s.serverError(w, "enabled scans", err)
		return
	}
	coveringKinds, err := s.retentionPanelStore.ListCoveringScanKinds(ctx)
	if err != nil {
		// A write clamps to the covering floor, so a missed cover raises the wrong one.
		s.serverError(w, "covering scans", err)
		return
	}
	scans := scanCadences(scanRows, coveringKinds)

	// Below the floor is not the operator's territory, so nothing is rejected (ADR-0081).
	obs := retention.ClampToFloor(
		parseDialValue(r.FormValue("observation_currency_days"), settings.ObservationCurrencyDays),
		retention.ObservationFloor(scans).Days())
	disp := retention.ClampToFloor(
		parseDialValue(r.FormValue("dispatch_cadence_multiple"), settings.DispatchCadenceMultiple),
		retention.FloorCadences)

	if err := s.retentionPanelStore.UpdateRetentionSettings(ctx, db.UpdateRetentionSettingsParams{
		ObservationCurrencyDays: obs,
		DispatchCadenceMultiple: disp,
		TranscriptCurrencyDays:  settings.TranscriptCurrencyDays,
		UpdatedBy:               pgtype.Int8{Int64: acct.ID, Valid: true},
	}); err != nil {
		s.serverError(w, "update retention", err)
		return
	}
	s.redirectBack(w, r, "/coverage")
}

func parseDialValue(raw string, fallback int64) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	// Zero is the terminal stop; anything else the track cannot express leaves the dial alone.
	if err != nil || v < 0 {
		return fallback
	}
	return v
}
