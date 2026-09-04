package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/exposure"
	"github.com/winniel123/verge-asm/internal/measure/connectoutcome"
	"github.com/winniel123/verge-asm/internal/signal"
)

const certExpiryWindow = 30 * 24 * time.Hour

func pgtypeTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

type statDeltas struct {
	AssetsWatched drift.Delta
	Exposed       drift.Delta
	Firewalled    drift.Delta
	NotReached    drift.Delta
	CertsExpiring drift.Delta
	OpenSignals   drift.Delta
	Critical      drift.Delta
	Known         bool
}

type firedSignal struct {
	Rule    string
	Subject string
}

func (s *server) previousBatchInstant(ctx context.Context) (at time.Time, ok bool, err error) {
	ts, err := s.store.PreviousBatchTime(ctx)
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
	prevAt, ok, err := s.previousBatchInstant(ctx)
	if err != nil {
		log.Printf("web: dashboard: previous batch instant: %v", err)
		return statDeltas{}
	}
	if !ok {
		return statDeltas{}
	}

	out := statDeltas{Known: true}

	if rows, serr := s.store.ListSpansOpenSince(ctx, pgtypeTimestamptz(prevAt)); serr == nil {
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
		return statDeltas{}
	}

	if exposed, firewalled, notReached, eok := s.exposureCountDeltas(ctx, prevAt); eok {
		out.Exposed, out.Firewalled, out.NotReached = exposed, firewalled, notReached
	} else {
		return statDeltas{}
	}

	open, critical, serr := s.signalDeltas(ctx, fired, prevAt)
	if serr != nil {
		log.Printf("web: dashboard: signal deltas: %v", serr)
		return statDeltas{}
	}
	out.OpenSignals, out.Critical = open, critical

	return out
}

func countCertsExpiring(open []drift.Span, ref time.Time) int {
	// Only a v2 leaf carries not_after, so an older span is skipped rather than guessed at (#464).
	horizon := ref.Add(certExpiryWindow)
	n := 0
	for _, sp := range open {
		if sp.Key.Facet != connectoutcome.FacetCertificate || sp.IsGap {
			continue
		}
		var v struct {
			NotAfter string `json:"not_after"`
		}
		if err := json.Unmarshal([]byte(sp.Value), &v); err != nil || v.NotAfter == "" {
			continue
		}
		na, err := time.Parse(time.RFC3339, v.NotAfter)
		if err != nil {
			continue
		}
		if na.After(ref) && !na.After(horizon) {
			n++
		}
	}
	return n
}

func (s *server) signalDeltas(ctx context.Context, fired []firedSignal, prevAt time.Time) (open, critical drift.Delta, err error) {
	// A firing that ended is never stored, so the previous count reads as net-new-since-last-batch.
	rows, err := s.store.ListSignalInstances(ctx)
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

func (s *server) exposureCountDeltas(ctx context.Context, prevAt time.Time) (exposed, firewalled, notReached drift.Delta, ok bool) {
	current, err := s.store.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		log.Printf("web: exposure delta: list reachability by class: %v", err)
		return drift.Delta{}, drift.Delta{}, drift.Delta{}, false
	}
	past, err := s.store.ListServiceReachabilitySpansByClassAt(ctx, pgtypeTimestamptz(prevAt))
	if err != nil {
		log.Printf("web: exposure delta: list reachability by class at: %v", err)
		return drift.Delta{}, drift.Delta{}, drift.Delta{}, false
	}
	// One covered binding for both snapshots, so they classify alike (CONTEXT.md Vantage class).
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		log.Printf("web: exposure delta: address scope coverage: %v", err)
		return drift.Delta{}, drift.Delta{}, drift.Delta{}, false
	}

	cur := projectStatsFromLegs(collapseReachLegs(reachRowsFromCurrent(current), covered))
	prev := projectStatsFromLegs(collapseReachLegs(reachRowsFromAt(past), covered))
	return drift.Delta{Current: cur.exposed, Previous: prev.exposed},
		drift.Delta{Current: cur.firewalled, Previous: prev.firewalled},
		drift.Delta{Current: cur.notReached, Previous: prev.notReached},
		true
}

func projectStatsFromLegs(byService map[string]map[string]legInfo) exposureStats {
	var stats exposureStats
	for _, m := range byService {
		ev, ok := exposure.Project(legFrom(m["internet"]), legFrom(m["internal"]))
		switch {
		case !ok:
			stats.notReached++
		case ev == exposure.Exposed:
			stats.exposed++
		case ev == exposure.Firewalled:
			stats.firewalled++
		}
	}
	return stats
}

func (s *server) currentExposedCount(ctx context.Context) (int, bool) {
	// A withheld delta carries no Current, so the cell derives the value the same way.
	rows, err := s.store.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		log.Printf("web: dashboard: exposed services count: %v", err)
		return 0, false
	}
	covered, err := s.addressScopeCovered(ctx)
	if err != nil {
		log.Printf("web: dashboard: address scope coverage: %v", err)
		return 0, false
	}
	return projectStatsFromLegs(collapseReachLegs(reachRowsFromCurrent(rows), covered)).exposed, true
}

func (s *server) currentCertsExpiring(ctx context.Context) (int, bool) {
	// An invalid @since selects every span still open, which is the current-state read this needs.
	rows, err := s.store.ListSpansOpenSince(ctx, pgtype.Timestamptz{})
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
