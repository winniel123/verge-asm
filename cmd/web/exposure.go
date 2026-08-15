package main

import (
	"net/http"
	"net/netip"
	"sort"

	"github.com/winniel123/verge-asm/internal/custody"
	"github.com/winniel123/verge-asm/internal/db"
	"github.com/winniel123/verge-asm/internal/drift"
	"github.com/winniel123/verge-asm/internal/exposure"
)

// The Exposure landing view (v1 spec §6.2, ADR-0017/ADR-0029). The screen is the
// exposure board — the 2×2 projection over the internet/internal Reach legs —
// deliberately NOT the estate inventory. The web layer's only job is to fold the
// Derived reachability corpus into the per-Service snapshot the exposure engine
// projects; internal/exposure owns every verdict and every precondition, and this
// handler never decides one.
//
// Two things this handler does that the engine cannot, because they are reads of
// the corpus:
//
//   - It re-verifies each Vantage's class every render from the prober's
//     presented (dialled) address against the operator's declared address scopes
//     (CONTEXT.md `Vantage class`), never from the static `class` column — which
//     is why the reachability read carries the host, not the stored class.
//   - It folds the two most recent reachability spans per (Service, vantage) into
//     the current leg value and its immediate predecessor, so the flagship
//     internet not-reached → reached transition and a Break on the composing
//     derivation are both read off real spans.

// exposureOneLegView is one Service rendered under a single surviving Reach leg —
// never an Exposure value (ADR-0017). Statement is the sentence the absent leg
// carries: "we never looked" (never configured) or "we stopped looking" (a Gap).
type exposureOneLegView struct {
	Service   string
	Class     string
	Value     string
	Statement string
}

func (s *server) exposurePage(w http.ResponseWriter, r *http.Request, acct db.Account) {
	ctx := r.Context()

	seeds, err := s.store.ListSeeds(ctx)
	if err != nil {
		s.serverError(w, "exposure: list seeds", err)
		return
	}
	vantages, err := s.store.ListVantages(ctx)
	if err != nil {
		s.serverError(w, "exposure: list vantages", err)
		return
	}
	spans, err := s.store.ListReachabilitySpansForExposure(ctx)
	if err != nil {
		s.serverError(w, "exposure: list reachability spans", err)
		return
	}

	// The operator's declared address scopes are the boundary a presented address
	// is tested against — a family-matched prefix comparison over the address, and
	// never its spelling, so this gate cannot turn on a rendering.
	var scopes []netip.Prefix
	for _, sd := range seeds {
		if sd.Kind == "address" && sd.AddressCidr != nil {
			scopes = append(scopes, *sd.AddressCidr)
		}
	}
	covered := func(a netip.Addr) bool {
		a = a.Unmap()
		for _, p := range scopes {
			if p.Contains(a) {
				return true
			}
		}
		return false
	}

	// Install-level class presence: a class holds a value where at least one
	// AVAILABLE vantage re-verifies to it this render. Exposure needs both legs;
	// with fewer than two classes present, no Exposure is constructible anywhere.
	internetPresent, internalPresent := false, false
	for _, v := range vantages {
		if v.Availability.String != "available" {
			continue
		}
		switch classifyHost(v.Host.String, covered) {
		case custody.ClassInternet:
			internetPresent = true
		case custody.ClassInternal:
			internalPresent = true
		}
	}

	services := foldServiceReach(spans, covered, internetPresent, internalPresent)
	screen := exposure.Build(services, internetPresent, internalPresent)

	s.render(w, "exposure", map[string]any{
		"Title": "Exposure", "Account": acct, "IsAdmin": acct.Role == roleAdmin,
		"InternetPresent": screen.InternetPresent,
		"InternalPresent": screen.InternalPresent,
		"Constructible":   screen.Constructible,
		"NoServices":      screen.NoServices,
		"HasBoard":        screen.Board.Total() > 0,
		"BoardTotal":      screen.Board.Total(),
		"Exposed":         screen.Board.Exposed,
		"EdgeOnly":        screen.Board.EdgeOnly,
		"Firewalled":      screen.Board.Firewalled,
		"Unreachable":     screen.Board.Unreachable,
		"OneLegged":       oneLegViews(screen.OneLegged),
		"Broken":          screen.Broken,
		"WhatMoved":       screen.WhatMoved,
	})
}

// classifyHost re-verifies a Vantage's class from its prober endpoint. The
// presented (dialled) address is the endpoint the instance connected to; where it
// is a literal address the class is verified from it, and where it is a hostname
// the dialled IP is a measurement this read does not have, so the vantage is
// `unverified` until measurement supplies it (no exposure claims — #14's
// disposal). An interface address is never a presented address.
func classifyHost(host string, covered func(netip.Addr) bool) custody.VantageClass {
	addr, err := netip.ParseAddr(host)
	if err != nil {
		return exposure.VerifyClass(nil, covered) // unverified
	}
	return exposure.VerifyClass([]netip.Addr{addr}, covered)
}

// legAccum collects one Vantage class's per-vantage current and previous
// reachability outcomes for one Service, plus whether any of its timelines Broke
// between the two most recent spans.
type legAccum struct {
	current []string
	prev    []string
	broken  bool
}

