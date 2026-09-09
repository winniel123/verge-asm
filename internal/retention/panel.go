package retention

import (
	"encoding/json"
	"sort"
	"time"
)

// ScanCadence is one enabled Scan as the two floor derivations read it.
type ScanCadence struct {
	Kind           string
	CadenceSeconds int64
}

// FloorExpression is a floor as ADR-0081 requires it be expressed: a multiple and a
// named Scan, computed live, never a day count.
type FloorExpression struct {
	Multiple       int64
	ScanKind       string
	CadenceSeconds int64
}

func (f FloorExpression) Bounded() bool {
	return f.ScanKind != "" && f.CadenceSeconds > 0 && f.Multiple > 0
}

func (f FloorExpression) Seconds() int64 {
	if !f.Bounded() {
		return 0
	}
	return f.Multiple * f.CadenceSeconds
}

// Rounding up keeps the floor out of the live tier.
func (f FloorExpression) Days() int64 {
	if !f.Bounded() {
		return 0
	}
	return (f.Seconds() + SecondsPerDay - 1) / SecondsPerDay
}

// A tie resolves on Kind so the rendered Scan name does not move between loads.
func pickScan(scans []ScanCadence, tighter func(a, b int64) bool) FloorExpression {
	var best ScanCadence
	for _, s := range scans {
		if s.CadenceSeconds <= 0 {
			continue
		}
		if best.CadenceSeconds == 0 || tighter(s.CadenceSeconds, best.CadenceSeconds) ||
			(s.CadenceSeconds == best.CadenceSeconds && s.Kind < best.Kind) {
			best = s
		}
	}
	if best.CadenceSeconds == 0 {
		return FloorExpression{Multiple: FloorCadences}
	}
	return FloorExpression{Multiple: FloorCadences, ScanKind: best.Kind, CadenceSeconds: best.CadenceSeconds}
}

// Below the tightest bound in force the control changes no row, which is what
// "not the operator's territory" means (ADR-0094).
func ObservationFloor(scans []ScanCadence) FloorExpression {
	return pickScan(scans, func(a, b int64) bool { return a < b })
}

// Below k cadences of the slowest Scan, Coverage cannot answer whether it ran (ADR-0081).
func DispatchFloor(scans []ScanCadence) FloorExpression {
	return pickScan(scans, func(a, b int64) bool { return a > b })
}

// PairFloor is one facet-source pair's own floor and the Scan supplying it — the row
// the enumeration beneath the observation dial carries (ADR-0081, ADR-0094).
type PairFloor struct {
	Facet  string
	Source string
	Floor  FloorExpression
	Rows   int64
}

func SortPairFloors(pairs []PairFloor) {
	sort.SliceStable(pairs, func(i, j int) bool {
		a, b := pairs[i], pairs[j]
		if a.Floor.Bounded() != b.Floor.Bounded() {
			return a.Floor.Bounded()
		}
		if a.Floor.Seconds() != b.Floor.Seconds() {
			return a.Floor.Seconds() < b.Floor.Seconds()
		}
		if a.Facet != b.Facet {
			return a.Facet < b.Facet
		}
		return a.Source < b.Source
	})
}

type ClampKind int

const (
	ClampBatch ClampKind = iota
	ClampBreak
	ClampRetention
)

// Clamp is one horizon in force. A Break names the leaf that moved; a retention
// horizon names nothing, and that asymmetry is what the list makes visible (ADR-0081).
type Clamp struct {
	Kind    ClampKind
	Label   string
	Leaf    string
	At      time.Time
	Bounded bool
}

type ClampRow struct {
	Clamp
	Binding bool
	Inert   bool
}

