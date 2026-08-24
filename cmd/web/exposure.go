package main

import (
	"log"
	"net/http"
	"sort"

	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
)

// The Exposure page — canonical `/exposure` (#300, T5, ADR-0110). Ported from
// design-system/examples/console/Exposure.jsx: the both-legs table (a service per
// row, its internal and internet reach legs side by side) with the summary stat
// band and the "one leg never concludes" callout, and — as a first-class rendered
// state, never an error — the WITHHELD state that names its cause when no internet
// vantage exists.
//
// Two honest holds against the mock, on the reports.go precedent (fabricated mock
// data is re-skinned to real current-state facts, never invented):
//
//   - The example's "Spec state" SegmentedControl (With vantage / Withheld) is a
//     design-review affordance to preview both states. Here the state is real:
//     WITHHELD renders exactly when the install has no internet vantage, and the
//     board renders otherwise. There is nothing to toggle, so the control is not
//     ported.
//   - The example's "+2" delta is a trend the product does not carry (exposure is a
//     current-state census, never a series — see reports.go / internal/exposure), so
//     the stat tiles render the honest current count without a fabricated delta.
//
// Real data flows from the same corpus T1's asset detail reads: the per-class
// reachability spans (ListServiceReachabilitySpansByClass) compose each leg, and
// the composition is the pure internal/exposure engine (Project, ADR-0017) — an
// Exposure exists only where BOTH legs hold a value. Per-leg display reuses
// assetExposure (subjects.go). No figure is fabricated: an unmeasured leg reads
// `unverified`, never `firewalled` or `exposed`.

// exposureRow is one Service in the both-legs table: the address it sits on, its
// `:port transport`, the internal and internet reach legs as display states, and
// when its reachability span opened.
type exposureRow struct {
	Asset    string
	Svc      string
	Internal string
	Internet string
	Since    string
}

// exposureStats is the summary band over the projected board.
type exposureStats struct {
	exposed    int
	firewalled int
	notReached int
}

func (s *server) exposurePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	// The internet leg is a provisioned prober classed to the internet (probers.go:
	// "provisioning a prober declares this vantage is on the internet"). Without one,
	// no exposure claim is constructible — internal reachability may be complete, but
	// an exposed/firewalled verdict needs the outside leg — so the whole board is
	// WITHHELD (ADR-0017, spec §6.2). This is a first-class rendered state that names
	// its cause, not an error.
	vantages, err := s.store.ListVantages(ctx)
	if err != nil {
		s.serverError(w, "list vantages", err)
		return
	}
	internetVantage := false
	for _, v := range vantages {
		if v.Class == "internet" {
			internetVantage = true
			break
		}
	}
	if !internetVantage {
		s.render(w, "exposure", map[string]any{
			"Title": "Exposure", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"NavActive": "exposure",
			"Withheld":  true,
		})
		return
	}

	rows, stats := s.foldExposure(r)

	// Vs-last-batch deltas on the stat band (P0.2, #443, ADR-0116): the specced "+2"
	// delta is a real datum, so the exposed / firewalled / not-reached tiles each
	// render their change against the previous batch, reconstructed from the span
	// corpus. Known=false where no previous batch exists — the tiles then show their
	// no-delta state, never a fabricated zero. The P2.6 Exposure markup reads these.
	data := map[string]any{
		"Title": "Exposure", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"NavActive":  "exposure",
		"Withheld":   false,
		"Rows":       rows,
		"Exposed":    stats.exposed,
		"Firewalled": stats.firewalled,
		"NotReached": stats.notReached,
	}
	if prevAt, ok, err := s.previousBatchInstant(ctx); err != nil {
		log.Printf("web: exposure: previous batch instant: %v", err)
	} else if ok {
		if exposed, firewalled, notReached, dok := s.exposureCountDeltas(ctx, prevAt); dok {
			data["ExposedDelta"] = exposed
			data["FirewalledDelta"] = firewalled
			data["NotReachedDelta"] = notReached
			data["HasDeltas"] = true
		}
	}
	s.render(w, "exposure", data)
}

