package main

import (
	"context"
	"encoding/json"
	"log"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/signal"
)

type deltasStore interface {
	ListServiceReachabilitySpansByClass(ctx context.Context) ([]db.ListServiceReachabilitySpansByClassRow, error)
	ListServiceReachabilitySpansByClassAt(ctx context.Context, at pgtype.Timestamptz) ([]db.ListServiceReachabilitySpansByClassAtRow, error)
	ListSignalInstances(ctx context.Context) ([]db.SignalInstance, error)
	ListSpansOpenSince(ctx context.Context, since pgtype.Timestamptz) ([]db.ListSpansOpenSinceRow, error)
	PreviousBatchTime(ctx context.Context) (pgtype.Timestamptz, error)
}

func pgtypeTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

type statDeltas struct {
	AssetsWatched drift.Delta
	Exposed       drift.Delta
	Firewalled    drift.Delta
	OneLegged     drift.Delta
	CertsExpiring drift.Delta
	OpenSignals   drift.Delta
	Critical      drift.Delta
	ExposureKnown bool // A later leg's failure may not blank the Exposed tile's figure (#2046).
	Known         bool
}

type firedSignal struct {
	Rule    string
	Subject string
}

func (s *server) previousBatchInstant(ctx context.Context) (at time.Time, ok bool, err error) {
	ts, err := s.deltasStore.PreviousBatchTime(ctx)
	if err != nil {
		return time.Time{}, false, err
	}
	if !ts.Valid {
		return time.Time{}, false, nil
	}
	return ts.Time, true, nil
}

func spanFromOpenSinceRow(row db.ListSpansOpenSinceRow) drift.Span {
	// A delta counts a population at an instant and compares no vectors, so Derivation is dropped.
	sp := drift.Span{
		Key: drift.TimelineKey{
			SubjectKind:   row.SubjectKind,
			SubjectKey:    row.SubjectKey,
			Facet:         row.Facet,
			Discriminator: row.Discriminator,
			Source:        row.Source,
		},
		Value:    string(row.Value),
		IsGap:    row.IsGap,
		OpenedAt: row.OpenedAt.Time,
	}
	if row.ClosedAt.Valid {
		sp.ClosedAt = row.ClosedAt.Time
	}
	return sp
}

func (s *server) dashboardDeltas(ctx context.Context, fired []firedSignal) statDeltas {
	prevAt, hasPrev, err := s.previousBatchInstant(ctx)
	if err != nil {
		// The tile's figure needs no previous batch, so this failure may not blank it (#2046).
		log.Printf("web: dashboard: previous batch instant: %v", err)
		hasPrev = false
	}

	// The Exposed tile reads its figure from this Current, so no second binding is taken (#2046).
	exposed, firewalled, oneLegged, eok := s.exposureCountDeltas(ctx, prevAt)
	if !eok {
		return statDeltas{}
	}
	if !hasPrev {
		// No instant to compare against, so the figure renders and no change does.
		return statDeltas{Exposed: drift.Delta{Current: exposed.Current}, ExposureKnown: true}
	}
	exposureOnly := statDeltas{
		Exposed: exposed, Firewalled: firewalled, OneLegged: oneLegged, ExposureKnown: true,
	}

	out := exposureOnly
	out.Known = true

	if rows, serr := s.deltasStore.ListSpansOpenSince(ctx, pgtypeTimestamptz(prevAt)); serr == nil {
		all := make([]drift.Span, 0, len(rows))
		assets := make([]drift.Span, 0, len(rows))
		for _, r := range rows {
			sp := spanFromOpenSinceRow(r)
			all = append(all, sp)
			if r.SubjectKind == "name" || r.SubjectKind == "service" {
				assets = append(assets, sp)
			}
		}
		out.AssetsWatched = drift.CountDelta(assets, prevAt, drift.DistinctSubjects)
		out.CertsExpiring = drift.Delta{
			Current:  countCertsExpiring(drift.CurrentlyOpen(all), s.now().UTC()),
			Previous: countCertsExpiring(drift.OpenAt(all, prevAt), prevAt),
		}
	} else {
		log.Printf("web: dashboard: list spans open since: %v", serr)
		return exposureOnly
	}

	open, critical, serr := s.signalDeltas(ctx, fired, prevAt)
	if serr != nil {
		log.Printf("web: dashboard: signal deltas: %v", serr)
		return exposureOnly
	}
	out.OpenSignals, out.Critical = open, critical

	return out
}

