// Package exposure computes the Exposure landing view's Derived state: the 2×2
// projection of two Reach legs, the four preconditions each with a non-alarming
// rendering, the one-legged "we never looked" fallback, and the flagship
// internet not-reached → reached transition.
//
// It is a pure function over an in-memory snapshot, decoupled from storage — the
// same deep-module seam the Signal engine uses (internal/signal): the web layer
// reads reachability spans and vantages out of the corpus and assembles the
// snapshot, and this package decides what the screen renders and never touches a
// database. That is what makes every precondition and the composition testable
// hermetically.
//
// Three rulings the package encodes structurally rather than by discipline:
//
//   - Exposure exists ONLY where both legs hold a value (ADR-0017). Project
//     yields a value on two valued legs and yields nothing otherwise; there is no
//     fifth value for a one-legged reading, and `internal-only` is not in the
//     enum. A one-legged reading renders the surviving leg's raw Reach.
//   - Reach is a class-scoped Vantage composition whose quantifier is
//     `existential` — a presence claim (ADR-0080). ComposeReach reads `reached`
//     from one vantage of the class and never demands unanimity, because a
//     service reachable from one internet position and geo-blocked at another is
//     reachable from the internet.
//   - The flagship alert is the internet Reach leg going not-reached → reached,
//     never an Exposure state, and it fires whether or not the other leg exists
//     (ADR-0029). Flagship reads a leg move and is computed independently of the
//     board.
package exposure

import (
	"net/netip"
	"sort"

	"github.com/winniel123/verge-asm/internal/custody"
)

// ReachValue is one class-scoped Reach leg value — `reached` or `not-reached`,
// and nothing else (CONTEXT.md `Reach`). There is no value for "we did not look":
// a leg with no decided outcome holds no value at all, carried by Leg.Status
// rather than by a third ReachValue.
type ReachValue string

const (
	Reached    ReachValue = "reached"
	NotReached ReachValue = "not-reached"
)

// The two connection-oriented reachability outcomes a `reachability` observation
// projects onto a Reach value. Anything else — a connectionless exchange that
// went unanswered — decided nothing and projects onto neither value (ADR-0083),
// so it never reads as not-reached.
const (
	outcomeReached    = "reached"
	outcomeNotReached = "not-reached"
)

// ComposeReach applies the existential quantifier over one Vantage class's
// per-vantage reachability outcomes (Reach is a class-scoped Vantage composition
// whose quantifier is existential, ADR-0080). It returns:
//
//   - (Reached, true)    where any vantage of the class reached the Service —
//     one position reaching it is `reached`, since that is a presence claim and a
//     service reachable from one internet position is reachable from the internet;
//   - (NotReached, true) where at least one vantage of the class decided
//     not-reached and none reached;
//   - ("", false)        where no vantage of the class decided at all — an empty
//     in-scope set, every vantage unavailable, or only connectionless-undecided
//     outcomes. The leg holds no value, which the caller carries as a Gap or a
//     never-configured leg, never as not-reached (ADR-0080's empty-set ruling: an
//     empty in-scope set is not vacuously anything).
func ComposeReach(outcomes []string) (ReachValue, bool) {
	decided := false
	for _, o := range outcomes {
		switch o {
		case outcomeReached:
			// Existential: one vantage reaching settles the whole leg.
			return Reached, true
		case outcomeNotReached:
			decided = true
		}
		// Any other outcome decided nothing on a connection-oriented exchange.
	}
	if decided {
		return NotReached, true
	}
	return "", false
}

// LegStatus records why a Reach leg holds — or does not hold — a value, which is
// what lets the two absences keep their two statements (ADR-0017 decision 4):
// a never-configured class and a configured-then-silent class render alike in
// shape and differ only in the statement they carry.
type LegStatus string

const (
	LegValued LegStatus = "valued"
	// LegNeverConfigured: the Vantage class was never configured on this install,
	// so the leg has no timeline at all — "we never looked". Distinct from a Gap
	// (ADR-0014: no timeline is not a Gap).
	LegNeverConfigured LegStatus = "never-configured"
	LegGap             LegStatus = "gap"
)

type Leg struct {
	Status LegStatus
	Value  ReachValue
}

func (l Leg) Valued() bool { return l.Status == LegValued }

