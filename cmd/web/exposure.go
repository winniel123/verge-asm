package main

import (
	"html/template"
	"log"
	"net/http"
	"sort"

	designfs "github.com/winniel123/verge-asm/design-system"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/exposure"
)

// The Exposure screen (screen 7, #560/#561) is served byte-for-byte from the frozen
// design-owned design-system/templates/exposure.tmpl (package v3.8.0, WORKFLOW v4),
// which replaces the repo-authored templates_exposure.go const (deleted). The tmpl
// keeps the "exposure" + "expleg" defines and renders inside the full app chrome
// ({{template "chrome" .}}); it declares the holes exposurePage shapes below —
// .Withheld, .Exposed, .Firewalled, .NotReached, .HasDeltas, .ExposedDelta.Change (int,
// via the signDelta funcmap entry, templates_shell.go), and .Rows[{Asset,Svc,Internal,
// Internet,Since}]. It styles against the design token vocabulary, so the render opts in
// with DesignTokens:true (the "head" block inlines tokens/*.css only then). exposure.tmpl
// auto-embeds through designfs's existing templates/*.tmpl glob, so no designfs.go change
// is needed. Reconciliation SPEC-CHANGE #20f (ruled): the withheld action targets
// Settings → Vantages (/settings/vantages, aliased in handlers.go), not /scope.
var _ = template.Must(tmpl.ParseFS(designfs.FS, "templates/exposure.tmpl"))

// The Exposure page — canonical `/exposure` (#300, T5, ADR-0110). Ported from
// design-system/examples/console/Exposure.jsx: the both-legs table (a service per
// row, its internal and internet reach legs side by side) with the summary stat
// band and the "one leg never concludes" callout, and — as a first-class rendered
// state, never an error — the WITHHELD state that names its cause when no internet
// vantage exists.
//
// The screen renders real current-state facts rather than the mock's sample data;
// the design is normative for look AND functionality (ADR-0116). Two deviations from
// the mock, each now backed by real state (no re-skin, nothing invented):
//
//   - The example's "Spec state" SegmentedControl (With vantage / Withheld) is a
//     design-review affordance to preview both states. Here the state is real:
//     WITHHELD renders exactly when the install has no internet vantage, and the
//     board renders otherwise. There is nothing to toggle, so the control is not
//     ported.
//   - The example's "+2" delta is now a real vs-last-batch datum (P0.2 #443, P2.6
//     #452, ADR-0116): the exposed tile renders its signed change against the
//     previous batch, reconstructed from the span corpus (deltas.go). Where no
//     previous batch exists (HasDeltas false) the tile shows no chip — its honest
//     no-delta state, never a fabricated +0. Firewalled and Not reached carry no
//     delta, matching the spec's own stat band, which chips only the exposed tile.
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

	// VERGE_DEV pixel-parity path (#560/#561). The frozen exposure.tmpl renders the
	// both-legs board (six rows, the +2 exposed delta, 41 firewalled, 7 not reached) and
	// the WITHHELD state — a curated corpus whose exact rows, counts and delta are the
	// design's, not a live-estate read. Reproducing them from the live derivations would
	// mean fabricating domain data, which SPEC-CHANGE forbids — so, exactly as the
	// SignIn/Setup/Coverage screens pin their dev fixture and serve it under devMode with a
	// drift test (TestExposureFixtureMatchesPackage), exposure serves the pinned fixtures.json
	// → exposure slice here so the seeded candidate renders byte-for-byte what the golden
	// composes. The withheld golden rides a dev ?variant=no-internet-vantage query (states.json),
	// which capture.mjs appends. A real deployment (devMode == false) falls through to the honest
	// live reads below.
	if s.devMode {
		s.render(w, r, "exposure", s.exposureFixtureData(acct, r.URL.Query().Get("variant")))
		return
	}

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
		s.render(w, r, "exposure", map[string]any{
			"Title": "Exposure", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
			"NavActive": "exposure",
			// exposure.tmpl styles against the design token vocabulary; the "head" block
			// inlines tokens/*.css only when this datum is set (as Coverage/Profile do).
			"DesignTokens": true,
			"Withheld":     true,
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
		"NavActive": "exposure",
		// exposure.tmpl styles against the design token vocabulary; the "head" block
		// inlines tokens/*.css only when this datum is set (as Coverage/Profile do).
		"DesignTokens": true,
		"Withheld":     false,
		"Rows":         rows,
		"Exposed":      stats.exposed,
		"Firewalled":   stats.firewalled,
		"NotReached":   stats.notReached,
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
	s.render(w, r, "exposure", data)
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