// foldServiceReach folds the two-most-recent reachability spans per
// (Service, vantage) into the per-Service snapshot the exposure engine projects.
// Each span's vantage is classified from its host (re-verified, not read off the
// static column); rn = 1 is the current value and rn = 2 its predecessor. A Break
// between a Service's two most recent spans on any vantage marks the composing
// derivation as changed — "rules changed, nothing to compare yet".
func foldServiceReach(rows []db.ListReachabilitySpansForExposureRow, covered func(netip.Addr) bool, internetPresent, internalPresent bool) []exposure.ServiceInput {
	// service -> class -> accumulated outcomes. A Break is tracked per (service,
	// vantage) across its rn=1/rn=2 pair and then rolled up onto the class.
	type svcClass struct{ service, class string }
	byClass := map[svcClass]*legAccum{}
	// vantage span pairs, to detect a Break between the two most recent spans.
	type svcVant struct {
		service   string
		vantageID int64
	}
	pairVectors := map[svcVant][]drift.Vector{}
	classOf := map[svcVant]string{}
	order := []string{}
	seen := map[string]bool{}

	for _, row := range rows {
		if !seen[row.SubjectKey] {
			seen[row.SubjectKey] = true
			order = append(order, row.SubjectKey)
		}
		class := classString(classifyHost(row.Host.String, covered))
		if class == "" {
			continue // an unverified vantage contributes to neither leg
		}
		sv := svcVant{service: row.SubjectKey, vantageID: row.VantageID.Int64}
		classOf[sv] = class
		pairVectors[sv] = append(pairVectors[sv], decodeVector(row.Derivation))

		key := svcClass{service: row.SubjectKey, class: class}
		acc := byClass[key]
		if acc == nil {
			acc = &legAccum{}
			byClass[key] = acc
		}
		outcome := decodeReachOutcome(row.Value, row.IsGap)
		if row.Rn == 1 {
			acc.current = append(acc.current, outcome)
		} else {
			acc.prev = append(acc.prev, outcome)
		}
	}

	// Roll a Break on any (service, vantage) up onto the class accumulator: two
	// spans under differing Derivation vectors are a Break (ADR-0008).
	for sv, vecs := range pairVectors {
		if len(vecs) == 2 && !vecs[0].Equal(vecs[1]) {
			if acc := byClass[svcClass{service: sv.service, class: classOf[sv]}]; acc != nil {
				acc.broken = true
			}
		}
	}

	out := make([]exposure.ServiceInput, 0, len(order))
	for _, svc := range order {
		in := exposure.ServiceInput{Service: svc}
		internet := byClass[svcClass{service: svc, class: "internet"}]
		internal := byClass[svcClass{service: svc, class: "internal"}]
		in.Internet = composeLeg(internet, internetPresent)
		in.Internal = composeLeg(internal, internalPresent)
		if (internet != nil && internet.broken) || (internal != nil && internal.broken) {
			in.Broken = true
		}
		if internet != nil {
			if before, ok := exposure.ComposeReach(internet.prev); ok {
				in.InternetBefore = before
				in.InternetBeforeSet = true
			}
		}
		out = append(out, in)
	}
	return out
}

// composeLeg turns one class's accumulated current outcomes into a Reach leg.
// Where the class is not configured on the install the leg has no timeline
// ("we never looked"); where it is configured but decided no current value the
// leg holds a Gap ("we stopped looking"); otherwise it holds the existential
// composition (ADR-0080).
func composeLeg(acc *legAccum, classPresent bool) exposure.Leg {
	if !classPresent {
		return exposure.Leg{Status: exposure.LegNeverConfigured}
	}
	if acc != nil {
		if v, ok := exposure.ComposeReach(acc.current); ok {
			return exposure.Leg{Status: exposure.LegValued, Value: v}
		}
	}
	return exposure.Leg{Status: exposure.LegGap}
}

func classString(c custody.VantageClass) string {
	switch c {
	case custody.ClassInternet:
		return "internet"
	case custody.ClassInternal:
		return "internal"
	default:
		return ""
	}
}

// decodeReachOutcome reads the outcome off a reachability span value. A gap span
// carries no outcome, so it projects onto neither Reach value (the leg's Gap is
// decided by the class having no composed value, not by a gap span's text).
func decodeReachOutcome(raw []byte, isGap bool) string {
	if isGap {
		return ""
	}
	return decodeReachability(raw).Outcome
}

// oneLegViews shapes the engine's one-legged rows for rendering, turning the
// closed-union reason into the operator-facing statement.
func oneLegViews(rows []exposure.OneLeggedRow) []exposureOneLegView {
	out := make([]exposureOneLegView, 0, len(rows))
	for _, r := range rows {
		statement := "we never looked"
		if r.Reason == exposure.StoppedLooking {
			statement = "we stopped looking"
		}
		out = append(out, exposureOneLegView{
			Service:   r.Service,
			Class:     r.Class,
			Value:     string(r.Value),
			Statement: statement,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Service < out[j].Service })
	return out
}