// ExposureValue is the 2×2 projection over the two valued Reach legs (ADR-0017).
// There are exactly four — `internal-only` was withdrawn, because it named a
// one-legged reading, which is a fact about which vantages the operator runs
// rather than about the Service.
type ExposureValue string

const (
	Exposed     ExposureValue = "exposed"     // both reached
	EdgeOnly    ExposureValue = "edge-only"   // internet reached, internal not-reached
	Firewalled  ExposureValue = "firewalled"  // internet not-reached, internal reached
	Unreachable ExposureValue = "unreachable" // both not-reached
)

// Project composes Exposure from the internet and internal legs. It yields a
// value only where BOTH legs are valued (ADR-0017); otherwise it yields
// ("", false), and the caller renders the surviving leg's raw Reach under
// "we never looked" — never a fifth value. The projection is a pure read of the
// 2×2, so a rule reads a leg and never this value.
func Project(internet, internal Leg) (ExposureValue, bool) {
	if !internet.Valued() || !internal.Valued() {
		return "", false
	}
	switch {
	case internet.Value == Reached && internal.Value == Reached:
		return Exposed, true
	case internet.Value == Reached && internal.Value == NotReached:
		return EdgeOnly, true
	case internet.Value == NotReached && internal.Value == Reached:
		return Firewalled, true
	default:
		return Unreachable, true
	}
}

// Flagship reports whether the internet leg's transition is the product's
// flagship move: not-reached → reached (ADR-0029). Alerting reads a Reach leg,
// never an Exposure state, and only the internet leg, in one direction — so this
// is computed on the internet leg alone and independently of whether an Exposure
// exists at all.
func Flagship(before, after ReachValue) bool {
	return before == NotReached && after == Reached
}

// VerifyClass classifies a Vantage every Batch from the addresses it was OBSERVED
// TO PRESENT, against the operator's declared address scopes — never a static
// config field (CONTEXT.md `Vantage class`). The presented set is what an outside
// observer saw: a prober's dialled address, known by construction, and the
// instance's own SSH_CLIENT as a prober reports it. An interface address is not a
// presented address.
//
// The quantifier is every-not-any and the closed direction is `internet`
// (ADR-0049's every-not-any, ADR-0079):
//
//   - no presented address observed at all → `unverified`. With no prober the
//     instance's address is unobserved, so Exposure is unreachable by
//     construction on that install — which is why Exposure requires a prober,
//     unconditionally.
//   - any one presented address uncovered by an address scope → `internet`, the
//     closed direction, because a vantage wrongly read as `internal` moves
//     observations onto the leg that never alerts.
//   - every presented address covered by a declared address scope → `internal`.
//
// covered tests address-scope coverage — a family-matched prefix comparison over
// the address and never its spelling, so this gate cannot turn on a rendering.
func VerifyClass(presented []netip.Addr, covered func(netip.Addr) bool) custody.VantageClass {
	if len(presented) == 0 {
		return custody.ClassUnverified
	}
	for _, a := range presented {
		if !covered(a.Unmap()) {
			return custody.ClassInternet
		}
	}
	return custody.ClassInternal
}

type ServiceInput struct {
	Service  string
	Internet Leg
	Internal Leg
	// Broken marks a Break on the composing Exposure derivation — rules changed,
	// so the two spans may not be compared and no value is projected (ADR-0007).
	Broken bool
	// InternetBefore is the internet leg's immediately-preceding Reach value, set
	// only where one exists; InternetBeforeSet distinguishes "no prior value"
	// (a leg that opened at its current value emits no Transition, ADR-0029) from
	// a genuine not-reached predecessor.
	InternetBefore    ReachValue
	InternetBeforeSet bool
}

// OneLeggedRow is a Service rendered under one surviving Reach leg — never an
// Exposure value (ADR-0017). Class names the surviving leg; Reason names the
// statement the absent leg carries.
type OneLeggedRow struct {
	Service string
	Class   string // "internet" or "internal"
	Value   ReachValue
	Reason  OneLeggedReason
}

// OneLeggedReason is the statement a one-legged reading carries about the leg it
// lacks (ADR-0017 decision 4): the two absences keep their two statements.
type OneLeggedReason string