// legInfo is one class-scoped reachability leg as the by-class read carries it.
type legInfo struct {
	outcome string
	isGap   bool
	present bool
}

// foldExposure builds the both-legs table and the summary band from the per-class
// reachability spans, joined to the open-span corpus for each Service's "since".
// A read failure degrades to an empty board rather than 500ing a viewer's page.
func (s *server) foldExposure(r *http.Request) ([]exposureRow, exposureStats) {
	ctx := r.Context()
	byClass, err := s.store.ListServiceReachabilitySpansByClass(ctx)
	if err != nil {
		return nil, exposureStats{}
	}

	legs := map[string]map[string]legInfo{}
	order := []string{}
	for _, row := range byClass {
		m := legs[row.SubjectKey]
		if m == nil {
			m = map[string]legInfo{}
			legs[row.SubjectKey] = m
			order = append(order, row.SubjectKey)
		}
		m[row.Class] = legInfo{outcome: decodeReachability(row.Value).Outcome, isGap: row.IsGap, present: true}
	}
	sort.Strings(order)

	// "Since" is the Service's earliest open reachability span, read off the same
	// open-span corpus the asset detail reads (ListAllOpenSpans). Best-effort: a
	// read failure just leaves every "since" blank.
	since := map[string]string{}
	if spans, err := s.store.ListAllOpenSpans(ctx); err == nil {
		for _, sp := range spans {
			if sp.SubjectKind != "service" || sp.Facet != "reachability" || !sp.OpenedAt.Valid {
				continue
			}
			d := sp.OpenedAt.Time.UTC().Format("2006-01-02")
			if cur, ok := since[sp.SubjectKey]; !ok || d < cur {
				since[sp.SubjectKey] = d
			}
		}
	}

	var rows []exposureRow
	var stats exposureStats
	for _, svc := range order {
		addr, port, transport := splitServiceKey(svc)
		internal := legs[svc]["internal"]
		internet := legs[svc]["internet"]

		sinceStr := since[svc]
		if sinceStr == "" {
			sinceStr = "—"
		}
		rows = append(rows, exposureRow{
			Asset:    addr,
			Svc:      ":" + port + " " + transport,
			Internal: legDisplay(internal),
			Internet: legDisplay(internet),
			Since:    sinceStr,
		})

		// The summary band counts the projected 2x2 (ADR-0017): a value exists only
		// where both legs hold one. Exposed and Firewalled are projected values; "not
		// reached" is a Service whose legs did not both conclude, honestly matching the
		// tile's "no leg concluded this batch".
		ev, ok := exposure.Project(legFrom(internet), legFrom(internal))
		switch {
		case !ok:
			stats.notReached++
		case ev == exposure.Exposed:
			stats.exposed++
		case ev == exposure.Firewalled:
			stats.firewalled++
		}
	}
	return rows, stats
}

// legDisplay maps a leg to the chip state the both-legs table renders, reusing the
// asset-detail mapping (subjects.go): a reached leg is `exposed`, a not-reached leg
// is `not-reached`, and a Gap or an unmeasured class is `unverified` — never a
// firewalled/exposed claim for a leg we did not conclude.
func legDisplay(l legInfo) string {
	if !l.present {
		return "unverified"
	}
	return assetExposure(l.outcome, l.isGap)
}

// legFrom lifts a by-class span into the internal/exposure engine's Leg: a decided
// connection outcome is a valued leg, a Gap is a silent (stopped-looking) leg, and
// an unmeasured class was never configured. Only a valued leg feeds Project.
func legFrom(l legInfo) exposure.Leg {
	if !l.present {
		return exposure.Leg{Status: exposure.LegNeverConfigured}
	}
	if l.isGap {
		return exposure.Leg{Status: exposure.LegGap}
	}
	switch l.outcome {
	case string(exposure.Reached):
		return exposure.Leg{Status: exposure.LegValued, Value: exposure.Reached}
	case string(exposure.NotReached):
		return exposure.Leg{Status: exposure.LegValued, Value: exposure.NotReached}
	default:
		return exposure.Leg{Status: exposure.LegNeverConfigured}
	}
}
