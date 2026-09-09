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
	CountHeldObservations(ctx context.Context) (int64, error)
	EarliestBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
	GetRetentionSettings(ctx context.Context) (db.GetRetentionSettingsRow, error)
	ListAddressScopeCidrs(ctx context.Context) ([]*netip.Prefix, error)
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
	HasDenominator bool
	PerYear        string
	PerYearBytes   string
	Denominator    string
}

type retentionPanelView struct {
	Observation retentionDialView
	Dispatch    retentionDialView
	Pairs       []retentionPairView
	Clamps      []retentionClampView
	Projection  retentionProjectionView
	Discarded   []retentionClampView
	IsAdmin     bool
	UpdatedAt   string
	UpdatedBy   string
}

// A ladder is a rendering position above a computed floor, never a default (ADR-0038).

var observationLadder = []int64{30, 90, 180, 365, 730}

var dispatchLadder = []int64{4, 8, 13, 26, 52}

const breakClampLimit = 200

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

func scanCadences(rows []db.Scan) []retention.ScanCadence {
	out := make([]retention.ScanCadence, 0, len(rows))
	for _, r := range rows {
		out = append(out, retention.ScanCadence{Kind: r.Kind, CadenceSeconds: r.CadenceSeconds})
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

func buildDial(field, title, unit string, floor retention.FloorExpression, floorUnits int64, ladder []int64, value int64, label func(int64) string) retentionDialView {
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
		view.FloorHint = "No enabled Scan supplies a cadence, so nothing floors this dial."
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

func pairViews(rows []db.ListFacetSourceFloorsRow) []retentionPairView {
	pairs := make([]retention.PairFloor, 0, len(rows))
	for _, r := range rows {
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
		leaf := retention.MovedLeaf(r.Previous, r.Derivation)
		if leaf == "" || seen[leaf] || !r.OpenedAt.Valid {
			continue
		}
		seen[leaf] = true
		out = append(out, retention.Clamp{
			Kind: retention.ClampBreak, Label: "Break", Leaf: leaf, At: r.OpenedAt.Time, Bounded: true,
		})
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

func declaredAddresses(rows []*netip.Prefix) (int64, string) {
	// An undeclared scope is the >99% install, so the projection renders anyway (#47).
	total := new(big.Int)
	for _, p := range rows {
		if p == nil {
			continue
		}
		total.Add(total, seed.AddressCount(*p))
	}
	if !total.IsInt64() || total.Sign() == 0 {
		return 0, total.String()
	}
	return total.Int64(), total.String()
}

func (s *server) retentionPanel(ctx context.Context, isAdmin bool) retentionPanelView {
	view := retentionPanelView{IsAdmin: isAdmin}

	settings, err := s.retentionPanelStore.GetRetentionSettings(ctx)
	if err != nil {
		log.Printf("web: coverage: retention settings: %v", err)
		return view
	}
	if settings.UpdatedAt.Valid {
		view.UpdatedAt = settings.UpdatedAt.Time.UTC().Format("2006-01-02 15:04 MST")
	}

	var scans []retention.ScanCadence
	if rows, serr := s.retentionPanelStore.ListEnabledScans(ctx); serr == nil {
		scans = scanCadences(rows)
	} else {
		log.Printf("web: coverage: enabled scans: %v", serr)
	}
	obsFloor := retention.ObservationFloor(scans)
	dispFloor := retention.DispatchFloor(scans)

	view.Observation = buildDial("observation_currency_days", "Observation currency", "days",
		obsFloor, obsFloor.Days(), observationLadder, settings.ObservationCurrencyDays, humanDays)
	view.Dispatch = buildDial("dispatch_cadence_multiple", "Dispatch retention", "cadences",
		dispFloor, retention.FloorCadences, dispatchLadder, settings.DispatchCadenceMultiple, humanCadences)

	if rows, perr := s.retentionPanelStore.ListFacetSourceFloors(ctx); perr == nil {
		view.Pairs = pairViews(rows)
	} else {
		log.Printf("web: coverage: facet-source floors: %v", perr)
	}

	now := s.now()
	clamps := make([]retention.Clamp, 0, 8)
	if at, berr := s.retentionPanelStore.EarliestBatchTime(ctx); berr == nil && at.Valid {
		clamps = append(clamps, retention.Clamp{
			Kind: retention.ClampBatch, Label: "First batch", At: at.Time, Bounded: true,
		})
	}
	if rows, kerr := s.retentionPanelStore.ListDerivationBreaks(ctx, breakClampLimit); kerr == nil {
		clamps = append(clamps, breakClamps(rows)...)
	} else {
		log.Printf("web: coverage: derivation breaks: %v", kerr)
	}
	obsAt, obsBounded := dialInstant(now, settings.ObservationCurrencyDays*retention.SecondsPerDay)
	clamps = append(clamps, retention.Clamp{
		Kind: retention.ClampRetention, Label: "Observation retention", At: obsAt, Bounded: obsBounded,
	})
	dispAt, dispBounded := retention.Cutoff(now, settings.DispatchCadenceMultiple, dispFloor.CadenceSeconds)
	clamps = append(clamps, retention.Clamp{
		Kind: retention.ClampRetention, Label: "Dispatch retention", At: dispAt, Bounded: dispBounded,
	})
	view.Clamps = clampViews(retention.OrderClamps(clamps), now)

	var held int64
	if n, cerr := s.retentionPanelStore.CountHeldObservations(ctx); cerr == nil {
		held = n
	} else {
		log.Printf("web: coverage: held observations: %v", cerr)
	}
	var declared int64
	var declaredText string
	if rows, aerr := s.retentionPanelStore.ListAddressScopeCidrs(ctx); aerr == nil {
		declared, declaredText = declaredAddresses(rows)
	}
	p := retention.Project(held, declared, rowsPerAddressPerYear(scans))
	view.Projection = retentionProjectionView{
		Held:           humanRows(p.RowsHeld),
		HeldBytes:      humanBytes(p.BytesHeld),
		HasDenominator: p.HasDenominator,
		Denominator:    declaredText,
	}
	if p.HasDenominator {
		view.Projection.PerYear = humanRows(p.RowsPerYear)
		view.Projection.PerYearBytes = humanBytes(p.BytesPerYear)
	}
	return view
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
	var scans []retention.ScanCadence
	if rows, serr := s.retentionPanelStore.ListEnabledScans(ctx); serr == nil {
		scans = scanCadences(rows)
	}

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
	// An unreadable post lands on the terminal stop, which is a position (ADR-0081).
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}