const (
	// NeverLooked: the missing class was never configured — "we never looked".
	NeverLooked    OneLeggedReason = "never-looked"
	StoppedLooking OneLeggedReason = "stopped-looking"
)

// Board is the populated 2×2: the Service list in each Exposure cell. Every
// member is enumerable in full — the board is a census, never a sampled or ranked
// view — so the cells hold lists, not just counts.
type Board struct {
	Exposed     []string
	EdgeOnly    []string
	Firewalled  []string
	Unreachable []string
}

func (b Board) Total() int {
	return len(b.Exposed) + len(b.EdgeOnly) + len(b.Firewalled) + len(b.Unreachable)
}

func (b *Board) add(v ExposureValue, service string) {
	switch v {
	case Exposed:
		b.Exposed = append(b.Exposed, service)
	case EdgeOnly:
		b.EdgeOnly = append(b.EdgeOnly, service)
	case Firewalled:
		b.Firewalled = append(b.Firewalled, service)
	case Unreachable:
		b.Unreachable = append(b.Unreachable, service)
	}
}

// Screen is the whole Exposure landing view's Derived state — the preconditions
// and the populated board TOGETHER, because a precondition panel and a board
// co-exist as distinct renders of the same screen and are not mutually exclusive
// (ADR-0017; spec §6.2).
type Screen struct {
	InternetPresent bool
	InternalPresent bool
	// Constructible is true where both classes are present, so an Exposure can be
	// composed at all. Below it, no Exposure is constructible anywhere and every
	// Service renders one-legged — the custody-of-nothing and no-prober cases land
	// here honestly, never as a false internal-only reading (spec §6.2).
	Constructible bool
	NoServices    bool

	Board     Board
	OneLegged []OneLeggedRow
	Broken    []string
	WhatMoved []string
}

func Build(services []ServiceInput, internetPresent, internalPresent bool) Screen {
	s := Screen{
		InternetPresent: internetPresent,
		InternalPresent: internalPresent,
		Constructible:   internetPresent && internalPresent,
	}
	if len(services) == 0 {
		s.NoServices = true
		return s
	}
	for _, svc := range services {
		// The flagship fires on the internet leg move whether or not the other leg
		// exists (ADR-0029) — computed first, before the board/one-legged routing.
		if svc.Internet.Valued() && svc.InternetBeforeSet &&
			Flagship(svc.InternetBefore, svc.Internet.Value) {
			s.WhatMoved = append(s.WhatMoved, svc.Service)
		}

		if svc.Broken {
			// Rules changed on the composing derivation: no value may be projected,
			// so the Service is a "rules changed" render — never a cell, never a false
			// reading — and it co-exists with the board the others populate.
			s.Broken = append(s.Broken, svc.Service)
			continue
		}

		if ev, ok := Project(svc.Internet, svc.Internal); ok {
			s.Board.add(ev, svc.Service)
			continue
		}

		// One-legged (or both legs absent): render the surviving leg's raw Reach.
		if row, ok := oneLegged(svc); ok {
			s.OneLegged = append(s.OneLegged, row)
		}
	}
	sort.Strings(s.Board.Exposed)
	sort.Strings(s.Board.EdgeOnly)
	sort.Strings(s.Board.Firewalled)
	sort.Strings(s.Board.Unreachable)
	sort.Strings(s.Broken)
	sort.Strings(s.WhatMoved)
	sort.Slice(s.OneLegged, func(i, j int) bool { return s.OneLegged[i].Service < s.OneLegged[j].Service })
	return s
}

func oneLegged(svc ServiceInput) (OneLeggedRow, bool) {
	switch {
	case svc.Internet.Valued() && !svc.Internal.Valued():
		return OneLeggedRow{
			Service: svc.Service,
			Class:   "internet",
			Value:   svc.Internet.Value,
			Reason:  reasonFor(svc.Internal.Status),
		}, true
	case svc.Internal.Valued() && !svc.Internet.Valued():
		return OneLeggedRow{
			Service: svc.Service,
			Class:   "internal",
			Value:   svc.Internal.Value,
			Reason:  reasonFor(svc.Internet.Status),
		}, true
	default:
		return OneLeggedRow{}, false
	}
}

func reasonFor(absent LegStatus) OneLeggedReason {
	if absent == LegGap {
		return StoppedLooking
	}
	return NeverLooked
}
