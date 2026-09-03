// Package exposure derives the Exposure landing view: the ADR-0017 2×2 over two
// Reach legs, ADR-0080's existential class composition, and the ADR-0029 flagship
// not-reached → reached move. It is a pure read of an in-memory snapshot.
package exposure

import (
	"net/netip"
	"sort"

	"github.com/winniel123/verge-asm/internal/custody"
)

type ReachValue string

const (
	Reached    ReachValue = "reached"
	NotReached ReachValue = "not-reached"
)

const (
	outcomeReached    = "reached"
	outcomeNotReached = "not-reached"
)

func ComposeReach(outcomes []string) (ReachValue, bool) {
	decided := false
	for _, o := range outcomes {
		switch o {
		case outcomeReached:
			// One vantage reaching settles the leg; Reach never demands unanimity (ADR-0080).
			return Reached, true
		case outcomeNotReached:
			decided = true
		}
		// An unanswered connectionless exchange decided nothing (ADR-0083).
	}
	if decided {
		return NotReached, true
	}
	// An empty in-scope set is not vacuously not-reached, so the leg holds no value (ADR-0080).
	return "", false
}

type LegStatus string // the two absences keep their two statements (ADR-0017 decision 4)

const (
	LegValued          LegStatus = "valued"
	LegNeverConfigured LegStatus = "never-configured" // no timeline is not a Gap (ADR-0014)
	LegGap             LegStatus = "gap"
)

type Leg struct {
	Status LegStatus
	Value  ReachValue
}

func (l Leg) Valued() bool { return l.Status == LegValued }

type ExposureValue string

const (
	Exposed     ExposureValue = "exposed"
	EdgeOnly    ExposureValue = "edge-only"
	Firewalled  ExposureValue = "firewalled"
	Unreachable ExposureValue = "unreachable"
)

func Project(internet, internal Leg) (ExposureValue, bool) {
	if !internet.Valued() || !internal.Valued() {
		// No fifth value: internal-only named a one-legged reading, not the Service (ADR-0017).
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

func Flagship(before, after ReachValue) bool {
	// Alerting reads a Reach leg, never an Exposure state, in one direction only (ADR-0029).
	return before == NotReached && after == Reached
}

func VerifyClass(presented []netip.Addr, covered func(netip.Addr) bool) custody.VantageClass {
	// A class is what an outsider observed, never a static config field (CONTEXT.md Vantage class).
	if len(presented) == 0 {
		// With no prober no address is observed at all, so Exposure requires a prober.
		return custody.ClassUnverified
	}
	// A spelling-sensitive coverage test would turn this class on a rendering.
	for _, a := range presented {
		if !covered(a.Unmap()) {
			// The closed direction: a vantage misread as internal never alerts (ADR-0049, ADR-0079).
			return custody.ClassInternet
		}
	}
	return custody.ClassInternal
}

type ServiceInput struct {
	Service           string
	Internet          Leg
	Internal          Leg
	Broken            bool // rules changed, so the two spans may not be compared (ADR-0007)
	InternetBefore    ReachValue
	InternetBeforeSet bool // a leg that opened at its value emits no Transition (ADR-0029)
}

type OneLeggedRow struct {
	Service string
	Class   string
	Value   ReachValue
	Reason  OneLeggedReason
}

type OneLeggedReason string

const (
	NeverLooked    OneLeggedReason = "never-looked"
	StoppedLooking OneLeggedReason = "stopped-looking"
)

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
	// The board is a census, never a sampled or ranked view, so a cell holds its members.
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

type Screen struct {
	InternetPresent bool
	InternalPresent bool
	Constructible   bool
	NoServices      bool

	Board     Board
	OneLegged []OneLeggedRow
	Broken    []string
	WhatMoved []string
}

func Build(services []ServiceInput, internetPresent, internalPresent bool) Screen {
	// A precondition panel and the board co-exist as renders of one screen (ADR-0017 §6.2).
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
		// The flagship fires on the internet leg move whether or not the other leg exists (ADR-0029).
		if svc.Internet.Valued() && svc.InternetBeforeSet &&
			Flagship(svc.InternetBefore, svc.Internet.Value) {
			s.WhatMoved = append(s.WhatMoved, svc.Service)
		}

		if svc.Broken {
			// Rules changed, so no value is projected: never a cell, never a false reading (ADR-0007).
			s.Broken = append(s.Broken, svc.Service)
			continue
		}

		if ev, ok := Project(svc.Internet, svc.Internal); ok {
			s.Board.add(ev, svc.Service)
			continue
		}

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