// OrderClamps renders the invariant "retention may never be the tighter clamp" as a
// sort order rather than as copy: a retention row above the tightest structural clamp
// is drawn inert, and inert rows can never take the ink (ADR-0081).
func OrderClamps(clamps []Clamp) []ClampRow {
	rows := make([]ClampRow, 0, len(clamps))
	for _, c := range clamps {
		rows = append(rows, ClampRow{Clamp: c})
	}
	// Tightest first: the clamp biting at the most recent instant sees least far back.
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.Bounded != b.Bounded {
			return a.Bounded
		}
		if !a.At.Equal(b.At) {
			return a.At.After(b.At)
		}
		return a.Kind < b.Kind
	})

	tightestStructural := -1
	for i, r := range rows {
		if r.Kind != ClampRetention && r.Bounded {
			tightestStructural = i
			break
		}
	}
	for i := range rows {
		if rows[i].Kind != ClampRetention {
			continue
		}
		// An unbounded dial binds at no instant, so it clamps nothing.
		if !rows[i].Bounded || tightestStructural == -1 || i < tightestStructural {
			rows[i].Inert = true
		}
	}
	for i := range rows {
		if !rows[i].Inert && rows[i].Bounded {
			rows[i].Binding = true
			break
		}
	}
	return rows
}

// BytesPerObservation is read off ADR-0081's own ceiling figure — 97,925,120 rows in
// about 13 GB — so the projection states no constant the ADR did not (ADR-0038).
const BytesPerObservation int64 = 142

// Projection prices the position the handle is parked on. Where no address scope is
// declared it has no denominator, so it shows what is held and never a forecast
// (ADR-0081; a forecast from a measured subject count is refused there).
type Projection struct {
	RowsHeld       int64
	BytesHeld      int64
	HasDenominator bool
	RowsPerYear    int64
	BytesPerYear   int64
}

func Project(rowsHeld int64, declaredAddresses int64, rowsPerAddressPerYear int64) Projection {
	p := Projection{RowsHeld: rowsHeld, BytesHeld: rowsHeld * BytesPerObservation}
	if declaredAddresses <= 0 || rowsPerAddressPerYear <= 0 {
		return p
	}
	p.HasDenominator = true
	p.RowsPerYear = declaredAddresses * rowsPerAddressPerYear
	p.BytesPerYear = p.RowsPerYear * BytesPerObservation
	return p
}

// DialStops builds the positions the handle may occupy. The ground below the floor is
// not among them, so a below-floor value is unreachable rather than refused (ADR-0081).
func DialStops(floor int64, ladder []int64) []int64 {
	if floor < 1 {
		floor = 1
	}
	stops := []int64{floor}
	for _, v := range ladder {
		if v > floor {
			stops = append(stops, v)
		}
	}
	return stops
}

// ClampToFloor raises a value the control could not have produced. A stale form or a
// hand-made POST lands on the nearest owned position and is never told off (ADR-0081).
func ClampToFloor(value, floor int64) int64 {
	if value <= 0 || floor <= 0 {
		return value
	}
	if value < floor {
		return floor
	}
	return value
}

// DerivationComponent is one member of a span's flattened derivation vector.
type DerivationComponent struct {
	Leaf    string `json:"leaf"`
	Version string `json:"version"`
}

// MovedLeaf names the leaf a Break moved on. A retention horizon can name nothing, and
// the clamp list draws that asymmetry rather than arguing it in a footnote (ADR-0081).
func MovedLeaf(previous, current []byte) string {
	prev, err := decodeVector(previous)
	if err != nil {
		return ""
	}
	cur, err := decodeVector(current)
	if err != nil {
		return ""
	}
	moved := make([]string, 0, 2)
	for leaf, v := range cur {
		if was, ok := prev[leaf]; !ok || was != v {
			moved = append(moved, leaf)
		}
	}
	for leaf := range prev {
		if _, ok := cur[leaf]; !ok {
			moved = append(moved, leaf)
		}
	}
	if len(moved) == 0 {
		return ""
	}
	sort.Strings(moved)
	return moved[0]
}

func decodeVector(raw []byte) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	var comps []DerivationComponent
	if err := json.Unmarshal(raw, &comps); err != nil {
		return nil, err
	}
	out := make(map[string]string, len(comps))
	for _, c := range comps {
		out[c.Leaf] = c.Version
	}
	return out, nil
}
