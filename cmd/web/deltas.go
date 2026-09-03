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

// Vs-last-batch stat deltas (P0.2, design-system PARITY-CHART.md; ADR-0116). Every
// stat tile on the Dashboard (P2.1) and Exposure (P2.6) renders a signed delta
// against the previous batch — the design is normative for functionality, so the
// datum is built here rather than the tile dropping the number (ADR-0116). This
// file is the derivation half: it reads the previous batch boundary and the span
// corpus, reconstructs each metric's value as it stood a batch ago from the
// never-compacted span timeline (internal/drift.OpenAt, ADR-0041), and exposes the
// Current/Previous pair. It paints no markup — the P2 screen tickets read these.
//
// A delta is a design DATUM, never a design decision: the number is real and
// computed, so no SPEC-CHANGE collision arises. Where there is no previous batch to
// compare against (a first-boot estate with at most one batch), the delta is
// withheld (Known=false) rather than compared against nothing — an honest absence,
// drawn by the tile's own no-delta state, not a fabricated +0.

// certExpiryWindow is the horizon a certificate is "expiring" within — 30 days, the
// window the Exposure/Dashboard "certs expiring" tile counts (PARITY-CHART.md P0.2;
// the `certificate-expiring` rule's spirit, internal/signal/endpoint.go). #445 builds
// the current-state count in handlers.go/probers.go; this delta computes its own
// window-count locally so the two dedupe trivially at merge rather than conflict.
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

// firedSignal is one currently-firing (rule, subject) pair, handed to the signal
// deltas from the census the page already evaluated so the corpus is folded once.
type firedSignal struct {
	Rule    string
	Subject string
}

// previousBatchInstant reads the boundary a vs-last-batch delta reconstructs the
// previous value at (PreviousBatchTime). ok is false where fewer than two distinct
// batches have committed — there is no previous batch, so no delta is derivable.
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

// spanFromOpenSinceRow lifts a persisted span row into the drift.Span the delta
// primitives read. Only the timeline key, the value, and the open/close instants
// are needed — the delta counts populations open at an instant, never compares
// vectors — so the Derivation vector is left unparsed.
func spanFromOpenSinceRow(row db.ListSpansOpenSinceRow) drift.Span {
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

// dashboardDeltas assembles the Dashboard's stat-tile deltas: assets watched,
// exposed services, certs expiring, open signals and critical. Best-effort like
// every dashboard read — a corpus failure logs and returns Known=false so the tiles
// degrade to their no-delta state rather than 500ing the landing page. The fired
// census is passed in (the caller already evaluated it) so the corpus is folded once.
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

	// Span-derived deltas: assets watched (distinct name/service subjects with an open
	// span) and certs expiring (endpoint certificate spans whose leaf expires within
	// the window). Both read one span corpus scoped to recent drift.
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

	// Exposure deltas: the projected 2x2 counts now vs at the previous batch.
	if exposed, firewalled, notReached, eok := s.exposureCountDeltas(ctx, prevAt); eok {
		out.Exposed, out.Firewalled, out.NotReached = exposed, firewalled, notReached
	} else {
		return statDeltas{}
	}

	// Signal deltas: open signals and critical, current census vs the count already
	// firing at the previous batch (read from the persisted first-seen identities).
	open, critical, serr := s.signalDeltas(ctx, fired, prevAt)
	if serr != nil {
		log.Printf("web: dashboard: signal deltas: %v", serr)
		return statDeltas{}
	}
	out.OpenSignals, out.Critical = open, critical

	return out
}

// countCertsExpiring counts the endpoint certificate spans whose leaf expires within
// certExpiryWindow of the reference instant — not already expired at `ref`, and
// expiring on or before ref+window. The expiry instant is read from the `not_after`
// key the presented certificate value now carries: the v2 leaf stores the parsed
// leaf's NotAfter (internal/measure/connectoutcome/tls.go, P0.4b #464, SPEC-CHANGE.md
// collision #8). A value with no `not_after` — a negative outcome, or a pre-v2 span —
// simply does not contribute; the count is a real read, never a fabricated instant.
func countCertsExpiring(open []drift.Span, ref time.Time) int {
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

// signalDeltas builds the open-signals and critical deltas. The current values are
// the live census the caller passed; the previous values are how many of those pairs
// were ALREADY firing at the previous batch, read from the persisted signal_instance
// first-seen (P0.1, #442) — a pair first seen before the boundary was open a batch
// ago, one first seen at or after it (or never minted) is new this batch. Firings
// that ended aren't stored, so this reads as net-new-since-last-batch, the honest
// delta the stored history supports; the number is real, never fabricated.
func (s *server) signalDeltas(ctx context.Context, fired []firedSignal, prevAt time.Time) (open, critical drift.Delta, err error) {
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

// severityIsCritical reports whether a rule is rated critical — the ramp read the
// "critical" tile counts (internal/signal.SeverityFor). An unknown rule is not
// critical (SeverityFor folds it to info), so a stale name never inflates the count.
func severityIsCritical(rule string) bool {
	sev, _ := signal.SeverityFor(rule)
	return sev == signal.SevCritical
}

// exposureCountDeltas computes the Exposure stat band's three vs-last-batch deltas —
// exposed, firewalled, not-reached — by projecting the reachability legs now and as
// they stood at the previous batch (ListServiceReachabilitySpansByClass and its
// as-of-@at twin) through the same pure exposure engine (internal/exposure.Project,
// ADR-0017). ok is false where a read fails or no previous batch exists, so the
// caller withholds the deltas rather than showing a half-computed one.
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
	// Class is DERIVED per read for both snapshots (#709), against one shared covered
	// binding, so current and previous counts classify identically.
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

// projectStatsFromLegs folds the per-(service, DERIVED class) reachability legs into the
// exposure summary band — the same projection foldExposure runs, reused here for the
// historical snapshot so current and previous counts are computed one way. An Exposure
// exists only where BOTH legs hold a value (ADR-0017): a service whose legs did not both
// conclude is "not reached", matching the tile's honest count. The legs are already
// collapsed to one per (service, derived class) by collapseReachLegs.
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

// --- standalone current-state counts (P2.1 stat band values) ---------------
//
// The Dashboard "Exposed services" and "Certs expiring ≤30d" stat cells (P2.1,
// Dashboard.jsx) render a current-state value alongside their vs-last-batch delta.
// The delta is withheld with no previous batch (statDeltas.Known=false), so the
// value cannot be read off the delta's Current — these read it directly, the same
// way the delta derives it, so value and chip agree when both are present.

// currentExposedCount is the count of services currently exposed to the internet —
// the "Exposed services" cell's value. It projects the current reachability legs
// through the same exposure engine the delta uses (projectExposureStats). ok is
// false on a read failure, so the cell degrades to an em dash rather than a
// fabricated zero.
func (s *server) currentExposedCount(ctx context.Context) (int, bool) {
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

// currentCertsExpiring is the count of endpoint certificates whose leaf expires
// within certExpiryWindow (≤30d) of now — the "Certs expiring ≤30d" cell's value.
// It reads the currently-open span corpus (an invalid @since selects every span
// still open — `closed_at IS NULL`) and counts the certificate facet's not_after the
// same way the delta does (countCertsExpiring), so the value and its delta agree.
// The count is honestly zero where no leaf carries a not_after; ok is false only on
// a read failure, so the cell degrades to an em dash.
func (s *server) currentCertsExpiring(ctx context.Context) (int, bool) {
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