func countCertsExpiring(open []drift.Span, ref time.Time) int {
	// Only a v3 leaf carries both dates, so an older span is skipped, not guessed (#464).
	n := 0
	for _, sp := range open {
		if sp.Key.Facet != connectoutcome.FacetCertificate || sp.IsGap {
			continue
		}
		var v struct {
			NotBefore string `json:"not_before"`
			NotAfter  string `json:"not_after"`
		}
		if err := json.Unmarshal([]byte(sp.Value), &v); err != nil {
			continue
		}
		nb, nbErr := time.Parse(time.RFC3339, v.NotBefore)
		na, naErr := time.Parse(time.RFC3339, v.NotAfter)
		if nbErr != nil || naErr != nil {
			continue
		}
		horizon, ok := signal.CertHorizon(nb, na)
		if ok && na.After(ref) && !na.After(ref.Add(horizon)) {
			n++
		}
	}
	return n
}

func (s *server) signalDeltas(ctx context.Context, fired []firedSignal, prevAt time.Time) (open, critical drift.Delta, err error) {
	// A firing that ended is never stored, so the previous count reads as net-new-since-last-batch.
	rows, err := s.deltasStore.ListSignalInstances(ctx)
	if err != nil {
		return drift.Delta{}, drift.Delta{}, err
	}
	type pair struct{ rule, subject string }
	firstSeen := make(map[pair]time.Time, len(rows))
	for _, row := range rows {
		if row.FirstSeen.Valid {
			firstSeen[pair{row.SignalName, row.SubjectKey}] = row.FirstSeen.Time
		}
	}

	curOpen, curCrit, prevOpen, prevCrit := 0, 0, 0, 0
	for _, f := range fired {
		isCrit := severityIsCritical(f.Rule)
		curOpen++
		if isCrit {
			curCrit++
		}
		fs, ok := firstSeen[pair{f.Rule, f.Subject}]
		if ok && fs.Before(prevAt) {
			prevOpen++
			if isCrit {
				prevCrit++
			}
		}
	}
	return drift.Delta{Current: curOpen, Previous: prevOpen},
		drift.Delta{Current: curCrit, Previous: prevCrit}, nil
}

func severityIsCritical(rule string) bool {
	sev, _ := signal.SeverityFor(rule)
	return sev == signal.SevCritical
}

type exposureLegs struct {
	covered func(netip.Addr) bool
	cur     []reachLegRow
	prev    []reachLegRow
}

func (s *server) readExposureLegs(ctx context.Context, prevAt time.Time) (exposureLegs, bool) {
	current, err := s.deltasStore.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		log.Printf("web: exposure delta: list reachability by class: %v", err)
		return exposureLegs{}, false
	}
	var past []db.ListServiceReachabilitySpansByClassAtRow
	// A zero instant names no previous batch, and the snapshot behind one is empty (#2046).
	if !prevAt.IsZero() {
		past, err = s.deltasStore.ListServiceReachabilitySpansByClassAt(ctx, pgtypeTimestamptz(prevAt))
		if err != nil {
			log.Printf("web: exposure delta: list reachability by class at: %v", err)
			return exposureLegs{}, false
		}
	}
	// One binding holds both snapshots, so a scope edit moves both legs (ADR-1895 §4, ADR-1945 §2).
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		log.Printf("web: exposure delta: address scope coverage: %v", err)
		return exposureLegs{}, false
	}
	return exposureLegs{
		covered: covered,
		cur:     reachRowsFromCurrent(current),
		prev:    reachRowsFromAt(past),
	}, true
}

func (s *server) exposureCountDeltas(ctx context.Context, prevAt time.Time) (exposed, firewalled, oneLegged drift.Delta, ok bool) {
	legs, ok := s.readExposureLegs(ctx, prevAt)
	if !ok {
		return drift.Delta{}, drift.Delta{}, drift.Delta{}, false
	}

	cur := censusFromLegs(collapseReachLegs(legs.cur, legs.covered))
	prev := censusFromLegs(collapseReachLegs(legs.prev, legs.covered))
	return drift.Delta{Current: cur.Exposed, Previous: prev.Exposed},
		drift.Delta{Current: cur.Firewalled, Previous: prev.Firewalled},
		drift.Delta{Current: cur.OneLegged, Previous: prev.OneLegged},
		true
}

func (s *server) currentCertsExpiring(ctx context.Context) (int, bool) {
	// An invalid @since selects every span still open, which is the current-state read this needs.
	rows, err := s.deltasStore.ListSpansOpenSince(ctx, pgtype.Timestamptz{})
	if err != nil {
		log.Printf("web: dashboard: certs-expiring count: %v", err)
		return 0, false
	}
	open := make([]drift.Span, 0, len(rows))
	for _, row := range rows {
		open = append(open, spanFromOpenSinceRow(row))
	}
	return countCertsExpiring(drift.CurrentlyOpen(open), s.now().UTC()), true
}
